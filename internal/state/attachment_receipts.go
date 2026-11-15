package state

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// AttachmentReceipt records acceptance before any provider-visible effect.
// MessageJSON contains descriptors only; immutable bytes live in conversation storage.
// AssignmentTaskID names the assignment whose continuation will carry it.
type AttachmentReceipt struct {
	AgentID, RequestID, PayloadHash, MessageJSON, Action, Delivery, Outcome string
	AssignmentTaskID                                                        string
	// OriginTaskID identifies the paused visit, even after a replacement or
	// delivery clears AssignmentTaskID. Its lineage identifies the run.
	OriginTaskID, OriginQuestion string
	// SourceAgentID is set, durably, when a claim moved queued input to its
	// assignment's current agent: this earlier conversation accepted it, owns
	// the original attachment bytes, and its replays still find the receipt
	// (TS-09.R69).
	SourceAgentID string
}

const attachmentReceiptColumns = `agent_id,request_id,payload_hash,message_json,action,delivery,outcome,assignment_task_id,source_agent_id,origin_task_id,origin_question`

func scanAttachmentReceipt(row *sql.Row) (AttachmentReceipt, error) {
	var r AttachmentReceipt
	err := row.Scan(&r.AgentID, &r.RequestID, &r.PayloadHash, &r.MessageJSON, &r.Action, &r.Delivery, &r.Outcome, &r.AssignmentTaskID, &r.SourceAgentID, &r.OriginTaskID, &r.OriginQuestion)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}

func (s *Store) AttachmentReceipt(agentID, requestID string) (AttachmentReceipt, error) {
	return scanAttachmentReceipt(s.db.QueryRow(`SELECT `+attachmentReceiptColumns+` FROM attachment_receipts WHERE agent_id=? AND request_id=?`, agentID, requestID))
}

// AcceptedAttachmentReceipt finds the receipt a conversation accepted for
// requestID, including one since moved to its assignment's new owner, so a
// replay of the original command never delivers it again (TS-09.R69).
func (s *Store) AcceptedAttachmentReceipt(agentID, requestID string) (AttachmentReceipt, error) {
	r, err := s.AttachmentReceipt(agentID, requestID)
	if !errors.Is(err, ErrNotFound) {
		return r, err
	}
	return scanAttachmentReceipt(s.db.QueryRow(`SELECT `+attachmentReceiptColumns+` FROM attachment_receipts WHERE source_agent_id=? AND request_id=? ORDER BY rowid LIMIT 1`, agentID, requestID))
}

// MovedAttachmentReceipts lists receipts whose attachment bytes were accepted
// by another conversation, for startup to copy any the claim did not finish.
func (s *Store) MovedAttachmentReceipts() ([]AttachmentReceipt, error) {
	rows, err := s.db.Query(`SELECT agent_id,request_id,message_json,source_agent_id FROM attachment_receipts WHERE source_agent_id <> ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AttachmentReceipt{}
	for rows.Next() {
		var r AttachmentReceipt
		if err := rows.Scan(&r.AgentID, &r.RequestID, &r.MessageJSON, &r.SourceAgentID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) CreateAttachmentReceipt(r AttachmentReceipt) error {
	_, err := s.db.Exec(`INSERT INTO attachment_receipts(agent_id,request_id,payload_hash,message_json,action,delivery,updated_at) VALUES(?,?,?,?,?,'prepared',?)`, r.AgentID, r.RequestID, r.PayloadHash, r.MessageJSON, r.Action, formatTime(time.Now().UTC()))
	return err
}

func (s *Store) SetAttachmentDelivery(agentID, requestID, delivery string) error {
	_, err := s.db.Exec(`UPDATE attachment_receipts SET delivery=?,updated_at=? WHERE agent_id=? AND request_id=?`, delivery, formatTime(time.Now().UTC()), agentID, requestID)
	return err
}

func (s *Store) SetAttachmentOutcome(agentID, requestID, outcome string) error {
	_, err := s.db.Exec(`UPDATE attachment_receipts SET outcome=? WHERE agent_id=? AND request_id=?`, outcome, agentID, requestID)
	return err
}

func (s *Store) RemoveAttachmentReceipt(agentID, requestID string) error {
	_, err := s.db.Exec(`DELETE FROM attachment_receipts WHERE agent_id=? AND request_id=?`, agentID, requestID)
	return err
}

const recoveringAttachmentDeliveries = `('released','failed','unknown')`

func (s *Store) AttachmentRecovery(agentID string) (AttachmentReceipt, error) {
	return scanAttachmentReceipt(s.db.QueryRow(`SELECT `+attachmentReceiptColumns+` FROM attachment_receipts
WHERE agent_id=? AND delivery IN `+recoveringAttachmentDeliveries+` AND dismissed=0 AND recovery_attempt_id=''
ORDER BY updated_at DESC,rowid DESC LIMIT 1`, agentID))
}

const recoverySuccessorMatch = `(successor.agent_id=source.agent_id OR successor.source_agent_id=source.agent_id)
  AND successor.request_id=source.recovery_attempt_id AND successor.payload_hash=source.recovery_attempt_hash`

const acceptedRecoverySuccessor = `EXISTS (SELECT 1 FROM attachment_receipts successor WHERE ` + recoverySuccessorMatch + `)`

const settledRecoverySuccessor = `EXISTS (SELECT 1 FROM attachment_receipts successor WHERE ` + recoverySuccessorMatch + `
	AND (successor.delivery<>'prepared' OR successor.assignment_task_id<>''))`

func (s *Store) AttachmentRecoveryStatus(agentID, requestID string) (string, error) {
	var status string
	err := s.db.QueryRow(`SELECT CASE WHEN dismissed=1 OR delivery='submitted' THEN 'settled'
	WHEN recovery_attempt_id<>'' AND `+settledRecoverySuccessor+` THEN 'settled'
	WHEN recovery_attempt_id<>'' OR delivery IN ('prepared','held') THEN 'pending'
	WHEN delivery IN `+recoveringAttachmentDeliveries+` THEN 'offered' ELSE 'unavailable' END
FROM attachment_receipts AS source WHERE agent_id=? AND request_id=?`, agentID, requestID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return "unavailable", nil
	}
	return status, err
}

// ClaimAttachmentRecovery reserves the exact source before Send can admit the
// successor through the ordinary conversation and assignment paths.
func (s *Store) ClaimAttachmentRecovery(agentID, sourceID, requestID, payloadHash string) (bool, error) {
	result, err := s.db.Exec(`UPDATE attachment_receipts SET recovery_attempt_id=?,recovery_attempt_hash=?
WHERE agent_id=? AND request_id=? AND delivery IN `+recoveringAttachmentDeliveries+` AND dismissed=0 AND recovery_attempt_id=''`,
		requestID, payloadHash, agentID, sourceID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}

// FinishAttachmentRecovery settles the source if the matching successor was
// retained, or releases the claim if admission left no durable receipt.
func (s *Store) FinishAttachmentRecovery(agentID, sourceID, requestID, payloadHash string) (bool, error) {
	result, err := s.db.Exec(`UPDATE attachment_receipts AS source SET dismissed=`+acceptedRecoverySuccessor+`, recovery_attempt_id='', recovery_attempt_hash=''
WHERE agent_id=? AND request_id=? AND recovery_attempt_id=? AND recovery_attempt_hash=?`, agentID, sourceID, requestID, payloadHash)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}

// RepairAttachmentRecovery settles a retained successor whose original
// handler could not finish the claim. An unassigned prepared receipt can
// still be rolled back by an active handler, so replay leaves it alone.
func (s *Store) RepairAttachmentRecovery(agentID, sourceID, requestID, payloadHash string) (bool, error) {
	result, err := s.db.Exec(`UPDATE attachment_receipts AS source SET dismissed=1,recovery_attempt_id='',recovery_attempt_hash=''
WHERE agent_id=? AND request_id=? AND recovery_attempt_id=? AND recovery_attempt_hash=?
  AND `+settledRecoverySuccessor, agentID, sourceID, requestID, payloadHash)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}

func (s *Store) DismissAttachmentRecovery(agentID, requestID string) (bool, error) {
	result, err := s.db.Exec(`UPDATE attachment_receipts SET dismissed=1
WHERE agent_id=? AND request_id=? AND delivery IN `+recoveringAttachmentDeliveries+` AND dismissed=0 AND recovery_attempt_id=''`, agentID, requestID)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n > 0, err
}

// A held request is never a durable runnable queue. Startup releases it for
// explicit recovery; submitted/unknown receipts remain non-retryable evidence.
// Input queued for an assignment is the exception: no provider frame carried
// it yet and its assignment continuation is durable (TS-09.R69).
func (s *Store) ReleaseRestartedAttachments() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`UPDATE attachment_receipts AS source SET dismissed=` + acceptedRecoverySuccessor + `, recovery_attempt_id='', recovery_attempt_hash=''
WHERE recovery_attempt_id<>''`); err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE attachment_receipts SET delivery='released',updated_at=? WHERE delivery IN ('held','prepared') AND assignment_task_id=''`, formatTime(time.Now().UTC())); err != nil {
		return err
	}
	return tx.Commit()
}

// queuedInputTaskClause selects the assignment whose next turn is a
// continuation rather than the turn already running: waiting, a prepared
// continuation, an admission in progress, or a turn that is yielding to wait.
const queuedInputTaskClause = `(state IN ('waiting','starting') OR (state = 'ready' AND continuation_pending = 1) OR (state = 'running' AND pending_yield = 1))`

// QueueAssignmentInput retains a person's input for the agent's current
// assignment when that assignment must resume before a turn can carry it. One
// statement checks the assignment and writes the receipt, so input is never
// queued behind work that already ended. It reports false when no such
// assignment exists; the caller then delivers the input ordinarily
// (TS-09.R69, TS-10.R29, INV §5/§15).
func (s *Store) QueueAssignmentInput(r AttachmentReceipt) (string, bool, error) {
	var taskID string
	err := s.db.QueryRow(`INSERT INTO attachment_receipts(agent_id,request_id,payload_hash,message_json,action,delivery,updated_at,assignment_task_id)
SELECT ?,?,?,?,?,'prepared',?,task_id FROM tasks
WHERE assigned_agent_id = ? AND `+queuedInputTaskClause+taskRunOpenClause+`
ORDER BY updated_at DESC LIMIT 1
RETURNING assignment_task_id`, r.AgentID, r.RequestID, r.PayloadHash, r.MessageJSON, r.Action, formatTime(time.Now().UTC()), r.AgentID).Scan(&taskID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return taskID, err == nil, err
}

// TakeQueuedAssignmentInput claims the oldest input queued for the agent's
// assignment that is now being admitted. The claim turns it into an ordinary
// prepared receipt of that agent in one transaction, so the activation that
// carries it and the sweep that hands a missed one to the runtime cannot both
// deliver it. Input accepted by an earlier owner of the same assignment moves
// to this agent with SourceAgentID naming the attachment owner (TS-09.R69).
func (s *Store) TakeQueuedAssignmentInput(agentID string) (AttachmentReceipt, bool, error) {
	return s.claimQueuedInput(`a.delivery = 'prepared' AND t.assigned_agent_id = ? AND t.state IN ('starting','running')`, agentID)
}

// claimQueuedInput moves the oldest queued receipt matching the predicate on
// receipt a and its assignment t to the assignment's agent.
func (s *Store) claimQueuedInput(predicate string, args ...any) (AttachmentReceipt, bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return AttachmentReceipt{}, false, err
	}
	defer tx.Rollback()
	var r AttachmentReceipt
	var target string
	err = tx.QueryRow(`SELECT a.agent_id,a.request_id,a.payload_hash,a.message_json,a.action,a.delivery,a.outcome,a.origin_task_id,a.origin_question,t.assigned_agent_id
FROM attachment_receipts a JOIN tasks t ON t.task_id = a.assignment_task_id
WHERE `+predicate+` ORDER BY a.updated_at, a.rowid LIMIT 1`, args...).Scan(&r.AgentID, &r.RequestID, &r.PayloadHash, &r.MessageJSON, &r.Action, &r.Delivery, &r.Outcome, &r.OriginTaskID, &r.OriginQuestion, &target)
	if errors.Is(err, sql.ErrNoRows) {
		return AttachmentReceipt{}, false, nil
	}
	if err != nil {
		return AttachmentReceipt{}, false, err
	}
	source := ""
	if target != r.AgentID {
		source = r.AgentID
	}
	if _, err := tx.Exec(`UPDATE attachment_receipts SET agent_id=?, source_agent_id=?, assignment_task_id='', updated_at=? WHERE agent_id=? AND request_id=?`,
		target, source, formatTime(time.Now().UTC()), r.AgentID, r.RequestID); err != nil {
		return AttachmentReceipt{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return AttachmentReceipt{}, false, err
	}
	r.SourceAgentID = source
	r.AgentID = target
	return r, true, nil
}

// QueuedInputWork settles input whose assignment ended and readies the
// continuations that will carry the rest. Input stays with its assignment
// through admission retries and owner replacement; only input for an
// assignment that ended is released to the person's explicit recovery, never
// redirected to other work. A waiting assignment with queued input continues;
// input for an assignment that is already executing is returned for ordinary
// delivery (TS-09.R69/R70).
func (s *Store) QueuedInputWork() (prepared []Task, executing []AttachmentReceipt, err error) {
	now := formatTime(time.Now().UTC())
	if err := s.releaseEndedAssignmentInput(now); err != nil {
		return nil, nil, err
	}
	rows, err := s.db.Query(`UPDATE tasks
SET state = 'ready', continuation_pending = 1, resume_needed = 1, ready_at = ?, revision = revision + 1, updated_at = ?
WHERE state = 'waiting' AND pending_yield = 0 AND pending_release = 0
  AND EXISTS (SELECT 1 FROM attachment_receipts a WHERE a.assignment_task_id = tasks.task_id AND a.delivery = 'prepared')`+taskRunOpenClause+`
RETURNING task_id`, now, now)
	if err != nil {
		return nil, nil, fmt.Errorf("state: prepare queued input continuations: %w", err)
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, nil, err
	}
	rows.Close()
	for _, id := range ids {
		task, err := s.ReadTask(id)
		if err != nil {
			return nil, nil, err
		}
		prepared = append(prepared, task)
	}
	// The assignment's current agent receives it; AgentID here is that agent.
	rows, err = s.db.Query(`SELECT t.assigned_agent_id, a.agent_id, a.request_id FROM attachment_receipts a JOIN tasks t ON t.task_id = a.assignment_task_id
WHERE a.delivery = 'prepared' AND COALESCE(t.assigned_agent_id, '') <> '' AND t.state = 'running' AND t.pending_yield = 0 AND t.pending_release = 0
ORDER BY a.updated_at, a.rowid LIMIT 16`)
	if err != nil {
		return nil, nil, fmt.Errorf("state: list executing queued input: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var r AttachmentReceipt
		if err := rows.Scan(&r.AgentID, &r.SourceAgentID, &r.RequestID); err != nil {
			return nil, nil, err
		}
		executing = append(executing, r)
	}
	return prepared, executing, rows.Err()
}

func (s *Store) releaseEndedAssignmentInput(now string) error {
	rows, err := s.db.Query(`UPDATE attachment_receipts SET delivery = 'released', assignment_task_id = '', updated_at = ?
WHERE rowid IN (SELECT rowid FROM attachment_receipts WHERE assignment_task_id <> '' AND delivery = 'prepared'
  AND NOT EXISTS (SELECT 1 FROM tasks WHERE tasks.task_id = attachment_receipts.assignment_task_id
    AND state <> 'finished'`+taskRunOpenClause+`) ORDER BY rowid LIMIT 16)
RETURNING agent_id, request_id`, now)
	if err != nil {
		return fmt.Errorf("state: release stale queued input: %w", err)
	}
	defer rows.Close()
	var released []AttachmentReceipt
	for rows.Next() {
		var receipt AttachmentReceipt
		if err := rows.Scan(&receipt.AgentID, &receipt.RequestID); err != nil {
			return err
		}
		released = append(released, receipt)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("state: release stale queued input: %w", err)
	}
	for _, receipt := range released {
		slog.Info("state: released queued assignment input", "agent", receipt.AgentID, "request_id", receipt.RequestID, "reason", "assignment_ended")
	}
	return nil
}

// QueuedAssignmentInput lists, oldest first and bounded, the input a person
// sent this conversation that is still retained for an assignment's next turn,
// so supervision can show it after navigation or restart (TS-09.R69/R71).
func (s *Store) QueuedAssignmentInput(agentID string) ([]AttachmentReceipt, error) {
	rows, err := s.db.Query(`SELECT agent_id,request_id,payload_hash,message_json,action,delivery,outcome,assignment_task_id FROM attachment_receipts
WHERE agent_id = ? AND assignment_task_id <> '' AND delivery = 'prepared' ORDER BY updated_at, rowid LIMIT 16`, agentID)
	if err != nil {
		return nil, fmt.Errorf("state: list queued assignment input: %w", err)
	}
	defer rows.Close()
	out := []AttachmentReceipt{}
	for rows.Next() {
		var r AttachmentReceipt
		if err := rows.Scan(&r.AgentID, &r.RequestID, &r.PayloadHash, &r.MessageJSON, &r.Action, &r.Delivery, &r.Outcome, &r.AssignmentTaskID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// BindAssignmentInput records a person's input, prepared, for the assignment
// whose turn will carry it (TS-09.R69).
func (s *Store) BindAssignmentInput(r AttachmentReceipt, taskID string) error {
	return bindAssignmentInput(s.db, r, taskID)
}

func bindAssignmentInput(db interface {
	Exec(string, ...any) (sql.Result, error)
}, r AttachmentReceipt, taskID string) error {
	_, err := db.Exec(`INSERT INTO attachment_receipts(agent_id,request_id,payload_hash,message_json,action,delivery,updated_at,assignment_task_id,origin_task_id,origin_question) VALUES(?,?,?,?,?,'prepared',?,?,?,?)`,
		r.AgentID, r.RequestID, r.PayloadHash, r.MessageJSON, r.Action, formatTime(time.Now().UTC()), taskID, r.OriginTaskID, r.OriginQuestion)
	return err
}

// ClaimQueuedInputForTurn turns one queued receipt into an ordinary prepared
// receipt of the assignment's current agent for runtime delivery, only while
// that assignment is still executing.
func (s *Store) ClaimQueuedInputForTurn(sourceAgentID, requestID string) (AttachmentReceipt, bool, error) {
	return s.claimQueuedInput(`a.agent_id = ? AND a.request_id = ? AND a.delivery = 'prepared' AND t.state = 'running' AND t.pending_yield = 0 AND t.pending_release = 0 AND COALESCE(t.assigned_agent_id, '') <> ''`, sourceAgentID, requestID)
}
