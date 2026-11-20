package runtime

import (
	"errors"
	"sync"
)

// ErrEditInProgress refuses new input while last-message Edit owns the
// agent's idle turn reservation (TS-01.R51). Nothing is accepted or queued.
var ErrEditInProgress = errors.New("runtime: this conversation is being edited; wait for the edit to finish")

// editReservations is the conversation-scoped idle-turn reservation. It is
// keyed by agent id and survives the runtime replacement Edit performs, so the
// candidate cannot receive Send, Steer, held promotion or automatic work before
// the edit's own replacement input (identified by its request id).
type editReservations struct {
	mu   sync.Mutex
	byID map[string]string
}

// ReserveEdit claims agentID's idle turn for requestID; false when another
// edit already holds it.
func (c *ChatRuntime) ReserveEdit(agentID, requestID string) bool {
	c.edits.mu.Lock()
	defer c.edits.mu.Unlock()
	if held, ok := c.edits.byID[agentID]; ok && held != requestID {
		return false
	}
	if c.edits.byID == nil {
		c.edits.byID = map[string]string{}
	}
	c.edits.byID[agentID] = requestID
	return true
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
	c.edits.mu.Lock()
	defer c.edits.mu.Unlock()
	if held, ok := c.edits.byID[agentID]; ok && (requestID == "" || held != requestID) {
		return ErrEditInProgress
	}
	return nil
}
