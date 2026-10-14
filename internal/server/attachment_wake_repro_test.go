package server

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"testing"
)

// Reproduces the HANDOFF review finding "Stopped agents from before image
// negotiation refuse images": a stopped agent whose capability snapshot predates
// the image_input key projects image_input:false, so both composers block an
// image Send that stopped-agent wake would deliver (FS-03.R83, TS-04.R90).
func TestStoppedAgentWithPreImageSnapshotDoesNotRefuseImages(t *testing.T) {
	t.Skip("reproduction for open investigation finding; the fix un-skips it")
	srv, h, id := attachmentTestServer(t, map[string]string{"FAKEACP_IMAGE_INPUT": "1"})
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
	if v, ok := session.Attachments["image_input"]; ok && v == false {
		t.Errorf("stopped agent without an image advertisement on record projects image_input:false; the composer refuses an image wake would deliver: %s", rec.Body.String())
	}
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	part := uploadPart{field: "image", name: "photo.png", data: pngBytes.Bytes()}
	resp, body := postAttachmentMessage(t, h, "/api/sessions/"+id+"/prompt", messageMetadata(t, "wake-image", "describe", part), part)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("image to stopped image-capable agent = %d %s", resp.StatusCode, body)
	}
}
