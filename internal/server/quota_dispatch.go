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
	s.cleanupQuotaChecks(ctx)
	due, err := s.stateStore.DueQuotaIntents(time.Now().UTC(), taskDispatchBatch)
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
		claimed, ok := s.claimDueQuota(q)
		if !ok {
			<-s.quotaStartSlots
			continue
		}
		go func(q state.QuotaIntent) {
			defer func() { <-s.quotaStartSlots }()
			if q.NextAction == state.QuotaActionCheck {
				s.runQuotaCheck(ctx, q)
				return
			}
			s.startQuotaContinuation(ctx, q)
		}(claimed)
	}
}

// Claims follow the indexed due order; provider/resume work uses bounded worker
// slots after this short admission, as ordinary task dispatch does.
func (s *Server) claimDueQuota(q state.QuotaIntent) (state.QuotaIntent, bool) {
	keys, err := s.quotaAdmissionKeys(q)
	if err != nil {
		s.log.Error("read quota admission owner", "err", err)
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

func (s *Server) startQuotaContinuation(ctx context.Context, q state.QuotaIntent) {
	if q.NextAction == state.QuotaActionCheck {
		return // a check never wakes the interrupted conversation
	}
	keys, err := s.quotaAdmissionKeys(q)
	if err != nil {
		s.log.Error("read quota admission owner", "err", err)
		if q.State == state.QuotaClaimed {
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
	s.quotaControlMu.Lock()
	defer s.quotaControlMu.Unlock()
	// The live policy lock spans eligibility, claim, and the turn's durable
	// provider boundary. Disable/cancel and automatic admission have one winner.
	s.configMu.Lock()
	defer s.configMu.Unlock()
	cfg, err := s.configStore.ReadConfig()
	if err != nil || !cfg.AutoContinue {
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
	started := false
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
		if ae := s.resumeSessionWithHooks(ctx, q.AgentID, resumeOverride{}, nil, start); ae != nil {
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
	if !started {
		deferBusy()
	}
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
