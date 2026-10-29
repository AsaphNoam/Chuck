import { useMemo, useState } from "react";
import { useProjects } from "../../api/config";
import { usePipelineWorkspaces } from "../../api/pipelines";
import { displayLabels } from "../../lib/labels";
import { formatRelative } from "./RunBrowser";
import { WorkspaceRemoveDialog, workspaceRemovable, workspaceStateLabel } from "./WorkspaceRemoveDialog";

/** WorkspacesPage lists retained run checkouts, including those whose run record
 *  was deleted; rows keep the run and project names they were created with
 *  (FS-14.R96, TS-09.R65). */
export function WorkspacesPage() {
  const projects = useProjects();
  const [project, setProject] = useState("");
  const [removing, setRemoving] = useState<string | null>(null);
  const query = usePipelineWorkspaces(project);
  const workspaces = useMemo(() => {
    const byID = new Map((query.data?.pages ?? []).flatMap((page) => page.workspaces).map((item) => [item.workspace_id, item]));
    return [...byID.values()];
  }, [query.data]);

  return <div className="pipeline-destination">
    <header className="pipeline-section-heading" data-ui="pipeline-section">
      <div><p className="pipeline-eyebrow">Retained checkouts</p><h2>Workspaces</h2><p>Checkouts that pipeline runs created, kept after the run ends until you remove them.</p></div>
      <label className="form-field pipeline-workspace-filter"><span>Project</span>
        <select value={project} onChange={(event) => setProject(event.target.value)}>
          <option value="">All projects</option>
          {displayLabels(Object.entries(projects.data ?? {}).map(([id, item]) => [id, item.title])).map(([id, title]) => <option key={id} value={id}>{title}</option>)}
        </select>
      </label>
    </header>
    {query.isLoading && <p className="pipeline-workspace-note" role="status">Loading workspaces…</p>}
    {query.error && <p className="form-error">{query.error.message}</p>}
    {!query.isLoading && !query.error && workspaces.length === 0 && <section className="pipeline-empty"><strong>No retained workspaces</strong><p>Runs started in a new worktree appear here.</p></section>}
    {workspaces.length > 0 && <ul className="pipeline-workspace-list">
      {workspaces.map((item) => <li className="pipeline-workspace-row" key={item.workspace_id}>
        <div className="pipeline-workspace-identity">
          <strong>{item.run_name || item.run_id}</strong>
          <small>{item.project_name || item.project}</small>
        </div>
        <dl className="pipeline-workspace-facts">
          <div><dt>Branch</dt><dd>{item.branch ? <code>{item.branch}</code> : "Project checkout"}</dd></div>
          <div><dt>Path</dt><dd><code>{item.cwd}</code></dd></div>
        </dl>
        <div className="pipeline-workspace-state">
          <span className={`pipeline-state pipeline-state-${item.state}`}>{workspaceStateLabel(item.state)}</span>
          <small>{formatRelative(item.updated_at)}</small>
          {item.error && <small className="form-error">{item.error}</small>}
        </div>
        {workspaceRemovable(item) && <button type="button" onClick={() => setRemoving(item.workspace_id)}>Remove checkout…</button>}
      </li>)}
    </ul>}
    {query.hasNextPage && <button type="button" className="pipeline-workspace-more" disabled={query.isFetchingNextPage} onClick={() => void query.fetchNextPage()}>{query.isFetchingNextPage ? "Loading…" : "More workspaces"}</button>}
    {removing && <WorkspaceRemoveDialog workspaceID={removing} onClose={() => setRemoving(null)} />}
  </div>;
}
