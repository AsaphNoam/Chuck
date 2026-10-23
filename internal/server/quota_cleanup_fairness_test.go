package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// ASRG-5 / TS-10.R47: a full batch of skipped or failing references cannot
// starve a later healthy exact-session cleanup.
func TestQuotaCleanupAdvancesPastBlockedBatch(t *testing.T) {
	s := testServer(t, true)
	s.registry.Chat().SetCommand(buildFakeACP(t))
	s.quotaCleanupTried = make(map[string]time.Time)
	s.quotaCleanupSpecs = make(map[string]runtime.QuotaCheckSpec)
	s.quotaChecks = make(map[string]bool)
	var last string
	for i := 0; i <= state.MaxQuotaDueBatch; i++ {
		id := fmt.Sprintf("cleanup_%02d", i)
		last = id
		if err := s.stateStore.WriteAgent(state.Agent{AgentID: id, Name: id, Role: "default", Project: "my-app",
			Backend: "codex", Model: "gpt", Interface: "chat", CreatedAt: time.Now().UTC()}); err != nil {
			t.Fatal(err)
		}
		q, err := s.stateStore.ObserveQuota(state.QuotaObservation{AgentID: id, EpisodeID: id, SourceGeneration: "source-generation",
			ExecutingTurn: "source-turn", Owner: state.QuotaOwner{Kind: "ordinary"},
			ObservedAt: time.Now().UTC().Add(-61 * time.Minute), State: state.QuotaUnknownReset, Check: true})
		if err != nil {
			t.Fatal(err)
		}
		if err := s.stateStore.ClaimQuota(id, q.EpisodeID, q.Revision, "token"); err != nil {
			t.Fatal(err)
		}
		q, _ = s.stateStore.GetQuota(id)
		if err := s.stateStore.StartQuotaCheck(id, q.EpisodeID, q.Revision, q.AttemptToken, "codex-acp"); err != nil {
			t.Fatal(err)
		}
		q, _ = s.stateStore.GetQuota(id)
		if err := s.stateStore.RecordQuotaCheckSession(id, q.EpisodeID, q.Revision, q.AttemptToken, id); err != nil {
			t.Fatal(err)
		}
		q, _ = s.stateStore.GetQuota(id)
		if err := s.stateStore.FinishQuotaCheck(id, q.EpisodeID, q.Revision, q.AttemptToken, state.QuotaCheckResult{Outcome: state.QuotaCheckFailed}); err != nil {
			t.Fatal(err)
		}
		switch {
		case i < 3:
			s.quotaChecks[id] = true
		case i < 6:
			s.quotaCleanupTried[id+"\x00"+id] = time.Now()
		case i < state.MaxQuotaDueBatch:
			// A provider failing initialization cannot resolve its recorded
			// reference; the later healthy selection still must run.
			s.quotaCleanupSpecs[id+"\x00"+id] = runtime.QuotaCheckSpec{BackendType: "codex-acp", Env: []string{"HOME=" + t.TempDir(), "FAKEACP_INIT_STDERR=cleanup unavailable"}}
		}
	}
	dir := t.TempDir()
	deleted := filepath.Join(dir, "deleted.json")
	s.quotaCleanupSpecs[last+"\x00"+last] = runtime.QuotaCheckSpec{BackendType: "codex-acp",
		Env: []string{"HOME=" + dir, "FAKEACP_SCENARIO=stream_text", "FAKEACP_CAPS=1", "FAKEACP_DELETE_LOG=" + deleted}}
	// First sweep visits exactly one bounded batch; second reaches the ninth.
	s.cleanupQuotaChecks(context.Background())
	if q, _ := s.stateStore.GetQuota(last); q.CheckSession == "" {
		t.Fatal("cleanup exceeded its batch")
	}
	s.cleanupQuotaChecks(context.Background())
	if q, err := s.stateStore.GetQuota(last); err != nil || q.CheckSession != "" {
		t.Fatalf("healthy cleanup starved: %+v err=%v", q, err)
	}
	if raw, err := os.ReadFile(deleted); err != nil || !strings.Contains(string(raw), `"`+last+`"`) {
		t.Fatalf("exact session not deleted: %s err=%v", raw, err)
	}
	// Exhaustion wraps the cursor; skipped references remain owned and retryable.
	s.cleanupQuotaChecks(context.Background())
	if s.quotaCleanupCursor != "" {
		t.Fatalf("cursor did not wrap: %q", s.quotaCleanupCursor)
	}
	if refs, err := s.stateStore.PendingQuotaCheckCleanups(state.MaxQuotaDueBatch); err != nil || len(refs) != state.MaxQuotaDueBatch {
		t.Fatalf("blocked references lost: %d err=%v", len(refs), err)
	}
}
