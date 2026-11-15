package server

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// settleQuotaTurn runs under the runtime's executing turn gate, before the
// terminal event or any automatic successor. Config policy and owner state
// therefore become local truth before another provider frame is possible.
func (s *Server) settleQuotaTurn(agentID, generation, turnID string, evidence *runtime.QuotaEvidence, successful bool) error {
	if evidence == nil {
		// A due dispatch or quota Cancel can bump the revision between the read
		// and the write; re-read and decide again a bounded number of times
		// (TS-01.R49, INV §8) instead of reporting a storage failure.
		var err error
		for attempt := 0; attempt < quotaSettleAttempts; attempt++ {
			if err = s.settleQuotaTurnWithoutEvidence(agentID, generation, turnID, successful); !errors.Is(err, state.ErrQuotaConflict) {
				return err
			}
		}
		return err
	}
	return s.settleQuotaTurnEvidence(agentID, generation, turnID, evidence)
}

const quotaSettleAttempts = 3

func (s *Server) settleQuotaTurnWithoutEvidence(agentID, generation, turnID string, successful bool) error {
	if successful {
		q, err := s.stateStore.GetQuota(agentID)
		if errors.Is(err, state.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if q.State == state.QuotaClaimed && q.ProviderStarted && q.ContinuedGeneration == generation && q.ContinuedTurn == turnID {
			return s.confirmQuotaSuccess(agentID, generation, turnID)
		}
		if q.State == state.QuotaContinued {
			return nil
		}
		if err := s.stateStore.ResolveQuota(agentID, generation, turnID); err != nil {
			return err
		}
		if q.Owner.Kind == "room" {
			return s.publishQuotaAgents([]string{agentID})
		}
		return nil
	}
	q, err := s.stateStore.GetQuota(agentID)
	if errors.Is(err, state.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if q.State == state.QuotaClaimed && q.ProviderStarted && q.ContinuedGeneration == generation && q.ContinuedTurn == turnID {
		return s.stateStore.HoldQuota(q.AgentID, q.EpisodeID, q.Revision, q.AttemptToken, "non_quota_failure")
	}
	if q.ManualGeneration == generation && q.ManualTurn == turnID &&
		q.State != state.QuotaCancelled && q.State != state.QuotaDisabled && q.State != state.QuotaContinued {
		return s.stateStore.HoldQuotaManualTurn(agentID, generation, turnID, q.Revision)
	}
	return nil
}

func (s *Server) settleQuotaTurnEvidence(agentID, generation, turnID string, evidence *runtime.QuotaEvidence) error {
	s.configMu.Lock()
	defer s.configMu.Unlock()
	cfg, err := s.configStore.ReadConfig()
	if err != nil {
		return err
	}
	observed := time.Now().UTC()
	status := state.QuotaUnknownReset
	var resetAt, dueAt *time.Time
	if evidence.ResetAt != nil && evidence.ResetAt.After(observed) {
		// Continuation waits past the displayed (rounded) reset minute (FS-01.R48).
		reset, due := evidence.ResetAt.UTC(), evidence.ResetAt.UTC().Add(state.QuotaResetBuffer)
		resetAt, dueAt = &reset, &due
		status = state.QuotaScheduled
	}
	if !cfg.AutoContinue {
		status, dueAt = state.QuotaDisabled, nil
	}
	sum := sha256.Sum256([]byte(generation + "\x00" + turnID + "\x00" + evidence.FailureID))
	q, err := s.stateStore.ObserveOwnedQuota(state.QuotaObservation{
		AgentID: agentID, EpisodeID: "quota_" + hex.EncodeToString(sum[:]),
		SourceGeneration: generation, ExecutingTurn: turnID, ObservedAt: observed, State: status,
		ResetAt: resetAt, DueAt: dueAt, ResetSource: resetSource(resetAt, evidence.ResetSource),
		// Without a reliable reset, bounded isolated checks may authorize the
		// continuation; explicit credit/plan/spend-cap errors stay manual.
		Check: status == state.QuotaUnknownReset && !evidence.Intervention,
	})
	if err != nil {
		return err
	}
	if q.Owner.Kind == "room" {
		if detail, err := s.stateStore.ReadThinkTank(q.Owner.RoomID); err == nil {
			s.publishThinkTankUpdate(detail)
		} else {
			return err
		}
	}
	return nil
}

// resetSource keeps only the closed provenance enum of an accepted reset.
func resetSource(resetAt *time.Time, source string) string {
	if resetAt == nil || source != "structured" && source != "text" && source != "snapshot" {
		return ""
	}
	return source
}

func (s *Server) quotaBlocksAgent(agentID string) (bool, error) {
	decision, err := s.stateStore.DecideTurnAdmission(state.TurnAdmissionRequest{AgentID: agentID, Check: state.AdmitWithQuotaGate})
	return err == nil && !decision.Allowed, err
}
