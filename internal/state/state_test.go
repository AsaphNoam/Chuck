package state

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"
	"time"
)

func TestMigrationFromV47PreservesLegacyAwarenessAndGuardsBoolean(t *testing.T) {
	home := t.TempDir()
	db, err := sql.Open("sqlite3", filepath.Join(home, "state.db"))
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	_, err = db.Exec(`
CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL);
CREATE TABLE sessions (
 agent_id TEXT PRIMARY KEY, name TEXT NOT NULL, role TEXT NOT NULL, project TEXT NOT NULL,
 backend TEXT NOT NULL, model TEXT NOT NULL, effort TEXT NOT NULL DEFAULT '', fast INTEGER NOT NULL DEFAULT 0,
 interface TEXT NOT NULL, grp TEXT NOT NULL DEFAULT '', cwd TEXT NOT NULL, system_prompt TEXT NOT NULL,
 env_keys TEXT NOT NULL DEFAULT '[]', skip_permissions INTEGER NOT NULL DEFAULT 0,
 add_dirs TEXT NOT NULL DEFAULT '[]', launch_config_json TEXT NOT NULL DEFAULT '{}',
 last_session_id TEXT NOT NULL DEFAULT '', last_seq INTEGER NOT NULL DEFAULT 0,
 last_context_pct REAL NOT NULL DEFAULT 0, last_context_used INTEGER, last_context_size INTEGER,
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL, turn_count INTEGER NOT NULL DEFAULT 0,
 event_count INTEGER NOT NULL DEFAULT 0, files_touched INTEGER NOT NULL DEFAULT 0,
 commands_run INTEGER NOT NULL DEFAULT 0, runtime_capabilities_json TEXT NOT NULL DEFAULT '{}',
 forked_from_agent_id TEXT, forked_from_seq INTEGER
);
INSERT INTO schema_migrations(version, applied_at)
SELECT value, '2026-01-01T00:00:00Z' FROM (SELECT 1 value UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4 UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9 UNION ALL SELECT 10 UNION ALL SELECT 11 UNION ALL SELECT 12 UNION ALL SELECT 13 UNION ALL SELECT 14 UNION ALL SELECT 15 UNION ALL SELECT 16 UNION ALL SELECT 17 UNION ALL SELECT 18 UNION ALL SELECT 19 UNION ALL SELECT 20 UNION ALL SELECT 21 UNION ALL SELECT 22 UNION ALL SELECT 23 UNION ALL SELECT 24 UNION ALL SELECT 25 UNION ALL SELECT 26 UNION ALL SELECT 27 UNION ALL SELECT 28 UNION ALL SELECT 29 UNION ALL SELECT 30 UNION ALL SELECT 31 UNION ALL SELECT 32 UNION ALL SELECT 33 UNION ALL SELECT 34 UNION ALL SELECT 35 UNION ALL SELECT 36 UNION ALL SELECT 37 UNION ALL SELECT 38 UNION ALL SELECT 39 UNION ALL SELECT 40 UNION ALL SELECT 41 UNION ALL SELECT 42 UNION ALL SELECT 43 UNION ALL SELECT 44 UNION ALL SELECT 45 UNION ALL SELECT 46 UNION ALL SELECT 47);
INSERT INTO sessions(agent_id, name, role, project, backend, model, interface, cwd, system_prompt, created_at, updated_at)
VALUES ('legacy', 'Legacy', 'r', 'p', 'b', 'm', 'chat', '/tmp', '', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');`)
	if err != nil {
		db.Close()
		t.Fatalf("build v47 database: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close legacy db: %v", err)
	}
	st, err := Open(home)
	if err != nil {
		t.Fatalf("migrate v47 database: %v", err)
	}
	defer st.Close()
	var aware bool
	if err := st.DB().QueryRow(`SELECT chuck_aware FROM sessions WHERE agent_id = 'legacy'`).Scan(&aware); err != nil {
		t.Fatalf("read migrated awareness: %v", err)
	}
	if !aware {
		t.Fatal("legacy session migrated with awareness off")
	}
	if _, err := st.DB().Exec(`UPDATE sessions SET chuck_aware = 2 WHERE agent_id = 'legacy'`); err == nil {
		t.Fatal("boolean CHECK accepted invalid awareness value")
	}
}

func newTestStore(t *testing.T) (*Store, string) {
	t.Helper()
	home := t.TempDir()
	st, err := Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	return st, home
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse time %q: %v", s, err)
	}
	return tm
}

func testAgent(id string, createdAt time.Time) Agent {
	return Agent{
		AgentID:   id,
		Name:      "Atlas",
		Role:      "implementer",
		Project:   "my-app",
		Backend:   "claude",
		Model:     "sonnet-4-6",
		Interface: "chat",
		CreatedAt: createdAt,
		Group:     "auth-migration",
	}
}

func TestOpenMigratesAndConfiguresSQLite(t *testing.T) {
	st, home := newTestStore(t)
	if _, err := os.Stat(filepath.Join(home, "state.db")); err != nil {
		t.Fatalf("state.db missing: %v", err)
	}

	var journalMode string
	if err := st.DB().QueryRow(`PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}
	var foreignKeys int
	if err := st.DB().QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatalf("foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}
	var busyTimeout int
	if err := st.DB().QueryRow(`PRAGMA busy_timeout`).Scan(&busyTimeout); err != nil {
		t.Fatalf("busy_timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", busyTimeout)
	}
	var synchronous int
	if err := st.DB().QueryRow(`PRAGMA synchronous`).Scan(&synchronous); err != nil {
		t.Fatalf("synchronous: %v", err)
	}
	if synchronous != 1 {
		t.Fatalf("synchronous = %d, want 1 (NORMAL)", synchronous)
	}

	for _, table := range []string{"schema_migrations", "agents", "running", "status", "messages", "turn_budget", "sessions", "sessions_fts", "tracked_files", "tracked_commands", "pipeline_runs", "pipeline_attempts", "pipeline_values", "pipeline_requests"} {
		var name string
		err := st.DB().QueryRow(
			`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`,
			table,
		).Scan(&name)
		if err != nil {
			t.Fatalf("table %s missing: %v", table, err)
		}
	}
	rows, err := st.DB().Query(`PRAGMA table_info(sessions_fts)`)
	if err != nil {
		t.Fatalf("sessions_fts columns: %v", err)
	}
	var hasDocumentID bool
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan sessions_fts column: %v", err)
		}
		hasDocumentID = hasDocumentID || name == "document_id"
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate sessions_fts columns: %v", err)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("close sessions_fts columns: %v", err)
	}
	if !hasDocumentID {
		t.Fatal("sessions_fts.document_id missing")
	}
	var version int
	if err := st.DB().QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatalf("schema_migrations version: %v", err)
	}
	// Derived from the migrations slice rather than a literal: a hand-maintained
	// number silently drifts the moment a migration is appended (INV §9).
	if want := migrations[len(migrations)-1].version; version != want {
		t.Fatalf("migration version = %d, want %d", version, want)
	}
}

func TestOpenIsIdempotentAndPreservesRows(t *testing.T) {
	st, home := newTestStore(t)
	agent := testAgent("a_8f3c12", mustTime(t, "2026-06-22T10:00:00Z"))
	if err := st.WriteAgent(agent); err != nil {
		t.Fatalf("WriteAgent: %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("Close before reopen: %v", err)
	}

	reopened, err := Open(home)
	if err != nil {
		t.Fatalf("Open again: %v", err)
	}
	defer reopened.Close()

	got, err := reopened.ReadAgent(agent.AgentID)
	if err != nil {
		t.Fatalf("ReadAgent after reopen: %v", err)
	}
	if !reflect.DeepEqual(got, agent) {
		t.Fatalf("agent after reopen = %+v, want %+v", got, agent)
	}
	var count int
	if err := reopened.DB().QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&count); err != nil {
		t.Fatalf("count migration: %v", err)
	}
	if want := len(migrations); count != want {
		t.Fatalf("migration version rows = %d, want %d", count, want)
	}
}

func TestRoundTripStateObjects(t *testing.T) {
	st, _ := newTestStore(t)
	busy := mustTime(t, "2026-06-22T10:00:05Z")
	read := mustTime(t, "2026-06-22T10:00:10Z")

	agent := testAgent("a_8f3c12", mustTime(t, "2026-06-22T10:00:00Z"))
	if err := st.WriteAgent(agent); err != nil {
		t.Fatalf("WriteAgent: %v", err)
	}
	if got, err := st.ReadAgent(agent.AgentID); err != nil || !reflect.DeepEqual(got, agent) {
		t.Fatalf("Agent round-trip: got %+v err %v, want %+v", got, err, agent)
	}

	run := RunningEntry{
		AgentID:   agent.AgentID,
		PID:       48213,
		SessionID: "claude-sess-xyz",
		Interface: "chat",
		TTY:       "/dev/ttys001",
		HookToken: "tok_live",
		StartedAt: mustTime(t, "2026-06-22T10:00:01Z"),
	}
	if err := st.WriteRunning(run); err != nil {
		t.Fatalf("WriteRunning: %v", err)
	}
	if got, err := st.ReadRunning(run.AgentID); err != nil || !reflect.DeepEqual(got, run) {
		t.Fatalf("RunningEntry round-trip: got %+v err %v, want %+v", got, err, run)
	}

	status := Status{
		AgentID:    agent.AgentID,
		State:      "busy",
		Detail:     "Editing src/auth.ts",
		LastTrace:  "tool: edit",
		BusySince:  &busy,
		ContextPct: 0.42,
		UpdatedAt:  123456,
	}
	if err := st.WriteStatus(status); err != nil {
		t.Fatalf("WriteStatus: %v", err)
	}
	if got, err := st.ReadStatus(status.AgentID); err != nil || !reflect.DeepEqual(got, status) {
		t.Fatalf("Status round-trip: got %+v err %v, want %+v", got, err, status)
	}

	recipient := testAgent("a_123abc", mustTime(t, "2026-06-22T10:00:02Z"))
	if err := st.WriteAgent(recipient); err != nil {
		t.Fatalf("WriteAgent recipient: %v", err)
	}
	_ = read
	msg := Message{
		FromAgent:   agent.AgentID,
		FromAddress: address(agent.Role, agent.Project),
		FromName:    agent.Name,
		ToAgent:     recipient.AgentID,
		Subject:     "Review request",
		Body:        "Please review auth.",
		CreatedAt:   mustTime(t, "2026-06-22T10:00:03Z"),
	}
	id, err := st.InsertMessage(msg)
	if err != nil {
		t.Fatalf("InsertMessage: %v", err)
	}
	if !regexp.MustCompile(`^m_[0-9a-f]{6}$`).MatchString(id) {
		t.Fatalf("message_id = %q, want ^m_[0-9a-f]{6}$", id)
	}
	messages, err := st.ListMessages(recipient.AgentID, false, 0)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("messages = %+v, want 1 row", messages)
	}
	got := messages[0]
	if got.MessageID != id || got.FromAgent != agent.AgentID || got.ToAgent != recipient.AgentID ||
		got.Body != msg.Body || got.Subject != msg.Subject || got.Read || got.DeliveredVia != "pending" {
		t.Fatalf("message round-trip = %+v", got)
	}

	// Unread filter + count, mark-read, delete.
	if n, err := st.UnreadCount(recipient.AgentID); err != nil || n != 1 {
		t.Fatalf("UnreadCount = %d err %v, want 1", n, err)
	}
	if err := st.MarkRead([]string{id}); err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if n, err := st.UnreadCount(recipient.AgentID); err != nil || n != 0 {
		t.Fatalf("UnreadCount after MarkRead = %d err %v, want 0", n, err)
	}
	readRows, err := st.ListMessages(recipient.AgentID, false, 0)
	if err != nil {
		t.Fatalf("ListMessages after MarkRead: %v", err)
	}
	if len(readRows) != 1 || readRows[0].DeliveredVia != "poll" || !readRows[0].Read || readRows[0].ReadAt == nil {
		t.Fatalf("message after MarkRead = %+v, want read via poll with read_at", readRows)
	}
	if unread, err := st.ListMessages(recipient.AgentID, true, 0); err != nil || len(unread) != 0 {
		t.Fatalf("unread-only after MarkRead = %+v err %v, want []", unread, err)
	}
	if err := st.DeleteMessages([]string{id}); err != nil {
		t.Fatalf("DeleteMessages: %v", err)
	}
	if all, err := st.ListMessages(recipient.AgentID, false, 0); err != nil || len(all) != 0 {
		t.Fatalf("messages after delete = %+v err %v, want []", all, err)
	}
}

func TestTurnBudgetConsumeAndBreach(t *testing.T) {
	st, _ := newTestStore(t)
	if err := st.ResetTurnBudget("a_budget", "t_000000000001"); err != nil {
		t.Fatalf("ResetTurnBudget: %v", err)
	}
	got, breached, err := st.ConsumeTurnBudget("a_budget", 10, 4, 15)
	if err != nil || breached {
		t.Fatalf("ConsumeTurnBudget first = %+v breached=%v err=%v, want no breach", got, breached, err)
	}
	if got.Inbound != 10 || got.Outbound != 4 || got.Remaining != 1 || got.Breached {
		t.Fatalf("budget after first consume = %+v, want 10/4 remaining 1", got)
	}
	got, breached, err = st.ConsumeTurnBudget("a_budget", 0, 2, 15)
	if err != nil || !breached {
		t.Fatalf("ConsumeTurnBudget breach = %+v breached=%v err=%v, want breach", got, breached, err)
	}
	if got.Inbound != 10 || got.Outbound != 4 || got.Remaining != 1 || !got.Breached {
		t.Fatalf("budget after breach = %+v, want unchanged 10/4 breached remaining 1", got)
	}
	got, err = st.CurrentTurnBudget("a_budget", 15)
	if err != nil {
		t.Fatalf("CurrentTurnBudget: %v", err)
	}
	if !got.Breached || got.Remaining != 1 {
		t.Fatalf("persisted budget = %+v, want breached remaining 1", got)
	}

	if err := st.ResetTurnBudget("a_budget", "t_000000000002"); err != nil {
		t.Fatalf("ResetTurnBudget second: %v", err)
	}
	got, err = st.CurrentTurnBudget("a_budget", 15)
	if err != nil {
		t.Fatalf("CurrentTurnBudget second: %v", err)
	}
	if got.TurnID != "t_000000000002" || got.Inbound != 0 || got.Outbound != 0 || got.Breached || got.Remaining != 15 {
		t.Fatalf("budget after reset = %+v, want fresh turn", got)
	}
}

// TestResetTurnBudgetReusesSingleRow guards the restart+resume path: turnSeq
// resets to 0 on a fresh process, so ResetTurnBudget re-targets low turn_ids
// while prior-session rows linger with higher rowids. ResetTurnBudget must keep
// at most one row per agent so the runtime's reset and the MCP handlers'
// reads/increments always agree on the current turn.
func TestResetTurnBudgetReusesSingleRow(t *testing.T) {
	st, _ := newTestStore(t)
	// Session 1: turn 1 runs to breach, then turn 2 starts.
	if err := st.ResetTurnBudget("a_budget", "t_000000000001"); err != nil {
		t.Fatalf("ResetTurnBudget t1: %v", err)
	}
	if _, _, err := st.ConsumeTurnBudget("a_budget", 15, 1, 15); err != nil {
		t.Fatalf("ConsumeTurnBudget breach: %v", err)
	}
	if err := st.ResetTurnBudget("a_budget", "t_000000000002"); err != nil {
		t.Fatalf("ResetTurnBudget t2: %v", err)
	}
	if _, _, err := st.ConsumeTurnBudget("a_budget", 5, 0, 15); err != nil {
		t.Fatalf("ConsumeTurnBudget t2: %v", err)
	}

	// Simulate a server restart + resume: turnSeq is back to 0, so the first
	// runtime turn resets t_...01 again. The handler's current-budget read must
	// see this freshly-reset row (0 used), not the stale t_...02.
	if err := st.ResetTurnBudget("a_budget", "t_000000000001"); err != nil {
		t.Fatalf("ResetTurnBudget t1 after restart: %v", err)
	}
	got, err := st.CurrentTurnBudget("a_budget", 15)
	if err != nil {
		t.Fatalf("CurrentTurnBudget: %v", err)
	}
	if got.TurnID != "t_000000000001" || got.Inbound != 0 || got.Outbound != 0 || got.Breached {
		t.Fatalf("post-restart budget = %+v, want fresh t_...01 (0/0, not breached)", got)
	}

	cur, breached, err := st.ConsumeTurnBudget("a_budget", 1, 0, 15)
	if err != nil || breached {
		t.Fatalf("ConsumeTurnBudget post-restart = %+v breached=%v err=%v, want no breach on fresh row", cur, breached, err)
	}
	if cur.TurnID != "t_000000000001" || cur.Inbound != 1 {
		t.Fatalf("ConsumeTurnBudget post-restart = %+v, want t_...01 inbound 1", cur)
	}

	// Exactly one row survives per agent (also caps unbounded growth).
	var n int
	if err := st.db.QueryRow(`SELECT COUNT(*) FROM turn_budget WHERE agent_id = ?`, "a_budget").Scan(&n); err != nil {
		t.Fatalf("count turn_budget rows: %v", err)
	}
	if n != 1 {
		t.Fatalf("turn_budget rows = %d, want 1", n)
	}
}

func TestDeleteExpiredMessagesRetention(t *testing.T) {
	st, _ := newTestStore(t)
	now := mustTime(t, "2026-06-29T12:00:00Z")
	agent := testAgent("a_sender", now)
	recipient := testAgent("a_recipient", now)
	for _, a := range []Agent{agent, recipient} {
		if err := st.WriteAgent(a); err != nil {
			t.Fatalf("WriteAgent: %v", err)
		}
	}
	insert := func(body string, created time.Time) string {
		t.Helper()
		id, err := st.InsertMessage(Message{
			FromAgent: agent.AgentID, FromAddress: address(agent.Role, agent.Project), FromName: agent.Name,
			ToAgent: recipient.AgentID, Body: body, CreatedAt: created,
		})
		if err != nil {
			t.Fatalf("InsertMessage %s: %v", body, err)
		}
		return id
	}
	oldRead := insert("old-read", now.Add(-25*time.Hour))
	recentRead := insert("recent-read", now.Add(-2*time.Hour))
	hardOld := insert("hard-old-unread", now.Add(-8*24*time.Hour))
	fresh := insert("fresh-unread", now.Add(-time.Hour))

	if _, err := st.DB().Exec(`UPDATE messages SET read = 1, read_at = ? WHERE message_id IN (?, ?)`,
		formatTime(now.Add(-25*time.Hour)), oldRead, recentRead); err != nil {
		t.Fatalf("mark fixture read: %v", err)
	}
	if _, err := st.DB().Exec(`UPDATE messages SET read_at = ? WHERE message_id = ?`,
		formatTime(now.Add(-2*time.Hour)), recentRead); err != nil {
		t.Fatalf("mark recent read: %v", err)
	}

	readDeleted, hardDeleted, err := st.DeleteExpiredMessages(now, 24*time.Hour, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("DeleteExpiredMessages: %v", err)
	}
	if readDeleted != 1 || hardDeleted != 1 {
		t.Fatalf("deleted read=%d hard=%d, want 1/1", readDeleted, hardDeleted)
	}
	rows, err := st.ListMessages(recipient.AgentID, false, 0)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	left := map[string]bool{}
	for _, m := range rows {
		left[m.MessageID] = true
	}
	if left[oldRead] || left[hardOld] || !left[recentRead] || !left[fresh] {
		t.Fatalf("remaining ids = %v, want recentRead+fresh only", left)
	}
}

func TestListEmptyAndReadNotFound(t *testing.T) {
	st, _ := newTestStore(t)
	if agents, err := st.ListAgents(); err != nil || len(agents) != 0 {
		t.Fatalf("ListAgents empty = %+v err %v, want [] nil", agents, err)
	}
	if running, err := st.ListRunning(); err != nil || len(running) != 0 {
		t.Fatalf("ListRunning empty = %+v err %v, want [] nil", running, err)
	}
	if statuses, err := st.ListStatus(); err != nil || len(statuses) != 0 {
		t.Fatalf("ListStatus empty = %+v err %v, want [] nil", statuses, err)
	}
	if messages, err := st.ListMessages("a_nope", false, 0); err != nil || len(messages) != 0 {
		t.Fatalf("ListMessages empty = %+v err %v, want [] nil", messages, err)
	}

	if _, err := st.ReadAgent("a_nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ReadAgent missing err = %v, want ErrNotFound", err)
	}
	if _, err := st.ReadRunning("a_nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ReadRunning missing err = %v, want ErrNotFound", err)
	}
	if _, err := st.ReadStatus("a_nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ReadStatus missing err = %v, want ErrNotFound", err)
	}
}

func TestNewAgentIDFormatUniquenessAndCollisionRetry(t *testing.T) {
	st, _ := newTestStore(t)
	re := regexp.MustCompile(`^a_[0-9a-f]{6}$`)
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		id, err := st.NewAgentID()
		if err != nil {
			t.Fatalf("NewAgentID: %v", err)
		}
		if !re.MatchString(id) {
			t.Fatalf("NewAgentID = %q, want ^a_[0-9a-f]{6}$", id)
		}
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
		if err := st.WriteAgent(testAgent(id, mustTime(t, "2026-06-22T10:00:00Z"))); err != nil {
			t.Fatalf("WriteAgent generated %s: %v", id, err)
		}
	}

	// The sweep above draws real random ids, so it can itself mint a_abcdef —
	// the id the stub below retries into. Clear the retry target first, or that
	// run exhausts all 10 tries and the test fails for an unrelated reason.
	if _, err := st.db.Exec(`DELETE FROM agents WHERE agent_id = ?`, "a_abcdef"); err != nil {
		t.Fatalf("clear retry target: %v", err)
	}

	originalRandRead := randRead
	defer func() { randRead = originalRandRead }()
	calls := 0
	randRead = func(b []byte) (int, error) {
		calls++
		if calls == 1 {
			copy(b, []byte{0x8f, 0x3c, 0x12})
			return len(b), nil
		}
		copy(b, []byte{0xab, 0xcd, 0xef})
		return len(b), nil
	}

	if err := st.WriteAgent(testAgent("a_8f3c12", mustTime(t, "2026-06-22T10:00:00Z"))); err != nil {
		t.Fatalf("WriteAgent collision row: %v", err)
	}
	id, err := st.NewAgentID()
	if err != nil {
		t.Fatalf("NewAgentID after collision: %v", err)
	}
	if id != "a_abcdef" {
		t.Fatalf("NewAgentID after collision = %q, want a_abcdef", id)
	}
	if calls != 2 {
		t.Fatalf("randRead calls = %d, want 2", calls)
	}
}

func TestDeleteAgentCascades(t *testing.T) {
	st, _ := newTestStore(t)
	agent := testAgent("a_8f3c12", mustTime(t, "2026-06-22T10:00:00Z"))
	recipient := testAgent("a_123abc", mustTime(t, "2026-06-22T10:00:01Z"))
	for _, a := range []Agent{agent, recipient} {
		if err := st.WriteAgent(a); err != nil {
			t.Fatalf("WriteAgent: %v", err)
		}
	}
	if err := st.WriteRunning(RunningEntry{
		AgentID: agent.AgentID, PID: 1, SessionID: "s", Interface: "chat",
		StartedAt: mustTime(t, "2026-06-22T10:00:02Z"),
	}); err != nil {
		t.Fatalf("WriteRunning: %v", err)
	}
	if err := st.WriteStatus(Status{AgentID: agent.AgentID, State: "idle"}); err != nil {
		t.Fatalf("WriteStatus: %v", err)
	}
	// Phase 5: messages have no agent FK (they outlive a stopped/deleted agent
	// until the janitor — techspec §4.3), so they are not cascade-deleted here.
	if _, err := st.InsertMessage(Message{
		FromAgent: agent.AgentID, FromAddress: address(agent.Role, agent.Project), FromName: agent.Name,
		ToAgent: recipient.AgentID, Body: "hello",
		CreatedAt: mustTime(t, "2026-06-22T10:00:03Z"),
	}); err != nil {
		t.Fatalf("InsertMessage: %v", err)
	}

	if err := st.DeleteAgent(agent.AgentID); err != nil {
		t.Fatalf("DeleteAgent: %v", err)
	}
	if _, err := st.ReadRunning(agent.AgentID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ReadRunning after DeleteAgent err = %v, want ErrNotFound", err)
	}
	if _, err := st.ReadStatus(agent.AgentID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ReadStatus after DeleteAgent err = %v, want ErrNotFound", err)
	}
	// The message survives the deleted sender (no FK cascade).
	messages, err := st.ListMessages(recipient.AgentID, false, 0)
	if err != nil {
		t.Fatalf("ListMessages after DeleteAgent: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("messages after DeleteAgent = %+v, want 1 (no cascade)", messages)
	}
}

func TestDeleteMethodsAreIdempotent(t *testing.T) {
	st, _ := newTestStore(t)
	if err := st.DeleteAgent("a_nope"); err != nil {
		t.Fatalf("DeleteAgent missing: %v", err)
	}
	if err := st.DeleteRunning("a_nope"); err != nil {
		t.Fatalf("DeleteRunning missing: %v", err)
	}
	if err := st.DeleteStatus("a_nope"); err != nil {
		t.Fatalf("DeleteStatus missing: %v", err)
	}
	if err := st.DeleteMessages([]string{"m_nope"}); err != nil {
		t.Fatalf("DeleteMessages missing: %v", err)
	}
	if err := st.MarkRead([]string{"m_nope"}); err != nil {
		t.Fatalf("MarkRead missing: %v", err)
	}
}

// TestStateDBIsOwnerOnly guards the security fix for a world-readable
// state.db: SQLite creates the file umask-relative, so Open must chmod it
// (and thereby its -wal/-shm siblings, which inherit its mode) to 0600.
func TestStateDBIsOwnerOnly(t *testing.T) {
	home := t.TempDir()
	// Simulate an old install's loose db before reopening.
	loose := filepath.Join(home, "state.db")
	if err := os.WriteFile(loose, nil, 0o644); err != nil {
		t.Fatalf("seed loose db: %v", err)
	}
	st, err := Open(home)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()
	fi, err := os.Stat(loose)
	if err != nil {
		t.Fatalf("stat db: %v", err)
	}
	if perm := fi.Mode().Perm(); perm&0o077 != 0 {
		t.Errorf("state.db perms = %04o, want no group/other bits", perm)
	}
}
