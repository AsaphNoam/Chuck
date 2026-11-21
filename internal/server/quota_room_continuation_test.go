package server

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// A room quota turn retains the room receipt protocol; task continuation
// preparation must neither replace its prompt nor require a task assignment.
func TestQuotaRoomContinuationKeepsRoomContract(t *testing.T) {
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
	if _, err := s.stateStore.BeginThinkTankAttempt(state.ThinkTankBegin{RoomID: room, Revision: d.Room.Revision,
		AgentID: id, Turn: next.Turn, Generation: gen, TurnID: "quota-source"}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	reset, due := now.Add(-2*time.Minute), now.Add(-time.Minute)
	q, err := s.stateStore.ObserveOwnedQuota(state.QuotaObservation{AgentID: id, EpisodeID: "room-continuation",
		SourceGeneration: gen, ExecutingTurn: "quota-source", ObservedAt: now.Add(-3 * time.Minute),
		ResetAt: &reset, DueAt: &due, State: state.QuotaScheduled})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.stateStore.EndConversationTurn(id, state.TurnEnd{Generation: gen, TurnID: "quota-source", Outcome: "quota_blocked"}); err != nil {
		t.Fatal(err)
	}
	s.startQuotaContinuation(context.Background(), q)
	waitPrompts(t, prompts, 1)
	raw, err := os.ReadFile(prompts)
	if err != nil {
		t.Fatal(err)
	}
	text := firstPromptText(t, raw)
	if !strings.HasPrefix(text, "Continue your quota-interrupted Think Tank opportunity.") || !strings.Contains(text, "read_receipt") || strings.Contains(text, "Chuck assignment context:") {
		t.Fatalf("room quota continuation lost its room contract: %q", text)
	}
}
