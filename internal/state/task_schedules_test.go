package state

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/recurrence"
)

func newYork(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load zone: %v", err)
	}
	return loc
}

func newSchedule(t *testing.T, st *Store, loc *time.Location, rule recurrence.Rule, onTimeOnly bool) TaskSchedule {
	t.Helper()
	id, err := NewScheduleID()
	if err != nil {
		t.Fatal(err)
	}
	sched, err := st.CreateSchedule(TaskSchedule{
		ScheduleID: id, Project: "p", DisplayName: "standup notes", Instruction: "write the notes",
		TargetKind: TargetLaunch, Role: "impl", Rule: rule, OnTimeOnly: onTimeOnly,
		CreatedByKind: "agent", CreatedByAgentID: "a_lead", CreatedByGeneration: "g1",
	}, "", loc)
	if err != nil {
		t.Fatalf("CreateSchedule: %v", err)
	}
	return sched
}

// processAt runs one due pass with the clock at the given local wall time.
func processAt(t *testing.T, st *Store, loc *time.Location, wall string, unavailable func(TaskSchedule) string) []ScheduleOutcome {
	t.Helper()
	at, err := time.ParseInLocation("2006-01-02 15:04:05", wall, loc)
	if err != nil {
		t.Fatal(err)
	}
	withFixedNow(t, at)
	if unavailable == nil {
		unavailable = func(TaskSchedule) string { return "" }
	}
	outcomes, err := st.ProcessDueSchedules(8, loc, unavailable)
	if err != nil {
		t.Fatalf("ProcessDueSchedules at %s: %v", wall, err)
	}
	return outcomes
}

func setNow(t *testing.T, loc *time.Location, wall string) {
	t.Helper()
	at, err := time.ParseInLocation("2006-01-02 15:04:05", wall, loc)
	if err != nil {
		t.Fatal(err)
	}
	withFixedNow(t, at)
}

// FS-16.A37 — "every Sunday at 9 AM for 3 weeks" creates an ordinary, ready
// task on exactly three Sundays, each carrying its schedule, occurrence and the
// schedule's creator, and then finishes.
func TestWeeklyScheduleCreatesExactlyItsOccurrencesThenFinishes(t *testing.T) {
	loc := newYork(t)
	setNow(t, loc, "2026-10-30 12:00:00")
	st, _ := newTestStore(t)
	sched := newSchedule(t, st, loc, recurrence.Rule{
		StartDate: "2026-11-01", Times: []string{"09:00"}, Frequency: recurrence.FreqWeekly,
		Weekdays: []string{"sun"}, EndKind: recurrence.EndCount, EndCount: 3,
	}, true)
	if sched.RuleText == "" || sched.NextDueAt == nil {
		t.Fatalf("created schedule = %+v", sched)
	}

	if got := processAt(t, st, loc, "2026-11-01 08:59:59", nil); len(got) != 0 {
		t.Fatalf("processed before due: %+v", got)
	}
	var created []Task
	// The middle Sunday follows the fall-back change and is still 9 AM local.
	for _, wall := range []string{"2026-11-01 09:00:01", "2026-11-08 09:00:02", "2026-11-15 09:04:00"} {
		got := processAt(t, st, loc, wall, nil)
		if len(got) != 1 || got[0].Task == nil {
			t.Fatalf("at %s: %+v", wall, got)
		}
		created = append(created, *got[0].Task)
		if again := processAt(t, st, loc, wall, nil); len(again) != 0 {
			t.Fatalf("occurrence handled twice at %s", wall)
		}
	}
	for i, task := range created {
		want := time.Date(2026, 11, 1+7*i, 9, 0, 0, 0, loc)
		if task.State != TaskReady || task.ScheduleID != sched.ScheduleID || task.LateStart ||
			task.ScheduledFor == nil || !task.ScheduledFor.Equal(want) ||
			task.CreatedByKind != "agent" || task.CreatedByAgentID != "a_lead" {
			t.Fatalf("occurrence %d = %+v", i, task)
		}
	}
	// A38 — earlier occurrences are still unfinished; nothing waited for them.
	if first, _ := st.ReadTask(created[0].TaskID); first.State != TaskReady {
		t.Fatalf("first occurrence = %s", first.State)
	}
	final, _ := st.ReadSchedule(sched.ScheduleID)
	if final.State != ScheduleFinished || final.OccurrencesElapsed != 3 || final.NextDueAt != nil ||
		final.LastTaskID != created[2].TaskID || final.LastDisposition != OccurrenceCreated {
		t.Fatalf("final schedule = %+v", final)
	}
	if got := processAt(t, st, loc, "2026-11-22 09:00:01", nil); len(got) != 0 {
		t.Fatalf("finished schedule ran: %+v", got)
	}
}

// FS-16.A36 (schedule part) — a daily schedule observed ten minutes after its
// third missed occurrence: with Run only on time on every one is recorded
// missed; with it off exactly one late task is created for the latest. Three
// minutes late is on time.
func TestDailyScheduleCoalescesMissedOccurrences(t *testing.T) {
	loc := newYork(t)
	setNow(t, loc, "2026-10-30 12:00:00")
	st, _ := newTestStore(t)
	daily := recurrence.Rule{StartDate: "2026-11-02", Times: []string{"09:00"}, Frequency: recurrence.FreqDaily, EndKind: recurrence.EndNever}
	strict := newSchedule(t, st, loc, daily, true)
	lenient := newSchedule(t, st, loc, daily, false)

	got := processAt(t, st, loc, "2026-11-02 09:03:00", nil)
	if len(got) != 2 || got[0].Task == nil || got[1].Task == nil || got[0].Task.LateStart {
		t.Fatalf("three minutes late: %+v", got)
	}

	got = processAt(t, st, loc, "2026-11-05 09:10:00", nil)
	byID := map[string]ScheduleOutcome{}
	for _, outcome := range got {
		byID[outcome.Schedule.ScheduleID] = outcome
	}
	if o := byID[strict.ScheduleID]; o.Task != nil || o.Schedule.LastDisposition != OccurrenceMissed || o.Schedule.OccurrencesElapsed != 4 {
		t.Fatalf("strict after missing three: task=%v %+v", o.Task, o.Schedule)
	}
	o := byID[lenient.ScheduleID]
	if o.Task == nil || !o.Task.LateStart || !o.Task.ScheduledFor.Equal(time.Date(2026, 11, 5, 9, 0, 0, 0, loc)) ||
		o.Schedule.OccurrencesElapsed != 4 {
		t.Fatalf("lenient after missing three: %+v", o)
	}
	tasks, _ := st.ListTasks("p")
	if len(tasks) != 3 {
		t.Fatalf("tasks = %d, want 2 on-time + 1 late", len(tasks))
	}
	if next := o.Schedule.NextDueAt; next == nil || !next.Equal(time.Date(2026, 11, 6, 9, 0, 0, 0, loc)) {
		t.Fatalf("next due = %v", next)
	}
}

// FS-16.A39 (state part) — pausing passes occurrences over as paused and
// resuming never replays them; an unusable target pauses the schedule with a
// reason; editing keeps the elapsed count; deleting keeps the created tasks.
func TestScheduleControlsNeverReplayOccurrences(t *testing.T) {
	loc := newYork(t)
	setNow(t, loc, "2026-10-30 12:00:00")
	st, _ := newTestStore(t)
	sched := newSchedule(t, st, loc, recurrence.Rule{
		StartDate: "2026-11-02", Times: []string{"09:00"}, Frequency: recurrence.FreqDaily, EndKind: recurrence.EndNever,
	}, true)
	processAt(t, st, loc, "2026-11-02 09:00:01", nil)

	paused, err := st.PauseSchedule(sched.ScheduleID, "")
	if err != nil || paused.State != SchedulePaused {
		t.Fatalf("pause = %+v, %v", paused, err)
	}
	if _, err := st.PauseSchedule(sched.ScheduleID, ""); !errors.Is(err, ErrScheduleState) {
		t.Fatalf("pause twice: %v", err)
	}
	if got := processAt(t, st, loc, "2026-11-04 09:00:01", nil); len(got) != 0 {
		t.Fatalf("paused schedule ran: %+v", got)
	}
	setNow(t, loc, "2026-11-04 12:00:00")
	resumed, err := st.ResumeSchedule(sched.ScheduleID, loc)
	if err != nil || resumed.State != ScheduleActive || resumed.OccurrencesElapsed != 3 ||
		resumed.LastDisposition != OccurrencePaused || !resumed.NextDueAt.Equal(time.Date(2026, 11, 5, 9, 0, 0, 0, loc)) {
		t.Fatalf("resume = %+v, %v", resumed, err)
	}
	if tasks, _ := st.ListTasks("p"); len(tasks) != 1 {
		t.Fatalf("resume replayed: %d tasks", len(tasks))
	}

	got := processAt(t, st, loc, "2026-11-05 09:00:01", func(TaskSchedule) string { return "the target agent is archived" })
	if len(got) != 1 || got[0].Task != nil || got[0].Schedule.State != SchedulePaused ||
		got[0].Schedule.PauseReason != "the target agent is archived" || got[0].Schedule.LastDisposition != OccurrencePaused {
		t.Fatalf("unavailable target: %+v", got)
	}

	edit := got[0].Schedule
	edit.Instruction = "write shorter notes"
	edit.Rule.Times = []string{"10:30"}
	setNow(t, loc, "2026-11-05 12:00:00")
	edited, err := st.UpdateSchedule(edit, edit.Revision, "", loc)
	if err != nil || edited.Instruction != "write shorter notes" || edited.OccurrencesElapsed != 4 ||
		edited.State != SchedulePaused || !edited.NextDueAt.Equal(time.Date(2026, 11, 6, 10, 30, 0, 0, loc)) {
		t.Fatalf("edit = %+v, %v", edited, err)
	}
	if _, err := st.UpdateSchedule(edit, edit.Revision, "", loc); !errors.Is(err, ErrScheduleConflict) {
		t.Fatalf("stale edit: %v", err)
	}

	if err := st.DeleteSchedule(sched.ScheduleID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ReadSchedule(sched.ScheduleID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("read deleted: %v", err)
	}
	if tasks, _ := st.ListTasks("p"); len(tasks) != 1 || tasks[0].ScheduleID != sched.ScheduleID {
		t.Fatalf("delete touched tasks: %+v", tasks)
	}
}

// FS-16.R57 — a rule with no future occurrence creates nothing.
func TestScheduleWithoutAFutureOccurrenceIsRefused(t *testing.T) {
	loc := newYork(t)
	setNow(t, loc, "2026-10-30 12:00:00")
	st, _ := newTestStore(t)
	id, _ := NewScheduleID()
	_, err := st.CreateSchedule(TaskSchedule{
		ScheduleID: id, Project: "p", DisplayName: "old", Instruction: "x", TargetKind: TargetLaunch, Role: "impl",
		Rule: recurrence.Rule{StartDate: "2026-10-01", Times: []string{"09:00"}, Frequency: recurrence.FreqDaily,
			EndKind: recurrence.EndUntil, EndDate: "2026-10-29"},
		CreatedByKind: "person",
	}, "", loc)
	if !errors.Is(err, ErrScheduleRule) {
		t.Fatalf("err = %v", err)
	}
	if list, _ := st.ListSchedules("p"); len(list) != 0 {
		t.Fatalf("refused schedule was written: %+v", list)
	}
}

// reopenStore closes st and opens the same home again, as a restart does.
func reopenStore(t *testing.T, st *Store, home string) *Store {
	t.Helper()
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err := Open(home)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// FS-16.R54/R58, TS-10.R56 — after a restart in a zone west of the one that
// handled an occurrence, the next occurrence runs at the new zone's 09:00,
// and the occurrence already handled is not replayed.
func TestScheduleRestartWestMovesPendingOccurrenceWithoutReplay(t *testing.T) {
	home := t.TempDir()
	setNow(t, time.UTC, "2026-01-01 12:00:00")
	st, err := Open(home)
	if err != nil {
		t.Fatal(err)
	}
	sched := newSchedule(t, st, time.UTC, recurrence.Rule{StartDate: "2026-01-02", Times: []string{"09:00"},
		Frequency: recurrence.FreqDaily, EndKind: recurrence.EndNever}, true)
	if got := processAt(t, st, time.UTC, "2026-01-02 09:00:01", nil); len(got) != 1 || got[0].Task == nil {
		t.Fatalf("first occurrence in UTC: %+v", got)
	}

	st = reopenStore(t, st, home)
	loc := newYork(t)
	for _, wall := range []string{"2026-01-02 09:00:01", "2026-01-03 04:00:01"} {
		if got := processAt(t, st, loc, wall, nil); len(got) != 0 {
			t.Fatalf("New York %s created %+v", wall, got)
		}
	}
	got := processAt(t, st, loc, "2026-01-03 09:00:01", nil)
	if len(got) != 1 || got[0].Task == nil || got[0].Task.LateStart ||
		!got[0].Task.ScheduledFor.Equal(time.Date(2026, 1, 3, 9, 0, 0, 0, loc)) ||
		got[0].Schedule.OccurrencesElapsed != 2 {
		t.Fatalf("schedule %s at New York 09:00: %+v", sched.ScheduleID, got)
	}
}

// FS-16.R54/R58, TS-10.R56 — after a restart east of the creating zone, an
// on-time-only occurrence runs at the new zone's 09:00 rather than being
// missed at the old instant, and the count still ends the rule.
func TestScheduleRestartEastRunsOnTimeAndKeepsCount(t *testing.T) {
	home := t.TempDir()
	loc := newYork(t)
	setNow(t, loc, "2026-01-01 12:00:00")
	st, err := Open(home)
	if err != nil {
		t.Fatal(err)
	}
	newSchedule(t, st, loc, recurrence.Rule{StartDate: "2026-01-02", Times: []string{"09:00"},
		Frequency: recurrence.FreqDaily, EndKind: recurrence.EndCount, EndCount: 2}, true)

	st = reopenStore(t, st, home)
	for _, day := range []int{2, 3} {
		got := processAt(t, st, time.UTC, fmt.Sprintf("2026-01-%02d 09:00:01", day), nil)
		if len(got) != 1 || got[0].Task == nil || got[0].Task.LateStart ||
			!got[0].Task.ScheduledFor.Equal(time.Date(2026, 1, day, 9, 0, 0, 0, time.UTC)) ||
			got[0].Schedule.OccurrencesElapsed != day-1 {
			t.Fatalf("UTC 09:00 on Jan %d: %+v", day, got)
		}
	}
	list, _ := st.ListSchedules("p")
	if len(list) != 1 || list[0].State != ScheduleFinished {
		t.Fatalf("count rule did not finish: %+v", list)
	}
}
