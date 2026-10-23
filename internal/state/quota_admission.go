package state

import (
	"database/sql"
	"errors"
)

// QuotaWorkEligible is a read-only preflight shared by checks and actual
// continuation. Admission revalidates its transactional predicates before effects.
func (s *Store) QuotaWorkEligible(q QuotaIntent, budget int) (bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	return quotaWorkEligibleTx(tx, q, budget)
}

func quotaWorkEligibleTx(tx *sql.Tx, q QuotaIntent, budget int) (bool, error) {
	var exists int
	if err := tx.QueryRow(`SELECT 1 FROM agents WHERE agent_id=? AND archived=0 AND interface='chat'`, q.AgentID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, ErrQuotaOwner
		}
		return false, err
	}
	switch q.Owner.Kind {
	case "ordinary":
		return quotaUnassignedTx(tx, q.AgentID)
	case "task":
		claim, err := quotaTaskClaimTx(tx, q)
		if err != nil {
			return false, err
		}
		if claim == ClaimCreated || claim == ClaimWoke {
			return true, nil
		}
		if err := tx.QueryRow(`SELECT COUNT(*) FROM running WHERE agent_id=?`, q.AgentID).Scan(&exists); err != nil {
			return false, err
		}
		if exists != 0 {
			return true, nil
		}
		if err := tx.QueryRow(`SELECT `+taskCapacityFreeClause, ClaimCreated, ClaimWoke, budget).Scan(&exists); err != nil {
			return false, err
		}
		return exists != 0, nil
	case "room":
		if err := requirePipelineRoomOpenTx(tx, q.Owner.RoomID); err != nil {
			if errors.Is(err, ErrThinkTankConflict) || errors.Is(err, ErrNotFound) {
				return false, ErrQuotaOwner
			}
			return false, err
		}
		d, err := readThinkTankDetail(tx, q.Owner.RoomID)
		if err != nil {
			return false, err
		}
		_, err = validateQuotaThinkTankContinuationTx(tx, ThinkTankBegin{RoomID: q.Owner.RoomID, Revision: d.Room.Revision,
			AgentID: q.AgentID, Turn: q.Owner.Turn, QuotaContinuation: &ThinkTankQuotaContinuation{
				EpisodeID: q.EpisodeID, Revision: q.Revision, AttemptToken: q.AttemptToken, SourceAttemptID: q.Owner.AttemptID}}, d)
		if err != nil {
			if errors.Is(err, ErrThinkTankConflict) {
				return false, ErrQuotaOwner
			}
			return false, err
		}
		return quotaUnassignedTx(tx, q.AgentID)
	default:
		return false, ErrQuotaOwner
	}
}

func quotaUnassignedTx(tx *sql.Tx, agentID string) (bool, error) {
	var assigned, rooms int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM tasks WHERE assigned_agent_id=? AND state IN (?,?,?)`, agentID, TaskStarting, TaskRunning, TaskWaiting).Scan(&assigned); err != nil {
		return false, err
	}
	if err := tx.QueryRow(`SELECT COUNT(*) FROM think_tank_attempts WHERE agent_id=? AND state=?`, agentID, ThinkTankAttemptRunning).Scan(&rooms); err != nil {
		return false, err
	}
	if assigned != 0 || rooms != 0 {
		return false, ErrQuotaOwner
	}
	return true, nil
}

func quotaTaskClaimTx(tx *sql.Tx, q QuotaIntent) (string, error) {
	var taskState, agentID, generation, claim string
	var pendingRelease, pendingYield int
	err := tx.QueryRow(`SELECT state, assigned_agent_id, assigned_generation, runtime_claim, pending_release, pending_yield
FROM tasks WHERE task_id=?`+taskRunOpenClause, q.Owner.TaskID).Scan(&taskState, &agentID, &generation, &claim, &pendingRelease, &pendingYield)
	if errors.Is(err, sql.ErrNoRows) || err == nil && (taskState != TaskRunning || agentID != q.AgentID || generation != q.SourceGeneration || pendingRelease != 0 || pendingYield != 0 || claim != ClaimBorrowed && claim != ClaimCreated && claim != ClaimWoke) {
		return "", ErrQuotaOwner
	}
	return claim, err
}

// AdmitQuotaTurn revalidates the domain's current exclusivity and closure under
// the same transaction that records the provider boundary and executing tuple.
func (s *Store) AdmitQuotaTurn(q QuotaIntent, generation, turn string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRow(`SELECT 1 FROM agents WHERE agent_id=? AND archived=0 AND interface='chat'`, q.AgentID).Scan(&exists); err != nil {
		return ErrQuotaOwner
	}
	switch q.Owner.Kind {
	case "ordinary":
		if _, err := quotaUnassignedTx(tx, q.AgentID); err != nil {
			return err
		}
	case "task":
		var assigned, source string
		err := tx.QueryRow(`SELECT assigned_agent_id, assigned_generation FROM tasks WHERE task_id=? AND state=? AND pending_release=0 AND pending_yield=0`+taskRunOpenClause, q.Owner.TaskID, TaskRunning).Scan(&assigned, &source)
		if errors.Is(err, sql.ErrNoRows) || err == nil && (assigned != q.AgentID || source != q.SourceGeneration) {
			return ErrQuotaOwner
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE tasks SET assigned_generation=?, execution_turn=?, revision=revision+1, updated_at=? WHERE task_id=?`, generation, turn, formatTime(timeNow()), q.Owner.TaskID); err != nil {
			return err
		}
	default:
		return ErrQuotaOwner // rooms use their guarded linked-attempt admission
	}
	if err := markQuotaProviderStartedTx(tx, q.AgentID, q.EpisodeID, q.Revision, q.AttemptToken, generation, turn); err != nil {
		return err
	}
	if err := resetTurnBudgetTx(tx, q.AgentID, turn); err != nil {
		return err
	}
	return tx.Commit()
}

// AdmitQuotaTaskResume converts a borrowed task assignment into a waking
// reservation when quota continuation must resume its stopped provider. A
// borrowed task normally owns no runtime slot; only a committed conversion
// makes this resume consume capacity. Existing created/woke reservations stay
// intact and already hold their slot.
func (s *Store) AdmitQuotaTaskResume(q QuotaIntent, budget int) (bool, error) {
	if q.Owner.Kind != "task" || budget < 1 {
		return false, ErrQuotaConflict
	}
	tx, err := s.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	fresh, err := scanQuota(tx.QueryRow(quotaSelect+` WHERE agent_id=?`, q.AgentID))
	if err != nil {
		return false, err
	}
	if fresh.EpisodeID != q.EpisodeID || fresh.Revision != q.Revision || fresh.AttemptToken != q.AttemptToken ||
		fresh.State != QuotaClaimed || fresh.ProviderStarted || fresh.Owner != q.Owner || fresh.SourceGeneration != q.SourceGeneration {
		return false, nil
	}
	claim, err := quotaTaskClaimTx(tx, q)
	if err != nil {
		return false, err
	}
	if claim == ClaimCreated || claim == ClaimWoke {
		return true, tx.Commit()
	}
	if claim != ClaimBorrowed {
		return false, ErrQuotaOwner
	}
	res, err := tx.Exec(`UPDATE tasks SET runtime_claim=?, revision=revision+1, updated_at=?
WHERE task_id=? AND state=? AND assigned_agent_id=? AND assigned_generation=? AND runtime_claim=?
AND pending_release=0 AND pending_yield=0
AND `+taskCapacityFreeClause+taskRunOpenClause,
		ClaimWoke, formatTime(timeNow()), q.Owner.TaskID, TaskRunning, q.AgentID, q.SourceGeneration, ClaimBorrowed,
		ClaimCreated, ClaimWoke, budget)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return n == 1, nil
}
