package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// FS-03.R100: a consumed Steer with a lost acknowledgement must never become
// an automatically delivered prompt. The wire logs, rather than receipt state,
// are the oracle for duplicate consumption, including legacy id-less callers.
func TestPromotedSteerLostAcknowledgementNeverRequeues(t *testing.T) {
	for _, requestID := range []string{"", "text-receipt"} {
		t.Run("request="+requestID, func(t *testing.T) {
			ack := filepath.Join(t.TempDir(), "ack")
			steerLog := filepath.Join(t.TempDir(), "steer.log")
			c, h, _, release, promptLog := busyAgent(t, "FAKEACP_STEERING=1", "FAKEACP_STEER_LOG="+steerLog, "FAKEACP_STEER_ACK_FILE="+ack)
			if held, err := c.SendMessageOrHold(context.Background(), h.AgentID, Message{Text: "correction", RequestID: requestID}); err != nil || !held {
				t.Fatalf("hold: %v %v", held, err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() { _, err := c.Steer(ctx, h.AgentID, ""); done <- err }()
			waitForPrompts(t, steerLog, 1)
			cancel()
			if err := <-done; err == nil {
				t.Fatal("lost acknowledgement reported success")
			}
			if held, _, err := c.HeldMessage(h.AgentID); err != nil || !held.empty() {
				t.Fatalf("uncertain input automatically held: %+v %v", held, err)
			}
			if err := os.WriteFile(ack, []byte("late success"), 0o600); err != nil {
				t.Fatal(err)
			}
			release()
			waitForRuntimeStatus(t, c, h.AgentID, "idle")
			if got := promptTexts(t, promptLog); len(got) != 1 {
				t.Fatalf("provider prompts after late success = %v; want original only", got)
			}
		})
	}
}
