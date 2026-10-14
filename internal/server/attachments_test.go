// FS-03.A62, FS-03.A64, FS-03.A65: multipart delivery, retry identity and admission bounds.
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/attachment"
	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/runtime"
)

type uploadPart struct {
	field string
	name  string
	data  []byte
}

func attachmentTestServer(t *testing.T, env map[string]string) (*Server, http.Handler, string) {
	t.Helper()
	fake := buildFakeACP(t)
	t.Setenv("FAKEACP_SCENARIO", "stream_text")
	for k, v := range env {
		t.Setenv(k, v)
	}
	srv := testServer(t, true)
	srv.registry.Chat().SetCommand(fake)
	if err := srv.configStore.WriteProject("attachproj", config.Project{Title: "Attachments", Cwd: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	if err := srv.configStore.WriteRole("attachrole", config.Role{Title: "Attachment test"}); err != nil {
		t.Fatal(err)
	}
	h := srv.routes()
	t.Cleanup(func() { srv.registry.Shutdown(context.Background()) })
	resp, body := postJSONHandler(t, h, "/api/sessions", map[string]any{"role": "attachrole", "project": "attachproj"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("launch = %d %s", resp.StatusCode, body)
	}
	var launched sessionResponse
	if err := json.Unmarshal(body, &launched); err != nil || launched.Agent.AgentID == "" {
		t.Fatalf("launch response %s: %v", body, err)
	}
	waitStatus(t, srv, launched.Agent.AgentID, "idle")
	return srv, h, launched.Agent.AgentID
}

func busyAttachmentTestServer(t *testing.T) (*Server, http.Handler, string, func()) {
	t.Helper()
	fake := buildFakeACP(t)
	holdFile := filepath.Join(t.TempDir(), "hold")
	t.Setenv("FAKEACP_SCENARIO", "hold_turn")
	t.Setenv("FAKEACP_HOLD_FILE", holdFile)
	srv := testServer(t, true)
	srv.registry.Chat().SetCommand(fake)
	if err := srv.configStore.WriteProject("attachproj", config.Project{Title: "Attachments", Cwd: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	if err := srv.configStore.WriteRole("attachrole", config.Role{Title: "Attachment test"}); err != nil {
		t.Fatal(err)
	}
	h := srv.routes()
	t.Cleanup(func() { _ = os.WriteFile(holdFile, []byte("go"), 0o600); srv.registry.Shutdown(context.Background()) })
	resp, body := postJSONHandler(t, h, "/api/sessions", map[string]any{"role": "attachrole", "project": "attachproj"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("launch = %d %s", resp.StatusCode, body)
	}
	var launched sessionResponse
	if err := json.Unmarshal(body, &launched); err != nil || launched.Agent.AgentID == "" {
		t.Fatalf("launch response %s: %v", body, err)
	}
	id := launched.Agent.AgentID
	waitStatus(t, srv, id, "idle")
	resp, body = postJSONHandler(t, h, "/api/sessions/"+id+"/prompt", map[string]string{"text": "first"})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("initial prompt = %d %s", resp.StatusCode, body)
	}
	waitStatus(t, srv, id, "busy")
	release := func() { _ = os.WriteFile(holdFile, []byte("go"), 0o600) }
	return srv, h, id, release
}

func postJSONHandler(t *testing.T, h http.Handler, path string, body any) (*http.Response, []byte) {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := newLocalRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	resp := rec.Result()
	resp.Body.Close()
	return resp, append([]byte(nil), rec.Body.Bytes()...)
}

func postAttachmentMessage(t *testing.T, h http.Handler, path, metadata string, parts ...uploadPart) (*http.Response, []byte) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("message", metadata); err != nil {
		t.Fatal(err)
	}
	for _, file := range parts {
		p, err := mw.CreateFormFile(file.field, file.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.Write(file.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := newLocalRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	resp := rec.Result()
	resp.Body.Close()
	return resp, append([]byte(nil), rec.Body.Bytes()...)
}

func serveLocal(t *testing.T, h http.Handler, method, path string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newLocalRequest(method, path, body))
	return rec
}

func messageMetadata(t *testing.T, requestID, text string, files ...uploadPart) string {
	t.Helper()
	var declarations []map[string]string
	for _, file := range files {
		declarations = append(declarations, map[string]string{"part": file.field, "name": file.name})
	}
	b, err := json.Marshal(map[string]any{"request_id": requestID, "text": text, "files": declarations})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

type attachmentAcceptance struct {
	Accepted           bool                    `json:"accepted"`
	RequestID          string                  `json:"request_id"`
	Delivery           string                  `json:"delivery"`
	AttachmentDelivery string                  `json:"attachment_delivery"`
	Attachments        []attachment.Descriptor `json:"attachments"`
}

func decodeAttachmentAcceptance(t *testing.T, body []byte) attachmentAcceptance {
	t.Helper()
	var out attachmentAcceptance
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode acceptance %s: %v", body, err)
	}
	return out
}

func attachmentFiles(t *testing.T, srv *Server, agentID string) []os.DirEntry {
	t.Helper()
	dir := filepath.Join(srv.configStore.Home(), "sessions", agentID, "attachments")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	files := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".") {
			files = append(files, entry)
		}
	}
	return files
}

// TS-02.R47 / FS-03.R85: a corrupt owned copy stays a per-item error after
// restart; it neither loses ownership nor blocks healthy attachment delivery.
func TestRestartWithCorruptOwnedAttachmentKeepsDeliveryAvailable(t *testing.T) {
	srv, h, broken := attachmentTestServer(t, nil)
	resp, body := postJSONHandler(t, h, "/api/sessions", map[string]any{"role": "attachrole", "project": "attachproj"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("second launch = %d %s", resp.StatusCode, body)
	}
	var second sessionResponse
	if err := json.Unmarshal(body, &second); err != nil {
		t.Fatal(err)
	}
	healthy := second.Agent.AgentID
	waitStatus(t, srv, healthy, "idle")

	part := uploadPart{field: "asset1", name: "kept.txt", data: []byte("kept bytes")}
	resp, body = postAttachmentMessage(t, h, "/api/sessions/"+broken+"/prompt", messageMetadata(t, "kept", "keep", part), part)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("seed Send = %d %s", resp.StatusCode, body)
	}
	kept := decodeAttachmentAcceptance(t, body).Attachments[0]
	waitStatus(t, srv, broken, "idle")
	store := attachment.New(srv.configStore.Home())
	orphan, err := store.Put(healthy, "orphan.txt", strings.NewReader("never accepted"))
	if err != nil {
		t.Fatal(err)
	}
	keptDir := filepath.Join(srv.configStore.Home(), "sessions", broken, "attachments", kept.ID)
	if err = os.WriteFile(filepath.Join(keptDir, "pending"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(keptDir, "blob"), []byte("corrupt!!!"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Restart recovery.
	srv.attachmentStartupErr = srv.recoverAttachmentStorage()
	if srv.attachmentStartupErr != nil {
		t.Fatalf("corrupt owned copy disabled attachment recovery: %v", srv.attachmentStartupErr)
	}
	if _, err = os.Stat(filepath.Join(keptDir, "manifest.json")); err != nil {
		t.Fatalf("corrupt owned copy lost its stored identity: %v", err)
	}
	if _, err = os.Stat(filepath.Join(keptDir, "pending")); !os.IsNotExist(err) {
		t.Fatalf("durably owned copy stayed provisional: %v", err)
	}
	if f, _, err := store.Open(healthy, orphan.ID); err == nil {
		f.Close()
		t.Fatal("healthy conversation's unaccepted copy survived recovery")
	}
	if rec := serveLocal(t, h, http.MethodGet, "/api/sessions/"+broken+"/attachments/"+kept.ID, nil); rec.Code == http.StatusOK {
		t.Fatalf("corrupt owned copy read = %d, want a per-item error", rec.Code)
	}
	for i, id := range []string{healthy, broken} {
		fresh := uploadPart{field: "asset1", name: "fresh.txt", data: []byte("fresh bytes")}
		resp, body = postAttachmentMessage(t, h, "/api/sessions/"+id+"/prompt", messageMetadata(t, "fresh-"+string(rune('a'+i)), "fresh", fresh), fresh)
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("new upload after recovery in %s = %d %s", id, resp.StatusCode, body)
		}
		waitStatus(t, srv, id, "idle")
	}
}

func TestMultipartSendPersistsBytesAndRetryIsIdempotent(t *testing.T) {
	srv, h, id := attachmentTestServer(t, nil)
	part := uploadPart{field: "asset1", name: "reference.txt", data: []byte("copied host file")}
	meta := messageMetadata(t, "send-1", "read this", part)
	path := "/api/sessions/" + id + "/prompt"
	resp, body := postAttachmentMessage(t, h, path, meta, part)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("multipart Send = %d %s", resp.StatusCode, body)
	}
	accepted := decodeAttachmentAcceptance(t, body)
	if !accepted.Accepted || accepted.RequestID != "send-1" || accepted.AttachmentDelivery != "submitted" || len(accepted.Attachments) != 1 {
		t.Fatalf("acceptance = %+v", accepted)
	}
	want := accepted.Attachments[0]
	if want.Name != part.name || want.Size != int64(len(part.data)) || want.Kind != attachment.KindFile || want.MIMEType == "" {
		t.Fatalf("descriptor = %+v", want)
	}
	waitStatus(t, srv, id, "idle")
	events := readEvents(t, srv, id)
	userEvents := 0
	for _, ev := range events {
		if ev.Type != runtime.EvUserPrompt {
			continue
		}
		userEvents++
		var msg runtime.UserPromptData
		if err := json.Unmarshal(ev.Data, &msg); err != nil {
			t.Fatal(err)
		}
		if msg.Text != "read this" || msg.RequestID != "send-1" || len(msg.Attachments) != 1 || msg.Attachments[0] != want {
			t.Fatalf("durable user event = %+v", msg)
		}
	}
	if userEvents != 1 {
		t.Fatalf("user events = %d, want one", userEvents)
	}

	readRec := serveLocal(t, h, http.MethodGet, "/api/sessions/"+id+"/attachments/"+want.ID, nil)
	if readRec.Code != http.StatusOK || !bytes.Equal(readRec.Body.Bytes(), part.data) || readRec.Header().Get("X-Content-Type-Options") != "nosniff" || readRec.Header().Get("Cache-Control") != "private, no-store" || !strings.HasPrefix(readRec.Header().Get("Content-Disposition"), "attachment;") {
		t.Fatalf("owned file read = %d headers=%v body=%q", readRec.Code, readRec.Header(), readRec.Body.Bytes())
	}

	resp, body = postAttachmentMessage(t, h, path, meta, part)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("same-id retry = %d %s", resp.StatusCode, body)
	}
	retry := decodeAttachmentAcceptance(t, body)
	if len(retry.Attachments) != 1 || retry.Attachments[0] != want || len(attachmentFiles(t, srv, id)) != 1 {
		t.Fatalf("retry changed owned identity: %+v files=%d", retry, len(attachmentFiles(t, srv, id)))
	}
	events = readEvents(t, srv, id)
	userEvents = 0
	for _, ev := range events {
		if ev.Type == runtime.EvUserPrompt {
			userEvents++
		}
	}
	if userEvents != 1 {
		t.Fatalf("retry created %d user events", userEvents)
	}

	changed := uploadPart{field: "asset1", name: "reference.txt", data: []byte("changed payload")}
	resp, body = postAttachmentMessage(t, h, path, messageMetadata(t, "send-1", "read this", changed), changed)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("changed retry = %d %s, want conflict", resp.StatusCode, body)
	}
	if len(attachmentFiles(t, srv, id)) != 1 {
		t.Fatalf("conflicting retry leaked a blob: %d", len(attachmentFiles(t, srv, id)))
	}
}

func TestAttachmentHoldWithdrawalReplacementAndRecoveryResubmit(t *testing.T) {
	srv, h, id, _ := busyAttachmentTestServer(t)
	upload := func(requestID, text string, data string) attachmentAcceptance {
		t.Helper()
		part := uploadPart{field: "asset", name: requestID + ".txt", data: []byte(data)}
		resp, body := postAttachmentMessage(t, h, "/api/sessions/"+id+"/prompt", messageMetadata(t, requestID, text, part), part)
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("held Send %s = %d %s", requestID, resp.StatusCode, body)
		}
		return decodeAttachmentAcceptance(t, body)
	}
	first := upload("held-1", "first", "one")
	if first.AttachmentDelivery != "held" || len(first.Attachments) != 1 {
		t.Fatalf("first held = %+v", first)
	}
	withdraw := serveLocal(t, h, http.MethodDelete, "/api/sessions/"+id+"/prompt", nil)
	if withdraw.Code != http.StatusOK {
		t.Fatalf("withdraw = %d %s", withdraw.Code, withdraw.Body)
	}
	second := upload("held-2", "second", "two")
	if second.AttachmentDelivery != "held" {
		t.Fatalf("replacement held = %+v", second)
	}
	resp := serveLocal(t, h, http.MethodGet, "/api/sessions/"+id+"/prompt", nil)
	var prompt struct {
		RequestID   string                  `json:"request_id"`
		Attachments []attachment.Descriptor `json:"attachments"`
		Recovery    runtime.Message         `json:"recovery"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prompt); err != nil {
		t.Fatal(err)
	}
	if prompt.RequestID != "held-2" || len(prompt.Attachments) != 1 || prompt.Recovery.RequestID != "held-1" || len(prompt.Recovery.Attachments) != 1 {
		t.Fatalf("held and recovery = %+v", prompt)
	}

	// Withdraw the replacement to make it the latest recovery record, then
	// resubmit that exact copy as explicit new intent.
	withdraw = serveLocal(t, h, http.MethodDelete, "/api/sessions/"+id+"/prompt", nil)
	if withdraw.Code != http.StatusOK {
		t.Fatalf("withdraw replacement = %d %s", withdraw.Code, withdraw.Body)
	}
	resp = serveLocal(t, h, http.MethodGet, "/api/sessions/"+id+"/prompt", nil)
	if err := json.NewDecoder(resp.Body).Decode(&prompt); err != nil {
		t.Fatal(err)
	}
	if prompt.Recovery.RequestID != "held-2" {
		t.Fatalf("latest recovery = %+v", prompt.Recovery)
	}
	jsonBody, _ := json.Marshal(map[string]any{"request_id": "recovered-2", "text": "retry second", "attachment_ids": []string{prompt.Recovery.Attachments[0].ID}})
	resp2, body := postJSONHandler(t, h, "/api/sessions/"+id+"/prompt", json.RawMessage(jsonBody))
	if resp2.StatusCode != http.StatusAccepted {
		t.Fatalf("recovered resubmit = %d %s", resp2.StatusCode, body)
	}
	accepted := decodeAttachmentAcceptance(t, body)
	if accepted.RequestID != "recovered-2" || accepted.AttachmentDelivery != "held" || len(accepted.Attachments) != 1 || accepted.Attachments[0].ID != prompt.Recovery.Attachments[0].ID {
		t.Fatalf("recovered receipt = %+v", accepted)
	}
	if len(attachmentFiles(t, srv, id)) != 2 {
		t.Fatalf("recovery resubmit duplicated blobs: %d", len(attachmentFiles(t, srv, id)))
	}
	resp = serveLocal(t, h, http.MethodGet, "/api/sessions/"+id+"/prompt", nil)
	var after struct {
		Recovery *runtime.Message `json:"recovery"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&after); err != nil {
		t.Fatal(err)
	}
	if after.Recovery == nil || after.Recovery.RequestID != "held-1" {
		t.Fatalf("older released recovery was not preserved: %+v", after.Recovery)
	}
}

func TestMultipartValidationRollsBackOwnedBlobs(t *testing.T) {
	srv, h, id := attachmentTestServer(t, nil)
	path := "/api/sessions/" + id + "/prompt"
	valid := uploadPart{field: "f1", name: "ok.txt", data: []byte("valid")}
	cases := []struct {
		name     string
		metadata string
		parts    []uploadPart
	}{
		{name: "malformed metadata", metadata: `{`, parts: []uploadPart{valid}},
		{name: "missing part", metadata: messageMetadata(t, "missing", "", valid)},
		{name: "duplicate declaration", metadata: `{"request_id":"dup","files":[{"part":"f1","name":"a"},{"part":"f1","name":"b"}]}`, parts: []uploadPart{valid}},
		{name: "undeclared trailing part", metadata: messageMetadata(t, "extra", "", valid), parts: []uploadPart{valid, uploadPart{field: "f2", name: "extra.txt", data: []byte("extra")}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body := postAttachmentMessage(t, h, path, tc.metadata, tc.parts...)
			if resp.StatusCode < 400 || resp.StatusCode >= 500 {
				t.Fatalf("malformed upload = %d %s", resp.StatusCode, body)
			}
			if got := len(attachmentFiles(t, srv, id)); got != 0 {
				t.Fatalf("malformed upload retained %d blobs", got)
			}
		})
	}
}

func TestImageCapabilityRejectionKeepsNoAcceptedBytes(t *testing.T) {
	srv, h, id := attachmentTestServer(t, nil)
	var pngBytes bytes.Buffer
	if err := png.Encode(&pngBytes, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	part := uploadPart{field: "image", name: "photo.png", data: pngBytes.Bytes()}
	resp, body := postAttachmentMessage(t, h, "/api/sessions/"+id+"/prompt", messageMetadata(t, "image-1", "describe", part), part)
	if resp.StatusCode < 400 {
		t.Fatalf("unsupported image = %d %s", resp.StatusCode, body)
	}
	if len(attachmentFiles(t, srv, id)) != 0 {
		t.Fatal("unsupported image remained owned")
	}
	if _, err := srv.stateStore.AttachmentReceipt(id, "image-1"); err == nil {
		t.Fatal("unsupported image left an accepted receipt")
	}
	for _, ev := range readEvents(t, srv, id) {
		if ev.Type == runtime.EvUserPrompt {
			t.Fatal("unsupported image created a user event")
		}
	}
}

func TestLocalImageReadAndNarrowRemoteMultipartLimit(t *testing.T) {
	srv, h, id := attachmentTestServer(t, nil)
	project, err := srv.configStore.ReadProject("attachproj")
	if err != nil {
		t.Fatal(err)
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project.Cwd, "local.png"), encoded.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	imageRec := serveLocal(t, h, http.MethodGet, "/api/sessions/"+id+"/image?path=local.png", nil)
	if imageRec.Code != http.StatusOK || !bytes.Equal(imageRec.Body.Bytes(), encoded.Bytes()) || imageRec.Header().Get("Content-Type") != "image/png" || imageRec.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.HasPrefix(imageRec.Header().Get("Content-Disposition"), "inline;") {
		t.Fatalf("local image read = %d headers=%v", imageRec.Code, imageRec.Header())
	}

	large := bytes.Repeat([]byte("x"), remoteBodyLimit+1)
	probe := func(w http.ResponseWriter, r *http.Request) {
		_, err := io.Copy(io.Discard, r.Body)
		if err != nil {
			http.Error(w, "body rejected", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
	filter := remoteBodyFilter(http.HandlerFunc(probe))
	part := uploadPart{field: "f", name: "big.bin", data: large}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("message", `{}`)
	f, _ := mw.CreateFormFile(part.field, part.name)
	_, _ = f.Write(part.data)
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/sessions/"+id+"/prompt", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	filter.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("multipart phone body = %d %s", rec.Code, rec.Body)
	}

	ordinary := httptest.NewRequest(http.MethodPost, "/api/sessions/"+id+"/rename", strings.NewReader(strings.Repeat("x", remoteBodyLimit+1)))
	rec = httptest.NewRecorder()
	filter.ServeHTTP(rec, ordinary)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("ordinary phone body = %d", rec.Code)
	}
	if !isMultipartMessageRoute(req) {
		t.Fatal("prompt multipart route was not recognized")
	}
	if isMultipartMessageRoute(httptest.NewRequest(http.MethodPost, "/api/sessions/"+id+"/rename", strings.NewReader("x"))) {
		t.Fatal("unrelated route matched multipart exception")
	}
}
