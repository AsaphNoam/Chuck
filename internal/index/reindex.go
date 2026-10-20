package index

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// Reindex wipes and rebuilds the archive index (sessions, sessions_fts,
// tracked_files, tracked_commands) from raw transcript.ndjson files.
// It is NOT safe to run while the server is live: resetTables deletes all
// sessions rows outside the replay transaction, so any agents active during
// the wipe are permanently lost from the index. Always stop the server before
// running reindex. Session fields that only SQLite records (retainedSession)
// are carried across the rebuild rather than replayed.
func Reindex(home string, db *sql.DB) error {
	if home == "" {
		return fmt.Errorf("index: home is required")
	}
	root := filepath.Join(home, "sessions")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return resetTables(db)
		}
		return fmt.Errorf("index: read sessions dir: %w", err)
	}
	// Read the authoritative fields before the wipe; failing here leaves the
	// archive untouched rather than rebuilding without them.
	retained, err := readRetainedSessions(db)
	if err != nil {
		return err
	}
	if err := resetTables(db); err != nil {
		return err
	}
	ix := New(db)
	var failures []error
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		agentID := entry.Name()
		// Isolate each agent: a bad transcript (unreadable, or an OnEvent/
		// OnTurnEnd/flush error) is captured and skipped so the remaining
		// agents are still reindexed rather than leaving the archive wiped.
		if err := reindexAgent(ix, root, agentID); err != nil {
			failures = append(failures, err)
		}
		if r, ok := retained[agentID]; ok {
			if err := restoreRetainedSession(db, agentID, r); err != nil {
				failures = append(failures, err)
			}
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("index: reindex skipped %d agent(s): %w", len(failures), errors.Join(failures...))
	}
	return nil
}

// reindexAgent replays one agent's transcript into the index. Any error is
// returned (not fatal to the whole reindex) so the caller can continue.
func reindexAgent(ix *Indexer, root, agentID string) error {
	path := filepath.Join(root, agentID, "transcript.ndjson")
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
	err := transcript.NewReader(path).ForEach(transcript.ReadOptions{IncludeMeta: true}, func(ev runtime.Event) error {
		if ev.Seq > lastSeq {
			lastSeq = ev.Seq
			updatedAt = ev.Ts
		}
		pendingFlush = true
		if ev.Type == runtime.EvSessionMeta {
			var meta runtime.SessionMetaData
			if err := json.Unmarshal(ev.Data, &meta); err != nil {
				return fmt.Errorf("index: session metadata %s: %w", agentID, err)
			}
			if meta.ResetContext {
				lastContext, lastCounts = 0, nil
			}
		}
		if ev.Type == runtime.EvAnnotation {
			if err := ix.OnEventAndFlushContent(agentID, ev, ev.Seq, ev.Ts); err != nil {
				return fmt.Errorf("index: annotation flush %s seq %d: %w", agentID, ev.Seq, err)
			}
			pendingFlush = false
		}
		if ev.Type == runtime.EvTurnEnd {
			var d runtime.TurnEndData
			_ = json.Unmarshal(ev.Data, &d)
			lastContext, lastCounts = d.ContextPct, d.ContextCounts
			if err := ix.OnEventAndTurnEnd(agentID, ev, runtime.TurnRollup{LastSeq: ev.Seq, LastContextPct: d.ContextPct, LastContext: d.ContextCounts, UpdatedAt: ev.Ts}); err != nil {
				return fmt.Errorf("index: turn_end %s seq %d: %w", agentID, ev.Seq, err)
			}
			pendingFlush = false
		} else if ev.Type != runtime.EvAnnotation {
			if err := ix.OnEvent(agentID, ev); err != nil {
				return fmt.Errorf("index: event %s seq %d: %w", agentID, ev.Seq, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("index: replay %s: %w", agentID, err)
	}
	if lastSeq > 0 && updatedAt != "" && pendingFlush {
		if err := ix.flush(agentID, runtime.TurnRollup{LastSeq: lastSeq, LastContextPct: lastContext, LastContext: lastCounts, UpdatedAt: updatedAt}, false, "partial"); err != nil {
			return fmt.Errorf("index: final flush %s: %w", agentID, err)
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

func readRetainedSessions(db *sql.DB) (map[string]retainedSession, error) {
	rows, err := db.Query(`SELECT agent_id, last_session_id, effort, fast FROM sessions`)
	if err != nil {
		return nil, fmt.Errorf("index: read retained session fields: %w", err)
	}
	defer rows.Close()
	out := map[string]retainedSession{}
	for rows.Next() {
		var id string
		var r retainedSession
		if err := rows.Scan(&id, &r.nativeID, &r.effort, &r.fast); err != nil {
			return nil, fmt.Errorf("index: scan retained session fields: %w", err)
		}
		out[id] = r
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("index: read retained session fields: %w", err)
	}
	return out, nil
}

// restoreRetainedSession reapplies retained fields to the replayed row. An
// empty retained ID never replaces the transcript's.
func restoreRetainedSession(db *sql.DB, agentID string, r retainedSession) error {
	if _, err := db.Exec(`
UPDATE sessions SET last_session_id = CASE WHEN ? <> '' THEN ? ELSE last_session_id END,
  effort = ?, fast = ?
WHERE agent_id = ?`, r.nativeID, r.nativeID, r.effort, r.fast, agentID); err != nil {
		return fmt.Errorf("index: restore retained session fields %s: %w", agentID, err)
	}
	return nil
}

func resetTables(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("index: begin reset: %w", err)
	}
	defer tx.Rollback()
	for _, stmt := range []string{
		`DELETE FROM sessions_fts`,
		`DELETE FROM tracked_commands`,
		`DELETE FROM tracked_files`,
		`DELETE FROM sessions`,
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("index: reset tables: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("index: commit reset: %w", err)
	}
	return nil
}
