package server

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/backend/providerexec"
	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/hooks"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

func TestServerLaunchSpecHasOneBuilder(t *testing.T) {
	_, path, _, _ := goruntime.Caller(0)
	files, err := filepath.Glob(filepath.Join(filepath.Dir(path), "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	var builders []string
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok || len(lit.Elts) == 0 {
				return true
			}
			if typ, ok := lit.Type.(*ast.SelectorExpr); ok && typ.Sel.Name == "LaunchSpec" {
				builders = append(builders, filepath.Base(name))
			}
			return true
		})
	}
	if len(builders) != 1 || builders[0] != "lifecycle_start.go" {
		t.Fatalf("launch spec builders = %v", builders)
	}
}

func TestBuildLaunchSpecPreservesOriginInputs(t *testing.T) {
	for _, tc := range []struct {
		origin, generation, driver, nativeID, primer string
		contextPct                                   float64
		fork                                         *runtime.ForkPlan
	}{
		{origin: "launch", generation: "attempt", driver: "xterm"},
		{origin: "clone", primer: "fork primer", fork: &runtime.ForkPlan{}},
		{origin: "resume", nativeID: "native", contextPct: 17, primer: "frozen primer"},
		{origin: "switch", nativeID: "target-native", contextPct: 17, primer: "frozen primer"},
		{origin: "switch cross backend", contextPct: 17, primer: "frozen primer"},
		{origin: "switch rollback", nativeID: "prior-native", contextPct: 17, primer: "frozen primer"},
	} {
		t.Run(tc.origin, func(t *testing.T) {
			srv := testServer(t, true)
			agent := state.Agent{AgentID: "a_123456", Interface: "chat", Effort: "high", Fast: true}
			contract := &state.ExecutionContract{}
			snap := state.SessionSnapshot{
				Cwd: "/cwd", WorkspaceID: "workspace", AddDirs: []string{"/extra"},
				SystemPrompt: "frozen persona", HistoryPrimer: tc.primer, ChuckAware: false,
				Effort: agent.Effort, Fast: agent.Fast, SkipPermissions: true,
				LastSessionID: tc.nativeID, LastContextPct: tc.contextPct, ExecutionContract: contract,
			}
			spec, ae := srv.buildLaunchSpec(startSpec{
				agent: agent, session: snap, backend: config.Backend{Type: "claude-acp"},
				provider:    providerexec.Selection{Path: "/provider", Source: "bundled"},
				resourceDir: filepath.Join(srv.configStore.Home(), "resources"),
				generation:  tc.generation, driver: tc.driver, fork: tc.fork,
			})
			if ae != nil {
				t.Fatal(ae)
			}
			generation := tc.generation
			if generation == "" {
				generation = spec.HookToken
			}
			if generation == "" || spec.Generation != generation || spec.Agent != agent ||
				spec.Cwd != "/cwd" || spec.WorkspaceID != "workspace" || len(spec.AddDirs) != 1 || spec.AddDirs[0] != "/extra" ||
				spec.SystemPrompt != "frozen persona" || spec.HistoryPrimer != tc.primer || spec.ChuckAware == nil || *spec.ChuckAware ||
				spec.Driver != tc.driver || spec.LastSessionID != tc.nativeID || spec.LastContextPct != tc.contextPct || spec.Fork != tc.fork ||
				spec.Effort != "high" || !spec.Fast || !spec.SkipPerms || spec.ExecutionContract != contract ||
				spec.ProviderExecutable != "/provider" || spec.ProviderSource != "bundled" || len(spec.MCPServers) != 1 {
				t.Fatalf("spec for %s = %+v", tc.origin, spec)
			}
		})
	}
}

func TestRenameDuringLaunchHandshakeSurvives(t *testing.T) {
	srv := testServer(t, true)
	srv.registry.Chat().SetCommand(buildFakeACP(t))
	t.Setenv("FAKEACP_SCENARIO", "stream_text")
	if err := srv.configStore.WriteProject("tmpproj", config.Project{Title: "Tmp", Cwd: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	if err := srv.configStore.WriteRole("impl", config.Role{Title: "Impl"}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { srv.registry.Shutdown(context.Background()) })
	handler := srv.routes()
	request := func(route, body string) *httptest.ResponseRecorder {
		req := newLocalRequest(http.MethodPost, route, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	dir := t.TempDir()
	dump, release := filepath.Join(dir, "new.json"), filepath.Join(dir, "release")
	t.Setenv("FAKEACP_NEW_DUMP", dump)
	t.Setenv("FAKEACP_NEW_HOLD_FILE", release)
	t.Cleanup(func() { _ = os.WriteFile(release, []byte("go"), 0o600) })

	launched := make(chan int, 1)
	go func() {
		launched <- request("/api/sessions", `{"role":"impl","project":"tmpproj","name":"Before"}`).Code
	}()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(dump); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("launch never reached session/new")
		}
		time.Sleep(5 * time.Millisecond)
	}
	agents, err := srv.stateStore.ListAgents()
	if err != nil || len(agents) != 1 {
		t.Fatalf("in-flight agents = %v, %v", agents, err)
	}
	id := agents[0].AgentID
	resp := request("/api/sessions/"+id+"/rename", `{"name":"After"}`)
	if resp.Code != http.StatusOK {
		t.Fatalf("rename during handshake = %d: %s", resp.Code, resp.Body.String())
	}
	if err := os.WriteFile(release, []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	if status := <-launched; status != http.StatusCreated {
		t.Fatalf("launch = %d, want 201", status)
	}
	agent, err := srv.stateStore.ReadAgent(id)
	if err != nil || agent.Name != "After" {
		t.Fatalf("agent after handshake = %+v, %v", agent, err)
	}
}

func TestDeleteProjectRechecksAgentAfterStartLease(t *testing.T) {
	srv := testServer(t, true)
	const project = "my-app"
	if ae := srv.acquireProjectStart(project); ae != nil {
		t.Fatal(ae)
	}
	leased := true
	defer func() {
		if leased {
			srv.releaseProjectStart(project)
		}
	}()
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		rec := httptest.NewRecorder()
		srv.routes().ServeHTTP(rec, newLocalRequest(http.MethodDelete, "/api/projects/"+project+"?delete_checkout=true", nil))
		done <- rec
	}()
	deadline := time.Now().Add(3 * time.Second)
	for {
		srv.archiveMu.Lock()
		claimed := srv.projectArchiving[project]
		srv.archiveMu.Unlock()
		if claimed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("delete never acquired the project claim")
		}
		time.Sleep(time.Millisecond)
	}
	const id = "a_123456"
	now := time.Now().UTC()
	if err := srv.stateStore.WriteAgent(state.Agent{AgentID: id, Project: project, Interface: "chat", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := srv.stateStore.WriteRunning(state.RunningEntry{AgentID: id, PID: os.Getpid(), Interface: "chat", StartedAt: now}); err != nil {
		t.Fatal(err)
	}
	srv.releaseProjectStart(project)
	leased = false
	select {
	case rec := <-done:
		if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "in_use") {
			t.Fatalf("delete after agent start = %d: %s", rec.Code, rec.Body.String())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("delete did not finish after release")
	}
	if _, err := srv.configStore.ReadProject(project); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverIncompleteLaunchesCleansRegistration(t *testing.T) {
	srv := testServer(t, true)
	agent := state.Agent{AgentID: "a_123456", CreatedAt: time.Now().UTC()}
	if err := srv.stateStore.WriteAgent(agent); err != nil {
		t.Fatal(err)
	}
	mcp, err := srv.writeMessagingMCPConfig(agent.AgentID, runtime.MCPServerSpec{})
	if err != nil {
		t.Fatal(err)
	}
	settings, err := hooks.WriteAgentSettings(srv.configStore.Home(), agent.AgentID, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	srv.recoverIncompleteLaunches()
	for _, path := range []string{mcp, settings} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("stat %q after recovery = %v", path, err)
		}
	}
	if _, err := srv.stateStore.ReadAgent(agent.AgentID); !errors.Is(err, state.ErrNotFound) {
		t.Fatalf("read incomplete identity after recovery = %v", err)
	}
}

func TestStartupRestoresStoppedCrossBackendSwitch(t *testing.T) {
	srv := testServer(t, true)
	const id = "a_123456"
	at := time.Now().UTC()
	target := state.Agent{AgentID: id, Name: "Renamed", Backend: "codex", Model: "target", Interface: "chat", Effort: "high", CreatedAt: at}
	if err := srv.stateStore.WriteAgent(target); err != nil {
		t.Fatal(err)
	}
	_, err := srv.stateStore.DB().Exec(`
INSERT INTO sessions(agent_id, name, role, project, backend, model, interface, effort, fast, cwd, system_prompt, created_at, updated_at)
VALUES (?, 'Before', '', '', 'claude', 'sonnet', 'terminal', 'medium', 1, '', '', ?, ?)`, id, at.Format(time.RFC3339), at.Format(time.RFC3339))
	if err != nil {
		t.Fatal(err)
	}
	srv.recoverIncompleteLaunches()
	got, err := srv.stateStore.ReadAgent(id)
	if err != nil || got.Name != "Renamed" || got.Backend != "claude" || got.Model != "sonnet" ||
		got.Interface != "terminal" || got.Effort != "medium" || !got.Fast {
		t.Fatalf("identity after interrupted Switch = %+v, %v", got, err)
	}
}
