package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// CR-A2-7: a provider quota failure leaves the agent in error, but cancelling
// that exact episode must admit a later waking message through the normal mail
// activation path.
func TestFailedProviderTurnAdmitsLaterMailAfterQuotaCancellation(t *testing.T) {
	for _, kind := range []string{"mail", "task"} {
		t.Run(kind, func(t *testing.T) {
			failure := `{"id":"turn-failed-mail","revision":1,"category":"limit","severity":"error","title":"quota","actions":[]}`
			t.Setenv("FAKEACP_FAILURE_UPDATE", failure)
			t.Setenv("FAKEACP_FAILURE_RESULT", failure)
			t.Setenv("FAKEACP_CAPS", "1")
			srv, ts, promptLog := activationTestServer(t)
			id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
			if resp, body := post(t, ts.URL+"/api/sessions/"+id+"/prompt", map[string]string{"text": "first"}); resp.StatusCode != http.StatusAccepted {
				t.Fatalf("first prompt = %d: %s", resp.StatusCode, body)
			}
			waitPrompts(t, promptLog, 1)
			var q state.QuotaIntent
			var err error
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				q, err = srv.stateStore.GetQuota(id)
				if err == nil {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			if err != nil {
				t.Fatalf("quota after failed provider turn: %v", err)
			}
			payload, _ := json.Marshal(map[string]any{"target": "quota_continuation", "episode_id": q.EpisodeID, "revision": q.Revision})
			resp, raw := post(t, ts.URL+"/api/sessions/"+id+"/cancel", json.RawMessage(payload))
			if resp.StatusCode != http.StatusAccepted || !strings.Contains(string(raw), `"cancelled":true`) {
				t.Fatalf("cancel = %d: %s", resp.StatusCode, raw)
			}
			if kind == "task" {
				task := newAgentTask(t, srv, id, "work after failed provider turn")
				running := dispatchUntilTaskState(t, srv, task.TaskID, state.TaskRunning)
				if running.RuntimeClaim != state.ClaimBorrowed {
					t.Fatalf("task did not borrow existing runtime: %+v", running)
				}
				waitPrompts(t, promptLog, 2)
				return
			}
			insertMail(t, srv, id, "wake after failed provider turn")
			for i := 0; i < 3; i++ {
				srv.executePendingMailActivations(context.Background(), id)
				time.Sleep(100 * time.Millisecond)
			}
			deadline = time.Now().Add(5 * time.Second)
			for promptCount(t, promptLog) < 2 && time.Now().Before(deadline) {
				srv.executePendingMailActivations(context.Background(), id)
				time.Sleep(50 * time.Millisecond)
			}
			if got := promptCount(t, promptLog); got != 2 {
				pending, _ := srv.stateStore.PendingActivations(state.ActivationKindMail, id, 8)
				st, _ := srv.stateStore.ReadStatus(id)
				q, _ := srv.stateStore.GetQuota(id)
				t.Fatalf("provider prompts=%d pending=%+v status=%+v quota=%+v", got, pending, st, q)
			}
		})
	}
}

func roomQuotaRecoverySetup(t *testing.T) (*Server, string, string) {
	t.Helper()
	t.Setenv("FAKEACP_CAPS", "1")
	t.Setenv("FAKEACP_FAILURE_RESULT", `{"id":"room-quota","revision":1,"category":"limit","severity":"error","title":"quota","actions":[]}`)
	srv, ts, promptLog := activationTestServer(t)
	a := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	b := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	room := createTestRoom(t, srv, []string{a, b}, 2)
	srv.progressThinkTanks(context.Background())
	waitPrompts(t, promptLog, 1)
	waitRoom(t, srv, room, func(d state.ThinkTankDetail) bool { return d.Room.Hold == "Quota reached" })
	return srv, promptLog, room
}

func TestThinkTankResumeSupersedesQuotaStates(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state string
	}{
		{name: "pending check", state: state.QuotaScheduled},
		{name: "needs attention", state: state.QuotaNeedsAttention},
		{name: "auto continue off", state: state.QuotaDisabled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, promptLog, room := roomQuotaRecoverySetup(t)
			d, err := srv.stateStore.ReadThinkTank(room)
			if err != nil || len(d.Attempts) == 0 {
				t.Fatalf("failed room attempt = %+v err=%v", d, err)
			}
			if tc.state != state.QuotaScheduled {
				if _, err := srv.stateStore.DB().Exec(`UPDATE quota_intents SET state=?, reason_code=CASE WHEN ?=? THEN 'check_exhausted' ELSE reason_code END, due_at=NULL WHERE agent_id=?`, tc.state, tc.state, state.QuotaNeedsAttention, d.Attempts[0].AgentID); err != nil {
					t.Fatal(err)
				}
			}
			oldQuota, err := srv.stateStore.GetQuota(d.Attempts[0].AgentID)
			if err != nil {
				t.Fatal(err)
			}
			rec := doJSON(t, srv.routes(), http.MethodPost, "/api/think-tanks/"+room+"/resume", "{}")
			if rec.Code != http.StatusOK {
				t.Fatalf("resume = %d %s", rec.Code, rec.Body.String())
			}
			for i := 0; i < 5; i++ {
				srv.progressThinkTanks(context.Background())
				time.Sleep(100 * time.Millisecond)
			}
			waitPrompts(t, promptLog, 2)
			q, err := srv.stateStore.GetQuota(d.Attempts[0].AgentID)
			if err != nil || q.EpisodeID == oldQuota.EpisodeID {
				t.Fatalf("resume did not supersede old quota episode: old=%+v current=%+v err=%v", oldQuota, q, err)
			}
		})
	}
}
