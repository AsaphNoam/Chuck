package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/pipeline"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// Retained pipeline workspace management (TS-12.R15/R16). The routes inherit
// the whole-mux localOnly guard (INV §14); no remote-control or MCP route
// reaches them.

type pipelineWorkspaceList struct {
	Workspaces []state.PipelineWorkspace `json:"workspaces"`
	NextCursor string                    `json:"next_cursor,omitempty"`
}

// pipelineWorkspaceDetail adds the on-demand Git observations to the record.
// Dirty state that could not be read is reported unknown, never clean.
type pipelineWorkspaceDetail struct {
	state.PipelineWorkspace
	Missing     bool   `json:"missing"`
	Dirty       bool   `json:"dirty"`
	DirtyKnown  bool   `json:"dirty_known"`
	Removable   bool   `json:"removable"`
	Blocked     string `json:"blocked,omitempty"`
	SetupOutput string `json:"setup_output,omitempty"`
}

type removeWorkspaceRequest struct {
	Revision           int64 `json:"revision"`
	DirtyKnown         bool  `json:"dirty_known"`
	Dirty              bool  `json:"dirty"`
	AcknowledgeUnknown bool  `json:"acknowledge_unknown"`
}

func (s *Server) handlePipelineWorkspaces(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project != "" && !config.ValidSlug(project) {
		writeAPIError(w, apiError(runtime.CodeValidation, "invalid project"))
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > pipeline.MaxListPage {
		limit = pipeline.MaxListPage
	}
	items, next, err := s.stateStore.ListPipelineWorkspaces(project, r.URL.Query().Get("cursor"), limit)
	if err != nil {
		writeAPIError(w, apiError(runtime.CodeInternal, "pipeline workspaces are unavailable"))
		return
	}
	writeJSON(w, http.StatusOK, pipelineWorkspaceList{Workspaces: items, NextCursor: next})
}

func (s *Server) handlePipelineWorkspace(w http.ResponseWriter, r *http.Request) {
	ws, ae := s.readWorkspaceParam(r)
	if ae != nil {
		writeAPIError(w, ae)
		return
	}
	writeJSON(w, http.StatusOK, s.pipelineWorkspaceDetail(r.Context(), ws))
}

func (s *Server) handleRemovePipelineWorkspace(w http.ResponseWriter, r *http.Request) {
	ws, ae := s.readWorkspaceParam(r)
	if ae != nil {
		writeAPIError(w, ae)
		return
	}
	var body removeWorkspaceRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAPIError(w, apiError(runtime.CodeValidation, "invalid JSON body"))
		return
	}
	removed, ae := s.removeRunWorkspace(r.Context(), ws.WorkspaceID, body)
	if ae != nil {
		writeAPIError(w, ae)
		return
	}
	writeJSON(w, http.StatusOK, s.pipelineWorkspaceDetail(r.Context(), removed))
}

func (s *Server) readWorkspaceParam(r *http.Request) (state.PipelineWorkspace, *runtime.APIError) {
	id := r.PathValue("id")
	if !config.ValidSlug(id) {
		return state.PipelineWorkspace{}, apiError(runtime.CodeValidation, "invalid workspace id")
	}
	ws, err := s.stateStore.ReadPipelineWorkspace(id)
	if errors.Is(err, state.ErrNotFound) {
		return ws, apiError(runtime.CodeNotFound, "no such pipeline workspace")
	}
	if err != nil {
		return ws, apiError(runtime.CodeInternal, "pipeline workspace is unavailable")
	}
	return ws, nil
}

// pipelineWorkspaceDetail runs the on-demand observations for one workspace.
func (s *Server) pipelineWorkspaceDetail(ctx context.Context, ws state.PipelineWorkspace) pipelineWorkspaceDetail {
	out := pipelineWorkspaceDetail{PipelineWorkspace: ws}
	if ws.SetupOK != nil && !*ws.SetupOK {
		out.SetupOutput = clipSetupOutput(ws.SetupOutput)
	}
	if ws.Mode != state.PipelineWorkspaceModeNewWorktree || ws.State == state.PipelineWorkspaceRemoved {
		out.Blocked = removalBlockedReason(ws)
		return out
	}
	if _, err := os.Lstat(ws.CheckoutPath); os.IsNotExist(err) {
		// A missing checkout holds nothing to lose.
		out.Missing, out.DirtyKnown = true, true
	} else if isExistingDir(ws.CheckoutPath) {
		if dirty, err := s.worktreeGit().IsDirty(ctx, ws.CheckoutPath); err == nil {
			out.Dirty, out.DirtyKnown = dirty, true
		}
	}
	if ae := s.workspaceRemovable(ws); ae != nil {
		out.Blocked = ae.Message
	} else {
		out.Removable = true
	}
	return out
}

func removalBlockedReason(ws state.PipelineWorkspace) string {
	if ws.State == state.PipelineWorkspaceRemoved {
		return "this checkout was already removed"
	}
	return "this run used an existing checkout; Chuck removes only checkouts it created for a run"
}

// workspaceRemovable is the removal eligibility rule: a Chuck-created checkout
// whose run has ended (or was deleted after ending), with no conversation or
// task start still using it (TS-12.R15).
func (s *Server) workspaceRemovable(ws state.PipelineWorkspace) *runtime.APIError {
	if ws.Mode != state.PipelineWorkspaceModeNewWorktree || !ws.Owned {
		return apiError(runtime.CodeValidation, removalBlockedReason(ws))
	}
	run, err := s.stateStore.ReadPipelineRun(ws.RunID)
	switch {
	case errors.Is(err, state.ErrNotFound):
		if !ws.RemovalEligible {
			return apiError(runtime.CodeConflict, "the run's end could not be confirmed")
		}
	case err != nil:
		return apiError(runtime.CodeInternal, "read pipeline run: "+err.Error())
	case run.State != "completed" && run.State != "stopped":
		return apiError(runtime.CodeConflict, "the run has not ended; stop or finish it before removing its checkout")
	default:
		starting, err := s.stateStore.PipelineRunHasStartingTasks(ws.RunID)
		if err != nil {
			return apiError(runtime.CodeInternal, err.Error())
		}
		if starting {
			return apiError(runtime.CodeConflict, "a run task is still starting in this checkout")
		}
	}
	// Any binding counts: an ordinary agent whose project points at the
	// retained checkout uses it as much as the run's own conversations.
	running, err := s.stateStore.RunningCheckouts()
	if err != nil {
		return apiError(runtime.CodeInternal, err.Error())
	}
	path := canonicalCheckoutPath(ws.CheckoutPath)
	inUse := s.startsInCheckout(ws.CheckoutPath)
	for _, r := range running {
		if r.WorkspaceID == ws.WorkspaceID || (r.Cwd != "" && canonicalCheckoutPath(r.Cwd) == path) {
			inUse++
		}
	}
	if inUse > 0 {
		return checkoutInUse(inUse)
	}
	return nil
}

// removeRunWorkspace is the explicit, consented removal (TS-12.R15). Under the
// workspace claim it re-checks eligibility, the disclosed dirty state and the
// checkout's identity, persists `removing` before the Git removal, and commits
// the `removed` tombstone after it. The branch and its commits are kept.
func (s *Server) removeRunWorkspace(ctx context.Context, workspaceID string, consent removeWorkspaceRequest) (state.PipelineWorkspace, *runtime.APIError) {
	_, release, err := s.claimWorktreeRecreate(ctx, runWorkspaceClaimKey(workspaceID))
	if err != nil {
		return state.PipelineWorkspace{}, apiError(runtime.CodeInternal, "wait for pipeline workspace: "+err.Error())
	}
	defer release()
	ws, err := s.stateStore.ReadPipelineWorkspace(workspaceID)
	if err != nil {
		return ws, apiError(runtime.CodeInternal, "read pipeline workspace: "+err.Error())
	}
	if ws.State == state.PipelineWorkspaceRemoved {
		return ws, nil
	}
	if consent.Revision != ws.Revision {
		return ws, apiError(runtime.CodeConflict, "the checkout changed since it was reviewed; review it again")
	}
	// From here no start may enter the checkout; one already inside refuses.
	done, ae := s.beginCheckoutRemoval(ws.CheckoutPath)
	if ae != nil {
		return ws, ae
	}
	defer done()
	if ae := s.workspaceRemovable(ws); ae != nil {
		return ws, ae
	}
	if !consent.DirtyKnown && !consent.AcknowledgeUnknown {
		return ws, apiError(runtime.CodeValidation, "uncommitted changes could not be checked; acknowledge that unknown changes may be lost")
	}
	present := true
	if _, err := os.Lstat(ws.CheckoutPath); os.IsNotExist(err) {
		present = false
	}
	if present {
		if ae := s.verifyRunCheckout(ctx, ws); ae != nil {
			return ws, ae
		}
		// A checkout disclosed as clean is rechecked; one that became dirty or
		// unreadable needs fresh consent (R7's clean-to-dirty rule).
		if consent.DirtyKnown && !consent.Dirty {
			dirty, err := s.worktreeGit().IsDirty(ctx, ws.CheckoutPath)
			if err != nil {
				return ws, apiError(runtime.CodeConflict, "uncommitted changes can no longer be checked; review the checkout again")
			}
			if dirty {
				return ws, apiError(runtime.CodeConflict, "the checkout gained uncommitted changes after review; review them and confirm again")
			}
		}
	}
	prior := ws.State
	ws, err = s.stateStore.UpdatePipelineWorkspace(ws.WorkspaceID, state.PipelineWorkspaceUpdate{State: state.PipelineWorkspaceRemoving}, prior)
	if err != nil {
		return ws, apiError(runtime.CodeConflict, "the checkout changed while removing; review it again")
	}
	if present {
		if err := s.worktreeGit().RemoveWorktree(ctx, ws.RepoPath, ws.CheckoutPath); err != nil {
			message := "remove checkout: " + err.Error()
			if reverted, rerr := s.stateStore.UpdatePipelineWorkspace(ws.WorkspaceID, state.PipelineWorkspaceUpdate{State: prior, Error: message}, state.PipelineWorkspaceRemoving); rerr == nil {
				ws = reverted
			}
			s.publishWorkspace(ws)
			return ws, apiError(runtime.CodeInternal, message)
		}
	}
	ws, err = s.stateStore.UpdatePipelineWorkspace(ws.WorkspaceID, state.PipelineWorkspaceUpdate{State: state.PipelineWorkspaceRemoved}, state.PipelineWorkspaceRemoving)
	if err != nil {
		return ws, apiError(runtime.CodeInternal, "record checkout removal: "+err.Error())
	}
	s.publishWorkspace(ws)
	return ws, nil
}

// reconcileRemovingWorkspaces finishes removals a restart interrupted: a
// checkout already gone becomes removed; one still present needs fresh
// validation and consent, so it returns to ready (TS-12.R15).
func (s *Server) reconcileRemovingWorkspaces() {
	items, err := s.stateStore.ListPipelineWorkspacesInState(state.PipelineWorkspaceRemoving, pipeline.MaxListPage)
	if err != nil {
		s.log.Warn("pipeline workspace: list interrupted removals", "err", err)
		return
	}
	for _, ws := range items {
		next := state.PipelineWorkspaceUpdate{State: state.PipelineWorkspaceRemoved}
		if _, err := os.Lstat(ws.CheckoutPath); !os.IsNotExist(err) {
			next = state.PipelineWorkspaceUpdate{State: state.PipelineWorkspaceReady, Error: "removal was interrupted; review the checkout and remove it again"}
		}
		if _, err := s.stateStore.UpdatePipelineWorkspace(ws.WorkspaceID, next, state.PipelineWorkspaceRemoving); err != nil {
			s.log.Warn("pipeline workspace: reconcile removal", "workspace", ws.WorkspaceID, "err", err)
		}
	}
}

// publishWorkspace republishes the owning run where its record remains and the
// agents frozen to the workspace, so open run details and agent metadata
// refetch the workspace state (TS-09.R65).
func (s *Server) publishWorkspace(ws state.PipelineWorkspace) {
	if s.pipelineMgr != nil {
		s.pipelineMgr.PublishRun(ws.RunID)
	}
	agents, err := s.stateStore.PipelineWorkspaceAgentIDs(ws.WorkspaceID)
	if err != nil {
		s.log.Warn("pipeline workspace: list agents to republish", "workspace", ws.WorkspaceID, "err", err)
		return
	}
	for _, id := range agents {
		if _, err := s.stateMgr.Touch(id); err != nil {
			s.log.Warn("pipeline workspace: republish agent", "agent", id, "err", err)
		}
	}
}
