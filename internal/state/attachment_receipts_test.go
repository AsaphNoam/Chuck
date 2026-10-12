package state

import (
	"errors"
	"github.com/AsaphNoam/Chuck/internal/attachment"
	"strings"
	"testing"
)

func TestAttachmentReceiptsRestartAndRecovery(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a := Agent{AgentID: "a_123456", Name: "Attachments", Project: "p", Role: "r", Backend: "b", Model: "m", Interface: "chat"}
	if err = s.WriteAgent(a); err != nil {
		t.Fatal(err)
	}
	asset, err := attachment.New(s.home).Put(a.AgentID, "empty.txt", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"held", "submitted", "unknown"} {
		r := AttachmentReceipt{AgentID: a.AgentID, RequestID: id, PayloadHash: "hash-" + id, MessageJSON: `{"text":"exact text","attachments":[]}`, Action: "prompt"}
		if err = s.CreateAttachmentReceipt(r); err != nil {
			t.Fatal(err)
		}
		if id != "unknown" {
			if err = s.SetAttachmentDelivery(a.AgentID, id, id); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err = s.ReleaseRestartedAttachments(); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{"held": "released", "submitted": "submitted", "unknown": "unknown"} {
		r, err := s.AttachmentReceipt(a.AgentID, id)
		if err != nil || r.Delivery != want {
			t.Fatalf("%s: %+v %v", id, r, err)
		}
	}
	recovery, err := s.AttachmentRecovery(a.AgentID)
	if err != nil || recovery.RequestID != "held" {
		t.Fatalf("recovery %+v %v", recovery, err)
	}
	if err = s.DismissAttachmentRecovery(a.AgentID, "submitted"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.AttachmentRecovery(a.AgentID); err != nil {
		t.Fatal("dismissing a different record removed recovery")
	}
	if err = s.DismissAttachmentRecovery(a.AgentID, "held"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.AttachmentRecovery(a.AgentID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("dismissed recovery %v", err)
	}
	if err = s.CreateAttachmentReceipt(AttachmentReceipt{AgentID: a.AgentID, RequestID: "held", PayloadHash: "changed", MessageJSON: "{}", Action: "prompt"}); err == nil {
		t.Fatal("duplicate accepted receipt was overwritten")
	}
	if err = s.DeleteAgent(a.AgentID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.AttachmentReceipt(a.AgentID, "submitted"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted conversation receipt %v", err)
	}
	if f, _, err := attachment.New(s.home).Open(a.AgentID, asset.ID); err == nil {
		f.Close()
		t.Fatal("deleted conversation retained owned bytes")
	}
}
