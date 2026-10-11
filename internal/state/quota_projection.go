package state

import (
	"errors"
	"time"
)

// QuotaProjection is the only quota shape exposed through HTTP and SSE. Owner
// authority, provider data and attempt tokens stay inside the control plane.
type QuotaProjection struct {
	EpisodeID         string     `json:"episode_id"`
	Revision          int64      `json:"revision"`
	ObservedAt        time.Time  `json:"observed_at"`
	ResetAt           *time.Time `json:"reset_at"`
	DueAt             *time.Time `json:"due_at"`
	ContinuationState string     `json:"continuation_state"`
	ReasonCode        string     `json:"reason_code,omitempty"`
}

func (s *Store) QuotaProjection(agentID string) (*QuotaProjection, error) {
	q, err := s.GetQuota(agentID)
	if errors.Is(err, ErrNotFound) || err == nil && q.State == QuotaContinued {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	state := q.State
	if state == QuotaDeferred {
		state = "waiting"
	} else if state == QuotaClaimed {
		state = "starting"
	}
	return &QuotaProjection{EpisodeID: q.EpisodeID, Revision: q.Revision, ObservedAt: q.ObservedAt,
		ResetAt: q.ResetAt, DueAt: q.DueAt, ContinuationState: state, ReasonCode: q.ReasonCode}, nil
}
