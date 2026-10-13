package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestOperatorAgentWireInventory(t *testing.T) {
	// This inventory is supplied from FS-22 and registered server routes, not
	// generated from the command constructors being tested.
	cases := []struct {
		args               []string
		method, path, body string
	}{
		{[]string{"list"}, "GET", "/api/sessions", ""},
		{[]string{"show", "a b"}, "GET", "/api/sessions/a%20b", ""},
		{[]string{"launch", "--role", "default", "--project", "p", "--fast=false", "--driver", "tmux"}, "POST", "/api/sessions", `{"role":"default","project":"p","fast":false,"driver":"tmux"}`},
		{[]string{"resume", "a"}, "POST", "/api/sessions/a/resume", `{}`},
		{[]string{"stop", "a"}, "POST", "/api/sessions/a/stop", `{}`},
		{[]string{"clone", "a"}, "POST", "/api/sessions/a/clone", `{}`},
		{[]string{"archive", "a"}, "POST", "/api/sessions/a/archive", `{}`},
		{[]string{"restore", "a"}, "POST", "/api/sessions/a/restore", `{}`},
		{[]string{"rename", "a", "--name", "New"}, "POST", "/api/sessions/a/rename", `{"name":"New"}`},
		{[]string{"identity", "a", "--group", ""}, "POST", "/api/sessions/a/identity", `{"group":""}`},
		{[]string{"cancel", "a"}, "POST", "/api/sessions/a/cancel", `{"target":"turn"}`},
		{[]string{"cancel", "a", "--target", "quota_continuation", "--episode-id", "e", "--revision", "7"}, "POST", "/api/sessions/a/cancel", `{"target":"quota_continuation","episode_id":"e","revision":7}`},
		{[]string{"switch-runtime", "a", "--interface", "chat", "--backend", "claude", "--model", "m"}, "POST", "/api/sessions/a/switch-runtime", `{"interface":"chat","backend":"claude","model":"m"}`},
		{[]string{"session-config", "a", "--fast=false"}, "POST", "/api/sessions/a/session-config", `{"fast":false}`},
		{[]string{"send", "a", "--text", "こんにちは\nline\n"}, "POST", "/api/sessions/a/prompt", `{"text":"こんにちは\nline\n"}`},
		{[]string{"steer", "a"}, "POST", "/api/sessions/a/steer", `{}`},
		{[]string{"held", "a"}, "GET", "/api/sessions/a/prompt", ""},
		{[]string{"withdraw", "a", "--recovery-request-id", "r"}, "DELETE", "/api/sessions/a/prompt?recovery_request_id=r", ""},
		{[]string{"approve", "a", "--tool-call-id", "child-call"}, "POST", "/api/sessions/a/permission", `{"tool_call_id":"child-call","decision":"approve"}`},
		{[]string{"deny", "a", "--tool-call-id", "call"}, "POST", "/api/sessions/a/permission", `{"tool_call_id":"call","decision":"deny"}`},
		{[]string{"background-task-stop", "a", "--task-id", "task", "--activity-id", "child"}, "POST", "/api/sessions/a/background-task-stop", `{"task_id":"task","activity_id":"child"}`},
		{[]string{"files", "a"}, "GET", "/api/sessions/a/files", ""},
		{[]string{"commands", "a"}, "GET", "/api/sessions/a/commands", ""},
		{[]string{"available-commands", "a"}, "GET", "/api/sessions/a/available-commands", ""},
		{[]string{"file-search", "a", "--query", "a&b"}, "GET", "/api/sessions/a/file-search?q=a%26b", ""},
		{[]string{"file", "a", "--path", "folder/a b"}, "GET", "/api/sessions/a/file?path=folder%2Fa+b", ""},
		{[]string{"messages", "a", "--limit", "12", "--unread-only"}, "GET", "/api/sessions/a/messages?limit=12&unread_only=true", ""},
		{[]string{"think-tank-results", "a", "--after", "17"}, "GET", "/api/sessions/a/think-tank-results?after=17", ""},
		{[]string{"annotate", "a", "--request-file", "-"}, "POST", "/api/sessions/a/annotations", `{"annotations":[{"anchor":"chat","excerpt":"x","instruction":"fix"}],"target":{"kind":"agent","agent_id":"b"}}`},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			called := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if r.Method != tc.method || r.URL.RequestURI() != tc.path {
					t.Errorf("wire = %s %s, want %s %s", r.Method, r.URL.RequestURI(), tc.method, tc.path)
				}
				b, _ := io.ReadAll(r.Body)
				if tc.body == "" {
					if len(b) != 0 {
						t.Errorf("unexpected body %s", b)
					}
				} else {
					var got, want any
					_ = json.Unmarshal(b, &got)
					_ = json.Unmarshal([]byte(tc.body), &want)
					if !reflect.DeepEqual(got, want) {
						t.Errorf("body %s want %s", b, tc.body)
					}
				}
				_, _ = w.Write([]byte(`{"accepted":true,"future":9007199254740993}`))
			}))
			defer server.Close()
			old := operatorClientFactory
			operatorClientFactory = func() (*operatorClient, error) {
				return &operatorClient{HTTP: server.Client(), BaseURL: server.URL}, nil
			}
			defer func() { operatorClientFactory = old }()
			cmd := newOperatorAgentCmd()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(io.Discard)
			cmd.SetIn(strings.NewReader(tc.body))
			cmd.SetArgs(append(tc.args, "--json"))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if !called {
				t.Fatal("no wire request")
			}
			if !strings.Contains(out.String(), "9007199254740993") {
				t.Fatalf("machine fields lost: %s", out.String())
			}
		})
	}
}

func TestOperatorAgentRejectsInputBeforeAnyRequest(t *testing.T) {
	for _, args := range [][]string{{"launch"}, {"send", "a"}, {"send", "a", "--text", "x", "--text-file", "-"}, {"send", "a", "--text", strings.Repeat("x", (64<<10)+1)}, {"cancel", "a", "--target", "quota_continuation"}, {"approve", "a"}, {"file", "a"}, {"session-config", "a", "--fast=false", "--request-file", "-"}} {
		cmd := newOperatorAgentCmd()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetIn(strings.NewReader(`{"fast":true}`))
		cmd.SetArgs(args)
		old := operatorClientFactory
		called := false
		operatorClientFactory = func() (*operatorClient, error) { called = true; return nil, operatorInvalid("unexpected") }
		err := cmd.Execute()
		operatorClientFactory = old
		if err == nil || called {
			t.Fatalf("%v err=%v requested=%v", args, err, called)
		}
	}
}

func TestOperatorMultipartAndOwnedDownload(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(source, []byte("file bytes\n"), 0600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			if r.URL.Path != "/api/sessions/a/attachments/owned" {
				t.Errorf("path %s", r.URL)
			}
			_, _ = w.Write([]byte("owned bytes"))
			return
		}
		if r.Method != "POST" || r.URL.Path != "/api/sessions/a/prompt" {
			t.Errorf("wire %s %s", r.Method, r.URL)
		}
		mr, err := r.MultipartReader()
		if err != nil {
			t.Error(err)
			return
		}
		part, err := mr.NextPart()
		if err != nil {
			t.Error(err)
			return
		}
		metadata, _ := io.ReadAll(part)
		var value struct {
			RequestID string                        `json:"request_id"`
			Files     []struct{ Part, Name string } `json:"files"`
		}
		if json.Unmarshal(metadata, &value) != nil || value.RequestID != "retry-1" || len(value.Files) != 1 || value.Files[0].Name != "note.txt" {
			t.Errorf("metadata %s", metadata)
		}
		if part.FormName() != "message" {
			t.Errorf("first part %s", part.FormName())
		}
		part, err = mr.NextPart()
		if err != nil {
			t.Error(err)
			return
		}
		content, _ := io.ReadAll(part)
		if string(content) != "file bytes\n" || part.FormName() != value.Files[0].Part {
			t.Errorf("part %s bytes %q", part.FormName(), content)
		}
		_, _ = w.Write([]byte(`{"sent":true,"request_id":"retry-1"}`))
	}))
	defer server.Close()
	old := operatorClientFactory
	operatorClientFactory = func() (*operatorClient, error) {
		return &operatorClient{HTTP: server.Client(), BaseURL: server.URL}, nil
	}
	defer func() { operatorClientFactory = old }()
	cmd := newOperatorAgentCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"send", "a", "--attach", source, "--request-id", "retry-1"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "copy.txt")
	cmd = newOperatorAgentCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"attachment", "a", "owned", "--output", output})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(output)
	if err != nil || string(data) != "owned bytes" {
		t.Fatalf("download %q %v", data, err)
	}
	info, _ := os.Stat(output)
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	cmd = newOperatorAgentCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"attachment", "a", "owned", "--output", output})
	if err := cmd.Execute(); err == nil {
		t.Fatal("overwrite allowed")
	}
}
