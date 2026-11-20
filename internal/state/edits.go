package state

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Edit transition phases (TS-02.R69).
const (
	EditPreparing  = "preparing"
	EditCommitting = "committing"
	EditCommitted  = "committed"
	EditAborted    = "aborted"
)

// ErrEditOpen refuses a second nonterminal edit, or committed undispatched
// replacement, for one agent.
var ErrEditOpen = errors.New("state: another edit owns this conversation")

// ErrEditExists: the request id already names an edit for this agent.
var ErrEditExists = errors.New("state: edit request id already exists")

// ErrEditConflict: the edit is not in the phase this transition requires.
var ErrEditConflict = errors.New("state: edit phase changed")

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339Nano) }

// EditTransition is the bounded, durable record of one last-message Edit
// (TS-02.R69). MessageJSON holds the replacement input until its ordinary
// receipt takes over at dispatch; nothing here copies transcript content.
type EditTransition struct {
	AgentID          string
	RequestID        string
	PayloadHash      string
	MessageJSON      string
	MessageSeq       int64
	ExpectedRevision int64
	Revision         int64
	ReplacesFrom     int64
	ReplacesThrough  int64
	OldSession       string
	CandidateSession string
	Phase            string
	Dispatched       bool
	Reason           string
	CreatedAt        string
	UpdatedAt        string
}

// EditOperation is the snapshot's view of an edit that still owns input.
type EditOperation struct {
	RequestID string `json:"request_id"`
	Phase     string `json:"phase"`
	Reason    string `json:"reason,omitempty"`
}

// Edit gate reasons (FS-01.R61).
const (
	EditReasonNotChat   = "Only chat conversations can be edited."
	EditReasonArchived  = "Restore this agent before editing."
	EditReasonBusy      = "Wait for the agent to finish its current turn."
	EditReasonQuota     = "A quota continuation is pending; cancel it or let it run before editing."
	EditReasonEditing   = "An edit is already in progress."
	EditReasonNoFork    = "This agent's runtime cannot prepare an edited conversation."
	EditReasonNoSession = "This conversation has no provider session to edit."
)

// EditAvailability is the agent-level Edit gate the projection and the
// edit route share; per-message eligibility is checked by the route.
func EditAvailability(a AgentState, nativeSessionID string, open bool) CloneAffordance {
	switch {
	case a.Interface != "chat":
		return CloneAffordance{Reason: EditReasonNotChat}
	case a.Archived:
		return CloneAffordance{Reason: EditReasonArchived}
	case open:
		return CloneAffordance{Reason: EditReasonEditing}
	case CloneBusyState(a.State):
		return CloneAffordance{Reason: EditReasonBusy}
	case a.Quota != nil && (a.Quota.ContinuationState == QuotaScheduled || a.Quota.ContinuationState == "waiting" || a.Quota.ContinuationState == "starting"):
		return CloneAffordance{Reason: EditReasonQuota}
	case !a.RuntimeCapabilities.Fork:
		return CloneAffordance{Reason: EditReasonNoFork}
	case nativeSessionID == "":
		return CloneAffordance{Reason: EditReasonNoSession}
	}
	return CloneAffordance{Available: true}
}

// Open reports whether the record still owns its agent's input.
func (e EditTransition) Open() bool {
	return e.Phase == EditPreparing || e.Phase == EditCommitting || e.Phase == EditCommitted && !e.Dispatched
}

const editColumns = `agent_id, request_id, payload_hash, message_json, message_seq, expected_revision, revision,
replaces_from, replaces_through, old_session, candidate_session, phase, dispatched, reason, created_at, updated_at`

func scanEdit(row interface{ Scan(...any) error }) (EditTransition, error) {
	var e EditTransition
	err := row.Scan(&e.AgentID, &e.RequestID, &e.PayloadHash, &e.MessageJSON, &e.MessageSeq, &e.ExpectedRevision, &e.Revision,
		&e.ReplacesFrom, &e.ReplacesThrough, &e.OldSession, &e.CandidateSession, &e.Phase, &e.Dispatched, &e.Reason, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return EditTransition{}, ErrNotFound
	}
	return e, err
}

// CreateEdit durably accepts an edit in the preparing phase.
func (s *Store) CreateEdit(e EditTransition) error {
	now := nowUTC()
	_, err := s.db.Exec(`INSERT INTO edit_transitions (`+editColumns+`) VALUES (?,?,?,?,?,?,0,0,0,?,'',?,0,'',?,?)`,
		e.AgentID, e.RequestID, e.PayloadHash, e.MessageJSON, e.MessageSeq, e.ExpectedRevision, e.OldSession, EditPreparing, now, now)
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		if _, readErr := s.ReadEdit(e.AgentID, e.RequestID); readErr == nil {
			return ErrEditExists
		}
		return ErrEditOpen
	}
	if err != nil {
		return fmt.Errorf("state: create edit: %w", err)
	}
	return nil
}

// ReadEdit returns one edit by its request id.
func (s *Store) ReadEdit(agentID, requestID string) (EditTransition, error) {
	return scanEdit(s.db.QueryRow(`SELECT `+editColumns+` FROM edit_transitions WHERE agent_id=? AND request_id=?`, agentID, requestID))
}

// OpenEdit returns the agent's edit that still owns its input, if any.
func (s *Store) OpenEdit(agentID string) (EditTransition, error) {
	return scanEdit(s.db.QueryRow(`SELECT `+editColumns+` FROM edit_transitions WHERE agent_id=? AND
(phase IN ('preparing','committing') OR (phase='committed' AND dispatched=0))`, agentID))
}

// OpenEdits lists every edit startup recovery must examine.
func (s *Store) OpenEdits() ([]EditTransition, error) {
	rows, err := s.db.Query(`SELECT ` + editColumns + ` FROM edit_transitions WHERE
phase IN ('preparing','committing') OR (phase='committed' AND dispatched=0) ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("state: read open edits: %w", err)
	}
	defer rows.Close()
	out := []EditTransition{}
	for rows.Next() {
		e, err := scanEdit(rows)
		if err != nil {
			return nil, fmt.Errorf("state: scan open edit: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// EditLineageStep is one committed edit's native-session replacement: the
// candidate session continues the old one for every turn before keptBefore.
type EditLineageStep struct {
	OldSession, CandidateSession string
	KeptBefore                   int64
}

// EditLineage lists the agent's committed edits, newest first (TS-04.R113).
func (s *Store) EditLineage(agentID string) ([]EditLineageStep, error) {
	rows, err := s.db.Query(`SELECT old_session, candidate_session, replaces_from FROM edit_transitions
WHERE agent_id=? AND phase='committed' ORDER BY revision DESC`, agentID)
	if err != nil {
		return nil, fmt.Errorf("state: read edit lineage: %w", err)
	}
	defer rows.Close()
	var out []EditLineageStep
	for rows.Next() {
		var step EditLineageStep
		if err := rows.Scan(&step.OldSession, &step.CandidateSession, &step.KeptBefore); err != nil {
			return nil, fmt.Errorf("state: scan edit lineage: %w", err)
		}
		out = append(out, step)
	}
	return out, rows.Err()
}

// HistoryRevision is the agent's active conversation revision: the newest
// committing or committed decision, zero for legacy history.
func (s *Store) HistoryRevision(agentID string) (int64, error) {
	var rev int64
	err := s.db.QueryRow(`SELECT COALESCE(MAX(revision),0) FROM edit_transitions WHERE agent_id=? AND phase IN ('committing','committed')`, agentID).Scan(&rev)
	if err != nil {
		return 0, fmt.Errorf("state: read history revision: %w", err)
	}
	return rev, nil
}

// DecideEdit records the irreversible committing decision with its revision,
// superseded range and candidate selection, only from preparing. The
// revision is allocated in the same transaction.
func (s *Store) DecideEdit(agentID, requestID string, from, through int64, candidate string) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("state: begin edit decision: %w", err)
	}
	defer tx.Rollback()
	var rev int64
	if err := tx.QueryRow(`SELECT COALESCE(MAX(revision),0)+1 FROM edit_transitions WHERE agent_id=? AND phase IN ('committing','committed')`, agentID).Scan(&rev); err != nil {
		return 0, fmt.Errorf("state: allocate revision: %w", err)
	}
	res, err := tx.Exec(`UPDATE edit_transitions SET phase='committing', revision=?, replaces_from=?, replaces_through=?,
candidate_session=?, reason='', updated_at=? WHERE agent_id=? AND request_id=? AND phase='preparing'`,
		rev, from, through, candidate, nowUTC(), agentID, requestID)
	if err != nil {
		return 0, fmt.Errorf("state: decide edit: %w", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return 0, ErrEditConflict
	}
	return rev, tx.Commit()
}

// FinishEdit moves a committing edit to committed and selects its candidate
// session in the same transaction; the staged candidate never wrote it
// (TS-01.R51, TS-02.R69).
func (s *Store) FinishEdit(agentID, requestID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("state: begin edit finish: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE sessions SET last_session_id = (SELECT candidate_session FROM edit_transitions
WHERE agent_id=? AND request_id=? AND phase='committing' AND candidate_session<>'') WHERE agent_id=? AND EXISTS
(SELECT 1 FROM edit_transitions WHERE agent_id=? AND request_id=? AND phase='committing' AND candidate_session<>'')`,
		agentID, requestID, agentID, agentID, requestID); err != nil {
		return fmt.Errorf("state: select edit session: %w", err)
	}
	res, err := tx.Exec(`UPDATE edit_transitions SET phase='committed', reason='', updated_at=? WHERE agent_id=? AND request_id=?
AND phase IN ('committing','committed')`, nowUTC(), agentID, requestID)
	if err != nil {
		return fmt.Errorf("state: finish edit: %w", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrEditConflict
	}
	return tx.Commit()
}

// DispatchEdit hands a committed edit's replacement to its ordinary receipt
// in one transaction: the edit is marked dispatched exactly when the receipt
// exists, so a failure on either side leaves exactly one recovery owner —
// the open edit before, the receipt after (TS-02.R69). delivery is
// 'prepared' for admission or 'released' to offer it back for recovery.
func (s *Store) DispatchEdit(r AttachmentReceipt, delivery string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("state: begin edit dispatch: %w", err)
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE edit_transitions SET dispatched=1, updated_at=? WHERE agent_id=? AND request_id=? AND phase='committed' AND dispatched=0`,
		nowUTC(), r.AgentID, r.RequestID)
	if err != nil {
		return fmt.Errorf("state: mark edit dispatched: %w", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrEditConflict
	}
	if editDispatchFault != nil {
		if err := editDispatchFault(); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`INSERT INTO attachment_receipts(agent_id,request_id,payload_hash,message_json,action,delivery,updated_at) VALUES(?,?,?,?,?,?,?)`,
		r.AgentID, r.RequestID, r.PayloadHash, r.MessageJSON, r.Action, delivery, formatTime(time.Now().UTC())); err != nil {
		return fmt.Errorf("state: create edit receipt: %w", err)
	}
	return tx.Commit()
}

// editDispatchFault is a test seam between the two writes of DispatchEdit.
var editDispatchFault func() error

// AbortEdit ends a preparing edit; the original conversation stays active.
func (s *Store) AbortEdit(agentID, requestID, reason string) error {
	return s.editUpdate(`phase='aborted', reason=`+sqlQuote(clipReason(reason)), `phase='preparing'`, agentID, requestID)
}

// SetEditReason records a bounded attention reason without changing phase.
func (s *Store) SetEditReason(agentID, requestID, reason string) error {
	return s.editUpdate(`reason=`+sqlQuote(clipReason(reason)), `1=1`, agentID, requestID)
}

func (s *Store) editUpdate(set, where, agentID, requestID string) error {
	res, err := s.db.Exec(`UPDATE edit_transitions SET `+set+`, updated_at=? WHERE agent_id=? AND request_id=? AND (`+where+`)`, nowUTC(), agentID, requestID)
	if err != nil {
		return fmt.Errorf("state: update edit: %w", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrEditConflict
	}
	return nil
}

func clipReason(reason string) string {
	if len(reason) > 1024 {
		reason = reason[:1024]
	}
	return reason
}

func sqlQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
