package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

func TestQuotaProgressCommitsNoticeBeforeTurnEnds(t *testing.T) {
	release := filepath.Join(t.TempDir(), "release")
	t.Setenv("FAKEACP_HOLD_FILE", release)
	s, ts, log := activationTestServer(t)
	t.Setenv("FAKEACP_SCENARIO", "hold_turn")
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	due := time.Now().UTC().Add(-time.Second)
	q, err := s.stateStore.ObserveQuota(state.QuotaObservation{
		AgentID: id, EpisodeID: "live-notice", SourceGeneration: s.registry.Generation(id),
		ExecutingTurn: "interrupted-turn", Owner: state.QuotaOwner{Kind: "ordinary"},
		ObservedAt: due.Add(-2 * time.Minute), ResetAt: quotaResetFor(due), DueAt: &due, State: state.QuotaScheduled,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.WriteFile(release, nil, 0o600) }()
	s.startQuotaContinuation(context.Background(), q)
	waitPrompts(t, log, 1)
	q = waitQuota(t, s, id, func(q state.QuotaIntent) bool { return q.NoticeConfirmed })
	if q.State != state.QuotaClaimed || q.SuccessConfirmed {
		t.Fatalf("live progress settled the turn unexpectedly: %+v", q)
	}
}
