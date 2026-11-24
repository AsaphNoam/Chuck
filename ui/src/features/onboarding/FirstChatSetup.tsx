import { useCallback, useEffect, useRef, useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { Link } from "react-router-dom";
import { useBackends, useConfig, useCreateProject, useLaunchAgent, useProjects, usePutConfig, useRefreshProvider, configErrorMessage } from "../../api/config";
import type { Backend, RefreshProviderResponse } from "../../schemas/backends";
import { BrowseDirectoryButton } from "../../components/ui";
import { DEFAULT_PROJECT_COLOR } from "../../lib/projectColors";
import { describeProviderRuntime, providerInstallURL, providerName } from "../../lib/providerRuntime";
import { backendSettingsHref, isProviderSetupErrorCode } from "../settings/backendSettingsLink";
import { getCurrentSessions } from "../../api/client";

type Check = { pending: boolean; result?: RefreshProviderResponse; error?: string; identity?: string };

function readinessIdentity(backend: Backend): string {
  return JSON.stringify({ type: backend.type, provider_mode: backend.provider_mode ?? "installed", env: backend.env ?? {}, default_model: backend.default_model, model: backend.models[backend.default_model] });
}

export function FirstChatSetup({ onComplete }: { onComplete: (agentId?: string) => void }) {
  const { data: catalog, refetch: refetchCatalog } = useBackends();
  const { data: config, refetch: refetchConfig } = useConfig();
  const projectsQuery = useProjects();
  const { mutateAsync: refreshProvider } = useRefreshProvider();
  const create = useCreateProject();
  const launch = useLaunchAgent();
  const complete = usePutConfig();
  const [checks, setChecks] = useState<Record<string, Check>>({});
  const generations = useRef<Record<string, number>>({});
  const started = useRef(false);
  const savedCatalog = useRef(catalog);
  savedCatalog.current = catalog;
  const [backend, setBackend] = useState("");
  const [project, setProject] = useState("");
  const projectChosen = useRef(false);
  const [folder, setFolder] = useState("");
  const [createdProject, setCreatedProject] = useState("");
  const [projectNeedsRepair, setProjectNeedsRepair] = useState(false);
  const [launched, setLaunched] = useState<{ id: string; backend: string; project: string; etag?: string }>();
  const [error, setError] = useState<string>();
  const [launchRepair, setLaunchRepair] = useState<string>();
  const [uncertain, setUncertain] = useState(false);
  const [recoveryChats, setRecoveryChats] = useState<Array<{ agent: { agent_id: string; name: string } }>>([]);
  const [pending, setPending] = useState(false);
  const claimed = useRef(false);
  const repairTarget = useRef<string>();
  const eligible = (config?.onboarding.launchable_projects ?? []).filter((id) => projectsQuery.data?.[id] && !projectsQuery.data[id].archived);

  const check = useCallback(async (id: string, bundled = false) => {
    const current = savedCatalog.current;
    const generation = (generations.current[id] ?? 0) + 1;
    generations.current[id] = generation;
    if (!current?.backends[id]) {
      setChecks((previous) => ({ ...previous, [id]: { pending: false, error: "This backend is unavailable. Choose another ready provider." } }));
      return;
    }
    setChecks((previous) => ({ ...previous, [id]: { pending: true } }));
    try {
      const result = await refreshProvider({ backendId: id, catalogEtag: current.catalogEtag, ...(bundled ? { backend: { ...current.backends[id], provider_mode: "bundled" as const } } : { checkOnly: true }) });
      if (generations.current[id] !== generation) return;
      setChecks((previous) => ({ ...previous, [id]: { pending: false, result, identity: readinessIdentity(result.backend ?? current.backends[id]) } }));
    } catch (failure) {
      if (generations.current[id] !== generation) return;
      setChecks((previous) => ({ ...previous, [id]: { pending: false, error: configErrorMessage(failure) } }));
    }
  }, [refreshProvider]);

  useEffect(() => {
    if (!catalog || started.current) return;
    started.current = true;
    const knownReady = ["claude", "codex"].filter((id) => {
      const entry = catalog.backends[id];
      return entry && catalog.credentials?.[id]?.status === "ok" && catalog.provider_runtimes[id]?.[entry.default_model]?.state === "available";
    });
    const preferred = knownReady.find((id) => catalog.backends[id].default) ?? knownReady[0];
    if (preferred) setBackend((previous) => previous || preferred);
    for (const id of ["claude", "codex"]) {
      if (catalog.backends[id]) void check(id);
    }
  }, [catalog, check]);

  useEffect(() => {
    const refresh = () => {
      if (document.visibilityState === "hidden" || !repairTarget.current) return;
      const id = repairTarget.current;
      repairTarget.current = undefined;
      void refetchCatalog().then((result) => {
        if (result.data) savedCatalog.current = result.data;
        return check(id);
      });
    };
    window.addEventListener("focus", refresh);
    document.addEventListener("visibilitychange", refresh);
    return () => {
      window.removeEventListener("focus", refresh);
      document.removeEventListener("visibilitychange", refresh);
    };
  });

  const ready = useCallback((id: string) => !!catalog?.backends[id] && checks[id]?.identity === readinessIdentity(catalog.backends[id]) && !checks[id]?.pending && checks[id]?.result?.runtime?.state === "available" && checks[id]?.result?.credentials?.status === "ok" && !checks[id]?.result?.check_error, [checks, catalog]);
  useEffect(() => {
    if (backend || !catalog) return;
    const defaultId = Object.keys(catalog.backends).find((id) => catalog.backends[id].default);
    const selected = defaultId && ready(defaultId) ? defaultId : ["claude", "codex"].find(ready);
    if (selected) setBackend(selected);
  }, [ready, catalog, backend]);
  useEffect(() => {
    if (!projectChosen.current && eligible.length) {
      projectChosen.current = true;
      setProject(eligible.includes(config?.default_project ?? "") ? config!.default_project : eligible[0]);
    }
  }, [eligible, project, config]);

  const finish = async (selection: { id: string; backend: string; project: string; etag?: string }) => {
    const savedDefault = config?.default_project;
    const defaultUsable = !!savedDefault && (config?.onboarding.launchable_projects ?? []).includes(savedDefault) && !!projectsQuery.data?.[savedDefault] && !projectsQuery.data[savedDefault].archived;
    await complete.mutateAsync({ onboarding_complete: true, onboarding_backend: selection.backend, onboarding_catalog_etag: selection.etag, ...(!defaultUsable ? { default_project: selection.project } : {}) });
    onComplete(selection.id);
  };
  const start = async () => {
    if (claimed.current) return;
    claimed.current = true;
    setPending(true);
    setError(undefined);
    setLaunchRepair(undefined);
    let phase: "create" | "launch" | "complete" = launched ? "complete" : "create";
    try {
      if (launched) { await finish(launched); return; }
      let projectId = createdProject || project;
      if (!projectId) {
        const cwd = folder.trim();
        const title = cwd.replace(/\/+$/, "").split("/").pop() || "My project";
        const result = await create.mutateAsync({ project: "", title, cwd, color: [...DEFAULT_PROJECT_COLOR] as [number, number, number], add_dirs: [], context_prompt: "", base_branch: "", setup_command: "", resource_dir: "" });
        setCreatedProject(result.project);
        projectId = result.project;
        if (result.warnings?.some((warning) => warning.code === "cwd_not_found")) {
          setProjectNeedsRepair(true);
          setError("The project was created, but its folder was not found. Repair its folder in Settings before starting your chat.");
          return;
        }
      }
      phase = "launch";
      const result = await launch.mutateAsync({ project: projectId, backend, role: config?.default_role || "default", interface: "chat" });
      const confirmed = { id: result.agent.agent_id, backend, project: projectId, etag: catalog?.catalogEtag };
      setLaunched(confirmed);
      phase = "complete";
      await finish(confirmed);
    } catch (failure) {
      const typed = failure as { body?: { error?: { code?: string }; code?: string }; code?: string };
      if (isProviderSetupErrorCode(typed.body?.error?.code ?? typed.body?.code ?? typed.code)) setLaunchRepair(backend);
      const status = (failure as { status?: number }).status;
      if (phase !== "complete" && (!status || status >= 500)) {
        setUncertain(true);
        void projectsQuery.refetch();
        if (phase === "launch") void getCurrentSessions().then(setRecoveryChats).catch((readFailure) => setError(`The chat launch outcome is unknown, and current chats could not be loaded: ${configErrorMessage(readFailure)}. Use Set up later to inspect your chats before starting another.`));
        setError(`The ${phase === "create" ? "project creation" : "chat launch"} outcome is unknown. Use Set up later to inspect Projects and your chats before starting another.`);
      } else setError(configErrorMessage(failure));
    } finally {
      claimed.current = false;
      setPending(false);
    }
  };
  const later = async () => {
    if (claimed.current) return;
    claimed.current = true;
    setPending(true);
    try { await complete.mutateAsync({ onboarding_complete: true }); onComplete(); }
    catch (failure) { setError(configErrorMessage(failure)); }
    finally { claimed.current = false; setPending(false); }
  };

  return <Dialog.Root open modal><Dialog.Portal>
    <Dialog.Overlay className="dialog-overlay onboarding-overlay" data-ui="dialog" data-slot="overlay" />
    <Dialog.Content className="dialog-content onboarding-wizard" data-ui="dialog" data-variant="onboarding" aria-describedby={undefined} onInteractOutside={(event) => event.preventDefault()} onEscapeKeyDown={(event) => event.preventDefault()}>
      <div className="onboarding-flow" data-ui="onboarding" data-variant="launch">
        <Dialog.Title>Welcome to Chuck</Dialog.Title>
        <div className="wizard-step" data-ui="onboarding" data-slot="step" data-variant="launch">
          <h3>Start your first chat</h3>
          <p className="wizard-step-desc">Use your existing Claude or Codex sign-in. Pick a project and start chatting.</p>
          <div aria-live="polite">
            {["claude", "codex"].map((id) => {
              const entry = catalog?.backends[id];
              if (!entry) return null;
              const outcome = checks[id];
              const result = outcome?.result;
              const detail = result?.credentials?.detail;
              const missing = result?.runtime?.state === "missing";
              const runtimeProblem = result?.runtime ? describeProviderRuntime(entry.type, result.runtime).problem : undefined;
              return <div className="form-field" key={id}>
                <strong>{providerName(entry.type)}</strong>
                <p className="form-hint">{outcome?.pending || !outcome ? "Checking…" : ready(id) ? "Ready" : outcome.error || result?.check_error?.message || runtimeProblem || (detail === "cli_incompatible" ? "Installed executable needs an update." : detail === "not_logged_in" ? "Sign in to continue." : "The check could not complete. Native sign-in could not be confirmed.")}</p>
                {!outcome?.pending && !ready(id) && <>
                  {detail === "not_logged_in" && <p className="form-hint">Run <code>chuck auth {id}</code> in your terminal, then check again.</p>}
                  {missing && <a href={providerInstallURL(entry.type)} target="_blank" rel="noopener">Install {providerName(entry.type)}</a>}
                  {missing && result?.runtime?.bundle_available && <button type="button" onClick={() => void check(id, true)} disabled={pending}>Use Chuck bundle</button>}
                  <button type="button" onClick={() => void check(id)} disabled={pending}>Check again</button>
                  <a href={backendSettingsHref(id)} target="_blank" rel="noopener" onClick={() => { repairTarget.current = id; }} aria-label={`Open backend settings for ${providerName(entry.type)} in a new tab`}>Open backend settings ↗</a>
                </>}
              </div>;
            })}
          </div>
          <div className="form-field"><label htmlFor="first-chat-provider">Provider</label><select id="first-chat-provider" value={backend} onChange={(event) => setBackend(event.target.value)} disabled={pending || !!launched}>
            {!backend && <option value="">Waiting for a ready provider</option>}
            {["claude", "codex"].filter((id) => ready(id) || id === backend).map((id) => <option key={id} value={id} disabled={!ready(id)}>{catalog?.backends[id]?.name || id}</option>)}
          </select><p className="form-hint">Your selected provider becomes the default for new agents if the original default is unchanged.</p></div>
          {backend && catalog && !catalog.backends[backend] && <p className="form-error">The selected backend is unavailable. Choose another ready provider.</p>}
          <div className="form-field"><label htmlFor="first-chat-project">Project</label><select id="first-chat-project" value={project} onChange={(event) => { projectChosen.current = true; setProject(event.target.value); }} disabled={pending || !!createdProject || !!launched}>
            <option value="">Choose a folder for a new project</option>
            {eligible.map((id) => <option key={id} value={id}>{projectsQuery.data?.[id]?.title || id}</option>)}
          </select></div>
          {!project && <div className="form-field"><label htmlFor="first-chat-folder">Project folder</label><div className="field-with-action"><input id="first-chat-folder" value={folder} onChange={(event) => { projectChosen.current = true; setFolder(event.target.value); }} disabled={pending || !!createdProject || !!launched} placeholder="~/Projects/my-app" /><BrowseDirectoryButton label="Choose project folder" disabled={pending || !!createdProject || !!launched} onPicked={(path) => { projectChosen.current = true; setFolder(path); }} onError={setError} /></div></div>}
          {backend && <details className="new-agent-options"><summary>Advanced</summary><p className="form-hint">Optionally link your native configuration or change provider settings before starting.</p><a href={backendSettingsHref(backend)} target="_blank" rel="noopener" onClick={() => { repairTarget.current = backend; }}>Open backend settings in a new tab</a></details>}
          {error && <p className="form-error" role="alert">{error}</p>}
          {launchRepair && <a href={backendSettingsHref(launchRepair)} target="_blank" rel="noopener" onClick={() => { repairTarget.current = launchRepair; }}>Open backend settings in a new tab</a>}
          {uncertain && <div className="form-info"><p>Check existing projects and chats before retrying.</p><a href="/settings?tab=projects" target="_blank" rel="noopener">Open Projects in a new tab</a>{recoveryChats.map(({ agent }) => <p key={agent.agent_id}><Link to={`/agent/${agent.agent_id}`} target="_blank">Open {agent.name || agent.agent_id}</Link></p>)}</div>}
          {launched && <p className="form-info">Your chat is ready. <Link to={`/agent/${launched.id}`} target="_blank">Open chat</Link>. Retry finishing setup to keep it from reopening.</p>}
          {projectNeedsRepair && <p className="form-hint"><a href="/settings?tab=projects" target="_blank" rel="noopener">Open project settings in a new tab</a>. <button type="button" onClick={() => { void refetchConfig().then((result) => { if (result.data?.onboarding.launchable_projects?.includes(createdProject)) { setProjectNeedsRepair(false); setError(undefined); } }); }}>Check repaired folder</button></p>}
          <div className="form-actions" data-slot="actions"><button type="button" className="ad-button-primary" onClick={() => void start()} disabled={pending || uncertain || projectNeedsRepair || (!launched && (!ready(backend) || (!project && !folder.trim()))) }>{pending ? launched ? "Finishing setup…" : "Starting chat…" : launched ? "Finish setup" : "Start chat"}</button></div>
        </div>
        <div className="onboarding-actions" data-slot="footer"><button type="button" className="btn-link" onClick={() => void later()} disabled={pending}>Set up later</button></div>
      </div>
    </Dialog.Content>
  </Dialog.Portal></Dialog.Root>;
}
