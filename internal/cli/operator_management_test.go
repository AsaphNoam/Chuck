package cli

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

type managementTransport struct {
	check    func(*http.Request)
	response string
	header   http.Header
}

func (t managementTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if t.check != nil {
		t.check(r)
	}
	header := t.header.Clone()
	if header == nil {
		header = make(http.Header)
	}
	return &http.Response{StatusCode: http.StatusOK, Header: header, Body: io.NopCloser(strings.NewReader(t.response))}, nil
}

func withManagementTransport(t *testing.T, check func(*http.Request), response string) {
	t.Helper()
	old := operatorClientFactory
	operatorClientFactory = func() (*operatorClient, error) {
		return &operatorClient{HTTP: &http.Client{Transport: managementTransport{check: check, response: response}}, BaseURL: "http://operator.test"}, nil
	}
	t.Cleanup(func() { operatorClientFactory = old })
}

func withManagementResponse(t *testing.T, check func(*http.Request), response string, header http.Header) {
	t.Helper()
	old := operatorClientFactory
	operatorClientFactory = func() (*operatorClient, error) {
		return &operatorClient{HTTP: &http.Client{Transport: managementTransport{check: check, response: response, header: header}}, BaseURL: "http://operator.test"}, nil
	}
	t.Cleanup(func() { operatorClientFactory = old })
}

func managementTestCommand() *cobra.Command {
	c := &cobra.Command{}
	c.SetOut(io.Discard)
	c.SetErr(io.Discard)
	addOperatorFlags(c)
	return c
}

func TestOperatorProjectDeleteRequiresYesWithoutRequest(t *testing.T) {
	called := false
	old := operatorClientFactory
	operatorClientFactory = func() (*operatorClient, error) {
		called = true
		return &operatorClient{HTTP: http.DefaultClient, BaseURL: "http://127.0.0.1"}, nil
	}
	t.Cleanup(func() { operatorClientFactory = old })

	cmd := newOperatorProjectCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"delete", "--project", "demo"})
	err := cmd.Execute()
	if err == nil || called {
		t.Fatalf("delete err=%v called=%v", err, called)
	}
}

func TestOperatorConfigSourceBindRequiresYesWithoutRequest(t *testing.T) {
	called := false
	old := operatorClientFactory
	operatorClientFactory = func() (*operatorClient, error) { called = true; return nil, nil }
	t.Cleanup(func() { operatorClientFactory = old })
	cmd := newOperatorConfigSourceCmd()
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"bind", "--backend-id", "claude"})
	if err := cmd.Execute(); err == nil || called {
		t.Fatalf("err=%v called=%v", err, called)
	}
}

func TestOperatorBackendReplaceForwardsETagAndUnknownFields(t *testing.T) {
	withManagementTransport(t, func(r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/backends" || r.Header.Get("If-Match") != "abc" {
			t.Fatalf("request = %s %s etag=%q", r.Method, r.URL, r.Header.Get("If-Match"))
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"future":true`) {
			t.Fatalf("unknown field lost: %s", body)
		}
	}, `{"backends":{}}`)
	cmd := newOperatorBackendCmd()
	cmd.SetIn(strings.NewReader(`{"future":true}`))
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"replace", "--request-file", "-", "--etag", "abc"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestOperatorManagementLiteralWireRoutes(t *testing.T) {
	cases := []struct {
		name, family string
		command      *cobra.Command
		args         []string
		stdin        string
		method, path string
		query        string
		header       string
		body         string
		response     string
		stdout       string
	}{
		{name: "project list", family: "project", command: newOperatorProjectCmd(), args: []string{"list"}, method: http.MethodGet, path: "/api/projects"},
		{name: "project show", family: "project", command: newOperatorProjectCmd(), args: []string{"show", "--project", "demo"}, method: http.MethodGet, path: "/api/projects", response: `{"demo":{}}`, stdout: `{}`},
		{name: "project create", family: "project", command: newOperatorProjectCmd(), args: []string{"create", "--request-file", "-"}, stdin: `{"title":"Demo","cwd":"/tmp/demo"}`, method: http.MethodPost, path: "/api/projects", body: `{"cwd":"/tmp/demo","title":"Demo"}`},
		{name: "project edit", family: "project", command: newOperatorProjectCmd(), args: []string{"edit", "--project", "demo", "--request-file", "-"}, stdin: `{"title":"Renamed"}`, method: http.MethodPut, path: "/api/projects/demo", body: `{"title":"Renamed"}`},
		{name: "project delete independent flags", family: "project", command: newOperatorProjectCmd(), args: []string{"delete", "--project", "demo", "--yes", "--force", "--delete-checkout", "--dirty-known", "--dirty"}, method: http.MethodDelete, path: "/api/projects/demo", query: "delete_checkout=true&dirty=true&dirty_known=true&force=true"},
		{name: "project delete force alone", family: "project", command: newOperatorProjectCmd(), args: []string{"delete", "--project", "demo", "--yes", "--force"}, method: http.MethodDelete, path: "/api/projects/demo", query: "force=true"},
		{name: "project delete checkout without force", family: "project", command: newOperatorProjectCmd(), args: []string{"delete", "--project", "demo", "--yes", "--delete-checkout"}, method: http.MethodDelete, path: "/api/projects/demo", query: "delete_checkout=true"},
		{name: "project delete dirty false without checkout", family: "project", command: newOperatorProjectCmd(), args: []string{"delete", "--project", "demo", "--yes", "--dirty-known"}, method: http.MethodDelete, path: "/api/projects/demo", query: "dirty_known=true"},
		{name: "project archive checkout flags", family: "project", command: newOperatorProjectCmd(), args: []string{"archive", "--project", "demo", "--yes", "--delete-checkout", "--dirty-known", "--dirty"}, method: http.MethodPost, path: "/api/projects/demo/archive", body: `{"delete_checkout":true,"dirty":true,"dirty_known":true}`},
		{name: "project archive checkout flag alone", family: "project", command: newOperatorProjectCmd(), args: []string{"archive", "--project", "demo", "--yes", "--delete-checkout"}, method: http.MethodPost, path: "/api/projects/demo/archive", body: `{"delete_checkout":true}`},
		{name: "project archive dirty-known flag alone", family: "project", command: newOperatorProjectCmd(), args: []string{"archive", "--project", "demo", "--yes", "--dirty-known"}, method: http.MethodPost, path: "/api/projects/demo/archive", body: `{"dirty_known":true}`},
		{name: "project restore", family: "project", command: newOperatorProjectCmd(), args: []string{"restore", "--project", "demo"}, method: http.MethodPost, path: "/api/projects/demo/restore", body: `{}`},
		{name: "group stop project scoped label", family: "group", command: newOperatorGroupCmd(), args: []string{"stop", "--project", "demo", "--group", "blue", "--yes"}, method: http.MethodPost, path: "/api/projects/demo/groups/stop", body: `{"group":"blue"}`},
		{name: "group archive project scoped label", family: "group", command: newOperatorGroupCmd(), args: []string{"archive", "--project", "demo", "--group", "blue", "--yes"}, method: http.MethodPost, path: "/api/projects/demo/groups/archive", body: `{"group":"blue"}`},
		{name: "worktree fork setup warnings retained", family: "worktree", command: newOperatorWorktreeCmd(), args: []string{"fork", "--project", "demo", "--title", "Fork", "--branch", "fork", "--base", "main"}, method: http.MethodPost, path: "/api/projects/demo/worktree-fork", body: `{"base":"main","branch":"fork","title":"Fork"}`, response: `{"warning":"setup command failed","state":"created"}`},
		{name: "worktree status warnings retained", family: "worktree", command: newOperatorWorktreeCmd(), args: []string{"status", "--project", "demo"}, method: http.MethodGet, path: "/api/projects/demo/worktree", response: `{"warning":"worktree metadata unavailable","dirty":true}`},
		{name: "archive global search", family: "archive", command: newOperatorArchiveCmd(), args: []string{"search", "--query", "needle", "--active", "true", "--limit", "7", "--offset", "3"}, method: http.MethodGet, path: "/api/archive", query: "active=true&limit=7&offset=3&q=needle"},
		{name: "archive project detail", family: "archive", command: newOperatorArchiveCmd(), args: []string{"search", "--project", "demo", "--query", "needle", "--active", "false", "--limit", "7", "--offset", "3"}, method: http.MethodGet, path: "/api/archive/projects/demo", query: "active=false&limit=7&offset=3&q=needle"},
		{name: "role list", family: "role", command: newOperatorRoleCmd(), args: []string{"list"}, method: http.MethodGet, path: "/api/roles"},
		{name: "role show", family: "role", command: newOperatorRoleCmd(), args: []string{"show", "--id", "reviewer"}, method: http.MethodGet, path: "/api/roles", response: `{"reviewer":{}}`, stdout: `{}`},
		{name: "role create", family: "role", command: newOperatorRoleCmd(), args: []string{"create", "--request-file", "-"}, stdin: `{"role":"reviewer","title":"Reviewer"}`, method: http.MethodPost, path: "/api/roles", body: `{"role":"reviewer","title":"Reviewer"}`},
		{name: "role edit", family: "role", command: newOperatorRoleCmd(), args: []string{"edit", "--id", "reviewer", "--request-file", "-"}, stdin: `{"title":"Reviewer v2"}`, method: http.MethodPut, path: "/api/roles/reviewer", body: `{"title":"Reviewer v2"}`},
		{name: "role delete force", family: "role", command: newOperatorRoleCmd(), args: []string{"delete", "--id", "reviewer", "--yes", "--force"}, method: http.MethodDelete, path: "/api/roles/reviewer", query: "force=true"},
		{name: "backend list", family: "backend", command: newOperatorBackendCmd(), args: []string{"list"}, method: http.MethodGet, path: "/api/backends"},
		{name: "backend show", family: "backend", command: newOperatorBackendCmd(), args: []string{"show", "--id", "claude"}, method: http.MethodGet, path: "/api/backends", response: `{"backends":{"claude":{}}}`, stdout: `{}`},
		{name: "backend create", family: "backend", command: newOperatorBackendCmd(), args: []string{"create", "--id", "custom", "--name", "Custom", "--type", "claude"}, method: http.MethodPost, path: "/api/backends", body: `{"backend_id":"custom","name":"Custom","type":"claude"}`},
		{name: "backend replace etag", family: "backend", command: newOperatorBackendCmd(), args: []string{"replace", "--request-file", "-", "--etag", `W/"catalog"`}, stdin: `{"backends":{},"future":true}`, method: http.MethodPut, path: "/api/backends", header: `W/"catalog"`, body: `{"backends":{},"future":true}`},
		{name: "backend refresh etag", family: "backend", command: newOperatorBackendCmd(), args: []string{"refresh", "--id", "claude", "--etag", `W/"catalog"`, "--model-id", "sonnet"}, method: http.MethodPost, path: "/api/backends/claude/refresh-provider", header: `W/"catalog"`, body: `{"model_id":"sonnet"}`},
		{name: "config show", family: "config", command: newOperatorConfigCmd(), args: []string{"show"}, method: http.MethodGet, path: "/api/config"},
		{name: "config edit", family: "config", command: newOperatorConfigCmd(), args: []string{"edit", "--request-file", "-"}, stdin: `{"task_concurrency":3}`, method: http.MethodPut, path: "/api/config", body: `{"task_concurrency":3}`},
		{name: "federation list project", family: "config-source", command: newOperatorConfigSourceCmd(), args: []string{"list", "--project", "fed project"}, method: http.MethodGet, path: "/api/config-sources", query: "project=fed+project"},
		{name: "federation preview", family: "config-source", command: newOperatorConfigSourceCmd(), args: []string{"preview", "--request-file", "-"}, stdin: `{"provider":"claude","root":"/tmp","mode":"read","project":"demo"}`, method: http.MethodPost, path: "/api/config-sources/preview", body: `{"mode":"read","project":"demo","provider":"claude","root":"/tmp"}`},
		{name: "federation bind token and consent", family: "config-source", command: newOperatorConfigSourceCmd(), args: []string{"bind", "--backend-id", "claude", "--request-file", "-", "--yes"}, stdin: `{"preview_token":"issued","overrides":{"model":"sonnet"}}`, method: http.MethodPut, path: "/api/config-sources/claude", body: `{"overrides":{"model":"sonnet"},"preview_token":"issued"}`},
		{name: "federation refresh", family: "config-source", command: newOperatorConfigSourceCmd(), args: []string{"refresh", "--backend-id", "claude"}, method: http.MethodPost, path: "/api/config-sources/claude/refresh"},
		{name: "federation detach consent", family: "config-source", command: newOperatorConfigSourceCmd(), args: []string{"detach", "--backend-id", "claude", "--yes"}, method: http.MethodDelete, path: "/api/config-sources/claude"},
		{name: "remote show", family: "remote", command: newOperatorRemoteCmd(), args: []string{"show"}, method: http.MethodGet, path: "/api/remote"},
		{name: "remote configure", family: "remote", command: newOperatorRemoteCmd(), args: []string{"configure", "--enabled", "--keep-awake"}, method: http.MethodPut, path: "/api/remote", body: `{"enabled":true,"keep_awake":true}`},
		{name: "remote pairing code", family: "remote", command: newOperatorRemoteCmd(), args: []string{"pairing-code"}, method: http.MethodPost, path: "/api/remote/pairings"},
		{name: "remote allow pairing", family: "remote", command: newOperatorRemoteCmd(), args: []string{"allow", "--id", "pair1"}, method: http.MethodPost, path: "/api/remote/pairings/pair1/allow"},
		{name: "remote decline pairing", family: "remote", command: newOperatorRemoteCmd(), args: []string{"decline", "--id", "pair1"}, method: http.MethodPost, path: "/api/remote/pairings/pair1/decline"},
		{name: "remote devices", family: "remote", command: newOperatorRemoteCmd(), args: []string{"devices"}, method: http.MethodGet, path: "/api/remote/devices"},
		{name: "remote rename device", family: "remote", command: newOperatorRemoteCmd(), args: []string{"rename", "--id", "phone1", "--name", "Phone"}, method: http.MethodPatch, path: "/api/remote/devices/phone1", body: `{"name":"Phone"}`},
		{name: "remote revoke device consent", family: "remote", command: newOperatorRemoteCmd(), args: []string{"revoke", "--id", "phone1", "--yes"}, method: http.MethodDelete, path: "/api/remote/devices/phone1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := `{}`
			if tc.response != "" {
				response = tc.response
			}
			withManagementTransport(t, func(r *http.Request) {
				if r.Method != tc.method || r.URL.Path != tc.path || r.URL.RawQuery != tc.query {
					t.Fatalf("request = %s %s?%s, want %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery, tc.method, tc.path, tc.query)
				}
				if got := r.Header.Get("If-Match"); got != tc.header {
					t.Fatalf("If-Match = %q, want %q", got, tc.header)
				}
				var got []byte
				if r.Body != nil {
					var err error
					got, err = io.ReadAll(r.Body)
					if err != nil {
						t.Fatal(err)
					}
				}
				if string(got) != tc.body {
					t.Fatalf("body = %s, want %s", got, tc.body)
				}
			}, response)
			if tc.stdin != "" {
				tc.command.SetIn(strings.NewReader(tc.stdin))
			}
			var out strings.Builder
			tc.command.SetOut(&out)
			tc.command.SetErr(io.Discard)
			tc.command.SetArgs(append(tc.args, "--json"))
			if err := tc.command.Execute(); err != nil {
				t.Fatal(err)
			}
			wantOutput := tc.stdout
			if wantOutput == "" {
				wantOutput = response
			}
			if got := out.String(); got != wantOutput+"\n" {
				t.Fatalf("stdout = %q, want server JSON %q", got, wantOutput+"\n")
			}
		})
	}
}

func TestOperatorManagementPreservesShowRawNumber(t *testing.T) {
	withManagementTransport(t, nil, `{"demo":{"future":9007199254740993}}`)
	cmd := newOperatorProjectCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"show", "--project", "demo", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out.String()), &got); err != nil {
		t.Fatal(err)
	}
	if string(got["future"]) != "9007199254740993" {
		t.Fatalf("raw number changed: %s", got["future"])
	}
}

func runManagementWithoutRequest(t *testing.T, cmd *cobra.Command, args []string, input string) error {
	t.Helper()
	calls := 0
	old := operatorClientFactory
	operatorClientFactory = func() (*operatorClient, error) {
		calls++
		return &operatorClient{HTTP: &http.Client{Transport: managementTransport{response: `{}`}}, BaseURL: "http://operator.test"}, nil
	}
	t.Cleanup(func() { operatorClientFactory = old })
	if input != "" {
		cmd.SetIn(strings.NewReader(input))
	}
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs(args)
	err := cmd.Execute()
	if calls != 0 {
		t.Fatalf("invalid operation made %d request(s), err=%v", calls, err)
	}
	return err
}

func TestOperatorManagementInvalidInputMakesNoRequest(t *testing.T) {
	oversized := strings.Repeat("x", int(operatorInputLimit)+1)
	cases := []struct {
		name  string
		cmd   *cobra.Command
		args  []string
		input string
	}{{
		name: "missing project target", cmd: newOperatorProjectCmd(), args: []string{"show", "--json"},
	}, {
		name: "missing required project fields", cmd: newOperatorProjectCmd(), args: []string{"create", "--request-file", "-", "--json"}, input: `{}`,
	}, {
		name: "conflicting project field", cmd: newOperatorProjectCmd(), args: []string{"create", "--project", "demo", "--request-file", "-", "--json"}, input: `{"project":"other","title":"Demo","cwd":"/tmp"}`,
	}, {
		name: "malformed request", cmd: newOperatorConfigCmd(), args: []string{"edit", "--request-file", "-", "--json"}, input: `{"task_concurrency":`,
	}, {
		name: "oversized request", cmd: newOperatorConfigCmd(), args: []string{"edit", "--request-file", "-", "--json"}, input: oversized,
	}, {
		name: "role list rejects request json without reading it", cmd: newOperatorRoleCmd(), args: []string{"list", "--request-file", "-", "--json"}, input: `malformed`,
	}, {
		name: "role show rejects request json without reading it", cmd: newOperatorRoleCmd(), args: []string{"show", "--id", "reviewer", "--request-file", "-", "--json"}, input: `malformed`,
	}, {
		name: "role delete rejects request json without reading it", cmd: newOperatorRoleCmd(), args: []string{"delete", "--id", "reviewer", "--yes", "--request-file", "-", "--json"}, input: `malformed`,
	}, {
		name: "immutable config version", cmd: newOperatorConfigCmd(), args: []string{"edit", "--request-file", "-", "--json"}, input: `{"version":14}`,
	}, {
		name: "project delete dirty without known", cmd: newOperatorProjectCmd(), args: []string{"delete", "--project", "demo", "--yes", "--dirty", "--json"},
	}, {
		name: "project archive dirty without known", cmd: newOperatorProjectCmd(), args: []string{"archive", "--project", "demo", "--yes", "--dirty", "--json"},
	}, {
		name: "project archive request dirty without known", cmd: newOperatorProjectCmd(), args: []string{"archive", "--project", "demo", "--yes", "--request-file", "-", "--json"}, input: `{"dirty":true}`,
	}, {
		name: "project delete consent", cmd: newOperatorProjectCmd(), args: []string{"delete", "--project", "demo", "--json"},
	}, {
		name: "group archive consent", cmd: newOperatorGroupCmd(), args: []string{"archive", "--project", "demo", "--group", "blue", "--json"},
	}, {
		name: "federation bind consent", cmd: newOperatorConfigSourceCmd(), args: []string{"bind", "--backend-id", "claude", "--request-file", "-", "--json"}, input: `{"preview_token":"issued"}`,
	}, {
		name: "federation detach consent", cmd: newOperatorConfigSourceCmd(), args: []string{"detach", "--backend-id", "claude", "--json"},
	}, {
		name: "missing preview token", cmd: newOperatorConfigSourceCmd(), args: []string{"bind", "--backend-id", "claude", "--request-file", "-", "--yes", "--json"}, input: `{"overrides":{}}`,
	}, {
		name: "backend replace missing etag", cmd: newOperatorBackendCmd(), args: []string{"replace", "--request-file", "-", "--json"}, input: `{"backends":{}}`,
	}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := runManagementWithoutRequest(t, tc.cmd, tc.args, tc.input); err == nil {
				t.Fatal("expected local refusal")
			}
		})
	}
}

func TestOperatorBackendListSavesExactResponseETagWithoutChangingJSONOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.etag")
	const body = `{"backends":{"claude":{"id":"claude"}},"future":9007199254740993}`
	withManagementResponse(t, func(r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/backends" || r.URL.RawQuery != "" {
			t.Fatalf("request = %s %s", r.Method, r.URL)
		}
	}, body, http.Header{"Etag": []string{`W/"catalog-7"`}})
	cmd := newOperatorBackendCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"list", "--etag-output", path, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != body+"\n" {
		t.Fatalf("stdout = %q, want exact server JSON", out.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `W/"catalog-7"`+"\n" {
		t.Fatalf("saved ETag = %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("ETag mode = %o, want 600", info.Mode().Perm())
	}
}

func TestOperatorBackendListETagDestinationRefusesOverwriteBeforeRequest(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.etag")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	err := runManagementWithoutRequest(t, newOperatorBackendCmd(), []string{"list", "--etag-output", path, "--json"}, "")
	if err == nil {
		t.Fatal("expected overwrite refusal")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "keep" {
		t.Fatalf("destination = %q, err=%v", data, err)
	}
}

func TestOperatorManagementLeavesEveryLeafWithPracticalExample(t *testing.T) {
	for _, root := range newOperatorManagementCommands() {
		var visit func(*cobra.Command)
		visit = func(parent *cobra.Command) {
			if len(parent.Commands()) == 0 {
				if strings.TrimSpace(parent.Example) == "" {
					t.Errorf("%s has no example", parent.CommandPath())
				}
				if strings.Contains(parent.Example, "--json") && parent.Flags().Lookup("json") == nil {
					t.Errorf("%s example uses an unsupported --json flag", parent.CommandPath())
				}
				return
			}
			for _, child := range parent.Commands() {
				visit(child)
			}
		}
		visit(root)
	}
}

func TestOperatorRoleEditHelpDescribesFullReplacement(t *testing.T) {
	cmd, _, err := newOperatorRoleCmd().Find([]string{"edit"})
	if err != nil {
		t.Fatal(err)
	}
	help := strings.ToLower(cmd.Long + "\n" + cmd.Example)
	if strings.Contains(help, "partial") {
		t.Errorf("role edit help still says partial: %s", help)
	}
	for _, want := range []string{"replace", "reset", "role show --id", "--json"} {
		if !strings.Contains(help, want) {
			t.Errorf("role edit help missing %q: %s", want, help)
		}
	}
}

func TestOperatorCollectionShowClassifiesMalformedCollection(t *testing.T) {
	for _, response := range []string{`[]`, `null`, `{"backends":null}`, `{}`} {
		t.Run(response, func(t *testing.T) {
			withManagementTransport(t, nil, response)
			cmd := newOperatorBackendCmd()
			cmd.SetArgs([]string{"show", "--id", "claude"})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			err := cmd.Execute()
			var op *operatorError
			if !errors.As(err, &op) || op.Code != "invalid_response" || op.ExitCode != 1 {
				t.Fatalf("error = %#v, want invalid_response exit 1", err)
			}
		})
	}
}
