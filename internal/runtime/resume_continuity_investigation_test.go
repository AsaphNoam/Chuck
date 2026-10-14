package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// FS-01.A7/A8: a generic native-load rejection currently produces an apparently successful
// fresh session and retains the old usage reading. The observed behavior was
// reproduced separately before recording this skipped regression (HANDOFF C4).
func TestResumeNativeHistoryContinuityInvestigation(t *testing.T) {
	t.Skip("continuity investigation C4: native-load rejection silently replaces the provider session")
	c, spec := newChatTest(t, "stream_text")
	ctx := context.Background()
	newDump := filepath.Join(t.TempDir(), "new_params.json")
	spec.Env = append(spec.Env,
		`FAKEACP_LOAD_REJECT_DATA={"detail":"missing rollout"}`,
		"FAKEACP_NEW_DUMP="+newDump,
	)
	spec.LastContextPct = .73
	spec.LastContext = &state.ContextCounts{ContextUsed: 146000, ContextSize: 200000}
	h, err := c.Resume(ctx, spec, "prior-session-id")
	if err != nil {
		// Failing closed retains the saved native identity for an explicit retry.
		return
	}
	t.Cleanup(func() { _ = c.Stop(ctx, h.AgentID) })
	if _, err := os.Stat(newDump); err != nil {
		t.Fatalf("fixture did not exercise session/new fallback: %v", err)
	}
	if h.SessionID != "prior-session-id" {
		t.Errorf("Resume reported success with replacement session %q and no history recovery result", h.SessionID)
	}
	status, err := c.store.ReadStatus(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if h.SessionID != "prior-session-id" && (status.ContextPct != 0 || status.ContextCounts != nil) {
		t.Errorf("fresh session reports the old provider's usage: pct=%v counts=%+v", status.ContextPct, status.ContextCounts)
	}
}
