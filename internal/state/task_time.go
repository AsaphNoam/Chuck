package state

import (
	"fmt"
	"time"
)

// OnTimeWindow is how late Chuck may first observe a start time and still call it
// on time (FS-16.R52, TS-10.R55). Only a stopped process or a sleeping Mac can
// exceed it; the two-second dispatcher tick never does.
const OnTimeWindow = 5 * time.Minute

// MissedStartTimeReason is the attention reason of a task parked because its
// start time passed unobserved with Run only on time on.
const MissedStartTimeReason = "missed start time"

// ResolveDueTimeArms settles at most limit time arms whose instant has passed,
// oldest first, and advances their tasks in the same transaction (TS-10.R54–R55).
// An arm observed within the on-time window is satisfied. A later observation is
// satisfied with the task marked late when Run only on time is off, and parks the
// task as a missed start otherwise. It returns the tasks whose state changed.
func (s *Store) ResolveDueTimeArms(limit int) ([]Task, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("state: begin due time arms: %w", err)
	}
	defer tx.Rollback()

	now := timeNow()
	rows, err := tx.Query(`
SELECT arm_id, task_id, due_at, on_time_only FROM task_arms
WHERE kind = ? AND state = ? AND due_at <= ?
ORDER BY due_at, arm_id LIMIT ?`, ArmTime, ArmUnsatisfied, formatTime(now), limit)
	if err != nil {
		return nil, fmt.Errorf("state: list due time arms: %w", err)
	}
	type dueArm struct {
		armID, taskID string
		dueAt         time.Time
		onTimeOnly    bool
	}
	var due []dueArm
	for rows.Next() {
		var arm dueArm
		var raw string
		if err := rows.Scan(&arm.armID, &arm.taskID, &raw, &arm.onTimeOnly); err != nil {
			rows.Close()
			return nil, fmt.Errorf("state: scan due time arm: %w", err)
		}
		if arm.dueAt, err = parseTime(raw); err != nil {
			rows.Close()
			return nil, wrapTimeErr("task arm due_at", err)
		}
		due = append(due, arm)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, fmt.Errorf("state: iterate due time arms: %w", err)
	}
	if len(due) == 0 {
		return []Task{}, nil
	}

	stamp := formatTime(now)
	var taskIDs, missed []string
	for _, arm := range due {
		taskIDs = append(taskIDs, arm.taskID)
		late := now.Sub(arm.dueAt) > OnTimeWindow
		if late && arm.onTimeOnly {
			missed = append(missed, arm.taskID)
			if _, err := tx.Exec(`UPDATE task_arms SET state = ? WHERE arm_id = ? AND state = ?`,
				ArmUnsatisfiable, arm.armID, ArmUnsatisfied); err != nil {
				return nil, fmt.Errorf("state: miss time arm: %w", err)
			}
			continue
		}
		if _, err := tx.Exec(`UPDATE task_arms SET state = ?, satisfied_at = ? WHERE arm_id = ? AND state = ?`,
			ArmSatisfied, stamp, arm.armID, ArmUnsatisfied); err != nil {
			return nil, fmt.Errorf("state: satisfy time arm: %w", err)
		}
		if late {
			if _, err := tx.Exec(`UPDATE tasks SET late_start = 1 WHERE task_id = ? AND state = ?`,
				arm.taskID, TaskArmed); err != nil {
				return nil, fmt.Errorf("state: mark late start: %w", err)
			}
		}
	}
	changed, err := advanceArmedTasks(tx, taskIDs, now)
	if err != nil {
		return nil, err
	}
	// The shared park path words the reason generically; a missed start says
	// what actually happened, in the same transaction that parked it.
	for _, id := range missed {
		if _, err := tx.Exec(`UPDATE tasks SET attention_reason = ? WHERE task_id = ? AND state = ?`,
			MissedStartTimeReason, id, TaskDependencyFailed); err != nil {
			return nil, fmt.Errorf("state: record missed start: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("state: commit due time arms: %w", err)
	}
	return s.readTasks(changed)
}

func (s *Store) readTasks(ids []string) ([]Task, error) {
	tasks := []Task{}
	for _, id := range ids {
		task, err := s.ReadTask(id)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}
