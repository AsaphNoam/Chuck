import type { PipelineWorkspace } from "../../schemas/pipeline";
import { workspaceRemovable, workspaceStateLabel } from "./WorkspaceRemoveDialog";

/** RunWorkspace names the checkout a run works in. A run that predates run
 *  workspaces has none and shows nothing (TS-12.R17). The removal action stays
 *  here for as long as the run is ended and the checkout exists, so a person
 *  who was away when it ended can still act (FS-14.R94). */
export function RunWorkspace({ workspace, terminal, onRemove }: { workspace: PipelineWorkspace; terminal: boolean; onRemove: () => void }) {
  const isolated = workspace.mode === "new_worktree";
  return <section className="pipeline-run-workspace" aria-label="Workspace" data-slot="workspace">
    <div className="pipeline-run-workspace-head">
      <strong>{isolated ? "New worktree" : "Project checkout"}</strong>
      <span className={`pipeline-state pipeline-state-${workspace.state}`}>{workspaceStateLabel(workspace.state)}</span>
    </div>
    <dl className="pipeline-workspace-facts">
      {workspace.branch && <div><dt>Branch</dt><dd><code>{workspace.branch}</code></dd></div>}
      <div><dt>Path</dt><dd><code>{workspace.cwd}</code></dd></div>
    </dl>
    {workspace.error && <p className="form-error">{workspace.error}</p>}
    {workspace.setup_ok === false && <div className="pipeline-warning"><strong>Project setup did not finish</strong><p>The checkout was created, but its setup step failed. Agents may be missing dependencies or generated files.</p></div>}
    {terminal && workspaceRemovable(workspace) && <div className="pipeline-action-choice">
      <button type="button" onClick={onRemove}>Remove checkout…</button>
      <small>Keeps the branch, its commits and the agents' conversations.</small>
    </div>}
  </section>;
}
