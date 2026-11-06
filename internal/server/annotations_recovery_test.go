package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
)

// FS-13.R37 inherits FS-03.R100: test the real HTTP/runtime/SQLite boundary,
// without acknowledging the withdrawn draft (a lost response or failed save).
func TestSelfAnnotationRetainsReleasedInput(t *testing.T) {
	for _, stop := range []bool{false, true} {
		name := "withdraw"
		if stop {
			name = "stop"
		}
		t.Run(name, func(t *testing.T) {
			srv, handler, id, _ := busyAttachmentTestServer(t)
			batch := runtime.AnnotationData{CommandID: "recover-1", Annotations: []runtime.Annotation{{Seq: 1, Excerpt: "target phrase", Instruction: "recover exact instruction"}}, Target: runtime.AnnotationTarget{Kind: "self"}}
			res, body := postJSONHandler(t, handler, "/api/sessions/"+id+"/annotations", batch)
			if res.StatusCode != http.StatusAccepted {
				t.Fatalf("annotation: %d %s", res.StatusCode, body)
			}
			held, _, err := srv.registry.HeldMessage(id)
			if err != nil || held.RequestID != "annotation-recover-1" {
				t.Fatalf("held: %+v %v", held, err)
			}
			if stop {
				if err := srv.registry.Stop(context.Background(), id); err != nil {
					t.Fatal(err)
				}
			} else {
				withdrawn := serveLocal(t, handler, http.MethodDelete, "/api/sessions/"+id+"/prompt", nil)
				var wire struct {
					Withdrawn bool            `json:"withdrawn"`
					Message   runtime.Message `json:"message"`
				}
				if err := json.Unmarshal(withdrawn.Body.Bytes(), &wire); err != nil || !wire.Withdrawn || wire.Message.RequestID != held.RequestID || wire.Message.Text != held.Text {
					t.Fatalf("withdraw: %v %s", err, withdrawn.Body)
				}
			}
			rec, err := srv.stateStore.AttachmentReceipt(id, held.RequestID)
			var saved runtime.Message
			if err != nil || rec.Delivery != "released" || json.Unmarshal([]byte(rec.MessageJSON), &saved) != nil || saved.Text != held.Text {
				t.Fatalf("released receipt: %+v %v", rec, err)
			}
			read := serveLocal(t, handler, http.MethodGet, "/api/sessions/"+id+"/prompt", nil)
			if read.Code != http.StatusOK || !json.Valid(read.Body.Bytes()) {
				t.Fatalf("recovery read: %s", read.Body)
			}
			// An unacknowledged withdrawal remains available on subsequent reads.
			if !stop {
				for range 2 {
					read := serveLocal(t, handler, http.MethodGet, "/api/sessions/"+id+"/prompt", nil)
					var wire struct {
						Recovery runtime.Message `json:"recovery"`
					}
					if err := json.Unmarshal(read.Body.Bytes(), &wire); err != nil || wire.Recovery.RequestID != held.RequestID || wire.Recovery.Text != held.Text {
						t.Fatalf("recovery: %s %v", read.Body, err)
					}
				}
			}
			res, body = postJSONHandler(t, handler, "/api/sessions/"+id+"/annotations", batch)
			if res.StatusCode != http.StatusAccepted || annotationEventCount(t, srv, id) != 1 {
				t.Fatalf("replay: %d %s", res.StatusCode, body)
			}
		})
	}
}

// Stop while the native steer request is in flight: consumption is unknown,
// so recovery must retain exact bytes and must never restore an automatic hold.
func TestSelfAnnotationUncertainSteerRecovery(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "steer.log")
	srv, ts, id, _ := busyChatServer(t, map[string]string{"FAKEACP_STEERING": "1", "FAKEACP_STEER_OUTCOME": "promptRequired", "FAKEACP_STEER_LOG": log, "FAKEACP_STEER_WAIT_FILE": filepath.Join(dir, "never-release")})
	type result struct {
		code int
		body []byte
	}
	done := make(chan result, 1)
	go func() { code, body := selfAnnotation(t, ts.URL, id, "uncertain-1", true); done <- result{code, body} }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if raw, err := os.ReadFile(log); err == nil && len(raw) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("steer did not reach provider")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := srv.registry.Stop(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-done:
		var wire struct {
			Error runtime.APIError `json:"error"`
		}
		if err := json.Unmarshal(got.body, &wire); err != nil || got.code != http.StatusConflict || wire.Error.Code != runtime.CodeDeliveryUncertain {
			t.Fatalf("uncertainty: %d %s", got.code, got.body)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("steer did not settle")
	}
	rec, err := srv.stateStore.AttachmentReceipt(id, "annotation-uncertain-1")
	var saved runtime.Message
	if err != nil || rec.Delivery != "unknown" || json.Unmarshal([]byte(rec.MessageJSON), &saved) != nil || saved.Text == "" {
		t.Fatalf("uncertain recovery: %+v %v", rec, err)
	}
	if held, _, _ := srv.registry.HeldMessage(id); held.Text != "" {
		t.Fatalf("uncertain input restored as hold: %+v", held)
	}
	read := serveLocal(t, srv.routes(), http.MethodGet, "/api/sessions/"+id+"/prompt", nil)
	var wire struct {
		Recovery struct {
			RequestID string `json:"request_id"`
			Text      string `json:"text"`
			Delivery  string `json:"delivery"`
		} `json:"recovery"`
	}
	if err := json.Unmarshal(read.Body.Bytes(), &wire); err != nil || read.Code != http.StatusOK || wire.Recovery.RequestID != saved.RequestID || wire.Recovery.Text != saved.Text || wire.Recovery.Delivery != "unknown" {
		t.Fatalf("recovery: %s", read.Body)
	}
}
