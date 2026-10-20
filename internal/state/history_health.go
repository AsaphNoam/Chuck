package state

import (
	"database/sql"
	"errors"
	"time"
)

// HistoryHealth is one conversation's persisted history health (TS-02.R52).
// An empty State means no row: legacy unknown, never assumed healthy.
type HistoryHealth struct {
	State       string
	Reason      string
	ReservedSeq int64
	OpenWriter  bool
}

// ReadHistoryHealth returns agentID's history health, or the zero value when
// none was ever recorded.
func (s *Store) ReadHistoryHealth(agentID string) (HistoryHealth, error) {
	var h HistoryHealth
	var open int
	err := s.db.QueryRow(`SELECT state, reason, reserved_seq, open_writer FROM conversation_health WHERE agent_id = ?`, agentID).
		Scan(&h.State, &h.Reason, &h.ReservedSeq, &open)
	if errors.Is(err, sql.ErrNoRows) {
		return HistoryHealth{}, nil
	}
	h.OpenWriter = open == 1
	return h, err
}

// WriteHistoryHealth records agentID's history health.
func (s *Store) WriteHistoryHealth(agentID string, h HistoryHealth) error {
	open := 0
	if h.OpenWriter {
		open = 1
	}
	_, err := s.db.Exec(`INSERT INTO conversation_health(agent_id, state, reason, reserved_seq, open_writer, updated_at)
VALUES(?, ?, ?, ?, ?, ?)
ON CONFLICT(agent_id) DO UPDATE SET state = excluded.state, reason = excluded.reason,
  reserved_seq = excluded.reserved_seq, open_writer = excluded.open_writer, updated_at = excluded.updated_at`,
		agentID, h.State, h.Reason, h.ReservedSeq, open, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
