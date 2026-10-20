package runtime

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/backend"
)

func steerArtifactAgent(t *testing.T, verified bool) (*ChatRuntime, *Handle, string) {
	t.Helper()
	c, spec := newChatTest(t, "hold_turn")
	c.steerProof = func(backend.BackendAdapter) bool { return verified }
	dir := t.TempDir()
	steerLog := filepath.Join(dir, "steer.log")
	spec.Env = append(spec.Env, "FAKEACP_HOLD_FILE="+filepath.Join(dir, "hold"), "FAKEACP_STEERING=1", "FAKEACP_STEER_LOG="+steerLog)
	h, err := c.Start(context.Background(), spec)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = c.Stop(context.Background(), h.AgentID) })
	return c, h, steerLog
}

// An adapter that advertises steering but lacks artifact proof offers no Steer
// and refuses it before any adapter call; ordinary Send still works (TS-04.R101).
func TestSteerWithheldWithoutArtifactProof(t *testing.T) {
	c, h, steerLog := steerArtifactAgent(t, false)
	if r, err := c.store.ReadRunning(h.AgentID); err != nil || r.SteeringAvailable {
		t.Fatalf("steering_available = %v err %v, want false", r.SteeringAvailable, err)
	}
	if _, err := c.Steer(context.Background(), h.AgentID, "late"); !errors.Is(err, ErrSteeringUnsupported) {
		t.Fatalf("Steer = %v, want ErrSteeringUnsupported", err)
	}
	if got := promptTexts(t, steerLog); len(got) != 0 {
		t.Fatalf("adapter received steering %v, want none", got)
	}
	if err := c.SendPrompt(context.Background(), h.AgentID, "ordinary"); err != nil {
		t.Fatalf("SendPrompt: %v", err)
	}
}

func TestSteerOfferedWithArtifactProof(t *testing.T) {
	c, h, _ := steerArtifactAgent(t, true)
	if r, err := c.store.ReadRunning(h.AgentID); err != nil || !r.SteeringAvailable {
		t.Fatalf("steering_available = %v err %v, want true", r.SteeringAvailable, err)
	}
}
