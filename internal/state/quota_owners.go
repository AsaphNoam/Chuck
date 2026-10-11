package state

import (
	"database/sql"
	"errors"
	"fmt"
)

// Historical quota receipts cannot hold a replaced member or another phase.
// The q alias is shared by all room-owner queries below.
const quotaRoomOwnerCurrentClause = `
AND EXISTS (SELECT 1 FROM think_tanks qr WHERE qr.room_id=json_extract(q.owner_json,'$.room_id')
AND ((json_extract(q.owner_json,'$.turn')='judge' AND qr.phase='ended' AND qr.judge_agent_id=q.agent_id)
OR (EXISTS (SELECT 1 FROM think_tank_members qm WHERE qm.room_id=qr.room_id AND qm.agent_id=q.agent_id AND qm.role='participant' AND qm.state<>'departed')
AND ((json_extract(q.owner_json,'$.turn')='opening' AND qr.phase='openings')
OR (json_extract(q.owner_json,'$.turn')='discussion' AND qr.phase='discussion')
OR (json_extract(q.owner_json,'$.turn')='closing' AND qr.phase='closing')))))`

// ObserveOwnedQuota resolves ownership from the actual executing tuple and
// commits its recovery marker before ordinary terminal settlement can run.
// Caller-supplied ownership is never authority.
func (s *Store) ObserveOwnedQuota(in QuotaObservation) (QuotaIntent, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return QuotaIntent{}, err
	}
	defer tx.Rollback()
	in.Owner = QuotaOwner{Kind: "ordinary"}
	a, err := scanThinkTankAttempt(tx.QueryRow(`SELECT `+thinkTankAttemptColumns+` FROM think_tank_attempts
WHERE agent_id=? AND generation=? AND turn_id=? AND state=?`, in.AgentID, in.SourceGeneration, in.ExecutingTurn, ThinkTankAttemptRunning))
	if err == nil {
		in.Owner = QuotaOwner{Kind: "room", RoomID: a.RoomID, AttemptID: a.AttemptID, Turn: a.Turn}
	} else if !errors.Is(err, ErrNotFound) && !errors.Is(err, sql.ErrNoRows) {
		return QuotaIntent{}, err
	} else {
		var taskID string
		err = tx.QueryRow(`SELECT task_id FROM tasks WHERE assigned_agent_id=? AND assigned_generation=?
AND state IN (?,?,?) AND pending_release=0 AND pending_yield=0`, in.AgentID, in.SourceGeneration, TaskStarting, TaskRunning, TaskWaiting).Scan(&taskID)
		if err == nil {
			in.Owner = QuotaOwner{Kind: "task", TaskID: taskID}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return QuotaIntent{}, err
		}
	}
	q, err := observeQuotaTx(tx, in)
	if err != nil {
		return QuotaIntent{}, err
	}
	if in.Owner.Kind == "room" {
		// The old token and staged disposition remain immutable history. This
		// turn charges no allowance and publishes nothing.
		res, err := tx.Exec(`UPDATE think_tank_attempts SET state=?, failure='Quota reached', finished_at=? WHERE attempt_id=? AND room_id=? AND agent_id=? AND generation=? AND turn_id=? AND turn=? AND state=?`,
			ThinkTankAttemptFailed, formatTime(in.ObservedAt), a.AttemptID, a.RoomID, in.AgentID, in.SourceGeneration, in.ExecutingTurn, a.Turn, ThinkTankAttemptRunning)
		if err != nil {
			return QuotaIntent{}, err
		}
		if n, err := res.RowsAffected(); err != nil || n != 1 {
			if err != nil {
				return QuotaIntent{}, err
			}
			return QuotaIntent{}, ErrQuotaConflict
		}
		if _, err := tx.Exec(`UPDATE think_tanks SET hold='Quota reached', revision=revision+1 WHERE room_id=?`, a.RoomID); err != nil {
			return QuotaIntent{}, err
		}
		if a.Turn == ThinkTankTurnJudge {
			if _, err := tx.Exec(`UPDATE think_tanks SET judge_status=?, judge_error='Quota reached' WHERE room_id=? AND judge_status=?`,
				ThinkTankJudgeStarting, a.RoomID, ThinkTankJudgeRunning); err != nil {
				return QuotaIntent{}, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return QuotaIntent{}, fmt.Errorf("state: commit owned quota: %w", err)
	}
	return q, nil
}

// QuotaHoldsTask distinguishes a preserved quota assignment from generic crash
// recovery. Completed/cancelled task cleanup remains independently authoritative.
func (s *Store) QuotaHoldsTask(task Task) (bool, error) {
	var found int
	err := s.db.QueryRow(`SELECT 1 FROM quota_intents WHERE agent_id=? AND (source_generation=? OR continued_generation=? OR manual_generation=?)
AND state != ? AND json_extract(owner_json,'$.task_id')=?`,
		task.AssignedAgentID, task.AssignedGeneration, task.AssignedGeneration, task.AssignedGeneration, QuotaContinued, task.TaskID).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}
