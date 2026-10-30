package state

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// decisionRun starts a Work → Review → Fix run whose Review step is an explicit
// required check, with the given stage visit running under owner/gen/handle.
func decisionRun(t *testing.T) (*Store, string, string) {
	t.Helper()
	st, _ := newTestStore(t)
	runID, _ := st.NewPipelineRunID()
	now := time.Now().UTC()
	workID, _ := st.NewTaskID()
	snapshot := `{"version":2,"stages":[{"id":"work","title":"Work","objective":"build"},{"id":"review","title":"Review","objective":"check","required_for_completion":true},{"id":"fix","title":"Fix","objective":"repair"}]}`
	if _, _, err := st.CreatePipelineRun(CreatePipelineRunParams{Run: PipelineRunRecord{
		RunID: runID, TemplateID: "wrf", TemplateSnapshot: json.RawMessage(snapshot),
		DisplayName: "WRF", Project: "proj", Goal: "ship", Inputs: json.RawMessage(`{}`), Assignments: json.RawMessage(`{}`),
		State: "queued", Revision: 1, PendingAction: "dispatch_stage_task", CurrentStageID: "work", CreatedAt: now, UpdatedAt: now,
	}, RequestID: "decide", RequestHash: "hash", InitialStageTask: &CreatePipelineStageTaskParams{
		RunID: runID, ExpectedRevision: 1, StageIndex: 0, AttemptNumber: 1, StageID: "work",
		Task: Task{TaskID: workID, Project: "proj", DisplayName: "Work", Instruction: "work", TargetKind: TargetLaunch, Role: "orchestrator", CreatedByKind: "pipeline"},
	}}); err != nil {
		t.Fatal(err)
	}
	runVisit(t, st, workID)
	return st, runID, workID
}

func runVisit(t *testing.T, st *Store, taskID string) {
	t.Helper()
	if _, err := st.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = 'owner', assigned_generation = 'gen', execution_handle = 'ta_'||task_id WHERE task_id = ?`, TaskRunning, taskID); err != nil {
		t.Fatal(err)
	}
}

func report(st *Store, runID, taskID string, r TaskResult) error {
	run, err := st.ReadPipelineRun(runID)
	if err != nil {
		return err
	}
	_, err = st.AcceptPipelineStageTaskResult(taskID, "owner", "gen", "ta_"+taskID, run.Revision, r)
	return err
}

// nextVisit closes the current visit and opens stageID as the newest visit.
func nextVisit(t *testing.T, st *Store, runID string, index int, stageID string) string {
	t.Helper()
	if _, err := st.DB().Exec(`UPDATE pipeline_stage_tasks SET state = 'closed' WHERE run_id = ?`, runID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB().Exec(`UPDATE tasks SET pending_release = 0 WHERE pending_release = 1`); err != nil {
		t.Fatal(err)
	}
	run, err := st.ReadPipelineRun(runID)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := st.NextPipelineStageAttempt(runID, index)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := st.NewTaskID()
	if _, _, _, err := st.CreatePipelineStageTask(CreatePipelineStageTaskParams{RunID: runID, ExpectedRevision: run.Revision, StageIndex: index, AttemptNumber: attempt, StageID: stageID,
		Task: Task{TaskID: id, Project: "proj", DisplayName: stageID, Instruction: stageID, TargetKind: TargetLaunch, Role: "orchestrator", CreatedByKind: "pipeline"}}); err != nil {
		t.Fatal(err)
	}
	runVisit(t, st, id)
	return id
}

func assertRefused(t *testing.T, st *Store, taskID string, err error, what string) {
	t.Helper()
	var refusal *DecisionError
	if !errors.As(err, &refusal) || refusal.Message == "" {
		t.Fatalf("%s: error = %v, want a correctable decision refusal", what, err)
	}
	task, readErr := st.ReadTask(taskID)
	if readErr != nil || task.State != TaskRunning || task.Outcome != "" || task.PendingRelease {
		t.Fatalf("%s left the assignment unactionable: %+v %v", what, task, readErr)
	}
}

// TS-09.R66/R68, FS-14.A62 — the decision travels with the result: a success
// takes the usual next step when none is given, a malformed decision is a
// correctable refusal that records nothing, and a revisit is recorded with its
// reason.
func TestStageDecisionIsRecordedWithTheResultOrRefusedForCorrection(t *testing.T) {
	st, runID, workID := decisionRun(t)

	assertRefused(t, st, workID, report(st, runID, workID, TaskResult{Outcome: OutcomeSuccess, Summary: "built", Decision: &PipelineDecision{Action: DecisionNext, StageID: "deploy", Reason: "ship"}}), "unknown step")
	assertRefused(t, st, workID, report(st, runID, workID, TaskResult{Outcome: OutcomeSuccess, Summary: "built", Decision: &PipelineDecision{Action: DecisionNext, StageID: "review"}}), "missing reason")
	assertRefused(t, st, workID, report(st, runID, workID, TaskResult{Outcome: OutcomeSuccess, Summary: "built", Decision: &PipelineDecision{Action: DecisionNeedsInput, Reason: "unclear"}}), "question missing")
	assertRefused(t, st, workID, report(st, runID, workID, TaskResult{Outcome: OutcomeFailure, Summary: "broke", Decision: &PipelineDecision{Action: DecisionComplete, Reason: "done anyway"}}), "complete after failure")

	if err := report(st, runID, workID, TaskResult{Outcome: OutcomeSuccess, Summary: "built"}); err != nil {
		t.Fatal(err)
	}
	if d, err := st.StageDecision(workID); err != nil || d == nil || d.Action != DecisionNext || d.StageID != "review" || !d.Implicit {
		t.Fatalf("implicit decision = %+v %v, want the usual next step", d, err)
	}

	reviewID := nextVisit(t, st, runID, 1, "review")
	if err := report(st, runID, reviewID, TaskResult{Outcome: OutcomeSuccess, Summary: "two defects", Decision: &PipelineDecision{Action: DecisionNext, StageID: "fix", Reason: "defects found"}}); err != nil {
		t.Fatal(err)
	}
	fixID := nextVisit(t, st, runID, 2, "fix")
	// The last step cannot end the run by implicit success.
	assertRefused(t, st, fixID, report(st, runID, fixID, TaskResult{Outcome: OutcomeSuccess, Summary: "fixed"}), "implicit end of plan")
	if err := report(st, runID, fixID, TaskResult{Outcome: OutcomeSuccess, Summary: "fixed", Decision: &PipelineDecision{Action: DecisionNext, StageID: "review", Reason: "re-review the fix"}}); err != nil {
		t.Fatal(err)
	}
	if d, err := st.StageDecision(fixID); err != nil || d.StageID != "review" || d.Reason != "re-review the fix" || d.Implicit {
		t.Fatalf("revisit decision = %+v %v", d, err)
	}
}

// TS-09.R68, FS-14.A62 — completion needs evidence for every explicit check:
// an omitted required review, a non-success or foreign task, and evidence
// naming the wrong step are refused; the current successful result of the
// required step itself is valid evidence, avoiding an extra completion turn.
func TestCompletionRequiresAcceptedEvidenceForEveryRequiredCheck(t *testing.T) {
	st, runID, workID := decisionRun(t)
	if err := report(st, runID, workID, TaskResult{Outcome: OutcomeSuccess, Summary: "built", Decision: &PipelineDecision{Action: DecisionNext, StageID: "fix", Reason: "skip review"}}); err != nil {
		t.Fatal(err)
	}
	fixID := nextVisit(t, st, runID, 2, "fix")
	complete := func(checks map[string]string) TaskResult {
		return TaskResult{Outcome: OutcomeSuccess, Summary: "done", Decision: &PipelineDecision{Action: DecisionComplete, Reason: "goal met", RequiredChecks: checks}}
	}
	assertRefused(t, st, fixID, report(st, runID, fixID, complete(nil)), "required review omitted")
	assertRefused(t, st, fixID, report(st, runID, fixID, complete(map[string]string{"review": workID})), "evidence from another step")
	assertRefused(t, st, fixID, report(st, runID, fixID, complete(map[string]string{"review": fixID})), "current task is not the review")
	assertRefused(t, st, fixID, report(st, runID, fixID, complete(map[string]string{"review": "tk_elsewhere"})), "foreign evidence")

	if err := report(st, runID, fixID, TaskResult{Outcome: OutcomeSuccess, Summary: "fixed", Decision: &PipelineDecision{Action: DecisionNext, StageID: "review", Reason: "required review"}}); err != nil {
		t.Fatal(err)
	}
	reviewID := nextVisit(t, st, runID, 1, "review")
	if err := report(st, runID, reviewID, complete(map[string]string{"review": reviewID})); err != nil {
		t.Fatalf("completion with the current review as evidence: %v", err)
	}
	if d, err := st.StageDecision(reviewID); err != nil || d.Action != DecisionComplete || d.RequiredChecks["review"] != reviewID {
		t.Fatalf("completion decision = %+v %v", d, err)
	}
}

// TS-09.R67/R68 — only the selected destination is validated: routing past the
// producer of a required input is a correctable refusal, while an oversized
// output that only an unselected room would read does not block completion.
func TestDecisionValidatesOnlyTheSelectedDestination(t *testing.T) {
	st, _ := newTestStore(t)
	runID, _ := st.NewPipelineRunID()
	workID, _ := st.NewTaskID()
	now := time.Now().UTC()
	snapshot := `{"version":2,"stages":[{"id":"work","title":"Work","objective":"build","outputs":[{"name":"draft","value":"draft"}]},` +
		`{"id":"review","title":"Review","objective":"check","outputs":[{"name":"verdict","value":"verdict"}]},` +
		`{"id":"publish","title":"Publish","objective":"publish","inputs":[{"name":"verdict","value":"verdict","required":true}]},` +
		`{"id":"room","title":"Room","objective":"choose","coordination":"think_tank","inputs":[{"name":"draft","value":"draft"}],"outputs":[{"name":"answer","value":"answer"}]}]}`
	if _, _, err := st.CreatePipelineRun(CreatePipelineRunParams{Run: PipelineRunRecord{
		RunID: runID, TemplateID: "dest", TemplateSnapshot: json.RawMessage(snapshot), DisplayName: "Dest", Project: "proj", Goal: "ship",
		Inputs: json.RawMessage(`{}`), Assignments: json.RawMessage(`{}`), State: "queued", Revision: 1, CurrentStageID: "work", CreatedAt: now, UpdatedAt: now,
	}, RequestID: "dest", RequestHash: "dest", InitialStageTask: &CreatePipelineStageTaskParams{
		RunID: runID, ExpectedRevision: 1, StageIndex: 0, AttemptNumber: 1, StageID: "work", OutputValues: map[string]string{"draft": "draft"},
		Task: Task{TaskID: workID, Project: "proj", DisplayName: "Work", Instruction: "work", TargetKind: TargetLaunch, Role: "orchestrator", CreatedByKind: "pipeline"},
	}}); err != nil {
		t.Fatal(err)
	}
	runVisit(t, st, workID)
	oversized := map[string]string{"draft": strings.Repeat("<", 44000)}
	assertRefused(t, st, workID, report(st, runID, workID, TaskResult{Outcome: OutcomeSuccess, Summary: "built", Outputs: oversized, Decision: &PipelineDecision{Action: DecisionNext, StageID: "publish", Reason: "skip review"}}), "required input never produced")
	if err := report(st, runID, workID, TaskResult{Outcome: OutcomeSuccess, Summary: "built", Outputs: oversized, Decision: &PipelineDecision{Action: DecisionNext, StageID: "room", Reason: "debate"}}); !errors.Is(err, ErrThinkTankOutputRefused) {
		t.Fatalf("selected oversized room error = %v, want ErrThinkTankOutputRefused", err)
	}
	if err := report(st, runID, workID, TaskResult{Outcome: OutcomeSuccess, Summary: "built", Outputs: oversized, Decision: &PipelineDecision{Action: DecisionComplete, Reason: "the optional room is unnecessary"}}); err != nil {
		t.Fatalf("completion rejected because of an unselected room: %v", err)
	}
}

// TS-09.R71 — the assignment context lists the plan, the latest accepted
// evidence for each required step and the allowance, and is non-null.
func TestPipelineAssignmentContextListsPlanAndEvidence(t *testing.T) {
	st, runID, workID := decisionRun(t)
	a, err := st.PipelineAssignmentContext(workID)
	if err != nil || a.RunID != runID || a.VisitSequence != 1 || a.CurrentStageID != "work" || len(a.Steps) != 3 || !a.Steps[1].Required() || a.Steps[2].Required() {
		t.Fatalf("assignment context = %+v %v", a, err)
	}
	if source, ok := a.RequiredChecks["review"]; !ok || source != "" || len(a.RequiredChecks) != 1 {
		t.Fatalf("required checks before review = %+v", a.RequiredChecks)
	}
	if a.Recovery.Limit != PipelineRecoveryRounds || a.Recovery.Used != 0 {
		t.Fatalf("recovery = %+v", a.Recovery)
	}
	raw, _ := json.Marshal(a)
	if string(raw) == "" || json.Valid(raw) == false {
		t.Fatal("assignment context does not serialize")
	}
	if _, err := st.PipelineAssignmentContext("tk_unrelated"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ordinary task context error = %v, want ErrNotFound", err)
	}
}
