package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// mixedRoomFixture saves ordinary → think_tank → ordinary and starts it.
func mixedRoomFixture(t *testing.T, approval bool) (*Manager, *fakeLifecycle, RunDetail) {
	t.Helper()
	manager, lifecycle, _ := pipelineManagerFixture(t)
	template := Template{
		Version: 2, Title: "Mixed", OrchestratorRole: "orchestrator",
		Inputs: []ValueDecl{{Name: "spec", Description: "Specification", Required: true}},
		Stages: []Stage{
			{ID: "draft", Title: "Draft", Objective: "Draft options.",
				Inputs:  []StageInput{{Name: "spec", Value: "spec", Required: true}},
				Outputs: []StageOutput{{Name: "options", Value: "options", Description: "Options"}}},
			{ID: "debate", Title: "Debate", Objective: "Choose an option.", Coordination: CoordinationThinkTank, ApprovalAfterSuccess: approval,
				ThinkTank: &ThinkTankStage{JudgeRole: "reviewer", Participants: []ThinkTankParticipant{
					{ID: "pro", Role: "implementer", Limit: 2}, {ID: "con", Role: "implementer", Limit: 2},
				}},
				Inputs:  []StageInput{{Name: "options", Value: "options", Required: true}},
				Outputs: []StageOutput{{Name: "decision", Value: "decision", Description: "Judge synthesis"}}},
			{ID: "build", Title: "Build", Objective: "Build the decision.",
				Inputs: []StageInput{{Name: "decision", Value: "decision", Required: true}}, Outputs: []StageOutput{}},
		},
	}
	if record, err := manager.templates.Create("mixed", template); err != nil || !record.Valid {
		t.Fatalf("create = %+v err=%v", record, err)
	}
	slot := RuntimeAssignment{Backend: "claude", Model: "sonnet"}
	detail, _, err := manager.Start(context.Background(), StartRequest{
		RequestID: "mixed", TemplateID: "mixed", Project: "app", Goal: "Pick and build",
		Inputs: map[string]string{"spec": "the spec"}, Orchestrator: slot,
		ThinkTankAssignments: map[string]ThinkTankAssignment{"debate": {
			Participants: map[string]RuntimeAssignment{"pro": slot, "con": {Backend: "codex", Model: "gpt"}}, Judge: slot,
		}},
	})
	if err != nil {
		t.Fatalf("Start = %v", err)
	}
	return manager, lifecycle, detail
}

// FS-14.A52, TS-09.R67 — a room synthesis too large for a later room's context
// is accepted as this room's result; routing to that room is the correctable
// refusal, and the owner can choose another step.
func TestRoomOutputOverflowRefusesOnlyTheSelectedRoom(t *testing.T) {
	manager, _, detail := mixedRoomFixture(t, false)
	runID := detail.Run.RunID
	finishOrdinaryStage(t, manager, runID, "a_owner", map[string]string{"options": "A or B"})
	room, _, _ := manager.currentStageTask(runID)
	updatedTemplate := detail.Template
	updatedTemplate.Stages[2].Coordination = CoordinationThinkTank
	updatedTemplate.Stages[2].Inputs = []StageInput{{Name: "decision", Value: "decision", Required: true}}
	updatedTemplate.Stages[2].Outputs = []StageOutput{{Name: "final", Value: "final", Description: "Final"}}
	raw, err := json.Marshal(updatedTemplate)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.store.DB().Exec(`UPDATE pipeline_runs SET template_snapshot_json = ? WHERE run_id = ?`, raw, runID); err != nil {
		t.Fatal(err)
	}
	publishSynthesis(t, manager, room.RoomID, strings.Repeat("<", 44000))
	if err := manager.AcceptRoomOutput(context.Background(), runID); err != nil {
		t.Fatal(err)
	}
	if task, err := manager.store.ReadTask(room.TaskID); err != nil || task.Outcome != state.OutcomeSuccess {
		t.Fatalf("room output = %+v err=%v", task, err)
	}
	decide, _, _ := manager.currentStageTask(runID)
	if decide.DecisionSourceTaskID != room.TaskID {
		t.Fatalf("current visit = %+v, want the owner's decision", decide)
	}
	if _, err := manager.store.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = 'a_owner', assigned_generation = 'gen', execution_handle = 'ta_d' WHERE task_id = ?`, state.TaskRunning, decide.TaskID); err != nil {
		t.Fatal(err)
	}
	run, _ := manager.store.ReadPipelineRun(runID)
	_, err = manager.store.AcceptPipelineStageTaskResult(decide.TaskID, "a_owner", "gen", "ta_d", run.Revision, state.TaskResult{Outcome: state.OutcomeSuccess, Summary: "build it"})
	if !errors.Is(err, state.ErrThinkTankOutputRefused) {
		t.Fatalf("routing to the oversized room = %v, want a refusal", err)
	}
	if task, _ := manager.store.ReadTask(decide.TaskID); task.State != state.TaskRunning || task.Outcome != "" {
		t.Fatalf("refused decision left the assignment unactionable: %+v", task)
	}
}

// TS-09.R66/R68 — a final room's synthesis is evidence for the owner's
// decision, never the run's completion by itself; the decision visit cannot
// serve as the room's required evidence.
func TestFinalRoomSynthesisWaitsForTheOwnersDecision(t *testing.T) {
	manager, _, detail := mixedRoomFixture(t, false)
	runID := detail.Run.RunID
	template := detail.Template
	template.Stages = template.Stages[:2]
	template.Stages[1].RequiredForCompletion = true
	raw, err := json.Marshal(template)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.store.DB().Exec(`UPDATE pipeline_runs SET template_snapshot_json = ? WHERE run_id = ?`, raw, runID); err != nil {
		t.Fatal(err)
	}
	finishOrdinaryStage(t, manager, runID, "a_owner", map[string]string{"options": "implementation ready for review"})
	room, _, _ := manager.currentStageTask(runID)
	publishSynthesis(t, manager, room.RoomID, "Must fix: implementation loses user input. Repair and re-review required.")
	if err := manager.AcceptRoomOutput(context.Background(), runID); err != nil {
		t.Fatal(err)
	}
	if run := readRun(t, manager, runID); run.State == "completed" {
		t.Fatalf("room synthesis completed the run: %+v", run)
	}
	decide, _, _ := manager.currentStageTask(runID)
	if decide.DecisionSourceTaskID != room.TaskID {
		t.Fatalf("current visit = %+v, want the owner's decision", decide)
	}
	if _, err := manager.store.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = 'a_owner', assigned_generation = 'gen', execution_handle = 'ta_d' WHERE task_id = ?`, state.TaskRunning, decide.TaskID); err != nil {
		t.Fatal(err)
	}
	finish := func(checks map[string]string) error {
		run := readRun(t, manager, runID)
		_, err := manager.store.AcceptPipelineStageTaskResult(decide.TaskID, "a_owner", "gen", "ta_d", run.Revision, state.TaskResult{Outcome: state.OutcomeSuccess, Summary: "review passed", Decision: &state.PipelineDecision{Action: state.DecisionComplete, Reason: "review is clean", RequiredChecks: checks}})
		return err
	}
	var refusal *state.DecisionError
	if err := finish(map[string]string{"debate": decide.TaskID}); !errors.As(err, &refusal) {
		t.Fatalf("decision visit as its own evidence = %v, want a refusal", err)
	}
	if err := finish(map[string]string{"debate": room.TaskID}); err != nil {
		t.Fatal(err)
	}
	if err := manager.store.CompleteTaskRelease(decide.TaskID); err != nil {
		t.Fatal(err)
	}
	if err := manager.Reconcile(context.Background(), runID); err != nil {
		t.Fatal(err)
	}
	if run := readRun(t, manager, runID); run.State != "completed" || run.FinalOutcome != state.OutcomeSuccess {
		t.Fatalf("run after the owner's completion = %+v", run)
	}
}

// finishOrdinaryStage reports the current ordinary stage through the agent
// branch and lets its release converge.
func finishOrdinaryStage(t *testing.T, manager *Manager, runID, agentID string, outputs map[string]string) {
	t.Helper()
	stage, found, err := manager.currentStageTask(runID)
	if err != nil || !found {
		t.Fatalf("current stage: %v", err)
	}
	if _, err := manager.store.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = ?, assigned_generation = 'gen', execution_handle = 'ta_x' WHERE task_id = ?`,
		state.TaskRunning, agentID, stage.TaskID); err != nil {
		t.Fatal(err)
	}
	if err := manager.store.BindPipelineStageTaskStandingAgent(stage.TaskID, agentID); err != nil {
		t.Fatal(err)
	}
	run, _ := manager.store.ReadPipelineRun(runID)
	if _, err := manager.store.AcceptPipelineStageTaskResult(stage.TaskID, agentID, "gen", "ta_x", run.Revision,
		state.TaskResult{Outcome: state.OutcomeSuccess, Summary: "done", Outputs: outputs}); err != nil {
		t.Fatal(err)
	}
	if err := manager.store.CompleteTaskRelease(stage.TaskID); err != nil {
		t.Fatal(err)
	}
	if err := manager.Reconcile(context.Background(), runID); err != nil {
		t.Fatal(err)
	}
}

// publishSynthesis stands in for the room engine's judge finalization.
func publishSynthesis(t *testing.T, manager *Manager, roomID, body string) {
	t.Helper()
	if _, err := manager.store.DB().Exec(`INSERT INTO think_tank_entries(room_id, seq, kind, agent_id, body, created_at) VALUES (?, (SELECT COALESCE(MAX(seq), 0) + 1 FROM think_tank_entries WHERE room_id = ?), 'synthesis', 'a_judge', ?, '2026-10-08T00:00:00Z')`, roomID, roomID, body); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.store.DB().Exec(`UPDATE think_tanks SET judge_status = 'completed', phase = 'ended' WHERE room_id = ?`, roomID); err != nil {
		t.Fatal(err)
	}
}

// FS-14.A48 — ordinary → Think Tank → ordinary: the room receives the produced
// input as stage context, nothing advances before the judge's synthesis is
// published, the exact synthesis becomes the named output once, and the next
// ordinary stage reuses the standing owner without any standing room turn.
func TestMixedRoomStageAcceptsExactSynthesisOnce(t *testing.T) {
	manager, lifecycle, detail := mixedRoomFixture(t, false)
	runID := detail.Run.RunID
	finishOrdinaryStage(t, manager, runID, "a_owner", map[string]string{"options": "A or B"})

	room, found, err := manager.currentStageTask(runID)
	if err != nil || !found || room.ExecutionKind != state.StageExecutionThinkTank || room.StageID != "debate" {
		t.Fatalf("room stage = %+v err=%v", room, err)
	}
	entries, err := manager.store.ListThinkTankEntries(room.RoomID, 0, 10)
	if err != nil || len(entries) == 0 || entries[0].Kind != state.ThinkTankEntryStageContext || !strings.Contains(entries[0].Body, "A or B") {
		t.Fatalf("stage context entry = %+v err=%v", entries, err)
	}

	// Idle, discussion and an unfinalized judge never advance.
	if _, err := manager.store.DB().Exec(`UPDATE think_tanks SET judge_status = 'running', phase = 'ended' WHERE room_id = ?`, room.RoomID); err != nil {
		t.Fatal(err)
	}
	manager.AcceptRoomOutputs(context.Background())
	if run, _ := manager.store.ReadPipelineRun(runID); run.State != "running" || run.CurrentStageID != "debate" {
		t.Fatalf("run advanced before publication: %+v", run)
	}

	synthesis := "Choose B.\n\nDissent: pro preferred A for speed."
	publishSynthesis(t, manager, room.RoomID, synthesis)
	if err := manager.AcceptRoomOutput(context.Background(), runID); err != nil {
		t.Fatal(err)
	}
	manager.AcceptRoomOutputs(context.Background()) // a replayed kick is a no-op

	task, err := manager.store.ReadTask(room.TaskID)
	if err != nil || task.State != state.TaskFinished || task.Outcome != state.OutcomeSuccess || task.OutcomeSource != state.StageExecutionThinkTank || task.Outputs["decision"] != synthesis {
		t.Fatalf("room task = %+v err=%v", task, err)
	}
	// The accepted stage task itself is republished, or an open Tasks view keeps
	// its running row (TS-10.R11, CR-A5-1).
	published := false
	for _, update := range manager.publisher.(*fakePublisher).taskUpdates {
		if update.TaskID == room.TaskID && update.State == state.TaskFinished &&
			update.Outcome == state.OutcomeSuccess && update.Revision == task.Revision {
			published = true
		}
	}
	if !published {
		t.Fatalf("accepted room stage task was not republished: %+v", manager.publisher.(*fakePublisher).taskUpdates)
	}
	values, _ := manager.store.ListPipelineValues(runID)
	count := 0
	for _, v := range values {
		if v.Name == "decision" {
			count++
			if v.Value != synthesis {
				t.Fatalf("decision = %q", v.Value)
			}
		}
	}
	if count != 1 {
		t.Fatalf("decision values = %d", count)
	}
	// The standing owner decides what follows the synthesis in a decision-only
	// visit, then its usual next step reuses the same owner (TS-09.R66).
	stages, _ := manager.store.ListPipelineStageTasks(runID)
	if len(stages) != 3 || stages[2].StageID != "debate" || stages[2].DecisionSourceTaskID != room.TaskID || stages[2].RoomID != "" {
		t.Fatalf("stages = %+v", stages)
	}
	decide, _ := manager.store.ReadTask(stages[2].TaskID)
	if decide.TargetKind != state.TargetAgent || decide.TargetAgentID != "a_owner" || !strings.Contains(decide.Instruction, "decision-only") {
		t.Fatalf("decision visit did not go to the standing owner: %+v", decide)
	}
	finishOrdinaryStage(t, manager, runID, "a_owner", nil)
	stages, _ = manager.store.ListPipelineStageTasks(runID)
	if len(stages) != 4 || stages[3].StageID != "build" {
		t.Fatalf("stages after decision = %+v", stages)
	}
	build, _ := manager.store.ReadTask(stages[3].TaskID)
	if build.TargetKind != state.TargetAgent || build.TargetAgentID != "a_owner" {
		t.Fatalf("build stage did not reuse the standing owner: %+v", build)
	}
	// The activation froze which accepted result its input came from.
	if refs := stages[3].InputRefs; len(refs) != 1 || refs[0].Value != "decision" || refs[0].SourceTaskID != room.TaskID || refs[0].SourceRoomID != room.RoomID {
		t.Fatalf("build input references = %+v", refs)
	}
	if len(lifecycle.launches) != 0 {
		t.Fatalf("pipeline launched directly: %+v", lifecycle.launches)
	}
}

// FS-14.A48 — a human gate still waits after accepted synthesis.
func TestRoomStageApprovalWaitsAfterAcceptance(t *testing.T) {
	manager, _, detail := mixedRoomFixture(t, true)
	runID := detail.Run.RunID
	finishOrdinaryStage(t, manager, runID, "a_owner", map[string]string{"options": "A or B"})
	room, _, _ := manager.currentStageTask(runID)
	publishSynthesis(t, manager, room.RoomID, "Choose B.")
	if err := manager.AcceptRoomOutput(context.Background(), runID); err != nil {
		t.Fatal(err)
	}
	run, _ := manager.store.ReadPipelineRun(runID)
	if run.State != "paused" || run.PendingAction != "await_approval" {
		t.Fatalf("run after accepted synthesis = %+v", run)
	}
	// Approval then hands the synthesis to the owner's decision visit.
	if _, err := manager.Continue(context.Background(), runID, run.Revision, ""); err != nil {
		t.Fatal(err)
	}
	if visit, _, _ := manager.currentStageTask(runID); visit.DecisionSourceTaskID != room.TaskID {
		t.Fatalf("visit after approval = %+v, want the owner's decision about %s", visit, room.TaskID)
	}
}

// FS-14.A52 — refused output pauses with Retry output acceptance; retry reuses
// the same entry and accepts no partial result; a stopped run accepts nothing.
func TestRoomOutputRefusalAndStopFence(t *testing.T) {
	manager, _, detail := mixedRoomFixture(t, false)
	runID := detail.Run.RunID
	finishOrdinaryStage(t, manager, runID, "a_owner", map[string]string{"options": "A or B"})
	room, _, _ := manager.currentStageTask(runID)
	publishSynthesis(t, manager, room.RoomID, strings.Repeat("x", MaxValueRunes+1))
	if err := manager.AcceptRoomOutput(context.Background(), runID); err != nil {
		t.Fatal(err)
	}
	run, _ := manager.store.ReadPipelineRun(runID)
	if run.State != "paused" || run.PendingAction != pendingAcceptRoomOutput || !strings.Contains(run.AttentionReason, "decision") {
		t.Fatalf("refused run = %+v", run)
	}
	// The automatic sweep leaves the held acceptance to the explicit retry.
	manager.AcceptRoomOutputs(context.Background())
	if task, _ := manager.store.ReadTask(room.TaskID); task.State == state.TaskFinished {
		t.Fatalf("held acceptance finished the task: %+v", task)
	}
	// Retry still refuses the same oversized entry without partial output.
	if _, err := manager.Retry(context.Background(), runID, run.Revision); err != nil {
		t.Fatal(err)
	}
	if values, _ := manager.store.ListPipelineValues(runID); hasValue(values, "decision") {
		t.Fatalf("partial output installed: %+v", values)
	}

	// A stopped run cannot accept even a previously published synthesis.
	if _, err := manager.store.DB().Exec(`UPDATE pipeline_runs SET state = 'stopping', pending_action = 'cleanup_run' WHERE run_id = ?`, runID); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.store.DB().Exec(`UPDATE think_tank_entries SET body = 'short' WHERE room_id = ? AND kind = 'synthesis'`, room.RoomID); err != nil {
		t.Fatal(err)
	}
	manager.AcceptRoomOutputs(context.Background())
	if values, _ := manager.store.ListPipelineValues(runID); hasValue(values, "decision") {
		t.Fatalf("stopped run accepted output: %+v", values)
	}
}

// FS-14.A49 / TS-09.R53 — acceptance intent and its automatic budget live on
// the stage authority. Recreating the manager after injected write failures
// still permits exactly three automatic effects, while explicit Retry reuses
// the same entry.
func TestRoomOutputAcceptanceBudgetSurvivesManagerRestart(t *testing.T) {
	manager, lifecycle, detail := mixedRoomFixture(t, false)
	runID := detail.Run.RunID
	finishOrdinaryStage(t, manager, runID, "a_owner", map[string]string{"options": "A or B"})
	room, _, _ := manager.currentStageTask(runID)
	publishSynthesis(t, manager, room.RoomID, "Choose B.")

	trigger := "test_fail_room_output_acceptance"
	_, err := manager.store.DB().Exec(`CREATE TRIGGER ` + trigger + ` BEFORE UPDATE OF state ON tasks
WHEN NEW.task_id = '` + room.TaskID + `' AND NEW.state = 'finished'
BEGIN SELECT RAISE(ABORT, 'injected acceptance failure'); END`)
	if err != nil {
		t.Fatal(err)
	}
	for attempt := 1; attempt <= maxRoomOutputAttempts; attempt++ {
		manager = NewManager(manager.store, manager.templates, lifecycle, manager.publisher)
		err := manager.AcceptRoomOutput(context.Background(), runID)
		if attempt < maxRoomOutputAttempts && err == nil {
			t.Fatalf("attempt %d unexpectedly succeeded", attempt)
		}
	}
	if _, err := manager.store.DB().Exec(`DROP TRIGGER ` + trigger); err != nil {
		t.Fatal(err)
	}
	// The third failed effect leaves the durable hold; a recreated manager's
	// automatic sweep must not prepare a fourth attempt.
	manager = NewManager(manager.store, manager.templates, lifecycle, manager.publisher)
	manager.AcceptRoomOutputs(context.Background())
	run, err := manager.store.ReadPipelineRun(runID)
	if err != nil || run.State != "paused" || run.PendingAction != pendingAcceptRoomOutput {
		t.Fatalf("exhausted acceptance = %+v, err=%v", run, err)
	}
	if _, err := manager.Retry(context.Background(), runID, run.Revision); err != nil {
		t.Fatal(err)
	}
	accepted, err := manager.store.ReadTask(room.TaskID)
	if err != nil || accepted.State != state.TaskFinished || accepted.Outputs["decision"] != "Choose B." {
		t.Fatalf("explicit acceptance = %+v, err=%v", accepted, err)
	}
}

func hasValue(values []state.PipelineValueRecord, name string) bool {
	for _, v := range values {
		if v.Name == name {
			return true
		}
	}
	return false
}

// FS-14.A49 / FS-21.A32 — Stop fences every later room claim and message; the
// retained run pins its room; deleting the terminal run releases the pin.
func TestRoomStageStopFencesAndPins(t *testing.T) {
	manager, _, detail := mixedRoomFixture(t, false)
	runID := detail.Run.RunID
	finishOrdinaryStage(t, manager, runID, "a_owner", map[string]string{"options": "A or B"})
	room, _, _ := manager.currentStageTask(runID)
	roomDetail, err := manager.store.ReadThinkTank(room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	run, _ := manager.store.ReadPipelineRun(runID)
	if _, err := manager.Stop(context.Background(), runID, run.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.store.ClaimThinkTankMemberSetup(room.RoomID, roomDetail.Members[0].AgentID); !errors.Is(err, state.ErrThinkTankConflict) {
		t.Fatalf("setup claim after Stop = %v", err)
	}
	if _, _, err := manager.store.AddThinkTankMessage(room.RoomID, "cmd-1", "hello", nil); !errors.Is(err, state.ErrThinkTankConflict) {
		t.Fatalf("message after Stop = %v", err)
	}
	if err := manager.store.DeleteThinkTank(room.RoomID); !errors.Is(err, state.ErrThinkTankConflict) {
		t.Fatalf("delete pinned room = %v", err)
	}
	if err := manager.Reconcile(context.Background(), runID); err != nil {
		t.Fatal(err)
	}
	stopped, _ := manager.store.ReadPipelineRun(runID)
	if stopped.State != "stopped" {
		t.Fatalf("run after cleanup = %+v", stopped)
	}
	task, _ := manager.store.ReadTask(room.TaskID)
	if task.State != state.TaskFinished || task.Outcome != state.OutcomeCancelled {
		t.Fatalf("room task after Stop = %+v", task)
	}
	if err := manager.store.DeletePipelineRun(runID); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.store.ClosePipelineThinkTank(room.RoomID); err != nil {
		t.Fatal(err)
	}
	if err := manager.store.DeleteThinkTank(room.RoomID); err != nil {
		t.Fatalf("delete after run deletion = %v", err)
	}
}

// TS-09.R54 — cleanup_run replays room closure after a failed first effect or
// a crash immediately after the stop fence, and waits for claimed setup.
func TestRoomStopRecoversCloseAndWaitsForClaimedSetup(t *testing.T) {
	manager, lifecycle, detail := mixedRoomFixture(t, false)
	runID := detail.Run.RunID
	finishOrdinaryStage(t, manager, runID, "a_owner", map[string]string{"options": "A or B"})
	room, _, _ := manager.currentStageTask(runID)
	var participant string
	d, err := manager.store.ReadThinkTank(room.RoomID)
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range d.Members {
		if member.SetupConfig != "" {
			participant = member.AgentID
			break
		}
	}
	if participant == "" {
		t.Fatal("room has no new participant")
	}
	if _, err := manager.store.DB().Exec(`UPDATE think_tank_members SET setup_state = 'launching', launch_generation = 'old' WHERE room_id = ? AND agent_id = ?`, room.RoomID, participant); err != nil {
		t.Fatal(err)
	}
	lifecycle.roomStopErrors = 1
	run, err := manager.store.ReadPipelineRun(runID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Stop(context.Background(), runID, run.Revision); err != nil {
		t.Fatal(err)
	}
	run, _ = manager.store.ReadPipelineRun(runID)
	if run.State != "stopping" || run.PendingAction != "cleanup_run" || len(lifecycle.roomStops) != 1 {
		t.Fatalf("failed close dropped stop intent: run=%+v calls=%v", run, lifecycle.roomStops)
	}
	var teardown bool
	if err := manager.store.DB().QueryRow(`SELECT stop_teardown FROM think_tank_members WHERE room_id = ? AND agent_id = ?`, room.RoomID, participant).Scan(&teardown); err != nil || !teardown {
		t.Fatalf("claimed launch not captured by Stop: flag=%v err=%v", teardown, err)
	}
	if err := manager.Reconcile(context.Background(), runID); err != nil {
		t.Fatal(err)
	}
	run, _ = manager.store.ReadPipelineRun(runID)
	if run.State != "stopping" || len(lifecycle.roomStops) != 2 {
		t.Fatalf("setup claim did not hold stop: run=%+v calls=%v", run, lifecycle.roomStops)
	}
	if _, err := manager.store.DB().Exec(`UPDATE think_tank_members SET setup_state = 'ready', stop_teardown = 0 WHERE room_id = ? AND agent_id = ?`, room.RoomID, participant); err != nil {
		t.Fatal(err)
	}
	if err := manager.Reconcile(context.Background(), runID); err != nil {
		t.Fatal(err)
	}
	run, _ = manager.store.ReadPipelineRun(runID)
	if run.State != "stopped" {
		t.Fatalf("settled room did not stop: %+v", run)
	}
}

func TestRoomStopReplaysCloseAfterCrashAtFence(t *testing.T) {
	manager, lifecycle, detail := mixedRoomFixture(t, false)
	runID := detail.Run.RunID
	finishOrdinaryStage(t, manager, runID, "a_owner", map[string]string{"options": "A or B"})
	run, err := manager.store.ReadPipelineRun(runID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.store.UpdatePipelineRunCAS(runID, run.Revision, state.PipelineRunUpdate{
		State: "stopping", PendingAction: "cleanup_run", CurrentStageID: run.CurrentStageID, MarkRoomTeardown: true,
	}); err != nil {
		t.Fatal(err)
	}
	// There is no StopRoom call here: the server crashed just after the fence.
	if err := manager.Reconcile(context.Background(), runID); err != nil {
		t.Fatal(err)
	}
	stopped, err := manager.store.ReadPipelineRun(runID)
	if err != nil || stopped.State != "stopped" || len(lifecycle.roomStops) != 1 {
		t.Fatalf("room close not recovered: run=%+v calls=%v err=%v", stopped, lifecycle.roomStops, err)
	}
}

func TestRoomStopRetainsFailedLaunchGenerationForCleanup(t *testing.T) {
	for _, kind := range []string{"participant", "judge"} {
		t.Run(kind, func(t *testing.T) {
			manager, _, detail := mixedRoomFixture(t, false)
			finishOrdinaryStage(t, manager, detail.Run.RunID, "a_owner", map[string]string{"options": "A or B"})
			room, _, _ := manager.currentStageTask(detail.Run.RunID)
			d, err := manager.store.ReadThinkTank(room.RoomID)
			if err != nil {
				t.Fatal(err)
			}
			agent := "a_failed_judge"
			if kind == "participant" {
				for _, m := range d.Members {
					if m.SetupConfig != "" {
						agent = m.AgentID
						break
					}
				}
				if _, err := manager.store.ClaimThinkTankMemberSetup(room.RoomID, agent, "claimed_gen"); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := manager.store.DB().Exec(`UPDATE think_tanks SET phase = 'ended', judge_status = 'ready' WHERE room_id = ?`, room.RoomID); err != nil {
					t.Fatal(err)
				}
				if _, err := manager.store.ReserveThinkTankJudge(room.RoomID, agent, "Judge", "app", "claimed_gen"); err != nil {
					t.Fatal(err)
				}
			}
			run, _ := manager.store.ReadPipelineRun(detail.Run.RunID)
			if _, err := manager.Stop(context.Background(), run.RunID, run.Revision); err != nil {
				t.Fatal(err)
			}
			if kind == "participant" {
				if _, err := manager.store.MarkThinkTankMemberSetup(room.RoomID, agent, "", "launch failed"); err != nil {
					t.Fatal(err)
				}
			} else if _, err := manager.store.MarkThinkTankJudgeLaunched(room.RoomID, "", "launch failed"); err != nil {
				t.Fatal(err)
			}
			var generation string
			var teardown bool
			if err := manager.store.DB().QueryRow(`SELECT launch_generation, stop_teardown FROM think_tank_members WHERE room_id = ? AND agent_id = ?`, room.RoomID, agent).Scan(&generation, &teardown); err != nil || generation != "claimed_gen" || !teardown {
				t.Fatalf("failed %s lost stop obligation: gen=%q teardown=%v err=%v", kind, generation, teardown, err)
			}
		})
	}
}

// TS-09.R55 — generic stage controls refuse a room-backed stage; a room hold
// pauses the run with the room's reason and recovery resumes it.
func TestRoomStageControlsAndPhaseProjection(t *testing.T) {
	manager, _, detail := mixedRoomFixture(t, false)
	runID := detail.Run.RunID
	finishOrdinaryStage(t, manager, runID, "a_owner", map[string]string{"options": "A or B"})
	room, _, _ := manager.currentStageTask(runID)
	if _, err := manager.store.SetThinkTankHold(room.RoomID, "Setup failed for Debate · pro."); err != nil {
		t.Fatal(err)
	}
	if err := manager.SyncRoomPhase(context.Background(), runID); err != nil {
		t.Fatal(err)
	}
	run, _ := manager.store.ReadPipelineRun(runID)
	if run.State != "paused" || !strings.Contains(run.AttentionReason, "Setup failed") {
		t.Fatalf("held room run = %+v", run)
	}
	for name, call := range map[string]func() error{
		"continue": func() error { _, err := manager.Continue(context.Background(), runID, run.Revision, "go"); return err },
		"retry":    func() error { _, err := manager.Retry(context.Background(), runID, run.Revision); return err },
		"replace": func() error {
			_, err := manager.Replace(context.Background(), runID, run.Revision, RuntimeAssignment{Backend: "claude", Model: "sonnet"})
			return err
		},
	} {
		var controlled *ControlError
		if err := call(); !errors.As(err, &controlled) || controlled.Code != "room_recovery_required" {
			t.Errorf("%s on room stage = %v", name, err)
		}
	}
	if _, err := manager.store.DB().Exec(`UPDATE think_tanks SET hold = '' WHERE room_id = ?`, room.RoomID); err != nil {
		t.Fatal(err)
	}
	if err := manager.SyncRoomPhase(context.Background(), runID); err != nil {
		t.Fatal(err)
	}
	if run, _ := manager.store.ReadPipelineRun(runID); run.State != "running" || run.AttentionReason != "" {
		t.Fatalf("recovered room run = %+v", run)
	}
}

// FS-14.R93/A58: the first ordinary owner after a room-first stage launches
// into the run's frozen group, as the room's own participants do.
func TestRoomFirstStandingOwnerReceivesRunGroup(t *testing.T) {
	m, _, mixed := mixedRoomFixture(t, false)
	template := mixed.Template
	template.Stages = template.Stages[1:]
	template.Stages[0].Inputs = []StageInput{{Name: "spec", Value: "spec", Required: true}}
	if record, err := m.templates.Create("room-first", template); err != nil || !record.Valid {
		t.Fatalf("template = %+v, err=%v", record, err)
	}
	slot := RuntimeAssignment{Backend: "claude", Model: "sonnet"}
	d, _, err := m.Start(context.Background(), StartRequest{
		RequestID: "room-first", TemplateID: "room-first", DisplayName: "Room First", Project: "app", Goal: "decide",
		Inputs: map[string]string{"spec": "options"}, Orchestrator: slot,
		ThinkTankAssignments: map[string]ThinkTankAssignment{"debate": {Participants: map[string]RuntimeAssignment{"pro": slot, "con": slot}, Judge: slot}},
	})
	if err != nil {
		t.Fatal(err)
	}
	room, found, err := m.currentStageTask(d.Run.RunID)
	if err != nil || !found {
		t.Fatalf("current stage: found=%v err=%v", found, err)
	}
	publishSynthesis(t, m, room.RoomID, "Build the chosen option.")
	if err := m.AcceptRoomOutput(context.Background(), d.Run.RunID); err != nil {
		t.Fatal(err)
	}
	stages, err := m.store.ListPipelineStageTasks(d.Run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	task, err := m.store.ReadTask(stages[len(stages)-1].TaskID)
	if err != nil || task.TargetKind != state.TargetLaunch {
		t.Fatalf("standing task = %+v err=%v", task, err)
	}
	if task.Group != "Room First" {
		t.Fatalf("first standing launch group = %q, want the frozen run name", task.Group)
	}
}
