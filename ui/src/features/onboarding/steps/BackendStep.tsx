import { useState } from "react";
import { configErrorMessage, useBackends, usePutBackends } from "../../../api/config";
import type { BackendsConfig, BackendType } from "../../../schemas/backends";
import { hasProviderSource, providerInstallURL, providerName } from "../../../lib/providerRuntime";
import { BACKEND_TYPE_LABELS, BACKEND_TYPE_OPTIONS } from "../../../lib/backendTypes";

interface BackendStepProps {
  // Reports the chosen seeded backend so later steps (e.g. the federation Config
  // step) target the right provider instead of assuming Claude.
  onDone: (backend: { id: string; type: BackendType }) => void;
  claimMutation?: () => boolean;
  releaseMutation?: () => void;
}

// seededIdForType maps a backend type to its seeded backend id (onboarding
// edits the seeded entry for the chosen type).
const seededIdForType: Record<BackendType, string> = {
  "claude-acp": "claude",
  "codex-acp": "codex",
  "opencode-acp": "opencode",
  "openhands-acp": "openhands",
};

// signInProvider maps a backend type to the `chuck auth` selector for
// providers that own an interactive sign-in. Chuck never runs, proxies, or
// observes that flow — onboarding only tells the person which command to run in
// their own terminal, then re-checks readiness (FS-04.R34, TS-04.R15).
const signInProvider: Partial<Record<BackendType, string>> = {
  "claude-acp": "claude",
  "codex-acp": "codex",
};

// adapterCommand is the executable Chuck launches with `acp` for backends whose
// CLI Chuck does not select or bundle (FS-09.R11/R12).
const adapterCommand: Record<"opencode-acp" | "openhands-acp", string> = {
  "opencode-acp": "opencode",
  "openhands-acp": "openhands",
};

function credentialGuidance(status: string, detail: string | null, type: BackendType): string {
  if (detail === "cli_not_installed") {
    return hasProviderSource(type)
      ? `${providerName(type)} was not found. Install it (${providerInstallURL(type)}), or choose Chuck bundle for this backend in Settings → Backends, then check again.`
      : `The ${BACKEND_TYPE_LABELS[type]} CLI was not found. Chuck runs \`${adapterCommand[type]} acp\`, so install the ${BACKEND_TYPE_LABELS[type]} CLI so that \`${adapterCommand[type]}\` is on the PATH of the environment Chuck runs in, then check again.`;
  }
  if (detail === "bundle_unavailable") {
    return `The Chuck bundle for ${providerName(type)} is not available in this installation. Choose Installed provider in Settings → Backends, or reinstall Chuck, then check again.`;
  }
  if (detail === "cli_invalid") {
    return `The ${providerName(type)} executable path set for this backend is not usable. Fix or clear it in Settings → Backends, then check again.`;
  }
  if (detail === "cli_incompatible") {
    return `The installed ${BACKEND_TYPE_LABELS[type]} adapter is too old for Chuck's readiness check, so its sign-in could not be confirmed. Update the adapter, then check again.`;
  }
  if (detail === "not_logged_in") {
    return signInProvider[type]
      ? `${BACKEND_TYPE_LABELS[type]} is not signed in. Run the sign-in command above in a terminal, then check again.`
      : `${BACKEND_TYPE_LABELS[type]} is not signed in. Complete its sign-in in ${BACKEND_TYPE_LABELS[type]}'s own tool, then check again.`;
  }
  if (detail === "no_api_key") {
    return type === "codex-acp"
      ? "No sign-in and no API key were found. Run the sign-in command above, or add an API key, then check again."
      : "No API key was found. Add the API key above, then check again.";
  }
  return status === "failed"
    ? "The credential check failed. Confirm this provider's sign-in or API key, then check again."
    : "The credential check could not run yet. Confirm the provider setup, then check again.";
}

export function BackendStep({ onDone, claimMutation, releaseMutation }: BackendStepProps) {
  const { data: existing, isLoading } = useBackends();
  const putBackends = usePutBackends();

  const [type, setType] = useState<BackendType>("claude-acp");
  const [apiKey, setApiKey] = useState("");
  const [llmApiKey, setLlmApiKey] = useState("");
  const [llmBaseUrl, setLlmBaseUrl] = useState("");
  const [credStatus, setCredStatus] = useState<string | null>(null);
  const [credDetail, setCredDetail] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const backendId = seededIdForType[type];
  const seeded = existing?.backends[backendId];
  // Name the target backend when several share this provider, so sign-in
  // uses the same provider selection as launch (FS-10.R21).
  const sameTypeCount = Object.values(existing?.backends ?? {}).filter((b) => b.type === type).length;
  const readyToValidate = !!existing && !isLoading;
  // Once a readiness result came back non-ok, the same submit becomes a plain
  // re-check: nothing about the document changed, only the provider's state.
  const hasUnreadyResult = credStatus !== null && credStatus !== "ok";
  const authProvider = signInProvider[type];
  // A hand-edited catalog can leave the chosen backend with no usable default
  // model. Onboarding no longer invents one (FS-04.R33), so say so plainly
  // rather than silently saving a broken default (INV §8, §11).
  const defaultModel = seeded?.default_model ?? "";
  const defaultModelEntry = defaultModel ? seeded?.models?.[defaultModel] : undefined;
  const catalogUnusable = readyToValidate && (!seeded || !defaultModelEntry);

  const handleValidate = async () => {
    if (!readyToValidate || catalogUnusable) return;
    if (claimMutation && !claimMutation()) return;
    setError(null);
    setCredStatus(null);
    // Credentials entered here belong to the backend, not to one model entry, so
    // they survive a later model change in Settings.
    const env: Record<string, string> = { ...(seeded?.env ?? {}) };
    if (type === "codex-acp" && apiKey) env["OPENAI_API_KEY"] = apiKey;
    if (type === "openhands-acp") {
      if (llmApiKey) env["LLM_API_KEY"] = llmApiKey;
      if (llmBaseUrl) env["LLM_BASE_URL"] = llmBaseUrl;
    }

    const backends: BackendsConfig["backends"] = existing?.backends ?? {};
    // Merge-preserve (INV §3): the seeded catalog — every model entry and its
    // default — is submitted unchanged. This save only overlays credentials;
    // it never writes back a partial on-screen view of the model catalog, and
    // it leaves every default flag as saved so an unready choice (or a
    // repeated Check again) never replaces the working default (FS-04.R17).
    const config: BackendsConfig = {
      version: 2,
      backends: {
        ...backends,
        [backendId]: {
          ...seeded,
          name: seeded?.name ?? BACKEND_TYPE_LABELS[type],
          type,
          default: seeded?.default ?? false,
          default_model: defaultModel,
          models: seeded?.models ?? {},
          env,
        },
      },
    };

    try {
      const resp = await putBackends.mutateAsync({ ...config, catalogEtag: existing?.catalogEtag });
      const cred = resp.credentials?.[backendId];
      const status = cred?.status ?? null;
      setCredStatus(status);
      setCredDetail(cred?.detail ?? null);
      if (status !== "ok") return;
      // Ready: only now does the chosen backend become the default, on top of
      // the catalog just saved.
      if (!resp.backends[backendId]?.default) {
        const promoted = Object.fromEntries(
          Object.entries(resp.backends).map(([id, b]) => [id, { ...b, default: id === backendId }]),
        );
        await putBackends.mutateAsync({ version: 2, backends: promoted, catalogEtag: resp.catalogEtag });
      }
      onDone({ id: backendId, type });
    } catch (e) {
      setError(configErrorMessage(e));
    } finally {
      releaseMutation?.();
    }
  };

  return (
    <div className="wizard-step" data-ui="onboarding" data-slot="step" data-variant="backend">
      <h3>Configure your AI backend</h3>
      <p className="wizard-step-desc">
        Choose a backend and confirm it is ready. You can change models and add more backends later
        in Settings.
      </p>

      <div className="form-field">
        <label htmlFor="onboarding-backend-type">Backend type</label>
        <select id="onboarding-backend-type" value={type} onChange={(e) => setType(e.target.value as BackendType)}>
          {BACKEND_TYPE_OPTIONS.map((t) => (
            <option key={t} value={t}>
              {BACKEND_TYPE_LABELS[t]}
            </option>
          ))}
        </select>
      </div>

      {defaultModelEntry && (
        <p className="form-hint">
          Using this backend's default model: {defaultModelEntry.name}.
        </p>
      )}

      {authProvider && (
        <div className="wizard-guidance" data-slot="guidance">
          <p>
            {BACKEND_TYPE_LABELS[type]} sign-in happens in {BACKEND_TYPE_LABELS[type]}'s own tool, not
            in Chuck. In a terminal, run:
          </p>
          <p>
            <code>chuck auth {authProvider}{sameTypeCount > 1 ? ` --backend ${backendId}` : ""}</code>
          </p>
          <p className="form-hint">
            {type === "codex-acp"
              ? "Already signed in to Codex? Nothing else is needed — an API key below is only an alternative."
              : "Chuck never sees or stores your credentials."}
          </p>
        </div>
      )}

      {type === "codex-acp" && (
        <div className="form-field">
          <label htmlFor="onboarding-openai-key">OpenAI API key (optional)</label>
          <input
            id="onboarding-openai-key"
            type="password"
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder="sk-..."
            autoComplete="off"
          />
        </div>
      )}

      {type === "openhands-acp" && (
        <>
          <div className="form-field">
            <label htmlFor="onboarding-llm-key">LLM API key</label>
            <input
              id="onboarding-llm-key"
              type="password"
              value={llmApiKey}
              onChange={(e) => setLlmApiKey(e.target.value)}
              placeholder="sk-..."
              autoComplete="off"
            />
          </div>
          <div className="form-field">
            <label htmlFor="onboarding-llm-base-url">LLM base URL (optional)</label>
            <input
              id="onboarding-llm-base-url"
              value={llmBaseUrl}
              onChange={(e) => setLlmBaseUrl(e.target.value)}
              placeholder="https://api.anthropic.com"
            />
          </div>
        </>
      )}

      {catalogUnusable && (
        <p className="form-error">
          This backend has no usable default model in <code>backends.json</code>. Fix its catalog in
          Settings → Backends, or choose another backend.
        </p>
      )}
      {credStatus && credStatus !== "ok" && (
        <p className="form-error">
          {credentialGuidance(credStatus, credDetail, type)}
        </p>
      )}
      {error && <p className="form-error">{error}</p>}

      <div className="form-actions" data-slot="actions">
        <button
          type="button"
          className="ad-button-primary"
          onClick={() => void handleValidate()}
          disabled={putBackends.isPending || !readyToValidate || catalogUnusable}
        >
          {putBackends.isPending
            ? "Checking…"
            : hasUnreadyResult
              ? "Check again"
              : "Validate & Continue"}
        </button>
      </div>
    </div>
  );
}
