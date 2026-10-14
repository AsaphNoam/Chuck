package server

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"testing"
)

// FS-03.A64: a stopped agent whose snapshot predates image negotiation must
// permit Send to reach fresh capability validation without manual Resume.
func TestStoppedAgentWithPreImageSnapshotDoesNotRefuseImages(t *testing.T) {
	t.Run("fresh peer accepts images", func(t *testing.T) { testLegacyImageWake(t, true) })
	t.Run("fresh peer refuses images", func(t *testing.T) { testLegacyImageWake(t, false) })
}

func testLegacyImageWake(t *testing.T, supportsImage bool) {
	env := map[string]string{}
	if supportsImage {
		env["FAKEACP_IMAGE_INPUT"] = "1"
	}
	srv, h, id := attachmentTestServer(t, env)
	if rec := serveLocal(t, h, http.MethodPost, "/api/sessions/"+id+"/stop", nil); rec.Code >= 300 {
		t.Fatalf("stop = %d %s", rec.Code, rec.Body.String())
	}
	// The snapshot shape written by a build before image negotiation existed.
	if _, err := srv.stateStore.DB().Exec(`UPDATE sessions SET runtime_capabilities_json = ? WHERE agent_id = ?`,
		`{"fork":false,"subagents":false,"background_tasks":false,"background_task_stop":false,"file_change_reports":false,"session_failure":false}`, id); err != nil {
		t.Fatal(err)
	}
	rec := serveLocal(t, h, http.MethodGet, "/api/sessions/"+id, nil)
	var session struct {
		Attachments map[string]any `json:"attachment_capabilities"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &session); err != nil {
		t.Fatalf("session read %d %s: %v", rec.Code, rec.Body.String(), err)
	}
	update, err := srv.stateMgr.Touch(id)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot struct {
		Attachments map[string]any `json:"attachment_capabilities"`
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatal(err)
	}
	for surface, caps := range map[string]map[string]any{"session": session.Attachments, "state_update": snapshot.Attachments} {
		if caps == nil || caps["files"] != true {
			t.Fatalf("%s lost attachment affordances: %v", surface, caps)
		}
		if _, ok := caps["image_input"]; ok {
			t.Errorf("%s must omit unknown image_input: %v", surface, caps)
		}
	}
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	part := uploadPart{field: "image", name: "photo.png", data: pngBytes.Bytes()}
	resp, body := postAttachmentMessage(t, h, "/api/sessions/"+id+"/prompt", messageMetadata(t, "wake-image", "describe", part), part)
	want := http.StatusAccepted
	if !supportsImage {
		want = http.StatusUnprocessableEntity
		if !bytes.Contains(body, []byte("attachment_capability")) {
			t.Fatalf("missing actionable capability refusal: %s", body)
		}
		if transcript, err := srv.registry.Transcript(id); err != nil {
			t.Fatal(err)
		} else {
			for _, event := range transcript {
				if event.Type == "user_text" && bytes.Contains(event.Data, []byte("wake-image")) {
					t.Fatal("unsupported image consumed before refusal")
				}
			}
		}
	}
	if resp.StatusCode != want {
		t.Fatalf("image to stopped agent = %d %s, want %d", resp.StatusCode, body, want)
	}
}
