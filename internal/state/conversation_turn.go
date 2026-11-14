package state

import (
	"database/sql"
	"errors"
	"fmt"
)

type statusExecer interface {
	Exec(string, ...any) (sql.Result, error)
}

type TurnOriginKind string

const (
	TurnOriginPerson            TurnOriginKind = "person"
	TurnOriginMail              TurnOriginKind = "mail"
	TurnOriginTask              TurnOriginKind = "task"
	TurnOriginQuota             TurnOriginKind = "quota"
	TurnOriginRoom              TurnOriginKind = "room"
	TurnOriginOwnerContinuation TurnOriginKind = "owner_continuation"
)

// TurnOrigin names the generation and domain record that opened a chat turn.
type TurnOrigin struct {
	Generation string
	Kind       TurnOriginKind
	ID         string
}

type ConversationTurn struct {
	TurnID string
	TurnOrigin
	State     string
	Outcome   string
	StartedAt string
	EndedAt   string
}

func (s *Store) BeginConversationTurn(agentID, turnID string, origin TurnOrigin) error {
	return s.resetTurnBudget(agentID, turnID, origin)
}

func (s *Store) ReadConversationTurn(agentID, generation string) (ConversationTurn, error) {
	return readConversationTurn(s.db, agentID, generation)
}

func readConversationTurn(db admissionReader, agentID, generation string) (ConversationTurn, error) {
	var t ConversationTurn
	err := db.QueryRow(`SELECT turn_id, generation, origin_kind, COALESCE(origin_id, ''), state, outcome,
COALESCE(started_at, ''), COALESCE(ended_at, '') FROM turn_budget
WHERE agent_id=? AND generation=? AND generation<>''`, agentID, generation).
		Scan(&t.TurnID, &t.Generation, &t.Kind, &t.ID, &t.State, &t.Outcome, &t.StartedAt, &t.EndedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ConversationTurn{}, ErrNotFound
	}
	return t, err
}

type TurnEnd struct {
	Generation string
	TurnID     string
	Outcome    string
}

// EndConversationTurn settles a refused begin without changing its status.
func (s *Store) EndConversationTurn(agentID string, end TurnEnd) error {
	return endConversationTurn(s.db, agentID, end)
}

func endConversationTurn(db statusExecer, agentID string, end TurnEnd) error {
	if end.Generation == "" {
		return nil
	}
	if _, err := db.Exec(`UPDATE turn_budget SET state='ended', outcome=?, ended_at=?
WHERE agent_id=? AND generation=? AND (?='' OR turn_id=?) AND state='running'`,
		end.Outcome, formatTime(timeNow()), agentID, end.Generation, end.TurnID, end.TurnID); err != nil {
		return fmt.Errorf("state: end conversation turn: %w", err)
	}
	return nil
}
