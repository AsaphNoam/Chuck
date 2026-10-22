package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPromotedSteerRejectsReplacedHeldSnapshot(t *testing.T) {
	steerLog := filepath.Join(t.TempDir(), "steer.log")
	c, h, _, _, _ := busyAgent(t, "FAKEACP_STEERING=1", "FAKEACP_STEER_LOG="+steerLog)
	if _, err := c.SendPromptOrHold(context.Background(), h.AgentID, "device A"); err != nil {
		t.Fatal(err)
	}
	old, _, err := c.HeldMessage(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.SendPromptOrHold(context.Background(), h.AgentID, "device B"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.SteerMessage(context.Background(), h.AgentID, Message{HeldID: old.HeldID}); !errors.Is(err, ErrHeldChanged) {
		t.Fatalf("stale promotion = %v; want held changed", err)
	}
	current, _, err := c.HeldMessage(h.AgentID)
	if err != nil || current.Text != "device B" {
		t.Fatalf("replacement = %+v %v", current, err)
	}
	if _, err := os.Stat(steerLog); !os.IsNotExist(err) {
		t.Fatalf("stale promotion reached the provider: %v", err)
	}
}

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
