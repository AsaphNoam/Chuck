package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// A wait discovered at the final wake guard remains a retryable conflict and
// cannot start the stopped runtime, even after the earlier candidacy read.
func TestWakeGuardPreservesAdmissionConflict(t *testing.T) {
	srv, ts := wakeTestServer(t)
	id := launchThenStop(t, srv, ts)
	ae := srv.resumeSessionWithHooks(context.Background(), id, resumeOverride{}, func() error {
		return apiError(runtime.CodeConflict, "task continuation requires admission")
	}, nil)
	if ae == nil || ae.Code != runtime.CodeConflict || srv.registry.Owns(id) {
		t.Fatalf("wake crossed admission guard or lost conflict: owned=%v error=%+v", srv.registry.Owns(id), ae)
	}
}

// FS-16.R33/R34: exercise the actual producers and startup rediscovery rather
// than invoking NotifyTaskWaiters, which cannot prove the wiring exists.
func TestActionableChangesWakeWaitingAssignment(t *testing.T) {
	for _, change := range []string{"exit", "delete", "lost_result", "park"} {
		t.Run(change, func(t *testing.T) {
			srv := testServer(t, true)
			owner := newLaunchTask(t, srv, "owner")
			if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state=?, assigned_agent_id=?, assigned_generation='g-owner', runtime_claim=?, execution_handle='exec-owner' WHERE task_id=?`, state.TaskRunning, "a_owner", state.ClaimBorrowed, owner.TaskID); err != nil {
				t.Fatal(err)
			}
			id, err := srv.stateStore.NewTaskID()
			if err != nil {
				t.Fatal(err)
			}
			child, err := srv.stateStore.CreateTask(state.Task{TaskID: id, Project: "tmpproj", DisplayName: "child", Instruction: "work", TargetKind: state.TargetLaunch, Role: "impl", CreatedByKind: "agent", CreatedByAgentID: "a_owner"})
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := srv.stateStore.WaitForTasks("a_owner", "g-owner", "exec-owner", []state.TaskWaitObservation{{TaskID: id, AfterRevision: child.Revision}}); err != nil {
				t.Fatal(err)
			}
			if _, err := srv.stateStore.CompleteTaskYield(owner.TaskID); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "exit":
				if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state=?, assigned_agent_id='a_child', assigned_generation='g-child', runtime_claim=? WHERE task_id=?`, state.TaskRunning, state.ClaimBorrowed, id); err != nil {
					t.Fatal(err)
				}
				srv.interruptTaskOnExit("a_child", "g-child", "test exit")
			case "delete":
				req := httptest.NewRequest(http.MethodDelete, "/api/tasks/"+id, nil)
				req.SetPathValue("id", id)
				res := httptest.NewRecorder()
				srv.handleDeleteTask(res, req)
				if res.Code != http.StatusNoContent {
					t.Fatalf("delete: %d %s", res.Code, res.Body.String())
				}
			case "lost_result":
				if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state=? WHERE task_id=?`, state.TaskRunning, id); err != nil {
					t.Fatal(err)
				}
				if _, err := srv.stateStore.RecordPersonTaskResult(id, state.TaskResult{Outcome: state.OutcomeSuccess, Summary: "committed before crash"}); err != nil {
					t.Fatal(err)
				}
				if err := srv.recoverTasks(context.Background()); err != nil {
					t.Fatal(err)
				}
			case "park":
				admitted, ok, err := srv.stateStore.AdmitReadyTask(id, state.TaskReservation{AgentID: "a_child", Generation: "g-child", AttemptID: "attempt-child", Claim: state.ClaimCreated}, 10)
				if err != nil || !ok {
					t.Fatalf("admit child: %v %v", ok, err)
				}
				parked, ok, err := srv.stateStore.ParkTaskStart(id, admitted.StartAttemptID, "invalid_configuration")
				if err != nil || !ok {
					t.Fatalf("park child: %v %v", ok, err)
				}
				srv.propagateTaskFailure(parked)
			}
			ready, err := srv.stateStore.ReadTask(owner.TaskID)
			if err != nil || ready.State != state.TaskReady || !ready.ResumeNeeded || !ready.ContinuationPending || ready.AssignedAgentID != "a_owner" || ready.WaitVersion != 1 {
				t.Fatalf("actionable %s stranded assignment: %+v %v", change, ready, err)
			}
		})
	}
}
