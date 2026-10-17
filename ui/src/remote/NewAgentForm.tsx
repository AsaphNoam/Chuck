import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { launchAgent } from "../api/client";
import { getRuntimeOptions, phoneFetch } from "./api";
import { useConnection } from "./connection";
import { navigate } from "./router";
import { useSuggestedName } from "../features/launch/useSuggestedName";
import { GroupPicker } from "../components/ui/GroupPicker";
import { isCanonicalNoPersona, personaLabel } from "../lib/personas";
import { displayLabel, displayLabels } from "../lib/labels";

const errorText = (error: unknown) => error instanceof Error ? error.message : String(error);

type Runtime = { backend: string; model: string; effort: string; fast: boolean };

function RuntimeFields({ value, onChange }: { value: Runtime; onChange: (value: Runtime) => void }) {
  const options = useQuery({ queryKey: ["runtime-options"], queryFn: getRuntimeOptions });
  const backends = options.data?.backends ?? [];
  const backend = backends.find((item) => item.id === value.backend) ?? backends.find((item) => item.default) ?? backends[0];
  const model = backend?.models.find((item) => item.id === value.model) ?? backend?.models.find((item) => item.id === backend.default_model) ?? backend?.models[0];
  const pickBackend = (backendID: string) => {
    const next = backends.find((item) => item.id === backendID);
    const nextModel = next?.models.find((item) => item.id === next.default_model) ?? next?.models[0];
    onChange({ backend: backendID, model: nextModel?.id ?? "", effort: nextModel?.default_effort ?? "", fast: false });
  };
  const pickModel = (modelID: string) => {
    const next = backend?.models.find((item) => item.id === modelID);
    onChange({ ...value, backend: backend?.id ?? value.backend, model: modelID, effort: next?.default_effort ?? "", fast: false });
  };
  return <>
    <label className="phone-field">Backend<select value={backend?.id ?? value.backend} onChange={(event) => pickBackend(event.target.value)}><option value="">Choose…</option>{backends.map((item) => <option key={item.id} value={item.id}>{item.name || item.id}</option>)}</select></label>
    <label className="phone-field">Model<select value={model?.id ?? value.model} onChange={(event) => pickModel(event.target.value)}><option value="">Choose…</option>{(backend?.models ?? []).map((item) => <option key={item.id} value={item.id}>{item.name || item.id}</option>)}</select></label>
    {(model?.efforts.length ?? 0) > 0 && <label className="phone-field">Effort<select value={value.effort} onChange={(event) => onChange({ ...value, effort: event.target.value })}><option value="">Model default</option>{model!.efforts.map((effort) => <option key={effort} value={effort}>{effort}</option>)}</select></label>}
    {model?.fast && <label className="phone-check"><input type="checkbox" checked={value.fast} onChange={(event) => onChange({ ...value, fast: event.target.checked })} /> Fast mode</label>}
  </>;
}

/** The phone's normal chat-agent launch (FS-20.R40). A project page fixes the
 *  project and opens the new agent. An annotation New task prefills the
 *  source's persona and project, lets the person choose another active project,
 *  and hands the launched id back instead of navigating (FS-13.R30). */
export function NewAgentForm({ project: initialProject, initialGroup = "", initialRole, chooseProject = false, onClose, onLaunched, summary }: {
  project: string;
  initialGroup?: string;
  initialRole?: string;
  chooseProject?: boolean;
  onClose: () => void;
  onLaunched?: (agentId: string) => void;
  /** What happens after launch, shown under the heading. */
  summary?: string;
}) {
  const offline = useConnection((state) => state.link !== "connected");
  const roles = useQuery({ queryKey: ["roles"], queryFn: () => phoneFetch<Record<string, { title: string; system_prompt?: string; skip_permissions?: boolean | null; chuck_aware?: boolean | null }>>("/api/roles") });
  const config = useQuery({ queryKey: ["config"], queryFn: () => phoneFetch<{ default_role?: string }>("/api/config") });
  const projects = useQuery({ queryKey: ["projects"], queryFn: () => phoneFetch<Record<string, { title: string; archived?: boolean }>>("/api/projects"), enabled: chooseProject });
  const [project, setProject] = useState(initialProject);
  const [role, setRole] = useState(initialRole ?? "");
  const [name, setName] = useSuggestedName(role);
  const [runtime, setRuntime] = useState<Runtime>({ backend: "", model: "", effort: "", fast: false });
  const runtimeOptions = useQuery({ queryKey: ["runtime-options"], queryFn: getRuntimeOptions });
  const selectedBackend = runtimeOptions.data?.backends.find((item) => item.id === runtime.backend)
    ?? runtimeOptions.data?.backends.find((item) => item.default) ?? runtimeOptions.data?.backends[0];
  const selectedModel = selectedBackend?.models.find((item) => item.id === runtime.model)
    ?? selectedBackend?.models.find((item) => item.id === selectedBackend.default_model) ?? selectedBackend?.models[0];
  const allAgents = useConnection((state) => state.agents);
  const groups = useMemo(() => Object.values(allAgents).filter((agent) => !agent.archived).map((agent) => agent.group?.trim()).filter((label): label is string => !!label), [allAgents]);
  const [group, setGroup] = useState(initialGroup);
  const [busy, setBusy] = useState(false); const [error, setError] = useState("");
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const roleEntries = Object.entries(roles.data ?? {});
  const personaLabels = roleEntries.map(([id, value]) => [id, personaLabel(id, value)] as [string, string]);
  const roleID = role;
  const roleInvalid = Boolean(role && roles.data && !roles.data[role]);
  // Only active configured projects can receive a launch.
  const projectChoices = Object.entries(projects.data ?? {}).filter(([id, value]) => !value.archived || id === project);
  const projectInvalid = chooseProject && !!projects.data && (!projects.data[project] || !!projects.data[project].archived);
  useEffect(() => { if (roleInvalid || projectInvalid) setAdvancedOpen(true); }, [roleInvalid, projectInvalid]);
  useEffect(() => {
    if (role || !roles.data || (!config.data && !config.isError)) return;
    const configured = config.data?.default_role;
    if (configured && roles.data[configured]) setRole(configured);
    else if (roleEntries.some(([id, value]) => isCanonicalNoPersona(id, value))) setRole("default");
    else setRole(roleEntries[0]?.[0] ?? "");
  }, [config.data, config.isError, role, roleEntries, roles.data]);
  const submit = async () => {
    if (!roleID || roleInvalid || projectInvalid) return;
    setBusy(true); setError("");
    try {
      const result = await launchAgent({ project, role: roleID, name: name.trim() || undefined, group: group.trim() || undefined, ...runtime });
      if (onLaunched) onLaunched(result.agent.agent_id);
      else navigate(`/agent/${encodeURIComponent(result.agent.agent_id)}`);
    } catch (err) { setError(errorText(err)); setAdvancedOpen(true); }
    finally { setBusy(false); }
  };
  const projectLabel = projects.data?.[project]?.title || project;
  return <form className="phone-card phone-form" aria-label="New agent" onSubmit={(event) => { event.preventDefault(); void submit(); }}>
    <h2>New agent</h2>
    <p className="phone-meta">{roleID ? displayLabel(personaLabels, roleID) || roleID : "Loading persona…"} · {selectedBackend?.name || "Configured backend"} · {selectedModel?.name || "Default model"} · Chat{runtime.fast ? " · Fast mode" : ""}{chooseProject ? ` · ${projectLabel}` : ""}{group.trim() ? ` · ${group.trim()}` : ""}</p>
    {summary && <p className="phone-meta">{summary}</p>}
    <details open={advancedOpen} onToggle={(event) => setAdvancedOpen(event.currentTarget.open)}>
      <summary>Advanced</summary>
      {chooseProject && <label className="phone-field">Project<select value={project} onChange={(event) => setProject(event.target.value)}>
        {!projects.data?.[project] && <option value={project}>{project}{projects.data ? " (unavailable)" : ""}</option>}
        {projectChoices.map(([id, value]) => <option key={id} value={id}>{value.title || id}{value.archived ? " (archived)" : ""}</option>)}
      </select></label>}
      {projectInvalid && <p className="phone-error">This project is unavailable. Choose another project.</p>}
      <label className="phone-field">Persona<select value={roleID} onChange={(event) => setRole(event.target.value)}>
        {roleInvalid && <option value={roleID}>{roleID} (unavailable)</option>}
        {displayLabels(personaLabels).map(([id, label]) => <option key={id} value={id}>{label}</option>)}
      </select></label>
      {roleInvalid && <p className="phone-error">This persona is unavailable. Choose another persona.</p>}
      <label className="phone-field">Name <input value={name} onChange={(event) => setName(event.target.value)} /></label>
      <label className="phone-field">Group<GroupPicker id="phone-new-agent-group" value={group} groups={groups} onChange={setGroup} /></label>
      <RuntimeFields value={runtime} onChange={setRuntime} />
    </details>
    {error && <p className="phone-error" role="alert">{error}</p>}
    <div className="phone-actions"><button type="button" onClick={onClose} disabled={busy}>Cancel</button><button className="ad-button-primary" type="submit" disabled={offline || busy || !roleID || roleInvalid || projectInvalid}>{onLaunched ? "Launch and send" : "Create agent"}</button></div>
  </form>;
}
