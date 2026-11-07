package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
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

// ARCH-QUOTA-1, FS-01.R43: a late provider quota result after an accepted
// turn Cancel creates no recovery episode and no continuation prompt.
func TestCancelledTurnLateQuotaCreatesNoContinuation(t *testing.T) {
	hold := filepath.Join(t.TempDir(), "release")
	t.Setenv("FAKEACP_CAPS", "1")
	t.Setenv("FAKEACP_HOLD_FILE", hold)
	t.Setenv("FAKEACP_HOLD_IGNORE_CANCEL", "1")
	t.Setenv("FAKEACP_FAILURE_RESULT", `{"id":"late-quota","revision":1,"category":"limit","severity":"error","title":"Account quota exhausted","actions":[]}`)
	srv, ts, promptLog := activationTestServer(t)
	t.Setenv("FAKEACP_SCENARIO", "hold_turn") // after the harness default; read at launch
	srv.registry.Chat().SetCancelGrace(0)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	if resp, body := post(t, ts.URL+"/api/sessions/"+id+"/prompt", map[string]string{"text": "first"}); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("prompt = %d: %s", resp.StatusCode, body)
	}
	waitPrompts(t, promptLog, 1)
	waitStatus(t, srv, id, "busy")
	if resp, raw := post(t, ts.URL+"/api/sessions/"+id+"/cancel", map[string]string{"target": "turn"}); resp.StatusCode != http.StatusAccepted || !strings.Contains(string(raw), `"cancelled":true`) {
		t.Fatalf("cancel = %d: %s", resp.StatusCode, raw)
	}
	if err := os.WriteFile(hold, []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, srv, id, "idle")
	srv.dispatchQuotaContinuations(context.Background())
	if q, err := srv.stateStore.GetQuota(id); !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("quota after cancelled turn = %+v, %v; want no episode", q, err)
	}
	if got := promptCount(t, promptLog); got != 1 {
		t.Fatalf("provider prompts = %d, want no continuation", got)
	}
}

// FS-01.A37/A39: the captured a_a7e830 terminal RPC refusal (monthly spend
// limit beside a named session reset, no AIR metadata) schedules one direct
// continuation of the original work at that reset plus 60s, with no
// preliminary check, and the API projection shows the existing schedule.
func TestTerminalRPCSpendSessionRefusalSchedulesContinuation(t *testing.T) {
	jerusalem, err := time.LoadLocation("Asia/Jerusalem")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().In(jerusalem)
	if now.Hour() == 22 && now.Minute() == 0 || now.Hour() == 21 && now.Minute() == 59 {
		t.Skip("observed at the reset minute boundary")
	}
	reset := time.Date(now.Year(), now.Month(), now.Day(), 22, 0, 0, 0, jerusalem)
	if !reset.After(now) {
		reset = reset.AddDate(0, 0, 1)
	}
	t.Setenv("FAKEACP_PROMPT_REJECT_DATA", `{}`)
	t.Setenv("FAKEACP_PROMPT_REJECT_MESSAGE", "Internal error: You've hit your monthly spend limit · raise it at https://claude.ai/settings/usage?from=cc_cli_limit_message · your session limit resets 10pm (Asia/Jerusalem)")
	srv, ts, promptLog := activationTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	if resp, body := post(t, ts.URL+"/api/sessions/"+id+"/prompt", map[string]string{"text": "first"}); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("prompt = %d: %s", resp.StatusCode, body)
	}
	waitPrompts(t, promptLog, 1)
	var q state.QuotaIntent
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if q, err = srv.stateStore.GetQuota(id); err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("quota after terminal RPC refusal: %v", err)
	}
	if q.State != state.QuotaScheduled || q.NextAction != state.QuotaActionContinue || q.ChecksStarted != 0 || q.ResetSource != "text" ||
		q.ResetAt == nil || !q.ResetAt.Equal(reset) || q.DueAt == nil || !q.DueAt.Equal(reset.Add(time.Minute)) {
		t.Fatalf("quota = %+v (reset %v due %v), want continue at %s plus one minute", q, q.ResetAt, q.DueAt, reset.UTC())
	}
	u, err := srv.stateMgr.Touch(id)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(u.Quota)
	if u.Quota == nil || u.Quota.NextAction != "continue" || !strings.Contains(string(raw), `"reset_at":"`+reset.UTC().Format(time.RFC3339)) {
		t.Fatalf("projection = %s", raw)
	}
	if got := promptCount(t, promptLog); got != 1 {
		t.Fatalf("provider prompts = %d, want no preliminary check or retry", got)
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
