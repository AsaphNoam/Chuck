package state

import (
	"errors"
	"testing"
	"time"
)

// ASRG-2: ownership can change after preflight. The allowance transaction
// must reject a finished task or paused room without charging the check.
func TestQuotaCheckRevalidatesOwnerWhenCharging(t *testing.T) {
	for _, kind := range []string{"task", "room"} {
		t.Run(kind, func(t *testing.T) {
			st, _ := newTestStore(t)
			now := timeNow().UTC()
			var q QuotaIntent
			var err error
			var invalidate func()
			if kind == "task" {
				id := "quota_task_check"
				if err := st.WriteAgent(testAgent(id, now)); err != nil {
					t.Fatal(err)
				}
				task := newTask(t, st, "proj", "unfinished quota task")
				if _, err := st.db.Exec(`UPDATE tasks SET state=?, assigned_agent_id=?, assigned_generation='source', runtime_claim=? WHERE task_id=?`, TaskRunning, id, ClaimBorrowed, task.TaskID); err != nil {
					t.Fatal(err)
				}
				q, err = st.ObserveQuota(QuotaObservation{AgentID: id, EpisodeID: "task-check", SourceGeneration: "source", ExecutingTurn: "turn",
					Owner: QuotaOwner{Kind: "task", TaskID: task.TaskID}, ObservedAt: now.Add(-61 * time.Minute), State: QuotaUnknownReset, Check: true})
				invalidate = func() {
					if _, err := st.db.Exec(`UPDATE tasks SET state=?, outcome=? WHERE task_id=?`, TaskFinished, OutcomeSuccess, task.TaskID); err != nil {
						t.Fatal(err)
					}
				}
			} else {
				d, source, _ := quotaRoomSetup(t, st, ThinkTankTurnDiscussion)
				q, err = st.ObserveOwnedQuota(QuotaObservation{AgentID: source.AgentID, EpisodeID: "room-check", SourceGeneration: source.Generation,
					ExecutingTurn: source.TurnID, ObservedAt: now.Add(-61 * time.Minute), State: QuotaUnknownReset, Check: true})
				invalidate = func() {
					if _, err := st.db.Exec(`UPDATE think_tanks SET control=? WHERE room_id=?`, ThinkTankPaused, d.Room.RoomID); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := st.ClaimQuota(q.AgentID, q.EpisodeID, q.Revision, "check-claim"); err != nil {
				t.Fatal(err)
			}
			q, err = st.GetQuota(q.AgentID)
			if err != nil {
				t.Fatal(err)
			}
			if eligible, err := st.QuotaWorkEligible(q, 2); err != nil || !eligible {
				t.Fatalf("initial eligibility=%v err=%v", eligible, err)
			}
			invalidate()
			if eligible, err := st.StartEligibleQuotaCheck(q, "codex-acp", 2); eligible || !errors.Is(err, ErrQuotaOwner) {
				t.Fatalf("changed owner admitted=%v err=%v", eligible, err)
			}
			if got, err := st.GetQuota(q.AgentID); err != nil || got.ChecksStarted != 0 || got.CheckStarted {
				t.Fatalf("check charged: %+v err=%v", got, err)
			}
		})
	}
}

// FS-01.A25/A27: waking borrowed owned work must reserve runtime capacity;
// capacity deferral preserves the task and its start-attempt allowance.
func TestQuotaTaskResumeUpgradeRespectsRuntimeBudget(t *testing.T) {
	st, _ := newTestStore(t)
	now := timeNow().UTC().Truncate(time.Second)
	quotaAgent := "quota_resume_task_agent"
	occupantAgent := "quota_resume_occupant_agent"
	for _, id := range []string{quotaAgent, occupantAgent} {
		if err := st.WriteAgent(testAgent(id, now)); err != nil {
			t.Fatal(err)
		}
	}
	quotaTask := newTask(t, st, "proj", "quota resume owner")
	occupant := newTask(t, st, "proj", "existing reserved runtime")
	if _, err := st.DB().Exec(`UPDATE tasks SET state=?, assigned_agent_id=?, assigned_generation=?, runtime_claim=? WHERE task_id=?`,
		TaskRunning, quotaAgent, "source-generation", ClaimBorrowed, quotaTask.TaskID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB().Exec(`UPDATE tasks SET state=?, assigned_agent_id=?, assigned_generation=?, runtime_claim=? WHERE task_id=?`,
		TaskRunning, occupantAgent, "occupant-generation", ClaimWoke, occupant.TaskID); err != nil {
		t.Fatal(err)
	}
	reset := now.Add(-time.Second)
	q, err := st.ObserveQuota(QuotaObservation{AgentID: quotaAgent, EpisodeID: "quota-resume-episode",
		SourceGeneration: "source-generation", ExecutingTurn: "source-turn",
		Owner: QuotaOwner{Kind: "task", TaskID: quotaTask.TaskID}, ObservedAt: reset.Add(-2 * time.Minute),
		ResetAt: quotaResetFor(reset), DueAt: &reset, State: QuotaScheduled})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ClaimQuota(q.AgentID, q.EpisodeID, q.Revision, "quota-resume-token"); err != nil {
		t.Fatal(err)
	}
	q, err = st.GetQuota(q.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if admitted, err := st.AdmitQuotaTaskResume(q, 1); err != nil || admitted {
		t.Fatalf("resume with full runtime budget admitted=%v err=%v", admitted, err)
	}
	current, err := st.ReadTask(quotaTask.TaskID)
	if err != nil || current.RuntimeClaim != ClaimBorrowed {
		t.Fatalf("busy resume changed task reservation: %+v err=%v", current, err)
	}
	if _, err := st.DB().Exec(`UPDATE tasks SET runtime_claim=? WHERE task_id=?`, ClaimBorrowed, occupant.TaskID); err != nil {
		t.Fatal(err)
	}
	if admitted, err := st.AdmitQuotaTaskResume(q, 1); err != nil || !admitted {
		t.Fatalf("resume after slot opened admitted=%v err=%v", admitted, err)
	}
	current, err = st.ReadTask(quotaTask.TaskID)
	if err != nil || current.RuntimeClaim != ClaimWoke {
		t.Fatalf("quota resume did not reserve waking slot: %+v err=%v", current, err)
	}
	// A task already holding a created/woke reservation keeps it through quota
	// recovery, even when the current limit is now full.
	if admitted, err := st.AdmitQuotaTaskResume(q, 1); err != nil || !admitted {
		t.Fatalf("existing waking reservation was not retained: admitted=%v err=%v", admitted, err)
	}
}

// CR-A2-2: a finished task whose release is still pending keeps its slot, so
// quota's borrowed-to-woke conversion cannot exceed the budget before cleanup.
func TestQuotaTaskResumeCountsPendingReleaseClaims(t *testing.T) {
	st, _ := newTestStore(t)
	now := timeNow().UTC().Truncate(time.Second)
	quotaAgent := "quota_pending_release_agent"
	if err := st.WriteAgent(testAgent(quotaAgent, now)); err != nil {
		t.Fatal(err)
	}
	quotaTask := newTask(t, st, "proj", "quota resume owner")
	if _, err := st.DB().Exec(`UPDATE tasks SET state=?, assigned_agent_id=?, assigned_generation=?, runtime_claim=? WHERE task_id=?`,
		TaskRunning, quotaAgent, "source-generation", ClaimBorrowed, quotaTask.TaskID); err != nil {
		t.Fatal(err)
	}
	reporter := newTask(t, st, "proj", "reported, release pending")
	if _, err := st.DB().Exec(`UPDATE tasks SET state=?, outcome=?, outcome_source='agent', pending_release=1, assigned_agent_id=?, assigned_generation=?, runtime_claim=? WHERE task_id=?`,
		TaskFinished, OutcomeSuccess, "a_reporter", "reporter-generation", ClaimCreated, reporter.TaskID); err != nil {
		t.Fatal(err)
	}
	reset := now.Add(-time.Second)
	q, err := st.ObserveQuota(QuotaObservation{AgentID: quotaAgent, EpisodeID: "quota-pending-release-episode",
		SourceGeneration: "source-generation", ExecutingTurn: "source-turn",
		Owner: QuotaOwner{Kind: "task", TaskID: quotaTask.TaskID}, ObservedAt: reset.Add(-2 * time.Minute),
		ResetAt: quotaResetFor(reset), DueAt: &reset, State: QuotaScheduled})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ClaimQuota(q.AgentID, q.EpisodeID, q.Revision, "quota-pending-release-token"); err != nil {
		t.Fatal(err)
	}
	if q, err = st.GetQuota(q.AgentID); err != nil {
		t.Fatal(err)
	}
	if admitted, err := st.AdmitQuotaTaskResume(q, 1); err != nil || admitted {
		t.Fatalf("resume admitted while a finished task still held the only slot: admitted=%v err=%v", admitted, err)
	}
	if err := st.CompleteTaskRelease(reporter.TaskID); err != nil {
		t.Fatal(err)
	}
	if admitted, err := st.AdmitQuotaTaskResume(q, 1); err != nil || !admitted {
		t.Fatalf("resume after the release completed admitted=%v err=%v", admitted, err)
	}
}
