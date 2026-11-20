package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func runEditCmd(t *testing.T, handler http.HandlerFunc, stdin string, args ...string) (string, string, error) {
	t.Helper()
	server := httptest.NewServer(handler)
	defer server.Close()
	old := operatorClientFactory
	operatorClientFactory = func() (*operatorClient, error) {
		return &operatorClient{HTTP: server.Client(), BaseURL: server.URL}, nil
	}
	defer func() { operatorClientFactory = old }()
	cmd := newOperatorAgentCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func TestOperatorEditLastBodyAndGeneratedRequestID(t *testing.T) {
	var got, path string
	out, errOut, err := runEditCmd(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got, path = string(b), r.Method+" "+r.URL.Path
		w.WriteHeader(202)
		_, _ = w.Write([]byte(`{"agent_id":"a","request_id":"x","phase":"preparing"}`))
	}, "line1\nünï \n\n", "edit-last", "a", "--message-seq", "7", "--expected-revision", "0", "--text-file", "-", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if path != "POST /api/sessions/a/edit-last" {
		t.Fatalf("path %s", path)
	}
	m := map[string]any{}
	if err := json.Unmarshal([]byte(got), &m); err != nil {
		t.Fatal(err)
	}
	id, _ := m["request_id"].(string)
	if m["text"] != "line1\nünï \n\n" || m["message_seq"] != float64(7) || m["expected_revision"] != float64(0) || !operatorValidReceiptID(id) || len(m) != 4 {
		t.Fatalf("body %s", got)
	}
	if !strings.Contains(errOut, id) || !strings.Contains(out, `"phase":"preparing"`) {
		t.Fatalf("out %q err %q", out, errOut)
	}
}

func TestOperatorEditLastRequestFilePassthroughAndRetryID(t *testing.T) {
	var got string
	body := `{"request_id":"r-1","message_seq":3,"expected_revision":2,"text":"a\n"}`
	_, errOut, err := runEditCmd(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		w.WriteHeader(202)
		_, _ = w.Write([]byte(`{}`))
	}, body, "edit-last", "a", "--request-file", "-", "--json")
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"expected_revision":2,"message_seq":3,"request_id":"r-1","text":"a\n"}` || errOut != "" {
		t.Fatalf("body %s err %q", got, errOut)
	}
}

func TestOperatorEditLastRejectsBeforeRequest(t *testing.T) {
	for _, args := range [][]string{
		{"edit-last", "a", "--expected-revision", "0", "--text", "x"},
		{"edit-last", "a", "--message-seq", "1", "--text", "x"},
		{"edit-last", "a", "--message-seq", "1", "--expected-revision", "0"},
		{"edit-last", "a", "--message-seq", "0", "--expected-revision", "0", "--text", "x"},
		{"edit-last", "a", "--message-seq", "1", "--expected-revision", "0", "--text", "x", "--request-id", "bad id"},
		{"edit-last", "a", "--message-seq", "1", "--expected-revision", "0", "--text", "x", "--text-file", "-"},
	} {
		_, _, err := runEditCmd(t, func(w http.ResponseWriter, r *http.Request) { t.Errorf("request sent for %v", args) }, "", args...)
		var op *operatorError
		if !errors.As(err, &op) || op.ExitCode != 2 {
			t.Errorf("%v: %v", args, err)
		}
	}
}

func TestOperatorEditLastErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
	}{{409, "edit_conflict"}, {422, "edit_unavailable"}, {409, "request_conflict"}} {
		_, _, err := runEditCmd(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = fmt.Fprintf(w, `{"code":%q,"message":"nope","details":{"reason":"busy"}}`, tc.code)
		}, "", "edit-last", "a", "--message-seq", "1", "--expected-revision", "0", "--text", "x", "--request-id", "r")
		var op *operatorError
		if !errors.As(err, &op) || op.Code != tc.code || op.HTTPStatus != tc.status || op.ExitCode != 1 {
			t.Errorf("%s: %#v", tc.code, err)
		}
	}
}

func TestOperatorEditLastStatusQuery(t *testing.T) {
	var path string
	out, _, err := runEditCmd(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.Method + " " + r.URL.Path
		_, _ = w.Write([]byte(`{"agent_id":"a","request_id":"r-1","phase":"committed","revision":1,"delivery":"sent","active":"revised"}`))
	}, "", "edit-last-status", "a", "r-1", "--json")
	if err != nil || path != "GET /api/sessions/a/edit-last/r-1" || !strings.Contains(out, `"phase":"committed"`) {
		t.Fatalf("%v %s %s", err, path, out)
	}
}

func TestOperatorFollowRefetchesActiveViewOnConversationRevision(t *testing.T) {
	marker := map[string]any{"agent_id": "__hydrated__", "data": map[string]bool{"hydrated": true}}
	var fullReads int
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/events":
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(200)
			_, _ = io.WriteString(w, operatorSSEEvent("state_update", marker))
		case "/api/sessions/a/transcript":
			if r.URL.Query().Has("since_seq") {
				_, _ = io.WriteString(w, `{"agent_id":"a","events":[{"agent_id":"a","seq":1,"type":"user_message","data":{"text":"old"}},{"agent_id":"a","seq":2,"type":"conversation_revision","data":{"request_id":"r","revision":1,"replaces_from_seq":1,"replaces_through_seq":1}}]}`)
				return
			}
			fullReads++
			_, _ = io.WriteString(w, `{"agent_id":"a","history_revision":1,"events":[{"agent_id":"a","seq":2,"type":"conversation_revision","data":{"request_id":"r","revision":1,"replaces_from_seq":1,"replaces_through_seq":1}},{"agent_id":"a","seq":3,"type":"user_message","data":{"text":"new"}}]}`)
		default:
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewServer(h)
	defer srv.Close()
	client := &operatorClient{HTTP: srv.Client(), BaseURL: srv.URL}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	out := &countOutput{cancel: cancel, target: 3}
	cmd := operatorFollowCmd(t, ctx, out)
	if err := operatorFollow(cmd, client, "a", 0); err != nil {
		t.Fatal(err)
	}
	s := out.buf.String()
	if fullReads != 1 || !strings.Contains(s, `"text":"new"`) || !strings.Contains(s, `"seq":3`) {
		t.Fatalf("reads=%d output=%q", fullReads, s)
	}
}
