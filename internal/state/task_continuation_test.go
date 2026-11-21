package state

import "testing"

// TS-10.R65: another provider turn keeps this execution's handle, but a late
// preparer cannot revive a yielded, cancelled or replaced execution.
func TestTaskContinuationBindsOnlyCurrentRunningExecution(t *testing.T) {
	st, _ := newTestStore(t)
	task := newTask(t, st, "proj", "continuation")
	if _, err := st.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = 'a_owner', assigned_generation = 'g_live', execution_handle = 'h_live', execution_turn = 'old_turn' WHERE task_id = ?`, TaskRunning, task.TaskID); err != nil {
		t.Fatal(err)
	}
	bound, ok, err := st.ConfirmTaskContinuation(task.TaskID, "h_live", "g_live", "next_turn")
	if err != nil || !ok || bound.ExecutionHandle != "h_live" || bound.ExecutionTurn != "next_turn" || bound.State != TaskRunning {
		t.Fatalf("continuation = %+v, %v, %v", bound, ok, err)
	}
	for _, stale := range []struct{ handle, generation string }{{"h_old", "g_live"}, {"h_live", "g_old"}} {
		if _, ok, err := st.ConfirmTaskContinuation(task.TaskID, stale.handle, stale.generation, "stale_turn"); err != nil || ok {
			t.Fatalf("stale execution admitted: %+v, %v, %v", stale, ok, err)
		}
	}
	if _, err := st.DB().Exec(`UPDATE tasks SET pending_yield = 1 WHERE task_id = ?`, task.TaskID); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := st.ConfirmTaskContinuation(task.TaskID, "h_live", "g_live", "late_turn"); err != nil || ok {
		t.Fatalf("yielded execution admitted: %v, %v", ok, err)
	}
	if _, err := st.DB().Exec(`UPDATE tasks SET pending_yield = 0, state = ?, pending_release = 1 WHERE task_id = ?`, TaskFinished, task.TaskID); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := st.ConfirmTaskContinuation(task.TaskID, "h_live", "g_live", "late_turn"); err != nil || ok {
		t.Fatalf("cancelled execution admitted: %v, %v", ok, err)
	}
	after, err := st.ReadTask(task.TaskID)
	if err != nil || after.ExecutionTurn != "next_turn" || after.ExecutionHandle != "h_live" {
		t.Fatalf("late preparer changed execution: %+v, %v", after, err)
	}
}
