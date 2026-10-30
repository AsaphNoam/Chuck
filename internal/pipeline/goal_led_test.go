package pipeline

import (
	"context"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// goalLedFixture starts a Work → Review → Fix run whose Review is an explicit
// required check, optionally behind an approval gate.
func goalLedFixture(t *testing.T, reviewApproval bool) (*Manager, *fakePublisher, string) {
	t.Helper()
	manager, _, publisher := pipelineManagerFixture(t)
	record, err := manager.templates.Read("quality")
	if err != nil {
		t.Fatal(err)
	}
	record.Template.Stages[1].RequiredForCompletion = true
	record.Template.Stages[1].ApprovalAfterSuccess = reviewApproval
	record.Template.Stages = append(record.Template.Stages, Stage{ID: "fix", Title: "Fix", Objective: "Repair findings.", Inputs: []StageInput{}, Outputs: []StageOutput{}})
	if updated, err := manager.templates.Update("quality", record.Template); err != nil || !updated.Valid {
		t.Fatalf("update template = %+v %v", updated, err)
	}
	detail, _ := startStagePipeline(t, manager, "goal-led", "a_owner", "gen-1")
	return manager, publisher, detail.Run.RunID
}

// finishVisit reports the current visit as the standing owner, settles its
// release as the dispatcher would at turn end, and reconciles the run.
func finishVisit(t *testing.T, manager *Manager, runID string, result state.TaskResult) state.PipelineStageTask {
	t.Helper()
	current, err := manager.store.LatestPipelineStageTask(runID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.store.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = 'a_owner', assigned_generation = 'gen-1', execution_handle = 'ta_'||task_id WHERE task_id = ?`, state.TaskRunning, current.TaskID); err != nil {
		t.Fatal(err)
	}
	if current.StandingAgentID == "" {
		if err := manager.store.BindPipelineStageTaskStandingAgent(current.TaskID, "a_owner"); err != nil {
			t.Fatal(err)
		}
	}
	run, err := manager.store.ReadPipelineRun(runID)
	if err != nil {
		t.Fatal(err)
	}
	// "self" in required_checks names the visit being reported.
	if d := result.Decision; d != nil {
		for k, v := range d.RequiredChecks {
			if v == "self" {
				d.RequiredChecks[k] = current.TaskID
			}
		}
	}
	if result.Outputs == nil && current.StageID == "work" {
		result.Outputs = map[string]string{"implementation": "done"}
	}
	if _, err := manager.store.AcceptPipelineStageTaskResult(current.TaskID, "a_owner", "gen-1", "ta_"+current.TaskID, run.Revision, result); err != nil {
		t.Fatalf("report %s: %v", current.StageID, err)
	}
	if err := manager.store.CompleteTaskRelease(current.TaskID); err != nil {
		t.Fatal(err)
	}
	if err := manager.Reconcile(context.Background(), runID); err != nil {
		t.Fatal(err)
	}
	return current
}

func next(stageID, reason string) *state.PipelineDecision {
	return &state.PipelineDecision{Action: state.DecisionNext, StageID: stageID, Reason: reason}
}

func visits(t *testing.T, manager *Manager, runID string) []string {
	t.Helper()
	stages, err := manager.store.ListPipelineStageTasks(runID)
	if err != nil {
		t.Fatal(err)
	}
	out := []string{}
	for _, s := range stages {
		out = append(out, s.StageID)
	}
	return out
}

func readRun(t *testing.T, manager *Manager, runID string) state.PipelineRunRecord {
	t.Helper()
	run, err := manager.store.ReadPipelineRun(runID)
	if err != nil {
		t.Fatal(err)
	}
	return run
}

// FS-14.A61/A62, TS-09.R66/R70 — a Work → Review → Fix run finishes without
// Continue: Review's findings lead to Fix and a re-review, which completes on
// its own evidence. Only the backward re-review spends a recovery round.
func TestGoalLedRunRepairsAndCompletesWithoutHumanContinuation(t *testing.T) {
	manager, publisher, runID := goalLedFixture(t, false)
	finishVisit(t, manager, runID, state.TaskResult{Outcome: state.OutcomeSuccess, Summary: "built"})
	finishVisit(t, manager, runID, state.TaskResult{Outcome: state.OutcomeSuccess, Summary: "two defects", Decision: next("fix", "review found defects")})
	finishVisit(t, manager, runID, state.TaskResult{Outcome: state.OutcomeSuccess, Summary: "fixed", Decision: next("review", "re-review the repair")})
	if run := readRun(t, manager, runID); run.RecoveryRoundsUsed != 1 || run.State != "queued" {
		t.Fatalf("after revisit run = %s used=%d, want queued with one round spent", run.State, run.RecoveryRoundsUsed)
	}
	current, err := manager.store.LatestPipelineStageTask(runID)
	if err != nil {
		t.Fatal(err)
	}
	task, _ := manager.store.ReadTask(current.TaskID)
	if !strings.Contains(task.Instruction, "re-review the repair") {
		t.Fatalf("revisit assignment lacks the owner's reason: %s", task.Instruction)
	}
	finishVisit(t, manager, runID, state.TaskResult{Outcome: state.OutcomeSuccess, Summary: "clean", Decision: &state.PipelineDecision{Action: state.DecisionComplete, Reason: "repair verified", RequiredChecks: map[string]string{"review": current.TaskID}}})
	run := readRun(t, manager, runID)
	if run.State != "completed" || run.FinalOutcome != state.OutcomeSuccess {
		t.Fatalf("run = %s/%s, want completed success", run.State, run.FinalOutcome)
	}
	if got := strings.Join(visits(t, manager, runID), ","); got != "work,review,fix,review" {
		t.Fatalf("visits = %s", got)
	}
	if n := len(publisher.notifications); n == 0 || publisher.notifications[n-1] != "completed" {
		t.Fatalf("notifications = %v, want completion", publisher.notifications)
	}
	for _, kind := range publisher.notifications {
		if kind == "needs_attention" {
			t.Fatalf("unattended run asked for attention: %v", publisher.notifications)
		}
	}
}

func complete(reason string) *state.PipelineDecision {
	return &state.PipelineDecision{Action: state.DecisionComplete, Reason: reason, RequiredChecks: map[string]string{"review": "self"}}
}

// FS-14.A61 — a clean review omits the unnecessary Fix step with its reason.
func TestCleanReviewCompletesWithoutFix(t *testing.T) {
	manager, _, runID := goalLedFixture(t, false)
	finishVisit(t, manager, runID, state.TaskResult{Outcome: state.OutcomeSuccess, Summary: "built"})
	review := finishVisit(t, manager, runID, state.TaskResult{Outcome: state.OutcomeSuccess, Summary: "clean", Decision: complete("no findings; Fix unnecessary")})
	if run := readRun(t, manager, runID); run.State != "completed" || run.RecoveryRoundsUsed != 0 {
		t.Fatalf("run = %s used=%d", run.State, run.RecoveryRoundsUsed)
	}
	if d, err := manager.store.StageDecision(review.TaskID); err != nil || d.Reason != "no findings; Fix unnecessary" {
		t.Fatalf("omission reason = %+v %v", d, err)
	}
	if got := strings.Join(visits(t, manager, runID), ","); got != "work,review" {
		t.Fatalf("visits = %s", got)
	}
}

// FS-14.A63/A66, TS-09.R70 — failures without a decision become owner
// recovery visits, three per allowance; the fourth pauses recoverably with a
// notification, never a final failure. Resume without text renews the
// allowance once and continues the same run.
func TestRecoveryExhaustionPausesAndResumeRenewsTheAllowance(t *testing.T) {
	manager, publisher, runID := goalLedFixture(t, false)
	for round := 1; round <= state.PipelineRecoveryRounds; round++ {
		finishVisit(t, manager, runID, state.TaskResult{Outcome: state.OutcomeFailure, Summary: "tests fail"})
		run := readRun(t, manager, runID)
		if run.State != "queued" || run.RecoveryRoundsUsed != round {
			t.Fatalf("round %d run = %s used=%d", round, run.State, run.RecoveryRoundsUsed)
		}
		current, _ := manager.store.LatestPipelineStageTask(runID)
		task, _ := manager.store.ReadTask(current.TaskID)
		if current.StageID != "work" || !strings.Contains(task.Instruction, "tests fail") || task.TargetAgentID != "a_owner" {
			t.Fatalf("recovery visit %d = %+v / %q", round, current, task.Instruction)
		}
	}
	finishVisit(t, manager, runID, state.TaskResult{Outcome: state.OutcomeFailure, Summary: "still failing"})
	run := readRun(t, manager, runID)
	if run.State != "paused" || run.AttentionReason != state.PauseRecoveryExhausted || run.FinalOutcome != "" {
		t.Fatalf("exhausted run = %s/%s/%s, want a recoverable pause", run.State, run.AttentionReason, run.FinalOutcome)
	}
	if n := len(publisher.notifications); publisher.notifications[n-1] != "needs_attention" {
		t.Fatalf("notifications = %v", publisher.notifications)
	}
	if got := len(visits(t, manager, runID)); got != 4 {
		t.Fatalf("visits after exhaustion = %d, want no extra visit", got)
	}
	// Resume after an external repair needs no invented text.
	if _, err := manager.Continue(context.Background(), runID, run.Revision, ""); err != nil {
		t.Fatal(err)
	}
	resumed := readRun(t, manager, runID)
	if resumed.State != "queued" || resumed.RecoveryRoundsUsed != 0 || resumed.AttentionReason != "" {
		t.Fatalf("resumed run = %s used=%d attention=%q", resumed.State, resumed.RecoveryRoundsUsed, resumed.AttentionReason)
	}
	// The same revision cannot renew twice.
	if _, err := manager.Continue(context.Background(), runID, run.Revision, ""); err == nil {
		t.Fatal("a duplicate resume renewed the allowance again")
	}
}

// FS-14.A63, R99 — a genuinely missing decision pauses with one concrete
// question; the answer continues the same goal on a new visit.
func TestQuestionPausesAndTheAnswerContinuesTheRun(t *testing.T) {
	manager, _, runID := goalLedFixture(t, false)
	finishVisit(t, manager, runID, state.TaskResult{Outcome: state.OutcomeBlocked, Summary: "storage unclear", Decision: &state.PipelineDecision{Action: state.DecisionNeedsInput, Question: "Postgres or SQLite?", Reason: "the spec allows both"}})
	run := readRun(t, manager, runID)
	if run.State != "paused" || run.AttentionReason != "needs_input" || run.PauseQuestion != "Postgres or SQLite?" {
		t.Fatalf("question pause = %+v", run)
	}
	if _, err := manager.Continue(context.Background(), runID, run.Revision, ""); err == nil {
		t.Fatal("an empty answer continued a question pause")
	}
	if _, err := manager.Continue(context.Background(), runID, run.Revision, "SQLite"); err != nil {
		t.Fatal(err)
	}
	current, _ := manager.store.LatestPipelineStageTask(runID)
	task, _ := manager.store.ReadTask(current.TaskID)
	if current.StageID != "work" || !strings.Contains(task.Instruction, "Postgres or SQLite?") || !strings.Contains(task.Instruction, "SQLite") {
		t.Fatalf("answer visit = %+v %q", current, task.Instruction)
	}
	if after := readRun(t, manager, runID); after.PauseQuestion != "" || after.RecoveryRoundsUsed != 0 {
		t.Fatalf("after answer = %+v", after)
	}
}

// FS-14.A68, TS-09.R68 — an approval gate holds the accepted decision, even a
// completion, until the person approves that visit; the gate cannot be skipped.
func TestApprovalGateHoldsCompletionUntilApproved(t *testing.T) {
	manager, _, runID := goalLedFixture(t, true)
	finishVisit(t, manager, runID, state.TaskResult{Outcome: state.OutcomeSuccess, Summary: "built"})
	finishVisit(t, manager, runID, state.TaskResult{Outcome: state.OutcomeSuccess, Summary: "clean", Decision: complete("ready")})
	run := readRun(t, manager, runID)
	if run.State != "paused" || run.PendingAction != "await_approval" {
		t.Fatalf("gate = %s/%s, want awaiting approval", run.State, run.PendingAction)
	}
	if _, err := manager.Continue(context.Background(), runID, run.Revision, ""); err != nil {
		t.Fatal(err)
	}
	if after := readRun(t, manager, runID); after.State != "completed" {
		t.Fatalf("after approval = %s", after.State)
	}
}
