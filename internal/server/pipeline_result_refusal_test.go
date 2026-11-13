package server

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// TS-09.R68, INV §§21/22 — the real MCP tool names the correction for the
// transactional guard that refused the result, without finishing the task.
func TestStageResultRefusalExplainsCorrection(t *testing.T) {
	srv := testServer(t, true)
	var logs bytes.Buffer
	srv.log = slog.New(slog.NewJSONHandler(&logs, nil))
	ts := httptest.NewServer(srv.routes())
	t.Cleanup(ts.Close)
	cs := contextMCPSession(t, srv, ts, "a_owner", "generation")
	now := time.Now().UTC()
	_, _, err := srv.stateStore.CreatePipelineRun(state.CreatePipelineRunParams{
		Run: state.PipelineRunRecord{
			RunID: "pr_refusal", TemplateID: "quality", TemplateSnapshot: json.RawMessage(`{"version":2}`),
			DisplayName: "Quality", Project: "tmpproj", Goal: "ship", Inputs: json.RawMessage(`{}`),
			Assignments: json.RawMessage(`{}`), State: "running", Revision: 1,
			CurrentStageID: "implement", CreatedAt: now, UpdatedAt: now,
		},
		RequestID: "stage-refusals", RequestHash: "hash",
		InitialStageTask: &state.CreatePipelineStageTaskParams{
			RunID: "pr_refusal", ExpectedRevision: 1, StageIndex: 0, AttemptNumber: 1, StageID: "implement",
			OutputValues: map[string]string{"implementation": "work"},
			Task: state.Task{TaskID: "tk_refusal", Project: "tmpproj", DisplayName: "Implement",
				Instruction: "implement", TargetKind: state.TargetLaunch, Role: "impl", CreatedByKind: "pipeline"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = 'a_owner', assigned_generation = 'generation', execution_handle = 'ta_current' WHERE task_id = 'tk_refusal'`, state.TaskRunning); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, code, retry, correction, handle string
		outputs                               map[string]string
		prepare                               string
	}{
		{name: "missing output", code: "stage_output_missing", retry: "after_change", correction: `requires a non-empty output named "implementation"`, handle: "ta_current"},
		{name: "undeclared output", code: "stage_output_undeclared", retry: "after_change", correction: `Output "other" is not declared`, handle: "ta_current", outputs: map[string]string{"other": "private value"}},
		{name: "missing handle", code: "stage_handle_required", retry: "after_change", correction: "execution_handle from get_assigned_task"},
		{name: "wrong handle", code: "stale_assignment", retry: "never", correction: "execution_handle is stale", handle: "ta_previous"},
		{name: "pending wait", code: "stage_waiting", retry: "never", correction: "report only after the task resumes", handle: "ta_current", prepare: `UPDATE tasks SET pending_yield = 1 WHERE task_id = 'tk_refusal'`},
		{name: "stopping run", code: "stage_run_closed", retry: "never", correction: "pipeline is stopping", handle: "ta_current", prepare: `UPDATE pipeline_runs SET state = 'stopping' WHERE run_id = 'pr_refusal'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.prepare != "" {
				if _, err := srv.stateStore.DB().Exec(tc.prepare); err != nil {
					t.Fatal(err)
				}
			}
			before, err := srv.stateStore.ReadPipelineRun("pr_refusal")
			if err != nil {
				t.Fatal(err)
			}
			logs.Reset()
			args := map[string]any{"outcome": "success", "summary": "private summary"}
			if tc.handle != "" {
				args["execution_handle"] = tc.handle
			}
			if tc.outputs != nil {
				args["outputs"] = tc.outputs
			}
			out, refused := callContextTool(t, cs, "report_task_result", args)
			retry, _ := out["retry"].(map[string]any)
			message, _ := out["message"].(string)
			if !refused || out["error"] != tc.code || retry["class"] != tc.retry || !strings.Contains(message, tc.correction) {
				t.Fatalf("report_task_result = %v, refused=%t, want %s/%s with %q", out, refused, tc.code, tc.retry, tc.correction)
			}
			var trace map[string]any
			if err := json.Unmarshal(logs.Bytes(), &trace); err != nil {
				t.Fatalf("expected one structured refusal trace: %v (%s)", err, &logs)
			}
			if trace["level"] != "WARN" || trace["code"] != tc.code || trace["run_id"] != "pr_refusal" || trace["task_id"] != "tk_refusal" ||
				trace["agent_id"] != "a_owner" || trace["generation"] != "generation" || strings.Contains(logs.String(), "private") {
				t.Fatalf("wrong refusal trace: %s", &logs)
			}
			task, err := srv.stateStore.ReadTask("tk_refusal")
			if err != nil || task.State != state.TaskRunning || task.Outcome != "" || task.PendingRelease {
				t.Fatalf("refusal changed the task: %+v, %v", task, err)
			}
			after, err := srv.stateStore.ReadPipelineRun("pr_refusal")
			if err != nil || after.Revision != before.Revision || after.State != before.State || after.PendingAction != before.PendingAction {
				t.Fatalf("refusal moved the run: before %+v, after %+v, %v", before, after, err)
			}
		})
	}
}
