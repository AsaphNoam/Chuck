package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state/statetest"
)

func (as *agentState) setExecTurn(turnID string) {
	as.mu.Lock()
	defer as.mu.Unlock()
	as.setExecTurnLocked(turnID)
}

type receiptLog struct {
	mu sync.Mutex
	m  map[string]string
}

func trackReceipts(c *ChatRuntime) *receiptLog {
	r := &receiptLog{m: map[string]string{}}
	c.SetMessageOutcomeHandler(func(_ string, id, outcome string) error {
		r.mu.Lock()
		r.m[id] = outcome
		r.mu.Unlock()
		return nil
	})
	return r
}

func (r *receiptLog) get(id string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.m[id]
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// EXEC-1 (TS-01.R45, INV §19): Stop lands after beginTurn succeeded but before
// the provider send. Nothing was written to the provider, so the submitted
// receipt is released for recovery instead of staying submitted.
func TestStopBeforeSendReleasesSubmittedReceipt(t *testing.T) {
	c, spec := newChatTest(t, "stream_text")
	spec.Generation = "g"
	ctx := context.Background()
	h, err := c.Start(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	as, err := c.lookup(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	receipts := trackReceipts(c)
	as.cancelMu.Lock() // an in-flight Cancel holds the send
	held, err := c.SendMessageOrHold(ctx, h.AgentID, Message{Text: "A", RequestID: "req-a"})
	if err != nil || held {
		as.cancelMu.Unlock()
		t.Fatalf("send held=%v err=%v", held, err)
	}
	if got := receipts.get("req-a"); got != "submitted" {
		as.cancelMu.Unlock()
		t.Fatalf("receipt before Stop = %q, want submitted", got)
	}
	if err := c.Stop(ctx, h.AgentID); err != nil {
		as.cancelMu.Unlock()
		t.Fatal(err)
	}
	as.cancelMu.Unlock()
	waitFor(t, "released receipt", func() bool { return receipts.get("req-a") == "released" })
}

// EXEC-1 sibling (TS-01.R45, INV §19): Stop lands after preparation recorded the
// prompt submitted but before beginTurn's own start check. The begin refuses,
// and the receipt is released rather than left submitted with no recovery.
func TestStopAfterSubmittedBeforeBeginReleasesReceipt(t *testing.T) {
	c, spec := newChatTest(t, "stream_text")
	ctx := context.Background()
	h, err := c.Start(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	as, err := c.lookup(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	last := map[string]string{}
	stopped := make(chan error, 1)
	c.SetMessageOutcomeHandler(func(_ string, id, outcome string) error {
		mu.Lock()
		last[id] = outcome
		mu.Unlock()
		if id == "req-a" && outcome == "submitted" {
			go func() { stopped <- c.Stop(ctx, h.AgentID) }()
			waitFor(t, "Stop to begin", func() bool {
				as.mu.Lock()
				defer as.mu.Unlock()
				return as.stopping
			})
		}
		return nil
	})
	_, _ = c.SendMessageOrHold(ctx, h.AgentID, Message{Text: "A", RequestID: "req-a"})
	if err := <-stopped; err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if last["req-a"] != "released" {
		t.Fatalf("receipt after a refused begin = %q, want released", last["req-a"])
	}
}

// blockSettlement pauses the next turn's real completion after it claimed
// promptSettlement and before finishTurn.
func blockSettlement(c *ChatRuntime) (reached <-chan struct{}, release func()) {
	entered, gate := make(chan struct{}), make(chan struct{})
	var once sync.Once
	c.SetQuotaSettlement(func(string, string, string, *QuotaEvidence, bool) error {
		once.Do(func() { close(entered); <-gate })
		return nil
	})
	return entered, sync.OnceFunc(func() { close(gate) })
}

func turnEnds(events []Event) int {
	n := 0
	for _, ev := range events {
		if ev.Type == EvTurnEnd {
			n++
		}
	}
	return n
}

// EXEC-2 (TS-01.R45, TS-02.R67, INV §5/§15): the end transaction fails. The
// turn still emits turn_end and hands its gate to the held successor, so a
// newer Send cannot overtake it.
func TestFailedTurnEndWriteStillEndsTurnAndRunsHeld(t *testing.T) {
	c, spec := newChatTest(t, "stream_text")
	spec.Generation = "g"
	ctx := context.Background()
	h, err := c.Start(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Stop(ctx, h.AgentID) })
	as, err := c.lookup(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	ch, unsub, err := c.Subscribe(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	defer unsub()
	reached, release := blockSettlement(c)
	defer release()
	if err := c.SendPrompt(ctx, h.AgentID, "A"); err != nil {
		t.Fatal(err)
	}
	<-reached
	turn, err := c.store.ReadConversationTurn(h.AgentID, as.generation)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.store.DB().Exec(fmt.Sprintf(`CREATE TRIGGER fail_turn_end BEFORE UPDATE ON turn_budget
WHEN NEW.state='ended' AND OLD.turn_id='%s' BEGIN SELECT RAISE(ABORT, 'injected end failure'); END;`, turn.TurnID)); err != nil {
		t.Fatal(err)
	}
	if held, err := c.SendPromptOrHold(ctx, h.AgentID, "B"); err != nil || !held {
		t.Fatalf("B held=%v err=%v", held, err)
	}
	release()
	first := drainTurn(t, ch)
	if turnEnds(first) != 1 {
		t.Fatalf("A emitted %d turn_end events", turnEnds(first))
	}
	second := drainTurn(t, ch)
	var prompt UserPromptData
	for _, ev := range second {
		if ev.Type == EvUserPrompt {
			_ = json.Unmarshal(ev.Data, &prompt)
		}
	}
	if prompt.Text != "B" || turnEnds(second) != 1 {
		t.Fatalf("successor prompt=%q turn_ends=%d, want held B to run", prompt.Text, turnEnds(second))
	}
}

// EXEC-3 (TS-01.R45, INV §5): a completion that claimed its settlement keeps
// its end when Stop arrives before finishTurn: outcome completed, one turn_end.
func TestClaimedCompletionEndsTurnDespiteStop(t *testing.T) {
	c, spec := newChatTest(t, "stream_text")
	spec.Generation = "g"
	ctx := context.Background()
	h, err := c.Start(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	as, err := c.lookup(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	generation := as.generation
	ch, unsub, err := c.Subscribe(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	defer unsub()
	reached, release := blockSettlement(c)
	defer release()
	if err := c.SendPrompt(ctx, h.AgentID, "A"); err != nil {
		t.Fatal(err)
	}
	<-reached
	as.mu.Lock()
	settled := as.turn.promptSettlement
	as.mu.Unlock()
	if held, err := c.SendMessageOrHold(ctx, h.AgentID, Message{Text: "B", RequestID: "held-b"}); err != nil || !held {
		t.Fatalf("B held=%v err=%v", held, err)
	}
	// Stop releases the held message after marking itself stopping; parking it
	// there makes the completion run between Stop's start and its final end.
	stopParked, resumeStop := make(chan struct{}), make(chan struct{})
	c.SetMessageOutcomeHandler(func(_ string, id, outcome string) error {
		if id == "held-b" && outcome == "released" {
			close(stopParked)
			<-resumeStop
		}
		return nil
	})
	stopped := make(chan error, 1)
	go func() { stopped <- c.Stop(ctx, h.AgentID) }()
	<-stopParked
	release()
	<-settled
	close(resumeStop)
	if err := <-stopped; err != nil {
		t.Fatal(err)
	}
	var events []Event
	for ev := range ch {
		events = append(events, ev)
	}
	if n := turnEnds(events); n != 1 {
		t.Fatalf("turn_end events = %d, want 1", n)
	}
	turn, err := c.store.ReadConversationTurn(h.AgentID, generation)
	if err != nil {
		t.Fatal(err)
	}
	if turn.State != "ended" || turn.Outcome != "completed" {
		t.Fatalf("turn row = %+v, want ended/completed", turn)
	}
}

// assertChatStatusComposed is TS-01.R46's harness check. A live agent whose turn
// gate is still held is settling and skipped; every other chat agent's stored
// state must equal the composition of its durable inputs.
func assertChatStatusComposed(t *testing.T, c *ChatRuntime) {
	t.Helper()
	settling := func(agentID string) bool {
		as, err := c.lookup(agentID)
		if err != nil {
			return false
		}
		as.mu.Lock()
		defer as.mu.Unlock()
		return as.turn.turnActive || as.turn.promptSettlement != nil || as.stopping && !as.stopped
	}
	mismatches, err := statetest.ChatStatusMismatches(c.store, settling)
	if err != nil {
		t.Errorf("read status composition inputs: %v", err)
	}
	for _, m := range mismatches {
		t.Errorf("TS-01.R46 status composition: %s", m)
	}
}
