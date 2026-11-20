package state

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/attachment"
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
	if did, err := s.DismissAttachmentRecovery(a.AgentID, "submitted"); err != nil || did {
		t.Fatalf("dismissed submitted input: %v %v", did, err)
	}
	if _, err = s.AttachmentRecovery(a.AgentID); err != nil {
		t.Fatal("dismissing a different record removed recovery")
	}
	for _, requestID := range []string{"held", "prepared"} {
		if did, err := s.DismissAttachmentRecovery(a.AgentID, requestID); err != nil || !did {
			t.Fatalf("dismiss %s: %v %v", requestID, did, err)
		}
	}
	if recovery, err = s.AttachmentRecovery(a.AgentID); err != nil || recovery.RequestID != "unknown" {
		t.Fatalf("uncertain input is not recoverable: %+v %v", recovery, err)
	}
	if did, err := s.DismissAttachmentRecovery(a.AgentID, "unknown"); err != nil || !did {
		t.Fatalf("dismiss uncertain input: %v %v", did, err)
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

func TestRecoveryClaimsReconcileAfterRestart(t *testing.T) {
	home := t.TempDir()
	s, err := Open(home)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	agentID := "a_123456"
	if err := s.WriteAgent(Agent{AgentID: agentID, Name: "Recovery", Project: "p", Role: "r", Backend: "b", Model: "m", Interface: "chat"}); err != nil {
		t.Fatal(err)
	}
	for _, example := range []struct{ source, successor string }{{"claimed-only", "missing"}, {"already-retained", "successor"}} {
		rec := AttachmentReceipt{AgentID: agentID, RequestID: example.source, PayloadHash: "old-" + example.source, MessageJSON: `{}`, Action: "prompt"}
		if err := s.CreateAttachmentReceipt(rec); err != nil {
			t.Fatal(err)
		}
		if err := s.SetAttachmentDelivery(agentID, example.source, "unknown"); err != nil {
			t.Fatal(err)
		}
		if did, err := s.ClaimAttachmentRecovery(agentID, example.source, example.successor, "new-"+example.successor); err != nil || !did {
			t.Fatalf("claim %s: %v %v", example.source, did, err)
		}
	}
	if err := s.CreateAttachmentReceipt(AttachmentReceipt{AgentID: agentID, RequestID: "successor", PayloadHash: "new-successor", MessageJSON: `{}`, Action: "prompt"}); err != nil {
		t.Fatal(err)
	}
	if repaired, err := s.RepairAttachmentRecovery(agentID, "already-retained", "successor", "new-successor"); repaired || err != nil {
		t.Fatalf("same-ID replay settled a provisional successor: %v %v", repaired, err)
	}
	if dismissed, err := s.DismissAttachmentRecovery(agentID, "already-retained"); dismissed || err != nil {
		t.Fatalf("same-ID replay released the active claim: %v %v", dismissed, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(home)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.ReleaseRestartedAttachments(); err != nil {
		t.Fatal(err)
	}
	for id, expected := range map[string]string{"claimed-only": "offered", "already-retained": "settled", "successor": "offered"} {
		if status, err := s.AttachmentRecoveryStatus(agentID, id); err != nil || status != expected {
			t.Fatalf("%s status = %q, %v; want %q", id, status, err, expected)
		}
	}
	if old, err := s.AttachmentReceipt(agentID, "already-retained"); err != nil || old.Delivery != "unknown" {
		t.Fatalf("recovery changed historical receipt delivery: %+v %v", old, err)
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
	if _, err = s.db.Exec(`DROP TABLE attachment_receipts; DELETE FROM schema_migrations WHERE version IN (51, 65, 69, 76, 79, 81)`); err != nil {
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
	if status, err := s.AttachmentRecoveryStatus("a_migrate", "old"); status != "settled" || err != nil {
		t.Fatalf("migrated dismissal status = %q, %v", status, err)
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

// QUEUED-TAIL-1 (FS-03.R48/R118, INV §19): input a continuation has claimed but
// not yet recorded stays listed as queued, so no read finds it in neither the
// queued list nor history. It leaves the list once submitted, and a restart
// that releases it for recovery removes it too.
func TestClaimedQueuedInputStaysListedUntilSubmitted(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a := Agent{AgentID: "a_queued", Name: "Queued", Project: "p", Role: "r", Backend: "b", Model: "m", Interface: "chat"}
	if err = s.WriteAgent(a); err != nil {
		t.Fatal(err)
	}
	now := formatTime(time.Now().UTC())
	if _, err = s.db.Exec(`INSERT INTO tasks(task_id, project, display_name, instruction, target_kind, state, created_by_kind, assigned_agent_id, created_at, updated_at)
VALUES ('tk_q', 'p', 'Assignment', '', 'agent', 'waiting', 'pipeline', ?, ?, ?)`, a.AgentID, now, now); err != nil {
		t.Fatal(err)
	}
	listed := func() []string {
		t.Helper()
		queued, err := s.QueuedAssignmentInput(a.AgentID)
		if err != nil {
			t.Fatal(err)
		}
		out := []string{}
		for _, q := range queued {
			out = append(out, q.RequestID+"@"+q.AssignmentTaskID)
		}
		return out
	}
	for _, id := range []string{"first", "second"} {
		if _, ok, err := s.QueueAssignmentInput(AttachmentReceipt{AgentID: a.AgentID, RequestID: id, PayloadHash: "h", Action: "prompt", MessageJSON: `{"text":"x"}`}); err != nil || !ok {
			t.Fatalf("queue %s = %v %v", id, ok, err)
		}
	}
	if _, err = s.db.Exec(`UPDATE tasks SET state='running' WHERE task_id='tk_q'`); err != nil {
		t.Fatal(err)
	}
	if rec, ok, err := s.TakeQueuedAssignmentInput(a.AgentID); err != nil || !ok || rec.RequestID != "first" {
		t.Fatalf("take = %+v %v %v", rec, ok, err)
	}
	if got := strings.Join(listed(), ","); got != "first@tk_q,second@tk_q" {
		t.Fatalf("listed during claim = %s, want the claimed input first, then the rest", got)
	}
	if err = s.SetAttachmentDelivery(a.AgentID, "first", "submitted"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(listed(), ","); got != "second@tk_q" {
		t.Fatalf("listed after submit = %s", got)
	}
	if _, ok, err := s.TakeQueuedAssignmentInput(a.AgentID); err != nil || !ok {
		t.Fatalf("second take = %v %v", ok, err)
	}
	if err = s.ReleaseRestartedAttachments(); err != nil {
		t.Fatal(err)
	}
	if got := listed(); len(got) != 0 {
		t.Fatalf("listed after restart = %v, want the released claim gone", got)
	}
}
