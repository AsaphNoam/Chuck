package index

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/strutil"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// execer is satisfied by both *sql.DB and *sql.Tx so rollup updates can run
// either standalone or inside an existing transaction.
type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

type queryRower interface {
	QueryRow(query string, args ...any) *sql.Row
}
type sqlRunner interface {
	execer
	queryRower
}

type Indexer struct {
	db          *sql.DB
	runner      sqlRunner
	tx          *sql.Tx
	mu          sync.Mutex
	content     map[string][]bufferedText
	fileCalls   map[string]map[string]fileCall // agent -> open toolCallID -> counted file
	ftsWritable func(*sql.Tx) (bool, error)

	// Tests use this hook to prove that a later event cannot enter between a
	// boundary event and its flush. It is nil in production.
	beforeBoundaryFlush func()
}

type bufferedText struct {
	seq        int64
	text       string
	provenance *SearchProvenance
	spanCount  int
	spanBytes  int
}

// fileCall is the file an open tool call already counted as one edit, so that
// call's later input updates do not count it again (TS-04.R98, FS-05.R15).
type fileCall struct {
	path       string
	activityID string
	seq        int64
}

// maxOpenFileCalls bounds one agent's open-call memory; past it the oldest
// call is forgotten (INV §16).
const maxOpenFileCalls = 64

func New(db *sql.DB) *Indexer {
	return &Indexer{db: db, runner: db, content: map[string][]bufferedText{}, fileCalls: map[string]map[string]fileCall{}, ftsWritable: ftsWritesAvailable}
}

func newTxIndexer(db *sql.DB, tx *sql.Tx) *Indexer {
	return &Indexer{db: db, runner: tx, tx: tx, content: map[string][]bufferedText{}, fileCalls: map[string]map[string]fileCall{}, ftsWritable: ftsWritesAvailable}
}

func (ix *Indexer) UpsertSessionMeta(agentID string, meta runtime.SessionMetaData) error {
	if agentID == "" {
		return fmt.Errorf("index: agent id is required")
	}
	// created_at/updated_at must be a real timestamp; fall back to now (never the
	// session id, which would sort lexically after real RFC3339 values and corrupt
	// the MAX(updated_at) guard below).
	now := meta.CreatedAt
	if now == "" {
		now = time.Now().UTC().Format(time.RFC3339)
	}
	envKeys, err := json.Marshal(meta.EnvKeys)
	if err != nil {
		return fmt.Errorf("index: marshal env keys: %w", err)
	}
	addDirs, err := json.Marshal(meta.AddDirs)
	if err != nil {
		return fmt.Errorf("index: marshal add dirs: %w", err)
	}
	launchConfig := "{}"
	if len(meta.LaunchConfig) > 0 {
		launchConfig = string(meta.LaunchConfig)
	}
	contract, err := state.EncodeExecutionContract(meta.ExecutionContract)
	if err != nil {
		return fmt.Errorf("index: encode execution contract: %w", err)
	}
	capabilities := "{}"
	if meta.RuntimeCapabilities != nil {
		raw, err := json.Marshal(meta.RuntimeCapabilities)
		if err != nil {
			return fmt.Errorf("index: marshal runtime capabilities: %w", err)
		}
		capabilities = string(raw)
	}
	chuckAware := meta.IsChuckAware()
	// Put the session upsert and metadata document replacement in one
	// transaction so both succeed or both roll back (TS-02.R16, INV §15).
	tx := ix.tx
	owned := tx == nil
	if owned {
		var err error
		tx, err = ix.db.Begin()
		if err != nil {
			return fmt.Errorf("index: begin metadata transaction: %w", err)
		}
		defer tx.Rollback()
	}
	_, err = tx.Exec(`
INSERT INTO sessions(agent_id, name, role, project, backend, model, effort, fast, interface, grp, cwd, system_prompt, chuck_aware, env_keys, skip_permissions, add_dirs, launch_config_json, runtime_capabilities_json, forked_from_agent_id, forked_from_seq, last_session_id, created_at, updated_at, history_primer, execution_contract_json, workspace_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, 0), ?, ?, ?, ?, ?, ?)
ON CONFLICT(agent_id) DO UPDATE SET
  name=excluded.name,
  role=excluded.role,
  project=excluded.project,
  backend=excluded.backend,
  model=excluded.model,
	 effort=excluded.effort,
	 fast=excluded.fast,
  interface=excluded.interface,
  grp=excluded.grp,
  cwd=excluded.cwd,
  workspace_id=CASE WHEN excluded.workspace_id <> '' THEN excluded.workspace_id ELSE sessions.workspace_id END,
  system_prompt=excluded.system_prompt,
  history_primer=excluded.history_primer,
  chuck_aware=excluded.chuck_aware,
  env_keys=excluded.env_keys,
  skip_permissions=excluded.skip_permissions,
  add_dirs=excluded.add_dirs,
  launch_config_json=excluded.launch_config_json,
  execution_contract_json=COALESCE(excluded.execution_contract_json, sessions.execution_contract_json),
  runtime_capabilities_json=excluded.runtime_capabilities_json,
  forked_from_agent_id=COALESCE(excluded.forked_from_agent_id, sessions.forked_from_agent_id),
  forked_from_seq=COALESCE(excluded.forked_from_seq, sessions.forked_from_seq),
  last_session_id=excluded.last_session_id,
  last_context_pct=CASE WHEN ? THEN 0 ELSE sessions.last_context_pct END,
  last_context_used=CASE WHEN ? THEN NULL ELSE sessions.last_context_used END,
  last_context_size=CASE WHEN ? THEN NULL ELSE sessions.last_context_size END,
  updated_at=MAX(excluded.updated_at, sessions.updated_at)`,
		agentID, meta.Name, meta.Role, meta.Project, meta.Backend, meta.Model, meta.Effort, meta.Fast, meta.Interface,
		meta.Group, meta.Cwd, meta.SystemPrompt, chuckAware, string(envKeys), meta.SkipPermissions, string(addDirs), launchConfig, capabilities, meta.ForkedFromAgentID, meta.ForkedFromSeq, meta.SessionID, now, now, meta.HistoryPrimer, contract, meta.WorkspaceID, meta.ResetContext, meta.ResetContext, meta.ResetContext)
	if err != nil {
		return fmt.Errorf("index: upsert session meta: %w", err)
	}
	ftsWritable, err := ix.ftsWritable(tx)
	if err != nil {
		return err
	}
	if ftsWritable {
		if err := replaceMetadataDocument(tx, agentID); err != nil {
			return err
		}
	}
	if owned {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("index: commit metadata transaction: %w", err)
		}
	}
	return nil
}

func (ix *Indexer) OnEvent(agentID string, ev runtime.Event) error {
	if agentID == "" {
		agentID = ev.AgentID
	}
	if agentID == "" {
		return fmt.Errorf("index: agent id is required")
	}
	if ev.Type == runtime.EvSessionMeta {
		var meta runtime.SessionMetaData
		if err := json.Unmarshal(ev.Data, &meta); err != nil {
			return fmt.Errorf("index: decode session_meta: %w", err)
		}
		return ix.UpsertSessionMeta(agentID, meta)
	}
	ix.mu.Lock()
	defer ix.mu.Unlock()
	return ix.onEventLocked(agentID, ev)
}

func (ix *Indexer) onEventLocked(agentID string, ev runtime.Event) error {
	document, err := BuildSearchDocument([]runtime.Event{ev})
	if err != nil {
		return err
	}
	if document.Content != "" {
		ix.addDocumentLocked(agentID, ev.Seq, document)
	}
	if err := ix.trackEvent(agentID, ev); err != nil {
		return err
	}
	_, err = updateEventCounters(ix.runner, agentID, ev)
	return err
}

func updateEventCounters(db execer, agentID string, ev runtime.Event) (int64, error) {
	if ev.Seq > 0 {
		res, err := db.Exec(`
UPDATE sessions
SET event_count = event_count + 1,
    last_seq = CASE WHEN last_seq < ? THEN ? ELSE last_seq END,
    updated_at = CASE WHEN updated_at < ? THEN ? ELSE updated_at END
WHERE agent_id = ?`, ev.Seq, ev.Seq, ev.Ts, ev.Ts, agentID)
		if err != nil {
			return 0, fmt.Errorf("index: update event counters: %w", err)
		}
		return res.RowsAffected()
	}
	return 0, nil
}

// OnQuotaNotice indexes a durable notice and acknowledges its attempt in one
// transaction. The quota receipt is the idempotence marker: a crash after the
// transcript append but before this commit leaves both updates pending.
func (ix *Indexer) OnQuotaNotice(q state.QuotaIntent, ev runtime.Event) error {
	return ix.onQuotaNotice(q, ev, nil)
}

// TranscriptCounters are the durable transcript's session-index counters.
// Stopped notice repair uses them because an older live writer could have
// indexed the notice before crashing without confirming its receipt.
type TranscriptCounters struct {
	EventCount int64
	LastSeq    int64
	UpdatedAt  string
}

func (ix *Indexer) ReconcileQuotaNotice(q state.QuotaIntent, ev runtime.Event, counters TranscriptCounters) error {
	if counters.EventCount < 1 || counters.LastSeq < ev.Seq || counters.UpdatedAt < ev.Ts {
		return fmt.Errorf("index: invalid quota transcript counters")
	}
	return ix.onQuotaNotice(q, ev, &counters)
}

func (ix *Indexer) onQuotaNotice(q state.QuotaIntent, ev runtime.Event, counters *TranscriptCounters) error {
	if ev.Type != runtime.EvNotice || ev.AgentID != q.AgentID || ev.Seq <= 0 || ev.Ts == "" {
		return fmt.Errorf("index: invalid quota notice event")
	}
	var notice runtime.NoticeData
	if err := json.Unmarshal(ev.Data, &notice); err != nil || notice.Origin != "chuck" || notice.ContinuationAttempt == "" || notice.ContinuationAttempt != q.NoticeKey {
		return fmt.Errorf("index: quota notice identity mismatch")
	}
	ix.mu.Lock()
	defer ix.mu.Unlock()
	tx, err := ix.db.Begin()
	if err != nil {
		return fmt.Errorf("index: begin quota notice transaction: %w", err)
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE quota_intents SET revision=revision+1, notice_confirmed=1, updated_at=?
WHERE agent_id=? AND episode_id=? AND revision=? AND state='claimed' AND attempt_token=?
AND provider_started=1 AND notice_key=? AND notice_confirmed=0`,
		time.Now().UTC().Format(time.RFC3339), q.AgentID, q.EpisodeID, q.Revision, q.AttemptToken, q.NoticeKey)
	if err != nil {
		return fmt.Errorf("index: acknowledge quota notice: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n != 1 {
		return state.ErrQuotaConflict
	}
	var affected int64
	if counters == nil {
		affected, err = updateEventCounters(tx, q.AgentID, ev)
	} else {
		var result sql.Result
		result, err = tx.Exec(`UPDATE sessions SET event_count=?, last_seq=?,
updated_at=CASE WHEN updated_at < ? THEN ? ELSE updated_at END WHERE agent_id=?`,
			counters.EventCount, counters.LastSeq, counters.UpdatedAt, counters.UpdatedAt, q.AgentID)
		if err == nil {
			affected, err = result.RowsAffected()
		}
	}
	if err != nil {
		return fmt.Errorf("index: update quota notice counters: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("index: quota notice session metadata is missing")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("index: commit quota notice transaction: %w", err)
	}
	return nil
}

func (ix *Indexer) OnTurnEnd(agentID string, rollup runtime.TurnRollup) error {
	return ix.flush(agentID, rollup, true, "turn")
}

// OnEventAndTurnEnd consumes the persisted boundary event and flushes through
// that exact sequence under one lock. Later events remain buffered for the next
// immutable document even if their goroutine reached the indexer first.
func (ix *Indexer) OnEventAndTurnEnd(agentID string, ev runtime.Event, rollup runtime.TurnRollup) error {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	if err := ix.onEventLocked(agentID, ev); err != nil {
		return err
	}
	if ix.beforeBoundaryFlush != nil {
		ix.beforeBoundaryFlush()
	}
	return ix.flushLocked(agentID, rollup, true, "turn")
}

// FlushContent persists indexed content that arrives outside an ACP turn, such
// as a durable dashboard annotation. It intentionally does not increase the
// turn count.
func (ix *Indexer) FlushContent(agentID string, lastSeq int64, updatedAt string) error {
	return ix.flush(agentID, runtime.TurnRollup{LastSeq: lastSeq, UpdatedAt: updatedAt}, false, "flush")
}

func (ix *Indexer) OnEventAndFlushContent(agentID string, ev runtime.Event, lastSeq int64, updatedAt string) error {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	if err := ix.onEventLocked(agentID, ev); err != nil {
		return err
	}
	if ix.beforeBoundaryFlush != nil {
		ix.beforeBoundaryFlush()
	}
	return ix.flushLocked(agentID, runtime.TurnRollup{LastSeq: lastSeq, UpdatedAt: updatedAt}, false, "flush")
}

func (ix *Indexer) flush(agentID string, rollup runtime.TurnRollup, countTurn bool, documentKind string) error {
	if agentID == "" {
		return fmt.Errorf("index: agent id is required")
	}
	ix.mu.Lock()
	defer ix.mu.Unlock()
	return ix.flushLocked(agentID, rollup, countTurn, documentKind)
}

// FlushPendingContent persists and releases text buffered through lastSeq
// without touching session rollups. A generation closing mid-turn calls it
// because no turn_end will flush that text (TS-02.R16).
func (ix *Indexer) FlushPendingContent(agentID string, lastSeq int64) error {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	content, retained := ix.pendingContentLocked(agentID, lastSeq)
	if content.Content == "" {
		return nil
	}
	tx := ix.tx
	owned := tx == nil
	if owned {
		var err error
		tx, err = ix.db.Begin()
		if err != nil {
			return fmt.Errorf("index: begin content flush: %w", err)
		}
		defer tx.Rollback()
	}
	ftsWritable, err := ix.ftsWritable(tx)
	if err != nil {
		return err
	}
	if ftsWritable {
		if err := insertTranscriptDocument(tx, agentID, fmt.Sprintf("close:%d", lastSeq), content); err != nil {
			return err
		}
	}
	if owned {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("index: commit content flush: %w", err)
		}
	}
	ix.retainContentLocked(agentID, retained)
	return nil
}

// pendingContentLocked joins buffered text through lastSeq in sequence order
// and returns the entries that remain buffered.
func (ix *Indexer) pendingContentLocked(agentID string, lastSeq int64) (SearchDocument, []bufferedText) {
	entries := ix.content[agentID]
	selected := make([]bufferedText, 0, len(entries))
	retained := make([]bufferedText, 0, len(entries))
	for _, entry := range entries {
		if entry.seq <= lastSeq {
			selected = append(selected, entry)
		} else {
			retained = append(retained, entry)
		}
	}
	sort.SliceStable(selected, func(i, j int) bool { return selected[i].seq < selected[j].seq })
	var builder SearchDocumentBuilder
	for _, entry := range selected {
		builder.appendDocument(SearchDocument{Content: entry.text, Provenance: entry.provenance})
	}
	return builder.Document(), retained
}

func (ix *Indexer) retainContentLocked(agentID string, retained []bufferedText) {
	if len(retained) == 0 {
		delete(ix.content, agentID)
	} else {
		count, bytes := 0, 0
		for i := range retained {
			if retained[i].provenance != nil {
				count += len(retained[i].provenance.Spans)
				raw, _ := json.Marshal(retained[i].provenance)
				bytes += len(raw)
			}
			retained[i].spanCount = count
			retained[i].spanBytes = bytes
		}
		ix.content[agentID] = retained
	}
}

func (ix *Indexer) flushLocked(agentID string, rollup runtime.TurnRollup, countTurn bool, documentKind string) error {
	content, retained := ix.pendingContentLocked(agentID, rollup.LastSeq)

	tx := ix.tx
	owned := tx == nil
	if owned {
		var err error
		tx, err = ix.db.Begin()
		if err != nil {
			return fmt.Errorf("index: begin turn flush: %w", err)
		}
		defer tx.Rollback()
	}

	if content.Content != "" {
		ftsWritable, err := ix.ftsWritable(tx)
		if err != nil {
			return err
		}
		if ftsWritable {
			documentID := fmt.Sprintf("%s:%d", documentKind, rollup.LastSeq)
			if err := insertTranscriptDocument(tx, agentID, documentID, content); err != nil {
				return err
			}
		}
	}
	turnInc := 0
	if countTurn {
		turnInc = 1
	}
	// The exact pair travels with the percentage; NULL means unknown (TS-02.R39).
	var lastUsed, lastSize any
	if c := rollup.LastContext; c != nil {
		lastUsed, lastSize = c.ContextUsed, c.ContextSize
	}
	if _, err := tx.Exec(`
UPDATE sessions
SET turn_count = turn_count + ?,
    last_seq = CASE WHEN last_seq < ? THEN ? ELSE last_seq END,
    last_context_pct = ?,
    last_context_used = ?,
    last_context_size = ?,
    updated_at = CASE WHEN ? <> '' THEN ? ELSE updated_at END,
    files_touched = (SELECT COUNT(*) FROM tracked_files WHERE agent_id = ?),
    commands_run = (SELECT COUNT(*) FROM tracked_commands WHERE agent_id = ?)
WHERE agent_id = ?`,
		turnInc, rollup.LastSeq, rollup.LastSeq, rollup.LastContextPct, lastUsed, lastSize, rollup.UpdatedAt, rollup.UpdatedAt, agentID, agentID, agentID); err != nil {
		return fmt.Errorf("index: update turn rollup: %w", err)
	}
	if owned {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("index: commit turn flush: %w", err)
		}
	}
	ix.retainContentLocked(agentID, retained)
	return nil
}

func (ix *Indexer) addContentLocked(agentID string, seq int64, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	ix.content[agentID] = append(ix.content[agentID], bufferedText{seq: seq, text: text})
}

func (ix *Indexer) addDocumentLocked(agentID string, seq int64, d SearchDocument) {
	entries := ix.content[agentID]
	count, encoded := 0, 0
	unavailable := d.Provenance == nil
	if len(entries) > 0 {
		e := entries[len(entries)-1]
		unavailable = unavailable || e.provenance == nil
		count = e.spanCount
		encoded = e.spanBytes
	}
	if d.Provenance != nil {
		count += len(d.Provenance.Spans)
		raw, _ := json.Marshal(d.Provenance)
		encoded += len(raw)
	}
	if unavailable || count > MaxSearchSpans || encoded > MaxSearchProvenanceBytes {
		if len(entries) == 0 || entries[len(entries)-1].provenance != nil {
			for i := range entries {
				entries[i].provenance = nil
			}
		}
		d.Provenance = nil
	}
	ix.content[agentID] = append(entries, bufferedText{seq: seq, text: d.Content, provenance: d.Provenance, spanCount: count, spanBytes: encoded})
}

func replaceMetadataDocument(tx *sql.Tx, agentID string) error {
	var name, role, project, grp, model, backend string
	if err := tx.QueryRow(`SELECT name, role, project, grp, model, backend FROM sessions WHERE agent_id = ?`, agentID).
		Scan(&name, &role, &project, &grp, &model, &backend); err != nil {
		return fmt.Errorf("index: read session for fts: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM sessions_fts WHERE agent_id = ? AND document_id = 'metadata'`, agentID); err != nil {
		return fmt.Errorf("index: delete metadata document: %w", err)
	}
	if _, err := tx.Exec(`
INSERT INTO sessions_fts(agent_id, document_id, name, role, project, grp, model, backend, content)
VALUES (?, 'metadata', ?, ?, ?, ?, ?, ?, '')`, agentID, name, role, project, grp, model, backend); err != nil {
		return fmt.Errorf("index: insert metadata document: %w", err)
	}
	return nil
}

func insertTranscriptDocument(tx *sql.Tx, agentID, documentID string, document SearchDocument) error {
	if _, err := tx.Exec(`DELETE FROM session_search_spans WHERE agent_id=? AND document_id=?`, agentID, documentID); err != nil {
		return err
	}
	if _, err := tx.Exec(`
INSERT INTO sessions_fts(agent_id, document_id, name, role, project, grp, model, backend, content)
VALUES (?, ?, '', '', '', '', '', '', ?)`, agentID, documentID, document.Content); err != nil {
		return fmt.Errorf("index: insert transcript document: %w", err)
	}
	if document.Provenance != nil && document.Content != "" {
		raw, err := json.Marshal(document.Provenance)
		if err != nil {
			return err
		}
		if len(raw) <= MaxSearchProvenanceBytes {
			if _, err := tx.Exec(`INSERT OR REPLACE INTO session_search_spans(agent_id,document_id,version,content_sha256,spans_json) VALUES(?,?,1,?,?)`, agentID, documentID, ContentSHA256(document.Content), string(raw)); err != nil {
				return fmt.Errorf("index: insert search provenance: %w", err)
			}
		}
	}
	return nil
}

func ftsWritesAvailable(tx *sql.Tx) (bool, error) {
	mode, err := state.DetectSessionsFTS(tx)
	return mode != state.SessionsFTSUnavailable, err
}

// searchableText derives the full-text search bag from the one shared
// normalized-event projection, so search text and pulled context cannot drift
// into two independent event-type switches (INV §2, TS-01.R22).
func searchableText(ev runtime.Event) (string, error) {
	projection, err := transcript.ProjectEvent(ev)
	if err != nil {
		return "", fmt.Errorf("index: %w", err)
	}
	return projection.SearchText(), nil
}

func (ix *Indexer) trackEvent(agentID string, ev runtime.Event) error {
	switch ev.Type {
	case runtime.EvToolCall:
		var d runtime.ToolCallData
		if err := json.Unmarshal(ev.Data, &d); err != nil {
			return fmt.Errorf("index: track tool_call: %w", err)
		}
		if p := pathFromArgs(d.Args); p != "" && isFileTool(d.Name) {
			display, err := ix.upsertFile(agentID, p, ev, d.ToolCallID, false)
			if err != nil {
				return err
			}
			ix.noteFileCallLocked(agentID, d.ToolCallID, display, ev)
		}
		if cmd := commandFromArgs(d.Args); cmd != "" && isCommandTool(d.Name) {
			if _, err := ix.runner.Exec(`
INSERT OR REPLACE INTO tracked_commands(agent_id, seq, ts, tool_call_id, command, exit_status, exit_error)
VALUES (?, ?, ?, ?, ?, 'in_progress', '')`, agentID, ev.Seq, ev.Ts, d.ToolCallID, cmd); err != nil {
				return fmt.Errorf("index: insert command: %w", err)
			}
		}
	case runtime.EvToolCallUpdate:
		// Later input for a call (TS-04.R98): count the file it names as that
		// call's one edit unless the call already counted it, and record or
		// correct the call's one command row.
		var d runtime.ToolCallUpdateData
		if err := json.Unmarshal(ev.Data, &d); err != nil {
			return fmt.Errorf("index: track tool_call_update: %w", err)
		}
		if p := pathFromArgs(d.Args); p != "" && isFileTool(d.Name) {
			display, _, err := ix.normalizePath(agentID, p)
			if err != nil {
				return err
			}
			if ix.fileCalls[agentID][d.ToolCallID].path != display {
				if _, err := ix.upsertFile(agentID, p, ev, d.ToolCallID, false); err != nil {
					return err
				}
				ix.noteFileCallLocked(agentID, d.ToolCallID, display, ev)
			}
		}
		if cmd := commandFromArgs(d.Args); cmd != "" && isCommandTool(d.Name) {
			res, err := ix.runner.Exec(`UPDATE tracked_commands SET command = ? WHERE agent_id = ? AND tool_call_id = ?`, cmd, agentID, d.ToolCallID)
			if err != nil {
				return fmt.Errorf("index: update command: %w", err)
			}
			if n, _ := res.RowsAffected(); n == 0 {
				if _, err := ix.runner.Exec(`
INSERT OR REPLACE INTO tracked_commands(agent_id, seq, ts, tool_call_id, command, exit_status, exit_error)
VALUES (?, ?, ?, ?, ?, 'in_progress', '')`, agentID, ev.Seq, ev.Ts, d.ToolCallID, cmd); err != nil {
					return fmt.Errorf("index: insert command: %w", err)
				}
			}
		}
	case runtime.EvToolResult:
		var d runtime.ToolResultData
		if err := json.Unmarshal(ev.Data, &d); err != nil {
			return fmt.Errorf("index: track tool_result: %w", err)
		}
		ix.releaseFileCallsLocked(agentID, func(id string, _ fileCall) bool { return id == d.ToolCallID })
		if _, err := ix.runner.Exec(`
UPDATE tracked_commands
SET exit_status = ?, exit_error = ?
WHERE agent_id = ? AND tool_call_id = ?`, strutil.FirstNonEmpty(d.Status, "completed"), d.Error, agentID, d.ToolCallID); err != nil {
			return fmt.Errorf("index: update command result: %w", err)
		}
	case runtime.EvFileReport:
		// A file-change report only adds paths no tool call or diff already
		// tracked; it never counts an edit twice or claims an inspectable patch
		// (FS-05.R38, TS-04.R66).
		var d runtime.FileReportData
		if err := json.Unmarshal(ev.Data, &d); err != nil {
			return fmt.Errorf("index: track file report: %w", err)
		}
		for _, p := range d.Paths {
			display, abs, err := ix.normalizePath(agentID, p)
			if err != nil {
				return err
			}
			if _, err := ix.runner.Exec(`
INSERT INTO tracked_files(agent_id, path, abs_path, edit_count, first_seq, last_seq, first_ts, last_ts, has_diff, diff_refs)
VALUES (?, ?, ?, 1, ?, ?, ?, ?, 0, '[]')
ON CONFLICT(agent_id, path) DO NOTHING`, agentID, display, abs, ev.Seq, ev.Seq, ev.Ts, ev.Ts); err != nil {
				return fmt.Errorf("index: track reported file: %w", err)
			}
		}
	case runtime.EvDiff:
		var d runtime.DiffData
		if err := json.Unmarshal(ev.Data, &d); err != nil {
			return fmt.Errorf("index: track diff: %w", err)
		}
		if d.Path != "" {
			if _, err := ix.upsertFile(agentID, d.Path, ev, d.ToolCallID, true); err != nil {
				return err
			}
		}
	case runtime.EvTurnEnd:
		// A canceled or failed call may end without a result; its turn ending
		// releases it. Child activities' calls wait for their own end.
		ix.releaseFileCallsLocked(agentID, func(_ string, c fileCall) bool { return c.activityID == "" })
	case runtime.EvActivityState:
		var d runtime.ActivityStateData
		if err := json.Unmarshal(ev.Data, &d); err == nil && d.State != "disconnected" {
			ix.releaseFileCallsLocked(agentID, func(_ string, c fileCall) bool { return c.activityID == ev.ActivityID })
		}
	}
	return nil
}

func (ix *Indexer) noteFileCallLocked(agentID, toolCallID, display string, ev runtime.Event) {
	calls := ix.fileCalls[agentID]
	if calls == nil {
		calls = map[string]fileCall{}
		ix.fileCalls[agentID] = calls
	}
	calls[toolCallID] = fileCall{path: display, activityID: ev.ActivityID, seq: ev.Seq}
	for len(calls) > maxOpenFileCalls {
		oldest := ""
		for id, c := range calls {
			if oldest == "" || c.seq < calls[oldest].seq {
				oldest = id
			}
		}
		delete(calls, oldest)
	}
}

func (ix *Indexer) releaseFileCallsLocked(agentID string, release func(string, fileCall) bool) {
	calls := ix.fileCalls[agentID]
	for id, c := range calls {
		if release(id, c) {
			delete(calls, id)
		}
	}
	if len(calls) == 0 {
		delete(ix.fileCalls, agentID)
	}
}

// upsertFile counts one edit of p and returns its display path.
func (ix *Indexer) upsertFile(agentID, p string, ev runtime.Event, toolCallID string, hasDiff bool) (string, error) {
	display, abs, err := ix.normalizePath(agentID, p)
	if err != nil {
		return "", err
	}
	diffRefs := "[]"
	if hasDiff {
		ref, _ := json.Marshal([]map[string]any{{"seq": ev.Seq, "tool_call_id": toolCallID}})
		diffRefs = string(ref)
	}
	_, err = ix.runner.Exec(`
INSERT INTO tracked_files(agent_id, path, abs_path, edit_count, first_seq, last_seq, first_ts, last_ts, has_diff, diff_refs)
VALUES (?, ?, ?, 1, ?, ?, ?, ?, ?, ?)
ON CONFLICT(agent_id, path) DO UPDATE SET
  edit_count = tracked_files.edit_count + 1,
  last_seq = excluded.last_seq,
  last_ts = excluded.last_ts,
  has_diff = CASE WHEN excluded.has_diff = 1 THEN 1 ELSE tracked_files.has_diff END,
  diff_refs = CASE
    WHEN excluded.has_diff = 1 AND tracked_files.diff_refs = '[]' THEN excluded.diff_refs
    WHEN excluded.has_diff = 1 THEN substr(tracked_files.diff_refs, 1, length(tracked_files.diff_refs)-1) || ',' || substr(excluded.diff_refs, 2)
    ELSE tracked_files.diff_refs
  END`,
		agentID, display, abs, ev.Seq, ev.Seq, ev.Ts, ev.Ts, boolInt(hasDiff), diffRefs)
	if err != nil {
		return "", fmt.Errorf("index: upsert file: %w", err)
	}
	return display, nil
}

func (ix *Indexer) normalizePath(agentID, p string) (display, abs string, err error) {
	cwd := ""
	if err := ix.runner.QueryRow(`SELECT cwd FROM sessions WHERE agent_id = ?`, agentID).Scan(&cwd); err != nil && err != sql.ErrNoRows {
		return "", "", fmt.Errorf("index: read session cwd: %w", err)
	}
	if filepath.IsAbs(p) {
		abs = filepath.Clean(p)
		if cwd != "" {
			if rel, err := filepath.Rel(cwd, abs); err == nil && !strings.HasPrefix(rel, "..") {
				return filepath.ToSlash(rel), abs, nil
			}
		}
		return abs, abs, nil
	}
	display = filepath.ToSlash(filepath.Clean(p))
	if cwd != "" {
		abs = filepath.Join(cwd, p)
	} else {
		abs = p
	}
	return display, abs, nil
}

func pathFromArgs(raw json.RawMessage) string {
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		return ""
	}
	for _, key := range []string{"path", "file_path", "filepath"} {
		if v, ok := args[key].(string); ok {
			return v
		}
	}
	return ""
}

func commandFromArgs(raw json.RawMessage) string {
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		return ""
	}
	for _, key := range []string{"command", "cmd", "script"} {
		if v, ok := args[key].(string); ok {
			return v
		}
	}
	return ""
}

func isFileTool(name string) bool {
	switch strings.ToLower(name) {
	case "edit", "write", "multiedit", "notebookedit", "create":
		return true
	default:
		return false
	}
}

func isCommandTool(name string) bool {
	switch strings.ToLower(name) {
	case "bash", "shell", "run", "terminal", "exec_command", "execute": // execute: ACP tool kind
		return true
	default:
		return false
	}
}

// CaptureHookFile records a file-edit event from POST /api/hook (terminal runtime producer).
// path is the edited file path; ts is RFC3339; seq is an optional transcript seq (0 if unknown).
func (ix *Indexer) CaptureHookFile(agentID, path, ts string, seq int64) error {
	if agentID == "" || path == "" {
		return fmt.Errorf("index: agent_id and path are required")
	}
	ev := runtime.Event{AgentID: agentID, Seq: seq, Ts: ts}
	if _, err := ix.upsertFile(agentID, path, ev, "", false); err != nil {
		return err
	}
	return bumpRollups(ix.db, agentID)
}

// CaptureHookCommand records a command event from POST /api/hook (terminal runtime producer).
// command is the shell command; ts is RFC3339; seq is an optional transcript seq (0 if unknown).
func (ix *Indexer) CaptureHookCommand(agentID, command, ts, toolCallID string, seq int64) error {
	if agentID == "" || command == "" {
		return fmt.Errorf("index: agent_id and command are required")
	}
	tx, err := ix.db.Begin()
	if err != nil {
		return fmt.Errorf("index: begin capture command: %w", err)
	}
	defer tx.Rollback()
	// Hook callers may omit seq (terminal producer with no transcript seq). seq<=0
	// would collide on the (agent_id, seq) PK and overwrite prior commands, so
	// allocate the next free seq for this agent instead.
	if seq <= 0 {
		if err := tx.QueryRow(`SELECT COALESCE(MAX(seq), 0) + 1 FROM tracked_commands WHERE agent_id = ?`, agentID).Scan(&seq); err != nil {
			return fmt.Errorf("index: next command seq: %w", err)
		}
	}
	if toolCallID == "" {
		toolCallID = fmt.Sprintf("hook_%d", seq)
	}
	if _, err := tx.Exec(`
INSERT OR REPLACE INTO tracked_commands(agent_id, seq, ts, tool_call_id, command, exit_status, exit_error)
VALUES (?, ?, ?, ?, ?, 'completed', '')`, agentID, seq, ts, toolCallID, command); err != nil {
		return fmt.Errorf("index: capture hook command: %w", err)
	}
	if err := bumpRollups(tx, agentID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("index: commit capture command: %w", err)
	}
	return nil
}

// bumpRollups refreshes the sessions row's files_touched/commands_run counts from
// the tracked_* tables. The runtime hot path maintains these on turn_end, but the
// hook-capture path has no turn boundary, so it must refresh them directly. A
// no-op (0 rows) when the agent has no sessions row yet (e.g. terminal-only).
func bumpRollups(db execer, agentID string) error {
	if _, err := db.Exec(`
UPDATE sessions
SET files_touched = (SELECT COUNT(*) FROM tracked_files WHERE agent_id = ?),
    commands_run  = (SELECT COUNT(*) FROM tracked_commands WHERE agent_id = ?)
WHERE agent_id = ?`, agentID, agentID, agentID); err != nil {
		return fmt.Errorf("index: bump rollups: %w", err)
	}
	return nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
