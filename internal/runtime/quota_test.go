package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

type quotaSettlementCall struct {
	agentID, generation, turnID string
	quota                       *QuotaEvidence
	successful                  bool
}

func quotaFailure(id string, revision int, category, severity string, actions ...string) string {
	b, _ := json.Marshal(map[string]any{
		"id": id, "revision": revision, "category": category, "severity": severity,
		"title": "provider supplied title", "actions": append([]string{}, actions...),
	})
	return string(b)
}

func startQuotaTestAgent(t *testing.T, scenario string, env ...string) (*ChatRuntime, *Handle, <-chan Event, *agentState) {
	t.Helper()
	c, spec := newChatTest(t, scenario)
	spec.Env = append(spec.Env, "FAKEACP_CAPS=1")
	spec.Env = append(spec.Env, env...)
	h, err := c.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = c.Stop(context.Background(), h.AgentID) })
	ch, unsub, err := c.Subscribe(h.AgentID)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	t.Cleanup(unsub)
	as, err := c.lookup(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if !as.capabilities().SessionFailure {
		t.Fatal("fake ACP advertises AIR sessionFailure but capability did not negotiate")
	}
	return c, h, ch, as
}

func recordQuotaSettlement(c *ChatRuntime) (*sync.Mutex, *[]quotaSettlementCall) {
	mu := &sync.Mutex{}
	calls := &[]quotaSettlementCall{}
	c.SetQuotaSettlement(func(agentID, generation, turnID string, quota *QuotaEvidence, successful bool) error {
		mu.Lock()
		defer mu.Unlock()
		var copied *QuotaEvidence
		if quota != nil {
			q := *quota
			copied = &q
		}
		*calls = append(*calls, quotaSettlementCall{agentID, generation, turnID, copied, successful})
		return nil
	})
	return mu, calls
}

func TestQuotaUpdateCompletesOrdinaryTurnOnceAndSuppressesHeldSuccessor(t *testing.T) {
	quota := quotaFailure("turn-1:error", 1, "limit", "error")
	promptLog := filepath.Join(t.TempDir(), "prompts.log")
	holdFile := filepath.Join(filepath.Dir(promptLog), "release")
	c, h, ch, as := startQuotaTestAgent(t, "hold_turn",
		"FAKEACP_FAILURE_UPDATE="+quota,
		"FAKEACP_FAILURE_RESULT="+quota,
		"FAKEACP_HOLD_FILE="+holdFile,
		"FAKEACP_PROMPT_LOG="+promptLog)
	settled, calls := recordQuotaSettlement(c)
	ctx := context.Background()
	if err := c.SendPrompt(ctx, h.AgentID, "first"); err != nil {
		t.Fatal(err)
	}
	waitForChunk(t, ch, "working")
	if held, err := c.SendPromptOrHold(ctx, h.AgentID, "held successor"); err != nil || !held {
		t.Fatalf("hold successor = %v, %v", held, err)
	}
	if err := os.WriteFile(holdFile, []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	events := drainTurn(t, ch)
	var ends []TurnEndData
	for _, ev := range events {
		if ev.Type == EvTurnEnd {
			var td TurnEndData
			if err := json.Unmarshal(ev.Data, &td); err != nil {
				t.Fatal(err)
			}
			ends = append(ends, td)
		}
	}
	if len(ends) != 1 || ends[0].StopReason != "error" || ends[0].Quota == nil || ends[0].Quota.FailureID != "turn-1:error" {
		as.mu.Lock()
		failure, cap := as.turnFailure, as.caps.SessionFailure
		as.mu.Unlock()
		t.Fatalf("terminal quota events = %+v, failure=%+v capability=%v, want one quota error", ends, failure, cap)
	}
	settled.Lock()
	if len(*calls) != 1 {
		settled.Unlock()
		t.Fatalf("settlement calls = %+v, want one", *calls)
	}
	call := (*calls)[0]
	settled.Unlock()
	if call.agentID != h.AgentID || call.generation != as.generation || call.turnID != events[len(events)-1].TurnID || call.quota == nil || call.quota.FailureID != "turn-1:error" || call.successful {
		t.Fatalf("settlement = %+v; want exact agent/generation/turn and quota before terminal event", call)
	}
	if promptCount := len(promptTexts(t, promptLog)); promptCount != 1 {
		t.Fatalf("provider prompt count = %d, want held successor suppressed", promptCount)
	}
	as.mu.Lock()
	blocked, retained := as.quotaBlocked, as.held.Text
	as.mu.Unlock()
	if !blocked || retained != "held successor" {
		t.Fatalf("quota block/held text = %v/%q, want blocked with retained successor", blocked, retained)
	}
}

func TestQuotaFailureIsNotInferredFromWarningOrRetry(t *testing.T) {
	for _, tc := range []struct {
		name    string
		failure string
	}{
		{"warning", quotaFailure("turn-1:error", 1, "limit", "warning")},
		{"retry", quotaFailure("turn-1:error", 1, "limit", "error", "retry")},
		{"new session", quotaFailure("turn-1:error", 1, "limit", "error", "new_session")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, h, ch, as := startQuotaTestAgent(t, "stream_text", "FAKEACP_FAILURE_UPDATE="+tc.failure)
			if err := c.SendPrompt(context.Background(), h.AgentID, "hello"); err != nil {
				t.Fatal(err)
			}
			events := drainTurn(t, ch)
			var end TurnEndData
			for _, ev := range events {
				if ev.Type == EvTurnEnd {
					_ = json.Unmarshal(ev.Data, &end)
				}
			}
			if end.StopReason != "end_turn" || end.Quota != nil {
				t.Fatalf("turn_end = %+v, want ordinary success", end)
			}
			as.mu.Lock()
			blocked := as.quotaBlocked
			as.mu.Unlock()
			if blocked {
				t.Fatal("warning/retry policy set quota block")
			}
		})
	}
}

func TestQuotaUpdateClearedBySuccessfulResultWithoutMetadata(t *testing.T) {
	quota := quotaFailure("recovering:error", 1, "limit", "error")
	c, h, ch, as := startQuotaTestAgent(t, "stream_text", "FAKEACP_FAILURE_UPDATE="+quota)
	if err := c.SendPrompt(context.Background(), h.AgentID, "retry after recovery"); err != nil {
		t.Fatal(err)
	}
	var end TurnEndData
	for _, ev := range drainTurn(t, ch) {
		if ev.Type == EvTurnEnd {
			if err := json.Unmarshal(ev.Data, &end); err != nil {
				t.Fatal(err)
			}
		}
	}
	if end.StopReason != "end_turn" || end.Quota != nil {
		t.Fatalf("successful result retained old quota update: %+v", end)
	}
	as.mu.Lock()
	defer as.mu.Unlock()
	if as.turnFailure != nil || as.turnQuota != nil {
		t.Fatalf("success left stale active failure: %+v / %+v", as.turnFailure, as.turnQuota)
	}
}

// ASRG-3, FS-03.R9, INV §5/§15 — a claimed Cancel terminates its turn as
// cancelled even when the provider still answers end_turn afterwards. The late
// completion used to settle as successful quota recovery and clear the block.
func TestCancelledTurnLateCompletionDoesNotConfirmQuotaRecovery(t *testing.T) {
	holdFile := filepath.Join(t.TempDir(), "release")
	c, h, ch, as := startQuotaTestAgent(t, "hold_turn",
		"FAKEACP_HOLD_FILE="+holdFile, "FAKEACP_HOLD_IGNORE_CANCEL=1")
	c.cancelGrace = 0
	mu, calls := recordQuotaSettlement(c)
	if err := c.SendPrompt(context.Background(), h.AgentID, "retry after quota"); err != nil {
		t.Fatal(err)
	}
	waitForChunk(t, ch, "working")
	as.mu.Lock()
	as.quotaBlocked = true
	as.mu.Unlock()
	if ok, err := c.Cancel(context.Background(), h.AgentID); err != nil || !ok {
		t.Fatalf("Cancel = %v, %v; want the active turn claimed", ok, err)
	}
	if err := os.WriteFile(holdFile, []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	var ends []TurnEndData
	for _, ev := range drainTurn(t, ch) {
		if ev.Type == EvTurnEnd {
			var td TurnEndData
			if err := json.Unmarshal(ev.Data, &td); err != nil {
				t.Fatal(err)
			}
			ends = append(ends, td)
		}
	}
	if len(ends) != 1 || ends[0].StopReason != "cancelled" {
		t.Fatalf("terminal events = %+v; want one cancelled turn_end", ends)
	}
	mu.Lock()
	if len(*calls) != 1 || (*calls)[0].successful {
		mu.Unlock()
		t.Fatalf("settlement calls = %+v; want one unsuccessful settlement", *calls)
	}
	mu.Unlock()
	as.mu.Lock()
	blocked := as.quotaBlocked
	as.mu.Unlock()
	if !blocked {
		t.Fatal("a cancelled late completion cleared the quota block")
	}
}

func TestQuotaStaleCompletionCannotSettleAfterTurnChanges(t *testing.T) {
	c, h, _, as := startQuotaTestAgent(t, "stream_text")
	mu, calls := recordQuotaSettlement(c)
	as.mu.Lock()
	as.turnActive = true
	as.execTurnID = "new-turn"
	as.mu.Unlock()
	c.completePromptTurn(as, "old-turn", "", json.RawMessage(`{"stopReason":"end_turn"}`), nil)
	mu.Lock()
	defer mu.Unlock()
	if len(*calls) != 0 {
		t.Fatalf("stale completion settlement calls=%+v", *calls)
	}
	if _, err := c.store.GetQuota(h.AgentID); !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("stale callback wrote quota state: %v", err)
	}
}

func TestRepairedQuotaCompletionUnblocksOnlyItsExactTuple(t *testing.T) {
	for _, tc := range []struct {
		name        string
		completed   string
		blocked     string
		wantStarted bool
	}{
		{name: "matching repaired completion", completed: "repaired-turn", blocked: "repaired-turn", wantStarted: true},
		{name: "older completion", completed: "old-turn", blocked: "newer-failed-write"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, h, ch, as := startQuotaTestAgent(t, "stream_text")
			as.generation = "runtime-generation"
			now := time.Now().UTC()
			q, err := c.store.ObserveQuota(state.QuotaObservation{
				AgentID: h.AgentID, EpisodeID: "episode-" + tc.name, SourceGeneration: "source-generation",
				ExecutingTurn: "source-turn", Owner: state.QuotaOwner{Kind: "ordinary"},
				ObservedAt: now, State: state.QuotaUnknownReset,
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := c.store.DB().Exec(`UPDATE quota_intents SET state=?, manual_generation=?, manual_turn=? WHERE agent_id=? AND episode_id=?`, state.QuotaContinued, as.generation, tc.completed, h.AgentID, q.EpisodeID); err != nil {
				t.Fatal(err)
			}
			q, err = c.store.GetQuota(h.AgentID)
			if err != nil || q.State != state.QuotaContinued {
				t.Fatalf("completed quota receipt: %+v err=%v", q, err)
			}
			as.mu.Lock()
			as.quotaBlocked = true
			as.quotaBlockedGeneration = as.generation
			as.quotaBlockedTurnID = tc.blocked
			as.mu.Unlock()
			if _, err := c.store.InsertMessage(state.Message{FromAgent: h.AgentID, FromAddress: "test", ToAgent: h.AgentID, Subject: "Notice", Body: "wake"}); err != nil {
				t.Fatal(err)
			}
			started, err := c.StartActivation(context.Background(), h.AgentID, "mail", func(turn string) error {
				return c.store.ResetTurnBudget(h.AgentID, turn)
			})
			if err != nil || started != tc.wantStarted {
				t.Fatalf("StartActivation = %v, %v, want started=%v", started, err, tc.wantStarted)
			}
			as.mu.Lock()
			blocked := as.quotaBlocked
			blockedTurn := as.quotaBlockedTurnID
			as.mu.Unlock()
			if tc.wantStarted && blocked || !tc.wantStarted && (!blocked || blockedTurn != tc.blocked) {
				t.Fatalf("quota block after activation: blocked=%v turn=%q", blocked, blockedTurn)
			}
			if tc.wantStarted {
				drainTurn(t, ch)
			}
		})
	}
}

func TestQuotaActivationLeavesInlineMailUnread(t *testing.T) {
	quota := quotaFailure("activation:error", 1, "limit", "error")
	c, h, ch, as := startQuotaTestAgent(t, "stream_text", "FAKEACP_FAILURE_UPDATE="+quota, "FAKEACP_FAILURE_RESULT="+quota)
	mu, calls := recordQuotaSettlement(c)
	if _, err := c.store.InsertMessage(state.Message{
		FromAgent: "a_sender", FromAddress: "reviewer@test", ToAgent: h.AgentID,
		Subject: "Review", Body: "Keep unread until quota recovery.",
	}); err != nil {
		t.Fatal(err)
	}
	var activationTurn string
	started, err := c.StartActivation(context.Background(), h.AgentID, "mail", func(turnID string) error {
		activationTurn = turnID
		return c.store.ResetTurnBudget(h.AgentID, turnID)
	})
	if err != nil || !started {
		t.Fatalf("StartActivation = %v, %v", started, err)
	}
	events := drainTurn(t, ch)
	var end TurnEndData
	for _, ev := range events {
		if ev.Type == EvTurnEnd {
			_ = json.Unmarshal(ev.Data, &end)
			if ev.TurnID != activationTurn {
				t.Fatalf("activation turn event owner = %q, want %q", ev.TurnID, activationTurn)
			}
		}
	}
	if end.StopReason != "error" || end.Quota == nil {
		t.Fatalf("activation turn_end = %+v, want quota error", end)
	}
	if unread, err := c.store.UnreadCount(h.AgentID); err != nil || unread != 1 {
		t.Fatalf("unread inline mail = %d, %v; want retained", unread, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*calls) != 1 || (*calls)[0].turnID != activationTurn || (*calls)[0].generation != as.generation {
		t.Fatalf("activation settlement = %+v, want exact activation owner", *calls)
	}
}

func TestQuotaNotificationIgnoredForReplayAndChildScope(t *testing.T) {
	c, h, _, as := startQuotaTestAgent(t, "stream_text")
	as.setExecTurn("root-turn")
	as.mu.Lock()
	as.turnActive = true
	as.loadReplay = true
	as.mu.Unlock()
	quota := quotaFailure("failure-1", 1, "limit", "error")
	params := []byte(`{"sessionId":"` + as.sessionID + `","update":{"sessionUpdate":"session_info_update","_meta":{"jetbrains":{"air":{"version":1,"sessionFailure":` + quota + `}}}}}`)
	c.onNotification(as, "session/update", params)
	as.mu.Lock()
	if as.turnFailure != nil || as.turnQuota != nil {
		as.mu.Unlock()
		t.Fatalf("replay captured quota: failure=%+v quota=%+v", as.turnFailure, as.turnQuota)
	}
	as.loadReplay = false
	as.caps.Subagents = true
	as.mu.Unlock()
	spawn, _ := json.Marshal(map[string]any{"sessionId": as.sessionID, "update": map[string]any{
		"sessionUpdate": "subagent_spawned", "subagentSessionId": "child-session",
	}})
	if !c.onSubagentUpdate(as, spawn) {
		t.Fatal("child spawn was not handled")
	}
	childParams := []byte(strings.Replace(string(params), as.sessionID, "child-session", 1))
	c.onNotification(as, "session/update", childParams)
	as.mu.Lock()
	defer as.mu.Unlock()
	if as.turnFailure != nil || as.turnQuota != nil {
		t.Fatalf("child activity captured root quota: failure=%+v quota=%+v (agent %s)", as.turnFailure, as.turnQuota, h.AgentID)
	}
}

func TestQuotaSameIDRevisionResultDoesNotOverrideNewerUpdate(t *testing.T) {
	update := quotaFailure("same:error", 2, "service", "warning")
	result := quotaFailure("same:error", 1, "limit", "error")
	c, h, ch, as := startQuotaTestAgent(t, "stream_text",
		"FAKEACP_FAILURE_UPDATE="+update,
		"FAKEACP_FAILURE_RESULT="+result)
	if err := c.SendPrompt(context.Background(), h.AgentID, "hello"); err != nil {
		t.Fatal(err)
	}
	events := drainTurn(t, ch)
	var end TurnEndData
	for _, ev := range events {
		if ev.Type == EvTurnEnd {
			_ = json.Unmarshal(ev.Data, &end)
		}
	}
	if end.Quota != nil || end.StopReason != "end_turn" {
		t.Fatalf("turn result = %+v, want newer warning revision to supersede older quota", end)
	}
	as.mu.Lock()
	failure := as.turnFailure
	as.mu.Unlock()
	if failure == nil || failure.ID != "same:error" || failure.Revision != 2 || failure.Quota {
		t.Fatalf("retained failure = %+v, want non-quota revision 2", failure)
	}
}

func TestQuotaEvidenceCarriesNoProviderText(t *testing.T) {
	result := `{"stopReason":"end_turn","_meta":{"jetbrains":{"air":{"version":1,"sessionFailure":{"id":"quota-1","revision":3,"category":"limit","severity":"error","title":"provider secret","actions":[]}}}}}`
	td, _ := mapPromptResult(json.RawMessage(result))
	raw, err := json.Marshal(td)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "provider secret") || strings.Contains(string(raw), "reset") {
		t.Fatalf("turn quota evidence exposed provider text or reset: %s", raw)
	}
	if !strings.Contains(string(raw), `"failure_id":"quota-1"`) || !strings.Contains(string(raw), `"revision":3`) {
		t.Fatalf("turn quota evidence missing correlation fields: %s", raw)
	}
}

func TestQuotaSettlementCarriesCorroboratedProviderReset(t *testing.T) {
	reset := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Minute)
	if reset.Minute() == 0 {
		reset = reset.Add(time.Minute) // the pinned formatter omits ":00"
	}
	text := "You've hit your weekly limit · resets " + strings.Replace(reset.Format("Jan 2, 2006, 3:04PM"), "PM", "pm", 1)
	text = strings.Replace(text, "AM", "am", 1) + " (UTC)"
	failure, _ := json.Marshal(map[string]any{"id": "weekly:error", "revision": 1, "category": "limit", "severity": "error", "title": text, "actions": []string{}})
	structured := reset.Add(17 * time.Second)
	rateLimit, _ := json.Marshal(map[string]any{"status": "rejected", "rateLimitType": "seven_day", "resetsAt": structured.Unix()})
	c, h, ch, _ := startQuotaTestAgent(t, "stream_text",
		"FAKEACP_RATE_LIMIT="+string(rateLimit), "FAKEACP_FAILURE_UPDATE="+string(failure), "FAKEACP_FAILURE_RESULT="+string(failure))
	mu, calls := recordQuotaSettlement(c)
	if err := c.SendPrompt(context.Background(), h.AgentID, "hello"); err != nil {
		t.Fatal(err)
	}
	for _, ev := range drainTurn(t, ch) {
		if ev.Type == EvTurnEnd && strings.Contains(string(ev.Data), "reset") {
			t.Fatalf("turn_end exposes reset provenance: %s", ev.Data)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*calls) != 1 || (*calls)[0].quota == nil || (*calls)[0].quota.ResetAt == nil || !(*calls)[0].quota.ResetAt.Equal(structured) {
		t.Fatalf("settlement = %+v, want structured reset %s corroborated by text %q", *calls, structured, text)
	}
}

// Q5/Q6 through ordinary settlement: windows that do not apply to this turn's
// model, an applicable window without a time, and conflicting text beside a
// structured reset all leave the reset unknown; a matching model window is used.
func TestQuotaSettlementKeepsResetApplicability(t *testing.T) {
	future := time.Now().UTC().Add(3 * time.Hour).Truncate(time.Second)
	window := func(kind string, at *time.Time) string {
		info := map[string]any{"status": "rejected", "rateLimitType": kind}
		if at != nil {
			info["resetsAt"] = at.Unix()
		}
		b, _ := json.Marshal(info)
		return string(b)
	}
	clauses := "You've hit your limit · resets " + strings.ToLower(future.Format("3PM")) + " (UTC) · resets " +
		strings.ToLower(future.Add(time.Hour).Format("3PM")) + " (UTC)"
	conflicting, _ := json.Marshal(map[string]any{"id": "c:error", "revision": 1, "category": "limit", "severity": "error", "title": clauses, "actions": []string{}})
	for _, tc := range []struct {
		name string
		env  []string
		want *time.Time
	}{
		{"unstamped opus window", []string{"FAKEACP_RATE_LIMIT=" + window("seven_day_opus", &future)}, nil},
		{"another model's window", []string{"FAKEACP_RATE_LIMIT=" + window("seven_day_opus", &future), "FAKEACP_RATE_MODEL=claude-sonnet-4-6"}, nil},
		{"matching model window", []string{"FAKEACP_RATE_LIMIT=" + window("seven_day_sonnet", &future), "FAKEACP_RATE_MODEL=claude-sonnet-4-6"}, &future},
		{"untimed blocker", []string{"FAKEACP_RATE_LIMIT=" + window("seven_day", nil)}, nil},
		{"conflicting text", []string{"FAKEACP_RATE_LIMIT=" + window("five_hour", &future), "FAKEACP_FAILURE_RESULT=" + string(conflicting)}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := append([]string{"FAKEACP_FAILURE_RESULT=" + quotaFailure("sonnet:error", 1, "limit", "error")}, tc.env...)
			c, h, ch, _ := startQuotaTestAgent(t, "stream_text", env...)
			mu, calls := recordQuotaSettlement(c)
			if err := c.SendPrompt(context.Background(), h.AgentID, "hello"); err != nil {
				t.Fatal(err)
			}
			drainTurn(t, ch)
			mu.Lock()
			defer mu.Unlock()
			if len(*calls) != 1 || (*calls)[0].quota == nil {
				t.Fatalf("settlement = %+v", *calls)
			}
			got := (*calls)[0].quota.ResetAt
			if tc.want == nil && got != nil || tc.want != nil && (got == nil || !got.Equal(*tc.want)) {
				t.Fatalf("reset = %v, want %v", got, tc.want)
			}
		})
	}
}
