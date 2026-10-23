package server

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/recurrence"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// taskWireFixturePath is the GET /api/tasks payload the Tasks view tests read.
// Go's encoder, not the frontend schema, defines its shape (INV §11, §17).
const taskWireFixturePath = "../../ui/src/features/tasks/fixtures/taskLists.json"

// TestTaskWireFixtureMatchesServerEncoding keeps the UI fixture byte-identical
// to what handleTasks would marshal for these tasks. Regenerate with
// CHUCK_UPDATE_TASK_FIXTURE=1 after a deliberate change to state.Task.
func TestTaskWireFixtureMatchesServerEncoding(t *testing.T) {
	got, err := json.MarshalIndent(taskWireFixture(), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	if os.Getenv("CHUCK_UPDATE_TASK_FIXTURE") == "1" {
		if err := os.WriteFile(taskWireFixturePath, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(taskWireFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s is stale; regenerate with CHUCK_UPDATE_TASK_FIXTURE=1", taskWireFixturePath)
	}
}

// taskWireFixture covers FS-16.A27–A29: chains, branch/join, delegation, an
// unrelated task by the same creator, a missing predecessor, signal and run
// waits, pipeline-owned work, durable waiting, interruption, pending cleanup
// and a settled history group, across two projects.
func taskWireFixture() map[string]map[string][]taskDetailResponse {
	base := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	at := func(minutes int) time.Time { return base.Add(time.Duration(minutes) * time.Minute) }
	ptr := func(value time.Time) *time.Time { return &value }
	task := func(id, project, name string, minute int, mutate func(*state.Task)) taskDetailResponse {
		value := state.Task{
			TaskID: id, Project: project, DisplayName: name, Instruction: "Instruction for " + name + ".",
			TargetKind: state.TargetLaunch, Role: "implementer", State: state.TaskArmed,
			CreatedByKind: "agent", CreatedByAgentID: "ag_lead", Revision: 1,
			CreatedAt: at(minute), UpdatedAt: at(minute + 1), Arms: []state.TaskArm{},
		}
		mutate(&value)
		return taskDetailResponse{Task: value, Attachments: []state.TaskAttachment{}}
	}
	finished := func(outcome string, minute int) func(*state.Task) {
		return func(value *state.Task) {
			value.State, value.Outcome, value.OutcomeSource = state.TaskFinished, outcome, "agent"
			value.OutcomeSummary = "Reported " + outcome + "."
			value.StartedAt, value.FinishedAt = ptr(at(minute)), ptr(at(minute+5))
		}
	}
	arm := func(taskID string, index int, sourceID, armState string) state.TaskArm {
		return state.TaskArm{
			ArmID: taskID + "_arm0" + string(rune('0'+index)), TaskID: taskID, Kind: state.ArmWorkResult,
			SourceKind: state.SourceTask, SourceID: sourceID, SatisfyingOutcomes: []string{state.OutcomeSuccess}, State: armState,
		}
	}
	lineage := func(taskID, parent, run, stage string, minute int) state.TaskLineage {
		return state.TaskLineage{TaskID: taskID, ParentTaskID: parent, PipelineRunID: run, PipelineStageID: stage, CreatedAt: at(minute)}
	}
	all := []taskDetailResponse{
		task("tk_a", "my-app", "Design schema", 0, func(value *state.Task) {
			finished(state.OutcomeSuccess, 1)(value)
			value.AssignedAgentID, value.Outputs = "ag_a", map[string]string{"schema": "docs/schema.md"}
		}),
		task("tk_b", "my-app", "Implement API", 2, func(value *state.Task) {
			value.State, value.AssignedAgentID, value.StartedAt = state.TaskRunning, "ag_b", ptr(at(8))
			value.Arms = []state.TaskArm{arm("tk_b", 0, "tk_a", state.ArmSatisfied)}
		}),
		task("tk_c", "my-app", "Write API docs", 3, func(value *state.Task) {
			value.Arms = []state.TaskArm{arm("tk_c", 0, "tk_b", state.ArmUnsatisfied)}
		}),
		task("tk_d", "my-app", "Split data", 4, finished(state.OutcomeSuccess, 5)),
		task("tk_e", "my-app", "Migrate users", 5, func(value *state.Task) {
			value.State, value.ReadyAt = state.TaskReady, ptr(at(10))
			value.Arms = []state.TaskArm{arm("tk_e", 0, "tk_d", state.ArmSatisfied)}
		}),
		task("tk_f", "my-app", "Migrate orders", 6, func(value *state.Task) {
			value.State, value.AssignedAgentID, value.WaitVersion = state.TaskWaiting, "ag_f", 1
			value.Arms = []state.TaskArm{arm("tk_f", 0, "tk_d", state.ArmSatisfied)}
		}),
		task("tk_g", "my-app", "Verify migration", 7, func(value *state.Task) {
			value.Arms = []state.TaskArm{arm("tk_g", 0, "tk_e", state.ArmUnsatisfied), arm("tk_g", 1, "tk_f", state.ArmUnsatisfied)}
		}),
		task("tk_p", "my-app", "Coordinate release", 8, func(value *state.Task) {
			value.State, value.TargetKind, value.TargetAgentID, value.AssignedAgentID = state.TaskRunning, state.TargetAgent, "ag_lead", "ag_lead"
			value.Role, value.CreatedByKind, value.CreatedByAgentID = "", "person", ""
		}),
		task("tk_q", "my-app", "Draft release notes", 9, func(value *state.Task) {
			value.State, value.AssignedAgentID, value.RetryEligible = state.TaskInterrupted, "ag_gone", true
			value.AttentionReason = "the assigned agent stopped before reporting a result"
			value.Lineage = lineage("tk_q", "tk_p", "", "", 9)
		}),
		task("tk_r", "my-app", "Tidy fixtures", 10, func(value *state.Task) { value.State = state.TaskReady }),
		task("tk_s", "my-app", "Follow up on removed work", 11, func(value *state.Task) {
			value.State, value.AttentionReason = state.TaskDependencyFailed, "a prerequisite can no longer be satisfied"
			value.Arms = []state.TaskArm{arm("tk_s", 0, "tk_gone", state.ArmUnsatisfiable)}
		}),
		task("tk_t", "my-app", "Ship release", 12, func(value *state.Task) {
			value.Arms = []state.TaskArm{
				{ArmID: "tk_t_arm00", TaskID: "tk_t", Kind: state.ArmSignal, SignalName: "ci-green", State: state.ArmUnsatisfied},
				{ArmID: "tk_t_arm01", TaskID: "tk_t", Kind: state.ArmWorkResult, SourceKind: state.SourcePipelineRun, SourceID: "pr_1",
					SatisfyingOutcomes: []string{state.OutcomeSuccess}, State: state.ArmUnsatisfied},
			}
		}),
		task("tk_v", "my-app", "Start at half five", 12, func(value *state.Task) {
			value.Arms = []state.TaskArm{{ArmID: "tk_v_arm00", TaskID: "tk_v", Kind: state.ArmTime,
				DueAt: ptr(time.Date(2026, 10, 1, 17, 30, 0, 0, time.UTC)), OnTimeOnly: true, State: state.ArmUnsatisfied}}
		}),
		task("tk_u", "my-app", "Build stage", 13, func(value *state.Task) {
			value.State, value.AssignedAgentID = state.TaskRunning, "ag_u"
			value.Lineage = lineage("tk_u", "", "pr_1", "build", 13)
		}),
		task("tk_k", "my-app", "Publish notes", 14, func(value *state.Task) {
			finished(state.OutcomeSuccess, 15)(value)
			value.AssignedAgentID, value.PendingRelease = "ag_k", true
			value.CleanupPhase, value.CleanupFailureCount = "release", 2
			value.CleanupLastError, value.CleanupNextRetryAt = "agent did not stop", ptr(at(30))
		}),
		task("tk_h1", "my-app", "Old spike", 15, finished(state.OutcomeFailure, 16)),
		task("tk_h2", "my-app", "Old follow-up", 16, func(value *state.Task) {
			value.State, value.Outcome, value.OutcomeSource, value.FinishedAt = state.TaskFinished, state.OutcomeCancelled, "host", ptr(at(20))
			value.Arms = []state.TaskArm{arm("tk_h2", 0, "tk_h1", state.ArmUnsatisfiable)}
		}),
		task("tk_o1", "other", "Audit logs", 20, finished(state.OutcomeBlocked, 21)),
		task("tk_o2", "other", "Rotate keys", 22, func(value *state.Task) {
			value.State, value.AssignedAgentID, value.CreatedByAgentID = state.TaskRunning, "ag_o", "ag_other"
		}),
	}
	out := map[string]map[string][]taskDetailResponse{}
	// handleTasks lists newest first (state.ListTasks).
	for index := len(all) - 1; index >= 0; index-- {
		project := all[index].Project
		if out[project] == nil {
			out[project] = map[string][]taskDetailResponse{"tasks": {}}
		}
		out[project]["tasks"] = append(out[project]["tasks"], all[index])
	}
	return out
}

// scheduleWireFixturePath is the GET /api/schedules payload the Schedules
// section tests read, encoded by Go like the task fixture (INV §11, §17).
const scheduleWireFixturePath = "../../ui/src/features/tasks/fixtures/scheduleLists.json"

// TestScheduleWireFixtureMatchesServerEncoding keeps the schedule fixture
// identical to what handleSchedules would marshal. Regenerate with
// CHUCK_UPDATE_TASK_FIXTURE=1 after a deliberate change to state.TaskSchedule.
func TestScheduleWireFixtureMatchesServerEncoding(t *testing.T) {
	got, err := json.MarshalIndent(map[string]any{"schedules": scheduleWireFixture()}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	if os.Getenv("CHUCK_UPDATE_TASK_FIXTURE") == "1" {
		if err := os.WriteFile(scheduleWireFixturePath, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(scheduleWireFixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s is stale; regenerate with CHUCK_UPDATE_TASK_FIXTURE=1", scheduleWireFixturePath)
	}
}

// scheduleWireFixture covers FS-16.A40's states: an active weekly schedule
// whose last occurrence is a running task, one that paused itself because its
// target was archived, and a finished one whose last occurrence was missed.
func scheduleWireFixture() []state.TaskSchedule {
	base := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	ptr := func(value time.Time) *time.Time { return &value }
	schedule := func(id, name string, rule recurrence.Rule, mutate func(*state.TaskSchedule)) state.TaskSchedule {
		value := state.TaskSchedule{
			ScheduleID: id, Project: "my-app", DisplayName: name, Instruction: "Instruction for " + name + ".",
			TargetKind: state.TargetLaunch, Role: "implementer", Rule: rule, RuleText: rule.Describe(),
			OnTimeOnly: true, State: state.ScheduleActive, CreatedByKind: "person", Revision: 3,
			CreatedAt: base, UpdatedAt: base, Attachments: []state.TaskAttachment{},
		}
		mutate(&value)
		return value
	}
	return []state.TaskSchedule{
		schedule("sc_weekly", "Weekly review", recurrence.Rule{StartDate: "2026-10-04", Times: []string{"09:00"},
			Frequency: recurrence.FreqWeekly, Weekdays: []string{"sun"}, EndKind: recurrence.EndCount, EndCount: 3},
			func(value *state.TaskSchedule) {
				value.NextDueAt, value.LastOccurrenceAt = ptr(base.AddDate(0, 0, 10)), ptr(base.AddDate(0, 0, 3))
				value.OccurrencesElapsed, value.LastDisposition, value.LastTaskID = 1, state.OccurrenceCreated, "tk_u"
			}),
		schedule("sc_paused", "Daily standup notes", recurrence.Rule{StartDate: "2026-10-01", Times: []string{"09:00", "17:30"},
			Frequency: recurrence.FreqWeekdays, EndKind: recurrence.EndNever},
			func(value *state.TaskSchedule) {
				value.TargetKind, value.TargetAgentID, value.Role = state.TargetAgent, "ag_gone", ""
				value.State, value.PauseReason, value.OnTimeOnly = state.SchedulePaused, "the target agent is archived", false
				value.NextDueAt, value.LastOccurrenceAt = ptr(base.AddDate(0, 0, 2)), ptr(base.AddDate(0, 0, 1))
				value.OccurrencesElapsed, value.LastDisposition = 4, state.OccurrencePaused
				value.Attachments = []state.TaskAttachment{{ContextRefID: "cx_notes", Label: "Notes"}}
			}),
		schedule("sc_done", "Quarter close", recurrence.Rule{StartDate: "2026-09-01", Times: []string{"08:00"},
			Frequency: recurrence.FreqEveryNDays, Interval: 10, EndKind: recurrence.EndUntil, EndDate: "2026-09-30"},
			func(value *state.TaskSchedule) {
				value.State, value.LastOccurrenceAt = state.ScheduleFinished, ptr(base.AddDate(0, 0, -2))
				value.OccurrencesElapsed, value.LastDisposition = 3, state.OccurrenceMissed
			}),
	}
}
