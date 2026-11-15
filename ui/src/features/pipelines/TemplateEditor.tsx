import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { useRoles } from "../../api/config";
import {
  getPipelineTemplate,
  PIPELINE_QUERY_KEYS,
  PipelineAPIError,
  pipelineDiagnostics,
  useDeletePipelineTemplate,
  useSavePipelineTemplate,
  validatePipelineTemplate,
} from "../../api/pipelines";
import { AutoGrowTextarea } from "../../components/ui";
import { personaLabel } from "../../lib/personas";
import { pipelineTemplateSchema, type PipelineDiagnostic, type PipelineProposal, type PipelineTemplate } from "../../schemas/pipeline";
import { newPipelineKey, pipelineOptionLabels, pipelineStageLabels } from "./pipelineLabels";
import { StageBindings } from "./StageBindings";
import { defaultThinkTank, ThinkTankStageFields } from "./ThinkTankStageFields";
import { templateDiagnostic } from "./templateDiagnostics";

export interface TemplateEditorSeed {
  id: string;
  template: PipelineTemplate;
  diagnostics?: PipelineDiagnostic[];
  proposal?: Extract<PipelineProposal, { kind: "save_template" }>;
}

const newStage = (): PipelineTemplate["stages"][number] => ({
  id: newPipelineKey("stage"), title: "", objective: "", inputs: [], outputs: [],
  coordination: "standing", dedicated_role: "", approval_after_success: false, required_for_completion: false,
});
const blankTemplate = (): PipelineTemplate => ({
  version: 2, title: "", orchestrator_role: "implementer", inputs: [], stages: [newStage()],
});

function sameTemplate(left: PipelineTemplate, right: PipelineTemplate) {
  const compare = (template: PipelineTemplate) => JSON.stringify(pipelineTemplateSchema.parse({
    ...template, orchestrator_instructions: template.orchestrator_instructions ?? "",
  }));
  return compare(left) === compare(right);
}

export function TemplateEditor({ seed, proposalUpdatesExisting = false, savedNotice, onSaved, onDeleted }: {
  seed?: TemplateEditorSeed | null;
  proposalUpdatesExisting?: boolean;
  savedNotice?: string;
  onSaved?: (id: string, notice: string) => void;
  onDeleted?: () => void;
}) {
  const queryClient = useQueryClient();
  const roles = useRoles();
  const save = useSavePipelineTemplate();
  const remove = useDeletePipelineTemplate();
  const [id, setID] = useState(() => seed?.id ?? newPipelineKey("pipeline"));
  const [draft, setDraft] = useState(() => seed ? structuredClone(seed.template) : blankTemplate());
  const [exists, setExists] = useState(Boolean(seed && (!seed.proposal || proposalUpdatesExisting)));
  const [proposal, setProposal] = useState(seed?.proposal);
  const [diagnostics, setDiagnostics] = useState(seed?.diagnostics ?? []);
  const [notice, setNotice] = useState(savedNotice ?? "");
  const [error, setError] = useState<string | null>(null);
  const [selectedStage, setSelectedStage] = useState(() => {
    const index = seed?.diagnostics?.[0] && templateDiagnostic(seed.diagnostics[0], draft, pipelineStageLabels(draft.stages)).stageIndex;
    return index !== undefined && draft.stages[index] ? index : 0;
  });
  const [busy, setBusy] = useState<"validate" | "save" | null>(null);
  const submitted = useRef<PipelineTemplate | null>(null);
  const focusNewStage = useRef(false);
  const form = useRef<HTMLFieldSetElement>(null);
  const labels = pipelineStageLabels(draft.stages);
  const roleOptions = pipelineOptionLabels(Object.entries(roles.data ?? {}).map(([key, role]) => [key, role.title?.trim() ? personaLabel(key, role) : ""]), "role");
  const stage = draft.stages[selectedStage];
  const path = `stages.${selectedStage}`;
  const visibleDiagnostics = diagnostics.map((item) => templateDiagnostic(item, draft, labels));

  useEffect(() => {
    if (!focusNewStage.current) return;
    focusNewStage.current = false;
    form.current?.querySelector<HTMLInputElement>(`[data-template-field="${path}.title"]`)?.focus();
  }, [path]);

  useEffect(() => {
    if (busy || diagnostics.length === 0) return;
    const first = templateDiagnostic(diagnostics[0], draft, pipelineStageLabels(draft.stages));
    const fields = [...form.current?.querySelectorAll<HTMLElement>("[data-template-field]") ?? []];
    const field = first.control;
    const target = fields.find((item) => item.dataset.templateField === field)
      ?? fields.reverse().find((item) => field.startsWith(`${item.dataset.templateField}.`));
    for (let parent = target; parent && parent !== form.current; parent = parent.parentElement ?? undefined) {
      if (parent instanceof HTMLDetailsElement) parent.open = true;
    }
    if (target?.matches("input, select, textarea, summary")) target.focus();
    else target?.querySelector<HTMLElement>("input, select, textarea, summary, button:not(:disabled)")?.focus();
  }, [busy, diagnostics, draft]);

  const showDiagnostics = (items: PipelineDiagnostic[]) => {
    setDiagnostics(items);
    if (!items.length) return;
    const index = templateDiagnostic(items[0], draft, labels).stageIndex;
    if (index !== undefined && draft.stages[index]) setSelectedStage(index);
  };

  const mutate = (fn: (next: PipelineTemplate) => void) => {
    setDraft((current) => {
      const next = structuredClone(current);
      fn(next);
      return next;
    });
    if (proposal) {
      setProposal(undefined);
      setNotice("The edit cleared the Chucky proposal confirmation. You are editing a manual draft.");
    } else setNotice("");
    setDiagnostics([]);
    setError(null);
  };

  const validate = async () => {
    setBusy("validate");
    setError(null);
    setNotice("");
    try {
      const record = await validatePipelineTemplate(id, draft);
      showDiagnostics(record.diagnostics);
      if (record.valid) setNotice("Template is valid.");
    } catch {
      setError("Could not validate the template. Try again.");
    } finally {
      setBusy(null);
    }
  };

  const saveDraft = async () => {
    setBusy("save");
    setError(null);
    setNotice("");
    try {
      if (submitted.current) {
        let record;
        try {
          record = await getPipelineTemplate(id);
        } catch (reason) {
          if (!(reason instanceof PipelineAPIError && reason.status === 404)) {
            setError("Could not check the earlier save. Try again to check before saving.");
            return;
          }
        }
        if (record?.id === id && sameTemplate(record.template, submitted.current)) {
          submitted.current = null;
          setExists(true);
          await queryClient.invalidateQueries({ queryKey: PIPELINE_QUERY_KEYS.templates });
          if (sameTemplate(draft, record.template)) {
            const message = proposal ? "Exact Chucky proposal saved." : "Template saved.";
            setNotice(message);
            setProposal(undefined);
            onSaved?.(record.id, message);
          } else setNotice("The earlier save succeeded. Your latest edits are ready to save.");
          return;
        }
        submitted.current = null;
      }

      submitted.current = draft;
      const record = await save.mutateAsync({ id, template: proposal?.payload.template ?? draft, exists });
      submitted.current = null;
      setExists(true);
      setDraft(structuredClone(record.template));
      setDiagnostics(record.diagnostics);
      const message = proposal ? "Exact Chucky proposal saved." : "Template saved.";
      setNotice(message);
      setProposal(undefined);
      onSaved?.(record.id, message);
    } catch (reason) {
      const items = pipelineDiagnostics(reason);
      showDiagnostics(items);
      if (reason instanceof PipelineAPIError && reason.status < 500) {
        submitted.current = null;
        if (reason.status === 409 && !exists && !seed) {
          setID(newPipelineKey("pipeline"));
          setError("This draft conflicted with another template. Save again to create a new template.");
        } else if (reason.status === 409) {
          setError("A template for this proposal already exists. Reopen the proposal from Templates to review it again.");
        } else if (reason.status === 404 && exists) {
          setError("This template was removed. Your draft is still here.");
        } else if (items.length === 0) setError("Could not save the template. Try again.");
      } else setError("Could not confirm the save. Save again to check whether it went through.");
    } finally {
      setBusy(null);
    }
  };

  const deleteDraft = () => {
    if (!exists) return;
    remove.mutate(id, {
      onSuccess: onDeleted,
      onError: () => setError("Could not delete the template. Try again."),
    });
  };

  const roleSelect = (value: string, field: string, set: (value: string) => void) => (
    <select value={value} data-template-field={field} onChange={(event) => set(event.target.value)}>
      {roleOptions.map(([key, label]) => <option key={key} value={key}>{label}</option>)}
      {!roleOptions.some(([key]) => key === value) && <option value={value}>{value ? "Unavailable role" : "Choose role"}</option>}
    </select>
  );

  return (
    <section className="pipeline-template-editor" data-ui="pipeline-template-editor">
      <Link className="pipeline-back-link" to="/pipelines/templates">← All templates</Link>
      <div className="pipeline-editor-heading" data-ui="pipeline-section"><div><h2>{draft.title || (exists ? "Untitled template" : "New template")}</h2></div></div>

      <fieldset ref={form} className="pipeline-template-fields" disabled={Boolean(busy) || remove.isPending} aria-label="Template configuration">
        <label className="form-field">
          <span>Template name</span><input value={draft.title} data-template-field="title" onChange={(event) => mutate((next) => { next.title = event.target.value; })} />
        </label>
        <div className="pipeline-editor-workspace" data-slot="workspace" data-template-field="stages">
          <aside className="pipeline-stage-navigator" data-slot="navigator">
            <div className="pipeline-stage-nav-heading"><h3>Stages</h3>
              <button type="button" disabled={draft.stages.length >= 32} title={draft.stages.length >= 32 ? "A template can have up to 32 stages" : undefined} onClick={() => {
                const added = newStage();
                mutate((next) => next.stages.push(added));
                focusNewStage.current = true;
                setSelectedStage(draft.stages.length);
              }}>Add stage</button>
            </div>
            <ol>{draft.stages.map((item, index) => {
              const requirements = [
                item.approval_after_success ? "Human approval required" : item.required_for_completion ? "Required for completion" : "",
                item.coordination === "think_tank" ? `Think Tank · ${item.think_tank?.participants.length ?? 0} participants` : item.coordination === "dedicated" ? "Dedicated coordinator" : "",
              ].filter(Boolean);
              return <li key={item.id}><button type="button" aria-current={selectedStage === index ? "step" : undefined} className={selectedStage === index ? "pipeline-stage-nav-item pipeline-stage-nav-item-active" : "pipeline-stage-nav-item"} onClick={() => setSelectedStage(index)}>
                <span>{index + 1}</span><span><strong>{labels.get(item.id)}</strong>{requirements.map((requirement) => <small key={requirement}>{requirement}</small>)}</span>
                {visibleDiagnostics.some((diagnostic) => diagnostic.stageIndex === index) && <em aria-label="Has issues">!</em>}
              </button></li>;
            })}</ol>
          </aside>
          <div className="pipeline-stage-focus" data-slot="stage">
            {stage && <article className="pipeline-stage-card" key={stage.id} data-template-field={path}>
              <div className="pipeline-stage-heading">
                <strong>{labels.get(stage.id)}</strong>
                <button type="button" className="ad-button-danger" disabled={draft.stages.length === 1} title={draft.stages.length === 1 ? "A template needs at least one stage" : undefined} onClick={() => {
                  mutate((next) => { next.stages.splice(selectedStage, 1); });
                  setSelectedStage(Math.max(0, selectedStage - 1));
                }}>Remove stage</button>
              </div>
              <label className="form-field"><span>Stage name</span><input value={stage.title} data-template-field={`${path}.title`} onChange={(event) => mutate((next) => { next.stages[selectedStage].title = event.target.value; })} /></label>
              <label className="form-field"><span>Stage instructions</span><AutoGrowTextarea rows={4} value={stage.objective} data-template-field={`${path}.objective`} onChange={(event) => mutate((next) => { next.stages[selectedStage].objective = event.target.value; })} /></label>

              <details className="pipeline-disclosure" data-template-field={`${path}.requirements`}>
                <summary>Completion and approval</summary>
                <div className="pipeline-disclosure-body">
                  <label className="pipeline-check"><input type="checkbox" checked={Boolean(stage.required_for_completion || stage.approval_after_success)} disabled={stage.approval_after_success}
                    onChange={(event) => mutate((next) => { next.stages[selectedStage].required_for_completion = event.target.checked; })} /> Required for completion</label>
                  <p>Chuck needs an accepted successful result for this stage before the run can finish.</p>
                  <label className="pipeline-check"><input type="checkbox" checked={Boolean(stage.approval_after_success)}
                    onChange={(event) => mutate((next) => { next.stages[selectedStage].approval_after_success = event.target.checked; })} /> Require human approval</label>
                  <p>Makes the stage required and pauses the run after a successful visit for your approval.</p>
                </div>
              </details>

              <details className="pipeline-disclosure" data-template-field={`${path}.coordination`}>
                <summary>Coordination</summary>
                <div className="pipeline-disclosure-body">
                  <fieldset className="pipeline-coordination"><legend>Who coordinates this stage?</legend>
                    {([["standing", "Standing owner"], ["dedicated", "Dedicated coordinator"], ["think_tank", "Think Tank"]] as const).map(([value, label]) => (
                      <label className="pipeline-check" key={value}><input type="radio" name="stage-coordination" checked={stage.coordination === value} onChange={() => {
                        const room = value === "think_tank" && !stage.think_tank ? defaultThinkTank(draft.orchestrator_role) : stage.think_tank;
                        mutate((next) => {
                          const target = next.stages[selectedStage];
                          target.coordination = value;
                          if (value !== "dedicated") target.dedicated_role = "";
                          if (value === "think_tank") target.think_tank = room;
                          else delete target.think_tank;
                        });
                      }} /> {label}</label>
                    ))}
                    {stage.coordination === "dedicated" && <label className="form-field"><span>Coordinator role</span>{roleSelect(stage.dedicated_role, `${path}.dedicated_role`, (value) => mutate((next) => { next.stages[selectedStage].dedicated_role = value; }))}</label>}
                  </fieldset>
                  {stage.coordination === "think_tank" && <ThinkTankStageFields stage={stage} stageIndex={selectedStage} roleOptions={roleOptions} mutate={mutate} />}
                </div>
              </details>
              <StageBindings draft={draft} stageIndex={selectedStage} mutate={mutate} />
            </article>}
          </div>
        </div>

        <details className="pipeline-disclosure" data-template-field="orchestration">
          <summary>Orchestration</summary>
          <div className="pipeline-disclosure-body">
            <label className="form-field"><span>Standing orchestrator role</span>{roleSelect(draft.orchestrator_role, "orchestrator_role", (value) => mutate((next) => { next.orchestrator_role = value; }))}</label>
            <label className="form-field"><span>Shared orchestrator instructions</span>
              <AutoGrowTextarea rows={2} value={draft.orchestrator_instructions ?? ""} data-template-field="orchestrator_instructions"
                onChange={(event) => mutate((next) => { next.orchestrator_instructions = event.target.value; })} />
            </label>
          </div>
        </details>

        <details className="pipeline-disclosure pipeline-editor-inputs" data-slot="inputs" data-template-field="inputs">
          <summary>Run inputs{draft.inputs.length > 0 && <span>{draft.inputs.length}</span>}</summary>
          <div className="pipeline-disclosure-body">
            <div className="pipeline-subsection-header">
              <p>Named values collected before a run can start.</p>
              <button type="button" onClick={() => mutate((next) => next.inputs.push({ name: "", description: "", required: true }))}>Add run input</button>
            </div>
            {draft.inputs.map((input, index) => (
              <div className="pipeline-inline-row" key={index}>
                <label className="form-field"><span>Input name</span><input value={input.name} data-template-field={`inputs.${index}.name`} onChange={(event) => mutate((next) => { next.inputs[index].name = event.target.value; })} /></label>
                <label className="form-field"><span>Description</span><input value={input.description} data-template-field={`inputs.${index}.description`} onChange={(event) => mutate((next) => { next.inputs[index].description = event.target.value; })} /></label>
                <label className="pipeline-check"><input type="checkbox" checked={input.required} onChange={(event) => mutate((next) => { next.inputs[index].required = event.target.checked; })} /> Required</label>
                <button type="button" className="ad-button-danger" onClick={() => mutate((next) => { next.inputs.splice(index, 1); })}>Remove input</button>
              </div>
            ))}
          </div>
        </details>
      </fieldset>

      {visibleDiagnostics.length > 0 && <ul className="pipeline-diagnostics" role="alert">
        {visibleDiagnostics.map(({ label, message }, index) => <li key={index}><strong>{label}</strong> — {message}</li>)}
      </ul>}
      {notice && <p className="form-info" role="status">{notice}</p>}
      {error && <p className="form-error" role="alert">{error}</p>}
      {proposal && <details className="pipeline-disclosure"><summary>Exact Chucky proposal</summary><pre className="pipeline-proposal-payload">{JSON.stringify(proposal.payload, null, 2)}</pre></details>}
      <div className="form-actions pipeline-editor-actions" data-slot="actions">
        {exists && !proposal && <button type="button" className="btn-danger" disabled={Boolean(busy) || remove.isPending} onClick={deleteDraft}>Delete</button>}
        <button type="button" disabled={Boolean(busy) || remove.isPending} onClick={() => void validate()}>{busy === "validate" ? "Validating…" : "Validate"}</button>
        <button type="button" className="ad-button-primary" disabled={Boolean(busy) || remove.isPending} onClick={() => void saveDraft()}>
          {busy === "save" ? "Saving…" : proposal ? "Confirm and save exact proposal" : "Save template"}
        </button>
      </div>
    </section>
  );
}
