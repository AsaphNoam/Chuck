package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type operatorStatusFixture struct {
	URL          string
	server       *http.Server
	mu           sync.Mutex
	paths        []string
	streamClosed chan struct{}
	mutations    int
}

func newOperatorStatusFixture(t *testing.T, handler http.HandlerFunc) *operatorStatusFixture {
	t.Helper()
	fixture := &operatorStatusFixture{streamClosed: make(chan struct{})}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/events" {
			defer close(fixture.streamClosed)
		}
		fixture.mu.Lock()
		fixture.paths = append(fixture.paths, r.Method+" "+r.URL.RequestURI())
		if r.Method != http.MethodGet {
			fixture.mutations++
		}
		fixture.mu.Unlock()
		handler(w, r)
	})}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("loopback unavailable: %v", err)
	}
	fixture.URL = "http://" + listener.Addr().String()
	fixture.server = server
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return fixture
}

func (f *operatorStatusFixture) calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.paths...)
}

func operatorStatusCommand(t *testing.T, fixture *operatorStatusFixture, id string) ([]byte, error) {
	t.Helper()
	old := operatorClientFactory
	operatorClientFactory = func() (*operatorClient, error) {
		return &operatorClient{BaseURL: fixture.URL, HTTP: &http.Client{Timeout: 3 * time.Second}}, nil
	}
	t.Cleanup(func() { operatorClientFactory = old })
	var output bytes.Buffer
	cmd := newOperatorAgentStatusCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&output)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{id, "--json"})
	err := cmd.Execute()
	return output.Bytes(), err
}

func operatorStatusWriteJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func operatorStatusStream(w http.ResponseWriter, r *http.Request, frames ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	for _, frame := range frames {
		_, _ = io.WriteString(w, frame)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}
	<-r.Context().Done()
}

func TestOperatorAgentStatusComposesLiveSnapshotHeldAndPendingPermissions(t *testing.T) {
	transcript := `{"agent_id":"agent/a","events":[` +
		`{"agent_id":"agent/a","seq":1,"type":"permission_request","data":{"tool_call_id":"root:1","name":"Bash","n":9007199254740993}},` +
		`{"agent_id":"agent/a","seq":2,"type":"permission_request","activity_id":"child-7","parent_activity_id":"root","data":{"tool_call_id":"child:2","name":"Edit"}},` +
		`{"agent_id":"agent/a","seq":3,"type":"permission_resolved","activity_id":"child-7","data":{"tool_call_id":"root:1","decision":"denied"}}],"has_more":false}`
	fixture := newOperatorStatusFixture(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/api/sessions/agent%2Fa":
			operatorStatusWriteJSON(w, 200, `{"agent":{"agent_id":"agent/a"},"status":"running","quota":{"remaining":0},"unknown":{"v":9007199254740993}}`)
		case "/api/events":
			operatorStatusStream(w, r,
				operatorSSEEvent("state_update", map[string]any{"agent_id": "other", "data": map[string]any{"context": "wrong"}}),
				operatorSSEEvent("state_update", map[string]any{"agent_id": "agent/a", "data": json.RawMessage(`{"context":{"used":0},"runtime_capabilities":{"tools":["Bash"]},"unknown":9007199254740993}`)}),
				operatorSSEEvent("state_update", map[string]any{"agent_id": "__hydrated__", "data": map[string]any{"hydrated": true}}),
			)
		case "/api/sessions/agent%2Fa/prompt":
			operatorStatusWriteJSON(w, 200, `{"held":{"text":"recover me\n雪"},"recovered":true}`)
		case "/api/sessions/agent%2Fa/transcript":
			if r.URL.Query().Get("include_meta") != "true" || r.URL.Query().Get("limit") != "500" {
				t.Errorf("transcript query = %s", r.URL.RawQuery)
			}
			operatorStatusWriteJSON(w, 200, transcript)
		default:
			http.NotFound(w, r)
		}
	})
	out, err := operatorStatusCommand(t, fixture, "agent/a")
	if err != nil {
		t.Fatal(err)
	}
	var result operatorAgentStatusResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("output %q: %v", out, err)
	}
	if !bytes.Contains(result.Session, []byte(`9007199254740993`)) || !bytes.Contains(result.Snapshot, []byte(`9007199254740993`)) {
		t.Fatalf("large unknown values were not preserved: session=%s snapshot=%s", result.Session, result.Snapshot)
	}
	if !bytes.Contains(result.Snapshot, []byte(`"runtime_capabilities"`)) || bytes.Contains(result.Snapshot, []byte(`"wrong"`)) {
		t.Fatalf("snapshot did not retain the exact agent capability payload: %s", result.Snapshot)
	}
	if !result.HeldAvailable || !bytes.Contains(result.Held, []byte("雪")) {
		t.Fatalf("held response = %s available=%v", result.Held, result.HeldAvailable)
	}
	if !result.Permissions.Complete || len(result.Permissions.Pending) != 1 || !bytes.Contains(result.Permissions.Pending[0], []byte(`"child:2"`)) || !bytes.Contains(result.Permissions.Pending[0], []byte(`"activity_id":"child-7"`)) {
		t.Fatalf("permissions = %+v", result.Permissions)
	}
	calls := fixture.calls()
	if len(calls) != 4 || !strings.HasPrefix(calls[0], "GET /api/sessions/agent%2Fa") || calls[1] != "GET /api/events" || calls[2] != "GET /api/sessions/agent%2Fa/prompt" || !strings.HasPrefix(calls[3], "GET /api/sessions/agent%2Fa/transcript?") {
		t.Fatalf("request sequence = %#v", calls)
	}
	if fixture.mutations != 0 {
		t.Fatalf("status made %d non-GET requests", fixture.mutations)
	}
	select {
	case <-fixture.streamClosed:
	case <-time.After(time.Second):
		t.Fatal("status left its SSE stream open")
	}
}

func TestOperatorAgentStatusUnknownCompletenessAndStoppedPrompt(t *testing.T) {
	fixture := newOperatorStatusFixture(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/sessions/a":
			operatorStatusWriteJSON(w, 200, `{"agent":{"agent_id":"a"},"status":"stopped"}`)
		case "/api/events":
			operatorStatusStream(w, r,
				operatorSSEEvent("state_update", map[string]any{"agent_id": "__hydrated__", "data": map[string]any{"hydrated": true}}),
			)
		case "/api/sessions/a/prompt":
			operatorStatusWriteJSON(w, http.StatusNotFound, `{"error":"no held prompt"}`)
		case "/api/sessions/a/transcript":
			operatorStatusWriteJSON(w, 200, `{"agent_id":"a","events":[],"has_more":true}`)
		default:
			http.NotFound(w, r)
		}
	})
	out, err := operatorStatusCommand(t, fixture, "a")
	if err != nil {
		t.Fatal(err)
	}
	var result operatorAgentStatusResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatal(err)
	}
	if result.HeldAvailable || string(result.Held) != "null" || result.Snapshot == nil || string(result.Snapshot) != "null" {
		t.Fatalf("held/snapshot = %s/%s available=%v", result.Held, result.Snapshot, result.HeldAvailable)
	}
	if result.Permissions.Complete || result.Permissions.Pending == nil || len(result.Permissions.Pending) != 0 {
		t.Fatalf("partial-window permissions = %+v", result.Permissions)
	}
}

func TestOperatorAgentStatusRejectsMissingTargetBeforeStream(t *testing.T) {
	fixture := newOperatorStatusFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions/missing" {
			operatorStatusWriteJSON(w, http.StatusNotFound, `{"error":"not found"}`)
			return
		}
		t.Errorf("unexpected request after missing target: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})
	out, err := operatorStatusCommand(t, fixture, "missing")
	if err == nil || len(out) != 0 {
		t.Fatalf("missing target output=%q err=%v", out, err)
	}
	if calls := fixture.calls(); len(calls) != 1 || calls[0] != "GET /api/sessions/missing" {
		t.Fatalf("requests after missing target = %#v", calls)
	}
}

func TestOperatorAgentStatusRejectsWrongSessionAndInvalidInputWithoutRequests(t *testing.T) {
	fixture := newOperatorStatusFixture(t, func(w http.ResponseWriter, r *http.Request) {
		operatorStatusWriteJSON(w, 200, `{"agent":{"agent_id":"different"}}`)
	})
	out, err := operatorStatusCommand(t, fixture, "a")
	if err == nil || len(out) != 0 {
		t.Fatalf("wrong target output=%q err=%v", out, err)
	}
	if len(fixture.calls()) != 1 {
		t.Fatalf("wrong-target response continued: %#v", fixture.calls())
	}

	old := operatorClientFactory
	called := false
	operatorClientFactory = func() (*operatorClient, error) { called = true; return nil, nil }
	t.Cleanup(func() { operatorClientFactory = old })
	cmd := newOperatorAgentStatusCmd()
	cmd.SetArgs([]string{"  ", "--json"})
	if err := cmd.Execute(); err == nil || called {
		t.Fatalf("empty id err=%v client-created=%v", err, called)
	}
}

func TestOperatorAgentStatusPromptNon404FailureDoesNotReturnSuccess(t *testing.T) {
	fixture := newOperatorStatusFixture(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/sessions/a":
			operatorStatusWriteJSON(w, 200, `{"agent":{"agent_id":"a"}}`)
		case "/api/events":
			operatorStatusStream(w, r, operatorSSEEvent("state_update", map[string]any{"agent_id": "__hydrated__", "data": map[string]any{"hydrated": true}}))
		case "/api/sessions/a/prompt":
			operatorStatusWriteJSON(w, http.StatusInternalServerError, `{"error":"broken"}`)
		default:
			http.NotFound(w, r)
		}
	})
	out, err := operatorStatusCommand(t, fixture, "a")
	if err == nil || len(out) != 0 {
		t.Fatalf("prompt failure output=%q err=%v", out, err)
	}
}

func TestOperatorAgentStatusOnlyAcceptsExactArguments(t *testing.T) {
	for _, args := range [][]string{{}, {"a", "b"}} {
		cmd := newOperatorAgentStatusCmd()
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil {
			t.Fatalf("arguments %v were accepted", args)
		}
	}
}

func TestOperatorStatusPermissionResolutionUsesExactToolCallID(t *testing.T) {
	page := json.RawMessage(`{"agent_id":"a","events":[` +
		`{"type":"permission_request","data":{"tool_call_id":"id"}},` +
		`{"type":"permission_request","data":{"tool_call_id":"id:child"}},` +
		`{"type":"permission_resolved","data":{"tool_call_id":"id"}}],"has_more":false}`)
	fixture := newOperatorStatusFixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/sessions/a/transcript" || r.URL.Query().Get("limit") != "500" {
			t.Errorf("unexpected transcript request %s %s", r.Method, r.URL.RequestURI())
		}
		operatorStatusWriteJSON(w, 200, string(page))
	})
	client := &operatorClient{BaseURL: fixture.URL, HTTP: &http.Client{}}
	permissions, err := operatorStatusPermissions(context.Background(), client, "a")
	if err != nil {
		t.Fatal(err)
	}
	if len(permissions.Pending) != 1 || !strings.Contains(string(permissions.Pending[0]), `"id:child"`) {
		t.Fatalf("prefix-colliding tool IDs were not kept distinct: %+v", permissions)
	}
}

// TS-03.R71 — a peer-Stop request whose wait ended (e.g. after a restart) is
// not reported as pending; only the server's live verdict keeps one.
func TestOperatorStatusTrustsServerLivenessForPeerStopRequests(t *testing.T) {
	for _, tc := range []struct {
		name, canonical string
		want            int
		complete        bool
	}{
		{"dead", `null`, 0, false},
		{"live", `{"agent_id":"a","type":"permission_request","data":{"tool_call_id":"stop:1","origin":"chuck_peer_stop"}}`, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := `{"agent_id":"a","events":[{"type":"permission_request","data":{"tool_call_id":"stop:1","origin":"chuck_peer_stop"}}],"has_more":false,"pending_permission":` + tc.canonical + `}`
			fixture := newOperatorStatusFixture(t, func(w http.ResponseWriter, r *http.Request) {
				operatorStatusWriteJSON(w, http.StatusOK, page)
			})
			client := &operatorClient{BaseURL: fixture.URL, HTTP: &http.Client{}}
			permissions, err := operatorStatusPermissions(context.Background(), client, "a")
			if err != nil {
				t.Fatal(err)
			}
			if len(permissions.Pending) != tc.want || permissions.Complete != tc.complete {
				t.Fatalf("permissions = %+v", permissions)
			}
		})
	}
}

func TestOperatorStatusIncludesCanonicalPendingPermissionOutsidePartialWindow(t *testing.T) {
	page := `{"agent_id":"a","events":[{"agent_id":"a","seq":90,"type":"assistant_text","data":{"delta":"working"}}],"has_more":true,"pending_permission":{"agent_id":"a","seq":12,"type":"permission_request","activity_id":"native-child","parent_activity_id":"root","data":{"tool_call_id":"native:12","name":"Bash","args":{"count":9007199254740993}}}}`
	fixture := newOperatorStatusFixture(t, func(w http.ResponseWriter, r *http.Request) {
		operatorStatusWriteJSON(w, http.StatusOK, page)
	})
	client := &operatorClient{BaseURL: fixture.URL, HTTP: &http.Client{}}
	permissions, err := operatorStatusPermissions(context.Background(), client, "a")
	if err != nil {
		t.Fatal(err)
	}
	if permissions.Complete || len(permissions.Pending) != 1 {
		t.Fatalf("permissions = %+v", permissions)
	}
	got := string(permissions.Pending[0])
	for _, want := range []string{`"native:12"`, `"activity_id":"native-child"`, `"parent_activity_id":"root"`, `9007199254740993`} {
		if !strings.Contains(got, want) {
			t.Fatalf("canonical raw permission missing %q: %s", want, got)
		}
	}
}

func TestOperatorStatusRejectsMalformedCanonicalPendingPermission(t *testing.T) {
	fixture := newOperatorStatusFixture(t, func(w http.ResponseWriter, r *http.Request) {
		operatorStatusWriteJSON(w, http.StatusOK, `{"agent_id":"a","events":[],"has_more":true,"pending_permission":{"agent_id":"other","type":"permission_request","data":{"tool_call_id":"x"}}}`)
	})
	client := &operatorClient{BaseURL: fixture.URL, HTTP: &http.Client{}}
	if _, err := operatorStatusPermissions(context.Background(), client, "a"); err == nil {
		t.Fatal("malformed canonical permission was accepted")
	}
}
