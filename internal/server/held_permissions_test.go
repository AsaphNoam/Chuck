package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

// permissionAppendFailureWriter fails only the permission_request append.
type permissionAppendFailureWriter struct{ runtime.TranscriptWriter }

func (w permissionAppendFailureWriter) Append(ev runtime.Event) error {
	if ev.Type == runtime.EvPermissionRequest {
		return errors.New("injected request append failure")
	}
	return w.TranscriptWriter.Append(ev)
}

type heldTranscript struct {
	Events     []runtime.Event `json:"events"`
	Pending    *runtime.Event  `json:"pending_permission"`
	Unretained []runtime.Event `json:"unretained_permissions"`
}

func readHeldTranscript(t *testing.T, srv *Server, path string) heldTranscript {
	t.Helper()
	rec := doGET(t, srv.routes(), path)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s = %d %s", path, rec.Code, rec.Body.String())
	}
	var out heldTranscript
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// TestUnsavedPermissionStaysApprovable pins FS-03.R99 / TS-03.R73: a live
// request whose transcript record failed stays approvable through the
// canonical window and the desktop reload, scoped to the live generation, kept
// out of history, and cleared once it resolves.
func TestUnsavedPermissionStaysApprovable(t *testing.T) {
	srv := testServer(t, true)
	srv.registry.Chat().SetCommand(buildServerFakeACP(t))
	srv.registry.SetPersistence(srv.configStore.Home(), func(home, id string, meta *runtime.SessionMetaData) (runtime.TranscriptWriter, error) {
		w, err := transcript.Open(home, id, meta)
		return permissionAppendFailureWriter{w}, err
	}, srv.indexer)
	id := seedHookAgent(t, srv)
	a, err := srv.stateStore.ReadAgent(id)
	if err != nil {
		t.Fatal(err)
	}
	a.Interface, a.Backend, a.Model = "chat", "claude", "sonnet"
	if err := srv.stateStore.WriteAgent(a); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.registry.Launch(context.Background(), runtime.LaunchSpec{Agent: a, Generation: "g_live", Cwd: t.TempDir(),
		BackendType: "claude-acp", ModelID: "claude-sonnet-4-6", Env: []string{"FAKEACP_SCENARIO=permission"}}); err != nil {
		t.Fatal(err)
	}
	defer srv.registry.Shutdown(context.Background())
	events, unsub := srv.eventBus.Subscribe()
	defer unsub()
	if err := srv.registry.Chat().SendPrompt(context.Background(), id, "ask first"); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(5 * time.Second)
wait:
	for {
		select {
		case event := <-events:
			raw, _ := json.Marshal(event.Data)
			if event.Type == "notification" && strings.Contains(string(raw), "permission_required") {
				break wait
			}
		case <-deadline:
			t.Fatal("no fallback permission notification")
		}
	}
	if !srv.registry.Chat().PermissionPending(id, "tc_p") {
		t.Fatal("no live pending request")
	}
	// A held request from an ended generation reuses the tool-call id but is
	// never offered (INV §4).
	stale, _ := json.Marshal(runtime.PermissionRequestData{ToolCallID: "tc_p", Name: "stale"})
	srv.holdUnretainedPermission(runtime.Event{AgentID: id, Generation: "g_old", Seq: 1 << 40, Type: runtime.EvPermissionRequest, Data: stale})

	window := readHeldTranscript(t, srv, "/api/sessions/"+id+"/transcript?limit=750")
	if window.Pending == nil || !strings.Contains(string(window.Pending.Data), `"Run ls"`) {
		t.Fatalf("pending_permission = %+v, want the live unsaved tc_p request", window.Pending)
	}
	for _, ev := range window.Events {
		if ev.Type == runtime.EvPermissionRequest {
			t.Fatalf("unsaved request entered history: %+v", ev)
		}
	}
	full := readHeldTranscript(t, srv, "/api/sessions/"+id+"/transcript")
	if len(full.Unretained) != 1 || !strings.Contains(string(full.Unretained[0].Data), `"Run ls"`) {
		t.Fatalf("unretained_permissions = %+v, want only the live request", full.Unretained)
	}

	resp, body := postJSONHandler(t, srv.routes(), "/api/sessions/"+id+"/permission", map[string]string{"tool_call_id": "tc_p", "decision": "approve"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("approve = %d %s", resp.StatusCode, body)
	}
	if after := readHeldTranscript(t, srv, "/api/sessions/"+id+"/transcript?limit=750"); after.Pending != nil {
		t.Fatalf("pending_permission after approval = %+v, want nil", after.Pending)
	}
	if after := readHeldTranscript(t, srv, "/api/sessions/"+id+"/transcript"); len(after.Unretained) != 0 {
		t.Fatalf("unretained_permissions after approval = %+v, want none", after.Unretained)
	}
	srv.permissionMu.Lock()
	held := len(srv.heldPermissions)
	srv.permissionMu.Unlock()
	if held != 0 {
		t.Fatalf("held permissions after approval = %d, want 0", held)
	}
}
