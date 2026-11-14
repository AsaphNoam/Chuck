package state

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestConversationTurnMigrationPreservesBudget(t *testing.T) {
	home := t.TempDir()
	db, err := sql.Open("sqlite3", filepath.Join(home, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
WITH RECURSIVE versions(n) AS (VALUES (1) UNION ALL SELECT n+1 FROM versions WHERE n<76)
INSERT INTO schema_migrations SELECT n, '' FROM versions;
CREATE TABLE turn_budget (agent_id TEXT, turn_id TEXT, inbound INTEGER, outbound INTEGER,
 breached INTEGER, delivery_turn_key TEXT, inline_message_ids TEXT, PRIMARY KEY(agent_id, turn_id));
INSERT INTO turn_budget VALUES ('a', 't1', 7, 3, 1, 'existing-key', '["m1"]');`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := Open(home)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var state, outcome, origin, generation, key, mail string
	var inbound, outbound, breached int
	err = s.db.QueryRow(`SELECT state, outcome, origin_kind, generation, delivery_turn_key,
inline_message_ids, inbound, outbound, breached FROM turn_budget WHERE agent_id='a'`).
		Scan(&state, &outcome, &origin, &generation, &key, &mail, &inbound, &outbound, &breached)
	if err != nil || state != "ended" || outcome != "" || origin != "" || generation != "" ||
		key != "existing-key" || mail != `["m1"]` || inbound != 7 || outbound != 3 || breached != 1 {
		t.Fatalf("migrated budget changed: %q/%q/%q/%q %q/%q %d/%d/%d, %v",
			state, outcome, origin, generation, key, mail, inbound, outbound, breached, err)
	}
	if _, err := s.ReadConversationTurn("a", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("legacy budget resolved as a conversation turn: %v", err)
	}
}

func TestConversationTurnReplacementAndEndAreGenerationScoped(t *testing.T) {
	s, _ := newTestStore(t)
	agentID := "a_turn"
	if err := s.WriteAgent(testAgent(agentID, time.Now())); err != nil {
		t.Fatal(err)
	}
	for _, origin := range []TurnOrigin{
		{Generation: "g1", Kind: TurnOriginPerson, ID: "receipt"},
		{Generation: "g2", Kind: TurnOriginTask, ID: "task"},
		{Generation: "g3", Kind: TurnOriginMail, ID: "mail"},
		{Generation: "g4", Kind: TurnOriginQuota, ID: "quota"},
		{Generation: "g5", Kind: TurnOriginRoom, ID: "room"},
		{Generation: "g6", Kind: TurnOriginOwnerContinuation, ID: "settled-turn"},
	} {
		if err := s.BeginConversationTurn(agentID, "t1", origin); err != nil {
			t.Fatal(err)
		}
		row, err := s.ReadConversationTurn(agentID, origin.Generation)
		if err != nil || row.TurnOrigin != origin || row.State != "running" || row.StartedAt == "" || row.EndedAt != "" || row.Outcome != "" {
			t.Fatalf("begin %+v: row %+v, err %v", origin, row, err)
		}
	}
	if err := s.EndConversationTurn(agentID, TurnEnd{Generation: "g5", TurnID: "t1", Outcome: "failed"}); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM turn_budget WHERE agent_id=?`, agentID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("turn row count %d, err %v", count, err)
	}
	st := Status{AgentID: agentID, State: "idle"}
	if err := s.WriteTurnEndStatus(st, TurnEnd{Generation: "g6", TurnID: "t1", Outcome: "completed"}); err != nil {
		t.Fatal(err)
	}
	if err := s.EndConversationTurn(agentID, TurnEnd{Generation: "g6", TurnID: "t1", Outcome: "failed"}); err != nil {
		t.Fatal(err)
	}
	row, err := s.ReadConversationTurn(agentID, "g6")
	if err != nil || row.State != "ended" || row.Outcome != "completed" || row.EndedAt == "" {
		t.Fatalf("settled turn %+v, err %v", row, err)
	}
}
