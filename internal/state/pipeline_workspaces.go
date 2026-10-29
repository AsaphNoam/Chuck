package state

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"
)

// Run-owned pipeline workspaces (TS-12.R12–R16). One row records the checkout a
// pipeline run selected at Start. Project and run ids are logical references:
// deleting the run or project keeps the record, its tombstone and ownership.
//
// A new-worktree row starts as an intent (`preparing`, owned=false). Ownership
// is set only after Git has created and verified the checkout, so a crash can
// leave an unowned intent but never a record authorizing deletion of something
// Chuck did not create (INV §15).

const (
	PipelineWorkspaceModeProject     = "project"
	PipelineWorkspaceModeNewWorktree = "new_worktree"

	PipelineWorkspacePreparing = "preparing"
	PipelineWorkspaceReady     = "ready"
	PipelineWorkspaceError     = "error"
	PipelineWorkspaceRemoving  = "removing"
	PipelineWorkspaceRemoved   = "removed"

	// pipelineWorkspaceErrorLimit bounds the stored preparation error (INV §8).
	pipelineWorkspaceErrorLimit = 2000
)

// PipelineWorkspace is one run's frozen workspace selection and lifecycle.
type PipelineWorkspace struct {
	WorkspaceID     string     `json:"workspace_id"`
	RunID           string     `json:"run_id"`
	RequestID       string     `json:"-"`
	Project         string     `json:"project"`
	ProjectName     string     `json:"project_name"`
	RunName         string     `json:"run_name"`
	Mode            string     `json:"mode"`
	SelectionDigest string     `json:"-"`
	RepoPath        string     `json:"repo_path,omitempty"`
	Base            string     `json:"base,omitempty"`
	BaseCommit      string     `json:"base_commit,omitempty"`
	Branch          string     `json:"branch,omitempty"`
	CheckoutPath    string     `json:"cwd"`
	SetupCommand    string     `json:"-"`
	State           string     `json:"state"`
	Owned           bool       `json:"owned"`
	Revision        int64      `json:"revision"`
	Error           string     `json:"error,omitempty"`
	SetupOK         *bool      `json:"setup_ok,omitempty"`
	SetupAt         *time.Time `json:"setup_at,omitempty"`
	SetupOutput     string     `json:"-"`
	RemovalEligible bool       `json:"removal_eligible"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

const pipelineWorkspaceColumns = `workspace_id, run_id, request_id, project, project_name, run_name, mode,
  selection_digest, repo_path, base, base_commit, branch, checkout_path, setup_command, state, owned,
  revision, error, setup_ok, setup_at, setup_output, removal_eligible, created_at, updated_at`

// NewPipelineWorkspaceID mints a slug-shaped id: it names the checkout's
// directory under the owned worktree root (TS-12.R12).
func (s *Store) NewPipelineWorkspaceID() (string, error) {
	return s.newPipelineID("pw-", "pipeline_workspaces", "workspace_id")
}

func insertPipelineWorkspaceTx(tx *sql.Tx, w PipelineWorkspace, now time.Time) error {
	if w.Revision <= 0 {
		w.Revision = 1
	}
	_, err := tx.Exec(`INSERT INTO pipeline_workspaces(`+pipelineWorkspaceColumns+`)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', NULL, NULL, '', 0, ?, ?)`,
		w.WorkspaceID, w.RunID, w.RequestID, w.Project, w.ProjectName, w.RunName, w.Mode,
		w.SelectionDigest, w.RepoPath, w.Base, w.BaseCommit, w.Branch, w.CheckoutPath, w.SetupCommand,
		w.State, boolInt(w.Owned), w.Revision, formatTime(now), formatTime(now))
	if err != nil {
		return fmt.Errorf("state: insert pipeline workspace: %w", err)
	}
	return nil
}

// ReadPipelineWorkspace returns one workspace by id, or ErrNotFound.
func (s *Store) ReadPipelineWorkspace(workspaceID string) (PipelineWorkspace, error) {
	return s.readPipelineWorkspace(`workspace_id = ?`, workspaceID)
}

// ReadPipelineWorkspaceForRun returns a run's workspace, or ErrNotFound for a
// run started before workspaces existed (TS-12.R17).
func (s *Store) ReadPipelineWorkspaceForRun(runID string) (PipelineWorkspace, error) {
	return s.readPipelineWorkspace(`run_id = ?`, runID)
}

func (s *Store) readPipelineWorkspace(where string, arg string) (PipelineWorkspace, error) {
	w, err := scanPipelineWorkspace(s.db.QueryRow(`SELECT `+pipelineWorkspaceColumns+` FROM pipeline_workspaces WHERE `+where, arg))
	if errors.Is(err, sql.ErrNoRows) {
		return PipelineWorkspace{}, ErrNotFound
	}
	if err != nil {
		return PipelineWorkspace{}, fmt.Errorf("state: read pipeline workspace: %w", err)
	}
	return w, nil
}

// PipelineWorkspaceUpdate is one lifecycle transition. Owned may only become
// true; nothing in this store clears verified ownership.
type PipelineWorkspaceUpdate struct {
	State string
	Error string
	Owned bool
}

// UpdatePipelineWorkspace applies a transition and bumps the revision a removal
// confirmation is checked against (TS-12.R15). fromStates, when non-empty, is
// the compare-and-set guard: the row must currently be in one of them.
func (s *Store) UpdatePipelineWorkspace(workspaceID string, update PipelineWorkspaceUpdate, fromStates ...string) (PipelineWorkspace, error) {
	errText := []rune(update.Error)
	if len(errText) > pipelineWorkspaceErrorLimit {
		errText = errText[len(errText)-pipelineWorkspaceErrorLimit:]
	}
	query := `UPDATE pipeline_workspaces SET state = ?, error = ?, owned = MAX(owned, ?),
  revision = revision + 1, updated_at = ? WHERE workspace_id = ?`
	args := []any{update.State, string(errText), boolInt(update.Owned), formatTime(timeNow()), workspaceID}
	if len(fromStates) > 0 {
		query += ` AND state IN (` + placeholders(len(fromStates)) + `)`
		for _, from := range fromStates {
			args = append(args, from)
		}
	}
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return PipelineWorkspace{}, fmt.Errorf("state: update pipeline workspace: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return PipelineWorkspace{}, fmt.Errorf("state: update pipeline workspace: %w", err)
	} else if n == 0 {
		if _, readErr := s.ReadPipelineWorkspace(workspaceID); readErr != nil {
			return PipelineWorkspace{}, readErr
		}
		return PipelineWorkspace{}, ErrPipelineWorkspaceConflict
	}
	return s.ReadPipelineWorkspace(workspaceID)
}

// ErrPipelineWorkspaceConflict reports a transition whose expected state no
// longer holds.
var ErrPipelineWorkspaceConflict = errors.New("state: pipeline workspace changed")

// RecordPipelineWorkspaceSetup stores a setup run's bounded outcome, as
// RecordProjectWorktreeSetup does for project checkouts (TS-12.R5).
func (s *Store) RecordPipelineWorkspaceSetup(workspaceID string, ok bool, output string) error {
	if len(output) > setupOutputLimit {
		output = output[len(output)-setupOutputLimit:]
	}
	res, err := s.db.Exec(`UPDATE pipeline_workspaces SET setup_ok = ?, setup_at = ?, setup_output = ?,
  revision = revision + 1, updated_at = ? WHERE workspace_id = ?`,
		boolInt(ok), formatTime(timeNow()), output, formatTime(timeNow()), workspaceID)
	if err != nil {
		return fmt.Errorf("state: record pipeline workspace setup: %w", err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("state: record pipeline workspace setup: %w", err)
	} else if n == 0 {
		return ErrNotFound
	}
	return nil
}

func scanPipelineWorkspace(row rowScanner) (PipelineWorkspace, error) {
	var (
		w                  PipelineWorkspace
		owned, eligible    int
		setupOK            sql.NullInt64
		setupAt            sql.NullString
		createdAt, updated string
	)
	if err := row.Scan(&w.WorkspaceID, &w.RunID, &w.RequestID, &w.Project, &w.ProjectName, &w.RunName, &w.Mode,
		&w.SelectionDigest, &w.RepoPath, &w.Base, &w.BaseCommit, &w.Branch, &w.CheckoutPath, &w.SetupCommand,
		&w.State, &owned, &w.Revision, &w.Error, &setupOK, &setupAt, &w.SetupOutput, &eligible, &createdAt, &updated); err != nil {
		return PipelineWorkspace{}, err
	}
	w.Owned, w.RemovalEligible = owned != 0, eligible != 0
	if setupOK.Valid {
		ok := setupOK.Int64 != 0
		w.SetupOK = &ok
	}
	at, err := parseOptionalTime(setupAt)
	if err != nil {
		return PipelineWorkspace{}, wrapTimeErr("pipeline_workspaces.setup_at", err)
	}
	w.SetupAt = at
	if w.CreatedAt, err = parseTime(createdAt); err != nil {
		return PipelineWorkspace{}, wrapTimeErr("pipeline_workspaces.created_at", err)
	}
	if w.UpdatedAt, err = parseTime(updated); err != nil {
		return PipelineWorkspace{}, wrapTimeErr("pipeline_workspaces.updated_at", err)
	}
	return w, nil
}

// ErrTaskTargetWorkspace refuses run work aimed at an existing agent whose
// frozen checkout is not the run's selected workspace (FS-14.R95).
var ErrTaskTargetWorkspace = errors.New("state: target agent is outside the run workspace")

// placeRunTaskTx applies a run's workspace rules to a task joining its
// lineage, inside the task's own write transaction (TS-09.R62/R64): a new
// launch receives the frozen run group whatever the request said, and an
// existing target must already work in the run's checkout. A run started
// before workspaces has no record and keeps its old behavior (TS-12.R17).
func placeRunTaskTx(tx *sql.Tx, task *Task, runID string) error {
	var runName string
	err := tx.QueryRow(`SELECT run_name FROM pipeline_workspaces WHERE run_id = ?`, runID).Scan(&runName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("state: read run workspace: %w", err)
	}
	switch task.TargetKind {
	case TargetLaunch:
		task.Group = runName
	case TargetAgent:
		return checkRunTarget(tx, runID, task.TargetAgentID)
	}
	return nil
}

// checkRunTarget is the one eligibility rule for an existing agent receiving a
// run's work. An isolated run needs the agent's frozen workspace identity; a
// project-mode run needs its frozen checkout path (TS-09.R63).
func checkRunTarget(q rowQuerier, runID, agentID string) error {
	var workspaceID, mode, checkout string
	err := q.QueryRow(`SELECT workspace_id, mode, checkout_path FROM pipeline_workspaces WHERE run_id = ?`, runID).Scan(&workspaceID, &mode, &checkout)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("state: read run workspace: %w", err)
	}
	var cwd, agentWorkspace string
	err = q.QueryRow(`SELECT cwd, workspace_id FROM sessions WHERE agent_id = ?`, agentID).Scan(&cwd, &agentWorkspace)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrTaskTargetWorkspace
	}
	if err != nil {
		return fmt.Errorf("state: read target session: %w", err)
	}
	if filepath.Clean(cwd) != filepath.Clean(checkout) {
		return ErrTaskTargetWorkspace
	}
	if mode == PipelineWorkspaceModeNewWorktree && agentWorkspace != workspaceID {
		return ErrTaskTargetWorkspace
	}
	return nil
}

// CheckPipelineTaskTarget repeats the creation-time target rule at admission,
// so a target that moved checkouts after the task was written is still refused.
func (s *Store) CheckPipelineTaskTarget(taskID, agentID string) error {
	var runID string
	err := s.db.QueryRow(`SELECT pipeline_run_id FROM task_lineage WHERE task_id = ?`, taskID).Scan(&runID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && runID == "") {
		return nil
	}
	if err != nil {
		return fmt.Errorf("state: read task lineage: %w", err)
	}
	return checkRunTarget(s.db, runID, agentID)
}

// ListPipelineWorkspaces is the bounded DB-only workspace list, newest first,
// optionally for one project. The cursor is the last workspace id returned; an
// empty next cursor ends the list (TS-12.R16).
func (s *Store) ListPipelineWorkspaces(project, cursor string, limit int) ([]PipelineWorkspace, string, error) {
	query := `SELECT ` + pipelineWorkspaceColumns + ` FROM pipeline_workspaces WHERE 1 = 1`
	args := []any{}
	if project != "" {
		query += ` AND project = ?`
		args = append(args, project)
	}
	if cursor != "" {
		query += ` AND (created_at, workspace_id) < (SELECT created_at, workspace_id FROM pipeline_workspaces WHERE workspace_id = ?)`
		args = append(args, cursor)
	}
	query += ` ORDER BY created_at DESC, workspace_id DESC LIMIT ?`
	args = append(args, limit+1)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("state: list pipeline workspaces: %w", err)
	}
	defer rows.Close()
	out := []PipelineWorkspace{}
	for rows.Next() {
		w, err := scanPipelineWorkspace(rows)
		if err != nil {
			return nil, "", fmt.Errorf("state: scan pipeline workspace: %w", err)
		}
		out = append(out, w)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("state: iterate pipeline workspaces: %w", err)
	}
	next := ""
	if len(out) > limit {
		out = out[:limit]
		next = out[limit-1].WorkspaceID
	}
	return out, next, nil
}

// ListPipelineWorkspacesInState returns at most limit workspaces in one state,
// oldest first, for bounded restart reconciliation.
func (s *Store) ListPipelineWorkspacesInState(workspaceState string, limit int) ([]PipelineWorkspace, error) {
	rows, err := s.db.Query(`SELECT `+pipelineWorkspaceColumns+` FROM pipeline_workspaces WHERE state = ?
ORDER BY updated_at, workspace_id LIMIT ?`, workspaceState, limit)
	if err != nil {
		return nil, fmt.Errorf("state: list pipeline workspaces: %w", err)
	}
	defer rows.Close()
	out := []PipelineWorkspace{}
	for rows.Next() {
		w, err := scanPipelineWorkspace(rows)
		if err != nil {
			return nil, fmt.Errorf("state: scan pipeline workspace: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// PipelineWorkspaceAgents returns up to limit agents whose frozen session
// names the workspace: the conversations a removal must not pull a checkout
// from under (TS-12.R15).
func (s *Store) PipelineWorkspaceAgents(workspaceID string, limit int) ([]string, error) {
	rows, err := s.db.Query(`SELECT agent_id FROM sessions WHERE workspace_id = ? ORDER BY agent_id LIMIT ?`, workspaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("state: list workspace agents: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("state: scan workspace agent: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// PipelineRunHasStartingTasks reports a run task between admission and its
// confirmed start, whose agent has no session row yet.
func (s *Store) PipelineRunHasStartingTasks(runID string) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM tasks t JOIN task_lineage l ON l.task_id = t.task_id
WHERE l.pipeline_run_id = ? AND t.state = ?`, runID, TaskStarting).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("state: read starting run tasks: %w", err)
	}
	return n > 0, nil
}

// PipelineWorkspaceIDForTask returns the workspace a task's run selected, or
// "" for standalone work and runs started before workspaces.
func (s *Store) PipelineWorkspaceIDForTask(taskID string) (string, error) {
	var workspaceID string
	err := s.db.QueryRow(`SELECT w.workspace_id FROM task_lineage l
JOIN pipeline_workspaces w ON w.run_id = l.pipeline_run_id WHERE l.task_id = ?`, taskID).Scan(&workspaceID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("state: read task workspace: %w", err)
	}
	return workspaceID, nil
}
