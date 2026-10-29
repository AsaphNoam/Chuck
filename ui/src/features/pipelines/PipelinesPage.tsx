import { useState } from "react";
import {
  Link,
  Navigate,
  NavLink,
  Outlet,
  useLocation,
  useNavigate,
  useParams,
  useSearchParams,
} from "react-router-dom";
import { PipelineAPIError, usePipelineProposals, usePipelineTemplates } from "../../api/pipelines";
import { Button } from "../../components/ui";
import type { PipelineProposal } from "../../schemas/pipeline";
import { ChuckyBuilder } from "./ChuckyBuilder";
import { RunDetail, RunsLedger } from "./RunBrowser";
import { RunStartDialog } from "./RunStartForm";
import { TemplateEditor, type TemplateEditorSeed } from "./TemplateEditor";

export function PipelinesLayout() {
  const proposals = usePipelineProposals();
  // The count each destination shows for the other counts pending offers only; a
  // declined entry is not waiting on anybody (FS-14.R49).
  const saveCount = proposals.data?.pending.filter((item) => item.kind === "save_template").length ?? 0;
  const startCount = proposals.data?.pending.filter((item) => item.kind === "start_run").length ?? 0;

  return (
    <div className="pipelines-page" data-ui="pipelines">
      <header className="pipeline-page-header" data-slot="header">
        <div className="pipeline-title-lockup">
          <span className="pipeline-page-icon" aria-hidden="true"><svg width="23" height="23" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"><rect x="3" y="3" width="6" height="6" rx="1.5" /><rect x="15" y="15" width="6" height="6" rx="1.5" /><path d="M6 9v6a3 3 0 0 0 3 3h6M15 6h6m-3-3v6" /></svg></span>
          <div><p className="pipeline-eyebrow">Sequential orchestration</p><h1>Pipelines</h1></div>
        </div>
        <p>Thoughtful workflows. Work you can follow.</p>
      </header>
      <nav className="pipeline-tabs" aria-label="Pipeline destinations" data-slot="navigation">
        <NavLink to="/pipelines/runs" className={({ isActive }) => isActive ? "pipeline-tab pipeline-tab-active" : "pipeline-tab"}>
          Runs{startCount > 0 && <span>{startCount}</span>}
        </NavLink>
        <NavLink to="/pipelines/workspaces" className={({ isActive }) => isActive ? "pipeline-tab pipeline-tab-active" : "pipeline-tab"}>Workspaces</NavLink>
        <NavLink to="/pipelines/templates" className={({ isActive }) => isActive ? "pipeline-tab pipeline-tab-active" : "pipeline-tab"}>
          Templates{saveCount > 0 && <span>{saveCount}</span>}
        </NavLink>
        <span className="pipeline-tabs-note">One stage at a time</span>
      </nav>
      <main className="pipeline-route" data-slot="content"><Outlet /></main>
    </div>
  );
}

export function PipelinesIndex() {
  const [search] = useSearchParams();
  const legacyRun = search.get("run");
  return <Navigate to={legacyRun ? `/pipelines/runs/${encodeURIComponent(legacyRun)}` : "/pipelines/runs"} replace />;
}

export function RunsPage() {
  const navigate = useNavigate();
  const templates = usePipelineTemplates();
  const [startOpen, setStartOpen] = useState(false);
  const [proposal, setProposal] = useState<Extract<PipelineProposal, { kind: "start_run" }> | null>(null);

  const reviewProposal = (item: Extract<PipelineProposal, { kind: "start_run" }>) => {
    setProposal(item);
    setStartOpen(true);
  };

  // FS-14.R36: with no runnable template the dialog could only open onto a
  // permanently disabled Next, so the gate names the missing step instead.
  const noTemplate = templates.isSuccess && templates.data.every((record) => !record.valid);

  return (
    <div className="pipeline-destination pipeline-runs-destination">
      <header className="pipeline-section-heading" data-ui="pipeline-section">
        <div><p className="pipeline-eyebrow">Operational ledger</p><h2>Runs</h2><p>Follow the work in motion, and the record it leaves behind.</p></div>
        <div className="pipeline-start-gate">
          <Button variant="primary" disabled={noTemplate} onClick={() => { setProposal(null); setStartOpen(true); }}><span aria-hidden="true">＋</span> Start run</Button>
          {noTemplate && <small>No template is ready to run yet. <Link to="/pipelines/templates">Create one in Templates</Link>.</small>}
        </div>
      </header>
      <ChuckyBuilder proposalKind="start_run" showLauncher={false} onRunProposal={reviewProposal} />
      <RunsLedger />
      <RunStartDialog
        open={startOpen}
        proposal={proposal}
        onOpenChange={setStartOpen}
        onStarted={(runID) => navigate(`/pipelines/runs/${encodeURIComponent(runID)}`)}
      />
    </div>
  );
}

export function PipelineRunPage() {
  const { runID = "" } = useParams();
  const navigate = useNavigate();
  // INV §1: RunDetail holds per-run state (continuation draft, mutation error, the appended-
  // attempts "seen" set) with nothing scoped to runID. Without a key, browser back/forward across
  // two visited run pages reuses this element and that state leaks across the lifecycle boundary.
  return <RunDetail key={runID} runID={runID} onDeleted={() => navigate("/pipelines/runs")} />;
}

export function TemplatesPage() {
  const templates = usePipelineTemplates();
  const navigate = useNavigate();

  const reviewProposal = (proposal: Extract<PipelineProposal, { kind: "save_template" }>) => {
    const seed: TemplateEditorSeed = { id: proposal.payload.id, template: proposal.payload.template, proposal };
    navigate(`/pipelines/templates/${encodeURIComponent(proposal.payload.id)}`, { state: { seed } });
  };

  return (
    <div className="pipeline-destination pipeline-templates-destination">
      <header className="pipeline-section-heading" data-ui="pipeline-section">
        <div className="pipeline-library-heading-copy">
          <p className="pipeline-eyebrow">Reusable definitions</p>
          <h2>Templates</h2>
          <p>Keep stage logic model-neutral; choose runtimes only when a run starts.</p>
        </div>
        <div className="pipeline-library-heading-actions">
          <Button variant="primary" onClick={() => navigate("/pipelines/templates/new")}>Create manually</Button>
        </div>
      </header>
      <ChuckyBuilder proposalKind="save_template" showLauncher onTemplateProposal={reviewProposal} />
      <section className="pipeline-library-content" aria-label="Saved pipeline templates">
        <div className="pipeline-library-label">
          <span>Saved templates</span>
          <span>{templates.data?.length ?? 0} total</span>
        </div>
        {templates.isLoading && <TemplateLibrarySkeleton />}
        {templates.error && <p className="form-error pipeline-library-error">{templates.error.message}</p>}
        {!templates.isLoading && (templates.data?.length ?? 0) === 0 && (
          <section className="pipeline-empty pipeline-empty-library">
            <strong>No templates yet</strong>
            <p>Create a definition manually or ask Chucky to shape a reusable pipeline from a description.</p>
          </section>
        )}
        {(templates.data?.length ?? 0) > 0 && (
          <div className="pipeline-template-library" data-ui="pipeline-template-library" data-slot="list">
            {templates.data?.map((record) => (
              <Link className="pipeline-template-row" key={record.id} to={`/pipelines/templates/${encodeURIComponent(record.id)}`} data-slot="item">
                <span className="pipeline-template-index">
                  <strong>{String(record.template.stages.length).padStart(2, "0")}</strong>
                  <small>{record.template.stages.length === 1 ? "stage" : "stages"}</small>
                </span>
                <span className="pipeline-template-identity">
                  <strong>{record.template.title || record.id}</strong>
                  <small><code>{record.id}</code></small>
                </span>
                <span className="pipeline-template-validation">
                  <span className={record.valid ? "pipeline-validation pipeline-validation-valid" : "pipeline-validation pipeline-validation-invalid"}>
                    {record.valid ? "Ready to run" : `${record.diagnostics.length} ${record.diagnostics.length === 1 ? "issue" : "issues"}`}
                  </span>
                  {!record.valid && <small>Open to review validation</small>}
                </span>
                <span aria-hidden="true" className="pipeline-row-arrow">→</span>
              </Link>
            ))}
          </div>
        )}
        {!templates.isLoading && (templates.data?.length ?? 0) > 0 && (
          <p className="pipeline-library-footnote">Runtime assignments are chosen for each run; these definitions stay model-neutral.</p>
        )}
      </section>
    </div>
  );
}

export function PipelineTemplatePage() {
  const { templateID = "" } = useParams();
  const location = useLocation();
  const navigate = useNavigate();
  const templates = usePipelineTemplates();
  const routeSeed = (location.state as { seed?: TemplateEditorSeed } | null)?.seed;
  const record = templates.data?.find((entry) => entry.id === templateID);
  const seed = routeSeed ?? (record ? { id: record.id, template: record.template } : null);

  if (templates.isLoading && !seed) return <TemplateEditorSkeleton />;
  if (templateID !== "new" && !seed) {
    const deleted = !templates.error || (templates.error instanceof PipelineAPIError && templates.error.status === 404);
    return <section className="pipeline-missing">
      <p className="pipeline-eyebrow">Template unavailable</p>
      <h2>{deleted ? "This template is gone." : "This template could not be loaded."}</h2>
      <p>{deleted ? "It may have been deleted in another tab. Existing runs still retain their frozen setup." : "The template library could not be read just now. Return to Templates and open it again."}</p>
      {!deleted && <p className="form-error">{templates.error.message}</p>}
      <Link to="/pipelines/templates">Return to Templates</Link>
    </section>;
  }

  return (
    <TemplateEditor
      seed={seed}
      createNew={templateID === "new"}
      onSaved={(id) => navigate(`/pipelines/templates/${encodeURIComponent(id)}`, { replace: true })}
      onDeleted={() => navigate("/pipelines/templates")}
    />
  );
}

function TemplateLibrarySkeleton() {
  return <div className="pipeline-template-library pipeline-skeleton" aria-label="Loading templates"><span /><span /><span /></div>;
}

function TemplateEditorSkeleton() {
  return <div className="pipeline-editor-shell pipeline-skeleton" aria-label="Loading template"><span /><span /><span /></div>;
}

export const PipelinesPage = PipelinesLayout;
