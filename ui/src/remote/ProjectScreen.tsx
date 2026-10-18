import { AutoGrowTextarea } from "../components/ui";
import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { listPipelineTemplates, pipelineDiagnostics, sharedWorkspaceConflicts, startPipelineRun } from "../api/pipelines";
import type { AgentState } from "../api/types";
import { deriveDashboardProjects } from "../features/dashboard/projectDashboardData";
import { getHome, phoneFetch } from "./api";
import { useConnection } from "./connection";
import { navigate } from "./router";
import { PhoneIcon } from "./PhoneIcon";
import { PhoneSheet } from "./PhoneSheet";
import { useGroupActions } from "../components/groups/GroupActions";
import { groupAgents } from "../lib/groups";
import { personaLabel } from "../lib/personas";
import { AgentActions, AgentPress } from "./AgentActions";
import { NewAgentForm } from "./NewAgentForm";
import { usePhoneGroupChoices } from "./groupChoices";

const errorText = (error: unknown) => error instanceof Error ? error.message : String(error);

function StartPipeline({ project, onClose }: { project: string; onClose: () => void }) {
  const offline = useConnection((state) => state.link !== "connected"); const templates = useQuery({ queryKey: ["templates"], queryFn: listPipelineTemplates });
  const valid = (templates.data ?? []).filter((template) => template.valid); const [templateID, setTemplateID] = useState(""); const template = valid.find((item) => item.id === templateID) ?? valid[0];
  const [name, setName] = useState(""); const [goal, setGoal] = useState(""); const [inputs, setInputs] = useState<Record<string, string>>({}); const [busy, setBusy] = useState(false); const [error, setError] = useState(""); const [conflicts, setConflicts] = useState<{ name: string }[]>([]);
  const submit = async (acknowledge = false) => { if (!template) return; setBusy(true); setError(""); try { const result = await startPipelineRun({ request_id: crypto.randomUUID(), template_id: template.id, display_name: name.trim() || template.template.title, project, goal, inputs, orchestrator: { backend: "", model: "", effort: "", fast: false }, dedicated_assignments: {} }, acknowledge); navigate(`/run/${encodeURIComponent(result.run.run.run_id)}`); } catch (err) { const shared = sharedWorkspaceConflicts(err); if (shared.length) setConflicts(shared); else setError([errorText(err), ...pipelineDiagnostics(err).map((item) => item.message)].join(": ")); } finally { setBusy(false); } };
  if (!template) return <p className="phone-empty">No valid pipeline templates. Create one on the Mac.</p>;
  return <form className="phone-card phone-form" aria-label="Start pipeline" onSubmit={(event) => { event.preventDefault(); void submit(); }}><h2>Start pipeline</h2><label className="phone-field">Template<select value={template.id} onChange={(event) => setTemplateID(event.target.value)}>{valid.map((item) => <option key={item.id} value={item.id}>{item.template.title}</option>)}</select></label><label className="phone-field">Display name <input value={name} placeholder={template.template.title} onChange={(event) => setName(event.target.value)} /></label><label className="phone-field">Run goal<AutoGrowTextarea rows={3} value={goal} onChange={(event) => setGoal(event.target.value)} /></label>{template.template.inputs.map((input) => <label key={input.name} className="phone-field">{input.name}{input.required ? "" : " (optional)"}<input value={inputs[input.name] ?? ""} onChange={(event) => setInputs({ ...inputs, [input.name]: event.target.value })} /></label>)}{error && <p className="phone-error">{error}</p>}{conflicts.length > 0 && <div className="phone-card"><p>Other work is using this workspace: {conflicts.map((item) => item.name).join(", ")}.</p><button type="button" disabled={offline || busy} onClick={() => void submit(true)}>Start anyway</button></div>}<div className="phone-actions"><button type="button" onClick={onClose}>Cancel</button><button className="ad-button-primary" type="submit" disabled={offline || busy || !goal.trim()}>Start pipeline</button></div></form>;
}

export function ProjectScreen({ projectID }: { projectID: string }) {
  const agents = useConnection((state) => state.agents); const revision = useConnection((state) => state.revision); const [form, setForm] = useState<"agent" | "pipeline" | null>(null); const [newGroup, setNewGroup] = useState("");
  const offline = useConnection((state) => state.link !== "connected");
  const roleCatalog = useQuery({ queryKey: ["roles", revision], queryFn: () => phoneFetch<Record<string, { title: string; system_prompt?: string; skip_permissions?: boolean | null; chuck_aware?: boolean | null }>>("/api/roles"), placeholderData: (previous) => previous });
  // Project configuration follows the live revision so desktop archival reaches an open screen (FS-20.R39).
  const projects = useQuery({ queryKey: ["projects", revision], queryFn: () => phoneFetch<Record<string, { title: string; color: [number, number, number]; archived?: boolean }>>("/api/projects"), placeholderData: (previous) => previous });
  const home = useQuery({ queryKey: ["home", revision], queryFn: getHome, placeholderData: (previous) => previous });
  const project = useMemo(() => deriveDashboardProjects(projects.data, agents).find((item) => item.id === projectID), [agents, projectID, projects.data]);
  const groupActions = useGroupActions({ project: projectID, projectTitle: project?.title, disabled: offline });
  const groups = useMemo(() => groupAgents(project?.agents ?? []), [project?.agents]);
  const choices = usePhoneGroupChoices((state) => state.choices);
  const toggleGroup = usePhoneGroupChoices((state) => state.toggle);
  const [actionAgent, setActionAgent] = useState<string | null>(null);
  if (projects.data?.[projectID]?.archived) return <p className="phone-empty">Archived on the Mac. <button type="button" className="phone-link ad-button-secondary" onClick={() => navigate("/")}>Home</button></p>;
  if (!project) return <p className="phone-empty">{projects.data ? "This project is no longer on the Mac." : "Loading…"} <button type="button" className="phone-link ad-button-secondary" onClick={() => navigate("/")}>Home</button></p>;
  const runs = (home.data?.active_runs ?? []).filter((run) => run.project === projectID);
  const roleLabel = (id: string) => roleCatalog.data?.[id] ? personaLabel(id, roleCatalog.data[id]) : id;
  return <div className="phone-project">
    <header className="phone-page-heading"><span className="phone-eyebrow">Project</span><h1>{project.title}</h1><p>{project.agents.length} agents · {project.stateSummary}</p></header>
    <div className="phone-actions"><button type="button" className="ad-button-primary" onClick={() => { setNewGroup(""); setForm("agent"); }}><PhoneIcon name="plus" size={16} />New agent</button><button type="button" onClick={() => setForm("pipeline")}><PhoneIcon name="pipeline" size={16} />Start pipeline</button></div>
    {form && <PhoneSheet title={form === "agent" ? "New agent" : "Start pipeline"} centered={form === "agent"} onClose={() => setForm(null)}>{form === "agent" ? <NewAgentForm project={projectID} initialGroup={newGroup} onClose={() => setForm(null)} /> : <StartPipeline project={projectID} onClose={() => setForm(null)} />}</PhoneSheet>}
    {groupActions.feedback}
    {actionAgent && <AgentActions agentId={actionAgent} onClose={() => setActionAgent(null)} />}
    {runs.length > 0 && <section className="phone-section" aria-label="Active runs"><div className="phone-section-title"><h2>Active pipelines</h2><span className="phone-count">{runs.length} runs</span></div><ul className="phone-list">{runs.map((run) => <li key={run.id}><button className="phone-row phone-run-card" type="button" onClick={() => navigate(`/run/${encodeURIComponent(run.id)}`)}><span className="phone-attention-meta"><span><PhoneIcon name="pipeline" size={16} />Pipeline</span><PhoneIcon name="arrow" size={17} /></span><span className="phone-row-title">{run.title}</span><span className="phone-row-reason">{run.reason}</span>{run.stage_number && run.stage_count ? <span className="phone-row-meta">Stage {run.stage_number} of {run.stage_count}</span> : null}</button></li>)}</ul></section>}
    <section className="phone-section" aria-label="Agents"><div className="phone-section-title"><h2>Agents</h2><span className="phone-count">{project.agents.length} agents</span></div>{project.agents.length === 0 ? <p className="phone-empty">No agents yet.</p> : <div className="phone-agent-groups">{groups.map((group) => { const isCollapsed = choices[JSON.stringify([projectID, group.key])] ?? false; const running = group.agents.filter((agent) => agent.running).length; return <section className="phone-agent-group" key={group.key} aria-label={`${group.label} group`}><header className="phone-agent-group-header"><button type="button" aria-expanded={!isCollapsed} onClick={() => toggleGroup(projectID, group.key)}><strong>{group.label}</strong><span>{group.agents.length} agents · {running} running</span></button><button type="button" className="phone-group-new" aria-label={`New agent in ${group.label}`} title={`New agent in ${group.label}`} disabled={offline} onClick={() => { setNewGroup(group.key === "_ungrouped" ? "" : group.key); setForm("agent"); }}><PhoneIcon name="plus" size={18} /></button>{group.key !== "_ungrouped" && <span className="phone-agent-group-actions"><button type="button" className="ad-button-danger" disabled={offline || groupActions.pending || running === 0} onClick={() => groupActions.open("stop", group.key, group.agents)}>Stop group</button><button type="button" className="ad-button-danger" disabled={offline || groupActions.pending} onClick={() => groupActions.open("archive", group.key, group.agents)}>Archive group</button></span>}</header>{!isCollapsed && <ul className="phone-list phone-agent-list">{group.agents.map((agent: AgentState) => <AgentPress key={agent.agent_id} onOpen={() => setActionAgent(agent.agent_id)}><button type="button" data-tone={agent.running ? agent.state : "stopped"} className={`phone-row phone-agent-row${agent.running ? "" : " phone-row-stopped"}`} onClick={() => navigate(`/agent/${encodeURIComponent(agent.agent_id)}`)}><span className="phone-avatar"><PhoneIcon name="agent" /></span><span className="phone-row-text"><span className="phone-row-title">{agent.name || `${roleLabel(agent.role)}@${project.id}`}</span><span className="phone-row-reason">{agent.detail || roleLabel(agent.role)}</span><span className="phone-row-meta">{roleLabel(agent.role)} · {agent.backend} · {agent.model}</span></span><span className="phone-agent-status"><span className="phone-status" data-tone={agent.running ? agent.state : "stopped"}>{agent.running ? agent.state.replace("_", " ") : "stopped"}</span><PhoneIcon name="arrow" size={16} /></span></button></AgentPress>)}</ul>}</section>; })}</div>}</section>
  </div>;
}
