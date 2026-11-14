package server

import (
	"context"
	"errors"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

func (s *Server) dispatchQuotaContinuations(ctx context.Context) {
	if s.registry == nil {
		return
	}
	s.startQuotaCleanup(ctx)
	due, err := s.stateStore.DueQuotaIntentsAfter(time.Now().UTC(), s.quotaDueAfter, s.quotaDueAfterAgent, taskDispatchBatch)
	if err != nil {
		s.log.Error("read due quota continuations", "err", err)
		return
	}
	for _, q := range due {
		select {
		case s.quotaStartSlots <- struct{}{}:
		default:
			return
		}
		s.quotaDueAfter, s.quotaDueAfterAgent = *q.DueAt, q.AgentID
		claimed, ok := s.claimDueQuota(q)
		if !ok {
			<-s.quotaStartSlots
			continue
		}
		// Only admission from the schedule reads the snapshot; a deferred
		// retry (busy, store failure) re-claims every dispatch tick and must
		// not poll the provider (TS-04.R111, FS-09.R83).
		precheck := q.State == state.QuotaScheduled
		go func(q state.QuotaIntent) {
			defer func() { <-s.quotaStartSlots }()
			if precheck && s.postponeToSnapshotReset(q) {
				return
			}
			if q.NextAction == state.QuotaActionCheck {
				s.runQuotaCheck(ctx, q)
				return
			}
			s.startQuotaContinuation(ctx, q)
		}(claimed)
	}
	if len(due) < taskDispatchBatch {
		s.quotaDueAfter, s.quotaDueAfterAgent = time.Time{}, ""
	}
}

// Claims follow the indexed due order; provider/resume work uses bounded worker
// slots after this short admission, as ordinary task dispatch does.
func (s *Server) claimDueQuota(q state.QuotaIntent) (state.QuotaIntent, bool) {
	keys, err := s.quotaAdmissionKeys(q)
	if err != nil {
		s.log.Error("read quota admission owner", "err", err)
		if errors.Is(err, state.ErrNotFound) {
			if err := s.stateStore.HoldDueQuotaWithoutOwner(q); err == nil {
				_, _ = s.stateMgr.Touch(q.AgentID)
			} else if !errors.Is(err, state.ErrQuotaConflict) {
				s.log.Error("hold due quota continuation with unavailable owner", "err", err)
			}
		}
		return state.QuotaIntent{}, false
	}
	release, ok := s.claimQuotaActions(keys)
	if !ok {
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
	if err := s.stateStore.ClaimQuota(q.AgentID, q.EpisodeID, q.Revision, mintHookToken()); err != nil {
		return state.QuotaIntent{}, false
	}
	fresh, err := s.stateStore.GetQuota(q.AgentID)
	if err != nil {
		s.log.Error("read admitted quota continuation", "err", err)
		return state.QuotaIntent{}, false
	}
	return fresh, true
}

// deferQuotaPolicyFailure returns an unstarted claim to the due sweep with a
// visible reason when the live policy cannot be reread, rather than stranding
// it until restart (FS-01.R44, TS-10.R40/R43). Callers hold configMu.
func (s *Server) deferQuotaPolicyFailure(q state.QuotaIntent, err error) {
	s.log.Error("read quota continuation policy", "err", err)
	if q.State != state.QuotaClaimed {
		return
	}
	if err := s.stateStore.DeferQuota(q.AgentID, q.EpisodeID, q.Revision, q.AttemptToken, time.Now().UTC().Add(taskDispatchInterval), "store_failure"); err != nil && !errors.Is(err, state.ErrQuotaConflict) {
		s.log.Error("defer quota claim after policy read failure", "err", err)
	}
	_, _ = s.stateMgr.Touch(q.AgentID)
}

func (s *Server) startQuotaContinuation(ctx context.Context, q state.QuotaIntent) {
	if q.NextAction == state.QuotaActionCheck {
		return // a check never wakes the interrupted conversation
	}
	keys, err := s.quotaAdmissionKeys(q)
	if err != nil {
		s.log.Error("read quota admission owner", "err", err)
		if q.State == state.QuotaClaimed && errors.Is(err, state.ErrNotFound) {
			if holdErr := s.stateStore.HoldQuota(q.AgentID, q.EpisodeID, q.Revision, q.AttemptToken, "owner_unavailable"); holdErr != nil && !errors.Is(holdErr, state.ErrQuotaConflict) {
				s.log.Error("hold quota continuation with unavailable owner", "err", holdErr)
			}
		}
		return
	}
	release, ok := s.claimQuotaActions(keys)
	if !ok {
		if q.State == state.QuotaClaimed {
			if err := s.stateStore.DeferQuota(q.AgentID, q.EpisodeID, q.Revision, q.AttemptToken, time.Now().UTC().Add(taskDispatchInterval), "busy"); err != nil && !errors.Is(err, state.ErrQuotaConflict) {
				s.log.Error("defer quota continuation behind manual action", "err", err)
			}
		}
		return
	}
	defer release()
	policyLocked := false
	lockPolicy := func() {
		s.quotaControlMu.Lock()
		s.configMu.Lock()
		policyLocked = true
	}
	unlockPolicy := func() {
		if policyLocked {
			policyLocked = false
			s.configMu.Unlock()
			s.quotaControlMu.Unlock()
		}
	}
	// The live policy lock spans eligibility, claim, and the turn's durable
	// provider boundary. Disable/cancel and automatic admission have one winner.
	lockPolicy()
	defer unlockPolicy()
	cfg, err := s.configStore.ReadConfig()
	if err != nil {
		s.deferQuotaPolicyFailure(q, err)
		return
	}
	if !cfg.AutoContinue {
		return
	}
	if q.State != state.QuotaClaimed {
		if err := s.stateStore.ClaimQuota(q.AgentID, q.EpisodeID, q.Revision, mintHookToken()); err != nil {
			return
		}
		q, err = s.stateStore.GetQuota(q.AgentID)
		if err != nil {
			s.log.Error("read claimed quota", "err", err)
			return
		}
	} else {
		fresh, err := s.stateStore.GetQuota(q.AgentID)
		if err != nil || fresh.State != state.QuotaClaimed || fresh.EpisodeID != q.EpisodeID || fresh.Revision != q.Revision || fresh.AttemptToken != q.AttemptToken {
			return
		}
	}
	defer func() { _, _ = s.stateMgr.Touch(q.AgentID) }()
	hold := func(reason string) {
		fresh, err := s.stateStore.GetQuota(q.AgentID)
		if err == nil && fresh.EpisodeID == q.EpisodeID && fresh.AttemptToken == q.AttemptToken {
			if err := s.stateStore.HoldQuota(q.AgentID, q.EpisodeID, fresh.Revision, q.AttemptToken, reason); err != nil {
				s.log.Error("hold quota continuation", "err", err)
			}
		}
	}
	deferBusy := func() {
		if err := s.stateStore.DeferQuota(q.AgentID, q.EpisodeID, q.Revision, q.AttemptToken, time.Now().UTC().Add(taskDispatchInterval), "busy"); err != nil {
			s.log.Error("defer quota continuation", "err", err)
		}
	}
	a, err := s.stateStore.ReadAgent(q.AgentID)
	if err != nil || a.Archived || a.Interface != "chat" {
		hold("archived")
		return
	}
	if ae := s.projectArchiveGate(a.Project, "project is archived"); ae != nil {
		hold("project_ineligible")
		return
	}
	if q.CheckReady && !s.quotaCheckStillMatches(q) {
		// A check authorizes only the provider/account/model selection it
		// tested; an edited or unprovable selection stays manual (TS-10.R46).
		hold("check_unsupported")
		return
	}
	started, passSettled := false, false
	if eligible, err := s.quotaWorkEligible(q); err != nil {
		if errors.Is(err, state.ErrQuotaOwner) {
			hold("owner_unavailable")
		} else {
			s.log.Error("read quota continuation eligibility", "agent", q.AgentID, "err", err)
			hold("store_failure")
		}
		return
	} else if !eligible {
		deferBusy()
		return
	}
	start := func() error {
		kind := state.ActivationKindQuotaContinue
		before := func(turn string) error {
			return s.stateStore.AdmitQuotaTurn(q, s.registry.Generation(q.AgentID), turn)
		}
		if q.Owner.Kind == "room" {
			kind = state.ActivationKindQuotaThinkTank
		}
		var begun *state.ThinkTankAttempt
		if q.Owner.Kind == "room" {
			before = func(turn string) error {
				a, err := s.beginQuotaRoomTurn(q, s.registry.Generation(q.AgentID), turn)
				if err == nil {
					begun = &a
				}
				return err
			}
		}
		var err error
		started, err = s.registry.StartActivation(ctx, q.AgentID, kind, before)
		if !started {
			s.abandonThinkTankStart(q.AgentID, begun, err)
		}
		return err
	}
	if _, err := s.stateStore.ReadRunning(q.AgentID); err == nil {
		if !s.claimLifecycle(q.AgentID) {
			deferBusy()
			return
		}
		defer s.releaseLifecycle(q.AgentID)
		if ae := s.acquireAgentStart(a.Project, q.AgentID); ae != nil {
			deferBusy()
			return
		}
		defer s.releaseAgentStart(a.Project, q.AgentID)
		if err := start(); err != nil {
			hold("owner_unavailable")
			return
		}
	} else if errors.Is(err, state.ErrNotFound) {
		if q.Owner.Kind == "task" {
			admitted, err := s.stateStore.AdmitQuotaTaskResume(q, s.taskConcurrencyBudget())
			if err != nil {
				if errors.Is(err, state.ErrQuotaOwner) {
					hold("owner_unavailable")
				} else {
					hold("store_failure")
				}
				return
			}
			if !admitted {
				deferBusy()
				return
			}
		}
		unlockPolicy()
		if ae := s.resumeSessionWithHooks(ctx, q.AgentID, resumeOverride{}, nil, func() error {
			lockPolicy()
			cfg, err := s.configStore.ReadConfig()
			if err != nil {
				s.deferQuotaPolicyFailure(q, err)
				passSettled = true
				return nil
			}
			if !cfg.AutoContinue {
				passSettled = true // disabling automatic continuation owns settlement
				return nil
			}
			current, err := s.stateStore.GetQuota(q.AgentID)
			if err != nil || current.Revision != q.Revision || current.EpisodeID != q.EpisodeID ||
				current.AttemptToken != q.AttemptToken || current.State != state.QuotaClaimed {
				passSettled = true // another action changed the episode
				return nil
			}
			if q.CheckReady && !s.quotaCheckStillMatches(q) {
				hold("check_unsupported")
				passSettled = true
				return nil
			}
			return start()
		}); ae != nil {
			if ae.Code == runtime.CodeConflict {
				deferBusy()
			} else {
				hold("provider_start_failed")
			}
			return
		}
	} else {
		hold("store_failure")
		return
	}
	if !started && !passSettled {
		deferBusy()
	}
}

// quotaWorkEligible has no owner/runtime side effects. Both check admission
// and continuation use the same current owner, capacity and busy predicates.
func (s *Server) quotaWorkEligible(q state.QuotaIntent) (bool, error) {
	if _, err := s.stateStore.ReadRunning(q.AgentID); err == nil {
		status, err := s.stateStore.ReadStatus(q.AgentID)
		if err != nil {
			return false, err
		}
		if status.State != "idle" && status.State != "error" {
			return false, nil
		}
	} else if !errors.Is(err, state.ErrNotFound) {
		return false, err
	}
	return s.stateStore.QuotaWorkEligible(q, s.taskConcurrencyBudget())
}

func (s *Server) beginQuotaRoomTurn(q state.QuotaIntent, generation, turn string) (state.ThinkTankAttempt, error) {
	d, err := s.stateStore.ReadThinkTank(q.Owner.RoomID)
	if err != nil {
		return state.ThinkTankAttempt{}, err
	}
	a, err := s.stateStore.BeginThinkTankAttempt(state.ThinkTankBegin{
		RoomID: q.Owner.RoomID, Revision: d.Room.Revision, AgentID: q.AgentID, Turn: q.Owner.Turn,
		Generation: generation, TurnID: turn,
		QuotaContinuation: &state.ThinkTankQuotaContinuation{EpisodeID: q.EpisodeID, Revision: q.Revision,
			AttemptToken: q.AttemptToken, SourceAttemptID: q.Owner.AttemptID},
	})
	if err != nil {
		return state.ThinkTankAttempt{}, err
	}
	name, project := "", ""
	for _, m := range d.Members {
		if m.AgentID == q.AgentID {
			name, project = m.AgentName, m.Project
		}
	}
	s.beginThinkTankCapture(q.AgentID, a, name, project)
	return a, nil
}

// postponeToSnapshotReset is TS-04.R111's pre-check: immediately before a
// claimed continuation or check starts, a fresh snapshot that still reports an
// applicable window exhausted with a strictly later reset moves the wait there,
// charging no check and waking nothing. Any other outcome changes nothing.
func (s *Server) postponeToSnapshotReset(q state.QuotaIntent) bool {
	spec, err := s.quotaCheckSpec(q.AgentID)
	if err != nil {
		return false
	}
	reset := s.registry.QuotaSnapshotReset(q.AgentID, spec.ModelID, q.ObservedAt)
	if reset == nil || (q.ResetAt != nil && !reset.After(*q.ResetAt)) {
		return false
	}
	if err := s.stateStore.PostponeQuotaToSnapshotReset(q.AgentID, q.EpisodeID, q.Revision, q.AttemptToken, *reset); err != nil {
		if !errors.Is(err, state.ErrQuotaConflict) {
			s.log.Error("postpone quota continuation to snapshot reset", "err", err)
		}
		return false
	}
	_, _ = s.stateMgr.Touch(q.AgentID)
	if q.Owner.Kind == "room" {
		if detail, err := s.stateStore.ReadThinkTank(q.Owner.RoomID); err == nil {
			s.publishThinkTankUpdate(detail)
		}
	}
	return true
}
