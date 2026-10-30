package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/messaging"
	"github.com/AsaphNoam/Chuck/internal/pipeline"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// workspaceTestServer is a fake-provider server whose "repo" project is a real
// Git repository, with a one-stage template that has a dedicated coordinator.
func workspaceTestServer(t *testing.T) (*Server, string, *httptest.Server) {
	t.Helper()
	srv, ts := wakeTestServer(t)
	// These fake owners never report; the stall pause would close their stage.
	srv.ownerStallNudgeOff = true
	repo := newWorktreeTestRepo(t)
	if err := srv.configStore.WriteProject("repo", config.Project{Title: "Repo", Cwd: repo, AddDirs: []string{}}); err != nil {
		t.Fatal(err)
	}
	template := pipeline.Template{
		Version: 2, Title: "Flow", OrchestratorRole: "impl", Inputs: []pipeline.ValueDecl{},
		Stages: []pipeline.Stage{{
			ID: "build", Title: "Build", Objective: "Build it", Coordination: "dedicated", DedicatedRole: "impl",
			Inputs: []pipeline.StageInput{}, Outputs: []pipeline.StageOutput{},
		}},
	}
	if record, err := srv.pipelineTemplates.Create("flow", template); err != nil || !record.Valid {
		t.Fatalf("create template = %+v err=%v", record, err)
	}
	return srv, repo, ts
}

func startWorkspaceRun(t *testing.T, srv *Server, requestID, name string, workspace *pipeline.WorkspaceSelection) pipeline.RunDetail {
	t.Helper()
	runtimeAssignment := pipeline.RuntimeAssignment{Backend: "claude", Model: "sonnet"}
	detail, _, err := srv.pipelineMgr.Start(context.Background(), pipeline.StartRequest{
		RequestID: requestID, TemplateID: "flow", DisplayName: name, Project: "repo", Goal: "Do it",
		Inputs: map[string]string{}, Orchestrator: runtimeAssignment,
		DedicatedAssignments: map[string]pipeline.RuntimeAssignment{"build": runtimeAssignment},
		Workspace:            workspace,
	})
	if err != nil {
		t.Fatalf("Start %s: %v", requestID, err)
	}
	return detail
}

// runningStageAgents dispatches until the run's standing owner and dedicated
// coordinator are both running, returning their agent ids.
func runningStageAgents(t *testing.T, srv *Server, runID string) (string, string) {
	t.Helper()
	stages, err := srv.stateStore.ListPipelineStageTasks(runID)
	if err != nil || len(stages) != 1 {
		t.Fatalf("stage tasks = %+v err=%v", stages, err)
	}
	standing := dispatchUntilTaskState(t, srv, stages[0].TaskID, state.TaskRunning)
	coordinator := dispatchUntilTaskState(t, srv, stages[0].CoordinatorTaskID, state.TaskRunning)
	return standing.AssignedAgentID, coordinator.AssignedAgentID
}

// delegate creates launch work as a running agent and returns the launched
// descendant once its task runs.
func delegate(t *testing.T, srv *Server, creator, group string) string {
	t.Helper()
	task, err := srv.CreateAgentTask(messaging.AgentTaskRequest{
		CreatorAgentID: creator, CreatorGeneration: srv.registry.Generation(creator),
		Project: "repo", DisplayName: "child of " + creator, Instruction: "help", Role: "impl", Group: group,
	})
	if err != nil {
		t.Fatalf("CreateAgentTask: %v", err)
	}
	return dispatchUntilTaskState(t, srv, task.TaskID, state.TaskRunning).AssignedAgentID
}

func assertPlacement(t *testing.T, srv *Server, agentID, wantCwd, wantWorkspace, wantGroup string) {
	t.Helper()
	snap, err := srv.stateStore.ReadSession(agentID)
	if err != nil {
		t.Fatalf("ReadSession %s: %v", agentID, err)
	}
	agent, err := srv.stateStore.ReadAgent(agentID)
	if err != nil {
		t.Fatalf("ReadAgent %s: %v", agentID, err)
	}
	if !sameCheckoutPath(snap.Cwd, wantCwd) || snap.WorkspaceID != wantWorkspace || agent.Group != wantGroup {
		t.Fatalf("agent %s cwd=%q workspace=%q group=%q, want cwd=%q workspace=%q group=%q",
			agentID, snap.Cwd, snap.WorkspaceID, agent.Group, wantCwd, wantWorkspace, wantGroup)
	}
}

// FS-14.A57/A58, FS-19.A8 — two isolated runs and one project-mode run in one
// project: every standing owner, dedicated coordinator and two generations of
// delegated workers start in their run's checkout and group, whatever group the
// delegating request named. An ordinary agent keeps the project checkout.
func TestPipelineRunWorkspacePlacesEveryNewAgent(t *testing.T) {
	srv, repo, ts := workspaceTestServer(t)
	runA := startWorkspaceRun(t, srv, "run-a", "Run A", &pipeline.WorkspaceSelection{Mode: "new_worktree", Branch: "run-a"})
	runB := startWorkspaceRun(t, srv, "run-b", "Run B", &pipeline.WorkspaceSelection{Mode: "new_worktree", Branch: "run-b"})
	runC := startWorkspaceRun(t, srv, "run-c", "Run C", nil)
	for _, run := range []pipeline.RunDetail{runA, runB, runC} {
		if run.Workspace == nil || run.Workspace.WorkspaceID == "" {
			t.Fatalf("run %s has no workspace: %+v", run.Run.RunID, run.Workspace)
		}
	}
	if runA.Workspace.State != state.PipelineWorkspacePreparing || runA.Workspace.Owned || runC.Workspace.Mode != "project" {
		t.Fatalf("intents = A %+v C %+v", runA.Workspace, runC.Workspace)
	}

	standingA, coordinatorA := runningStageAgents(t, srv, runA.Run.RunID)
	wsA, err := srv.stateStore.ReadPipelineWorkspace(runA.Workspace.WorkspaceID)
	if err != nil || wsA.State != state.PipelineWorkspaceReady || !wsA.Owned {
		t.Fatalf("workspace A = %+v err=%v", wsA, err)
	}
	if branch := gitOutput(t, wsA.CheckoutPath, "rev-parse", "--abbrev-ref", "HEAD"); branch != "run-a" {
		t.Fatalf("checkout A branch = %q", branch)
	}
	for _, id := range []string{standingA, coordinatorA} {
		assertPlacement(t, srv, id, wsA.CheckoutPath, wsA.WorkspaceID, "Run A")
	}
	child := delegate(t, srv, standingA, "Somewhere else")
	assertPlacement(t, srv, child, wsA.CheckoutPath, wsA.WorkspaceID, "Run A")
	grandchild := delegate(t, srv, child, "")
	assertPlacement(t, srv, grandchild, wsA.CheckoutPath, wsA.WorkspaceID, "Run A")

	standingB, _ := runningStageAgents(t, srv, runB.Run.RunID)
	wsB, _ := srv.stateStore.ReadPipelineWorkspace(runB.Workspace.WorkspaceID)
	assertPlacement(t, srv, standingB, wsB.CheckoutPath, wsB.WorkspaceID, "Run B")
	if sameCheckoutPath(wsA.CheckoutPath, wsB.CheckoutPath) {
		t.Fatal("isolated runs share a checkout")
	}

	standingC, coordinatorC := runningStageAgents(t, srv, runC.Run.RunID)
	for _, id := range []string{standingC, coordinatorC} {
		assertPlacement(t, srv, id, repo, runC.Workspace.WorkspaceID, "Run C")
	}

	// FS-19.A8: an ordinary agent in the same project keeps the project checkout.
	ordinary := launchAndWaitIdle(t, ts, "impl", "repo")
	assertPlacement(t, srv, ordinary, repo, "", "")
}

// FS-14.A60 (R95) — an isolated run refuses existing targets in a sibling run's
// checkout or the project checkout; a project-mode run accepts an agent in the
// project checkout. Admission repeats the check for a target changed later.
func TestPipelineRunWorkspaceRefusesTargetsInOtherCheckouts(t *testing.T) {
	srv, _, ts := workspaceTestServer(t)
	runA := startWorkspaceRun(t, srv, "run-a", "Run A", &pipeline.WorkspaceSelection{Mode: "new_worktree", Branch: "run-a"})
	runB := startWorkspaceRun(t, srv, "run-b", "Run B", &pipeline.WorkspaceSelection{Mode: "new_worktree", Branch: "run-b"})
	runC := startWorkspaceRun(t, srv, "run-c", "Run C", nil)
	standingA, coordinatorA := runningStageAgents(t, srv, runA.Run.RunID)
	standingB, _ := runningStageAgents(t, srv, runB.Run.RunID)
	standingC, _ := runningStageAgents(t, srv, runC.Run.RunID)
	ordinary := launchAndWaitIdle(t, ts, "impl", "repo")

	assign := func(creator, target string) (state.Task, error) {
		return srv.CreateAgentTask(messaging.AgentTaskRequest{
			CreatorAgentID: creator, CreatorGeneration: srv.registry.Generation(creator),
			Project: "repo", DisplayName: "assign", Instruction: "help", TargetAgentID: target,
		})
	}
	for _, target := range []string{standingB, ordinary} {
		_, err := assign(standingA, target)
		var toolErr *messaging.ToolError
		if !errors.As(err, &toolErr) || toolErr.Code != "target_ineligible" || !strings.Contains(toolErr.Message, "launch a new agent") {
			t.Fatalf("assign %s from run A = %v, want target_ineligible", target, err)
		}
	}
	if _, err := assign(standingA, coordinatorA); err != nil {
		t.Fatalf("same-checkout target refused: %v", err)
	}
	if _, err := assign(standingC, ordinary); err != nil {
		t.Fatalf("project-mode run refused a project-checkout target: %v", err)
	}

	// A target that left the run's checkout after creation is parked at admission.
	task, err := assign(standingA, coordinatorA)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := srv.stateStore.DB().Exec(`UPDATE sessions SET workspace_id = '' WHERE agent_id = ?`, coordinatorA); err != nil {
		t.Fatal(err)
	}
	if err := srv.stateStore.CheckPipelineTaskTarget(task.TaskID, coordinatorA); !errors.Is(err, state.ErrTaskTargetWorkspace) {
		t.Fatalf("admission recheck = %v, want ErrTaskTargetWorkspace", err)
	}
}

// FS-14.A57 — a branch that already exists refuses Start before anything is
// created; a run whose preparation later fails starts no agent and records why.
func TestPipelineRunWorkspacePreparationFailures(t *testing.T) {
	srv, repo, _ := workspaceTestServer(t)
	gitOutput(t, repo, "branch", "taken")
	_, _, err := srv.pipelineMgr.Start(context.Background(), pipeline.StartRequest{
		RequestID: "taken", TemplateID: "flow", Project: "repo", Goal: "Do it", Inputs: map[string]string{},
		Orchestrator:         pipeline.RuntimeAssignment{Backend: "claude", Model: "sonnet"},
		DedicatedAssignments: map[string]pipeline.RuntimeAssignment{"build": {Backend: "claude", Model: "sonnet"}},
		Workspace:            &pipeline.WorkspaceSelection{Mode: "new_worktree", Branch: "taken"},
	})
	var controlErr *pipeline.ControlError
	if !errors.As(err, &controlErr) || !strings.Contains(err.Error()+diagnosticText(controlErr), "already exists") {
		t.Fatalf("colliding branch Start = %v", err)
	}
	if runs, _ := srv.stateStore.ListPipelineRuns(10, 0); len(runs) != 0 {
		t.Fatalf("refused Start created runs: %+v", runs)
	}

	// The branch appears between Start and preparation: no agent starts.
	run := startWorkspaceRun(t, srv, "raced", "Raced", &pipeline.WorkspaceSelection{Mode: "new_worktree", Branch: "raced"})
	gitOutput(t, repo, "branch", "raced")
	if _, _, _, ae := srv.ensureRunWorkspace(context.Background(), run.Workspace.WorkspaceID); ae == nil {
		t.Fatal("preparation adopted a branch Chuck did not create")
	}
	ws, _ := srv.stateStore.ReadPipelineWorkspace(run.Workspace.WorkspaceID)
	if ws.State != state.PipelineWorkspaceError || ws.Owned || !strings.Contains(ws.Error, "already exists") {
		t.Fatalf("failed preparation = %+v", ws)
	}
	if _, err := os.Lstat(ws.CheckoutPath); !os.IsNotExist(err) {
		t.Fatalf("failed preparation left %s: %v", ws.CheckoutPath, err)
	}
	if head := gitOutput(t, repo, "rev-parse", "raced"); head == "" {
		t.Fatal("the pre-existing branch was deleted")
	}
}

// FS-14.A60 (R96), TS-12.R14 — an owned checkout lost out of band is recreated
// from its branch; a replaced directory or missing branch fails without repair;
// an explicit-removal tombstone refuses recreation.
func TestPipelineRunWorkspaceRecoveryAndTombstone(t *testing.T) {
	srv, repo, _ := workspaceTestServer(t)
	run := startWorkspaceRun(t, srv, "run", "Run", &pipeline.WorkspaceSelection{Mode: "new_worktree", Branch: "run"})
	ctx := context.Background()
	ws, _, _, ae := srv.ensureRunWorkspace(ctx, run.Workspace.WorkspaceID)
	if ae != nil {
		t.Fatalf("prepare: %s", ae.Message)
	}
	if err := os.RemoveAll(ws.CheckoutPath); err != nil {
		t.Fatal(err)
	}
	if _, recreated, _, ae := srv.ensureRunWorkspace(ctx, ws.WorkspaceID); ae != nil || !recreated {
		t.Fatalf("out-of-band loss: recreated=%v err=%v", recreated, ae)
	}
	if branch := gitOutput(t, ws.CheckoutPath, "rev-parse", "--abbrev-ref", "HEAD"); branch != "run" {
		t.Fatalf("recreated branch = %q", branch)
	}

	// TS-12.R14: right repository and branch are not enough. A checkout moved,
	// repaired at the new path and moved back is registered elsewhere, so Git
	// may prune it; readiness refuses it until the registration is repaired.
	moved := ws.CheckoutPath + "-moved"
	if err := os.Rename(ws.CheckoutPath, moved); err != nil {
		t.Fatal(err)
	}
	gitOutput(t, repo, "worktree", "repair", moved)
	if err := os.Rename(moved, ws.CheckoutPath); err != nil {
		t.Fatal(err)
	}
	if _, _, _, ae := srv.ensureRunWorkspace(ctx, ws.WorkspaceID); ae == nil || !strings.Contains(ae.Message, "another path") {
		t.Fatalf("stale registration accepted: %v", ae)
	}
	gitOutput(t, repo, "worktree", "repair", ws.CheckoutPath)
	if _, _, _, ae := srv.ensureRunWorkspace(ctx, ws.WorkspaceID); ae != nil {
		t.Fatalf("repaired registration: %s", ae.Message)
	}

	// A replacement directory is neither repaired nor removed.
	gitOutput(t, repo, "worktree", "remove", "--force", ws.CheckoutPath)
	if err := os.MkdirAll(ws.CheckoutPath, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(ws.CheckoutPath, "mine.txt")
	if err := os.WriteFile(marker, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, ae := srv.ensureRunWorkspace(ctx, ws.WorkspaceID); ae == nil {
		t.Fatal("a replacement directory was accepted")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("replacement contents changed: %v", err)
	}
	if err := os.RemoveAll(ws.CheckoutPath); err != nil {
		t.Fatal(err)
	}
	gitOutput(t, repo, "worktree", "prune")
	gitOutput(t, repo, "branch", "-D", "run")
	if _, _, _, ae := srv.ensureRunWorkspace(ctx, ws.WorkspaceID); ae == nil || !strings.Contains(ae.Message, "no longer exists") {
		t.Fatalf("missing branch = %v", ae)
	}

	if _, err := srv.stateStore.UpdatePipelineWorkspace(ws.WorkspaceID, state.PipelineWorkspaceUpdate{State: state.PipelineWorkspaceRemoved}); err != nil {
		t.Fatal(err)
	}
	gitOutput(t, repo, "branch", "run")
	if _, _, _, ae := srv.ensureRunWorkspace(ctx, ws.WorkspaceID); ae == nil || !strings.Contains(ae.Message, "will not recreate") {
		t.Fatalf("tombstone = %v", ae)
	}
	if _, err := os.Lstat(ws.CheckoutPath); !os.IsNotExist(err) {
		t.Fatal("a removed workspace was recreated")
	}
}

// FS-14.A57 — resume keeps the frozen checkout and workspace identity.
func TestPipelineRunWorkspaceSurvivesResume(t *testing.T) {
	srv, _, _ := workspaceTestServer(t)
	run := startWorkspaceRun(t, srv, "run", "Run", &pipeline.WorkspaceSelection{Mode: "new_worktree", Branch: "run"})
	standing, _ := runningStageAgents(t, srv, run.Run.RunID)
	before, err := srv.stateStore.ReadSession(standing)
	if err != nil {
		t.Fatal(err)
	}
	handler := srv.routes()
	deadline := time.Now().Add(2 * time.Second)
	for srv.lifecycleInFlight(standing) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	// A later project edit cannot move the retained conversation.
	if err := srv.configStore.WriteProject("repo", config.Project{Title: "Repo", Cwd: t.TempDir(), AddDirs: []string{}}); err != nil {
		t.Fatal(err)
	}
	if rec := doRequest(t, handler, http.MethodPost, "/api/sessions/"+standing+"/stop", nil); rec.Code != http.StatusOK {
		t.Fatalf("stop = %d %s", rec.Code, rec.Body.String())
	}
	if rec := doRequest(t, handler, http.MethodPost, "/api/sessions/"+standing+"/resume", nil); rec.Code != http.StatusOK {
		t.Fatalf("resume = %d %s", rec.Code, rec.Body.String())
	}
	after, err := srv.stateStore.ReadSession(standing)
	if err != nil {
		t.Fatal(err)
	}
	if after.Cwd != before.Cwd || after.WorkspaceID != before.WorkspaceID || after.WorkspaceID == "" {
		t.Fatalf("resume moved the agent: before %q/%q after %q/%q", before.Cwd, before.WorkspaceID, after.Cwd, after.WorkspaceID)
	}
}

// TS-09.R65 — effective agent metadata projects the frozen session binding
// through the retained workspace record: live, after explicit removal and
// after the run itself is deleted.
func TestPipelineRunWorkspaceAgentMetadata(t *testing.T) {
	srv, _, _ := workspaceTestServer(t)
	handler := srv.routes()
	run := startWorkspaceRun(t, srv, "meta", "Meta", &pipeline.WorkspaceSelection{Mode: "new_worktree", Branch: "meta"})
	standing, coordinator := runningStageAgents(t, srv, run.Run.RunID)
	wsID := run.Workspace.WorkspaceID
	metadata := func(wantState string) {
		t.Helper()
		rec := doRequest(t, handler, http.MethodGet, "/api/sessions/"+standing, nil)
		var resp struct {
			Workspace *struct {
				WorkspaceID string `json:"workspace_id"`
				Mode        string `json:"mode"`
				Cwd         string `json:"cwd"`
				Branch      string `json:"branch"`
				State       string `json:"state"`
			} `json:"workspace"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || rec.Code != http.StatusOK || resp.Workspace == nil {
			t.Fatalf("session = %d %s", rec.Code, rec.Body.String())
		}
		if w := resp.Workspace; w.WorkspaceID != wsID || w.Mode != state.PipelineWorkspaceModeNewWorktree || w.Branch != "meta" || w.Cwd == "" || w.State != wantState {
			t.Fatalf("session workspace = %+v, want state %q", *w, wantState)
		}
		update, err := srv.stateMgr.Touch(standing)
		if err != nil || update.Workspace == nil || update.Workspace.State != wantState {
			t.Fatalf("agent state workspace = %+v err=%v", update.Workspace, err)
		}
	}
	metadata(state.PipelineWorkspaceReady)

	endRun(t, srv, run.Run.RunID)
	for _, id := range []string{standing, coordinator} {
		deadline := time.Now().Add(2 * time.Second)
		for srv.lifecycleInFlight(id) && time.Now().Before(deadline) {
			time.Sleep(5 * time.Millisecond)
		}
		if rec := doRequest(t, handler, http.MethodPost, "/api/sessions/"+id+"/stop", nil); rec.Code != http.StatusOK {
			t.Fatalf("stop = %d %s", rec.Code, rec.Body.String())
		}
	}
	ws := workspaceDetail(t, srv, wsID)
	if !ws.Removable {
		t.Fatalf("workspace not removable: %+v", ws)
	}
	if rec := doRequest(t, handler, http.MethodPost, "/api/pipeline-workspaces/"+wsID+"/remove", removeWorkspaceRequest{Revision: ws.Revision, DirtyKnown: true}); rec.Code != http.StatusOK {
		t.Fatalf("remove = %d %s", rec.Code, rec.Body.String())
	}
	metadata(state.PipelineWorkspaceRemoved)

	if rec := doRequest(t, handler, http.MethodDelete, "/api/pipeline-runs/"+run.Run.RunID, nil); rec.Code >= 300 {
		t.Fatalf("delete run = %d %s", rec.Code, rec.Body.String())
	}
	metadata(state.PipelineWorkspaceRemoved)
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func diagnosticText(err *pipeline.ControlError) string {
	if err == nil {
		return ""
	}
	var b strings.Builder
	for _, d := range err.Diagnostics {
		b.WriteString(d.Message)
	}
	return b.String()
}

func workspaceDetail(t *testing.T, srv *Server, id string) pipelineWorkspaceDetail {
	t.Helper()
	rec := doRequest(t, srv.routes(), http.MethodGet, "/api/pipeline-workspaces/"+id, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail = %d %s", rec.Code, rec.Body.String())
	}
	var detail pipelineWorkspaceDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	return detail
}

func endRun(t *testing.T, srv *Server, runID string) {
	t.Helper()
	if _, err := srv.stateStore.DB().Exec(`UPDATE pipeline_runs SET state = 'stopped' WHERE run_id = ?`, runID); err != nil {
		t.Fatal(err)
	}
}

// FS-14.A59/A60 (R94, R96), TS-12.R15/R16 — removal is refused while the run
// is live or an agent still runs there; it discloses dirty state, refuses a
// stale or understated confirmation, keeps the branch, is idempotent, and
// never applies to a project-mode checkout. A deleted run's checkout stays
// listed and removable.
func TestPipelineRunWorkspaceExplicitRemoval(t *testing.T) {
	srv, repo, _ := workspaceTestServer(t)
	ctx := context.Background()
	handler := srv.routes()
	remove := func(id string, body removeWorkspaceRequest) *httptest.ResponseRecorder {
		return doRequest(t, handler, http.MethodPost, "/api/pipeline-workspaces/"+id+"/remove", body)
	}

	busy := startWorkspaceRun(t, srv, "busy", "Busy", &pipeline.WorkspaceSelection{Mode: "new_worktree", Branch: "busy"})
	runningStageAgents(t, srv, busy.Run.RunID)
	ws := workspaceDetail(t, srv, busy.Workspace.WorkspaceID)
	if ws.Removable || !strings.Contains(ws.Blocked, "not ended") {
		t.Fatalf("live run detail = %+v", ws)
	}
	endRun(t, srv, busy.Run.RunID)
	if rec := remove(ws.WorkspaceID, removeWorkspaceRequest{Revision: ws.Revision, DirtyKnown: true}); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "still running") {
		t.Fatalf("remove with running agents = %d %s", rec.Code, rec.Body.String())
	}

	run := startWorkspaceRun(t, srv, "run", "Run", &pipeline.WorkspaceSelection{Mode: "new_worktree", Branch: "run"})
	prepared, _, _, ae := srv.ensureRunWorkspace(ctx, run.Workspace.WorkspaceID)
	if ae != nil {
		t.Fatal(ae.Message)
	}
	endRun(t, srv, run.Run.RunID)
	if err := os.WriteFile(filepath.Join(prepared.CheckoutPath, "wip.txt"), []byte("draft"), 0o600); err != nil {
		t.Fatal(err)
	}
	detail := workspaceDetail(t, srv, prepared.WorkspaceID)
	if !detail.Removable || !detail.DirtyKnown || !detail.Dirty {
		t.Fatalf("dirty detail = %+v", detail)
	}
	if rec := remove(detail.WorkspaceID, removeWorkspaceRequest{Revision: detail.Revision, DirtyKnown: true, Dirty: false}); rec.Code != http.StatusConflict {
		t.Fatalf("understated confirmation = %d %s", rec.Code, rec.Body.String())
	}
	if rec := remove(detail.WorkspaceID, removeWorkspaceRequest{Revision: detail.Revision - 1, DirtyKnown: true, Dirty: true}); rec.Code != http.StatusConflict {
		t.Fatalf("stale confirmation = %d %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(prepared.CheckoutPath); err != nil {
		t.Fatalf("refused removal touched the checkout: %v", err)
	}
	rec := remove(detail.WorkspaceID, removeWorkspaceRequest{Revision: detail.Revision, DirtyKnown: true, Dirty: true})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"state":"removed"`) {
		t.Fatalf("remove = %d %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Lstat(prepared.CheckoutPath); !os.IsNotExist(err) {
		t.Fatalf("checkout survived removal: %v", err)
	}
	if head := gitOutput(t, repo, "rev-parse", "--verify", "run"); head == "" {
		t.Fatal("removal deleted the branch")
	}
	if rec := remove(detail.WorkspaceID, removeWorkspaceRequest{}); rec.Code != http.StatusOK {
		t.Fatalf("repeated removal = %d %s", rec.Code, rec.Body.String())
	}

	project := startWorkspaceRun(t, srv, "project", "Project", nil)
	endRun(t, srv, project.Run.RunID)
	pd := workspaceDetail(t, srv, project.Workspace.WorkspaceID)
	if pd.Removable {
		t.Fatalf("project-mode checkout offered for removal: %+v", pd)
	}
	if rec := remove(pd.WorkspaceID, removeWorkspaceRequest{Revision: pd.Revision, DirtyKnown: true}); rec.Code == http.StatusOK {
		t.Fatalf("project checkout removal accepted: %s", rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(repo, "README.md")); err != nil {
		t.Fatalf("project checkout damaged: %v", err)
	}

	deleted := startWorkspaceRun(t, srv, "deleted", "Deleted run", &pipeline.WorkspaceSelection{Mode: "new_worktree", Branch: "deleted"})
	if _, _, _, ae := srv.ensureRunWorkspace(ctx, deleted.Workspace.WorkspaceID); ae != nil {
		t.Fatal(ae.Message)
	}
	endRun(t, srv, deleted.Run.RunID)
	if rec := doRequest(t, handler, http.MethodDelete, "/api/pipeline-runs/"+deleted.Run.RunID, nil); rec.Code >= 300 {
		t.Fatalf("delete run = %d %s", rec.Code, rec.Body.String())
	}
	rec = doRequest(t, handler, http.MethodGet, "/api/pipeline-workspaces?project=repo", nil)
	var list pipelineWorkspaceList
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("list = %d %s", rec.Code, rec.Body.String())
	}
	found := false
	for _, item := range list.Workspaces {
		found = found || (item.WorkspaceID == deleted.Workspace.WorkspaceID && item.RunName == "Deleted run")
	}
	if !found || len(list.Workspaces) != 4 {
		t.Fatalf("retained list = %+v", list.Workspaces)
	}
	if retained := workspaceDetail(t, srv, deleted.Workspace.WorkspaceID); !retained.Removable || retained.Missing {
		t.Fatalf("retained checkout after run deletion = %+v", retained)
	}
}

// TS-12.R15 — every user of the checkout blocks removal, whatever its binding
// and however many retained conversations precede it: an ordinary agent whose
// project points at the retained checkout, a run conversation beyond any page
// of idle ones, and a claimed start racing the removal in either order.
func TestPipelineRunWorkspaceRemovalSeesEveryUser(t *testing.T) {
	srv, _, ts := workspaceTestServer(t)
	ctx := context.Background()
	ended := func(name string) state.PipelineWorkspace {
		t.Helper()
		run := startWorkspaceRun(t, srv, name, name, &pipeline.WorkspaceSelection{Mode: "new_worktree", Branch: name})
		ws, _, _, ae := srv.ensureRunWorkspace(ctx, run.Workspace.WorkspaceID)
		if ae != nil {
			t.Fatal(ae.Message)
		}
		endRun(t, srv, run.Run.RunID)
		return ws
	}
	assertRefused := func(ws state.PipelineWorkspace, why string) {
		t.Helper()
		if _, ae := srv.removeRunWorkspace(ctx, ws.WorkspaceID, removeWorkspaceRequest{Revision: ws.Revision, DirtyKnown: true}); ae == nil {
			t.Fatalf("removed a checkout in use: %s", why)
		}
		if _, err := os.Stat(ws.CheckoutPath); err != nil {
			t.Fatalf("refused removal touched the checkout (%s): %v", why, err)
		}
	}

	// Many idle run conversations precede the one still running.
	crowded := ended("crowded")
	for i := 0; i <= 256; i++ {
		id := fmt.Sprintf("a-%03d", i)
		if _, err := srv.stateStore.DB().Exec(`INSERT INTO agents(agent_id, name, role, project, backend, model, interface, created_at)
VALUES (?, '', 'impl', 'repo', 'claude', 'sonnet', 'chat', '2026-10-29T00:00:00Z')`, id); err != nil {
			t.Fatal(err)
		}
		if _, err := srv.stateStore.DB().Exec(`INSERT INTO sessions(agent_id, name, role, project, backend, model, interface, cwd, system_prompt, created_at, updated_at, workspace_id)
VALUES (?, '', 'impl', 'repo', 'claude', 'sonnet', 'chat', ?, '', '2026-10-29T00:00:00Z', '2026-10-29T00:00:00Z', ?)`, id, crowded.CheckoutPath, crowded.WorkspaceID); err != nil {
			t.Fatal(err)
		}
	}
	if err := srv.stateStore.WriteRunning(state.RunningEntry{AgentID: "a-256", PID: 1, SessionID: "s", Interface: "chat", StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	assertRefused(crowded, "257th conversation running")

	// The same path claimed by an in-flight start, either side first.
	racing := ended("racing")
	if !srv.claimLifecycle("starter") {
		t.Fatal("claim")
	}
	if _, ae := srv.resolveStartCheckout(ctx, "starter", "repo", "", racing.CheckoutPath); ae != nil {
		t.Fatal(ae.Message)
	}
	assertRefused(racing, "start entered first")
	srv.releaseLifecycle("starter")
	done, ae := srv.beginCheckoutRemoval(racing.CheckoutPath)
	if ae != nil {
		t.Fatal(ae.Message)
	}
	if !srv.claimLifecycle("late") {
		t.Fatal("claim")
	}
	if _, ae := srv.resolveStartCheckout(ctx, "late", "repo", "", racing.CheckoutPath); ae == nil || ae.Code != runtime.CodeConflict {
		t.Fatalf("start during removal = %v", ae)
	}
	srv.releaseLifecycle("late")
	done()

	// An ordinary agent launched there carries no workspace id.
	other := ended("other")
	if err := srv.configStore.WriteProject("repo", config.Project{Title: "Repo", Cwd: other.CheckoutPath, AddDirs: []string{}}); err != nil {
		t.Fatal(err)
	}
	ordinary := launchAndWaitIdle(t, ts, "impl", "repo")
	if snap, err := srv.stateStore.ReadSession(ordinary); err != nil || snap.WorkspaceID != "" {
		t.Fatalf("ordinary fixture = %+v %v", snap, err)
	}
	assertRefused(other, "ordinary binding running")
}
