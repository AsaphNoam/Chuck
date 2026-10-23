package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// quotaCleanupRetry bounds how often an unresolved exact probe cleanup spawns
// another short-lived adapter process.
const quotaCleanupRetry = 10 * time.Minute

// quotaCheckEnv is a process's provider environment without hook, resource or
// coordination layers: the selection an isolated check may reuse.
func (s *Server) quotaCheckEnv(be config.Backend, model config.Model, providerEnv map[string]string) []string {
	return composeChildEnv(be.Type, s.configStore.Home(), be.Env, model.Env, nil, nil, providerEnv)
}

// quotaCheckSpec returns the selection the interrupted work's continuation
// would use (TS-04.R97, TS-10.R46): the live original process's frozen
// selection while it runs, otherwise what a resume derives from the catalog
// now. Backend Settings edits never retarget a running process, so the catalog
// is not consulted for a live one.
func (s *Server) quotaCheckSpec(agentID string) (runtime.QuotaCheckSpec, error) {
	a, err := s.stateStore.ReadAgent(agentID)
	if err != nil {
		return runtime.QuotaCheckSpec{}, err
	}
	if _, err := s.stateStore.ReadRunning(agentID); err == nil {
		live, ok := s.registry.QuotaCheckSelection(agentID)
		if !ok {
			return runtime.QuotaCheckSpec{}, errors.New("quota check: live selection unknown")
		}
		live.Effort, live.Fast = a.Effort, a.Fast
		return live, nil
	} else if !errors.Is(err, state.ErrNotFound) {
		return runtime.QuotaCheckSpec{}, err
	}
	snap, err := s.stateStore.ReadSession(agentID)
	if err != nil {
		return runtime.QuotaCheckSpec{}, err
	}
	backends, err := s.configStore.ReadBackends()
	if errors.Is(err, config.ErrNotFound) || errors.Is(err, config.ErrCorrupt) {
		backends, err = config.DefaultBackends(), nil
	}
	if err != nil {
		return runtime.QuotaCheckSpec{}, err
	}
	be, ok := backends.Backends[a.Backend]
	if !ok {
		return runtime.QuotaCheckSpec{}, errors.New("quota check: unknown backend")
	}
	model, ok := be.Models[a.Model]
	if !ok {
		return runtime.QuotaCheckSpec{}, errors.New("quota check: unknown model")
	}
	// Probe the selection the continuation would realize, not today's catalogue
	// value; an undecidable one refuses before any probe (FS-01.R53).
	contract, ae := continuedContract(snap, a, be, model, false)
	if ae != nil {
		return runtime.QuotaCheckSpec{}, ae
	}
	providerEnv, _, ae := providerLaunchLayer(be, model)
	if ae != nil {
		return runtime.QuotaCheckSpec{}, ae
	}
	return runtime.QuotaCheckSpec{BackendType: be.Type, ModelID: contract.ProviderModelID(), Effort: a.Effort, Fast: a.Fast,
		Env: s.quotaCheckEnv(be, model, providerEnv), SourceCwd: snap.Cwd}, nil
}

// quotaSelectionKey fingerprints a check selection so a later continuation
// can prove it still targets the quota that was checked.
func quotaSelectionKey(spec runtime.QuotaCheckSpec) string {
	env := append([]string(nil), spec.Env...)
	sort.Strings(env)
	h := sha256.New()
	for _, part := range append([]string{spec.BackendType, spec.ModelID, spec.Effort, strconv.FormatBool(spec.Fast), spec.SourceCwd}, env...) {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// quotaCheckStillMatches reports whether the continuation for a ready check
// would run under the exact selection that check used. A restart forgets the
// checked selection, so equivalence is then unprovable.
func (s *Server) quotaCheckStillMatches(q state.QuotaIntent) bool {
	spec, err := s.quotaCheckSpec(q.AgentID)
	if err != nil {
		return false
	}
	s.quotaCheckMu.Lock()
	defer s.quotaCheckMu.Unlock()
	return s.quotaChecked[q.AgentID] == q.EpisodeID+"\x00"+quotaSelectionKey(spec)
}

func (s *Server) markQuotaCheck(agentID string, running bool) {
	s.quotaCheckMu.Lock()
	defer s.quotaCheckMu.Unlock()
	if s.quotaChecks == nil {
		s.quotaChecks = map[string]bool{}
	}
	if running {
		s.quotaChecks[agentID] = true
	} else {
		delete(s.quotaChecks, agentID)
	}
}

// runQuotaCheck admits and runs one claimed check outside the interrupted
// conversation. Eligibility and the allowance charge commit under the same
// locks as continuation admission; the provider check runs after release.
func (s *Server) runQuotaCheck(ctx context.Context, q state.QuotaIntent) {
	defer func() { _, _ = s.stateMgr.Touch(q.AgentID) }()
	spec, specErr := s.quotaCheckSpec(q.AgentID)
	started, ok := s.admitQuotaCheck(q, spec, specErr)
	if !ok {
		return
	}
	s.markQuotaCheck(q.AgentID, true)
	defer s.markQuotaCheck(q.AgentID, false)
	checkCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Any accepted cancellation, disable, supersession or deletion moves the
	// row off this exact claim; stop the provider check promptly when it does.
	go func() {
		tick := time.NewTicker(500 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-checkCtx.Done():
				return
			case <-tick.C:
				fresh, err := s.stateStore.GetQuota(q.AgentID)
				if errors.Is(err, state.ErrNotFound) || err == nil && (fresh.State != state.QuotaClaimed || fresh.AttemptToken != started.AttemptToken) {
					cancel()
					return
				}
			}
		}
	}()
	cleanupSpec := spec
	spec.OnSessionCreated = func(sessionID string) error {
		fresh, err := s.stateStore.GetQuota(q.AgentID)
		if err != nil || fresh.AttemptToken != started.AttemptToken || fresh.State != state.QuotaClaimed {
			return state.ErrQuotaConflict
		}
		s.quotaCheckMu.Lock()
		if s.quotaCleanupSpecs == nil {
			s.quotaCleanupSpecs = map[string]runtime.QuotaCheckSpec{}
		}
		s.quotaCleanupSpecs[q.AgentID+"\x00"+sessionID] = cleanupSpec
		s.quotaCheckMu.Unlock()
		return s.stateStore.RecordQuotaCheckSession(q.AgentID, fresh.EpisodeID, fresh.Revision, fresh.AttemptToken, sessionID)
	}
	out := s.registry.RunQuotaCheck(checkCtx, spec)
	// Resolve a disposed probe first so its result never appears beside a
	// stale cleanup reference.
	if out.Cleaned && out.SessionID != "" {
		s.resolveQuotaCleanup(q.AgentID, out.SessionID)
	}
	fresh, err := s.stateStore.GetQuota(q.AgentID)
	if err == nil && fresh.AttemptToken == started.AttemptToken {
		if out.Outcome == state.QuotaCheckReady {
			s.quotaCheckMu.Lock()
			if s.quotaChecked == nil {
				s.quotaChecked = map[string]string{}
			}
			s.quotaChecked[q.AgentID] = fresh.EpisodeID + "\x00" + quotaSelectionKey(spec)
			s.quotaCheckMu.Unlock()
		}
		if err := s.stateStore.FinishQuotaCheck(q.AgentID, fresh.EpisodeID, fresh.Revision, fresh.AttemptToken, state.QuotaCheckResult{
			Outcome: out.Outcome, ResetAt: out.ResetAt, ResetSource: out.ResetSource, Intervention: out.Intervention, Cleaned: out.Cleaned,
		}); err != nil && !errors.Is(err, state.ErrQuotaConflict) {
			s.log.Error("record quota check", "err", err)
		}
	}
}

func (s *Server) resolveQuotaCleanup(agentID, sessionID string) {
	if err := s.stateStore.ResolveQuotaCheckCleanup(agentID, sessionID); err != nil {
		s.log.Error("resolve quota check cleanup", "err", err)
		return
	}
	s.quotaCheckMu.Lock()
	delete(s.quotaCleanupSpecs, agentID+"\x00"+sessionID)
	delete(s.quotaCleanupTried, agentID+"\x00"+sessionID)
	s.quotaCheckMu.Unlock()
}

func (s *Server) admitQuotaCheck(q state.QuotaIntent, spec runtime.QuotaCheckSpec, specErr error) (state.QuotaIntent, bool) {
	keys, err := s.quotaAdmissionKeys(q)
	if err != nil {
		_ = s.stateStore.HoldQuota(q.AgentID, q.EpisodeID, q.Revision, q.AttemptToken, "owner_unavailable")
		return state.QuotaIntent{}, false
	}
	release, ok := s.claimQuotaActions(keys)
	if !ok {
		// A busy deferral creates no provider check and spends no allowance.
		if err := s.stateStore.DeferQuota(q.AgentID, q.EpisodeID, q.Revision, q.AttemptToken, time.Now().UTC().Add(taskDispatchInterval), "busy"); err != nil && !errors.Is(err, state.ErrQuotaConflict) {
			s.log.Error("defer quota check behind manual action", "err", err)
		}
		return state.QuotaIntent{}, false
	}
	defer release()
	s.quotaControlMu.Lock()
	defer s.quotaControlMu.Unlock()
	s.configMu.Lock()
	defer s.configMu.Unlock()
	cfg, err := s.configStore.ReadConfig()
	if err != nil || !cfg.AutoContinue {
		return state.QuotaIntent{}, false
	}
	fresh, err := s.stateStore.GetQuota(q.AgentID)
	if err != nil || fresh.State != state.QuotaClaimed || fresh.EpisodeID != q.EpisodeID || fresh.AttemptToken != q.AttemptToken || fresh.NextAction != state.QuotaActionCheck {
		return state.QuotaIntent{}, false
	}
	hold := func(reason string) (state.QuotaIntent, bool) {
		if err := s.stateStore.HoldQuota(fresh.AgentID, fresh.EpisodeID, fresh.Revision, fresh.AttemptToken, reason); err != nil && !errors.Is(err, state.ErrQuotaConflict) {
			s.log.Error("hold quota check", "err", err)
		}
		return state.QuotaIntent{}, false
	}
	a, err := s.stateStore.ReadAgent(q.AgentID)
	if err != nil || a.Archived || a.Interface != "chat" {
		return hold("archived")
	}
	if ae := s.projectArchiveGate(a.Project, "project is archived"); ae != nil {
		return hold("project_ineligible")
	}
	if specErr != nil || spec.BackendType != "claude-acp" && spec.BackendType != "codex-acp" {
		return hold("check_unsupported")
	}
	deferBusy := func() (state.QuotaIntent, bool) {
		if err := s.stateStore.DeferQuota(fresh.AgentID, fresh.EpisodeID, fresh.Revision, fresh.AttemptToken, time.Now().UTC().Add(taskDispatchInterval), "busy"); err != nil && !errors.Is(err, state.ErrQuotaConflict) {
			s.log.Error("defer ineligible quota check", "err", err)
		}
		return state.QuotaIntent{}, false
	}
	// Hold the lifecycle and archive gates only through eligibility/charging,
	// never during the isolated provider request.
	if !s.claimLifecycle(q.AgentID) {
		return deferBusy()
	}
	defer s.releaseLifecycle(q.AgentID)
	if ae := s.acquireAgentStart(a.Project, q.AgentID); ae != nil {
		return deferBusy()
	}
	defer s.releaseAgentStart(a.Project, q.AgentID)
	if eligible, err := s.quotaWorkEligible(fresh); err != nil {
		if errors.Is(err, state.ErrQuotaOwner) {
			return hold("owner_unavailable")
		}
		s.log.Error("read quota check eligibility", "agent", q.AgentID, "err", err)
		return hold("store_failure")
	} else if !eligible {
		return deferBusy()
	}
	if fresh.CheckSession != "" {
		// One exact cleanup reference at a time; the cleanup pass resolves it.
		if err := s.stateStore.DeferQuota(fresh.AgentID, fresh.EpisodeID, fresh.Revision, fresh.AttemptToken, time.Now().UTC().Add(quotaCleanupRetry), "busy"); err != nil && !errors.Is(err, state.ErrQuotaConflict) {
			s.log.Error("defer quota check behind cleanup", "err", err)
		}
		return state.QuotaIntent{}, false
	}
	if eligible, err := s.stateStore.StartEligibleQuotaCheck(fresh, spec.BackendType, s.taskConcurrencyBudget()); err != nil {
		if errors.Is(err, state.ErrQuotaOwner) {
			return hold("owner_unavailable")
		}
		if !errors.Is(err, state.ErrQuotaConflict) {
			s.log.Error("admit quota check", "agent", q.AgentID, "err", err)
			return hold("store_failure")
		}
		return state.QuotaIntent{}, false
	} else if !eligible {
		return deferBusy()
	}
	started, err := s.stateStore.GetQuota(q.AgentID)
	if err != nil {
		return state.QuotaIntent{}, false
	}
	return started, true
}

// cleanupQuotaChecks retries exact disposal of recorded probe sessions that no
// running check owns, never loading, prompting or scanning other history.
func (s *Server) cleanupQuotaChecks(ctx context.Context) {
	s.quotaCheckMu.Lock()
	pending, err := s.stateStore.PendingQuotaCheckCleanupsAfter(state.MaxQuotaDueBatch, s.quotaCleanupCursor)
	if err == nil {
		if len(pending) == 0 {
			s.quotaCleanupCursor = ""
		} else {
			s.quotaCleanupCursor = pending[len(pending)-1].AgentID
		}
	}
	s.quotaCheckMu.Unlock()
	if err != nil {
		s.log.Error("read pending quota check cleanups", "err", err)
		return
	}
	now := time.Now()
	for _, q := range pending {
		s.quotaCheckMu.Lock()
		if s.quotaCleanupTried == nil {
			s.quotaCleanupTried = map[string]time.Time{}
		}
		key := q.AgentID + "\x00" + q.CheckSession
		skip := s.quotaChecks[q.AgentID] || now.Sub(s.quotaCleanupTried[key]) < quotaCleanupRetry
		if !skip {
			s.quotaCleanupTried[key] = now
		}
		spec, known := s.quotaCleanupSpecs[key]
		s.quotaCheckMu.Unlock()
		if skip {
			continue
		}
		if !known {
			// After a restart the creating selection is gone; the current one
			// can still dispose the exact recorded session or fail visibly.
			var err error
			if spec, err = s.quotaCheckSpec(q.AgentID); err != nil {
				continue
			}
		}
		spec.BackendType = q.CheckBackend
		if err := s.registry.CleanupQuotaCheckSession(ctx, spec, q.CheckSession); err != nil {
			s.log.Warn("quota check cleanup failed; will retry", "agent", q.AgentID, "err", err)
			continue
		}
		s.resolveQuotaCleanup(q.AgentID, q.CheckSession)
		_, _ = s.stateMgr.Touch(q.AgentID)
	}
}
