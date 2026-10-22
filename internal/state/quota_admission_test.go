package state

import (
	"testing"
	"time"
)

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
