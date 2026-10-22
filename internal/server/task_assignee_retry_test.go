package server

import (
	"context"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// FS-16.R23/R34: contention/failure while resuming a confirmed launch-spec
// assignment must not fork its conversation on the next actual native start.
func TestDeferredAssignmentResumesOriginalAgent(t *testing.T) {
	for _, continuation := range []bool{false, true} {
		for _, failure := range []bool{false, true} {
			name := "retry"
			if continuation {
				name = "continuation"
			}
			if failure {
				name += "_failure"
			} else {
				name += "_contention"
			}
			t.Run(name, func(t *testing.T) {
				srv, _ := wakeTestServer(t)
				task := newLaunchTask(t, srv, "retain assignee")
				srv.dispatchReadyTasks(context.Background())
				running := waitTaskState(t, srv, task.TaskID, state.TaskRunning)
				id := running.AssignedAgentID
				if err := srv.StopStage(context.Background(), id); err != nil {
					t.Fatal(err)
				}
				if continuation {
					// Seed the durable settled-yield boundary; the actual next start
					// must use the stored assignee, regardless of target launch kind.
					if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state=?, runtime_claim='', assigned_generation='', continuation_pending=1, resume_needed=1, pending_yield=0 WHERE task_id=?`, state.TaskReady, task.TaskID); err != nil {
						t.Fatal(err)
					}
				} else {
					if _, _, err := srv.stateStore.InterruptTask(task.TaskID, "stopped"); err != nil {
						t.Fatal(err)
					}
					if _, err := srv.stateStore.RetryTask(task.TaskID); err != nil {
						t.Fatal(err)
					}
				}
				ready, err := srv.stateStore.ReadTask(task.TaskID)
				if err != nil {
					t.Fatal(err)
				}
				plan, ok := srv.planTaskStart(ready)
				if !ok {
					t.Fatal("retry was not plannable")
				}
				if _, ok, err := srv.stateStore.AdmitReadyTask(task.TaskID, plan, 10); err != nil || !ok {
					t.Fatalf("admit retry: %v %v", ok, err)
				}
				if failure {
					_, _, err = srv.stateStore.FailTaskStart(task.TaskID, plan.AttemptID, "resume failed")
				} else {
					_, _, err = srv.stateStore.AbandonTaskStart(task.TaskID, plan.AttemptID)
				}
				if err != nil {
					t.Fatal(err)
				}
				srv.dispatchReadyTasks(context.Background())
				resumed := waitTaskState(t, srv, task.TaskID, state.TaskRunning)
				if resumed.AssignedAgentID != id || !srv.registry.Owns(id) || resumed.AssignedGeneration == running.AssignedGeneration {
					t.Fatalf("retry did not resume original conversation: before=%+v after=%+v", running, resumed)
				}
			})
		}
	}
}
