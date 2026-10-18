package runtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"
)

// Host decisions a waiter can observe. Only HostApprove permits the guarded
// operation, and only after Admit succeeds (TS-04.R94–R95).
const (
	HostApprove   = "approve"
	HostDeny      = "deny"
	HostTimeout   = "timeout"
	HostCancelled = "cancelled"
)

// ErrHostPermissionStale means the requester generation or turn that the host
// request was bound to is no longer the one executing.
var ErrHostPermissionStale = errors.New("runtime: host permission requester is no longer current")

// HostPermissionRequest describes one Chuck-owned decision. It reuses the
// pending-permission registry, the operator decision route, and the timeout,
// but no ACP request exists: provider permission modes, skip-permissions and
// Chuck-tool automatic approval never reach it (FS-03.R91, TS-04.R94).
type HostPermissionRequest struct {
	Generation string // requester generation the caller's MCP session is bound to
	Origin     string
	Name       string
	Reason     string
	Args       json.RawMessage
	PeerStop   *PeerStopSubject
}

// HostPermission is one outstanding host decision bound to the requester
// generation and executing turn captured when it was opened.
type HostPermission struct {
	ID string

	c          *ChatRuntime
	as         *agentState
	generation string
	turnID     string
	decision   chan string   // buffered: the single settled decision
	voidSeq    uint64        // as.hostVoidSeq at open; a later cancellation voids it
	published  chan struct{} // closed once the request and waiting status are published
}

// awaitPublished holds a settler until the opener has published the request,
// so a resolution never precedes its request or its waiting status (INV §5).
func (p *pendingPerm) awaitPublished() {
	if p.host != nil {
		<-p.host.published
	}
}

// RequestHostPermission opens a host decision on the requester's live turn,
// publishing an ordinary permission_request and the waiting_input status.
func (c *ChatRuntime) RequestHostPermission(agentID string, req HostPermissionRequest) (*HostPermission, error) {
	as, err := c.lookup(agentID)
	if err != nil {
		return nil, err
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return nil, err
	}
	h := &HostPermission{ID: req.Origin + "_" + hex.EncodeToString(raw[:]), c: c, as: as,
		generation: req.Generation, decision: make(chan string, 1), published: make(chan struct{})}
	timeout := permissionTimeout()
	expiresAt := ""
	if timeout > 0 {
		expiresAt = time.Now().UTC().Add(timeout).Format(time.RFC3339)
	}
	data := PermissionRequestData{
		ToolCallID: h.ID, Name: req.Name, Reason: req.Reason, Args: req.Args,
		Options: []PermOption{
			{OptionID: "allow_once", Label: "Approve once", Kind: "allow_once"},
			{OptionID: "reject_once", Label: "Deny", Kind: "reject_once"},
		},
		ExpiresAt: expiresAt, Origin: req.Origin, PeerStop: req.PeerStop,
	}
	if data.Args == nil {
		data.Args = json.RawMessage("{}")
	}
	p := &pendingPerm{name: req.Name, host: h,
		optByKind: map[string]string{"allow_once": "allow_once", "reject_once": "reject_once"}}

	as.mu.Lock()
	if as.stopped || as.stopping || as.generation != req.Generation || !as.turnActive || as.execTurnID == "" {
		as.mu.Unlock()
		return nil, ErrHostPermissionStale
	}
	h.turnID = as.execTurnID
	h.voidSeq = as.hostVoidSeq
	as.pending[h.ID] = p
	as.mu.Unlock()
	if timeout > 0 {
		p.timer = time.AfterFunc(timeout, func() { c.onPermissionTimeout(as, h.ID) })
	}
	// A settler that claims the request in this gap waits on published, so the
	// request, then its waiting status, precede any resolution.
	c.updateStatus(as, "waiting_input", "Permission: "+req.Name, "PermissionRequest: "+req.Name, keepBusySince)
	c.emit(as, EvPermissionRequest, data)
	close(h.published)
	return h, nil
}

// PermissionPending reports whether toolCallID is still awaiting a decision
// in agentID's live runtime. Durable events cannot answer this: a request
// from an ended generation has no record of its end (TS-04.R94).
func (c *ChatRuntime) PermissionPending(agentID, toolCallID string) bool {
	as, err := c.lookup(agentID)
	if err != nil {
		return false
	}
	as.mu.Lock()
	defer as.mu.Unlock()
	_, ok := as.pending[toolCallID]
	return ok
}

// Wait blocks until the operator decides, the request times out or is
// cancelled, or ctx ends. A ctx end withdraws the request as cancelled through
// the same single-winner path, so it cannot later be approved.
func (h *HostPermission) Wait(ctx context.Context) string {
	select {
	case d := <-h.decision:
		return d
	case <-ctx.Done():
	}
	h.c.resolvePending(h.as, h.ID, "cancelled", "")
	return <-h.decision
}

// Cancel withdraws a still-pending request as cancelled. It is a no-op once
// the request has been decided.
func (h *HostPermission) Cancel() {
	h.c.resolvePending(h.as, h.ID, "cancelled", "")
}

// Admit claims an approved request for execution exactly once. It succeeds
// only while the captured requester generation and turn are still executing
// and no cancellation voided the approval; after it succeeds, a later
// cancellation cannot undo the admitted operation (TS-04.R95, INV §5).
func (h *HostPermission) Admit() error {
	as := h.as
	as.mu.Lock()
	defer as.mu.Unlock()
	if _, held := as.hostApproved[h.ID]; !held {
		return ErrHostPermissionStale
	}
	delete(as.hostApproved, h.ID)
	if as.hostVoidSeq != h.voidSeq || as.stopped || as.stopping || as.generation != h.generation ||
		!as.turnActive || as.execTurnID != h.turnID {
		return ErrHostPermissionStale
	}
	return nil
}

// approved parks an approval where requester cancellation can void it until
// Admit. It runs before the decision is published to the waiter.
func (h *HostPermission) approved(as *agentState) {
	as.mu.Lock()
	if as.hostApproved == nil {
		as.hostApproved = map[string]*HostPermission{}
	}
	as.hostApproved[h.ID] = h
	as.mu.Unlock()
}

// voidHostApprovalsLocked is requester cancellation's side of the admission
// race: every host request opened before it, approved yet or not, can no
// longer be admitted. Caller holds as.mu.
func (as *agentState) voidHostApprovalsLocked() {
	as.hostVoidSeq++
	clear(as.hostApproved)
}

func (h *HostPermission) settle(decision string) {
	select {
	case h.decision <- decision:
	default:
	}
}

func hostDecision(outcome, optionID string) string {
	switch {
	case outcome == "selected" && optionID == "allow_once":
		return HostApprove
	case outcome == "selected" && optionID == "reject_once":
		return HostDeny
	default:
		return HostCancelled
	}
}
