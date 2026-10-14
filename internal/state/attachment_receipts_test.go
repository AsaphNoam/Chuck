package state

import (
	"errors"
	"github.com/AsaphNoam/Chuck/internal/attachment"
	"os"
	"path/filepath"
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
	for _, id := range []string{"held", "submitted", "unknown", "prepared"} {
		r := AttachmentReceipt{AgentID: a.AgentID, RequestID: id, PayloadHash: "hash-" + id, MessageJSON: `{"text":"exact text","attachments":[]}`, Action: "prompt"}
		if err = s.CreateAttachmentReceipt(r); err != nil {
			t.Fatal(err)
		}
		if id != "prepared" {
			if err = s.SetAttachmentDelivery(a.AgentID, id, id); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err = s.ReleaseRestartedAttachments(); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{"held": "released", "submitted": "submitted", "unknown": "unknown", "prepared": "released"} {
		r, err := s.AttachmentReceipt(a.AgentID, id)
		if err != nil || r.Delivery != want {
			t.Fatalf("%s: %+v %v", id, r, err)
		}
	}
	recovery, err := s.AttachmentRecovery(a.AgentID)
	if err != nil || recovery.RequestID != "prepared" {
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
	if err = s.DismissAttachmentRecovery(a.AgentID, "prepared"); err != nil {
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

func TestDeleteAgentKeepsOwnershipUntilAttachmentCleanupSucceeds(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a := Agent{AgentID: "a_cleanup", Name: "Attachments", Project: "p", Role: "r", Backend: "b", Model: "m", Interface: "chat"}
	if err = s.WriteAgent(a); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "source.txt")
	if err = os.WriteFile(source, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	asset, err := attachment.New(s.home).Put(a.AgentID, "notes.txt", strings.NewReader("owned"))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.CreateAttachmentReceipt(AttachmentReceipt{AgentID: a.AgentID, RequestID: "r1", PayloadHash: "h", MessageJSON: "{}", Action: "prompt"}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(s.home, "sessions", a.AgentID, "attachments", asset.ID)
	if err = os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	if err = s.DeleteAgent(a.AgentID); err == nil {
		t.Fatal("cleanup failure was reported as successful deletion")
	}
	if _, err = s.ReadAgent(a.AgentID); err != nil {
		t.Fatalf("failed cleanup dropped durable identity: %v", err)
	}
	if _, err = s.AttachmentReceipt(a.AgentID, "r1"); err != nil {
		t.Fatalf("failed cleanup dropped receipt ownership: %v", err)
	}
	if err = os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err = s.DeleteAgent(a.AgentID); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if _, err = s.ReadAgent(a.AgentID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("retried deletion kept identity: %v", err)
	}
	if _, err = os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("retried deletion kept owned bytes: %v", err)
	}
	if raw, err := os.ReadFile(source); err != nil || string(raw) != "source" {
		t.Fatalf("source file changed: %q %v", raw, err)
	}
}

func TestAttachmentReceiptForwardMigrationPreservesCheckpointDatabase(t *testing.T) {
	home := t.TempDir()
	s, err := Open(home)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.WriteAgent(testAgent("a_migrate", timeNow())); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`DROP TABLE attachment_receipts; DELETE FROM schema_migrations WHERE version = 51`); err != nil {
		t.Fatal(err)
	}
	var priorSchema string
	for _, migration := range migrations {
		if migration.version == 50 {
			priorSchema = migration.sql
		}
	}
	if _, err = s.db.Exec(priorSchema); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`INSERT INTO attachment_receipts(agent_id,request_id,payload_hash,message_json,action,delivery,outcome,dismissed,updated_at) VALUES('a_migrate','old','hash','{}','steer','unknown','steered',1,'2026-10-12T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`UPDATE attachment_receipts SET delivery='prepared' WHERE request_id='old'`); err == nil {
		t.Fatal("checkpoint schema was rewritten in place")
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(home)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	old, err := s.AttachmentReceipt("a_migrate", "old")
	if err != nil || old.Delivery != "unknown" || old.Outcome != "steered" || old.PayloadHash != "hash" {
		t.Fatalf("existing receipt changed: %+v %v", old, err)
	}
	var dismissed int
	if err = s.db.QueryRow(`SELECT dismissed FROM attachment_receipts WHERE request_id='old'`).Scan(&dismissed); err != nil || dismissed != 1 {
		t.Fatalf("dismissal changed: %d %v", dismissed, err)
	}
	if err = s.CreateAttachmentReceipt(AttachmentReceipt{AgentID: "a_migrate", RequestID: "new", PayloadHash: "new hash", MessageJSON: strings.Repeat(" ", 70<<10), Action: "prompt"}); err != nil {
		t.Fatal("normalized descriptor overhead not admitted after upgrade:", err)
	}
	if err = s.ReleaseRestartedAttachments(); err != nil {
		t.Fatal(err)
	}
	recovery, err := s.AttachmentRecovery("a_migrate")
	if err != nil || recovery.RequestID != "new" {
		t.Fatalf("prepared recovery not migrated: %+v %v", recovery, err)
	}
}
