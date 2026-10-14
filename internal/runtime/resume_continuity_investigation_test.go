package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// FS-01.A7/A8: native-load rejection must fail without replacing native history.
func TestResumeNativeHistoryContinuityInvestigation(t *testing.T) {
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
	if err == nil || h != nil || !strings.Contains(err.Error(), "session/load") {
		t.Fatalf("Resume = %+v, %v; want native load failure", h, err)
	}
	if _, err := os.Stat(newDump); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("session/new replaced rejected native history: %v", err)
	}
	if _, err := c.lookup(spec.Agent.AgentID); !errors.Is(err, ErrNoHandle) {
		t.Fatalf("failed restore registered a runtime: %v", err)
	}
}

func TestResumeContextUsageMatchesRestoredOrFreshSession(t *testing.T) {
	for _, sessionID := range []string{"prior-session-id", ""} {
		t.Run(sessionID, func(t *testing.T) {
			c, spec := newChatTest(t, "stream_text")
			spec.Env = append(spec.Env, "FAKEACP_LOAD_EMPTY=1")
			spec.LastContextPct = .73
			spec.LastContext = &state.ContextCounts{ContextUsed: 146000, ContextSize: 200000}
			h, err := c.Resume(context.Background(), spec, sessionID)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = c.Stop(context.Background(), h.AgentID) })
			status, err := c.store.ReadStatus(h.AgentID)
			if err != nil {
				t.Fatal(err)
			}
			if sessionID == "" {
				if status.ContextPct != 0 || status.ContextCounts != nil {
					t.Fatalf("fresh context retained usage: %+v", status)
				}
			} else if status.ContextPct != .73 || status.ContextCounts == nil || *status.ContextCounts != *spec.LastContext {
				t.Fatalf("restored context lost usage: %+v", status)
			}
		})
	}
}
