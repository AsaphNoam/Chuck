package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/attachment"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

const inputSnapshot = `{"version":2,"title":"x","orchestrator_role":"impl","inputs":[],"stages":[{"id":"stage","title":"Stage","objective":"work","inputs":[],"outputs":[]}]}`

// TS-09.R69, TS-10.R29 — input accepted for a starting assignment stays bound
// to it when that start fails and the same assignment retries.
func TestQueuedInputSurvivesStartRetry(t *testing.T) {
	srv := testServer(t, true)
	if err := srv.stateStore.WriteAgent(state.Agent{AgentID: "a_old", Interface: "chat"}); err != nil {
		t.Fatal(err)
	}
	task := createPipelineSnapshotTask(t, srv, "pr_retry_input", "tk_retry_input", inputSnapshot)
	if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state='starting', assigned_agent_id='a_old', start_attempt_id='attempt', started_at=created_at WHERE task_id=?`, task.TaskID); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := srv.stateStore.QueueAssignmentInput(state.AttachmentReceipt{AgentID: "a_old", RequestID: "retry-input", PayloadHash: "hash", Action: "prompt", MessageJSON: `{"text":"critical correction"}`}); err != nil || !ok {
		t.Fatalf("queue = %v %v", ok, err)
	}
	if _, ok, err := srv.stateStore.FailTaskStart(task.TaskID, "attempt", "temporary start failure"); err != nil || !ok {
		t.Fatalf("fail start = %v %v", ok, err)
	}
	if _, _, err := srv.stateStore.QueuedInputWork(); err != nil {
		t.Fatal(err)
	}
	if rec, err := srv.stateStore.AttachmentReceipt("a_old", "retry-input"); err != nil || rec.Delivery != "prepared" || rec.AssignmentTaskID != task.TaskID {
		t.Fatalf("receipt after start retry = %+v %v, want it still bound to the retrying assignment", rec, err)
	}
	// The retained input stays inspectable after navigation or restart,
	// apart from the transcript (TS-09.R71).
	rec := doJSON(t, srv.routes(), "GET", "/api/sessions/a_old/prompt", ``)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"queued":[{`) || !strings.Contains(rec.Body.String(), "critical correction") || !strings.Contains(rec.Body.String(), task.TaskID) {
		t.Fatalf("held prompt read = %d %s, want the queued correction", rec.Code, rec.Body.String())
	}
}

// TS-09.R69/R70 — an attachment-only message queued for a parked owner moves
// to the automatic replacement's assignment, and its new owner gets its own
// copy of the bytes when the message is delivered; it is never declared
// submitted without them.
func TestQueuedAttachmentFollowsReplacementOwner(t *testing.T) {
	srv := testServer(t, true)
	for _, id := range []string{"a_old", "a_new"} {
		if err := srv.stateStore.WriteAgent(state.Agent{AgentID: id, Interface: "chat"}); err != nil {
			t.Fatal(err)
		}
	}
	store := attachment.New(srv.configStore.Home())
	descriptor, err := store.Put("a_old", "evidence.txt", strings.NewReader("required evidence"))
	if err != nil {
		t.Fatal(err)
	}
	message, _ := json.Marshal(runtime.Message{Attachments: []attachment.Descriptor{descriptor}})
	task := createPipelineSnapshotTask(t, srv, "pr_attach", "tk_attach", inputSnapshot)
	if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state='waiting', assigned_agent_id='a_old', started_at=created_at WHERE task_id=?`, task.TaskID); err != nil {
		t.Fatal(err)
	}
	if err := srv.stateStore.BindPipelineStageTaskStandingAgent(task.TaskID, "a_old"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := srv.stateStore.QueueAssignmentInput(state.AttachmentReceipt{AgentID: "a_old", RequestID: "attachment-only", PayloadHash: "hash", Action: "prompt", MessageJSON: string(message)}); err != nil || !ok {
		t.Fatalf("queue = %v %v", ok, err)
	}
	if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state='dependency_failed' WHERE task_id=?`, task.TaskID); err != nil {
		t.Fatal(err)
	}
	task, _ = srv.stateStore.ReadTask(task.TaskID)
	srv.recoverParkedStageTask(task)
	successor, err := srv.stateStore.LatestPipelineStageTask("pr_attach")
	if err != nil || successor.TaskID == task.TaskID {
		t.Fatalf("replacement = %+v %v", successor, err)
	}
	if rec, err := srv.stateStore.AttachmentReceipt("a_old", "attachment-only"); err != nil || rec.Delivery != "prepared" || rec.AssignmentTaskID != successor.TaskID {
		t.Fatalf("receipt after replacement = %+v %v, want it pending for the successor", rec, err)
	}
	// The successor's new owner is executing: the queued message is claimed
	// for it, moved to its conversation, and its attachment cloned there.
	if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state='running', assigned_agent_id='a_new' WHERE task_id=?`, successor.TaskID); err != nil {
		t.Fatal(err)
	}
	_, executing, err := srv.stateStore.QueuedInputWork()
	if err != nil || len(executing) != 1 || executing[0].AgentID != "a_new" || executing[0].SourceAgentID != "a_old" {
		t.Fatalf("executing input = %+v %v", executing, err)
	}
	rec, ok, err := srv.stateStore.ClaimQueuedInputForTurn("a_old", "attachment-only")
	if err != nil || !ok || rec.AgentID != "a_new" || rec.SourceAgentID != "a_old" {
		t.Fatalf("claim = %+v %v %v", rec, ok, err)
	}
	var msg runtime.Message
	if err := json.Unmarshal([]byte(rec.MessageJSON), &msg); err != nil {
		t.Fatal(err)
	}
	if err := runtime.CloneQueuedAttachments(srv.configStore.Home(), rec, msg); err != nil {
		t.Fatal(err)
	}
	if f, _, err := store.Open("a_new", descriptor.ID); err != nil {
		t.Fatalf("new owner cannot open the attachment: %v", err)
	} else {
		f.Close()
	}
	if moved, err := srv.stateStore.AttachmentReceipt("a_new", "attachment-only"); err != nil || moved.Delivery != "prepared" {
		t.Fatalf("moved receipt = %+v %v", moved, err)
	}
}

// FS-14.R99, TS-09.R69 — answering a paused run's question in its owner's
// conversation continues the same run on a new visit, as the run page does.
func TestConversationAnswerContinuesQuestionPause(t *testing.T) {
	srv, ts, _ := activationTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	task := createPipelineSnapshotTask(t, srv, "pr_question", "tk_question", inputSnapshot)
	if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state='finished', outcome='success', assigned_agent_id=?, started_at=created_at, finished_at=created_at WHERE task_id=?`, id, task.TaskID); err != nil {
		t.Fatal(err)
	}
	if err := srv.stateStore.BindPipelineStageTaskStandingAgent(task.TaskID, id); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.stateStore.DB().Exec(`UPDATE pipeline_stage_tasks SET state='closed' WHERE task_id=?`, task.TaskID); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.stateStore.DB().Exec(`UPDATE pipeline_runs SET state='paused', attention_reason='needs_input', pause_question='Postgres or SQLite?', current_agent_id=? WHERE run_id='pr_question'`, id); err != nil {
		t.Fatal(err)
	}
	resp, body := post(t, ts.URL+"/api/sessions/"+id+"/prompt", map[string]string{"text": "Use SQLite", "request_id": "question-answer"})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("answer = %d %s", resp.StatusCode, body)
	}
	run, _ := srv.stateStore.ReadPipelineRun("pr_question")
	visit, _ := srv.stateStore.LatestPipelineStageTask("pr_question")
	next, _ := srv.stateStore.ReadTask(visit.TaskID)
	if run.State == "paused" || visit.TaskID == task.TaskID || !strings.Contains(next.Instruction, "Postgres or SQLite?") || !strings.Contains(next.Instruction, "Use SQLite") {
		t.Fatalf("conversation answer did not continue the run: run=%s visit=%s", run.State, visit.TaskID)
	}
	// A network replay returns the recorded acceptance and creates nothing.
	if resp, _ := post(t, ts.URL+"/api/sessions/"+id+"/prompt", map[string]string{"text": "Use SQLite", "request_id": "question-answer"}); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("replay = %d", resp.StatusCode)
	}
	if again, _ := srv.stateStore.LatestPipelineStageTask("pr_question"); again.TaskID != visit.TaskID {
		t.Fatalf("replay created another visit: %s", again.TaskID)
	}
}
