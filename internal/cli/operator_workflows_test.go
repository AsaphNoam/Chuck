package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

type workflowWireCase struct {
	name, method, requestURI string
	args                     []string
	stdin                    string
	body                     string
}

func TestOperatorWorkflowCommandsUseExistingWireContracts(t *testing.T) {
	cases := []workflowWireCase{
		{name: "task list", method: http.MethodGet, requestURI: "/api/tasks?project=alpha+one", args: []string{"task", "list", "--project", "alpha one", "--json"}},
		{name: "task show", method: http.MethodGet, requestURI: "/api/tasks/task%201", args: []string{"task", "show", "task 1", "--json"}},
		{name: "task create", method: http.MethodPost, requestURI: "/api/tasks", args: []string{"task", "create", "--project", "p1", "--display-name", "Cache review", "--target-kind", "agent", "--target-agent", "a1", "--text", "Inspect cache\n雪", "--fast", "--json"}, body: `{"display_name":"Cache review","fast":true,"instruction":"Inspect cache\n雪","project":"p1","target_agent_id":"a1","target_kind":"agent"}`},
		{name: "task cancel", method: http.MethodPost, requestURI: "/api/tasks/t1/cancel", args: []string{"task", "cancel", "t1", "--json"}, body: `{}`},
		{name: "task result", method: http.MethodPost, requestURI: "/api/tasks/t1/result", args: []string{"task", "result", "t1", "--outcome", "failure", "--text", "Needs a fix", "--details", "Reproduced", "--json"}, body: `{"details":"Reproduced","outcome":"failure","summary":"Needs a fix"}`},
		{name: "task retry", method: http.MethodPost, requestURI: "/api/tasks/t1/retry", args: []string{"task", "retry", "t1", "--json"}, body: `{}`},
		{name: "task rearm", method: http.MethodPost, requestURI: "/api/tasks/t1/rearm", args: []string{"task", "rearm", "t1", "--json"}, body: `{}`},
		{name: "task delete", method: http.MethodDelete, requestURI: "/api/tasks/t1", args: []string{"task", "delete", "t1", "--yes", "--json"}},
		{name: "signal fire", method: http.MethodPost, requestURI: "/api/signals", args: []string{"signal", "fire", "--project", "p1", "--name", "build ready", "--json"}, body: `{"name":"build ready","project":"p1"}`},
		{name: "think-tank list", method: http.MethodGet, requestURI: "/api/think-tanks", args: []string{"think-tank", "list", "--json"}},
		{name: "think-tank show", method: http.MethodGet, requestURI: "/api/think-tanks/room%201", args: []string{"think-tank", "show", "room 1", "--json"}},
		{name: "think-tank create", method: http.MethodPost, requestURI: "/api/think-tanks", args: []string{"think-tank", "create", "--command-id", "create-1", "--title", "Review", "--goal", "Choose cache", "--project", "p1", "--openings", "--request-file", "-", "--json"}, stdin: `{"participants":[{"agent_id":"a1","limit":2}]}`, body: `{"command_id":"create-1","goal":"Choose cache","openings":true,"origin_project":"p1","participants":[{"agent_id":"a1","limit":2}],"title":"Review"}`},
		{name: "think-tank entries", method: http.MethodGet, requestURI: "/api/think-tanks/r1/entries?after=3", args: []string{"think-tank", "entries", "r1", "--after", "3", "--json"}},
		{name: "think-tank activity", method: http.MethodGet, requestURI: "/api/think-tanks/r1/activity?after=4", args: []string{"think-tank", "activity", "r1", "--after", "4", "--json"}},
		{name: "think-tank files", method: http.MethodGet, requestURI: "/api/think-tanks/r1/files", args: []string{"think-tank", "files", "r1", "--json"}},
		{name: "think-tank commands", method: http.MethodGet, requestURI: "/api/think-tanks/r1/commands", args: []string{"think-tank", "commands", "r1", "--json"}},
		{name: "think-tank source-file", method: http.MethodGet, requestURI: "/api/think-tanks/r1/sources/s%201/file", args: []string{"think-tank", "source-file", "r1", "s 1", "--json"}},
		{name: "think-tank send", method: http.MethodPost, requestURI: "/api/think-tanks/r1/messages", args: []string{"think-tank", "send", "r1", "--command-id", "message-1", "--text-file", "-", "--mention", "a1:0:7", "--mention", "a2:8:11", "--json"}, stdin: "Compare hit rate\n雪\n", body: `{"body":"Compare hit rate\n雪\n","command_id":"message-1","mentions":[{"agent_id":"a1","start":0,"end":7},{"agent_id":"a2","start":8,"end":11}]}`},
		{name: "think-tank annotate", method: http.MethodPost, requestURI: "/api/think-tanks/r1/annotations", args: []string{"think-tank", "annotate", "r1", "--command-id", "annotation-1", "--request-file", "-", "--json"}, stdin: `{"target":{"kind":"room"},"annotations":[{"anchor":"entry","seq":1,"excerpt":"cache","instruction":"Compare"}]}`, body: `{"annotations":[{"anchor":"entry","seq":1,"excerpt":"cache","instruction":"Compare"}],"command_id":"annotation-1","target":{"kind":"room"}}`},
		{name: "think-tank pause", method: http.MethodPost, requestURI: "/api/think-tanks/r1/pause", args: []string{"think-tank", "pause", "r1", "--json"}, body: `{}`},
		{name: "think-tank resume", method: http.MethodPost, requestURI: "/api/think-tanks/r1/resume", args: []string{"think-tank", "resume", "r1", "--json"}, body: `{}`},
		{name: "think-tank end", method: http.MethodPost, requestURI: "/api/think-tanks/r1/end", args: []string{"think-tank", "end", "r1", "--json"}, body: `{}`},
		{name: "think-tank retry", method: http.MethodPost, requestURI: "/api/think-tanks/r1/retry", args: []string{"think-tank", "retry", "r1", "--target", "turn", "--attempt-id", "attempt-1", "--command-id", "retry-1", "--json"}, body: `{"attempt_id":"attempt-1","command_id":"retry-1","target":"turn"}`},
		{name: "think-tank participant-turn-limit", method: http.MethodPost, requestURI: "/api/think-tanks/r1/participants/a1/turn-limit", args: []string{"think-tank", "participant-turn-limit", "r1", "a1", "--command-id", "limit-1", "--expected-limit", "2", "--limit", "5", "--json"}, body: `{"command_id":"limit-1","expected_limit":2,"limit":5}`},
		{name: "think-tank delete", method: http.MethodDelete, requestURI: "/api/think-tanks/r1", args: []string{"think-tank", "delete", "r1", "--yes", "--json"}},
	}
	if len(cases) != 25 {
		t.Fatalf("workflow case inventory has %d entries, want 25", len(cases))
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotURI, gotBody, gotContentType string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotURI, gotContentType = r.Method, r.RequestURI, r.Header.Get("Content-Type")
				data, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read body: %v", err)
				}
				gotBody = string(data)
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"ok":true,"state":"accepted"}`)
			}))
			defer server.Close()
			operatorClientFactory = func() (*operatorClient, error) {
				return &operatorClient{HTTP: server.Client(), BaseURL: server.URL}, nil
			}
			t.Cleanup(func() { operatorClientFactory = newOperatorClient })

			root := newWorkflowTestRoot()
			root.SetArgs(tc.args)
			root.SetIn(strings.NewReader(tc.stdin))
			var stdout, stderr bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			if _, err := root.ExecuteC(); err != nil {
				t.Fatalf("execute: %v (stderr %q)", err, stderr.String())
			}
			if gotMethod != tc.method || gotURI != tc.requestURI {
				t.Fatalf("request = %s %s, want %s %s", gotMethod, gotURI, tc.method, tc.requestURI)
			}
			if tc.body != "" {
				if gotContentType != "application/json" {
					t.Fatalf("Content-Type = %q, want application/json", gotContentType)
				}
				if !sameJSON(t, gotBody, tc.body) {
					t.Fatalf("body = %s, want %s", gotBody, tc.body)
				}
			} else if gotBody != "" || gotContentType != "" {
				t.Fatalf("unexpected body/header: body=%q Content-Type=%q", gotBody, gotContentType)
			}
			if stdout.String() != `{"ok":true,"state":"accepted"}`+"\n" {
				t.Fatalf("stdout = %q", stdout.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q", stderr.String())
			}
		})
	}
}

func TestOperatorWorkflowGeneratedRoomCommandIDIsReusableShape(t *testing.T) {
	var got map[string]json.RawMessage
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.RequestURI != "/api/think-tanks/r1/annotations" {
			t.Errorf("request = %s %s", r.Method, r.RequestURI)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode body: %v", err)
		}
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()
	operatorClientFactory = func() (*operatorClient, error) {
		return &operatorClient{HTTP: server.Client(), BaseURL: server.URL}, nil
	}
	t.Cleanup(func() { operatorClientFactory = newOperatorClient })
	root := newWorkflowTestRoot()
	root.SetArgs([]string{"think-tank", "annotate", "r1", "--request-file", "-", "--json"})
	root.SetIn(strings.NewReader(`{"target":{"kind":"room"},"annotations":[]}`))
	if _, err := root.ExecuteC(); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := json.Unmarshal(got["command_id"], &id); err != nil {
		t.Fatalf("generated command_id: %v", err)
	}
	if !regexp.MustCompile(`^request-[0-9a-f]{16}$`).MatchString(id) {
		t.Fatalf("generated command_id %q has invalid retryable shape", id)
	}
}

func TestOperatorWorkflowRequestFilePreservesServerFieldsAndZeroValues(t *testing.T) {
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.RequestURI != "/api/think-tanks/r1/participants/a1/turn-limit" {
			t.Errorf("request = %s %s", r.Method, r.RequestURI)
		}
		body, _ = io.ReadAll(r.Body)
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()
	operatorClientFactory = func() (*operatorClient, error) {
		return &operatorClient{HTTP: server.Client(), BaseURL: server.URL}, nil
	}
	t.Cleanup(func() { operatorClientFactory = newOperatorClient })
	root := newWorkflowTestRoot()
	root.SetArgs([]string{"think-tank", "participant-turn-limit", "r1", "a1", "--request-file", "-", "--json"})
	root.SetIn(strings.NewReader(`{"command_id":"limit-zero","expected_limit":0,"limit":5,"future_counter":9223372036854775807,"enabled":false}`))
	if _, err := root.ExecuteC(); err != nil {
		t.Fatal(err)
	}
	want := `{"command_id":"limit-zero","expected_limit":0,"limit":5,"future_counter":9223372036854775807,"enabled":false}`
	if !sameJSON(t, string(body), want) {
		t.Fatalf("body = %s, want %s", body, want)
	}
}

func TestOperatorWorkflowEveryOperationDocumentsItsContract(t *testing.T) {
	for _, family := range newOperatorWorkflowCommands() {
		for _, operation := range family.Commands() {
			if operation.Short == "" || operation.Long == "" || operation.Example == "" {
				t.Errorf("%s %s is missing Short, Long or Example help", family.Name(), operation.Use)
			}
		}
	}
}

func TestOperatorWorkflowInputAndConsentFailBeforeRequest(t *testing.T) {
	factoryCalls := 0
	operatorClientFactory = func() (*operatorClient, error) {
		factoryCalls++
		return nil, fmt.Errorf("unexpected dashboard request")
	}
	t.Cleanup(func() { operatorClientFactory = newOperatorClient })
	cases := []struct {
		name  string
		args  []string
		stdin string
	}{
		{name: "missing task instruction", args: []string{"task", "create", "--project", "p1", "--display-name", "n", "--target-kind", "agent"}},
		{name: "conflicting task instruction sources", args: []string{"task", "create", "--project", "p1", "--display-name", "n", "--target-kind", "agent", "--instruction", "one", "--text", "two"}},
		{name: "missing result text", args: []string{"task", "result", "t1", "--outcome", "failure"}},
		{name: "missing signal name", args: []string{"signal", "fire", "--project", "p1"}},
		{name: "missing task list project", args: []string{"task", "list"}},
		{name: "missing room retry target", args: []string{"think-tank", "retry", "r1"}},
		{name: "missing room message body", args: []string{"think-tank", "send", "r1"}},
		{name: "conflicting room body sources", args: []string{"think-tank", "send", "r1", "--body", "one", "--text", "two"}},
		{name: "malformed mention range", args: []string{"think-tank", "send", "r1", "--text", "message", "--mention", "a1"}},
		{name: "missing turn limit fields", args: []string{"think-tank", "participant-turn-limit", "r1", "a1", "--limit", "0"}},
		{name: "delete requires consent", args: []string{"think-tank", "delete", "r1"}},
		{name: "exact task argument count", args: []string{"task", "show", "t1", "extra"}},
		{name: "exact source argument count", args: []string{"think-tank", "source-file", "r1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			factoryCalls = 0
			root := newWorkflowTestRoot()
			root.SetArgs(tc.args)
			root.SetIn(strings.NewReader(tc.stdin))
			var stdout, stderr bytes.Buffer
			root.SilenceErrors = true
			root.SilenceUsage = true
			root.SetOut(&stdout)
			root.SetErr(&stderr)
			_, err := root.ExecuteC()
			if err == nil {
				t.Fatal("expected local validation error")
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			var op *operatorError
			if !errors.As(err, &op) {
				err = operatorInvalid(err.Error())
				_ = errors.As(err, &op)
			}
			if op.ExitCode != 2 {
				t.Fatalf("error = %#v, want local input error", err)
			}
			if factoryCalls != 0 {
				t.Fatalf("operator client factory called %d times, want no request", factoryCalls)
			}
		})
	}
}

func TestOperatorWorkflowDuplicateRequestFieldAndHTTPError(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"error":{"code":"conflict","message":"stale attempt"}}`)
	}))
	defer server.Close()
	operatorClientFactory = func() (*operatorClient, error) {
		return &operatorClient{HTTP: server.Client(), BaseURL: server.URL}, nil
	}
	t.Cleanup(func() { operatorClientFactory = newOperatorClient })

	root := newWorkflowTestRoot()
	root.SetArgs([]string{"think-tank", "retry", "r1", "--target", "turn", "--request-file", "-", "--json"})
	root.SetIn(strings.NewReader(`{"target":"setup"}`))
	var stdout, stderr bytes.Buffer
	root.SilenceErrors, root.SilenceUsage = true, true
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	_, err := root.ExecuteC()
	if err == nil || requests != 0 {
		t.Fatalf("duplicate request field error=%v requests=%d, want local failure and no request", err, requests)
	}

	root = newWorkflowTestRoot()
	root.SetArgs([]string{"think-tank", "retry", "r1", "--target", "turn", "--json"})
	root.SilenceErrors, root.SilenceUsage = true, true
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	executed, err := root.ExecuteC()
	if err == nil || requests != 1 {
		t.Fatalf("HTTP failure error=%v requests=%d, want one failed request", err, requests)
	}
	operatorDiagnostic(executed, err)
	if stdout.Len() != 0 {
		t.Fatalf("HTTP failure stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), `"code":"conflict"`) || !strings.Contains(stderr.String(), `"http_status":409`) {
		t.Fatalf("HTTP diagnostic = %q", stderr.String())
	}
}

func newWorkflowTestRoot() *cobra.Command {
	root := &cobra.Command{Use: "chuck", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(newOperatorWorkflowCommands()...)
	return root
}

func sameJSON(t *testing.T, got, want string) bool {
	t.Helper()
	var gotValue, wantValue any
	gotDecoder := json.NewDecoder(strings.NewReader(got))
	gotDecoder.UseNumber()
	wantDecoder := json.NewDecoder(strings.NewReader(want))
	wantDecoder.UseNumber()
	if err := gotDecoder.Decode(&gotValue); err != nil {
		t.Fatalf("decode actual JSON %q: %v", got, err)
	}
	if err := wantDecoder.Decode(&wantValue); err != nil {
		t.Fatalf("decode expected JSON %q: %v", want, err)
	}
	return fmt.Sprint(gotValue) == fmt.Sprint(wantValue) || bytes.Equal(mustJSON(t, gotValue), mustJSON(t, wantValue))
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
