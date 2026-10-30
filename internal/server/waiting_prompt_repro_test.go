package server

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// WAIT-PROMPT-1 / TS-10.R29: a person's prompt to an assignee whose task is
// waiting on child work must reach that agent through the assignment
// continuation. Today the route prepares the continuation, rejects the prompt
// with 409 "retry shortly" and keeps no copy; the continuation turn re-waits
// within seconds, so the instruction is never delivered.
func TestPromptToWaitingAssigneeIsDeliveredByContinuation(t *testing.T) {
	t.Skip("WAIT-PROMPT-1: prompt to a waiting assignee is rejected and dropped; un-skip in the fix session")
	srv, ts, promptLog := activationTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	generation := srv.taskGeneration(id)
	taskID, err := srv.stateStore.NewTaskID()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := srv.stateStore.CreateTask(state.Task{TaskID: taskID, Project: "tmpproj", DisplayName: "Work", Instruction: "coordinate", TargetKind: state.TargetAgent, TargetAgentID: id, CreatedByKind: "pipeline"}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = ?, assigned_generation = ?, started_at=created_at WHERE task_id = ?`, state.TaskWaiting, id, generation, taskID); err != nil {
		t.Fatal(err)
	}
	const instruction = "use opus as the fixing agent"
	if resp, body := post(t, ts.URL+"/api/sessions/"+id+"/prompt", map[string]string{"text": instruction, "request_id": "req-wait-prompt"}); resp.StatusCode >= http.StatusBadRequest {
		t.Fatalf("prompt to waiting assignee rejected: %d %s", resp.StatusCode, body)
	}
	srv.dispatchReadyTasks(context.Background())
	dispatchUntilTaskState(t, srv, taskID, state.TaskRunning)
	waitPrompts(t, promptLog, 1)
	raw, err := os.ReadFile(promptLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), instruction) {
		t.Fatalf("continuation turn did not carry the person's prompt: %s", raw)
	}
}
