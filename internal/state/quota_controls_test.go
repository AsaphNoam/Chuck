package state

import (
	"errors"
	"testing"
	"time"
)

func TestQuotaControlSnapshotOnlyCancelsCapturedEpisode(t *testing.T) {
	st, _ := newTestStore(t)
	due := timeNow().UTC().Add(time.Hour).Truncate(time.Second)
	first := quotaFixture(t, st, "quota_control_snapshot", "episode-old", due)
	snapshot, err := st.SnapshotAgentQuota(first.AgentID)
	if err != nil || snapshot == nil {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	replacementReset, replacementDue := due.Add(time.Minute), due.Add(2*time.Minute)
	second, err := st.ObserveQuota(QuotaObservation{AgentID: first.AgentID, EpisodeID: "episode-new", SourceGeneration: "generation-2", ExecutingTurn: "turn-2",
		Owner: QuotaOwner{Kind: "ordinary"}, ObservedAt: due, ResetAt: &replacementReset, DueAt: &replacementDue, State: QuotaScheduled})
	if err != nil {
		t.Fatal(err)
	}
	changed, err := st.CancelQuotaSnapshot(*snapshot)
	if err != nil || changed {
		t.Fatalf("stale snapshot cancellation changed=%v err=%v", changed, err)
	}
	current, err := st.GetQuota(first.AgentID)
	if err != nil || current.EpisodeID != second.EpisodeID || current.State != QuotaScheduled {
		t.Fatalf("new episode was invalidated: %+v err=%v", current, err)
	}
	newSnapshot, err := st.SnapshotAgentQuota(first.AgentID)
	if err != nil || newSnapshot == nil {
		t.Fatalf("new snapshot=%+v err=%v", newSnapshot, err)
	}
	changed, err = st.CancelQuotaSnapshot(*newSnapshot)
	if err != nil || !changed {
		t.Fatalf("current snapshot cancellation changed=%v err=%v", changed, err)
	}
}

func TestCancelledTaskQuotaStillPreservesAssignmentUntilOwnerClosure(t *testing.T) {
	st, _ := newTestStore(t)
	now := timeNow().UTC().Truncate(time.Second)
	due := now.Add(time.Hour)
	agent := "quota_control_task"
	if err := st.WriteAgent(testAgent(agent, now)); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.Exec(`INSERT INTO tasks(task_id,project,display_name,instruction,target_kind,state,created_by_kind,created_at,updated_at,assigned_agent_id,assigned_generation)
VALUES('task_quota_control','my-app','task','instruction','role','running','user',?,?,?,?)`, formatTime(now), formatTime(now), agent, "generation-task"); err != nil {
		t.Fatal(err)
	}
	q, err := st.ObserveQuota(QuotaObservation{AgentID: agent, EpisodeID: "episode-task", SourceGeneration: "generation-task", ExecutingTurn: "turn-task",
		Owner: QuotaOwner{Kind: "task", TaskID: "task_quota_control"}, ObservedAt: now, ResetAt: quotaResetFor(due), DueAt: &due, State: QuotaScheduled})
	if err != nil {
		t.Fatal(err)
	}
	snapshots, err := st.SnapshotOwnerQuotas(QuotaOwner{Kind: "task", TaskID: "task_quota_control"})
	if err != nil || len(snapshots) != 1 {
		t.Fatalf("owner snapshots=%+v err=%v", snapshots, err)
	}
	if _, err := st.CancelQuotaSnapshots(snapshots); err != nil {
		t.Fatal(err)
	}
	task := Task{TaskID: "task_quota_control", AssignedAgentID: agent, AssignedGeneration: "generation-task"}
	held, err := st.QuotaHoldsTask(task)
	if err != nil || !held {
		t.Fatalf("cancelled schedule dropped quota owner hold=%v err=%v quota=%+v", held, err, q)
	}
}

func TestAgentArchiveFlagStoreLeavesQuotaForCrossStoreCommit(t *testing.T) {
	st, _ := newTestStore(t)
	now := timeNow().UTC().Truncate(time.Second)
	due := now.Add(time.Hour)
	q := quotaFixture(t, st, "quota_archive_control", "episode-archive", due)
	if err := st.SetAgentsArchived([]string{q.AgentID}, true); err != nil {
		t.Fatal(err)
	}
	archived, err := st.ReadAgent(q.AgentID)
	if err != nil || !archived.Archived {
		t.Fatalf("archive=%+v err=%v", archived, err)
	}
	current, err := st.GetQuota(q.AgentID)
	if err != nil || current.State != QuotaScheduled {
		t.Fatalf("quota after archive=%+v err=%v", current, err)
	}
}

func TestScheduledQuotaRejectsPastResetAndDueBeforeReset(t *testing.T) {
	st, _ := newTestStore(t)
	now := timeNow().UTC().Truncate(time.Second)
	if err := st.WriteAgent(testAgent("quota_bad_schedule", now)); err != nil {
		t.Fatal(err)
	}
	observed := now.Add(-time.Minute)
	pastReset := observed.Add(-time.Second)
	futureReset := observed.Add(time.Minute)
	tooEarlyDue := futureReset.Add(-time.Second)
	for _, tc := range []struct {
		name       string
		reset, due time.Time
	}{{"past_reset", pastReset, now}, {"due_before_reset", futureReset, tooEarlyDue}} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := st.ObserveQuota(QuotaObservation{AgentID: "quota_bad_schedule", EpisodeID: "episode-" + tc.name, SourceGeneration: "g", ExecutingTurn: "t",
				Owner: QuotaOwner{Kind: "ordinary"}, ObservedAt: observed, ResetAt: &tc.reset, DueAt: &tc.due, State: QuotaScheduled}); !errors.Is(err, ErrQuotaConflict) {
				t.Fatalf("invalid schedule error=%v", err)
			}
		})
	}
}

func TestPipelineRunSnapshotFindsTaskAndRoomQuotaOwners(t *testing.T) {
	st, _ := newTestStore(t)
	now := timeNow().UTC().Truncate(time.Second)
	due := now.Add(time.Hour)
	for _, id := range []string{"quota_pipeline_task", "quota_pipeline_room"} {
		if err := st.WriteAgent(testAgent(id, now)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.db.Exec(`INSERT INTO tasks(task_id,project,display_name,instruction,target_kind,state,created_by_kind,created_at,updated_at,assigned_agent_id,assigned_generation)
VALUES('pipeline-task','my-app','task','instruction','role','running','user',?,?,?,?)`, formatTime(now), formatTime(now), "quota_pipeline_task", "g"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.Exec(`INSERT INTO task_lineage(task_id,pipeline_run_id,created_at) VALUES('pipeline-task','run-owner',?)`, formatTime(now)); err != nil {
		t.Fatal(err)
	}
	taskQuota, err := st.ObserveQuota(QuotaObservation{AgentID: "quota_pipeline_task", EpisodeID: "pipeline-task-episode", SourceGeneration: "g", ExecutingTurn: "t",
		Owner: QuotaOwner{Kind: "task", TaskID: "pipeline-task"}, ObservedAt: now, ResetAt: quotaResetFor(due), DueAt: &due, State: QuotaScheduled})
	if err != nil {
		t.Fatal(err)
	}
	roomID := "pipeline-room"
	if _, err := st.db.Exec(`INSERT INTO think_tanks(room_id,command_id,title,goal,origin_project,openings,phase,control,created_at,updated_at,pipeline_run_id)
VALUES(?,?,'room','goal','my-app',0,'discussion','running',?,?, 'run-owner')`, roomID, "pipeline-room-command", formatTime(now), formatTime(now)); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.Exec(`INSERT INTO think_tank_attempts(attempt_id,room_id,agent_id,turn,token,generation,turn_id,head,checkpoint,delivered_to,state,created_at)
VALUES('pipeline-room-attempt',?,?,'discussion','pipeline-room-token','rg','rt',0,0,0,'failed',?)`, roomID, "quota_pipeline_room", formatTime(now)); err != nil {
		t.Fatal(err)
	}
	roomQuota, err := st.ObserveQuota(QuotaObservation{AgentID: "quota_pipeline_room", EpisodeID: "pipeline-room-episode", SourceGeneration: "rg", ExecutingTurn: "rt",
		Owner: QuotaOwner{Kind: "room", RoomID: roomID, AttemptID: "pipeline-room-attempt", Turn: ThinkTankTurnDiscussion}, ObservedAt: now, ResetAt: quotaResetFor(due), DueAt: &due, State: QuotaScheduled})
	if err != nil {
		t.Fatal(err)
	}
	snapshots, err := st.SnapshotQuotaOwnerID("pipeline", "run_id", "run-owner")
	if err != nil || len(snapshots) != 2 {
		t.Fatalf("pipeline run owner snapshots=%+v err=%v", snapshots, err)
	}
	seen := map[string]bool{}
	for _, snap := range snapshots {
		seen[snap.AgentID] = true
	}
	if !seen[taskQuota.AgentID] || !seen[roomQuota.AgentID] {
		t.Fatalf("missing inherited pipeline owner episodes: %+v", snapshots)
	}
}
