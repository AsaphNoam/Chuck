package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// The HTTP retry boundary must retain one canonical input even if the first
// success reply was lost. Changed content under the same ID is never delivered.
func TestTextReceiptReplayAndConflict(t *testing.T) {
	srv, handler, id := attachmentTestServer(t, nil)
	path := "/api/sessions/" + id + "/prompt"
	input := map[string]string{"text": "exact input", "request_id": "text-replay"}
	for i := 0; i < 2; i++ {
		res, body := postJSONHandler(t, handler, path, input)
		if res.StatusCode != http.StatusAccepted {
			t.Fatalf("attempt %d: %d %s", i, res.StatusCode, body)
		}
	}
	input["text"] = "changed input"
	res, body := postJSONHandler(t, handler, path, input)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("changed payload: %d %s", res.StatusCode, body)
	}
	events, err := srv.registry.Transcript(id)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, event := range events {
		if event.Type == runtime.EvUserPrompt {
			var prompt runtime.UserPromptData
			if err := json.Unmarshal(event.Data, &prompt); err != nil {
				t.Fatal(err)
			}
			if prompt.RequestID == "text-replay" {
				count++
			}
		}
	}
	if count != 1 {
		t.Fatalf("canonical input count = %d; want 1", count)
	}
}

// Unknown consumption must stay a non-success on HTTP/CLI replay while the
// exact text remains available to the existing explicit recovery flow.
func TestUnknownTextReceiptIsConflictAndRecoverable(t *testing.T) {
	srv, handler, id := attachmentTestServer(t, nil)
	msg := runtime.Message{Text: "possibly consumed", RequestID: "uncertain-text"}
	data, _ := json.Marshal(msg)
	if err := srv.stateStore.CreateAttachmentReceipt(state.AttachmentReceipt{
		AgentID: id, RequestID: msg.RequestID, Action: "steer", MessageJSON: string(data),
		PayloadHash: messagePayloadHash("steer", msg),
	}); err != nil {
		t.Fatal(err)
	}
	if err := srv.stateStore.SetAttachmentDelivery(id, msg.RequestID, "unknown"); err != nil {
		t.Fatal(err)
	}
	res, body := postJSONHandler(t, handler, "/api/sessions/"+id+"/steer", msg)
	if res.StatusCode != http.StatusConflict || !json.Valid(body) {
		t.Fatalf("uncertainty response: %d %s", res.StatusCode, body)
	}
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(body, &envelope)
	if envelope.Error.Code != runtime.CodeDeliveryUncertain {
		t.Fatalf("uncertainty code: %s", body)
	}
	recovery, err := srv.stateStore.AttachmentRecovery(id)
	if err != nil || recovery.MessageJSON != string(data) {
		t.Fatalf("recovery = %+v %v", recovery, err)
	}
	read := serveLocal(t, handler, http.MethodGet, "/api/sessions/"+id+"/prompt", nil)
	var snapshot struct {
		Recovery struct {
			Text        string            `json:"text"`
			Delivery    string            `json:"delivery"`
			Attachments []json.RawMessage `json:"attachments"`
		} `json:"recovery"`
	}
	if err := json.Unmarshal(read.Body.Bytes(), &snapshot); err != nil || read.Code != http.StatusOK ||
		snapshot.Recovery.Text != msg.Text || snapshot.Recovery.Delivery != "unknown" || snapshot.Recovery.Attachments == nil {
		t.Fatalf("text recovery wire: %d %s: %v", read.Code, read.Body.String(), err)
	}
}

// A definite prompt refusal remains a non-success on replay; a failed receipt
// must never be reported as an accepted Send.
func TestFailedTextReceiptReplayIsNotAccepted(t *testing.T) {
	srv, handler, id := attachmentTestServer(t, nil)
	msg := runtime.Message{Text: "refused input", RequestID: "failed-text"}
	data, _ := json.Marshal(msg)
	if err := srv.stateStore.CreateAttachmentReceipt(state.AttachmentReceipt{
		AgentID: id, RequestID: msg.RequestID, Action: "prompt", MessageJSON: string(data),
		PayloadHash: messagePayloadHash("prompt", msg),
	}); err != nil {
		t.Fatal(err)
	}
	if err := srv.stateStore.SetAttachmentDelivery(id, msg.RequestID, "failed"); err != nil {
		t.Fatal(err)
	}
	res, body := postJSONHandler(t, handler, "/api/sessions/"+id+"/prompt", msg)
	if res.StatusCode == http.StatusAccepted || res.StatusCode != http.StatusConflict || !json.Valid(body) {
		t.Fatalf("failed prompt replay: %d %s", res.StatusCode, body)
	}
}
