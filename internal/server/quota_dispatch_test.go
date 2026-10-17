package server

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// FS-01.A24/A27: known-reset fixture evidence resumes the same conversation,
// issues one code-owned continuation, and records one Chuck notice. This test
// proves dispatch mechanics, not any real provider's timestamp support.
func TestQuotaDispatcherContinuesSameConversationOnce(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		t.Run(map[bool]string{false: "running", true: "stopped"}[stopped], func(t *testing.T) {
			s, ts, log := activationTestServer(t)
			id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
			generation := s.registry.Generation(id)
			if stopped {
				if ae := s.stopAgent(context.Background(), id); ae != nil {
					t.Fatal(ae)
				}
			}
			due := time.Now().UTC().Add(-time.Second)
			q, err := s.stateStore.ObserveQuota(state.QuotaObservation{AgentID: id, EpisodeID: "known-reset-fixture", SourceGeneration: generation, ExecutingTurn: "interrupted-turn", Owner: state.QuotaOwner{Kind: "ordinary"}, ObservedAt: due.Add(-2 * time.Minute), ResetAt: quotaResetFor(due), DueAt: &due, State: state.QuotaScheduled})
			if err != nil {
				t.Fatal(err)
			}
			s.startQuotaContinuation(context.Background(), q)
			waitPrompts(t, log, 1)
			deadline := time.Now().Add(5 * time.Second)
			for {
				q, err = s.stateStore.GetQuota(id)
				if err != nil {
					t.Fatal(err)
				}
				if q.State == state.QuotaContinued {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("continuation did not settle: %+v", q)
				}
				time.Sleep(10 * time.Millisecond)
			}
			if q.ContinuedTurn == "" || q.ContinuedTurn == q.ExecutingTurn {
				t.Fatalf("not a fresh executing turn: %+v", q)
			}
			s.startQuotaContinuation(context.Background(), q)
			if got := promptCount(t, log); got != 1 {
				t.Fatalf("duplicate provider prompts: %d", got)
			}
			waitStatus(t, s, id, "idle")
			events := readEvents(t, s, id)
			notices := 0
			for _, ev := range events {
				if ev.Type != runtime.EvNotice {
					continue
				}
				var n runtime.NoticeData
				if err := json.Unmarshal(ev.Data, &n); err != nil {
					t.Fatal(err)
				}
				if n.Title == "Chuck continued this conversation when the quota reset" {
					notices++
				}
			}
			if notices != 1 {
				t.Fatalf("continuation notices=%d", notices)
			}
			var wantCount, wantLastSeq int64
			var wantUpdatedAt string
			for _, ev := range events {
				if ev.Seq > 0 {
					wantCount++
				}
				if ev.Seq > wantLastSeq {
					wantLastSeq = ev.Seq
				}
				if ev.Ts > wantUpdatedAt {
					wantUpdatedAt = ev.Ts
				}
			}
			indexDeadline := time.Now().Add(5 * time.Second)
			for {
				var eventCount, lastSeq int64
				var updatedAt string
				if err := s.stateStore.DB().QueryRow(`SELECT event_count, last_seq, updated_at FROM sessions WHERE agent_id=?`, id).Scan(&eventCount, &lastSeq, &updatedAt); err != nil {
					t.Fatal(err)
				}
				if eventCount == wantCount && lastSeq == wantLastSeq && updatedAt == wantUpdatedAt {
					break
				}
				if time.Now().After(indexDeadline) {
					t.Fatalf("session index count/seq/updated=%d/%d/%s, transcript=%d/%d/%s", eventCount, lastSeq, updatedAt, wantCount, wantLastSeq, wantUpdatedAt)
				}
				time.Sleep(10 * time.Millisecond)
			}
			// FS-01.A24: live reload, phone-sized window and archived reads retain
			// the exact durable order and Chuck attribution of the notice.
			h := s.routes()
			path := "/api/sessions/" + id + "/transcript"
			for _, suffix := range []string{"", "?limit=100"} {
				got := getWindow(t, h, newLocalRequest(http.MethodGet, path+suffix, nil))
				if !reflect.DeepEqual(got.Events, events) {
					t.Fatalf("reloaded transcript differs: %s", suffix)
				}
			}
			resp, body := post(t, ts.URL+"/api/sessions/"+id+"/archive", map[string]any{})
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("archive = %d %s", resp.StatusCode, body)
			}
			archived := getWindow(t, h, newLocalRequest(http.MethodGet, path, nil))
			if !reflect.DeepEqual(archived.Events, events) {
				t.Fatal("archived transcript changed continuation history or attribution")
			}
		})
	}
}

// FS-01.A25: a provider terminal quota preserves an exclusive live assignment;
// subsequent recovery uses that same task and assignee and records no result.
func TestQuotaTaskAssignmentSurvivesProviderInterruptionAndRestart(t *testing.T) {
	t.Setenv("FAKEACP_CAPS", "1")
	t.Setenv("FAKEACP_FAILURE_RESULT", `{"id":"quota-terminal","revision":1,"category":"limit","severity":"error","title":"Account quota exhausted","actions":[]}`)
	s, _, log := activationTestServer(t)
	task := newLaunchTask(t, s, "unfinished quota work")
	s.dispatchReadyTasks(context.Background())
	task = waitTaskState(t, s, task.TaskID, state.TaskRunning)
	deadline := time.Now().Add(5 * time.Second)
	var q state.QuotaIntent
	for {
		var err error
		q, err = s.stateStore.GetQuota(task.AssignedAgentID)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("quota episode absent: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if q.Owner.Kind != "task" || q.Owner.TaskID != task.TaskID || q.NextAction != state.QuotaActionCheck || q.ResetAt != nil {
		t.Fatalf("lost owner: %+v", q)
	}
	if err := s.recoverTasks(context.Background()); err != nil {
		t.Fatal(err)
	}
	task, err := s.stateStore.ReadTask(task.TaskID)
	if err != nil || task.State != state.TaskRunning || task.Outcome != "" || task.AssignedAgentID != q.AgentID {
		t.Fatalf("restart changed assignment: %+v %v", task, err)
	}
	reset := time.Now().UTC().Add(40 * time.Millisecond)
	due := reset.Add(state.QuotaResetBuffer)
	q, err = s.stateStore.ObserveQuota(state.QuotaObservation{AgentID: q.AgentID, EpisodeID: q.EpisodeID, SourceGeneration: q.SourceGeneration, ExecutingTurn: q.ExecutingTurn, Owner: q.Owner, ObservedAt: q.ObservedAt, ResetAt: &reset, DueAt: &due, State: state.QuotaScheduled})
	if err != nil {
		t.Fatal(err)
	}
	// Fast-forward past the reset buffer rather than sleeping a real minute.
	if _, err := s.stateStore.DB().Exec(`UPDATE quota_intents SET due_at=? WHERE agent_id=?`, reset.UTC().Format(time.RFC3339), q.AgentID); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Until(reset) + time.Millisecond)
	s.startQuotaContinuation(context.Background(), q)
	waitPrompts(t, log, 2)
	task, err = s.stateStore.ReadTask(task.TaskID)
	if err != nil || task.State != state.TaskRunning || task.Outcome != "" || task.AssignedAgentID != q.AgentID {
		t.Fatalf("continuation replaced assignment: %+v %v", task, err)
	}
}

// quotaResetFor is the latest reset a continuation due at due may follow.
func quotaResetFor(due time.Time) *time.Time {
	reset := due.Add(-state.QuotaResetBuffer)
	return &reset
}
