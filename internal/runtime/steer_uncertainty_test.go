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

// EXEC-9, FS-02.R3, TS-01.R44/R45, INV §5: a Steer to an idle agent reserves the
// turn gate while the native call is in flight, but no turn has begun. A tool
// frame streamed in that window must not write busy, because releasing the
// reservation has no turn end to restore idle.
func TestIdleSteerReservationNeverWritesBusy(t *testing.T) {
	for _, acknowledged := range []bool{false, true} {
		name := "lost acknowledgement"
		if acknowledged {
			name = "injected"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			ack, steerLog := filepath.Join(dir, "ack"), filepath.Join(dir, "steer.log")
			c, spec := newChatTest(t, "stream_text")
			spec.Env = append(spec.Env, "FAKEACP_STEERING=1", "FAKEACP_STEER_LOG="+steerLog, "FAKEACP_STEER_ACK_FILE="+ack)
			h, err := c.Start(context.Background(), spec)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = c.Stop(context.Background(), h.AgentID) })
			as, err := c.lookup(h.AgentID)
			if err != nil {
				t.Fatal(err)
			}
			if got := readState(t, c, h.AgentID); got != "idle" {
				t.Fatalf("status before the Steer = %q, want idle", got)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := c.Steer(ctx, h.AgentID, "nudge"); done <- err }()
			waitForPrompts(t, steerLog, 1)
			c.mapScoped(as, activityScope{}, "tool_call", toolCallFrame("background", `{"file_path":"a.go"}`))
			if got := readState(t, c, h.AgentID); got != "idle" {
				t.Errorf("status while the Steer is in flight = %q, want idle", got)
			}
			if acknowledged {
				if err := os.WriteFile(ack, []byte("go"), 0o600); err != nil {
					t.Fatal(err)
				}
			} else {
				cancel()
			}
			if err := <-done; acknowledged != (err == nil) {
				t.Fatalf("Steer error = %v with acknowledged=%v", err, acknowledged)
			}
			as.mu.Lock()
			active := as.turn.turnActive
			as.mu.Unlock()
			if active {
				t.Fatal("the Steer reservation still holds the turn gate")
			}
			if got := readState(t, c, h.AgentID); got != "idle" {
				t.Fatalf("status after the Steer released its reservation = %q, want idle", got)
			}
		})
	}
}
