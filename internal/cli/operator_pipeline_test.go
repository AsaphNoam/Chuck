package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// This table is supplied independently of the command constructors: each row
// executes a command and checks the actual method, escaped route, query, and
// decoded request body received by a loopback HTTP peer.
func TestOperatorPipelineWireContracts(t *testing.T) {
	type wire struct{ method, path, body string }
	cases := []struct {
		name  string
		args  []string
		input string
		want  wire
	}{
		{"template list", []string{"template", "list", "--json"}, "", wire{http.MethodGet, "/api/pipelines", ""}},
		{"template validate", []string{"template", "validate", "--request-file", "-", "--json"}, `{"id":"quality","template":{"version":2}}`, wire{http.MethodPost, "/api/pipelines/validate", `{"id":"quality","template":{"version":2}}`}},
		{"template show escaped id", []string{"template", "show", "quality draft", "--json"}, "", wire{http.MethodGet, "/api/pipelines/quality%20draft", ""}},
		{"template create", []string{"template", "create", "--request-file", "-", "--json"}, `{"id":"quality","template":{"version":2,"inputs":[],"stages":[]}}`, wire{http.MethodPost, "/api/pipelines", `{"id":"quality","template":{"version":2,"inputs":[],"stages":[]}}`}},
		{"template update", []string{"template", "update", "quality", "--request-file", "-", "--json"}, `{"version":2,"inputs":[],"stages":[]}`, wire{http.MethodPut, "/api/pipelines/quality", `{"version":2,"inputs":[],"stages":[]}`}},
		{"template delete", []string{"template", "delete", "quality", "--yes", "--json"}, "", wire{http.MethodDelete, "/api/pipelines/quality", ""}},
		{"proposal list", []string{"proposal", "list", "--json"}, "", wire{http.MethodGet, "/api/pipeline-proposals", ""}},
		{"proposal decline", []string{"proposal", "decline", "proposal id", "--json"}, "", wire{http.MethodPost, "/api/pipeline-proposals/proposal%20id/decline", ""}},
		{"proposal delete", []string{"proposal", "delete", "proposal id", "--yes", "--json"}, "", wire{http.MethodDelete, "/api/pipeline-proposals/proposal%20id", ""}},
		{"run list page", []string{"run", "list", "--limit", "17", "--offset", "34", "--json"}, "", wire{http.MethodGet, "/api/pipeline-runs?limit=17&offset=34", ""}},
		{"run start compact", []string{"run", "start", "--template", "quality", "--project", "demo", "--goal", "ship", "--backend", "codex", "--model", "gpt", "--input", "spec=do it", "--stage", "work=claude/sonnet", "--ack-shared-workspace", "--json"}, "", wire{http.MethodPost, "/api/pipeline-runs", ""}},
		{"run start request full shape", []string{"run", "start", "--request-file", "-", "--json"}, `{"template_id":"quality","project":"demo","goal":"ship","inputs":{"n":4},"orchestrator":{"backend":"codex","model":"gpt","effort":"high","fast":true},"think_tank_assignments":{"debate":{"participants":{"a":{"backend":"claude","model":"sonnet","fast":true}},"judge":{"backend":"codex","model":"gpt","effort":"medium"}}}}`, wire{http.MethodPost, "/api/pipeline-runs", ""}},
		{"run show", []string{"run", "show", "run id", "--json"}, "", wire{http.MethodGet, "/api/pipeline-runs/run%20id", ""}},
		{"run continue", []string{"run", "continue", "r1", "--revision", "7", "--input", "again", "--json"}, "", wire{http.MethodPost, "/api/pipeline-runs/r1/continue", `{"revision":7,"input":"again"}`}},
		{"run retry", []string{"run", "retry", "r1", "--revision", "8", "--json"}, "", wire{http.MethodPost, "/api/pipeline-runs/r1/retry", `{"revision":8}`}},
		{"run replace", []string{"run", "replace", "r1", "--request-file", "-", "--json"}, `{"revision":9,"orchestrator":{"backend":"codex","model":"gpt","effort":"high","fast":true}}`, wire{http.MethodPost, "/api/pipeline-runs/r1/replace", `{"revision":9,"orchestrator":{"backend":"codex","model":"gpt","effort":"high","fast":true}}`}},
		{"run repair cleanup", []string{"run", "repair-cleanup", "r1", "--request-file", "-", "--json"}, `{"revision":10}`, wire{http.MethodPost, "/api/pipeline-runs/r1/repair-cleanup", `{"revision":10}`}},
		{"run stop", []string{"run", "stop", "r1", "--revision", "11", "--json"}, "", wire{http.MethodPost, "/api/pipeline-runs/r1/stop", `{"revision":11}`}},
		{"run delete", []string{"run", "delete", "r1", "--yes", "--json"}, "", wire{http.MethodDelete, "/api/pipeline-runs/r1", ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got wire
			var gotIfMatch string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got.method = r.Method
				got.path = r.URL.EscapedPath()
				gotIfMatch = r.Header.Get("If-Match")
				if r.URL.RawQuery != "" {
					got.path += "?" + r.URL.RawQuery
				}
				body, _ := io.ReadAll(r.Body)
				if len(body) > 0 {
					var value any
					if json.Unmarshal(body, &value) == nil {
						normalized, _ := json.Marshal(value)
						got.body = string(normalized)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodDelete {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				_, _ = w.Write([]byte(`{"accepted":true,"sequence":9007199254740993}`))
			}))
			defer server.Close()
			oldFactory := operatorClientFactory
			defer func() { operatorClientFactory = oldFactory }()
			operatorClientFactory = func() (*operatorClient, error) {
				return &operatorClient{HTTP: server.Client(), BaseURL: server.URL}, nil
			}
			root := NewRootCmd()
			root.SilenceErrors = true
			root.SilenceUsage = true
			root.SetArgs(append([]string{"pipeline"}, tc.args...))
			var out, diag bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&diag)
			root.SetIn(strings.NewReader(tc.input))
			if err := root.Execute(); err != nil {
				t.Fatalf("execute: %v; stderr=%s", err, diag.String())
			}
			if diag.Len() != 0 {
				t.Fatalf("unexpected stderr: %s", diag.String())
			}
			if !json.Valid(bytes.TrimSpace(out.Bytes())) {
				t.Fatalf("success output is not JSON: %s", out.String())
			}
			if tc.want.method == http.MethodDelete {
				if strings.TrimSpace(out.String()) != "{}" {
					t.Fatalf("empty-response output = %q; want {}", out.String())
				}
			} else if !bytes.Contains(out.Bytes(), []byte("9007199254740993")) {
				t.Fatalf("JSON output lost the exact large integer: %s", out.String())
			}
			if got.method != tc.want.method || got.path != tc.want.path {
				t.Fatalf("wire method/path = %s %s; want %s %s", got.method, got.path, tc.want.method, tc.want.path)
			}
			if gotIfMatch != "" {
				t.Fatalf("unexpected If-Match header: %q", gotIfMatch)
			}
			if tc.want.body != "" {
				var want any
				if err := json.Unmarshal([]byte(tc.want.body), &want); err != nil {
					t.Fatal(err)
				}
				wantBytes, _ := json.Marshal(want)
				if got.body != string(wantBytes) {
					t.Fatalf("body = %s; want %s", got.body, wantBytes)
				}
			}
			if tc.name == "run start compact" {
				var body map[string]json.RawMessage
				if err := json.Unmarshal([]byte(got.body), &body); err != nil {
					t.Fatal(err)
				}
				var requestID string
				_ = json.Unmarshal(body["request_id"], &requestID)
				var inputs map[string]string
				_ = json.Unmarshal(body["inputs"], &inputs)
				var assignments map[string]map[string]any
				_ = json.Unmarshal(body["assignments"], &assignments)
				var acknowledged bool
				_ = json.Unmarshal(body["acknowledge_shared_workspace"], &acknowledged)
				if requestID == "" || inputs["spec"] != "do it" || assignments["standing"]["backend"] != "codex" || !acknowledged {
					t.Fatalf("compact start body=%s", got.body)
				}
			}
			if tc.name == "run start request full shape" {
				var body map[string]json.RawMessage
				_ = json.Unmarshal([]byte(got.body), &body)
				var requestID string
				_ = json.Unmarshal(body["request_id"], &requestID)
				if requestID == "" || !bytes.Contains(body["think_tank_assignments"], []byte(`"fast":true`)) || !bytes.Contains(body["orchestrator"], []byte(`"effort":"high"`)) {
					t.Fatalf("request-file start lost shape: %s", got.body)
				}
			}
		})
	}
}

func TestOperatorPipelineLocalRefusalsSendNoRequest(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests++; _, _ = w.Write([]byte(`{}`)) }))
	defer server.Close()
	oldFactory := operatorClientFactory
	defer func() { operatorClientFactory = oldFactory }()
	operatorClientFactory = func() (*operatorClient, error) {
		return &operatorClient{HTTP: server.Client(), BaseURL: server.URL}, nil
	}
	cases := [][]string{
		{"template", "create", "--json"},
		{"template", "update", "quality", "--json"},
		{"template", "delete", "quality", "--json"},
		{"proposal", "delete", "proposal-1", "--json"},
		{"run", "delete", "run-1", "--json"},
		{"run", "continue", "run-1", "--json"},
		{"run", "retry", "run-1", "--revision", "0", "--json"},
		{"run", "replace", "run-1", "--request-file", "-", "--json"},
		{"run", "repair-cleanup", "run-1", "--json"},
		{"run", "list", "--limit", "101", "--json"},
		{"run", "start", "--request-file", "-", "--input", "x=1", "--json"},
		{"run", "start", "--request-file", "-", "--backend", "codex", "--json"},
		{"run", "start", "--template", "t", "--project", "p", "--goal", "g", "--backend", "codex", "--model", "m", "--stage", "work=codex/m", "--stage", "work=claude/m", "--json"},
	}
	for _, args := range cases {
		root := NewRootCmd()
		root.SilenceErrors = true
		root.SilenceUsage = true
		root.SetArgs(append([]string{"pipeline"}, args...))
		root.SetIn(strings.NewReader(`{"inputs":{"x":"from file"}}`))
		root.SetOut(io.Discard)
		root.SetErr(io.Discard)
		if err := root.Execute(); err == nil {
			t.Errorf("%v unexpectedly succeeded", args)
		}
	}
	if requests != 0 {
		t.Fatalf("refusals sent %d HTTP requests", requests)
	}
}
