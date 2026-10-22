package state

import (
	"fmt"
	"testing"
)

// TS-10.R49: a lost hint must be rediscoverable in bounded batches. Stale
// watches and cancelled assignments must not wake or pin the recovery batch.
func TestWaitReconciliationDrainsCurrentActionableWatches(t *testing.T) {
	st, _ := newTestStore(t)
	source := newTask(t, st, "proj", "completed child")
	if _, err := st.DB().Exec(`UPDATE tasks SET state=? WHERE task_id=?`, TaskFinished, source.TaskID); err != nil {
		t.Fatal(err)
	}
	var owners []string
	for i := 0; i < 11; i++ {
		owner := newTask(t, st, "proj", fmt.Sprintf("owner-%d", i))
		state := TaskWaiting
		version := 1
		if i == 0 {
			state = TaskFinished
		}
		if i == 1 {
			version = 2
		}
		if _, err := st.DB().Exec(`UPDATE tasks SET state=?, assigned_agent_id=?, wait_version=? WHERE task_id=?`, state, fmt.Sprintf("a_owner_%d", i), version, owner.TaskID); err != nil {
			t.Fatal(err)
		}
		if _, err := st.DB().Exec(`INSERT INTO task_waits(waiting_task_id,wait_version,source_task_id,after_revision) VALUES(?,1,?,0)`, owner.TaskID, source.TaskID); err != nil {
			t.Fatal(err)
		}
		owners = append(owners, owner.TaskID)
	}
	for _, want := range []int{8, 1, 0} {
		changed, err := st.ReconcileTaskWaits(8)
		if err != nil || len(changed) != want {
			t.Fatalf("batch = %d %v, want %d", len(changed), err, want)
		}
	}
	for i, id := range owners {
		owner, err := st.ReadTask(id)
		if err != nil {
			t.Fatal(err)
		}
		if i < 2 {
			if owner.ResumeNeeded {
				t.Fatalf("closed or stale watch woke: %+v", owner)
			}
		} else if owner.State != TaskReady || !owner.ContinuationPending {
			t.Fatalf("watch stranded: %+v", owner)
		}
	}
}
