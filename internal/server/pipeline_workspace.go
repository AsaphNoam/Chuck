package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/pipeline"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// Pipeline run workspaces (FS-14.R91–R96, TS-09.R61–R63, TS-12.R12–R15). Git
// work stays here, above internal/worktree; the pipeline controller only stores
// the plan this file returns.

// PlanWorkspace resolves a run's selection without creating anything. Project
// mode freezes the project's current checkout; new-worktree mode resolves the
// repository, base commit and branch the later preparation will use.
func (s *Server) PlanWorkspace(ctx context.Context, workspaceID, projectID string, selection pipeline.WorkspaceSelection) (state.PipelineWorkspace, error) {
	project, err := s.configStore.ReadProject(projectID)
	if err != nil {
		return state.PipelineWorkspace{}, fmt.Errorf("unknown project %q", projectID)
	}
	cwd, err := config.ExpandTilde(project.Cwd)
	if err != nil {
		return state.PipelineWorkspace{}, fmt.Errorf("bad project cwd: %v", err)
	}
	plan := state.PipelineWorkspace{Project: projectID, ProjectName: project.Title, Mode: selection.Mode}
	if selection.Mode != state.PipelineWorkspaceModeNewWorktree {
		plan.CheckoutPath, plan.State = cwd, state.PipelineWorkspaceReady
		if row, owned, err := s.readOwnedWorktree(projectID); err == nil && owned && sameCheckoutPath(row.CheckoutPath, cwd) {
			plan.Branch = row.Branch
		}
		return plan, nil
	}

	branch := strings.TrimSpace(selection.Branch)
	if ae := validateBranchName(branch); ae != nil {
		return state.PipelineWorkspace{}, errors.New(ae.Message)
	}
	git := s.worktreeGit()
	inside, err := git.IsInsideWorkTree(ctx, cwd)
	if err != nil {
		return state.PipelineWorkspace{}, errors.New(gitUnavailableMessage(err))
	}
	if !inside {
		return state.PipelineWorkspace{}, fmt.Errorf("project %q is not inside a Git working tree; use the project checkout", projectID)
	}
	repo, err := git.CommonDir(ctx, cwd)
	if err != nil {
		return state.PipelineWorkspace{}, fmt.Errorf("cannot read the repository anchor: %v", err)
	}
	base := strings.TrimSpace(selection.Base)
	if base == "" {
		resolved, ae := s.effectiveBase(ctx, project, cwd)
		if ae != nil {
			return state.PipelineWorkspace{}, errors.New(ae.Message)
		}
		base = resolved
	}
	commit, err := git.ResolveCommit(ctx, repo, base)
	if err != nil {
		return state.PipelineWorkspace{}, fmt.Errorf("base %q does not resolve to a commit in %s", base, repo)
	}
	if exists, err := git.BranchExists(ctx, repo, branch); err != nil {
		return state.PipelineWorkspace{}, fmt.Errorf("cannot check branch: %v", err)
	} else if exists {
		return state.PipelineWorkspace{}, fmt.Errorf("branch %q already exists; choose a new branch name", branch)
	}
	plan.RepoPath, plan.Base, plan.BaseCommit, plan.Branch = repo, base, commit, branch
	plan.SetupCommand, plan.State = project.SetupCommand, state.PipelineWorkspacePreparing
	if workspaceID != "" {
		if plan.CheckoutPath, err = s.configStore.PipelineWorktreePath(workspaceID); err != nil {
			return state.PipelineWorkspace{}, err
		}
	}
	return plan, nil
}

// targetWorkspaceMismatch is the bounded refusal for run work aimed at an agent
// in another checkout, with both recoveries (FS-14.R95, TS-09.R63).
const targetWorkspaceMismatch = "that agent works in a different checkout than this pipeline run; launch a new agent for this run by naming a role, or choose an agent in the run's checkout"

// runWorkspaceClaimKey shares the per-checkout recreation claim with project
// worktrees. Project ids are slugs, so the prefix cannot collide with one.
func runWorkspaceClaimKey(workspaceID string) string { return "pipeline-workspace:" + workspaceID }

// ensureRunWorkspace is the one readiness step for a run workspace (TS-12.R13/
// R14). Under the workspace claim it prepares a new checkout, resumes an
// interrupted preparation, verifies an existing one or recreates an owned
// checkout lost out of band. A removed workspace is never recreated.
func (s *Server) ensureRunWorkspace(ctx context.Context, workspaceID string) (w state.PipelineWorkspace, recreated bool, warning string, ae *runtime.APIError) {
	_, release, err := s.claimWorktreeRecreate(ctx, runWorkspaceClaimKey(workspaceID))
	if err != nil {
		return w, false, "", apiError(runtime.CodeInternal, "wait for pipeline workspace: "+err.Error())
	}
	defer release()
	// Fail closed: a known binding whose record cannot be read never falls
	// back to the project checkout (TS-09.R62).
	w, err = s.stateStore.ReadPipelineWorkspace(workspaceID)
	if err != nil {
		return w, false, "", apiError(runtime.CodeInternal, fmt.Sprintf("read pipeline workspace %s: %v", workspaceID, err))
	}
	switch w.State {
	case state.PipelineWorkspaceRemoving, state.PipelineWorkspaceRemoved:
		return w, false, "", apiError(runtime.CodeValidation, fmt.Sprintf(
			"the checkout for pipeline run %q was removed; Chuck will not recreate it. Branch %q and its commits remain in %s — check it out yourself, or start a new run",
			w.RunName, w.Branch, w.RepoPath))
	}
	if w.Mode != state.PipelineWorkspaceModeNewWorktree {
		recreated, warning, ae = s.ensureWorktreeCheckout(ctx, w.Project, w.CheckoutPath)
		return w, recreated, warning, ae
	}
	if !w.Owned {
		return s.prepareRunWorkspace(ctx, w)
	}
	if isExistingDir(w.CheckoutPath) {
		if ae := s.verifyRunCheckout(ctx, w); ae != nil {
			return w, false, "", ae
		}
	} else {
		if ae := s.recreateRunCheckout(ctx, w); ae != nil {
			return w, false, "", ae
		}
		recreated = true
	}
	// An owned checkout still `preparing` was interrupted between ownership and
	// setup; finish setup under this claim before anything starts in it.
	if recreated || w.State != state.PipelineWorkspaceReady {
		warning = s.runWorkspaceSetup(ctx, w)
		if w, err = s.stateStore.UpdatePipelineWorkspace(w.WorkspaceID, state.PipelineWorkspaceUpdate{State: state.PipelineWorkspaceReady}, state.PipelineWorkspacePreparing, state.PipelineWorkspaceReady, state.PipelineWorkspaceError); err != nil {
			return w, recreated, warning, apiError(runtime.CodeInternal, "record pipeline workspace ready: "+err.Error())
		}
	}
	return w, recreated, warning, nil
}

// prepareRunWorkspace creates the run's branch and checkout. Ownership commits
// only after Git created and verified both; any earlier failure removes only
// what this attempt created and records an actionable error (TS-12.R13).
func (s *Server) prepareRunWorkspace(ctx context.Context, w state.PipelineWorkspace) (state.PipelineWorkspace, bool, string, *runtime.APIError) {
	fail := func(message string) (state.PipelineWorkspace, bool, string, *runtime.APIError) {
		if updated, err := s.stateStore.UpdatePipelineWorkspace(w.WorkspaceID, state.PipelineWorkspaceUpdate{State: state.PipelineWorkspaceError, Error: message}, state.PipelineWorkspacePreparing, state.PipelineWorkspaceError); err == nil {
			w = updated
			s.publishWorkspace(w)
		} else {
			s.log.Error("pipeline workspace: record preparation error", "workspace", w.WorkspaceID, "err", err)
		}
		return w, false, "", apiError(runtime.CodeValidation, message)
	}
	path, err := s.configStore.EnsurePipelineWorktreeRoot(w.WorkspaceID)
	if err != nil {
		return fail("cannot prepare the worktree directory: " + err.Error())
	}
	if !sameCheckoutPath(path, w.CheckoutPath) {
		return fail(fmt.Sprintf("recorded checkout %q is not the owned location %q", w.CheckoutPath, path))
	}
	// Anything already at the path or branch was not recorded as created by
	// Chuck; recovery never guesses ownership or adopts it.
	if _, err := os.Lstat(path); err == nil {
		return fail(fmt.Sprintf("%s already exists but Chuck did not record creating it; move it aside, then retry the stage", path))
	}
	git := s.worktreeGit()
	if exists, err := git.BranchExists(ctx, w.RepoPath, w.Branch); err != nil {
		return fail("cannot check branch: " + err.Error())
	} else if exists {
		return fail(fmt.Sprintf("branch %q already exists in %s; start a new run with another branch name", w.Branch, w.RepoPath))
	}
	if err := git.CreateBranch(ctx, w.RepoPath, w.Branch, w.BaseCommit); err != nil {
		return fail("cannot create branch: " + err.Error())
	}
	if err := git.AddWorktreeExisting(ctx, w.RepoPath, path, w.Branch); err != nil {
		return fail(s.rollbackRunWorkspace(w, "cannot create the worktree: "+err.Error()))
	}
	if ae := s.verifyRunCheckout(ctx, w); ae != nil {
		return fail(s.rollbackRunWorkspace(w, ae.Message))
	}
	owned, err := s.stateStore.UpdatePipelineWorkspace(w.WorkspaceID, state.PipelineWorkspaceUpdate{State: state.PipelineWorkspacePreparing, Owned: true}, state.PipelineWorkspacePreparing, state.PipelineWorkspaceError)
	if err != nil {
		return fail(s.rollbackRunWorkspace(w, "record pipeline workspace ownership: "+err.Error()))
	}
	warning := s.runWorkspaceSetup(ctx, owned)
	ready, err := s.stateStore.UpdatePipelineWorkspace(w.WorkspaceID, state.PipelineWorkspaceUpdate{State: state.PipelineWorkspaceReady}, state.PipelineWorkspacePreparing)
	if err != nil {
		return owned, false, warning, apiError(runtime.CodeInternal, "record pipeline workspace ready: "+err.Error())
	}
	s.publishWorkspace(ready)
	return ready, false, warning, nil
}

// rollbackRunWorkspace undoes this attempt's checkout and still-unchanged
// branch, returning the message to record. A branch that moved is kept.
func (s *Server) rollbackRunWorkspace(w state.PipelineWorkspace, message string) string {
	ctx, cancel := context.WithTimeout(context.Background(), mutateCleanupTimeout)
	defer cancel()
	git := s.worktreeGit()
	var errs []error
	if _, err := os.Lstat(w.CheckoutPath); err == nil {
		if err := git.RemoveWorktree(ctx, w.RepoPath, w.CheckoutPath); err != nil {
			errs = append(errs, err)
		}
	}
	if head, err := git.ResolveCommit(ctx, w.RepoPath, "refs/heads/"+w.Branch); err == nil && head == w.BaseCommit {
		if err := git.DeleteBranch(ctx, w.RepoPath, w.Branch); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return message + "; cleanup incomplete: " + err.Error()
	}
	return message
}

// verifyRunCheckout requires the recorded path, repository and branch to agree
// before anything runs there (TS-12.R14). Mismatched contents are never
// repaired or removed.
func (s *Server) verifyRunCheckout(ctx context.Context, w state.PipelineWorkspace) *runtime.APIError {
	mismatch := func(detail string) *runtime.APIError {
		return apiError(runtime.CodeValidation, fmt.Sprintf(
			"the checkout for pipeline run %q at %s is not branch %q of %s (%s) — move that directory aside so Chuck can recreate it",
			w.RunName, w.CheckoutPath, w.Branch, w.RepoPath, detail))
	}
	if err := s.configStore.ValidatePipelineWorktreePath(w.WorkspaceID, w.CheckoutPath); err != nil {
		return mismatch(err.Error())
	}
	git := s.worktreeGit()
	repo, err := git.CommonDir(ctx, w.CheckoutPath)
	if err != nil || !sameCheckoutPath(repo, w.RepoPath) {
		return mismatch("different repository")
	}
	if branch, err := git.CurrentBranch(ctx, w.CheckoutPath); err != nil || branch != w.Branch {
		return mismatch("different branch")
	}
	return nil
}

// recreateRunCheckout restores an owned checkout lost out of band from its
// recorded branch; a missing branch fails without substituting another.
func (s *Server) recreateRunCheckout(ctx context.Context, w state.PipelineWorkspace) *runtime.APIError {
	if _, err := os.Lstat(w.CheckoutPath); err == nil {
		return apiError(runtime.CodeValidation, fmt.Sprintf(
			"%s is not the checkout for pipeline run %q — move it aside so Chuck can recreate branch %q", w.CheckoutPath, w.RunName, w.Branch))
	}
	git := s.worktreeGit()
	exists, err := git.BranchExists(ctx, w.RepoPath, w.Branch)
	if err != nil {
		return apiError(runtime.CodeValidation, fmt.Sprintf("cannot read repository %q: %v", w.RepoPath, err))
	}
	if !exists {
		return apiError(runtime.CodeValidation, fmt.Sprintf(
			"the checkout for pipeline run %q is missing and its branch %q no longer exists in %s — recreate the branch to continue",
			w.RunName, w.Branch, w.RepoPath))
	}
	if _, err := s.configStore.EnsurePipelineWorktreeRoot(w.WorkspaceID); err != nil {
		return apiError(runtime.CodeValidation, "cannot prepare the worktree directory: "+err.Error())
	}
	if err := git.PruneWorktrees(ctx, w.RepoPath); err != nil {
		s.log.Warn("pipeline workspace: prune before recreate", "workspace", w.WorkspaceID, "err", err)
	}
	if err := git.AddWorktreeExisting(ctx, w.RepoPath, w.CheckoutPath, w.Branch); err != nil {
		return apiError(runtime.CodeValidation, fmt.Sprintf("cannot recreate the checkout on branch %q: %v", w.Branch, err))
	}
	return nil
}

// runWorkspaceSetup runs the setup command frozen at Start, recording its
// bounded outcome on the workspace (TS-12.R5/R12).
func (s *Server) runWorkspaceSetup(ctx context.Context, w state.PipelineWorkspace) string {
	return s.runSetupCommand(ctx, w.SetupCommand, w.CheckoutPath, func(ok bool, output string) error {
		return s.stateStore.RecordPipelineWorkspaceSetup(w.WorkspaceID, ok, output)
	})
}

// startCheckout is what the shared readiness seam resolved for one start.
type startCheckout struct {
	Cwd       string
	Recreated bool
	Branch    string
	Warning   string
}

// resolveStartCheckout is the one readiness seam launch, resume and switch
// share (TS-12.R14, INV §2). A run workspace goes through ensureRunWorkspace;
// a fresh launch passes an empty cwd and receives the workspace's, while
// resume/switch pass their frozen cwd, which must still match. Everything else
// keeps the project-keyed resolver.
func (s *Server) resolveStartCheckout(ctx context.Context, projectID, workspaceID, cwd string) (startCheckout, *runtime.APIError) {
	if workspaceID == "" {
		recreated, warning, ae := s.ensureWorktreeCheckout(ctx, projectID, cwd)
		out := startCheckout{Cwd: cwd, Recreated: recreated, Warning: warning}
		if recreated {
			row, _ := s.ownedWorktree(projectID)
			out.Branch = row.Branch
		}
		return out, ae
	}
	w, recreated, warning, ae := s.ensureRunWorkspace(ctx, workspaceID)
	if ae != nil {
		return startCheckout{}, ae
	}
	if cwd == "" {
		cwd = w.CheckoutPath
	}
	if w.Project != projectID || !sameCheckoutPath(w.CheckoutPath, cwd) {
		return startCheckout{}, apiError(runtime.CodeValidation, fmt.Sprintf(
			"this agent's checkout %s does not match pipeline workspace %s", cwd, workspaceID))
	}
	return startCheckout{Cwd: cwd, Recreated: recreated, Branch: w.Branch, Warning: warning}, nil
}
