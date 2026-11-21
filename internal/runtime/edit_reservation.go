package runtime

import (
	"errors"
	"sync"
)

// ErrEditInProgress refuses new input while last-message Edit owns the
// agent's idle turn reservation (TS-01.R51). Nothing is accepted or queued.
var ErrEditInProgress = errors.New("runtime: this conversation is being edited; wait for the edit to finish")

// ErrEditBusy refuses an Edit reservation while the runtime still owns work:
// a claimed turn, held input, a Steer fallback or background tasks (TS-01.R51).
var ErrEditBusy = errors.New("runtime: the agent still has work in progress")

// editReservations is the conversation-scoped idle-turn reservation. It is
// keyed by agent id and survives the runtime replacement Edit performs, so the
// candidate cannot receive Send, Steer, held promotion or automatic work before
// the edit's own replacement input (identified by its request id). The turn
// claim consults it under the agent's own lock, so a reservation and a turn
// claim exclude each other (INV §5).
type editReservations struct {
	mu   sync.Mutex
	byID map[string]string
}

// blocks reports whether a turn for requestID ("" for automatic work) is
// refused by agentID's reservation.
func (e *editReservations) blocks(agentID, requestID string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	held, ok := e.byID[agentID]
	return ok && (requestID == "" || held != requestID)
}

// ReserveEdit claims agentID's idle turn for requestID. It fails with
// ErrEditInProgress when another edit holds it, and with ErrEditBusy when a
// live runtime still owns work. The check and the reservation happen under the
// agent's turn lock, so no turn can be claimed between them.
func (c *ChatRuntime) ReserveEdit(agentID, requestID string) error {
	if as, err := c.lookup(agentID); err == nil {
		as.mu.Lock()
		defer as.mu.Unlock()
		if as.ownsWorkLocked() {
			return ErrEditBusy
		}
	}
	c.edits.mu.Lock()
	defer c.edits.mu.Unlock()
	if held, ok := c.edits.byID[agentID]; ok && held != requestID {
		return ErrEditInProgress
	}
	if c.edits.byID == nil {
		c.edits.byID = map[string]string{}
	}
	c.edits.byID[agentID] = requestID
	return nil
}

// ownsWorkLocked is the work ReserveEdit refuses. Caller must hold as.mu.
func (as *agentState) ownsWorkLocked() bool {
	return as.turn.turnActive || !as.turn.held.message().empty() || !as.turn.steerFallback.message().empty() || as.openTasks > 0
}

// OwnsWork reports whether agentID's live runtime still owns work an Edit
// reservation refuses. The durable idle row is written before turn_end's
// subscribers return and the turn claim is released after them (finishTurn),
// so this stays true briefly after idle is visible.
func (c *ChatRuntime) OwnsWork(agentID string) bool {
	as, err := c.lookup(agentID)
	if err != nil {
		return false
	}
	as.mu.Lock()
	defer as.mu.Unlock()
	return as.ownsWorkLocked()
}

// ReleaseEdit ends requestID's reservation; a stale release is a no-op.
func (c *ChatRuntime) ReleaseEdit(agentID, requestID string) {
	c.edits.mu.Lock()
	defer c.edits.mu.Unlock()
	if c.edits.byID[agentID] == requestID {
		delete(c.edits.byID, agentID)
	}
}

// admitAgainstEdit refuses input other than the reserving edit's own.
func (c *ChatRuntime) admitAgainstEdit(agentID, requestID string) error {
	if c.edits.blocks(agentID, requestID) {
		return ErrEditInProgress
	}
	return nil
}
