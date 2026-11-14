package runtime

import (
	"context"
	"encoding/json"
	"syscall"
	"testing"
)

func TestCrashResolvesPendingProviderPermission(t *testing.T) {
	c, h, _, events := startPermAgent(t, false, "")
	history := recordHistory(c)
	if err := c.SendPrompt(context.Background(), h.AgentID, "run ls"); err != nil {
		t.Fatal(err)
	}
	request := waitForEvent(t, events, EvPermissionRequest)
	var pending PermissionRequestData
	if err := json.Unmarshal(request.Data, &pending); err != nil || pending.ToolCallID == "" {
		t.Fatalf("permission request: %s (%v)", request.Data, err)
	}
	if err := syscall.Kill(-h.Pid, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	waitForEvent(t, events, EvTurnEnd)
	for _, event := range history() {
		var resolved PermissionResolvedData
		if event.Type == EvPermissionResolved && json.Unmarshal(event.Data, &resolved) == nil &&
			resolved.ToolCallID == pending.ToolCallID && resolved.Decision == "cancelled" {
			return
		}
	}
	t.Fatalf("missing cancellation for request %s before turn end", pending.ToolCallID)
}
