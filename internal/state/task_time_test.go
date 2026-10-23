package state

import (
	"errors"
	"testing"
	"time"
)

func timeArm(at time.Time, onTimeOnly bool) TaskArm {
	return TaskArm{Kind: ArmTime, DueAt: &at, OnTimeOnly: onTimeOnly}
}

func resolveDueAt(t *testing.T, st *Store, now time.Time) []Task {
	t.Helper()
	withFixedNow(t, now)
	changed, err := st.ResolveDueTimeArms(8)
	if err != nil {
		t.Fatalf("ResolveDueTimeArms: %v", err)
	}
	return changed
}

// FS-16.A35 — a time arm is one more conjunct: after 17:30 the task still waits
// for its other prerequisite, then becomes ready once; a past time is refused
// without writing a task.
func TestTimeArmCombinesWithTaskArmAndRefusesPastTimes(t *testing.T) {
	created := mustTime(t, "2026-10-23T12:00:00Z")
	due := mustTime(t, "2026-10-23T17:30:00Z")
	withFixedNow(t, created)
	st, _ := newTestStore(t)
	prereq := newTask(t, st, "p", "prereq")
	task := newTask(t, st, "p", "timed", timeArm(due, true), taskArm(prereq.TaskID, OutcomeSuccess))
	if task.State != TaskArmed || task.Arms[0].DueAt == nil || !task.Arms[0].DueAt.Equal(due) {
		t.Fatalf("created task = %+v", task)
	}

	if changed := resolveDueAt(t, st, due.Add(-time.Second)); len(changed) != 0 {
		t.Fatalf("resolved before due: %+v", changed)
	}
	resolveDueAt(t, st, due.Add(2*time.Second))
	if got, _ := st.ReadTask(task.TaskID); got.State != TaskArmed || got.LateStart {
		t.Fatalf("after due, before prerequisite: state=%s late=%v", got.State, got.LateStart)
	}
	if err := st.RegisterWorkResult(WorkResult{SourceKind: SourceTask, SourceID: prereq.TaskID, Outcome: OutcomeSuccess}); err != nil {
		t.Fatal(err)
	}
	changed, err := st.EvaluateSource(SourceTask, prereq.TaskID)
	if err != nil || len(changed) != 1 || changed[0].State != TaskReady {
		t.Fatalf("EvaluateSource = %+v, %v", changed, err)
	}

	withFixedNow(t, created)
	for _, at := range []time.Time{created, created.Add(-time.Minute)} {
		id, _ := st.NewTaskID()
		_, err := st.CreateTask(Task{TaskID: id, Project: "p", DisplayName: "past", Instruction: "x",
			TargetKind: TargetLaunch, Role: "impl", CreatedByKind: "person", Arms: []TaskArm{timeArm(at, true)}})
		if !errors.Is(err, ErrTaskStartTime) {
			t.Fatalf("past time %s: err = %v", at, err)
		}
		if _, err := st.ReadTask(id); !errors.Is(err, ErrNotFound) {
			t.Fatalf("refused create wrote a task: %v", err)
		}
	}
	id, _ := st.NewTaskID()
	_, err = st.CreateTask(Task{TaskID: id, Project: "p", DisplayName: "two", Instruction: "x",
		TargetKind: TargetLaunch, Role: "impl", CreatedByKind: "person",
		Arms: []TaskArm{timeArm(due, true), timeArm(due.Add(time.Hour), true)}})
	if !errors.Is(err, ErrTaskStartTime) {
		t.Fatalf("second time arm: err = %v", err)
	}
}

// FS-16.A36 (one-time part) — lateness is decided by when Chuck first observes
// the instant: within five minutes is on time; later parks with Run only on
// time on, and starts marked late with it off. Re-arming repairs a missed start.
func TestTimeArmLatenessFollowsRunOnlyOnTime(t *testing.T) {
	created := mustTime(t, "2026-10-23T12:00:00Z")
	due := mustTime(t, "2026-10-23T17:30:00Z")
	withFixedNow(t, created)
	st, _ := newTestStore(t)
	onTime := newTask(t, st, "p", "three-minutes", timeArm(due, true))
	strict := newTask(t, st, "p", "strict", timeArm(due.Add(time.Hour), true))
	lenient := newTask(t, st, "p", "lenient", timeArm(due.Add(time.Hour), false))

	resolveDueAt(t, st, due.Add(3*time.Minute))
	if got, _ := st.ReadTask(onTime.TaskID); got.State != TaskReady || got.LateStart {
		t.Fatalf("three minutes late: state=%s late=%v", got.State, got.LateStart)
	}

	resolveDueAt(t, st, due.Add(time.Hour+10*time.Minute))
	got, _ := st.ReadTask(strict.TaskID)
	if got.State != TaskDependencyFailed || got.AttentionReason != MissedStartTimeReason || got.Arms[0].State != ArmUnsatisfiable {
		t.Fatalf("strict missed: %+v", got)
	}
	got, _ = st.ReadTask(lenient.TaskID)
	if got.State != TaskReady || !got.LateStart {
		t.Fatalf("lenient missed: state=%s late=%v", got.State, got.LateStart)
	}
	if again := resolveDueAt(t, st, due.Add(2*time.Hour)); len(again) != 0 {
		t.Fatalf("resolved twice: %+v", again)
	}

	rearmed, err := st.RearmTask(strict.TaskID, []TaskArm{timeArm(due.Add(3*time.Hour), true)})
	if err != nil || rearmed.State != TaskArmed || rearmed.AttentionReason != "" {
		t.Fatalf("rearm missed start = %+v, %v", rearmed, err)
	}
}
