package state

import "fmt"

// ConfirmTaskContinuation binds another turn of the same running execution.
// Its handle stays valid, while cancellation targets the newly admitted turn.
// A report, yield, cancellation or closed pipeline wins by making this guarded
// update refuse before the provider receives an assignment (TS-10.R65).
func (s *Store) ConfirmTaskContinuation(taskID, handle, generation, turnID string) (Task, bool, error) {
	if handle == "" || generation == "" || turnID == "" {
		return Task{}, false, nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Task{}, false, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE tasks SET execution_turn = ?, revision = revision + 1, updated_at = ?
WHERE task_id = ? AND state = ? AND execution_handle = ? AND assigned_generation = ?
  AND pending_release = 0 AND pending_yield = 0`+taskRunOpenClause,
		turnID, formatTime(timeNow()), taskID, TaskRunning, handle, generation)
	if err != nil {
		return Task{}, false, fmt.Errorf("state: confirm task continuation: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil || n == 0 {
		return Task{}, false, err
	}
	var stageCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM pipeline_stage_tasks WHERE task_id = ?`, taskID).Scan(&stageCount); err != nil {
		return Task{}, false, err
	}
	if stageCount != 0 {
		var agentID string
		if err := tx.QueryRow(`SELECT assigned_agent_id FROM tasks WHERE task_id = ?`, taskID).Scan(&agentID); err != nil {
			return Task{}, false, err
		}
		if err := bindPipelineStageTaskStandingAgentTx(tx, taskID, agentID, &generation); err != nil {
			return Task{}, false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Task{}, false, err
	}
	task, err := s.ReadTask(taskID)
	return task, err == nil, err
}
