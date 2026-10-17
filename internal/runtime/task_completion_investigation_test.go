package runtime

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// TC-3, investigation-task-completion-2026-10-17: the Claude native continuation
// delivered tools after Chuck had already closed its session/prompt. A terminal
// tool notification must not leave an idle, non-cancellable runtime marked busy.
// This isolates the host status defect; it does not simulate native task wakeup.
func TestInvestigationTC3LateToolResultDoesNotReopenCompletedTurn(t *testing.T) {
	t.Skip("TC-3: reproduced on 2026-10-17; /fix should unskip after fencing out-of-turn status updates")
	c, spec := newChatTest(t, "tool_flow")
	ctx := context.Background()
	h, err := c.Start(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Stop(ctx, h.AgentID) })
	ch, unsubscribe, err := c.Subscribe(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	defer unsubscribe()
	if err := c.SendPrompt(ctx, h.AgentID, "edit the file"); err != nil {
		t.Fatal(err)
	}
	drainTurn(t, ch)
	as, err := c.lookup(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	// Wait for the completed turn's gate to settle, independently of status.
	deadline := time.Now().Add(time.Second)
	for {
		as.mu.Lock()
		active := as.turnActive
		as.mu.Unlock()
		if !active {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the initial turn has not settled")
		}
		time.Sleep(time.Millisecond)
	}
	c.onNotification(as, "session/update", json.RawMessage(`{"sessionId":"fake-sess-1","update":{"sessionUpdate":"tool_call_update","toolCallId":"tc_1","status":"completed"}}`))
	status, err := c.store.ReadStatus(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "idle" {
		t.Fatalf("completed runtime became %q (%s) after a terminal tool notification; want idle", status.State, status.Detail)
	}
}
