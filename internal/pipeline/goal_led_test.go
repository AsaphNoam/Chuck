package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

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

// FS-14.A65/A68, TS-09.R70 — an interrupted owner is resumed automatically
// on its same identity; an owner whose native session cannot be used is
// replaced by a visibly fresh owner on the run's frozen runtime carrying the
// handoff, the old identity retained. Each spends a round; a spent allowance
// pauses instead.
func TestInterruptedOwnerRecoversAutomaticallyWithinTheAllowance(t *testing.T) {
	manager, _, runID := goalLedFixture(t, false)
	interrupt := func() state.PipelineStageTask {
		t.Helper()
		current, _ := manager.store.LatestPipelineStageTask(runID)
		if _, err := manager.store.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = 'a_owner' WHERE task_id = ?`, state.TaskInterrupted, current.TaskID); err != nil {
			t.Fatal(err)
		}
		return current
	}
	visit := interrupt()
	if ok, err := manager.RecoverStageTask(visit.TaskID, "interrupted", false); err != nil || !ok {
		t.Fatalf("automatic retry = %v %v", ok, err)
	}
	if task, _ := manager.store.ReadTask(visit.TaskID); task.State != state.TaskReady || task.AssignedAgentID != "a_owner" {
		t.Fatalf("retried visit = %+v", task)
	}
	if run := readRun(t, manager, runID); run.State != "queued" || run.RecoveryRoundsUsed != 1 {
		t.Fatalf("after retry run = %s used=%d", run.State, run.RecoveryRoundsUsed)
	}

	visit = interrupt()
	// An attachment-only message queued for the old owner moves, whole, to
	// the successor assignment rather than being declared delivered.
	if err := manager.store.WriteAgent(state.Agent{AgentID: "a_owner", Interface: "chat"}); err != nil {
		t.Fatal(err)
	}
	queued := `{"text":"","attachments":[{"id":"asset_review","name":"evidence.txt","kind":"file"}]}`
	if err := manager.store.BindAssignmentInput(state.AttachmentReceipt{AgentID: "a_owner", RequestID: "attachment-only", PayloadHash: "h", MessageJSON: queued, Action: "prompt"}, visit.TaskID); err != nil {
		t.Fatal(err)
	}
	if ok, err := manager.RecoverStageTask(visit.TaskID, "owner_unavailable", true); err != nil || !ok {
		t.Fatalf("automatic replacement = %v %v", ok, err)
	}
	replacement, _ := manager.store.LatestPipelineStageTask(runID)
	task, _ := manager.store.ReadTask(replacement.TaskID)
	old, _ := manager.store.ReadTask(visit.TaskID)
	if replacement.TaskID == visit.TaskID || task.TargetKind != state.TargetLaunch || task.Backend != "codex" || task.Model != "gpt" {
		t.Fatalf("replacement = %+v / %+v", replacement, task)
	}
	if rec, err := manager.store.AttachmentReceipt("a_owner", "attachment-only"); err != nil || rec.Delivery != "prepared" || rec.AssignmentTaskID != replacement.TaskID || rec.MessageJSON != queued {
		t.Fatalf("queued input after replacement = %+v %v, want it pending for the successor", rec, err)
	}
	if old.State != state.TaskFinished || old.OutcomeSummary != "replaced" {
		t.Fatalf("old owner visit = %+v, want retained as replaced", old)
	}
	if run := readRun(t, manager, runID); run.RecoveryRoundsUsed != 2 {
		t.Fatalf("after replacement used=%d", run.RecoveryRoundsUsed)
	}

	if _, err := manager.store.DB().Exec(`UPDATE pipeline_runs SET recovery_rounds_used = ? WHERE run_id = ?`, state.PipelineRecoveryRounds, runID); err != nil {
		t.Fatal(err)
	}
	visit = interrupt()
	if ok, err := manager.RecoverStageTask(visit.TaskID, "interrupted", false); err != nil || ok {
		t.Fatalf("recovery past the allowance = %v %v", ok, err)
	}
	run := readRun(t, manager, runID)
	if run.State != "paused" || run.AttentionReason != state.PauseRecoveryExhausted {
		t.Fatalf("exhausted = %s/%s", run.State, run.AttentionReason)
	}
	// Resume continues the same interrupted owner with a fresh allowance; its
	// optional instructions are retained for that owner's resumed turn.
	if err := manager.store.BindPipelineStageTaskStandingAgent(visit.TaskID, "a_owner"); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Continue(context.Background(), runID, run.Revision, "Repair the migration first"); err != nil {
		t.Fatal(err)
	}
	if after := readRun(t, manager, runID); after.State != "queued" || after.RecoveryRoundsUsed != 0 {
		t.Fatalf("resumed = %s used=%d", after.State, after.RecoveryRoundsUsed)
	}
	if task, _ := manager.store.ReadTask(visit.TaskID); task.State != state.TaskReady {
		t.Fatalf("resumed visit = %s", task.State)
	}
	receipt, err := manager.store.AttachmentReceipt("a_owner", fmt.Sprintf("run_%s_%d", runID, run.Revision))
	if err != nil || receipt.Delivery != "prepared" || receipt.AssignmentTaskID != visit.TaskID || !strings.Contains(receipt.MessageJSON, "Repair the migration first") {
		t.Fatalf("resume instructions = %+v %v, want them retained for the resumed visit", receipt, err)
	}
}

// TS-09.R70 — a retained controller action that keeps failing pauses
// recoverably after a finite policy, stops automatic retries, and Resume
// retries it once with its intent intact.
func TestPersistentControllerFailurePausesAndResumes(t *testing.T) {
	m, _, _ := pipelineManagerFixture(t)
	detail, visit := startStagePipeline(t, m, "controller-exhaustion", "a_owner", "gen-1")
	if _, err := m.store.DB().Exec(`UPDATE tasks SET execution_handle = 'ta_ctl' WHERE task_id = ?`, visit.TaskID); err != nil {
		t.Fatal(err)
	}
	if _, err := m.store.AcceptPipelineStageTaskResult(visit.TaskID, "a_owner", "gen-1", "ta_ctl", detail.Run.Revision,
		state.TaskResult{Outcome: state.OutcomeSuccess, Summary: "built", Outputs: map[string]string{"implementation": "done"}}); err != nil {
		t.Fatal(err)
	}
	if err := m.store.CompleteTaskRelease(visit.TaskID); err != nil {
		t.Fatal(err)
	}
	run := readRun(t, m, detail.Run.RunID)
	var err error
	for i := 0; i < state.MaxPipelineControllerFailures; i++ {
		if run, err = m.store.RecordPipelineControllerFailure(run, errors.New("persistent cursor write failure"), time.Now().Add(-time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	if run.State != "paused" || run.PendingAction != "release_stage_task" || run.AttentionReason != state.PauseControllerRecoveryFailed {
		t.Fatalf("after %d failures run = %s/%s/%s, want a recoverable pause holding the intent", run.ControllerFailures, run.State, run.PendingAction, run.AttentionReason)
	}
	if due, err := m.store.ListDuePipelineControllerActions(time.Now(), 8); err != nil || len(due) != 0 {
		t.Fatalf("held action still retried automatically: %+v %v", due, err)
	}
	if _, err := m.Continue(context.Background(), run.RunID, run.Revision, ""); err != nil {
		t.Fatalf("Resume after repair: %v", err)
	}
	if got := visits(t, m, run.RunID); len(got) != 2 {
		t.Fatalf("visits after Resume = %v, want the decided next step once", got)
	}
}
