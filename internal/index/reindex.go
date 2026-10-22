package index

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// Reindex repairs the archive index (sessions, sessions_fts,
// tracked_files, tracked_commands) from raw transcript.ndjson files.
// It is NOT safe to run while the server is live; always stop the server before
// running reindex. Session fields that only SQLite records (retainedSession)
// are carried across each replacement rather than replayed.
func Reindex(home string, db *sql.DB) error {
	if home == "" {
		return fmt.Errorf("index: home is required")
	}
	root := filepath.Join(home, "sessions")
	ids, err := reindexInventory(root, db)
	if err != nil {
		return err
	}
	var failures []error
	fatal := false
	for _, agentID := range ids {
		if err := reindexAgent(db, home, agentID); err != nil {
			if isFatalReindexError(err) {
				failures = append(failures, err)
				fatal = true
				break
			}
			failures = append(failures, err)
		}
	}
	if len(failures) > 0 {
		if fatal {
			return fmt.Errorf("index: reindex stopped after %d source failure(s); earlier repairs committed: %w", len(failures)-1, errors.Join(failures...))
		}
		return fmt.Errorf("index: reindex skipped %d agent(s); restore readable sources and rerun: %w", len(failures), errors.Join(failures...))
	}
	return nil
}

// reindexAgent publishes one complete replacement. Source failures leave the
// prior snapshot intact; fatalReindexError stops the remaining repair work.
func reindexAgent(db *sql.DB, home, agentID string) error {
	var prior int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE agent_id=?`, agentID).Scan(&prior); err != nil {
		return fatalErr(agentID, "inspect prior snapshot", err)
	}
	hadPrior := prior != 0
	limit, err := transcript.CompletePrefix(home, agentID)
	if err != nil || limit == 0 {
		return sourceErr(err, fmt.Sprintf("index: source %s: transcript unavailable (%s)", agentID, snapshotStatus(hadPrior)))
	}
	tx, err := db.Begin()
	if err != nil {
		return fatalErr(agentID, "begin reindex", err)
	}
	defer tx.Rollback()
	retained, err := readRetainedSessionTx(tx, agentID)
	if err != nil && err != sql.ErrNoRows {
		return fatalErr(agentID, "read retained", err)
	}
	hadPrior = err == nil
	if err := deleteAgentTx(tx, agentID); err != nil {
		return fatalErr(agentID, "delete prior projection", err)
	}
	ix := newTxIndexer(db, tx)
	var lastSeq int64
	var lastContext float64
	var lastCounts *state.ContextCounts
	var updatedAt string
	// pendingFlush tracks whether events have been buffered since the last
	// turn_end flush. It must trigger a final flush not only when NO turn_end
	// was ever seen, but also when a completed turn is followed by a partial
	// (crash-truncated) turn — otherwise the partial turn's assistant text sits
	// only in the in-memory buffer and is silently dropped from sessions_fts.
	var pendingFlush bool
	seenMeta := false
	err = transcript.ForEachFilePrefix(home, agentID, limit, transcript.ReadOptions{IncludeMeta: true}, func(ev runtime.Event) error {
		if ev.Seq > lastSeq {
			lastSeq = ev.Seq
			updatedAt = ev.Ts
		}
		pendingFlush = true
		if ev.Type == runtime.EvSessionMeta {
			var meta runtime.SessionMetaData
			if err := json.Unmarshal(ev.Data, &meta); err != nil {
				return sourceErr(err, fmt.Sprintf("index: session metadata %s", agentID))
			}
			if _, err := state.EncodeExecutionContract(meta.ExecutionContract); err != nil {
				return sourceErr(err, fmt.Sprintf("index: invalid execution contract %s", agentID))
			}
			seenMeta = true
			if meta.ResetContext {
				lastContext, lastCounts = 0, nil
			}
		}
		if ev.Type != runtime.EvSessionMeta {
			if _, err := searchableText(ev); err != nil {
				return sourceErr(err, fmt.Sprintf("index: malformed event %s seq %d", agentID, ev.Seq))
			}
		}
		if ev.Type == runtime.EvAnnotation {
			if err := ix.OnEventAndFlushContent(agentID, ev, ev.Seq, ev.Ts); err != nil {
				return classifyReplayError(agentID, fmt.Sprintf("annotation flush seq %d", ev.Seq), err)
			}
			pendingFlush = false
		}
		if ev.Type == runtime.EvTurnEnd {
			var d runtime.TurnEndData
			if err := json.Unmarshal(ev.Data, &d); err != nil {
				return sourceErr(err, fmt.Sprintf("index: malformed turn_end %s seq %d", agentID, ev.Seq))
			}
			lastContext, lastCounts = d.ContextPct, d.ContextCounts
			if err := ix.OnEventAndTurnEnd(agentID, ev, runtime.TurnRollup{LastSeq: ev.Seq, LastContextPct: d.ContextPct, LastContext: d.ContextCounts, UpdatedAt: ev.Ts}); err != nil {
				return classifyReplayError(agentID, fmt.Sprintf("turn_end seq %d", ev.Seq), err)
			}
			pendingFlush = false
		} else if ev.Type != runtime.EvAnnotation {
			if err := ix.OnEvent(agentID, ev); err != nil {
				return classifyReplayError(agentID, fmt.Sprintf("event seq %d", ev.Seq), err)
			}
		}
		return nil
	})
	if err != nil {
		if isFatalReindexError(err) {
			return err
		}
		return sourceErr(err, fmt.Sprintf("index: replay %s (%s)", agentID, snapshotStatus(hadPrior)))
	}
	if !seenMeta {
		return sourceErr(nil, fmt.Sprintf("index: source %s has no usable session metadata (%s)", agentID, snapshotStatus(hadPrior)))
	}
	if lastSeq > 0 && updatedAt != "" && pendingFlush {
		if err := ix.flush(agentID, runtime.TurnRollup{LastSeq: lastSeq, LastContextPct: lastContext, LastContext: lastCounts, UpdatedAt: updatedAt}, false, "partial"); err != nil {
			return fatalErr(agentID, "final flush", err)
		}
	}
	var exists int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM sessions WHERE agent_id=?`, agentID).Scan(&exists); err != nil {
		return fatalErr(agentID, "verify session", err)
	}
	if exists != 1 {
		return sourceErr(nil, fmt.Sprintf("index: replay %s produced no session (%s)", agentID, snapshotStatus(hadPrior)))
	}
	if hadPrior {
		res, err := tx.Exec(`UPDATE sessions SET last_session_id=CASE WHEN ?<>'' THEN ? ELSE last_session_id END, effort=?, fast=? WHERE agent_id=?`, retained.nativeID, retained.nativeID, retained.effort, retained.fast, agentID)
		if err != nil {
			return fatalErr(agentID, "restore retained fields", err)
		}
		n, err := res.RowsAffected()
		if err != nil || n != 1 {
			if err == nil {
				err = fmt.Errorf("affected %d rows", n)
			}
			return fatalErr(agentID, "restore retained fields", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fatalErr(agentID, "commit reindex", err)
	}
	return nil
}

func snapshotStatus(prior bool) string {
	if prior {
		return "prior snapshot preserved"
	}
	return "no prior snapshot"
}

func sourceErr(cause error, msg string) error {
	if cause == nil {
		return errors.New(msg)
	}
	return fmt.Errorf("%s: %w", msg, cause)
}

func classifyReplayError(agent, op string, err error) error {
	var syntax *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &syntax) || errors.As(err, &typeErr) {
		return sourceErr(err, fmt.Sprintf("index: %s %s", op, agent))
	}
	return fatalErr(agent, op, err)
}

type fatalReindexError struct {
	msg   string
	cause error
}

func (e *fatalReindexError) Error() string { return e.msg + ": " + e.cause.Error() }
func (e *fatalReindexError) Unwrap() error { return e.cause }
func fatalErr(agent, op string, err error) error {
	return &fatalReindexError{msg: fmt.Sprintf("index: %s %s", op, agent), cause: err}
}
func isFatalReindexError(err error) bool {
	var fatal *fatalReindexError
	return errors.As(err, &fatal)
}

func reindexInventory(root string, db *sql.DB) ([]string, error) {
	seen := map[string]bool{}
	rows, err := db.Query(`SELECT agent_id FROM sessions`)
	if err != nil {
		return nil, fmt.Errorf("index: inventory sessions: %w", err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("index: inventory sessions: %w", err)
		}
		seen[id] = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("index: inventory sessions: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("index: inventory sessions: %w", err)
	}
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		out := make([]string, 0, len(seen))
		for id := range seen {
			out = append(out, id)
		}
		sort.Strings(out)
		return out, nil
	}
	if err != nil {
		return nil, fmt.Errorf("index: read sessions dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			seen[e.Name()] = true
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

func readRetainedSessionTx(tx *sql.Tx, agentID string) (retainedSession, error) {
	var r retainedSession
	err := tx.QueryRow(`SELECT last_session_id, effort, fast FROM sessions WHERE agent_id=?`, agentID).Scan(&r.nativeID, &r.effort, &r.fast)
	return r, err
}

func deleteAgentTx(tx *sql.Tx, agentID string) error {
	if ok, err := ftsWritesAvailable(tx); err != nil {
		return fmt.Errorf("index: detect FTS for %s: %w", agentID, err)
	} else if ok {
		if _, err := tx.Exec(`DELETE FROM sessions_fts WHERE agent_id=?`, agentID); err != nil {
			return fmt.Errorf("index: delete FTS for %s: %w", agentID, err)
		}
	}
	for _, table := range []string{"tracked_commands", "tracked_files", "sessions"} {
		if _, err := tx.Exec(`DELETE FROM `+table+` WHERE agent_id=?`, agentID); err != nil {
			return fmt.Errorf("index: delete %s for %s: %w", table, agentID, err)
		}
	}
	return nil
}

// RemoveAgent drops one agent's index rows and buffered text. It is the index
// half of a failed clone's rollback (TS-02.R35, INV §15); the rows are rebuilt
// from the transcript by reindex, so nothing else depends on them.
func (ix *Indexer) RemoveAgent(agentID string) error {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	delete(ix.content, agentID)
	delete(ix.fileCalls, agentID)
	tx, err := ix.db.Begin()
	if err != nil {
		return fmt.Errorf("index: begin remove: %w", err)
	}
	defer tx.Rollback()
	tables := []string{"tracked_commands", "tracked_files", "sessions"}
	if ok, err := ix.ftsWritable(tx); err != nil {
		return fmt.Errorf("index: remove: %w", err)
	} else if ok {
		tables = append(tables, "sessions_fts")
	}
	for _, table := range tables {
		if _, err := tx.Exec(`DELETE FROM `+table+` WHERE agent_id = ?`, agentID); err != nil {
			return fmt.Errorf("index: remove %s: %w", table, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("index: commit remove: %w", err)
	}
	return nil
}

// retainedSession is the part of a sessions row that is committed straight to
// SQLite and never appended to the transcript, so replay cannot rebuild it
// (TS-02.R9): the hook-learned native conversation ID (TS-02.R54) and the
// live effort/fast settings. Every other column is a replayable projection.
type retainedSession struct {
	nativeID string
	effort   string
	fast     bool
}
