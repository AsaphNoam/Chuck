package runtime

import (
	"errors"
	"testing"
)

// EDIT-REVIEW-3 (TS-01.R51, INV §5): the Edit reservation and the turn claim
// are decided under the same agent lock. A reservation refuses while the
// runtime owns work, and once taken no automatic or other claim can start a
// turn; only the edit's own replacement may.
func TestEditReservationExcludesTurnClaims(t *testing.T) {
	c := &ChatRuntime{agents: map[string]*agentState{}}
	as := &agentState{agentID: "a", edits: &c.edits}
	c.agents["a"] = as

	for name, busy := range map[string]func(){
		"claimed turn":    func() { as.turn.turnActive = true },
		"held input":      func() { as.turn.held = heldMessage{Text: "next"} },
		"steer fallback":  func() { as.turn.steerFallback = heldMessage{Text: "steer"} },
		"background work": func() { as.openTasks = 1 },
	} {
		as.turn = currentTurn{}
		as.openTasks = 0
		busy()
		if err := c.ReserveEdit("a", "edit-1"); !errors.Is(err, ErrEditBusy) {
			t.Fatalf("%s: reserve = %v, want busy", name, err)
		}
	}
	as.turn = currentTurn{}
	as.openTasks = 0
	if err := c.ReserveEdit("a", "edit-1"); err != nil {
		t.Fatalf("idle reserve: %v", err)
	}
	as.mu.Lock()
	_, automatic := as.claimTurnLocked()
	_, other := as.claimTurnForLocked("send-1")
	ownID, own := as.claimTurnForLocked("edit-1")
	as.mu.Unlock()
	if automatic || other {
		t.Fatalf("claims under reservation: automatic=%v other=%v", automatic, other)
	}
	if !own || ownID == "" {
		t.Fatal("the edit's own replacement could not claim the turn")
	}
	if err := c.ReserveEdit("a", "edit-2"); err == nil {
		t.Fatal("a second edit reserved a claimed turn")
	}
}
