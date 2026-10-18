package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func openToolInputs(as *agentState) int {
	as.mu.Lock()
	defer as.mu.Unlock()
	return len(as.toolInputs)
}

func toolCallFrame(id, args string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"sessionId":"s","update":{"sessionUpdate":"tool_call","toolCallId":%q,"kind":"edit","rawInput":%s}}`, id, args))
}

// TS-04.R98, INV §16: a streamed call interrupted by Cancel can end its turn
// without a result. The turn ending releases its input, so repeated canceled
// turns retain nothing.
func TestCanceledTurnsReleaseToolInput(t *testing.T) {
	c, h, _, ch := startPermAgent(t, false, "")
	as, err := c.lookup(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := c.SendPrompt(context.Background(), h.AgentID, "run tool"); err != nil {
			t.Fatal(err)
		}
		waitForEvent(t, ch, EvPermissionRequest)
		c.mapScoped(as, activityScope{}, "tool_call", toolCallFrame(fmt.Sprintf("interrupted-%d", i), `{"new_text":"retained body"}`))
		if n := openToolInputs(as); n != 1 {
			t.Fatalf("turn %d: open inputs = %d, want the live call", i+1, n)
		}
		if _, err := c.Cancel(context.Background(), h.AgentID); err != nil {
			t.Fatal(err)
		}
		waitForEvent(t, ch, EvTurnEnd)
		if n := openToolInputs(as); n != 0 {
			t.Fatalf("after canceled turn %d: retained %d input bodies, want 0", i+1, n)
		}
	}
}

// A turn that failed before settling is released when the next turn starts;
// a background continuation (no owning turn) and a live child activity keep
// their calls until their result or the child ends.
func TestToolInputOwnershipRelease(t *testing.T) {
	as := &agentState{toolNames: map[string]string{}}
	admit := func(scope activityScope, id string) {
		for _, m := range mapSessionUpdate(toolCallFrame(id, `{"file_path":"a.go"}`)) {
			as.admitToolInput(scope, scopeMapped(scope, m))
		}
	}
	child := activityScope{ActivityID: "act_child", TurnID: "A"}
	as.setExecTurn("A")
	admit(activityScope{}, "root-a")
	admit(child, "child-a")
	as.setExecTurn("")
	admit(activityScope{}, "background")
	as.setExecTurn("B")
	as.mu.Lock()
	_, rootOpen := as.toolInputs["root-a"]
	_, childOpen := as.toolInputs[child.toolCallID("child-a")]
	_, bgOpen := as.toolInputs["background"]
	as.releaseActivityToolInputsLocked(child.ActivityID)
	_, childAfter := as.toolInputs[child.toolCallID("child-a")]
	as.mu.Unlock()
	if rootOpen || !childOpen || !bgOpen || childAfter {
		t.Fatalf("root(A) open=%v child open=%v background open=%v child after end=%v", rootOpen, childOpen, bgOpen, childAfter)
	}
	for _, m := range mapSessionUpdate(json.RawMessage(`{"sessionId":"s","update":{"sessionUpdate":"tool_call_update","toolCallId":"background","status":"completed"}}`)) {
		as.admitToolInput(activityScope{}, m)
	}
	if n := openToolInputs(as); n != 0 || as.toolInputBytes != 0 {
		t.Fatalf("after background result: open=%d bytes=%d, want 0/0", n, as.toolInputBytes)
	}
}

// A terminal child state releases that child's calls through the live mapper.
func TestEndedChildReleasesToolInput(t *testing.T) {
	c, h, _, _ := startPermAgent(t, false, "")
	as, err := c.lookup(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	scope := activityScope{ActivityID: activityIDFor("child-session")}
	as.mu.Lock()
	as.children = map[string]activityScope{"child-session": scope}
	as.mu.Unlock()
	c.mapScoped(as, scope, "tool_call", toolCallFrame("c1", `{"file_path":"a.go"}`))
	if n := openToolInputs(as); n != 1 {
		t.Fatalf("open inputs = %d, want the child's call", n)
	}
	c.onSubagentUpdate(as, json.RawMessage(`{"sessionId":"s","update":{"sessionUpdate":"subagent_state_update","subagentSessionId":"child-session","state":"cancelled"}}`))
	if n := openToolInputs(as); n != 0 {
		t.Fatalf("after child stopped: open inputs = %d, want 0", n)
	}
}

// Active input retention is bounded by call count and bytes; the oldest call
// is forgotten first and the newest is kept even when it alone is large.
func TestOpenToolInputIsBounded(t *testing.T) {
	as := &agentState{toolNames: map[string]string{}}
	admit := func(id, args string) {
		for _, m := range mapSessionUpdate(toolCallFrame(id, args)) {
			as.admitToolInput(activityScope{}, m)
		}
	}
	for i := 0; i <= maxOpenToolInputs; i++ {
		admit(fmt.Sprintf("t%d", i), `{}`)
	}
	if _, ok := as.toolInputs["t0"]; ok || len(as.toolInputs) != maxOpenToolInputs {
		t.Fatalf("open=%d oldest retained=%v, want %d without t0", len(as.toolInputs), ok, maxOpenToolInputs)
	}
	big := `{"new_text":"` + strings.Repeat("x", maxOpenToolInputBytes) + `"}`
	admit("big1", big)
	admit("big2", big)
	if len(as.toolInputs) != 1 || as.toolInputs["big2"] == nil {
		t.Fatalf("open=%d, want only the newest oversized call", len(as.toolInputs))
	}
	if in := as.toolInputs["big2"]; as.toolInputBytes != len(in.title)+len(in.args) {
		t.Fatalf("bytes=%d, want only big2's %d", as.toolInputBytes, len(in.title)+len(in.args))
	}
}
