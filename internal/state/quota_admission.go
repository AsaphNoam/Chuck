package state

import (
	"database/sql"
	"errors"
)

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
		if err := tx.QueryRow(`SELECT COUNT(*) FROM tasks WHERE assigned_agent_id=? AND state IN (?,?,?)`, q.AgentID, TaskStarting, TaskRunning, TaskWaiting).Scan(&exists); err != nil {
			return err
		}
		if exists != 0 {
			return ErrQuotaOwner
		}
		if err := tx.QueryRow(`SELECT COUNT(*) FROM think_tank_attempts WHERE agent_id=? AND state=?`, q.AgentID, ThinkTankAttemptRunning).Scan(&exists); err != nil {
			return err
		}
		if exists != 0 {
			return ErrQuotaOwner
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
