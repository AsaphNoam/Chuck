package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
	"github.com/AsaphNoam/Chuck/internal/transcript"
)

type windowBody struct {
	Events            []runtime.Event `json:"events"`
	HasMore           *bool           `json:"has_more"`
	PendingPermission *runtime.Event  `json:"pending_permission"`
	LatestAssistant   string          `json:"latest_assistant"`
}

// seedWindowTranscript writes an agent whose transcript is far larger than a
// window: the pending permission and the latest reply sit before it, and
// filler events of fillerBytes each follow.
func seedWindowTranscript(t *testing.T, s *Server, id string, filler, fillerBytes int) {
	t.Helper()
	if err := s.stateStore.WriteAgent(state.Agent{AgentID: id, Role: "implementer", Project: "demo", Backend: "claude", Interface: "chat", CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	w, err := transcript.Open(s.configStore.Home(), id, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	add := func(typ, activity string, data any) {
		raw, _ := json.Marshal(data)
		if err := w.Append(runtime.Event{Type: typ, Data: raw, ActivityID: activity}); err != nil {
			t.Fatal(err)
		}
	}
	add(runtime.EvPermissionRequest, "", runtime.PermissionRequestData{ToolCallID: "p1", Name: "Bash"})
	add(runtime.EvPermissionRequest, "", runtime.PermissionRequestData{ToolCallID: "p2", Name: "Edit"})
	add(runtime.EvPermissionResolved, "", runtime.PermissionResolvedData{ToolCallID: "p1", Decision: "approve"})
	add(runtime.EvAssistantText, "", runtime.AssistantTextData{Delta: "Latest "})
	add(runtime.EvAssistantText, "", runtime.AssistantTextData{Delta: "reply"})
	add(runtime.EvAssistantText, "child-1", runtime.AssistantTextData{Delta: "child text"})
	for i := 0; i < filler; i++ {
		add(runtime.EvToolCall, "", map[string]string{"tool_call_id": fmt.Sprint("c", i), "title": strings.Repeat("x", fillerBytes)})
	}
}

func getWindow(t *testing.T, h http.Handler, r *http.Request) windowBody {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s = %d %s", r.URL, rec.Code, rec.Body)
	}
	var body windowBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

// A windowed read returns a bounded tail with continuation, and still reports
// the pending permission and latest reply that fall before it (FS-20.R13).
func TestTranscriptWindowBoundsAndContinues(t *testing.T) {
	s := testServer(t, false)
	h := s.routes()
	seedWindowTranscript(t, s, "a_long", 900, 10)
	const total = 906

	full := getWindow(t, h, newLocalRequest(http.MethodGet, "/api/sessions/a_long/transcript", nil))
	if len(full.Events) != total || full.HasMore != nil {
		t.Fatalf("unwindowed read = %d events, has_more %v; want the whole transcript in the old shape", len(full.Events), full.HasMore)
	}

	page := getWindow(t, h, newLocalRequest(http.MethodGet, "/api/sessions/a_long/transcript?limit=100", nil))
	if len(page.Events) != 100 || page.Events[0].Seq != total-99 || page.Events[99].Seq != total || page.HasMore == nil || !*page.HasMore {
		t.Fatalf("latest window = %d events from seq %d, has_more %v", len(page.Events), page.Events[0].Seq, page.HasMore)
	}
	if page.PendingPermission == nil || !strings.Contains(string(page.PendingPermission.Data), `"p2"`) {
		t.Fatalf("pending_permission = %+v, want the unresolved p2 outside the window", page.PendingPermission)
	}
	if page.LatestAssistant != "Latest reply" {
		t.Fatalf("latest_assistant = %q", page.LatestAssistant)
	}

	// Walking before_seq back to the start covers every event exactly once.
	seen := map[int64]bool{}
	for _, ev := range page.Events {
		seen[ev.Seq] = true
	}
	for more := true; more; {
		before := page.Events[0].Seq
		page = getWindow(t, h, newLocalRequest(http.MethodGet, fmt.Sprintf("/api/sessions/a_long/transcript?limit=100&before_seq=%d", before), nil))
		if len(page.Events) == 0 || page.Events[len(page.Events)-1].Seq != before-1 {
			t.Fatalf("page before %d does not adjoin it: %d events", before, len(page.Events))
		}
		for _, ev := range page.Events {
			if seen[ev.Seq] {
				t.Fatalf("seq %d returned twice", ev.Seq)
			}
			seen[ev.Seq] = true
		}
		more = *page.HasMore
	}
	if len(seen) != total {
		t.Fatalf("continuation covered %d events, want %d", len(seen), total)
	}

	clamped := getWindow(t, h, newLocalRequest(http.MethodGet, "/api/sessions/a_long/transcript?limit=100000", nil))
	if len(clamped.Events) != transcriptWindowMax {
		t.Fatalf("clamped window = %d events, want %d", len(clamped.Events), transcriptWindowMax)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newLocalRequest(http.MethodGet, "/api/sessions/a_long/transcript?limit=-1", nil))
	if errorCode(t, rec) != runtime.CodeValidation {
		t.Fatalf("negative limit = %d %s, want a validation error", rec.Code, rec.Body)
	}
}

func TestTranscriptWindowBoundsBytes(t *testing.T) {
	s := testServer(t, false)
	seedWindowTranscript(t, s, "a_big", 40, 100_000)
	page := getWindow(t, s.routes(), newLocalRequest(http.MethodGet, "/api/sessions/a_big/transcript?limit=500", nil))
	size := 0
	for _, ev := range page.Events {
		size += eventSize(ev)
	}
	if size > transcriptWindowMaxBytes || len(page.Events) >= 40 || !*page.HasMore {
		t.Fatalf("byte-bounded window = %d events, %d bytes, has_more %v", len(page.Events), size, *page.HasMore)
	}
	if page.PendingPermission == nil || page.LatestAssistant != "Latest reply" {
		t.Fatalf("byte-bounded window lost pending %v or latest %q", page.PendingPermission, page.LatestAssistant)
	}
}

// The tailnet listener windows a transcript read even when the phone names no
// limit, so no phone request can read the whole session (FS-20.R13).
func TestRemoteTranscriptIsAlwaysWindowed(t *testing.T) {
	s := testServer(t, true)
	seedWindowTranscript(t, s, "a_phone", 900, 10)
	h := s.remoteRoutes(testDomain, testWhoIs(map[string]string{"100.64.0.2:5000": "node-phone"}))
	token := pairTestDevice(t, s, "d1", "node-phone")
	page := getWindow(t, h, phoneRequest(http.MethodGet, "/api/sessions/a_phone/transcript", "", token))
	if len(page.Events) != transcriptWindowDefault || page.HasMore == nil || !*page.HasMore {
		t.Fatalf("phone read = %d events, has_more %v; want the default window", len(page.Events), page.HasMore)
	}
}

// TS-04.R94: a host-owned peer Stop request from an ended generation is never
// offered as actionable, while an ordinary request keeps its existing shape.
func TestTranscriptWindowFoldsEndedHostRequests(t *testing.T) {
	request := func(id, origin string) runtime.Event {
		raw, _ := json.Marshal(runtime.PermissionRequestData{ToolCallID: id, Name: "x", Origin: origin})
		return runtime.Event{Type: runtime.EvPermissionRequest, Data: raw}
	}
	live := map[string]bool{"chuck_peer_stop_live": true}
	tw := &transcriptWindow{limit: 10, hostPending: func(id string) bool { return live[id] }}
	for i, ev := range []runtime.Event{
		request("acp-1", ""),
		request("chuck_peer_stop_old", runtime.PermissionOriginPeerStop),
	} {
		ev.Seq = int64(i + 1)
		_ = tw.visit(ev)
	}
	got := tw.response("a")["pending_permission"].(*runtime.Event)
	if got == nil || !strings.Contains(string(got.Data), "acp-1") {
		t.Fatalf("pending = %v, want the ordinary request", got)
	}
	_ = tw.visit(request("chuck_peer_stop_live", runtime.PermissionOriginPeerStop))
	got = tw.response("a")["pending_permission"].(*runtime.Event)
	if got == nil || !strings.Contains(string(got.Data), "chuck_peer_stop_live") {
		t.Fatalf("pending = %v, want the live host request", got)
	}
	live = map[string]bool{}
	if got := tw.response("a")["pending_permission"].(*runtime.Event); got != nil && strings.Contains(string(got.Data), "peer_stop") {
		t.Fatalf("stopped requester still offers a host request: %s", got.Data)
	}
}
