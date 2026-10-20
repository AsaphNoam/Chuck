package server

import (
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// historyHealthStore adapts the state store to the journal's persisted
// history-health contract (TS-02.R52).
type historyHealthStore struct{ store *state.Store }

func (h historyHealthStore) ReadHistoryHealth(agentID string) (runtime.HistoryHealth, error) {
	got, err := h.store.ReadHistoryHealth(agentID)
	return runtime.HistoryHealth(got), err
}

func (h historyHealthStore) WriteHistoryHealth(agentID string, health runtime.HistoryHealth) error {
	return h.store.WriteHistoryHealth(agentID, state.HistoryHealth(health))
}
