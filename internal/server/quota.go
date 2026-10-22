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
			return s.stateStore.ResolveQuota(agentID, generation, turnID)
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
		if q.ManualGeneration == generation && q.ManualTurn == turnID {
			return s.stateStore.HoldQuotaManualTurn(agentID, generation, turnID)
		}
		return nil
	}
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
	if resetAt == nil || source != "structured" && source != "text" {
		return ""
	}
	return source
}

func (s *Server) quotaBlocksAgent(agentID string) (bool, error) {
	q, err := s.stateStore.GetQuota(agentID)
	if errors.Is(err, state.ErrNotFound) {
		return false, nil
	}
	return err == nil && q.State != state.QuotaContinued && q.State != state.QuotaCancelled, err
}
