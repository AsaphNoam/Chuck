import { AutoGrowTextarea } from "../../components/ui";
import { ConfirmDialog } from "../../components/ui/ConfirmDialog";
import { IncludeOtherProjects, useProjectFirst } from "../../components/ui/IncludeOtherProjects";
import { useEffect, useMemo, useRef, useState } from "react";
import { addFiles, type DraftAttachment } from "../../components/chat/attachments";
import * as Dialog from "@radix-ui/react-dialog";
import { useNavigate } from "react-router-dom";
import {
  fetchThinkTankTemplate,
  newCommandID,
  useCreateThinkTank,
  useRetainedCommand,
  useSaveThinkTankTemplate,
  useThinkTankTemplates,
  type ThinkTankLaunch,
  type ThinkTankParticipantInput,
  type ThinkTankTemplate,
} from "../../api/thinkTanks";
import { useBackends, useProjects, useRoles } from "../../api/config";
import { launchSupportFor } from "../../schemas/backends";
import { projectTitle as projectTitleOf } from "../../lib/labels";
import { personaIdentityLabel } from "../../lib/personas";
import { useAgentStore } from "../../store/agentStore";
import { NewAgentModal } from "../launch/NewAgentModal";
import { thinkTankPath } from "./roomText";

const DEFAULT_LIMIT = 3;
const MAX_PARTICIPANTS = 32;
const MAX_GOAL = 8000;
const MAX_TITLE = 120;
const MAX_INSTRUCTIONS = 8000;

type Row = {
  key: string; agentID?: string; launch?: ThinkTankLaunch; label: string; project: string; runtime: string;
  limit: number; mayLeave: boolean; instructions: string;
};

/** clipRunes limits text to max Unicode code points, matching the server's
 *  rune limit; native maxLength counts UTF-16 units instead (FS-21.R63). */
const clipRunes = (text: string, max: number) => {
  const runes = Array.from(text);
  return runes.length > max ? runes.slice(0, max).join("") : text;
};

/** runtimeLabel names an agent's model and effort as its card does. */
const runtimeLabel = (model?: string, effort?: string) => [model, effort].filter(Boolean).join(" · ");

/** ThinkTankSetupDialog starts a room from a non-archived project (FS-21.R27,
 *  R34): a goal, two or more existing or new chat agents from any non-archived
 *  project in a fixed order, each with its own turn limit, permission to leave
 *  (on by default) and optional room-specific instructions, and optional
 *  independent openings and end-only synthesis (both off by default). Goal
 *  files, judge-only synthesis instructions, goal and membership are fixed
 *  once started (R54–R56, R63). A saved template loads every slot as an
 *  editable fresh launch in this project, and the setup can be saved as one
 *  (R60–R62). */
export function ThinkTankSetupDialog({ open, onClose, originProject }: { open: boolean; onClose: () => void; originProject: string }) {
  const navigate = useNavigate();
  const projects = useProjects();
  const { data: roles } = useRoles();
  const { data: backends } = useBackends();
  const agents = useAgentStore((state) => state.agents);
  const create = useCreateThinkTank();
  const templates = useThinkTankTemplates(open);
  const saveTemplate = useSaveThinkTankTemplate();
  const saveCommand = useRetainedCommand();
  const [title, setTitle] = useState("");
  const [goal, setGoal] = useState("");
  const [rows, setRows] = useState<Row[]>([]);
  const [openings, setOpenings] = useState(false);
  const [judge, setJudge] = useState<ThinkTankLaunch | null>(null);
  const [instructions, setInstructions] = useState("");
  const [files, setFiles] = useState<DraftAttachment[]>([]);
  const picker = useRef<HTMLInputElement>(null);
  const [launchFor, setLaunchFor] = useState<"" | "participant" | "judge">("");
  const [editing, setEditing] = useState("");
  const [replacing, setReplacing] = useState("");
  const [replacement, setReplacement] = useState("");
  const returnFocus = useRef("");
  const [pick, setPick] = useState("");
  const [error, setError] = useState("");
  const [commandID, setCommandID] = useState(newCommandID);
  const [templateID, setTemplateID] = useState("");
  const [pendingTemplate, setPendingTemplate] = useState<string | null>(null);
  const [baseline, setBaseline] = useState("");
  const [saving, setSaving] = useState(false);
  const [saveName, setSaveName] = useState("");
  const [saveNote, setSaveNote] = useState("");
  // Each template load takes a token; a late load from a closed, replaced or
  // changed-project dialog no longer matches and is dropped (TS-14.R33).
  const loadToken = useRef(0);
  useEffect(() => { loadToken.current++; }, [open, originProject]);

  const projectTitle = (id: string) => projectTitleOf(projects.data, id);
  const candidates = useMemo(() => Object.values(agents)
    .filter((a) => a.interface === "chat" && !a.archived && !projects.data?.[a.project]?.archived)
    .filter((a) => !rows.some((r) => r.agentID === a.agent_id))
    .sort((a, b) => (a.project + a.name).localeCompare(b.project + b.name)), [agents, projects.data, rows]);
  // Added rows stay in `rows` whatever this list shows (FS-21.R58).
  const choices = useProjectFirst({ items: candidates, project: originProject, selected: pick, clearSelected: () => setPick(""), resetKey: `${open}:${originProject}` });

  /** unavailable names a launch choice the current catalogs no longer offer,
   *  so the row is repaired instead of silently substituted (FS-21.R61). */
  const unavailable = (launch?: ThinkTankLaunch | null): string => {
    if (!launch) return "";
    if (launch.role && roles && !roles[launch.role]) return `Persona “${launch.role}” is no longer available.`;
    const backend = launch.backend ? backends?.backends[launch.backend] : undefined;
    if (launch.backend && backends && !backend) return `Runtime “${launch.backend}” is no longer available.`;
    if (launch.model && backend && !backend.models[launch.model]) return `Model “${launch.model}” is no longer available.`;
    const { effort, fast } = unsupported(launch);
    if (effort) return `${launch.effort} effort is no longer available for ${launch.model}.`;
    if (fast) return `Fast mode is no longer available for ${launch.model}.`;
    return "";
  };
  /** unsupported checks saved effort and fast mode against the model's current
   *  declaration and chat launch support, as launch validation will. */
  const unsupported = (launch: ThinkTankLaunch) => {
    const backend = launch.backend ? backends?.backends[launch.backend] : undefined;
    const model = backend && launch.model ? backend.models[launch.model] : undefined;
    if (!backend || !model) return { effort: false, fast: false };
    const chat = launchSupportFor(backends?.backend_support, backend.type, "chat");
    return {
      effort: !!launch.effort && (!(model.efforts ?? []).includes(launch.effort) || chat?.effort === false),
      fast: !!launch.fast && (!model.fast || chat?.fast === false),
    };
  };
  /** editable drops unavailable runtime choices so the launch editor offers
   *  current defaults for them; an unavailable persona stays flagged there. */
  const editable = (launch: ThinkTankLaunch): ThinkTankLaunch => {
    if (!backends) return launch;
    const backend = launch.backend ? backends?.backends[launch.backend] : undefined;
    if (!backend) return { ...launch, backend: "", model: "", effort: "", fast: false };
    if (launch.model && !backend.models[launch.model]) return { ...launch, model: "", effort: "", fast: false };
    const { effort, fast } = unsupported(launch);
    return { ...launch, effort: effort ? "" : launch.effort, fast: fast ? false : launch.fast };
  };

  const snapshot = JSON.stringify({ rows: rows.map(({ key: _key, ...r }) => r), openings, judge, instructions });
  const dirty = snapshot !== baseline && (rows.length > 0 || openings || judge !== null || instructions !== "" || baseline !== "");
  // A roster or discussion edit supersedes a pending template load, so a late
  // response never replaces edits made after requesting it (FS-21.R61).
  useEffect(() => { loadToken.current++; }, [snapshot]);

  const update = (key: string, patch: Partial<Row>) => setRows((current) => current.map((r) => r.key === key ? { ...r, ...patch } : r));
  const move = (index: number, delta: number) => setRows((current) => {
    const next = [...current];
    const [row] = next.splice(index, 1);
    next.splice(index + delta, 0, row);
    return next;
  });
  const reset = () => {
    setTitle(""); setGoal(""); setRows([]); setOpenings(false); setJudge(null); setInstructions(""); setFiles([]); setError(""); setPick(""); setCommandID(newCommandID());
    setTemplateID(""); setBaseline(""); setSaving(false); setSaveName(""); setSaveNote("");
  };
  const attach = (list: FileList) => {
    const next = addFiles(files, list);
    setFiles(next.files);
    setError(next.errors[0] ?? "");
    setCommandID(newCommandID());
  };

  const addExisting = () => {
    const agent = agents[pick];
    if (!agent) return;
    setRows((current) => [...current, { key: agent.agent_id, agentID: agent.agent_id, label: agent.name, project: agent.project, runtime: runtimeLabel(agent.model, agent.effort), limit: DEFAULT_LIMIT, mayLeave: true, instructions: "" }]);
    setPick("");
  };

  /** applyTemplate fills roster and discussion settings as fresh launches in
   *  this dialog's project, keeping title, goal and files (FS-21.R61). */
  const applyTemplate = (t: ThinkTankTemplate | null) => {
    const launchOf = (l: ThinkTankTemplate["setup"]["participants"][number]["launch"]): ThinkTankLaunch =>
      ({ role: l.role, project: originProject, backend: l.backend, model: l.model, effort: l.effort, fast: l.fast, name: l.name || undefined });
    const nextRows: Row[] = (t?.setup.participants ?? []).map((p) => ({
      key: newCommandID(), launch: launchOf(p.launch), label: p.launch.name || "New agent", project: originProject,
      runtime: runtimeLabel(p.launch.model, p.launch.effort), limit: p.limit, mayLeave: p.may_leave, instructions: p.instructions,
    }));
    const nextJudge = t?.setup.judge ? launchOf(t.setup.judge) : null;
    const nextInstructions = t?.setup.judge_instructions ?? "";
    const nextOpenings = t?.setup.openings ?? false;
    setRows(nextRows); setOpenings(nextOpenings); setJudge(nextJudge); setInstructions(nextInstructions);
    setBaseline(t ? JSON.stringify({ rows: nextRows.map(({ key: _key, ...r }) => r), openings: nextOpenings, judge: nextJudge, instructions: nextInstructions }) : "");
    setTemplateID(t?.id ?? "");
    setCommandID(newCommandID());
  };
  const chooseTemplate = (id: string) => {
    setPendingTemplate(null);
    setError("");
    if (!id) {
      loadToken.current++;
      applyTemplate(null);
      return;
    }
    const token = ++loadToken.current;
    fetchThinkTankTemplate(id).then((t) => {
      if (token === loadToken.current) applyTemplate(t);
    }, (err) => {
      if (token === loadToken.current) setError(err instanceof Error ? `The template could not be loaded: ${err.message}` : "The template could not be loaded.");
    });
  };
  const requestTemplate = (id: string) => {
    if (dirty) setPendingTemplate(id);
    else chooseTemplate(id);
  };

  const rowProblem = rows.some((r) => unavailable(r.launch)) || unavailable(judge);
  const limitsProblem = rows.some((r) => !Number.isInteger(r.limit) || r.limit < 1 || r.limit > 1000);
  const problems = [
    !title.trim() && "Give the Think Tank a title.",
    !goal.trim() && "Describe the goal.",
    rows.length < 2 && "Add at least two participants.",
    limitsProblem && "Turn limits are whole numbers from 1 to 1000.",
    rowProblem && "Edit or remove the participants with unavailable settings.",
  ].filter(Boolean) as string[];

  const participantInputs = (): ThinkTankParticipantInput[] =>
    rows.map((r) => ({ agent_id: r.agentID, new: r.launch, limit: r.limit, may_leave: r.mayLeave, instructions: r.instructions.trim() || undefined }));

  const submit = () => {
    setError("");
    if (problems.length) {
      setError(problems[0]);
      return;
    }
    create.mutate({
      command_id: commandID,
      title,
      goal,
      origin_project: originProject,
      openings,
      judge: judge ?? undefined,
      judge_instructions: judge && instructions.trim() ? instructions : undefined,
      files,
      participants: participantInputs(),
    }, {
      onSuccess: (room) => {
        reset();
        onClose();
        navigate(thinkTankPath(room.room_id));
      },
      onError: (err) => setError(err instanceof Error ? err.message : "The Think Tank could not be started."),
    });
  };

  const saveSetup = () => {
    setError("");
    setSaveNote("");
    const problem = rows.length < 2 ? "Add at least two participants." : limitsProblem ? "Turn limits are whole numbers from 1 to 1000." : !saveName.trim() ? "Name the template." : "";
    if (problem) {
      setError(problem);
      return;
    }
    const intent = {
      name: saveName,
      setup: { participants: participantInputs(), openings, judge: judge ?? undefined, judge_instructions: judge && instructions.trim() ? instructions : undefined },
    };
    saveTemplate.mutate({ command_id: saveCommand.idFor(intent), ...intent }, {
      onSuccess: (saved) => { saveCommand.acknowledge(); setSaving(false); setSaveNote(`Saved “${saved.name}”.`); },
      onError: (err) => setError(err instanceof Error ? `The setup could not be saved: ${err.message}` : "The setup could not be saved."),
    });
  };

  const editRow = rows.find((r) => r.key === editing);
  const closeEdit = () => {
    returnFocus.current = editing;
    setEditing("");
  };

  return (
    <>
      <Dialog.Root open={open && !launchFor && !editing} onOpenChange={(o) => { if (!o) onClose(); }}>
        <Dialog.Portal>
          <Dialog.Overlay className="dialog-overlay" data-ui="dialog" data-slot="overlay" />
          <Dialog.Content
            className="dialog-content think-tank-setup"
            data-ui="dialog"
            data-slot="content"
            data-variant="default"
            onOpenAutoFocus={(event) => {
              // Closing a row's launch editor returns focus to that row (FS-21.R62).
              const target = returnFocus.current && document.getElementById(`think-tank-edit-${returnFocus.current}`);
              returnFocus.current = "";
              if (target) {
                event.preventDefault();
                target.focus();
              }
            }}
          >
            <Dialog.Title>New Think Tank</Dialog.Title>
            <Dialog.Description>Independent agents discuss one goal in turn. Each keeps its own conversation.</Dialog.Description>
            <form className="config-form" onSubmit={(event) => { event.preventDefault(); submit(); }}>
              <div className="form-field">
                <label htmlFor="think-tank-template">Template</label>
                <select id="think-tank-template" value={templateID} onChange={(e) => requestTemplate(e.target.value)}>
                  <option value="">Blank setup</option>
                  {(templates.templates ?? []).map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
                </select>
                {templates.hasNextPage && (
                  <button type="button" className="ad-button-secondary" onClick={() => void templates.fetchNextPage()} disabled={templates.isFetchingNextPage}>
                    {templates.isFetchingNextPage ? "Loading templates…" : "Show more templates"}
                  </button>
                )}
                {templates.isError && <p className="form-hint">Saved templates are unavailable right now.</p>}
              </div>
              <div className="form-field">
                <label htmlFor="think-tank-title">Title</label>
                <input id="think-tank-title" value={title} maxLength={MAX_TITLE} onChange={(e) => { setTitle(e.target.value); setCommandID(newCommandID()); }} placeholder="A short name for this Think Tank" />
              </div>
              <div className="form-field">
                <label htmlFor="think-tank-goal">Goal</label>
                <AutoGrowTextarea id="think-tank-goal" value={goal} maxLength={MAX_GOAL} rows={3} onChange={(e) => { setGoal(e.target.value); setCommandID(newCommandID()); }} placeholder="What should the participants work out?" />
              </div>
              <div className="form-field think-tank-setup-files">
                <input ref={picker} className="ad-visually-hidden" type="file" multiple aria-label="Choose files to attach" onChange={(e) => { if (e.target.files) attach(e.target.files); e.target.value = ""; }} />
                <button type="button" className="ad-button-primary" onClick={() => picker.current?.click()}>Attach files…</button>
                {files.length > 0 && (
                  <ul aria-label="Attached files">
                    {files.map((f) => (
                      <li key={f.id}>
                        <span>{f.name}</span>
                        <button type="button" className="ad-button-icon" aria-label={`Remove ${f.name}`} onClick={() => { setFiles((c) => c.filter((x) => x.id !== f.id)); setCommandID(newCommandID()); }}>×</button>
                      </li>
                    ))}
                  </ul>
                )}
                <p className="form-hint">Every participant and the judge can read these with the goal.</p>
              </div>
              <fieldset className="think-tank-setup-participants">
                <legend>Participants, in speaking order</legend>
                {rows.length === 0 && <p className="form-hint">Add existing agents from any active project, or new ones.</p>}
                <ol>
                  {rows.map((r, index) => {
                    const problem = unavailable(r.launch);
                    return (
                      <li key={r.key} className="think-tank-setup-row" data-unavailable={problem ? "true" : undefined}>
                        <div className="think-tank-setup-name"><strong>{r.label}</strong><span>{projectTitle(r.project)}{r.launch ? " · new agent" : ""}</span>{r.runtime && <span className="think-tank-setup-runtime">{r.runtime}</span>}</div>
                        {problem && <p className="form-error think-tank-setup-problem">{problem} Edit or remove this participant.</p>}
                        <label>Turn limit<input type="number" min={1} max={1000} value={r.limit} onChange={(e) => update(r.key, { limit: Number(e.target.value) })} /></label>
                        <label className="think-tank-check"><input type="checkbox" checked={r.mayLeave} onChange={(e) => update(r.key, { mayLeave: e.target.checked })} /> May leave</label>
                        <div className="think-tank-setup-order">
                          <button type="button" className="ad-button-icon" disabled={index === 0} onClick={() => move(index, -1)} aria-label={`Move ${r.label} earlier`}>↑</button>
                          <button type="button" className="ad-button-icon" disabled={index === rows.length - 1} onClick={() => move(index, 1)} aria-label={`Move ${r.label} later`}>↓</button>
                          <button type="button" id={`think-tank-edit-${r.key}`} aria-label={`Edit ${r.label}`}
                            onClick={() => { if (r.launch) setEditing(r.key); else { setReplacing(r.key); setReplacement(""); } }}>Edit</button>
                          <button type="button" className="ad-button-danger" onClick={() => setRows((c) => c.filter((x) => x.key !== r.key))}>Remove</button>
                        </div>
                        {replacing === r.key && (
                          <div className="think-tank-setup-add think-tank-setup-replace">
                            <select value={replacement} onChange={(e) => setReplacement(e.target.value)} aria-label={`Replace ${r.label} with`}>
                              <option value="">Replace with another agent</option>
                              {choices.visible.map((a) => <option key={a.agent_id} value={a.agent_id}>{`${a.name} (${projectTitle(a.project)})`}</option>)}
                            </select>
                            <button type="button" disabled={!replacement} onClick={() => {
                              const agent = agents[replacement];
                              if (!agent) return;
                              update(r.key, { agentID: agent.agent_id, label: agent.name, project: agent.project, runtime: runtimeLabel(agent.model, agent.effort) });
                              setReplacing("");
                              document.getElementById(`think-tank-edit-${r.key}`)?.focus();
                            }}>Replace</button>
                            <button type="button" onClick={() => { setReplacing(""); document.getElementById(`think-tank-edit-${r.key}`)?.focus(); }}>Cancel</button>
                          </div>
                        )}
                        <details className="think-tank-setup-instructions" open={r.instructions !== "" || undefined}>
                          <summary>Instructions for {r.label}</summary>
                          <AutoGrowTextarea aria-label={`Instructions for ${r.label}`} value={r.instructions} rows={2}
                            onChange={(e) => { update(r.key, { instructions: clipRunes(e.target.value, MAX_INSTRUCTIONS) }); setCommandID(newCommandID()); }}
                            placeholder="Optional: direction for this participant in this room only." />
                        </details>
                      </li>
                    );
                  })}
                </ol>
                {rows.length < MAX_PARTICIPANTS && (
                  <>
                    <button type="button" className="ad-button-primary think-tank-setup-new" onClick={() => setLaunchFor("participant")}>+ Add new agent…</button>
                    <div className="think-tank-setup-add">
                      <select value={pick} onChange={(e) => setPick(e.target.value)} aria-label="Existing agent">
                        <option value="">Or choose an existing agent</option>
                        {choices.visible.map((a) => <option key={a.agent_id} value={a.agent_id}>{[`${a.name} (${projectTitle(a.project)})`, runtimeLabel(a.model, a.effort), a.state === "busy" && "busy"].filter(Boolean).join(" · ")}</option>)}
                      </select>
                      <button type="button" onClick={addExisting} disabled={!pick}>Add</button>
                    </div>
                    <IncludeOtherProjects className="think-tank-check" checked={choices.includeOthers} onChange={choices.setIncludeOthers} />
                  </>
                )}
                <p className="form-hint">A limit is a ceiling, not a target. Busy agents join without being interrupted. Only each participant sees its own instructions.</p>
              </fieldset>
              <label className="think-tank-check"><input type="checkbox" checked={openings} onChange={(e) => setOpenings(e.target.checked)} /> Independent openings — each participant answers before seeing the others</label>
              <div className="think-tank-check">
                <label><input type="checkbox" checked={judge !== null} onChange={(e) => e.target.checked ? setLaunchFor("judge") : setJudge(null)} /> Final synthesis by a fresh judge after the discussion ends</label>
                {judge && <span className="form-hint">{judge.name || personaIdentityLabel(judge.role ?? "", roles?.[judge.role ?? ""])} in {projectTitle(judge.project)}{runtimeLabel(judge.model, judge.effort) && ` · ${runtimeLabel(judge.model, judge.effort)}`} · <button type="button" className="ad-button-secondary" onClick={() => setLaunchFor("judge")}>Change</button></span>}
                {unavailable(judge) && <p className="form-error">{unavailable(judge)} Change the judge settings.</p>}
              </div>
              {judge && (
                <div className="form-field">
                  <label htmlFor="think-tank-judge-instructions">Synthesis instructions</label>
                  <AutoGrowTextarea id="think-tank-judge-instructions" value={instructions} maxLength={MAX_GOAL} rows={2} onChange={(e) => { setInstructions(e.target.value); setCommandID(newCommandID()); }} placeholder="Optional: how the judge should act and respond, e.g. “Recommend one option and list the risks.”" />
                  <p className="form-hint">Only the judge receives these.</p>
                </div>
              )}
              <div className="form-field think-tank-setup-save">
                {saving ? (
                  <div className="think-tank-setup-add">
                    <input aria-label="Template name" value={saveName} maxLength={MAX_TITLE} onChange={(e) => setSaveName(e.target.value)} placeholder="Template name"
                      onKeyDown={(e) => { if (e.key === "Enter") { e.preventDefault(); saveSetup(); } }} autoFocus />
                    <button type="button" onClick={saveSetup} disabled={saveTemplate.isPending}>{saveTemplate.isPending ? "Saving…" : "Save"}</button>
                    <button type="button" onClick={() => setSaving(false)} disabled={saveTemplate.isPending}>Cancel</button>
                  </div>
                ) : (
                  <button type="button" onClick={() => { setSaving(true); setSaveNote(""); setSaveName((n) => n || title.trim()); }}>Save setup…</button>
                )}
                {saveNote && <p className="form-hint" role="status">{saveNote}</p>}
                <p className="form-hint">Saves the participants and discussion settings, not the title, goal or files.</p>
              </div>
              {error && <p className="form-error" role="alert">{error}</p>}
              <div className="form-actions">
                <button type="button" onClick={onClose} disabled={create.isPending}>Cancel</button>
                <button type="submit" className="ad-button-primary" disabled={create.isPending}>{create.isPending ? "Starting…" : "Start Think Tank"}</button>
              </div>
            </form>
          </Dialog.Content>
        </Dialog.Portal>
      </Dialog.Root>
      <ConfirmDialog
        open={open && pendingTemplate !== null}
        title="Replace the current setup?"
        confirmLabel="Replace setup"
        onCancel={() => setPendingTemplate(null)}
        onConfirm={() => chooseTemplate(pendingTemplate ?? "")}
      >
        <p>Your participant and discussion changes will be replaced. The title, goal and files stay.</p>
      </ConfirmDialog>
      <NewAgentModal
        open={launchFor !== ""}
        groupPicker={false}
        title={launchFor === "judge" ? "Judge settings" : "New participant"}
        initialProject={originProject}
        onClose={() => setLaunchFor("")}
        onConfigure={(params) => {
          const launch: ThinkTankLaunch = { role: params.role, project: params.project, backend: params.backend, model: params.model, effort: params.effort, fast: params.fast, name: params.name };
          if (launchFor === "judge") setJudge(launch);
          else setRows((current) => [...current, { key: newCommandID(), launch, label: params.name || "New agent", project: params.project, runtime: runtimeLabel(params.model, params.effort), limit: DEFAULT_LIMIT, mayLeave: true, instructions: "" }]);
          setCommandID(newCommandID());
        }}
      />
      {editRow?.launch && (
        <NewAgentModal
          key={editRow.key}
          open
          groupPicker={false}
          title={`Edit ${editRow.label}`}
          configureLabel="Apply changes"
          initialProject={editRow.launch.project}
          initialLaunch={editable(editRow.launch)}
          onClose={closeEdit}
          onConfigure={(params) => {
            const launch: ThinkTankLaunch = { role: params.role, project: params.project, backend: params.backend, model: params.model, effort: params.effort, fast: params.fast, name: params.name };
            update(editRow.key, { launch, label: params.name || "New agent", project: params.project, runtime: runtimeLabel(params.model, params.effort) });
            setCommandID(newCommandID());
          }}
        />
      )}
    </>
  );
}
