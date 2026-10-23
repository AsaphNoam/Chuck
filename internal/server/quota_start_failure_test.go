package server

import (
	"context"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// ASRG-1 / TS-14.R6: failure after linked admission must retire exactly that
// attempt and capture without a prompt, contribution or completed allowance.
func TestQuotaRoomStartFailureAbandonsAttempt(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		stopped, failCompensation bool
	}{
		{name: "running"}, {name: "resumed", stopped: true}, {name: "compensation retry", failCompensation: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, ts, prompts := activationTestServer(t)
			a := launchAndWaitIdle(t, ts, "impl", "tmpproj")
			b := launchAndWaitIdle(t, ts, "impl", "tmpproj")
			room := createTestRoom(t, s, []string{a, b}, 2)
			d, err := s.stateStore.ReadThinkTank(room)
			if err != nil {
				t.Fatal(err)
			}
			next, ok := state.NextThinkTankOpportunity(d)
			if !ok {
				t.Fatal("no room opportunity")
			}
			id, gen := next.AgentID, s.registry.Generation(next.AgentID)
			if tc.stopped {
				if ae := s.stopAgent(context.Background(), id); ae != nil {
					t.Fatal(ae)
				}
			}
			_, err = s.stateStore.BeginThinkTankAttempt(state.ThinkTankBegin{RoomID: room, Revision: d.Room.Revision,
				AgentID: id, Turn: next.Turn, Generation: gen, TurnID: "quota-source"})
			if err != nil {
				t.Fatal(err)
			}
			now := time.Now().UTC()
			reset, due := now.Add(-2*time.Minute), now.Add(-time.Minute)
			q, err := s.stateStore.ObserveOwnedQuota(state.QuotaObservation{AgentID: id, EpisodeID: "room-start-failure",
				SourceGeneration: gen, ExecutingTurn: "quota-source", ObservedAt: now.Add(-3 * time.Minute),
				ResetAt: &reset, DueAt: &due, State: state.QuotaScheduled})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.stateStore.DB().Exec(`CREATE TRIGGER fail_quota_activation BEFORE INSERT ON status
WHEN NEW.last_trace='QuotaContinuation' BEGIN SELECT RAISE(FAIL, 'injected activation status failure'); END`); err != nil {
				t.Fatal(err)
			}
			if tc.failCompensation {
				if _, err := s.stateStore.DB().Exec(`CREATE TRIGGER fail_room_compensation BEFORE UPDATE ON think_tank_attempts
WHEN OLD.turn_id<>'quota-source' AND NEW.state='failed' BEGIN SELECT RAISE(FAIL, 'injected compensation failure'); END`); err != nil {
					t.Fatal(err)
				}
			}
			s.startQuotaContinuation(context.Background(), q)
			got, err := s.stateStore.GetQuota(id)
			if err != nil || got.State != state.QuotaNeedsAttention {
				t.Fatalf("quota=%+v err=%v", got, err)
			}
			if n := promptCount(t, prompts); n != 0 {
				t.Fatalf("provider prompts=%d", n)
			}
			if tc.failCompensation {
				s.thinkTankCaptureMu.Lock()
				pending := len(s.thinkTankFailures)
				s.thinkTankCaptureMu.Unlock()
				if pending != 1 {
					t.Fatalf("failed compensation lost recovery owner: %d", pending)
				}
				if _, err := s.stateStore.DB().Exec(`DROP TRIGGER fail_room_compensation`); err != nil {
					t.Fatal(err)
				}
				s.retryThinkTankFailures()
			}
			d, err = s.stateStore.ReadThinkTank(room)
			if err != nil {
				t.Fatal(err)
			}
			if len(d.Attempts) != 2 {
				t.Fatalf("attempts=%+v", d.Attempts)
			}
			for _, attempt := range d.Attempts {
				if attempt.State != state.ThinkTankAttemptFailed {
					t.Fatalf("unsettled attempt=%+v", attempt)
				}
			}
			for _, member := range d.Members {
				if member.Completed != 0 {
					t.Fatalf("failed start charged allowance: %+v", member)
				}
			}
			var contributions int
			if err := s.stateStore.DB().QueryRow(`SELECT COUNT(*) FROM think_tank_entries WHERE room_id=?`, room).Scan(&contributions); err != nil {
				t.Fatal(err)
			}
			if contributions != 0 {
				t.Fatalf("failed start published %d entries", contributions)
			}
			s.thinkTankCaptureMu.Lock()
			capture := s.thinkTankCaptures[id]
			s.thinkTankCaptureMu.Unlock()
			if capture != nil {
				t.Fatalf("capture survived failed start: %+v", capture)
			}
		})
	}
}
