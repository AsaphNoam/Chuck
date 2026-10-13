package server

import (
	"errors"
	"net/http"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

func TestReceiptFailureAfterDurableMessagePreservesCopiesAndRetry(t *testing.T) {
	srv, h, id := attachmentTestServer(t, nil)
	srv.registry.SetMessageOutcomeHandler(func(agent, request, delivery string) error {
		if delivery == "submitted" {
			return errors.New("injected receipt update failure")
		}
		return srv.stateStore.SetAttachmentDelivery(agent, request, delivery)
	})
	part := uploadPart{field: "f", name: "failure.txt", data: []byte("immutable bytes before failed receipt")}
	meta := messageMetadata(t, "receipt-failure", "exact message", part)
	resp, body := postAttachmentMessage(t, h, "/api/sessions/"+id+"/prompt", meta, part)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("accepted local message became retryable failure: %d %s", resp.StatusCode, body)
	}
	accepted := decodeAttachmentAcceptance(t, body)
	if accepted.AttachmentDelivery != "unknown" || len(accepted.Attachments) != 1 {
		t.Fatalf("receipt %+v", accepted)
	}
	events := readEvents(t, srv, id)
	count := 0
	for _, ev := range events {
		if ev.Type == runtime.EvUserPrompt {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("durable message count %d", count)
	}
	read := serveLocal(t, h, http.MethodGet, "/api/sessions/"+id+"/attachments/"+accepted.Attachments[0].ID, nil)
	if read.Code != http.StatusOK || read.Body.String() != string(part.data) {
		t.Fatal("durable event lost attachment bytes")
	}
	resp, body = postAttachmentMessage(t, h, "/api/sessions/"+id+"/prompt", meta, part)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("retry %d %s", resp.StatusCode, body)
	}
	retry := decodeAttachmentAcceptance(t, body)
	if retry.Attachments[0].ID != accepted.Attachments[0].ID {
		t.Fatal("retry reassigned owned copy")
	}
	count = 0
	for _, ev := range readEvents(t, srv, id) {
		if ev.Type == runtime.EvUserPrompt {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("retry duplicated accepted event: %d", count)
	}
	// No provider frame was released before the failed hook. Startup offers
	// this prepared copy for explicit recovery; unknown dispatched ones stay out.
	if err := srv.stateStore.ReleaseRestartedAttachments(); err != nil {
		t.Fatal(err)
	}
	recovered, err := srv.stateStore.AttachmentRecovery(id)
	if err != nil || recovered.RequestID != "receipt-failure" {
		t.Fatalf("unsubmitted recovery %+v %v", recovered, err)
	}
	if err = srv.stateStore.SetAttachmentDelivery(id, recovered.RequestID, "unknown"); err != nil {
		t.Fatal(err)
	}
	if _, err = srv.stateStore.AttachmentRecovery(id); !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("ambiguous consumed request offered recovery: %v", err)
	}
}

func TestAttachmentUploadAdmissionIsPerAgentAndProcessBounded(t *testing.T) {
	srv := testServer(t, false)
	for _, id := range []string{"a_one", "a_two", "a_three", "a_four"} {
		if !srv.claimAttachmentUpload(id) {
			t.Fatal("allowed slot refused")
		}
	}
	if srv.claimAttachmentUpload("a_one") || srv.claimAttachmentUpload("a_five") {
		t.Fatal("unbounded or duplicate upload admitted")
	}
	srv.releaseAttachmentUpload("a_two")
	if !srv.claimAttachmentUpload("a_five") {
		t.Fatal("released upload slot leaked")
	}
}
