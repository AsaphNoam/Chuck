import { useEffect, useState } from "react";
import { useRoles, useProjects, useBackends, useConfig, useLaunchAgent, usePutConfig, configErrorMessage } from "../../../api/config";
import { useSuggestedName } from "../../launch/useSuggestedName";
import { useUiStore } from "../../../store/uiStore";
import { displayLabel, displayLabels } from "../../../lib/labels";
import { isCanonicalNoPersona, personaLabel } from "../../../lib/personas";

interface LaunchStepProps {
  onDone: () => void;
  /** Slug of the project created in ProjectStep. Preferred over the seeded
   * default so onboarding launches a project whose cwd actually exists. */
  initialProject?: string;
  claimMutation?: () => boolean;
  releaseMutation?: () => void;
}

export function LaunchStep({ onDone, initialProject, claimMutation, releaseMutation }: LaunchStepProps) {
  const { data: rolesData } = useRoles();
  const { data: projectsData } = useProjects();
  const { data: backendsData } = useBackends();
  const { data: configData, isError: configError } = useConfig();
  const launch = useLaunchAgent();
  const putConfig = usePutConfig();
  const pushError = useUiStore((state) => state.pushError);

  const roleEntries = Object.entries(rolesData ?? {});
  const personaLabels = roleEntries.map(([id, value]) => [id, personaLabel(id, value)] as [string, string]);
  const projectEntries = Object.entries(projectsData ?? {}).filter(([, project]) => !project.archived);

  const defaultBackendId =
    Object.entries(backendsData?.backends ?? {}).find(([, b]) => b.default)?.[0] ??
    Object.keys(backendsData?.backends ?? {})[0] ??
    "";

  const defaultProject = (initialProject && projectsData?.[initialProject] && !projectsData[initialProject].archived)
    ? initialProject
    : projectEntries[0]?.[0] ?? "";

  const [role, setRole] = useState("");
  const [project, setProject] = useState(defaultProject);
  const [error, setError] = useState<string | null>(null);
  const [launchSucceeded, setLaunchSucceeded] = useState(false);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const roleInvalid = Boolean(role && rolesData && !rolesData[role]);
  useEffect(() => {
    if (roleInvalid) setAdvancedOpen(true);
  }, [roleInvalid]);

  const [name, setName] = useSuggestedName(role);

  // Apply defaults once data loads.
  useEffect(() => {
    if (role || roleEntries.length === 0 || (!configData && !configError)) return;
    const configured = configData?.default_role;
    if (configured && roleEntries.some(([id]) => id === configured)) setRole(configured);
    else if (roleEntries.some(([id, value]) => isCanonicalNoPersona(id, value))) setRole("default");
    else setRole(roleEntries[0][0]);
  }, [configData, configError, role, roleEntries]);

  useEffect(() => {
    if (project) return;
    if (initialProject) setProject(initialProject);
    else if (projectEntries.length > 0) setProject(projectEntries[0][0]);
  }, [projectEntries.length, initialProject]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (project && projectsData && (!projectsData[project] || projectsData[project].archived)) setProject("");
  }, [project, projectsData]);

  const handleLaunch = () => {
    if (roleInvalid || !role) return;
    if (claimMutation && !claimMutation()) return;
    setError(null);
    launch.mutate(
      { name: name || undefined, role, project, backend: defaultBackendId || undefined, interface: "chat" },
      {
        onSuccess: () => {
          setLaunchSucceeded(true);
          putConfig.mutate({ onboarding_complete: true }, {
            onSuccess: onDone,
            onError: (e) => {
              pushError("Failed to mark onboarding complete", configErrorMessage(e));
              // Agent launched successfully; keep the wizard visible but show the error.
              // User can manually dismiss or the service will eventually reconcile the state.
            },
            onSettled: releaseMutation,
          });
        },
      onError: (e) => {
          setAdvancedOpen(true);
          setError(configErrorMessage(e));
          releaseMutation?.();
        },
      },
    );
  };

  return (
    <div className="wizard-step" data-ui="onboarding" data-slot="step" data-variant="launch">
      <h3>Launch your first agent</h3>
      <p className="wizard-step-desc">
        You're all set! Launch an agent to complete setup.
      </p>

      <div className="form-field">
        <label>Project</label>
        <select value={project} onChange={(e) => setProject(e.target.value)}>
          {displayLabels(projectEntries.map(([id, p]) => [id, p.title])).map(([id, title]) => (
            <option key={id} value={id}>{title}</option>
          ))}
        </select>
      </div>
      <div className="new-agent-runtime">
        <span>Runs with</span>
        <strong>{backendsData?.backends[defaultBackendId]?.name || "Configured backend"}</strong>
        <span>Chat</span>
        <span>{role ? displayLabel(personaLabels, role) || role : "Loading persona…"}</span>
      </div>
      <details className="new-agent-options" open={advancedOpen} onToggle={(event) => setAdvancedOpen(event.currentTarget.open)}>
        <summary>Advanced</summary>
        <div className="form-field">
          <label htmlFor="onboarding-persona">Persona</label>
          <select id="onboarding-persona" value={role} onChange={(e) => setRole(e.target.value)}>
            {roleInvalid && <option value={role}>{role} (unavailable)</option>}
            {displayLabels(roleEntries.map(([id, r]) => [id, personaLabel(id, r)])).map(([id, title]) => (
              <option key={id} value={id}>{title}</option>
            ))}
          </select>
          {roleInvalid && <span className="form-error">This persona is unavailable. Choose another persona.</span>}
        </div>
        <div className="form-field">
          <label htmlFor="onboarding-agent-name">Agent name</label>
          <input id="onboarding-agent-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Atlas" />
        </div>
      </details>

      {error && <p className="form-error">{error}</p>}

      <div className="form-actions" data-slot="actions">
        <button type="button" className="ad-button-primary" onClick={handleLaunch} disabled={launch.isPending || putConfig.isPending || !role || roleInvalid || !project}>
          {launch.isPending ? "Launching…" : putConfig.isPending ? "Completing setup…" : "Launch"}
        </button>
        {launchSucceeded && !putConfig.isSuccess && (
          <p className="form-info">Agent launched successfully. Completing setup…</p>
        )}
      </div>
    </div>
  );
}
