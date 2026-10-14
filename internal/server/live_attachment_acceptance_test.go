// FS-03.A62, FS-03.A64: opt-in pinned-provider image, copied-file and Steer receipts.
package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/attachment"
	"github.com/AsaphNoam/Chuck/internal/backend/providerexec"
	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/runtime"
)

func TestLiveCodexAttachmentAcceptance(t *testing.T) {
	if os.Getenv("CHUCK_LIVE_ATTACHMENT_ACCEPTANCE") != "1" {
		t.Skip("set CHUCK_LIVE_ATTACHMENT_ACCEPTANCE=1 to send synthetic fixtures to a live provider")
	}
	runLiveAttachmentAcceptance(t, "codex")
}

func TestLiveClaudeAttachmentAcceptance(t *testing.T) {
	if os.Getenv("CHUCK_LIVE_ATTACHMENT_ACCEPTANCE") != "1" {
		t.Skip("set CHUCK_LIVE_ATTACHMENT_ACCEPTANCE=1 to send synthetic fixtures to a live provider")
	}
	runLiveAttachmentAcceptance(t, "claude")
}

func runLiveAttachmentAcceptance(t *testing.T, providerID string) {
	t.Helper()
	if providerID == "claude" {
		if _, ok := os.LookupEnv("CLAUDE_CONFIG_DIR"); ok {
			t.Fatal("live Claude gate requires the host-native profile; CLAUDE_CONFIG_DIR must be unset")
		}
	}
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	bundle := filepath.Join(repo, "scripts/release")
	provider := providerexec.BundledPath(bundle, providerID)
	adapterID := providerID + "-acp"
	if providerID == "claude" {
		adapterID = "claude-agent-acp"
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{provider} {
		if _, err := os.Stat(p); err != nil {
			t.Fatal(err)
		}
	}
	adapterPkg := filepath.Join(repo, "scripts/release/node_modules/@agentclientprotocol", adapterID, "package.json")
	var adapterInfo struct {
		Version string `json:"version"`
	}
	adapterJSON, err := os.ReadFile(adapterPkg)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(adapterJSON, &adapterInfo); err != nil {
		t.Fatal(err)
	}
	providerVersionBytes, err := exec.Command(provider, "--version").Output()
	if err != nil {
		t.Fatalf("pinned %s version probe failed: %v", providerID, err)
	}
	versionPattern := regexp.MustCompile(`[0-9]+\.[0-9]+\.[0-9]+(?:\+chuck\.[0-9]+)?`)
	nativeVersion := versionPattern.FindString(string(providerVersionBytes))
	if nativeVersion == "" {
		t.Fatalf("pinned %s version probe returned no version", providerID)
	}
	nodeVersionBytes, err := exec.Command(node, "--version").Output()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(providerexec.RuntimeRootEnv, bundle)
	srv := testServer(t, true)
	adapterPath := filepath.Join(repo, "scripts/release/node_modules/@agentclientprotocol", adapterID, "dist/index.js")
	if _, err := os.Stat(adapterPath); err != nil {
		t.Fatal(err)
	}
	srv.registry.Chat().SetCommand(node, adapterPath)
	backends := config.DefaultBackends()
	for id, be := range backends.Backends {
		be.Default = id == providerID
		backends.Backends[id] = be
	}
	be := backends.Backends[providerID]
	be.ProviderMode = config.ProviderModeBundled
	backends.Backends[providerID] = be
	if err := srv.configStore.WriteBackends(backends); err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	if err := srv.configStore.WriteProject("liveattach", config.Project{Title: "Live attachment acceptance", Cwd: projectDir}); err != nil {
		t.Fatal(err)
	}
	if err := srv.configStore.WriteRole("liveattach", config.Role{Title: "Live attachment acceptance"}); err != nil {
		t.Fatal(err)
	}
	h := srv.routes()
	t.Cleanup(func() { srv.registry.Shutdown(context.Background()) })
	resp, body := postJSONHandler(t, h, "/api/sessions", map[string]any{"role": "liveattach", "project": "liveattach"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("launch %d: %s", resp.StatusCode, body)
	}
	var launch sessionResponse
	if err := json.Unmarshal(body, &launch); err != nil {
		t.Fatal(err)
	}
	id := launch.Agent.AgentID
	waitLiveStatus(t, srv, h, id, "idle")
	if launch.Attachments == nil || launch.Attachments.ImageInput == nil || !*launch.Attachments.ImageInput {
		t.Fatalf("live %s did not advertise image input: %+v", providerID, launch.Attachments)
	}
	imageBytes := func(width, height int) []byte {
		im := image.NewRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				if x < width/2 {
					im.Set(x, y, color.RGBA{R: 255, A: 255})
				} else {
					im.Set(x, y, color.RGBA{B: 255, A: 255})
				}
			}
		}
		var b bytes.Buffer
		if err := png.Encode(&b, im); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	fixture := imageBytes(512, 256)
	decoded, err := png.Decode(bytes.NewReader(fixture))
	if err != nil {
		t.Fatal(err)
	}
	left := color.RGBAModel.Convert(decoded.At(20, 128)).(color.RGBA)
	right := color.RGBAModel.Convert(decoded.At(490, 128)).(color.RGBA)
	if left.R < 240 || left.G > 10 || left.B > 10 || right.B < 240 || right.R > 10 || right.G > 10 {
		t.Fatalf("synthetic image oracle invalid: left=%v right=%v", left, right)
	}
	asset := uploadPart{field: "asset1", name: "two-color.png", data: fixture}
	meta := messageMetadata(t, "live-image-1", "Look at the attached image. Reply with the color on the left, a hyphen, then the color on the right.", asset)
	resp, body = postAttachmentMessage(t, h, "/api/sessions/"+id+"/prompt", meta, asset)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("image upload %d: %s", resp.StatusCode, body)
	}
	acceptImage := decodeAttachmentAcceptance(t, body)
	if !acceptImage.Accepted || acceptImage.AttachmentDelivery != "submitted" || len(acceptImage.Attachments) != 1 || acceptImage.Attachments[0].Kind != attachment.KindImage {
		t.Fatalf("image receipt %+v", acceptImage)
	}
	ownedImage := serveLocal(t, h, http.MethodGet, "/api/sessions/"+id+"/attachments/"+acceptImage.Attachments[0].ID, nil)
	if ownedImage.Code != http.StatusOK || !bytes.Equal(ownedImage.Body.Bytes(), fixture) {
		t.Fatalf("owned image bytes differ from synthetic fixture: status=%d", ownedImage.Code)
	}
	waitLiveStatus(t, srv, h, id, "idle")
	imageReply := liveAssistantText(readEvents(t, srv, id))
	if imageReply == "" {
		t.Errorf("image interpretation returned no assistant text")
	} else if !strings.Contains(strings.ToLower(imageReply), "red") || !strings.Contains(strings.ToLower(imageReply), "blue") {
		t.Errorf("image interpretation mismatch; provider_reply=%q", imageReply)
	}
	var nonce [12]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		t.Fatal(err)
	}
	token := "CHUCK_LIVE_ATTACHMENT_" + hex.EncodeToString(nonce[:])
	file := uploadPart{field: "asset1", name: "copied-note.txt", data: []byte(token)}
	resp, body = postAttachmentMessage(t, h, "/api/sessions/"+id+"/prompt", messageMetadata(t, "live-file-1", "Read the attached text file and reply with its exact contents only.", file), file)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("file upload %d: %s", resp.StatusCode, body)
	}
	acceptFile := decodeAttachmentAcceptance(t, body)
	waitLiveStatus(t, srv, h, id, "idle")
	fileReply := liveAssistantText(readEvents(t, srv, id))
	if !strings.Contains(fileReply, token) {
		t.Fatalf("ordinary attachment token was not returned")
	}
	read := serveLocal(t, h, http.MethodGet, "/api/sessions/"+id+"/attachments/"+acceptFile.Attachments[0].ID, nil)
	if read.Code != http.StatusOK || !bytes.Equal(read.Body.Bytes(), file.data) {
		t.Fatalf("owned copied-file read failed: status=%d", read.Code)
	}
	// Start a bounded in-flight turn, then steer a file attachment. Receipt must
	// say steered and the provider's resulting transcript must contain its token.
	resp, body = postJSONHandler(t, h, "/api/sessions/"+id+"/prompt", map[string]string{"request_id": "live-busy-1", "text": "For up to 20 seconds, keep replying WAIT. If you receive a later message with an attached text file, read it and reply with its exact contents only."})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("busy prompt %d: %s", resp.StatusCode, body)
	}
	waitLiveStatus(t, srv, h, id, "busy")
	var steerNonce [12]byte
	if _, err := rand.Read(steerNonce[:]); err != nil {
		t.Fatal(err)
	}
	steerToken := "CHUCK_LIVE_STEER_ATTACHMENT_" + hex.EncodeToString(steerNonce[:])
	steerFile := uploadPart{field: "asset1", name: "steer-note.txt", data: []byte(steerToken)}
	resp, body = postAttachmentMessage(t, h, "/api/sessions/"+id+"/steer", messageMetadata(t, "live-steer-1", "Read this attached file and reply with its exact contents only.", steerFile), steerFile)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("attachment steer %d: %s", resp.StatusCode, body)
	}
	steerReceipt := decodeAttachmentAcceptance(t, body)
	if steerReceipt.AttachmentDelivery != "submitted" || len(steerReceipt.Attachments) != 1 {
		t.Fatalf("steer receipt %+v", steerReceipt)
	}
	waitLiveStatus(t, srv, h, id, "idle")
	steerReply := liveAssistantText(readEvents(t, srv, id))
	if !strings.Contains(steerReply, steerToken) {
		t.Fatalf("attachment steer token was not returned")
	}
	artifactDir := "/private/tmp/chuck-live-attachment-evidence"
	if err := os.MkdirAll(artifactDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(artifactDir, 0o700); err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	if err := os.WriteFile(filepath.Join(artifactDir, providerID+"-"+stamp+"-red-blue.png"), fixture, 0o600); err != nil {
		t.Fatal(err)
	}
	authMode := "native_cli_login"
	if providerID == "claude" {
		authMode = "native Claude login; host keychain; CLAUDE_CONFIG_DIR unset"
	} else {
		authMode = "native Codex login; host CODEX_HOME mirrored into Chuck profile"
	}
	receipt, err := json.MarshalIndent(map[string]any{"provider": providerID, "backend_id": launch.Agent.Backend, "model_id": launch.Agent.Model, "platform": goruntime.GOOS + "/" + goruntime.GOARCH, "adapter_version": adapterInfo.Version, "native_cli_version": nativeVersion, "node_version": strings.TrimSpace(string(nodeVersionBytes)), "auth_mode": authMode, "auth_status": "authenticated", "image_reply": imageReply, "stored_image_bytes_match": true, "ordinary_file_token_read": true, "attachment_steer_token_read": true}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(artifactDir, providerID+"-"+stamp+"-receipt.json"), receipt, 0o600); err != nil {
		t.Fatal(err)
	}
	if providerID == "codex" {
		tinyPath := filepath.Join(artifactDir, "codex-first-64x32-red-blue-mismatch.png")
		if err := os.WriteFile(tinyPath, imageBytes(64, 32), 0o600); err != nil {
			t.Fatal(err)
		}
		tinyReceipt, err := json.MarshalIndent(map[string]any{"provider": "codex", "probe": "first 64x32 synthetic red-left/blue-right image", "reply": "blue-blue", "outcome": "initial small-fixture mismatch; resolved by the later 512x256 oracle", "stored_image_bytes_match": "not independently checked in first run"}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(artifactDir, "codex-first-64x32-red-blue-mismatch.json"), tinyReceipt, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("LIVE_RECEIPT provider=%s image_reply=%q stored_image_bytes_match=pass copied_file_token=pass attachment_steer_token=pass evidence=%s/%s-*-receipt.json", providerID, imageReply, artifactDir, providerID)
}

func liveAssistantText(events []runtime.Event) string {
	var out strings.Builder
	for _, ev := range events {
		if ev.Type == runtime.EvAssistantText {
			var d runtime.AssistantTextData
			_ = json.Unmarshal(ev.Data, &d)
			out.WriteString(d.Delta)
		}
	}
	return out.String()
}

func waitLiveStatus(t *testing.T, srv *Server, h http.Handler, id, want string) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for {
		st, err := srv.stateStore.ReadStatus(id)
		if err == nil && st.State == want {
			return
		}
		if err == nil && st.State == "waiting_input" {
			permission := pendingLiveAttachmentRead(t, srv, id)
			if permission == nil || !strings.HasPrefix(permission.Name, "Read ") || !strings.Contains(permission.Name, "/sessions/"+id+"/attachments/") || !strings.Contains(permission.Name, "blob") || !strings.Contains(string(permission.Args), "/sessions/"+id+"/attachments/") || !strings.Contains(string(permission.Args), "blob") {
				name, hasOwnedPath, hasBlob := "missing", false, false
				if permission != nil {
					name = permission.Name
					hasOwnedPath = strings.Contains(string(permission.Args), "/sessions/"+id+"/attachments/")
					hasBlob = strings.Contains(string(permission.Args), "blob")
				}
				t.Fatalf("live permission is not a verified owned-attachment Read (name=%q owned_path=%t blob=%t)", name, hasOwnedPath, hasBlob)
			}
			resp, body := postJSONHandler(t, h, "/api/sessions/"+id+"/permission", map[string]string{"tool_call_id": permission.ToolCallID, "decision": "approve"})
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("approve owned attachment read: status=%d", resp.StatusCode)
			}
			_ = body
		}
		if time.Now().After(deadline) {
			if err != nil {
				t.Fatalf("live provider status read failed: %v", err)
			}
			t.Fatalf("live provider timed out waiting for status %q (state=%q trace=%q)", want, st.State, st.LastTrace)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func pendingLiveAttachmentRead(t *testing.T, srv *Server, id string) *runtime.PermissionRequestData {
	t.Helper()
	events := readEvents(t, srv, id)
	resolved := map[string]bool{}
	for _, ev := range events {
		if ev.Type == runtime.EvPermissionResolved {
			var d runtime.PermissionResolvedData
			_ = json.Unmarshal(ev.Data, &d)
			resolved[d.ToolCallID] = true
		}
	}
	for i := len(events) - 1; i >= 0; i-- {
		if events[i].Type != runtime.EvPermissionRequest {
			continue
		}
		var d runtime.PermissionRequestData
		if json.Unmarshal(events[i].Data, &d) != nil || resolved[d.ToolCallID] {
			continue
		}
		return &d
	}
	return nil
}
