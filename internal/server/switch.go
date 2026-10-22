package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
		writeAPIError(w, apiError(runtime.CodeInternal, err.Error()))
		return
	}
	previousSnapshot, err := s.stateStore.ReadSession(id)
	if errors.Is(err, state.ErrNotFound) {
		writeAPIError(w, apiError(runtime.CodeValidation, "no persisted session to switch"))
		return
	}
	if err != nil {
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
			writeAPIError(w, apiError(runtime.CodeInternal, "prepare codex profile: "+err.Error()))
			return
		}
	}

	// 1. Cancel any in-flight turn and let it settle (streamed events already
	//    persisted), then stop the old runtime (removes running row, status done).
	s.cancelAndWait(r.Context(), id, switchCancelTimeout)
	if err := s.registry.Stop(r.Context(), id); err != nil {
		if !errors.Is(err, runtime.ErrNoHandle) {
			writeAPIError(w, orphanReapAPIError("stop current runtime: ", err))
			return
		}
		// ErrNoHandle: no handler in registry, but check for orphan runtimes (invariant §4).
		// An unreaped orphan must not gain a second process (FS-01.R21/R54).
		if rerr := s.reapOrphanRuntime(id); rerr != nil {
			writeAPIError(w, orphanReapAPIError("reap orphan runtime: ", rerr))
			return
		}
	}

	// 2. Clean the OLD runtime's MCP token + hook settings (keyed by agent_id)
	//    BEFORE composing the target spec. Because the agent_id is unchanged,
	//    composeSwitchSpec re-registers a fresh MCP token and rewrites the
	//    per-agent hook settings file under the same id; cleaning afterward would
	//    revoke the new token and delete the settings file the resume needs (and
	//    orphan the old MCP token, whose cleanup closure the new register overwrote).
	s.cleanupMessagingMCP(id)
	s.cleanupHookSettings(id)

	// 3. Compose the target launch spec (registers a fresh MCP token + writes the
	//    per-agent hook settings file for the target identity).
	// Any failure from here on happens AFTER the old runtime was stopped + cleaned,
	// so it must roll back to the previous identity (re-register + re-resume) rather
	// than leave the agent dead with no running row — the same recovery step 5 uses.
	var worktreeNotice worktreeStartNotice
	spec, ae := s.composeSwitchSpecContext(r.Context(), target, resumeID, &worktreeNotice)
	if ae != nil {
		s.rollbackSwitch(r.Context(), w, agent, prev.SessionID, previousSnapshot, errors.New(ae.Message))
		return
	}
	// Carry the requested terminal driver into the resume (validated above); chat
	// targets ignore it. Rollback re-launches the previous identity under the
	// default driver, which is correct for the only shipped terminal driver.
	spec.Driver = req.Driver
	// The composer continued the frozen model choice for an interface/driver-only
	// switch and froze an explicit one for a new backend/model; the launch object
	// records that explicit choice as provenance.
	if target.Backend != agent.Backend || target.Model != agent.Model {
		var err error
		spec.LaunchConfig, err = explicitModelConfig(spec.LaunchConfig)
		if err != nil {
			s.rollbackSwitch(r.Context(), w, agent, prev.SessionID, previousSnapshot, fmt.Errorf("record model choice: %w", err))
			return
		}
	}
	if usePrimer {
		primer, err := s.buildHistoryPrimer(r.Context(), spec, s.switchPrimerTokenBudget())
		if err != nil {
			s.rollbackSwitch(r.Context(), w, agent, prev.SessionID, previousSnapshot, fmt.Errorf("build history primer: %w", err))
			return
		}
		// The current native context owns one bounded bridge. Replacing it keeps
		// subsequent loads safe without modifying or stacking into the persona.
		spec.HistoryPrimer = primer
	}

	// 4. Persist the new identity (agent_id UNCHANGED) so the resume composes the
	//    new interface/backend/model and the card re-renders its badges.
	if err := s.stateStore.WriteAgent(target); err != nil {
		s.rollbackSwitch(r.Context(), w, agent, prev.SessionID, previousSnapshot, fmt.Errorf("persist identity: %w", err))
		return
	}

	// 5. Resume under the target runtime (registry dispatches by target interface).
	handle, err := s.registry.Resume(r.Context(), spec)
	if err != nil {
		s.rollbackSwitch(r.Context(), w, agent, prev.SessionID, previousSnapshot, err)
		return
	}
	target.Fast = handle.Fast
	if err := s.stateStore.WriteAgent(target); err != nil {
		s.rollbackSwitch(r.Context(), w, agent, prev.SessionID, previousSnapshot, err)
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
	s.cleanupMessagingMCP(prevAgent.AgentID)
	s.cleanupHookSettings(prevAgent.AgentID)
	if err := s.stateStore.WriteAgent(prevAgent); err != nil {
		s.failSwitch(w, prevAgent.AgentID, "restore identity: "+err.Error())
		return
	}
	// Compose from the pre-switch snapshot: a target startup may already have
	// written its own metadata (bridge, launch object, contract) before failing.
	spec, ae := s.composeSwitchSpecFrom(rollbackCtx, prevAgent, prevSessionID, previousSnapshot, nil)
	if ae != nil {
		// composeSwitchSpec mints a fresh hook token + MCP registration before it
		// can fail; tear those down so a failed rollback leaves no spoofable
		// registration residue for the stopped agent (INV §4/§15).
		s.teardownAgentRegistration(prevAgent.AgentID)
		s.failSwitch(w, prevAgent.AgentID, "recompose previous: "+ae.Message)
		return
	}
	if _, err := s.registry.Resume(rollbackCtx, spec); err != nil {
		s.teardownAgentRegistration(prevAgent.AgentID)
		s.failSwitch(w, prevAgent.AgentID, err.Error())
		return
	}
	writeAPIError(w, apiError(runtime.CodeSwitchFailedRolledBack, "switch failed, rolled back to previous runtime: "+cause.Error()))
}

// failSwitch records the unrecoverable state (status error) and returns 500
// switch_failed; the agent is recoverable via archive resume (§5.4).
func (s *Server) failSwitch(w http.ResponseWriter, agentID, detail string) {
	if st, err := s.stateStore.ReadStatus(agentID); err == nil {
		st.State = "error"
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
	recreated, warning, wae := s.ensureWorktreeCheckout(ctx, target.Project, snap.Cwd)
	if wae != nil {
		return runtime.LaunchSpec{}, wae
	}
	if recreated && notice != nil {
		row, _ := s.ownedWorktree(target.Project)
		*notice = worktreeStartNotice{Recreated: true, Branch: row.Branch, Warning: warning}
	}
	// Ensure the shared-resources directory before any registration side effect
	// (FS-11.R6/R9). add_dirs/prompt carry over from the frozen snapshot; the env
	// var is recomposed here (INV §2).
	resourceDir, rae := s.ensureProjectResources(target.Project)
	if rae != nil {
		return runtime.LaunchSpec{}, rae
	}

	token := mintHookToken()
	s.rememberHookToken(target.AgentID, token)
	mcpSpec, err := s.registerMessagingMCP(target, token)
	if err != nil {
		s.forgetHookToken(target.AgentID)
		return runtime.LaunchSpec{}, apiError(runtime.CodeInternal, err.Error())
	}
	extraArgs, err := s.composeHookRegistration(target, be.Type)
	if err != nil {
		s.forgetHookToken(target.AgentID)
		s.cleanupMessagingMCP(target.AgentID)
		return runtime.LaunchSpec{}, apiError(runtime.CodeInternal, err.Error())
	}

	spec := runtime.LaunchSpec{
		Agent:              target,
		Generation:         token,
		Cwd:                snap.Cwd,
		AddDirs:            snap.AddDirs,
		SystemPrompt:       snap.SystemPrompt,
		HistoryPrimer:      snap.HistoryPrimer,
		ChuckAware:         &snap.ChuckAware,
		BackendType:        be.Type,
		ModelID:            contract.ProviderModelID(),
		Effort:             target.Effort,
		Fast:               target.Fast,
		Env:                composeChildEnv(be.Type, s.configStore.Home(), be.Env, model.Env, s.hookEnv(target, token), projectResourcesEnv(resourceDir), providerEnv),
		QuotaCheckEnv:      s.quotaCheckEnv(be, model, providerEnv),
		ProviderExecutable: provider.Path,
		ProviderSource:     provider.Source,
		SkipPerms:          snap.SkipPermissions,
		HookToken:          token,
		MCPServers:         []runtime.MCPServerSpec{mcpSpec},
		ExtraArgs:          extraArgs,
		LastSessionID:      resumeID,
		LastContextPct:     snap.LastContextPct,
		LastContext:        snap.LastContext,
		// Switch continues the same logical session, so the frozen federation launch
		// object carries over unchanged — switch never re-resolves the source (§2.5).
		LaunchConfig:      snap.LaunchConfig,
		ExecutionContract: contract,
	}
	return s.applyKnowledgeOverlay(spec), nil
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
