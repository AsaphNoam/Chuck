package server

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// WAIT-PROMPT-1 / TS-09.R69 / TS-10.R29: a person's prompt to an assignee whose
// task is waiting on child work must reach that agent through the assignment
// continuation. The route once prepared the continuation, rejected the prompt
// with 409 "retry shortly" and kept no copy; the continuation turn re-waited
// within seconds, so the instruction was never delivered.
func TestPromptToWaitingAssigneeIsDeliveredByContinuation(t *testing.T) {
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
	// A network retry of the same submission returns the same acceptance.
	for i := 0; i < 2; i++ {
		if resp, body := post(t, ts.URL+"/api/sessions/"+id+"/prompt", map[string]string{"text": instruction, "request_id": "req-wait-prompt"}); resp.StatusCode != http.StatusAccepted || !strings.Contains(string(body), `"delivery":"queued"`) {
			t.Fatalf("prompt to waiting assignee not queued: %d %s", resp.StatusCode, body)
		}
	}
	// The restart release of untracked holds leaves queued assignment input.
	if err := srv.stateStore.ReleaseRestartedAttachments(); err != nil {
		t.Fatal(err)
	}
	if rec, err := srv.stateStore.AttachmentReceipt(id, "req-wait-prompt"); err != nil || rec.Delivery != "prepared" || rec.AssignmentTaskID != taskID {
		t.Fatalf("queued receipt after restart release = %+v %v", rec, err)
	}
	srv.dispatchReadyTasks(context.Background())
	dispatchUntilTaskState(t, srv, taskID, state.TaskRunning)
	waitPrompts(t, promptLog, 1)
	raw, err := os.ReadFile(promptLog)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(raw), instruction) != 1 {
		t.Fatalf("continuation turn did not carry the person's prompt exactly once: %s", raw)
	}
	if rec, err := srv.stateStore.AttachmentReceipt(id, "req-wait-prompt"); err != nil || rec.Delivery != "submitted" || rec.AssignmentTaskID != "" {
		t.Fatalf("delivered receipt = %+v %v", rec, err)
	}
}
