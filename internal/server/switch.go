package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/AsaphNoam/Chuck/internal/backend"
	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// switchCancelTimeout bounds the wait for an in-flight turn to settle before the
// switch stops the old runtime (techspec §9; config plumbing deferred).
const switchCancelTimeout = 5 * time.Second
const switchRollbackTimeout = 30 * time.Second

// switchRuntimeRequest is the POST body (techspec §8.1); any subset, ≥1 field
// must differ from current.
type switchRuntimeRequest struct {
	Interface string `json:"interface"`
	Backend   string `json:"backend"`
	Model     string `json:"model"`
	Effort    string `json:"effort"`
	Driver    string `json:"driver"` // target terminal driver: ""/"xterm" | "tmux" | "iterm2" (§3.5)
}

// switchRuntimeResponse is the 200 body (techspec §8.1).
type switchRuntimeResponse struct {
	AgentID        string               `json:"agent_id"`
	Interface      string               `json:"interface"`
	Backend        string               `json:"backend"`
	Model          string               `json:"model"`
	Effort         string               `json:"effort"`
	Running        *state.RunningEntry  `json:"running,omitempty"`
	HistoryHandoff string               `json:"history_handoff"` // "native_resume" | "primer"
	Worktree       *worktreeStartNotice `json:"worktree,omitempty"`
}

// handleSwitchRuntime implements POST /api/sessions/{id}/switch-runtime
// (techspec §5.1–5.4): stop the current runtime, persist the new identity, and
// resume on the same agent_id. Same-backend compatible switches use native
// resume; cross-backend/incompatible switches use a bounded Chuck transcript
// history primer.
func (s *Server) handleSwitchRuntime(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req switchRuntimeRequest
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAPIError(w, apiError(runtime.CodeValidation, "invalid JSON body"))
			return
		}
	}

	// Derive partial updates only under the exclusive claim: another completed
	// switch must not be overwritten by fields read before this transition owns it.
	if !s.acquireSwitch(id) {
		writeAPIError(w, apiError(runtime.CodeSwitchInProgress, "a switch is already in progress for this agent"))
		return
	}
	defer s.releaseSwitch(id)
	if !s.claimLifecycle(id) {
		writeAPIError(w, apiError(runtime.CodeConflict, "a lifecycle transition is already in progress"))
		return
	}
	defer s.releaseLifecycle(id)

	agent, err := s.stateStore.ReadAgent(id)
	if err != nil {
		writeAPIError(w, apiError(runtime.CodeNotFound, "no such agent: "+id))
		return
	}
	if agent.Archived {
		writeAPIError(w, apiError(runtime.CodeAgentArchived, "agent is archived"))
		return
	}
	if ae := s.projectArchiveGate(agent.Project, "project is archived"); ae != nil {
		writeAPIError(w, ae)
		return
	}
	if ae := s.acquireAgentStart(agent.Project, id); ae != nil {
		writeAPIError(w, ae)
		return
	}
	defer s.releaseAgentStart(agent.Project, id)

	// Compute the target identity (merge requested fields over current).
	target := agent
	if req.Interface != "" {
		target.Interface = req.Interface
	}
	if req.Backend != "" {
		target.Backend = req.Backend
	}
	if req.Model != "" {
		target.Model = req.Model
	}
	if req.Effort != "" {
		target.Effort = req.Effort
	} else if target.Backend != agent.Backend || target.Model != agent.Model {
		if backends, err := s.readBackendsOrDefault(); err == nil {
			if be, ok := backends.Backends[target.Backend]; ok {
				if model, ok := be.Models[target.Model]; ok {
					target.Effort, _ = resolveEffort("", model, nil)
				}
			}
		}
	}
	if target.Interface == agent.Interface && target.Backend == agent.Backend && target.Model == agent.Model && target.Effort == agent.Effort {
		writeAPIError(w, apiError(runtime.CodeNoChange, "switch request equals current state"))
		return
	}
	if target.Interface != "chat" && target.Interface != "terminal" {
		writeAPIError(w, apiError(runtime.CodeInvalidField, "invalid interface: "+target.Interface))
		return
	}
	if target.Interface == "terminal" {
		if ae := validateTerminalDriver(req.Driver); ae != nil {
			writeAPIError(w, ae)
			return
		}
	}

	// The switch operates on a live agent (cancel → stop → resume). No running row
	// → nothing to switch.
	prev, err := s.stateStore.ReadRunning(id)
	if errors.Is(err, state.ErrNotFound) {
		writeAPIError(w, apiError(runtime.CodeAgentNotRunning, "agent is not running"))
		return
	}
	if err != nil {
		s.logSwitchFailure(id, runtime.CodeInternal, "read running: "+err.Error())
		writeAPIError(w, apiError(runtime.CodeInternal, err.Error()))
		return
	}
	previousSnapshot, err := s.stateStore.ReadSession(id)
	if errors.Is(err, state.ErrNotFound) {
		writeAPIError(w, apiError(runtime.CodeValidation, "no persisted session to switch"))
		return
	}
	if err != nil {
		s.logSwitchFailure(id, runtime.CodeInternal, "read previous session: "+err.Error())
		writeAPIError(w, apiError(runtime.CodeInternal, "read previous session: "+err.Error()))
		return
	}

	// Resolve native resume vs primer via the target adapter (§5.3): cross-backend
	// swaps never share native history; same-backend model swaps use native resume
	// only when the adapter says that is supported.
	ad, ok := backend.For(s.backendType(target.Backend))
	if !ok {
		writeAPIError(w, apiError(runtime.CodeInvalidField, "unknown backend: "+target.Backend))
		return
	}
	handoff := "native_resume"
	sameBackend := target.Backend == agent.Backend
	usePrimer := !sameBackend || (target.Model != agent.Model && !ad.CanSwitchModelOnResume())
	resumeID := ad.ResolveResumeID(prev.SessionID, sameBackend && !usePrimer)
	if usePrimer {
		resumeID = ""
		handoff = "primer"
	}

	// Validate the target backend/model/session snapshot exist before tearing
	// anything down — WITHOUT the registration side effects. composeSwitchSpec
	// (below) writes the per-agent hook settings file and registers a fresh MCP
	// token, both keyed by the unchanged agent_id; doing that here would let the
	// old-artifact cleanup (step 2) wipe the target's just-created registration.
	if ae := s.validateSwitchTarget(target); ae != nil {
		writeAPIError(w, ae)
		return
	}
	if usePrimer && !ad.SupportsHistoryPrimer() {
		writeAPIError(w, apiError(runtime.CodeValidation, "this provider has no verified history handoff; keep the current model or switch to Claude or Codex"))
		return
	}

	// A codex switch resumes into the isolated CODEX_HOME profile, whose refresh
	// (spawnCmd → RefreshCodexProfile) fails closed on a newly unsafe/unreadable
	// personal asset. Prepare that generation BEFORE stopping the working runtime:
	// otherwise the same broken asset fails both the target resume and the
	// rollback resume, leaving the agent dead with spoofable registration
	// artifacts (INV §4/§15). Preparing here keeps the old runtime live on failure;
	// the target/rollback resumes re-run the now-succeeding, idempotent refresh.
	if s.backendType(agent.Backend) == "codex-acp" || s.backendType(target.Backend) == "codex-acp" {
		if err := config.RefreshCodexProfile(config.CodexProfileDir(s.configStore.Home())); err != nil {
			s.logSwitchFailure(id, runtime.CodeInternal, "prepare codex profile: "+err.Error())
			writeAPIError(w, apiError(runtime.CodeInternal, "prepare codex profile: "+err.Error()))
			return
		}
	}

	// From here the switch tears down the source and must run to an outcome
	// (target live, rolled back, or error status) even if the client goes away:
	// a request-scoped cancel would abort the target start and force a rollback.
	ctx := context.WithoutCancel(r.Context())

	// 1. Cancel any in-flight turn and let it settle (streamed events already
	//    persisted), then stop the old runtime (removes running row, status done).
	s.cancelAndWait(ctx, id, switchCancelTimeout)
	if err := s.registry.Stop(ctx, id); err != nil {
		if !errors.Is(err, runtime.ErrNoHandle) {
			ae := orphanReapAPIError("stop current runtime: ", err)
			s.logSwitchFailure(id, ae.Code, ae.Message)
			writeAPIError(w, ae)
			return
		}
		// ErrNoHandle: no handler in registry, but check for orphan runtimes (invariant §4).
		// An unreaped orphan must not gain a second process (FS-01.R21/R54).
		if rerr := s.reapOrphanRuntime(id); rerr != nil {
			ae := orphanReapAPIError("reap orphan runtime: ", rerr)
			s.logSwitchFailure(id, ae.Code, ae.Message)
			writeAPIError(w, ae)
			return
		}
	}

	// 2. Clean the OLD runtime's MCP token + hook settings (keyed by agent_id)
	//    BEFORE composing the target spec. Because the agent_id is unchanged,
	//    composeSwitchSpec re-registers a fresh MCP token and rewrites the
	//    per-agent hook settings file under the same id; cleaning afterward would
	//    revoke the new token and delete the settings file the resume needs (and
	//    orphan the old MCP token, whose cleanup closure the new register overwrote).
	s.teardownAgentRegistration(id)

	// 3. Compose the target launch spec (registers a fresh MCP token + writes the
	//    per-agent hook settings file for the target identity).
	// Any failure from here on happens AFTER the old runtime was stopped + cleaned,
	// so it must roll back to the previous identity (re-register + re-resume) rather
	// than leave the agent dead with no running row — the same recovery step 5 uses.
	var worktreeNotice worktreeStartNotice
	ae, stopErr := s.startRuntimeClaimed(ctx, false, func() (runtime.LaunchSpec, *runtime.APIError) {
		return s.composeSwitchSpecContext(ctx, target, resumeID, &worktreeNotice)
	}, func(spec *runtime.LaunchSpec) *runtime.APIError {
		spec.Driver = req.Driver
		// Preserve the frozen selection on interface-only switches. Record
		// an explicit choice when switching the backend or model.
		if target.Backend != agent.Backend || target.Model != agent.Model {
			var err error
			spec.LaunchConfig, err = explicitModelConfig(spec.LaunchConfig)
			if err != nil {
				return apiError(runtime.CodeInternal, "record model choice: "+err.Error())
			}
		}
		if usePrimer {
			primer, err := s.buildHistoryPrimer(ctx, *spec, s.switchPrimerTokenBudget())
			if err != nil {
				return apiError(runtime.CodeInternal, "build history primer: "+err.Error())
			}
			// Replace the prior bridge without stacking it into the persona.
			spec.HistoryPrimer = primer
		}
		if err := s.stateStore.WriteAgentRuntimeIdentity(target); err != nil {
			return apiError(runtime.CodeInternal, "persist identity: "+err.Error())
		}
		return nil
	})
	if stopErr != nil {
		s.failSwitch(w, id, "stop failed target: "+stopErr.Error())
		return
	}
	if ae != nil {
		s.rollbackSwitch(ctx, w, agent, prev.SessionID, previousSnapshot, errors.New(ae.Message))
		return
	}
	if usePrimer {
		if err := appendBackendSwitchMarker(s.registry, id, agent.Backend+"/"+agent.Model, target.Backend+"/"+target.Model, time.Now().UTC()); err != nil {
			// The target is already active: keep it, and say exactly that
			// rather than implying no switch happened (TS-03.R73). No raw
			// storage error crosses the API.
			s.log.Error("append backend switch marker", "agent", id, "err", err)
			writeAPIError(w, apiError(runtime.CodeInternal, "Switched to "+target.Backend+"/"+target.Model+
				", but the switch could not be recorded in the conversation history. Refresh before taking another action."))
			return
		}
	}

	// The identity write + the resume's running/status writes already published
	// state_update via the runtime's state-touch, so the card re-renders.
	running, _ := s.stateStore.ReadRunning(id)
	writeJSON(w, http.StatusOK, switchRuntimeResponse{
		AgentID: id, Interface: target.Interface, Backend: target.Backend, Model: target.Model, Effort: target.Effort,
		Running: ptrRunning(running), HistoryHandoff: handoff, Worktree: func() *worktreeStartNotice {
			if worktreeNotice.Recreated {
				return &worktreeNotice
			}
			return nil
		}(),
	})
}

// rollbackSwitch re-launches the previous identity after a failed Resume (§5.4).
// If rollback also fails, it leaves the status row at error and returns 500
// switch_failed.
func (s *Server) rollbackSwitch(_ context.Context, w http.ResponseWriter, prevAgent state.Agent, prevSessionID string, previousSnapshot state.SessionSnapshot, cause error) {
	rollbackCtx, cancel := context.WithTimeout(context.Background(), switchRollbackTimeout)
	defer cancel()
	// A target that got as far as its transcript replaced the session snapshot.
	// Put the previous selection and native session back together before the
	// rollback start, which may fail before it writes a snapshot of its own.
	restored := previousSnapshot
	if prevSessionID != "" {
		restored.LastSessionID = prevSessionID // the ID the rollback start resumes
	}
	if err := s.stateStore.RestoreSwitchSource(prevAgent, restored); err != nil {
		s.failSwitch(w, prevAgent.AgentID, "restore identity: "+err.Error())
		return
	}
	// The transcript's last session record rebuilds the row on reindex and
	// carries the search metadata; supersede the target's there too.
	if err := s.appendRestoredSessionMeta(prevAgent, restored); err != nil {
		s.log.Warn("record restored switch source", "agent_id", prevAgent.AgentID, "err", err)
	}
	// Compose from the pre-switch snapshot: a target startup may already have
	// written its own metadata (bridge, launch object, contract) before failing.
	ae, stopErr := s.startRuntimeClaimed(rollbackCtx, false, func() (runtime.LaunchSpec, *runtime.APIError) {
		spec, ae := s.composeSwitchSpecFrom(rollbackCtx, prevAgent, prevSessionID, previousSnapshot, nil)
		if ae != nil {
			return runtime.LaunchSpec{}, apiError(ae.Code, "recompose previous: "+ae.Message)
		}
		return spec, nil
	}, nil)
	if stopErr != nil {
		s.failSwitch(w, prevAgent.AgentID, "stop failed rollback: "+stopErr.Error())
		return
	}
	if ae != nil {
		s.failSwitch(w, prevAgent.AgentID, ae.Message)
		return
	}
	s.logSwitchFailure(prevAgent.AgentID, runtime.CodeSwitchFailedRolledBack, cause.Error())
	writeAPIError(w, apiError(runtime.CodeSwitchFailedRolledBack, "switch failed, rolled back to previous runtime: "+cause.Error()))
}

// logSwitchFailure writes the one correlated record for a switch that answers
// with a server-side failure (INV 22): agent, operation, typed code, bounded reason.
func (s *Server) logSwitchFailure(agentID, code, reason string) {
	s.log.Warn("switch runtime failed", "agent_id", agentID, "operation", "switch-runtime", "code", code, "reason", clipDetail(reason))
}

// appendRestoredSessionMeta commits the previous selection as the stopped
// agent's newest session record, through the same journal and index path a
// start uses.
func (s *Server) appendRestoredSessionMeta(a state.Agent, snap state.SessionSnapshot) error {
	aware, caps := snap.ChuckAware, snap.RuntimeCapabilities
	data, err := json.Marshal(runtime.SessionMetaData{
		Name: a.Name, Role: a.Role, Project: a.Project, Group: a.Group,
		Backend: snap.Backend, Model: snap.Model, Effort: snap.Effort, Fast: snap.Fast, Interface: snap.Interface,
		Cwd: snap.Cwd, WorkspaceID: snap.WorkspaceID, SystemPrompt: snap.SystemPrompt, HistoryPrimer: snap.HistoryPrimer,
		ChuckAware: &aware, EnvKeys: snap.EnvKeys, SkipPermissions: snap.SkipPermissions, AddDirs: snap.AddDirs,
		LaunchConfig: snap.LaunchConfig, ExecutionContract: snap.ExecutionContract, RuntimeCapabilities: &caps,
		CreatedAt: snap.CreatedAt, SessionID: snap.LastSessionID,
	})
	if err != nil {
		return err
	}
	_, err = s.registry.AppendHistory(a.AgentID, runtime.CommitRequest{Event: runtime.Event{
		Type: runtime.EvSessionMeta, Data: data, Ts: time.Now().UTC().Format(time.RFC3339),
	}})
	return err
}

// failSwitch records the unrecoverable state (status error) and returns 500
// switch_failed; the agent is recoverable via archive resume (§5.4).
func (s *Server) failSwitch(w http.ResponseWriter, agentID, detail string) {
	s.logSwitchFailure(agentID, runtime.CodeSwitchFailed, detail)
	if st, err := s.stateStore.ReadStatus(agentID); err == nil {
		st.State = state.ComposeChatStatus(state.ChatProcessFailed, state.ConversationTurn{}, false)
		st.Detail = "switch failed: " + clipDetail(detail)
		st.BusySince = nil
		_ = s.stateStore.WriteStatus(st)
		if _, terr := s.stateMgr.Touch(agentID); terr != nil {
			s.log.Debug("touch after failed switch", "agent", agentID, "err", terr)
		}
	}
	writeAPIError(w, apiError(runtime.CodeSwitchFailed, detail))
}

// validateSwitchTarget checks the target identity is launchable — the frozen
// session snapshot exists and the target backend/model are known — without any
// registration side effects, so the switch can reject a bad request before it
// stops the live agent. composeSwitchSpec repeats these lookups (and does the
// registration) once the old artifacts are cleaned.
func (s *Server) validateSwitchTarget(target state.Agent) *runtime.APIError {
	snap, err := s.stateStore.ReadSession(target.AgentID)
	if err != nil {
		if errors.Is(err, state.ErrNotFound) {
			return apiError(runtime.CodeValidation, "no persisted session to switch")
		}
		return apiError(runtime.CodeInternal, err.Error())
	}
	backends, err := s.configStore.ReadBackends()
	if err != nil {
		if errors.Is(err, config.ErrNotFound) || errors.Is(err, config.ErrCorrupt) {
			backends = config.DefaultBackends()
		} else {
			return apiError(runtime.CodeInternal, "read backends: "+err.Error())
		}
	}
	be, ok := backends.Backends[target.Backend]
	if !ok {
		return apiError(runtime.CodeInvalidField, "unknown backend: "+target.Backend)
	}
	if _, ok := be.Models[target.Model]; !ok {
		return apiError(runtime.CodeInvalidField, "unknown model: "+target.Model)
	}
	if err := config.ValidateModelEffort(be, be.Models[target.Model], target.Effort); err != nil {
		return apiError(runtime.CodeInvalidField, err.Error())
	}
	// An undecidable legacy selection refuses before the source runtime stops
	// (FS-01.R53); the rollback composer would refuse it too.
	if _, ae := continuedContract(snap, target, be, be.Models[target.Model], false); ae != nil {
		return ae
	}
	// Only claude-acp supports the terminal interface; reject a terminal switch to
	// any other backend rather than land a statusless agent that drops the composed
	// spec (mirrors the launch guard, §6 capability honesty).
	if target.Interface == "terminal" && !terminalSupported(be.Type) {
		return apiError(runtime.CodeTerminalUnavailable, terminalUnsupportedReason(be.Type))
	}
	// A missing target provider must not stop the working runtime (TS-04.R71).
	providerEnv, _, ae := providerLaunchLayer(be, be.Models[target.Model])
	if ae != nil {
		return ae
	}
	// Nor may a bound Claude target whose child would read another profile.
	return s.preflightClaudeChildProfile(context.Background(), be.Type, snap.LaunchConfig, s.quotaCheckEnv(be, be.Models[target.Model], providerEnv))
}

// composeSwitchSpec builds the resume LaunchSpec for the target identity from the
// frozen session snapshot (cwd/system_prompt/skip_permissions/add_dirs) plus the
// target backend/model env, minting a fresh hook token and MCP registration
// (mirrors handleResume). Switch overrides only backend/model/interface; the
// permission policy and add_dirs stay frozen to the original launch (§12.4).
func (s *Server) composeSwitchSpec(target state.Agent, resumeID string) (runtime.LaunchSpec, *runtime.APIError) {
	return s.composeSwitchSpecContext(context.Background(), target, resumeID, nil)
}

func (s *Server) composeSwitchSpecContext(ctx context.Context, target state.Agent, resumeID string, notice *worktreeStartNotice) (runtime.LaunchSpec, *runtime.APIError) {
	snap, err := s.stateStore.ReadSession(target.AgentID)
	if errors.Is(err, state.ErrNotFound) {
		return runtime.LaunchSpec{}, apiError(runtime.CodeValidation, "no persisted session to switch")
	}
	if err != nil {
		return runtime.LaunchSpec{}, apiError(runtime.CodeInternal, err.Error())
	}
	return s.composeSwitchSpecFrom(ctx, target, resumeID, snap, notice)
}

// composeSwitchSpecFrom builds the switch/rollback spec from an explicit
// snapshot. Rollback passes the previous identity's pre-switch snapshot, so a
// failed target start cannot leak its metadata into the restored runtime.
func (s *Server) composeSwitchSpecFrom(ctx context.Context, target state.Agent, resumeID string, snap state.SessionSnapshot, notice *worktreeStartNotice) (runtime.LaunchSpec, *runtime.APIError) {
	backends, err := s.configStore.ReadBackends()
	if err != nil {
		if errors.Is(err, config.ErrNotFound) || errors.Is(err, config.ErrCorrupt) {
			backends = config.DefaultBackends()
		} else {
			return runtime.LaunchSpec{}, apiError(runtime.CodeInternal, "read backends: "+err.Error())
		}
	}
	be, ok := backends.Backends[target.Backend]
	if !ok {
		return runtime.LaunchSpec{}, apiError(runtime.CodeInvalidField, "unknown backend: "+target.Backend)
	}
	model, ok := be.Models[target.Model]
	if !ok {
		return runtime.LaunchSpec{}, apiError(runtime.CodeInvalidField, "unknown model: "+target.Model)
	}
	// Same identity continues the frozen selection; a new backend/model is an
	// explicit catalogue choice (TS-01.R40).
	contract, ae := continuedContract(snap, target, be, model, false)
	if ae != nil {
		return runtime.LaunchSpec{}, ae
	}

	// validateSwitchTarget already proved the provider selectable before the
	// stop; re-resolve here because this is the actual process start.
	providerEnv, provider, pae := providerLaunchLayer(be, model)
	if pae != nil {
		return runtime.LaunchSpec{}, pae
	}
	// Resolve the working directory through the one shared step before any
	// registration side effect (TS-01.R26, TS-12.R4). Switch keeps the frozen
	// snapshot cwd for the same reason resume does.
	checkout, wae := s.resolveStartCheckout(ctx, target.AgentID, target.Project, snap.WorkspaceID, snap.Cwd)
	if wae != nil {
		return runtime.LaunchSpec{}, wae
	}
	if checkout.Recreated && notice != nil {
		*notice = worktreeStartNotice{Recreated: true, Branch: checkout.Branch, Warning: checkout.Warning}
	}
	// Ensure the shared-resources directory before any registration side effect
	// (FS-11.R6/R9). add_dirs/prompt carry over from the frozen snapshot; the env
	// var is recomposed here (INV §2).
	resourceDir, rae := s.ensureProjectResources(target.Project)
	if rae != nil {
		return runtime.LaunchSpec{}, rae
	}

	snap.LastSessionID = resumeID
	snap.Effort, snap.Fast, snap.ExecutionContract = target.Effort, target.Fast, contract
	return s.buildLaunchSpec(startSpec{
		agent: target, session: snap, backend: be, model: model, provider: provider,
		providerEnv: providerEnv, resourceDir: resourceDir,
	})
}

// cancelAndWait cancels the in-flight turn and waits up to timeout for the agent
// to leave the busy state (best-effort; the streamed events are already persisted).
func (s *Server) cancelAndWait(ctx context.Context, id string, timeout time.Duration) {
	_, _ = s.registry.Cancel(ctx, id)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		st, err := s.stateStore.ReadStatus(id)
		if err != nil || st.State != "busy" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func (s *Server) backendType(backendID string) string {
	backends, err := s.configStore.ReadBackends()
	if err != nil {
		backends = config.DefaultBackends()
	}
	if be, ok := backends.Backends[backendID]; ok {
		return be.Type
	}
	return ""
}

func (s *Server) switchPrimerTokenBudget() int {
	cfg := s.cfg
	if fromDisk, err := s.configStore.ReadConfig(); err == nil {
		cfg = fromDisk
	}
	if cfg.Switch.PrimerTokenBudget <= 0 || cfg.Switch.PrimerTokenBudget > defaultPrimerTokenBudget {
		return defaultPrimerTokenBudget
	}
	return cfg.Switch.PrimerTokenBudget
}

func (s *Server) acquireSwitch(id string) bool {
	s.switchMu.Lock()
	defer s.switchMu.Unlock()
	if s.switching[id] {
		return false
	}
	s.switching[id] = true
	return true
}

func (s *Server) releaseSwitch(id string) {
	s.switchMu.Lock()
	delete(s.switching, id)
	s.switchMu.Unlock()
}

func ptrRunning(r state.RunningEntry) *state.RunningEntry {
	if r.AgentID == "" {
		return nil
	}
	return &r
}

func clipDetail(s string) string {
	if len(s) <= 120 {
		return s
	}
	return s[:120]
}
