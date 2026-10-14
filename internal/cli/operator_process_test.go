package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/server"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// FS-22's process evidence uses the built executable against the real server
// and its deterministic ACP peer, rather than a command-table-derived oracle.
func TestOperatorBuiltBinaryRealServerJourney(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CHUCK_HOME", home)
	t.Setenv("CHUCK_RUNTIME_ROOT", "")
	build := func(target, pkg string) {
		t.Helper()
		c := exec.Command("go", "build", "-o", target, pkg)
		if b, err := c.CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", pkg, err, b)
		}
	}
	bin := filepath.Join(t.TempDir(), "chuck")
	fake := filepath.Join(t.TempDir(), "fakeacp")
	build(bin, "../../cmd/chuck")
	build(fake, "../runtime/testdata/fakeacp")
	t.Setenv("FAKEACP_SCENARIO", "hold_turn")
	t.Setenv("FAKEACP_HOLD_FILE", filepath.Join(home, "release"))
	t.Setenv("FAKEACP_STEERING", "1")
	t.Setenv("FAKEACP_IMAGE_INPUT", "1")
	cfgStore := config.NewWithHome(home)
	if err := cfgStore.EnsureLayout(); err != nil {
		t.Fatal(err)
	}
	if err := cfgStore.SeedIfAbsent(); err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	cfg := config.DefaultConfig()
	cfg.Port = port
	cfg.OnboardingComplete = true
	if err := cfgStore.WriteConfig(cfg); err != nil {
		t.Fatal(err)
	}
	if err := cfgStore.WriteProject("cli-test", config.Project{Title: "CLI test", Cwd: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	st, err := state.Open(home)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	registry := runtime.NewRegistry(st)
	registry.Chat().SetCommand(fake)
	srv := server.New(cfgStore, st, registry, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- srv.Start(ctx) }()
	defer func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("server shutdown: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Error("server shutdown timed out")
		}
	}()
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	until := time.Now().Add(10 * time.Second)
	for {
		resp, err := http.Get(base + "/api/health")
		if err == nil {
			_ = resp.Body.Close()
			break
		}
		if time.Now().After(until) {
			t.Fatalf("server startup: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	run := func(input string, exit int, args ...string) json.RawMessage {
		t.Helper()
		c := exec.CommandContext(t.Context(), bin, args...)
		c.Stdin = strings.NewReader(input)
		var out, diag bytes.Buffer
		c.Stdout = &out
		c.Stderr = &diag
		err := c.Run()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		if code != exit {
			t.Fatalf("%v exit %d want %d\nstdout %s\nstderr %s", args, code, exit, out.String(), diag.String())
		}
		if exit != 0 {
			if out.Len() != 0 {
				t.Fatalf("rejection stdout %s", out.String())
			}
			if !json.Valid(bytes.TrimSpace(diag.Bytes())) {
				t.Fatalf("diagnostic is not one JSON value: %s", diag.String())
			}
			return nil
		}
		if diag.Len() != 0 {
			t.Fatalf("success diagnostic %s", diag.String())
		}
		if !json.Valid(bytes.TrimSpace(out.Bytes())) {
			t.Fatalf("success invalid JSON %s", out.String())
		}
		return append(json.RawMessage(nil), out.Bytes()...)
	}
	run("", 0, "status", "--json")
	run("", 2, "agent", "send", "missing", "--json")
	launch := run("", 0, "agent", "launch", "--role", "default", "--project", "cli-test", "--json")
	var created struct {
		Agent struct {
			ID string `json:"agent_id"`
		} `json:"agent"`
	}
	if err := json.Unmarshal(launch, &created); err != nil || created.Agent.ID == "" {
		t.Fatalf("launch %s %v", launch, err)
	}
	id := created.Agent.ID
	waitState := func(want string) {
		t.Helper()
		end := time.Now().Add(10 * time.Second)
		for time.Now().Before(end) {
			status, err := st.ReadStatus(id)
			if err == nil && status.State == want {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("agent did not reach %s", want)
	}
	waitState("idle")
	run("", 0, "agent", "send", id, "--text", "first turn", "--json")
	waitState("busy")
	input := "Unicode: שלום\nsecond line\n"
	held := run(input, 0, "agent", "send", id, "--text-file", "-", "--json")
	if !bytes.Contains(held, []byte(`"held":true`)) && !bytes.Contains(held, []byte(`"delivery":"held"`)) {
		t.Fatalf("busy send %s", held)
	}
	hold := run("", 0, "agent", "held", id, "--json")
	if !bytes.Contains(hold, []byte("second line")) {
		t.Fatalf("held text %s", hold)
	}
	run("", 0, "agent", "send", id, "--text", "replacement", "--json")
	run("", 0, "agent", "withdraw", id, "--json")
	steered := run("", 0, "agent", "steer", id, "--text", "correction", "--json")
	if !bytes.Contains(steered, []byte(`"steered":true`)) && !bytes.Contains(steered, []byte(`"outcome":"steered"`)) {
		t.Fatalf("steer %s", steered)
	}
	run("", 0, "agent", "cancel", id, "--json")
	waitState("idle")
	source := filepath.Join(home, "input.txt")
	if err := os.WriteFile(source, []byte("immutable local bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	run("", 0, "agent", "send", id, "--attach", source, "--request-id", "cli-file-1", "--json")
	if err := os.WriteFile(source, []byte("changed source"), 0600); err != nil {
		t.Fatal(err)
	}
	transcript := run("", 0, "agent", "transcript", id, "--json", "--include-meta")
	var history struct {
		Events []struct {
			Data struct {
				Attachments []struct {
					ID string `json:"id"`
				} `json:"attachments"`
			} `json:"data"`
		} `json:"events"`
	}
	if err := json.Unmarshal(transcript, &history); err != nil {
		t.Fatal(err)
	}
	owned := ""
	for _, ev := range history.Events {
		if len(ev.Data.Attachments) > 0 {
			owned = ev.Data.Attachments[0].ID
		}
	}
	if owned == "" {
		t.Fatalf("no durable attachment: %s", transcript)
	}
	copy := filepath.Join(home, "download.txt")
	run("", 0, "agent", "attachment", id, owned, "--output", copy, "--json")
	b, err := os.ReadFile(copy)
	if err != nil || string(b) != "immutable local bytes" {
		t.Fatalf("owned copy %q %v", b, err)
	}
	run("", 2, "agent", "attachment", id, owned, "--output", copy, "--json")
	// Lose one successful multipart response through a loopback proxy, then
	// explicitly reuse identical caller input. The retained hold keeps one copy.
	lossSource := filepath.Join(home, "loss.txt")
	if err := os.WriteFile(lossSource, []byte("retry bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	var submissions atomic.Int32
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequestWithContext(r.Context(), r.Method, base+r.URL.RequestURI(), r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		req.Header = r.Header.Clone()
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Error(err)
			return
		}
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/prompt") && submissions.Add(1) == 1 && resp.StatusCode < 300 {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Error(err)
				return
			}
			_ = conn.Close()
			return
		}
		for k, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(k, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
	}))
	defer proxy.Close()
	proxyPort := proxy.Listener.Addr().(*net.TCPAddr).Port
	proxyCfg := cfg
	proxyCfg.Port = proxyPort
	if err := cfgStore.WriteConfig(proxyCfg); err != nil {
		t.Fatal(err)
	}
	run("", 3, "agent", "send", id, "--attach", lossSource, "--request-id", "loss-1", "--json")
	run("", 0, "agent", "send", id, "--attach", lossSource, "--request-id", "loss-1", "--json")
	if err := os.WriteFile(lossSource, []byte("different retry bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	run("", 1, "agent", "send", id, "--attach", lossSource, "--request-id", "loss-1", "--json")
	if submissions.Load() != 3 {
		t.Fatalf("mutation replayed automatically: submissions=%d", submissions.Load())
	}
	if err := cfgStore.WriteConfig(cfg); err != nil {
		t.Fatal(err)
	}
	proxy.Close()
	heldRetry := run("", 0, "agent", "held", id, "--json")
	var retryHold struct {
		RequestID   string                      `json:"request_id"`
		Attachments []struct{ ID, Name string } `json:"attachments"`
	}
	_ = json.Unmarshal(heldRetry, &retryHold)
	retryOwned := ""
	if retryHold.RequestID == "loss-1" {
		if len(retryHold.Attachments) != 1 {
			t.Fatalf("duplicate hold attachments %s", heldRetry)
		}
		retryOwned = retryHold.Attachments[0].ID
	} else {
		// Acceptance may deliver immediately rather than hold; retain exactly one
		// durable prompt in either case without asserting completed model work.
		retryHistory := run("", 0, "agent", "transcript", id, "--json")
		var events struct {
			Events []struct {
				Data struct {
					Attachments []struct{ ID, Name string } `json:"attachments"`
				} `json:"data"`
			} `json:"events"`
		}
		_ = json.Unmarshal(retryHistory, &events)
		copies := 0
		for _, ev := range events.Events {
			for _, a := range ev.Data.Attachments {
				if a.Name == "loss.txt" {
					copies++
					retryOwned = a.ID
				}
			}
		}
		if copies != 1 {
			t.Fatalf("identical retry produced %d retained prompts: %s", copies, retryHistory)
		}
	}
	retryOutput := filepath.Join(home, "retry-output.txt")
	run("", 0, "agent", "attachment", id, retryOwned, "--output", retryOutput, "--json")
	retained, err := os.ReadFile(retryOutput)
	if err != nil || string(retained) != "retry bytes" {
		t.Fatalf("immutable retry copy %q %v", retained, err)
	}
	run("", 0, "agent", "withdraw", id, "--json")
	run("", 0, "agent", "cancel", id, "--json")
	waitState("idle")
	run("", 0, "agent", "stop", id, "--json")
	run("", 0, "agent", "archive", id, "--json")
	run("", 0, "archive", "search", "--project", "cli-test", "--limit", "10", "--json")
	run("", 0, "agent", "restore", id, "--json")
	run("", 0, "agent", "resume", id, "--json")
	waitState("idle")
	// A1: the existing bare form still resumes its one inactive match, then
	// explicit legacy resume retains its readable output and exit behavior.
	run("", 0, "agent", "stop", id, "--json")
	legacy := exec.Command(bin, "default@cli-test")
	var legacyOut, legacyErr bytes.Buffer
	legacy.Stdout = &legacyOut
	legacy.Stderr = &legacyErr
	if err := legacy.Run(); err != nil {
		t.Fatalf("legacy launch %v: %s %s", err, legacyOut.String(), legacyErr.String())
	}
	if !strings.Contains(legacyOut.String(), "resuming existing session") || !strings.Contains(legacyOut.String(), id) {
		t.Fatalf("bare form did not resume its existing agent: %s", legacyOut.String())
	}
	waitState("idle")
	run("", 0, "agent", "stop", id, "--json")
	legacy = exec.Command(bin, "resume", id)
	legacyOut.Reset()
	legacyErr.Reset()
	legacy.Stdout = &legacyOut
	legacy.Stderr = &legacyErr
	if err := legacy.Run(); err != nil || !strings.Contains(legacyOut.String(), "resumed ") || !strings.Contains(legacyOut.String(), id) {
		t.Fatalf("legacy resume %v: %s %s", err, legacyOut.String(), legacyErr.String())
	}
	waitState("idle")
	run("", 2, "project", "delete", "--project", "cli-test", "--json")
	run(`{"appearance_skin":"studio"}`, 0, "config", "edit", "--request-file", "-", "--json")
	run(`{"port":9999}`, 2, "config", "edit", "--request-file", "-", "--json")
	run("", 0, "role", "show", "--id", "default", "--json")
	run("", 0, "remote", "show", "--json")
	// A7 management journey: exercise real task and room routes over the
	// running server, including server-side ownership/consent guards.
	task := run(fmt.Sprintf(`{"project":"cli-test","display_name":"cli task","instruction":"do this","target_kind":"agent","target_agent_id":%q}`, id), 0, "task", "create", "--request-file", "-", "--json")
	var taskWire struct {
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal(task, &taskWire); err != nil || taskWire.TaskID == "" {
		t.Fatalf("task create %s %v", task, err)
	}
	run(fmt.Sprintf(`{"name":"cli-signal","project":"cli-test"}`), 0, "signal", "fire", "--request-file", "-", "--json")
	run("", 0, "task", "cancel", taskWire.TaskID, "--json")
	run("", 1, "task", "retry", taskWire.TaskID, "--json")
	run(`{"arms":[]}`, 1, "task", "rearm", taskWire.TaskID, "--request-file", "-", "--json")
	run("", 2, "task", "delete", taskWire.TaskID, "--json")
	run("", 0, "task", "delete", taskWire.TaskID, "--yes", "--json")
	second := run("", 0, "agent", "launch", "--role", "default", "--project", "cli-test", "--json")
	var secondWire struct {
		Agent struct {
			ID string `json:"agent_id"`
		} `json:"agent"`
	}
	if err := json.Unmarshal(second, &secondWire); err != nil || secondWire.Agent.ID == "" {
		t.Fatalf("second launch %s %v", second, err)
	}
	endSecond := time.Now().Add(10 * time.Second)
	for time.Now().Before(endSecond) {
		if status, err := st.ReadStatus(secondWire.Agent.ID); err == nil && status.State == "idle" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	room := run(fmt.Sprintf(`{"origin_project":"cli-test","goal":"compare","openings":false,"participants":[{"agent_id":%q,"limit":2},{"agent_id":%q,"limit":2}]}`, id, secondWire.Agent.ID), 0, "think-tank", "create", "--request-file", "-", "--json")
	var roomWire struct {
		RoomID string `json:"room_id"`
	}
	if err := json.Unmarshal(room, &roomWire); err != nil || roomWire.RoomID == "" {
		t.Fatalf("room create %s %v", room, err)
	}
	run(`{"body":"hello"}`, 0, "think-tank", "send", roomWire.RoomID, "--request-file", "-", "--json")
	run("", 0, "think-tank", "pause", roomWire.RoomID, "--json")
	run("", 0, "think-tank", "resume", roomWire.RoomID, "--json")
	run(`{"target":"turn"}`, 0, "think-tank", "retry", roomWire.RoomID, "--request-file", "-", "--json")
	run("", 0, "think-tank", "pause", roomWire.RoomID, "--json")
	run("", 0, "think-tank", "end", roomWire.RoomID, "--json")
	endRoom := time.Now().Add(10 * time.Second)
	for time.Now().Before(endRoom) {
		detail := run("", 0, "think-tank", "show", roomWire.RoomID, "--json")
		if bytes.Contains(detail, []byte(`"phase":"ended"`)) || bytes.Contains(detail, []byte(`"phase":"complete"`)) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	run("", 0, "agent", "stop", secondWire.Agent.ID, "--json")
	run("", 2, "think-tank", "delete", roomWire.RoomID, "--json")
	run("", 1, "think-tank", "delete", roomWire.RoomID, "--yes", "--json")
	// A script leaving transcript follow must close its observer, not stop work.
	follow := exec.Command(bin, "agent", "transcript", id, "--follow", "--json")
	var followed bytes.Buffer
	var followDiag bytes.Buffer
	follow.Stdout = &followed
	follow.Stderr = &followDiag
	if err := follow.Start(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if err := follow.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	if err := follow.Wait(); err != nil {
		t.Fatalf("follow interrupt: %v; stderr=%s; stdout=%s", err, followDiag.String(), followed.String())
	}
	if _, err := st.ReadRunning(id); err != nil {
		t.Fatalf("follow changed running agent: %v", err)
	}
	pipeCtx, pipeCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pipeCancel()
	broken := exec.CommandContext(pipeCtx, bin, "agent", "transcript", id, "--follow", "--json")
	pipe, err := broken.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var brokenDiag bytes.Buffer
	broken.Stderr = &brokenDiag
	if err := broken.Start(); err != nil {
		t.Fatal(err)
	}
	_ = pipe.Close()
	if err := broken.Wait(); err != nil {
		t.Fatalf("follow broken pipe: %v; %s", err, brokenDiag.String())
	}
	if _, err := st.ReadRunning(id); err != nil {
		t.Fatalf("broken output pipe stopped agent: %v", err)
	}
	// A7 pipeline journey: save a template, start a durable run, page and inspect
	// it, reject stale controls, then stop, inspect cleanup refusal, and delete.
	templateBody := `{"id":"cli-pipeline","template":{"version":2,"title":"CLI stage","orchestrator_role":"default","inputs":[],"stages":[{"id":"work","title":"Work","objective":"Complete the task.","coordination":"standing","inputs":[],"outputs":[]}]}}`
	run(templateBody, 0, "pipeline", "template", "create", "--request-file", "-", "--json")
	started := run("", 0, "pipeline", "run", "start", "--template", "cli-pipeline", "--project", "cli-test", "--goal", "pipeline process journey", "--backend", "codex", "--model", "gpt-5.6-sol", "--ack-shared-workspace", "--json")
	var pipelineWire struct {
		Run struct {
			Run struct {
				RunID    string `json:"run_id"`
				Revision int64  `json:"revision"`
			} `json:"run"`
		} `json:"run"`
	}
	if err := json.Unmarshal(started, &pipelineWire); err != nil || pipelineWire.Run.Run.RunID == "" || pipelineWire.Run.Run.Revision <= 0 {
		t.Fatalf("pipeline start %s err=%v", started, err)
	}
	pipelineID := pipelineWire.Run.Run.RunID
	run("", 0, "pipeline", "run", "list", "--limit", "10", "--offset", "0", "--json")
	detail := run("", 0, "pipeline", "run", "show", pipelineID, "--json")
	var detailWire struct {
		Run struct {
			RunID    string `json:"run_id"`
			Revision int64  `json:"revision"`
		} `json:"run"`
	}
	if err := json.Unmarshal(detail, &detailWire); err != nil || detailWire.Run.RunID != pipelineID || detailWire.Run.Revision <= 0 {
		t.Fatalf("pipeline detail %s err=%v", detail, err)
	}
	stale := detailWire.Run.Revision + 1
	run(fmt.Sprintf(`{"revision":%d,"input":"continue"}`, stale), 1, "pipeline", "run", "continue", pipelineID, "--request-file", "-", "--json")
	run(fmt.Sprintf(`{"revision":%d}`, stale), 1, "pipeline", "run", "retry", pipelineID, "--request-file", "-", "--json")
	run(fmt.Sprintf(`{"revision":%d,"orchestrator":{"backend":"codex","model":"gpt-5.6-sol"}}`, stale), 1, "pipeline", "run", "replace", pipelineID, "--request-file", "-", "--json")
	run(fmt.Sprintf(`{"revision":%d}`, stale), 1, "pipeline", "run", "repair-cleanup", pipelineID, "--request-file", "-", "--json")
	run("", 1, "pipeline", "run", "stop", pipelineID, "--revision", fmt.Sprint(stale), "--json")
	run("", 0, "pipeline", "run", "stop", pipelineID, "--revision", fmt.Sprint(detailWire.Run.Revision), "--json")
	waitPipelineState := func(want string) int64 {
		t.Helper()
		end := time.Now().Add(10 * time.Second)
		for time.Now().Before(end) {
			listed := run("", 0, "pipeline", "run", "list", "--limit", "10", "--offset", "0", "--json")
			var page []struct {
				RunID    string `json:"run_id"`
				State    string `json:"state"`
				Revision int64  `json:"revision"`
			}
			if json.Unmarshal(listed, &page) == nil {
				for _, item := range page {
					if item.RunID == pipelineID && item.State == want {
						return item.Revision
					}
				}
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("pipeline run did not reach %s", want)
		return 0
	}
	finalRevision := waitPipelineState("stopped")
	run(fmt.Sprintf(`{"revision":%d}`, finalRevision), 1, "pipeline", "run", "repair-cleanup", pipelineID, "--request-file", "-", "--json")
	run("", 0, "pipeline", "run", "delete", pipelineID, "--yes", "--json")
	run("", 0, "pipeline", "template", "delete", "cli-pipeline", "--yes", "--json")
	run("", 0, "agent", "stop", id, "--json")
	// A2: a native child's pending tool call is an exact permission target;
	// approving/denying it must release that child and stale decisions refuse.
	t.Setenv("FAKEACP_SCENARIO", "subagent_flow")
	t.Setenv("FAKEACP_CHILD_PERMISSION", "1")
	t.Setenv("FAKEACP_CAPS", "1")
	for _, decision := range []string{"approve", "deny"} {
		created := run("", 0, "agent", "launch", "--role", "default", "--project", "cli-test", "--backend", "codex", "--model", "gpt-5.6-sol", "--json")
		var childAgent struct {
			Agent struct {
				ID string `json:"agent_id"`
			} `json:"agent"`
		}
		if json.Unmarshal(created, &childAgent) != nil || childAgent.Agent.ID == "" {
			t.Fatalf("child launch %s", created)
		}
		childID := childAgent.Agent.ID
		end := time.Now().Add(10 * time.Second)
		for time.Now().Before(end) {
			status, err := st.ReadStatus(childID)
			if err == nil && status.State == "idle" {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		run("", 0, "agent", "send", childID, "--text", "exercise child permission", "--json")
		toolID := ""
		activity := ""
		end = time.Now().Add(10 * time.Second)
		for time.Now().Before(end) {
			status := run("", 0, "agent", "status", childID, "--json")
			var pending struct {
				Permissions struct {
					Pending []struct {
						Activity string `json:"activity_id"`
						Data     struct {
							ID string `json:"tool_call_id"`
						} `json:"data"`
					} `json:"pending"`
				} `json:"permissions"`
			}
			if json.Unmarshal(status, &pending) != nil {
				t.Fatalf("status %s", status)
			}
			if len(pending.Permissions.Pending) > 0 {
				toolID = pending.Permissions.Pending[0].Data.ID
				activity = pending.Permissions.Pending[0].Activity
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if toolID == "" || activity == "" {
			t.Fatalf("native child permission absent: id=%q activity=%q", toolID, activity)
		}
		run("", 1, "agent", decision, childID, "--tool-call-id", "stale-call", "--json")
		run("", 1, "agent", decision, childID, "--tool-call-id", "tc_1", "--json")
		run("", 0, "agent", decision, childID, "--tool-call-id", toolID, "--json")
		run("", 1, "agent", decision, childID, "--tool-call-id", toolID, "--json")
		end = time.Now().Add(10 * time.Second)
		settled := false
		for time.Now().Before(end) {
			status, err := st.ReadStatus(childID)
			if err == nil && status.State == "idle" {
				settled = true
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
		if !settled {
			t.Fatal("permission decision did not release native child turn")
		}
		run("", 0, "agent", "stop", childID, "--json")
	}
	// A9: real administrative reads/writes preserve ETags, redaction and preview
	// consent; these fixtures never enable a tailnet or call a real provider.
	run(`{"role":"cli-persona","title":"CLI persona","system_prompt":""}`, 0, "role", "create", "--request-file", "-", "--json")
	run(`{"title":"Edited persona","system_prompt":"Be precise"}`, 0, "role", "edit", "--id", "cli-persona", "--request-file", "-", "--json")
	// Role edit is a complete replacement: explicit false settings survive when
	// resubmitted, and omitted fields reset.
	var roleOut struct {
		Title           string `json:"title"`
		SystemPrompt    string `json:"system_prompt"`
		ChuckAware      *bool  `json:"chuck_aware"`
		SkipPermissions *bool  `json:"skip_permissions"`
	}
	readRole := func() {
		t.Helper()
		roleOut = struct {
			Title           string `json:"title"`
			SystemPrompt    string `json:"system_prompt"`
			ChuckAware      *bool  `json:"chuck_aware"`
			SkipPermissions *bool  `json:"skip_permissions"`
		}{}
		if err := json.Unmarshal(run("", 0, "role", "show", "--id", "cli-persona", "--json"), &roleOut); err != nil {
			t.Fatal(err)
		}
	}
	run(`{"title":"Full","system_prompt":"Keep me","chuck_aware":false,"skip_permissions":false}`, 0, "role", "edit", "--id", "cli-persona", "--request-file", "-", "--json")
	readRole()
	if roleOut.Title != "Full" || roleOut.SystemPrompt != "Keep me" || roleOut.ChuckAware == nil || *roleOut.ChuckAware || roleOut.SkipPermissions == nil || *roleOut.SkipPermissions {
		t.Fatalf("full replacement %+v", roleOut)
	}
	run(`{"title":"Full v2","system_prompt":"Keep me","chuck_aware":false,"skip_permissions":false}`, 0, "role", "edit", "--id", "cli-persona", "--request-file", "-", "--json")
	readRole()
	if roleOut.Title != "Full v2" || roleOut.SystemPrompt != "Keep me" || roleOut.ChuckAware == nil || *roleOut.ChuckAware || roleOut.SkipPermissions == nil || *roleOut.SkipPermissions {
		t.Fatalf("resubmitted explicit false %+v", roleOut)
	}
	run(`{"title":"Title only"}`, 0, "role", "edit", "--id", "cli-persona", "--request-file", "-", "--json")
	readRole()
	if roleOut.Title != "Title only" || roleOut.SystemPrompt != "" || roleOut.ChuckAware != nil || roleOut.SkipPermissions != nil {
		t.Fatalf("omitted fields were not reset %+v", roleOut)
	}
	run("", 2, "role", "delete", "--id", "cli-persona", "--json")
	run("", 0, "role", "delete", "--id", "cli-persona", "--yes", "--json")
	etagPath := filepath.Join(home, "catalog.etag")
	catalog := run("", 0, "backend", "list", "--etag-output", etagPath, "--json")
	etag, err := os.ReadFile(etagPath)
	if err != nil || len(etag) == 0 {
		t.Fatalf("catalog ETag %q %v", etag, err)
	}
	run(string(catalog), 1, "backend", "replace", "--etag", `"stale"`, "--request-file", "-", "--json")
	run(string(catalog), 0, "backend", "replace", "--etag", strings.TrimSpace(string(etag)), "--request-file", "-", "--json")
	root := filepath.Join(home, "native-claude")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "settings.json"), []byte(`{"model":"user-model","env":{"ANTHROPIC_API_KEY":"fixture-secret"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	preview := run(fmt.Sprintf(`{"provider":"claude-code","root":%q,"mode":"linked","claims":["launch_defaults"],"project":"cli-test"}`, root), 0, "config-source", "preview", "--request-file", "-", "--json")
	if bytes.Contains(preview, []byte("fixture-secret")) {
		t.Fatal("federation preview disclosed source secret")
	}
	var consent struct {
		Token string `json:"preview_token"`
	}
	if err := json.Unmarshal(preview, &consent); err != nil || consent.Token == "" {
		t.Fatalf("preview consent %s %v", preview, err)
	}
	bind := fmt.Sprintf(`{"preview_token":%q,"overrides":{}}`, consent.Token)
	run(bind, 2, "config-source", "bind", "--backend-id", "claude", "--request-file", "-", "--json")
	run(`{"preview_token":"invalid","overrides":{}}`, 1, "config-source", "bind", "--backend-id", "claude", "--yes", "--request-file", "-", "--json")
	bound := run(bind, 0, "config-source", "bind", "--backend-id", "claude", "--yes", "--request-file", "-", "--json")
	if bytes.Contains(bound, []byte("fixture-secret")) {
		t.Fatal("federation bind disclosed source secret")
	}
	run("", 0, "config-source", "list", "--project", "cli-test", "--json")
	run("", 0, "config-source", "refresh", "--backend-id", "claude", "--json")
	run("", 2, "config-source", "detach", "--backend-id", "claude", "--json")
	run("", 0, "config-source", "detach", "--backend-id", "claude", "--yes", "--json")
	run("", 0, "remote", "configure", "--enabled=false", "--keep-awake=false", "--json")
	run("", 0, "remote", "devices", "--json")
}
