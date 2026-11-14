package state

import (
	"database/sql"
	"errors"
)

// TurnAdmission identifies which admission check the caller makes at its
// existing claim boundary. A waiting mail assignment can request a task wake.
type TurnAdmission uint8

const (
	AdmitMailAssignment TurnAdmission = iota
	AdmitWithQuotaGate
	AdmitRoomAssignment
	AdmitTaskRoom
	AdmitQuotaOwner
	AdmitQuotaUnassigned
	AdmitTaskRestart
	AdmitRoomRestart
	AdmitRuntimeTurn
)

type TurnAdmissionRequest struct {
	AgentID     string
	Generation  string
	Check       TurnAdmission
	Quota       QuotaIntent
	Budget      int
	Task        Task
	RoomAttempt ThinkTankAttempt
}

type TurnAdmissionDecision struct {
	Allowed  bool
	WakeTask bool
	Quota    QuotaIntent
}

type admissionReader interface {
	QueryRow(string, ...any) *sql.Row
}

const noRunningRoomAttempt = `NOT EXISTS (SELECT 1 FROM think_tank_attempts WHERE agent_id = ? AND state = 'running')`

func (s *Store) DecideTurnAdmission(req TurnAdmissionRequest) (TurnAdmissionDecision, error) {
	if req.Check == AdmitQuotaOwner {
		tx, err := s.db.Begin()
		if err != nil {
			return TurnAdmissionDecision{}, err
		}
		defer tx.Rollback()
		return decideTurnAdmission(tx, req)
	}
	return decideTurnAdmission(s.db, req)
}

func decideTurnAdmission(db admissionReader, req TurnAdmissionRequest) (TurnAdmissionDecision, error) {
	var decision TurnAdmissionDecision
	switch req.Check {
	case AdmitWithQuotaGate:
		q, err := scanQuota(db.QueryRow(quotaSelect+` WHERE agent_id=?`, req.AgentID))
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, ErrNotFound) {
			decision.Allowed = true
			return decision, nil
		}
		decision.Quota = q
		decision.Allowed = err == nil && (q.State == QuotaContinued || q.State == QuotaCancelled)
		return decision, err
	case AdmitMailAssignment:
		var taskState string
		var pendingYield, pendingRelease bool
		err := db.QueryRow(`SELECT state, pending_yield, pending_release FROM tasks WHERE assigned_agent_id=? AND `+
			taskCommitmentClause, req.AgentID).Scan(&taskState, &pendingYield, &pendingRelease)
		if errors.Is(err, sql.ErrNoRows) {
			decision.Allowed = true
			return decision, nil
		}
		decision.Allowed = err == nil && !pendingYield && !pendingRelease && taskState == TaskRunning
		decision.WakeTask = err == nil && !pendingYield && !pendingRelease && taskState == TaskWaiting
		return decision, err
	case AdmitRoomAssignment:
		err := db.QueryRow(`SELECT NOT EXISTS(SELECT 1 FROM tasks WHERE assigned_agent_id=? AND `+
			activeTaskAssignmentClause+`)`, req.AgentID).Scan(&decision.Allowed)
		return decision, err
	case AdmitTaskRoom:
		err := db.QueryRow(`SELECT `+noRunningRoomAttempt, req.AgentID).Scan(&decision.Allowed)
		return decision, err
	case AdmitQuotaOwner:
		tx, ok := db.(*sql.Tx)
		if !ok {
			return decision, ErrQuotaConflict
		}
		allowed, err := quotaWorkEligibleTx(tx, req.Quota, req.Budget)
		decision.Allowed = allowed
		return decision, err
	case AdmitQuotaUnassigned:
		for _, check := range []TurnAdmission{AdmitRoomAssignment, AdmitTaskRoom} {
			other, err := decideTurnAdmission(db, TurnAdmissionRequest{AgentID: req.AgentID, Check: check})
			if err != nil {
				return decision, err
			}
			if !other.Allowed {
				return decision, ErrQuotaOwner
			}
		}
		decision.Allowed = true
		return decision, nil
	case AdmitTaskRestart:
		task := req.Task
		var held int
		err := db.QueryRow(`SELECT 1 FROM quota_intents WHERE agent_id=?
AND (source_generation=? OR continued_generation=? OR manual_generation=?)
AND state != ? AND json_extract(owner_json,'$.task_id')=?`,
			task.AssignedAgentID, task.AssignedGeneration, task.AssignedGeneration,
			task.AssignedGeneration, QuotaContinued, task.TaskID).Scan(&held)
		if errors.Is(err, sql.ErrNoRows) {
			decision.Allowed = true
			return decision, nil
		}
		return decision, err
	case AdmitRoomRestart:
		held, err := quotaContinuationRunning(db, req.RoomAttempt)
		decision.Allowed = err == nil && !held
		return decision, err
	case AdmitRuntimeTurn:
		turn, err := readConversationTurn(db, req.AgentID, req.Generation)
		if errors.Is(err, ErrNotFound) {
			err = nil
		}
		decision.Allowed = err == nil && turn.State != "running"
		return decision, err
	default:
		return decision, errors.New("state: unknown turn admission check")
	}
}
