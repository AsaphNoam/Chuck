package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/AsaphNoam/Chuck/internal/backend"
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

// PERR-1/PERR-2: Codex ACP 2.1.1 returns exhausted 429 retries as end_turn with
// a terminal rate_limited AIR failure. Ordinary and activation turns both end
// as visible errors, settle unsuccessfully without account-quota evidence
// (TS-04.R86) and leave inline mail unread (FS-03.R8, FS-01.R44).
func TestTerminalProviderFailureEndsTurnAsError(t *testing.T) {
	const message = "exceeded retry limit, last status: 429 Too Many Requests"
	const failure = `{"id":"turn-429:error","revision":1,"category":"limit","severity":"error","title":"` + message + `","actions":["retry"]}`
	for _, activation := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "activation"}[activation], func(t *testing.T) {
			c, h, ch, as := startQuotaTestAgent(t, "stream_text", "FAKEACP_FAILURE_RESULT="+failure)
			mu, calls := recordQuotaSettlement(c)
			if _, err := c.store.InsertMessage(state.Message{
				FromAgent: "a_sender", FromAddress: "reviewer@test", ToAgent: h.AgentID,
				Subject: "Review", Body: "Keep unread until a successful turn.",
			}); err != nil {
				t.Fatal(err)
			}
			if activation {
				started, err := c.StartActivation(context.Background(), h.AgentID, "mail", func(turnID string) error {
					return c.store.ResetTurnBudget(h.AgentID, turnID)
				})
				if err != nil || !started {
					t.Fatalf("StartActivation = %v, %v", started, err)
				}
			} else if err := c.SendPrompt(context.Background(), h.AgentID, "answer this"); err != nil {
				t.Fatal(err)
			}
			var end TurnEndData
			visible := 0
			for _, ev := range drainTurn(t, ch) {
				switch ev.Type {
				case EvError:
					var data ErrorData
					if err := json.Unmarshal(ev.Data, &data); err != nil {
						t.Fatal(err)
					}
					if data.Message == message {
						visible++
					}
				case EvTurnEnd:
					if err := json.Unmarshal(ev.Data, &end); err != nil {
						t.Fatal(err)
					}
				}
			}
			if end.StopReason != "error" || end.Quota != nil || visible != 1 {
				t.Errorf("turn_end = %+v with %d visible errors; want one error without quota evidence", end, visible)
			}
			if status, err := c.store.ReadStatus(h.AgentID); err != nil || status.State != "error" {
				t.Errorf("status = %+v, %v; want error", status, err)
			}
			if unread, err := c.store.UnreadCount(h.AgentID); err != nil || unread != 1 {
				t.Errorf("unread inline mail = %d, %v; want retained", unread, err)
			}
			as.mu.Lock()
			blocked := as.quotaBlocked
			as.mu.Unlock()
			mu.Lock()
			defer mu.Unlock()
			if blocked || len(*calls) != 1 || (*calls)[0].successful || (*calls)[0].quota != nil {
				t.Errorf("settlement = %+v, quota block %v; want one non-quota unsuccessful settlement", *calls, blocked)
			}
		})
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

// ARCH-QUOTA-1, TS-04.R86: quota evidence arriving with or before the late
// completion of an accepted Cancel cannot replace cancelled or reach settlement.
func TestCancelledTurnIgnoresLateQuotaEvidence(t *testing.T) {
	q := quotaFailure("late:error", 1, "limit", "error")
	for _, carrier := range []string{"FAKEACP_FAILURE_RESULT", "FAKEACP_FAILURE_UPDATE"} {
		t.Run(carrier, func(t *testing.T) {
			hold := filepath.Join(t.TempDir(), "release")
			c, h, ch, _ := startQuotaTestAgent(t, "hold_turn", "FAKEACP_HOLD_FILE="+hold,
				"FAKEACP_HOLD_IGNORE_CANCEL=1", carrier+"="+q)
			c.cancelGrace = 0
			mu, calls := recordQuotaSettlement(c)
			if err := c.SendPrompt(context.Background(), h.AgentID, "continue"); err != nil {
				t.Fatal(err)
			}
			waitForChunk(t, ch, "working")
			if ok, err := c.Cancel(context.Background(), h.AgentID); err != nil || !ok {
				t.Fatalf("Cancel = %v, %v", ok, err)
			}
			if err := os.WriteFile(hold, []byte("go"), 0o600); err != nil {
				t.Fatal(err)
			}
			for _, ev := range drainTurn(t, ch) {
				if ev.Type == EvTurnEnd {
					var end TurnEndData
					if err := json.Unmarshal(ev.Data, &end); err != nil {
						t.Fatal(err)
					}
					if end.StopReason != "cancelled" || end.Quota != nil {
						t.Fatalf("cancelled turn became %+v; want cancelled without quota", end)
					}
				}
			}
			mu.Lock()
			defer mu.Unlock()
			if len(*calls) != 1 || (*calls)[0].quota != nil || (*calls)[0].successful {
				t.Fatalf("settlement calls = %+v; want one unsuccessful call without quota", *calls)
			}
		})
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

func TestCancelledQuotaEpisodeAllowsAutomaticActivationAfterBlockClear(t *testing.T) {
	c, h, ch, as := startQuotaTestAgent(t, "stream_text")
	as.mu.Lock()
	as.quotaBlocked = true
	as.quotaBlockedGeneration = as.generation
	as.quotaBlockedTurnID = "failed-turn"
	as.mu.Unlock()
	c.ClearQuotaBlock(h.AgentID, as.generation, "failed-turn")
	if _, err := c.store.InsertMessage(state.Message{FromAgent: "sender", FromAddress: "test", ToAgent: h.AgentID, Subject: "wake", Body: "wake"}); err != nil {
		t.Fatal(err)
	}
	started, err := c.StartActivation(context.Background(), h.AgentID, "mail", func(turn string) error {
		return c.store.ResetTurnBudget(h.AgentID, turn)
	})
	if err != nil || !started {
		t.Fatalf("StartActivation after cancelled quota = %v, %v; want started", started, err)
	}
	drainTurn(t, ch)
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

// nextJerusalem10pm is the incident's `10pm (Asia/Jerusalem)` resolved from
// now. The running 22:00 minute is ambiguous by contract, so it is skipped.
func nextJerusalem10pm(t *testing.T) time.Time {
	t.Helper()
	loc := mustZone(t, "Asia/Jerusalem")
	now := time.Now().In(loc)
	if now.Hour() == 22 && now.Minute() == 0 || now.Hour() == 21 && now.Minute() == 59 {
		t.Skip("observed at the reset minute boundary")
	}
	want := time.Date(now.Year(), now.Month(), now.Day(), 22, 0, 0, 0, loc)
	if !want.After(now) {
		want = want.AddDate(0, 0, 1)
	}
	return want.UTC()
}

// startRPCRefusalAgent starts a provider whose prompts end in a terminal
// -32603 RPC error with the given data and message, carrying no AIR failure.
func startRPCRefusalAgent(t *testing.T, backend string, air bool, data, message string) (*ChatRuntime, *Handle, <-chan Event, *agentState) {
	t.Helper()
	c, spec := newChatTest(t, "stream_text")
	spec.BackendType = backend
	spec.Env = append(spec.Env, "TZ=UTC", "FAKEACP_PROMPT_REJECT_DATA="+data, "FAKEACP_PROMPT_REJECT_MESSAGE="+message)
	if air {
		spec.Env = append(spec.Env, "FAKEACP_CAPS=1")
	}
	h, err := c.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = c.Stop(context.Background(), h.AgentID) })
	ch, unsub, err := c.Subscribe(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(unsub)
	as, err := c.lookup(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	return c, h, ch, as
}

// FS-01.A37/A39, TS-04.R105: a terminal session/prompt RPC refusal without
// AIR metadata enters quota recovery only through a verified provider carrier.
// The incident rows replay a_a7e830's captured message; its errorKind value
// was not recorded, so omitted and unrecognized values are both exercised.
// Codex carriers follow codex-acp 2.1.1 createTurnErrorData.
func TestTerminalRPCQuotaRefusalEntersRecovery(t *testing.T) {
	codexQuota := `{"message":"You’ve hit your usage limit. Try again at Oct 21st, 2099 9:00 AM.","codexErrorInfo":"usageLimitExceeded"}`
	for _, tc := range []struct {
		name, backend, data, message string
		air, quota                   bool
		reset                        string // "incident" is the next 10pm in Asia/Jerusalem
	}{
		{"incident, category not recorded", "claude-acp", `{}`, incidentText, false, true, "incident"},
		{"incident under negotiated AIR", "claude-acp", `{}`, incidentText, true, true, "incident"},
		{"unrecognized category uses quota grammar", "claude-acp", `{"errorKind":"future_kind"}`, incidentText, true, true, "incident"},
		{"billing category without text", "claude-acp", `{"errorKind":"billing_error"}`, "", true, true, ""},
		{"account on hold", "claude-acp", `{"errorKind":"account_on_hold"}`, "", false, true, ""},
		{"explicit throttling overrides matching text", "claude-acp", `{"errorKind":"rate_limit"}`, incidentText, true, false, ""},
		{"generic protocol error", "claude-acp", `{}`, "", true, false, ""},
		{"clock without quota grammar", "claude-acp", `{}`, "Internal error: request failed; retry at 22:00 (Asia/Jerusalem)", false, false, ""},
		{"oversized carrier", "claude-acp", `{"errorKind":"billing_error","pad":"` + strings.Repeat("x", maxSessionFailureBytes) + `"}`, "", false, false, ""},
		{"codex usage limit", "codex-acp", codexQuota, "", false, true, "2099-10-21T09:00:00Z"},
		{"codex generic throttling", "codex-acp", `{"message":"Too many requests","codexErrorInfo":{"responseTooManyFailedAttempts":{"httpStatusCode":429}}}`, "", false, false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var want time.Time
			switch tc.reset {
			case "":
			case "incident":
				want = nextJerusalem10pm(t)
			default:
				want = utc(tc.reset)
			}
			c, h, ch, as := startRPCRefusalAgent(t, tc.backend, tc.air, tc.data, tc.message)
			mu, calls := recordQuotaSettlement(c)
			if err := c.SendPrompt(context.Background(), h.AgentID, "continue the work"); err != nil {
				t.Fatal(err)
			}
			var end TurnEndData
			for _, ev := range drainTurn(t, ch) {
				if ev.Type == EvTurnEnd {
					_ = json.Unmarshal(ev.Data, &end)
				}
			}
			mu.Lock()
			defer mu.Unlock()
			if len(*calls) != 1 || ((*calls)[0].quota != nil) != tc.quota || (end.Quota != nil) != tc.quota || end.StopReason != "error" {
				t.Fatalf("settlement = %+v, turn_end = %+v; want quota=%v", *calls, end, tc.quota)
			}
			as.mu.Lock()
			blocked := as.quotaBlocked
			as.mu.Unlock()
			if blocked != tc.quota {
				t.Fatalf("quota block = %v, want %v", blocked, tc.quota)
			}
			if !tc.quota {
				return
			}
			if got := (*calls)[0].quota; got.FailureID != "rpc:"+(*calls)[0].turnID || got.Intervention ||
				tc.reset == "" && got.ResetAt != nil || tc.reset != "" && (got.ResetAt == nil || !got.ResetAt.Equal(want) || got.ResetSource != "text") {
				t.Fatalf("quota evidence = %+v (reset %v), want turn-bound failure with reset %v", got, got.ResetAt, want)
			}
		})
	}
}

// QUOTA-RPC-2, TS-04.R107/TS-10.R62: a time-only reset keeps the meaning it
// had when the terminal RPC error reached the transport. Settlement after the
// stated time (and here, before midnight) must not reread it as tomorrow.
func TestTerminalRPCResetResolvesAtArrivalNotSettlement(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		time.Sleep(21*time.Hour + 59*time.Minute + 59*time.Second) // bubble epoch is midnight UTC
		want := time.Now().UTC().Truncate(24 * time.Hour).Add(22 * time.Hour)
		reqR, reqW := io.Pipe()
		respR, respW := io.Pipe()
		tr := NewTransport(reqW, nil, nil)
		go func() { _ = tr.Run(respR) }()
		go func() {
			var req struct {
				ID int64 `json:"id"`
			}
			_ = json.NewDecoder(reqR).Decode(&req)
			fmt.Fprintf(respW, `{"jsonrpc":"2.0","id":%d,"error":{"code":-32603,"message":"Internal error: You've hit your session limit · resets 10pm (UTC)","data":{}}}`+"\n", req.ID)
		}()
		_, err := tr.Call(context.Background(), "session/prompt", map[string]any{})
		if err == nil {
			t.Fatal("prompt succeeded, want the provider refusal")
		}
		time.Sleep(61 * time.Second) // persistence/logging delay before settlement

		c := NewChatRuntime(nil)
		ad, _ := backend.For("claude-acp")
		as := &agentState{agentID: "a", generation: "g1", execTurnID: "t1", turnActive: true, adapter: ad, processZone: time.UTC}
		var got *QuotaEvidence
		stop := &struct{}{}
		c.SetQuotaSettlement(func(_, _, _ string, q *QuotaEvidence, _ bool) error {
			got = q
			panic(stop) // settlement is the boundary; the storeless turn ends here
		})
		func() {
			defer func() {
				if r := recover(); r != stop {
					panic(r)
				}
			}()
			c.completePromptTurn(as, "t1", "", json.RawMessage(`{"stopReason":"error"}`), err)
		}()
		if got == nil || got.ResetAt == nil || !got.ResetAt.Equal(want) {
			t.Fatalf("settled at %s: quota = %+v, want reset %s", time.Now().UTC(), got, want)
		}
		_ = respW.Close()
		_ = reqR.Close()
	})
}

// The shared classifier also settles reserved activations, and the same words
// in a user prompt never create a quota interruption (FS-01.A37).
func TestTerminalRPCQuotaRefusalSharedByActivationNotProse(t *testing.T) {
	c, h, ch, as := startRPCRefusalAgent(t, "claude-acp", true, `{}`, incidentText)
	mu, calls := recordQuotaSettlement(c)
	if _, err := c.store.InsertMessage(state.Message{FromAgent: "a_sender", FromAddress: "reviewer@test", ToAgent: h.AgentID,
		Subject: "Review", Body: "Keep unread until quota recovery."}); err != nil {
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
	drainTurn(t, ch)
	mu.Lock()
	if len(*calls) != 1 || (*calls)[0].turnID != activationTurn || (*calls)[0].generation != as.generation || (*calls)[0].quota == nil {
		mu.Unlock()
		t.Fatalf("activation settlement = %+v, want quota on the activation turn", *calls)
	}
	mu.Unlock()

	c2, h2, ch2, _ := startQuotaTestAgent(t, "stream_text")
	mu2, calls2 := recordQuotaSettlement(c2)
	if err := c2.SendPrompt(context.Background(), h2.AgentID, incidentText); err != nil {
		t.Fatal(err)
	}
	drainTurn(t, ch2)
	mu2.Lock()
	defer mu2.Unlock()
	if len(*calls2) != 1 || (*calls2)[0].quota != nil || !(*calls2)[0].successful {
		t.Fatalf("prose settlement = %+v, want ordinary success", *calls2)
	}
}
