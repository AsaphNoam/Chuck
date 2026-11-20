package state

import (
	"errors"
	"testing"
)

// EDIT-REVIEW-2 (TS-02.R69, INV §15/§19): the replacement is owned by the
// open edit until its receipt exists, then by the receipt alone. A failure on
// either side of the dispatch transaction leaves exactly one recovery owner.
func TestDispatchEditTransfersOwnershipInOneTransaction(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	const id = "a_edit"
	if err := s.WriteAgent(Agent{AgentID: id, Name: "E", Project: "p", Role: "r", Backend: "b", Model: "m", Interface: "chat"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateEdit(EditTransition{AgentID: id, RequestID: "edit-1", PayloadHash: "h", MessageJSON: `{"text":"x"}`, MessageSeq: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DecideEdit(id, "edit-1", 1, 3, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.FinishEdit(id, "edit-1"); err != nil {
		t.Fatal(err)
	}
	receipt := AttachmentReceipt{AgentID: id, RequestID: "edit-1", PayloadHash: "h", MessageJSON: `{"text":"x"}`, Action: "prompt"}

	// Failure inside the transaction: the edit still owns the replacement.
	fault := errors.New("injected")
	editDispatchFault = func() error { return fault }
	err = s.DispatchEdit(receipt, "prepared")
	editDispatchFault = nil
	if !errors.Is(err, fault) {
		t.Fatalf("dispatch with fault = %v", err)
	}
	if open, _ := s.OpenEdits(); len(open) != 1 || open[0].Dispatched {
		t.Fatalf("failed dispatch left open edits %+v, want the undispatched edit", open)
	}
	if _, err := s.AttachmentReceipt(id, "edit-1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("failed dispatch left a receipt: %v", err)
	}

	// After commit, a process exit before admission: the receipt owns it and
	// startup releases it for explicit recovery.
	if err := s.DispatchEdit(receipt, "prepared"); err != nil {
		t.Fatal(err)
	}
	if open, _ := s.OpenEdits(); len(open) != 0 {
		t.Fatalf("dispatched edit still open: %+v", open)
	}
	if err := s.DispatchEdit(receipt, "prepared"); !errors.Is(err, ErrEditConflict) {
		t.Fatalf("second dispatch = %v, want conflict", err)
	}
	if err := s.ReleaseRestartedAttachments(); err != nil {
		t.Fatal(err)
	}
	if rec, err := s.AttachmentRecovery(id); err != nil || rec.RequestID != "edit-1" {
		t.Fatalf("recovery after restart = %+v %v", rec, err)
	}
}

// EDIT-REVIEW-3 (FS-01.R61): background work keeps Edit unavailable even on
// an idle chat, because preparation stops the runtime that owns it.
func TestEditAvailabilityWaitsForBackgroundWork(t *testing.T) {
	idle := AgentState{Interface: "chat", State: "idle", RuntimeCapabilities: RuntimeCapabilities{Fork: true}}
	if gate := EditAvailability(idle, "s1", false); !gate.Available {
		t.Fatalf("idle gate = %+v", gate)
	}
	idle.BackgroundRunning = 1
	if gate := EditAvailability(idle, "s1", false); gate.Available || gate.Reason != EditReasonBackground {
		t.Fatalf("background gate = %+v", gate)
	}
}
