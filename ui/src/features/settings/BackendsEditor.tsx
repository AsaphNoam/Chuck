import { useEffect, useRef, useState } from "react";
import { useBackends, usePutBackends, useRefreshProvider, configErrorMessage, type CatalogETagged } from "../../api/config";
import type {
  BackendsConfig,
  Backend,
  BackendSupport,
  Model,
  CredResult,
  CreateBackendResponse,
  RefreshProviderResponse,
} from "../../schemas/backends";
import { launchSupportFor } from "../../schemas/backends";
import { hasProviderSource, providerExecutableKey } from "../../lib/providerRuntime";
import { ProviderSection } from "./ProviderSection";
import { BACKEND_TYPE_LABELS, BACKEND_TYPE_OPTIONS } from "../../lib/backendTypes";
import { ModelRow, type ModelCapability } from "./ModelRow";
import { ConfigSourcePanel } from "./ConfigSourcePanel";
import { AddBackendDialog } from "./AddBackendDialog";
import { SettingsHeader } from "./SettingsHeader";
import { isProviderSetupErrorCode } from "./backendSettingsLink";

type Pair = { key: string; value: string };

function toPairs(env: Record<string, string> | undefined): Pair[] {
  return Object.entries(env ?? {}).map(([key, value]) => ({ key, value }));
}

function fromPairs(pairs: Pair[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const { key, value } of pairs) {
    if (key.trim()) out[key.trim()] = value;
  }
  return out;
}

function SensitiveInput({ value, fieldKey, onChange }: {
  value: string;
  fieldKey: string;
  onChange: (v: string) => void;
}) {
  const sensitive = /KEY|TOKEN|SECRET/i.test(fieldKey);
  const [revealed, setRevealed] = useState(false);
  return (
    <span className="sensitive-wrap">
      <input
        type={sensitive && !revealed ? "password" : "text"}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="env-value"
      />
      {sensitive && (
        <button type="button" className="btn-link" onClick={() => setRevealed((r) => !r)}>
          {revealed ? "hide" : "show"}
        </button>
      )}
    </span>
  );
}

function EnvEditor({ pairs, onChange }: { pairs: Pair[]; onChange: (p: Pair[]) => void }) {
  return (
    <div className="env-editor">
      {pairs.map((pair, i) => (
        <div key={i} className="env-row">
          <input
            value={pair.key}
            placeholder="KEY"
            onChange={(e) => {
              const next = [...pairs];
              next[i] = { ...pair, key: e.target.value };
              onChange(next);
            }}
            className="env-key"
          />
          <SensitiveInput
            fieldKey={pair.key}
            value={pair.value}
            onChange={(v) => {
              const next = [...pairs];
              next[i] = { ...pair, value: v };
              onChange(next);
            }}
          />
          <button type="button" className="ad-button-icon" onClick={() => onChange(pairs.filter((_, j) => j !== i))}>×</button>
        </div>
      ))}
      <button type="button" className="btn-link" onClick={() => onChange([...pairs, { key: "", value: "" }])}>
        + Env var
      </button>
    </div>
  );
}

interface BackendEntry {
  id: string;
  backend: Backend;
  envPairs: Pair[];
}

function backendEntries(cfg: BackendsConfig): BackendEntry[] {
  return Object.entries(cfg.backends ?? {})
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([id, backend]) => ({ id, backend, envPairs: toPairs(backend.env) }));
}

function entriesToConfig(entries: BackendEntry[], defaultId: string): BackendsConfig {
  const backends: Record<string, Backend> = {};
  for (const { id, backend, envPairs } of entries) {
    backends[id] = {
      ...backend,
      default: id === defaultId,
      env: fromPairs(envPairs),
    };
  }
  return { version: 2, backends };
}

// modelCapability derives what NEW model declarations a backend's current
// type may offer (FS-09.R62): effort needs at least one available interface
// that supports it, fast needs chat support specifically (terminal never
// supports fast). Uncertainty is tracked per declaration so malformed support
// for one interface cannot hide known support for the other.
function modelCapability(support: BackendSupport | undefined, type: Backend["type"]): ModelCapability {
  const chat = launchSupportFor(support, type, "chat");
  const terminal = launchSupportFor(support, type, "terminal");
  const effortAllowed = Boolean((chat?.available && chat.effort) || (terminal?.available && terminal.effort));
  return {
    effortAllowed,
    effortUnknown: !effortAllowed && (chat === undefined || terminal === undefined),
    fastAllowed: Boolean(chat?.available && chat.fast),
    fastUnknown: chat === undefined,
  };
}

// canonicalCatalog renders a catalog in a key-order- and default-insensitive
// form, so the draft and the saved catalog compare equal when nothing the
// server would store differs.
function canonicalCatalog(value: unknown): string {
  const norm = (v: unknown): unknown => {
    if (Array.isArray(v)) return v.map(norm);
    if (v && typeof v === "object") {
      const out: Record<string, unknown> = {};
      for (const k of Object.keys(v as Record<string, unknown>).sort()) {
        const child = norm((v as Record<string, unknown>)[k]);
        if (child === undefined || child === false || child === "" ||
          (child && typeof child === "object" && !Array.isArray(child) && Object.keys(child).length === 0)) continue;
        out[k] = child;
      }
      return out;
    }
    return v;
  };
  return JSON.stringify(norm(value));
}

function setEnvPair(pairs: Pair[], key: string, value: string): Pair[] {
  const rest = pairs.filter((p) => p.key.trim() !== key);
  return value === "" ? rest : [...rest, { key, value }];
}

function refreshSummary(res: RefreshProviderResponse): string {
  if (res.check_error) return `Provider check failed: ${res.check_error.message}`;
  if (!res.runtime || !res.credentials) return "Provider check did not return readiness details. Retry the check.";
  const version = res.runtime.version ? `Version ${res.runtime.version}` : res.runtime.state === "available" ? "Version unknown" : "Provider unavailable";
  const signIn = res.credentials.status === "ok"
    ? "signed in"
    : res.credentials.detail === "not_logged_in"
      ? "sign-in needed"
      : res.credentials.detail === "no_api_key"
        ? "sign-in status unavailable"
        : res.credentials.detail === "invalid_api_key"
          ? "API key rejected"
          : res.credentials.status === "failed"
            ? "sign-in status unavailable"
            : "sign-in not checked";
  const models = {
    added: `${res.catalog?.added_count ?? 0} new model${res.catalog?.added_count === 1 ? "" : "s"} added`,
    unchanged: "no new models",
    disabled: "model import is off",
    unavailable: "no local model list found",
  }[res.catalog?.status ?? "unavailable"];
  return `${version}; ${signIn}; ${models}.`;
}

function credChip(result: CredResult) {
  const cls = result.status === "ok" ? "cred-ok" : result.status === "failed" ? "cred-failed" : "cred-skipped";
  return (
    <span className={`cred-chip ${cls}`} title={result.detail ?? ""}>
      {result.status}
    </span>
  );
}

export function BackendsEditor({ requestedBackendId }: { requestedBackendId?: string }) {
  const { data, isLoading, refetch } = useBackends();
  const putBackends = usePutBackends();
  const refreshProvider = useRefreshProvider();
  const [refreshing, setRefreshing] = useState<string | null>(null);
  const [refreshMessages, setRefreshMessages] = useState<Record<string, string>>({});

  const [entries, setEntries] = useState<BackendEntry[]>([]);
  const [savedBackends, setSavedBackends] = useState<Record<string, Backend>>({});
  const [defaultId, setDefaultId] = useState<string>("");
  const [catalogEtag, setCatalogEtag] = useState<string | undefined>();
  const [savedBackendIds, setSavedBackendIds] = useState<Set<string>>(new Set());
  const [credentials, setCredentials] = useState<Record<string, CredResult>>({});
  const [error, setError] = useState<string | null>(null);
  const [showBlankModels, setShowBlankModels] = useState(false);
  const [addOpen, setAddOpen] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [staleCatalog, setStaleCatalog] = useState(false);
  const [advancedBackendId, setAdvancedBackendId] = useState<string | null>(null);
  const [focusedBackendId, setFocusedBackendId] = useState<string | null>(null);
  const backendCards = useRef(new Map<string, HTMLDivElement>());

  const seedDraft = (cfg: CatalogETagged<BackendsConfig>) => {
    const es = backendEntries(cfg);
    setEntries(es);
    setSavedBackendIds(new Set(Object.keys(cfg.backends ?? {})));
    setSavedBackends(cfg.backends ?? {});
    setDefaultId(es.find((e) => e.backend.default)?.id ?? es[0]?.id ?? "");
    setCatalogEtag(cfg.catalogEtag);
  };

  // Seed the draft from the server exactly once. Re-seeding on every query
  // change would let a background refetch — including the one an item create or
  // a source connection triggers — silently discard unrelated unsaved edits
  // (FS-04.R40). Save and create merge their own results explicitly.
  const seeded = useRef(false);
  useEffect(() => {
    if (!data || seeded.current) return;
    seeded.current = true;
    seedDraft(data);
  }, [data]);

  useEffect(() => {
    if (!requestedBackendId || !data) return;
    if (!data.backends[requestedBackendId]) {
      setFocusedBackendId(null);
      setAdvancedBackendId(null);
      return;
    }
    setFocusedBackendId(requestedBackendId);
    setAdvancedBackendId(requestedBackendId);
    const frame = requestAnimationFrame(() => {
      const card = backendCards.current.get(requestedBackendId);
      card?.scrollIntoView({ block: "center" });
      card?.focus();
    });
    return () => cancelAnimationFrame(frame);
  }, [requestedBackendId, data]);

  const updateEntry = (id: string, patch: Partial<BackendEntry>) => {
    setEntries((prev) => prev.map((e) => (e.id === id ? { ...e, ...patch } : e)));
  };

  const updateBackend = (id: string, patch: Partial<Backend>) => {
    setEntries((prev) =>
      prev.map((e) => (e.id === id ? { ...e, backend: { ...e.backend, ...patch } } : e)),
    );
  };

  // mergeBackend inserts or replaces exactly one entry, leaving every other
  // unsaved edit in the draft untouched.
  const mergeBackend = (id: string, backend: Backend) => {
    setEntries((prev) => {
      const entry: BackendEntry = { id, backend, envPairs: toPairs(backend.env) };
      const at = prev.findIndex((e) => e.id === id);
      if (at === -1) return [...prev, entry];
      const next = [...prev];
      next[at] = entry;
      return next;
    });
    setSavedBackendIds((prev) => new Set(prev).add(id));
    setSavedBackends((prev) => ({ ...prev, [id]: backend }));
    if (backend.default) setDefaultId(id);
  };

  const handleCreated = ({ backend_id: id, backend, connection, catalogEtag: nextCatalogEtag }: CatalogETagged<CreateBackendResponse>) => {
    setError(null);
    mergeBackend(id, backend);
    setCatalogEtag(nextCatalogEtag);
    setAddOpen(false);
    if (!connection) {
      setNotice(null);
    } else if (connection.status === "connected") {
      const added = connection.models_added ?? 0;
      setNotice(
        `Connected ${backend.name} to your native configuration` +
          (added > 0 ? ` and imported ${added} configured model${added === 1 ? "" : "s"}.` : ".") +
          " Imported models are what your configuration names, not a check of availability or entitlement.",
      );
    } else {
      setNotice(
        `${backend.name} was created but is not connected: ${connection.error?.message ?? "the connection failed"}.` +
          " Use its Configuration source panel to try again.",
      );
    }
  };

  // syncBackendFromServer refreshes one card from the authoritative catalog after
  // a connection imported models into it, without re-seeding the whole draft.
  const syncBackendFromServer = async (id: string) => {
    const fresh = await refetch();
    const backend = fresh.data?.backends?.[id];
    if (backend) {
      mergeBackend(id, backend);
      setSavedBackends((prev) => ({ ...prev, [id]: backend }));
    }
    setCatalogEtag(fresh.data?.catalogEtag);
  };

  const removeBackend = (id: string) => {
    setEntries((prev) => {
      const next = prev.filter((e) => e.id !== id);
      if (defaultId === id && next.length > 0) setDefaultId(next[0].id);
      return next;
    });
  };

  const handleSave = () => {
    setError(null);
    // Same rule and wording as creation (AddBackendDialog); the server refuses it too (FS-09.R82).
    const unnamed = entries.find((e) => !e.backend.name.trim());
    if (unnamed) {
      setError(`Enter a name for backend ${unnamed.id}.`);
      return;
    }
    // The server refuses an empty provider model string with internal keys; say
    // it here in the person's terms and mark the row itself.
    for (const { backend } of entries) {
      const blank = Object.values(backend.models ?? {}).find((m) => !m.model.trim());
      if (blank) {
        setShowBlankModels(true);
        setError(`Enter a provider model string for ${blank.name.trim() || "the new model"} in ${backend.name}.`);
        return;
      }
    }
    const config = { ...entriesToConfig(entries, defaultId), catalogEtag };
    putBackends.mutate(config, {
      onSuccess: (resp) => {
        setStaleCatalog(false);
        setCredentials(resp.credentials ?? {});
        seedDraft(resp); // re-sync the draft from the normalized response
      },
      onError: async (e) => {
        setError(configErrorMessage(e));
        if ((e as { body?: { error?: { code?: string } } })?.body?.error?.code === "backend_catalog_changed") {
          setStaleCatalog(true);
          await refetch();
        }
      },
    });
  };

  // One refresh runs at a time across the editor: each one advances the
  // catalog ETag, so a second concurrent request would only conflict.
  const updateAfterProviderAction = (id: string, submitted: Backend | undefined, res: CatalogETagged<RefreshProviderResponse>) => {
    setCredentials((prev) => {
      const next = { ...prev };
      if (res.credentials) next[id] = res.credentials;
      else delete next[id];
      return next;
    });
    const result = refreshSummary(res);
    setRefreshMessages((prev) => ({ ...prev, [id]: submitted ? `${res.backend?.name ?? "Backend"} settings saved. ${result}` : result }));
    if (res.backend && submitted) {
      setSavedBackends((prev) => ({ ...prev, [id]: res.backend! }));
      setSavedBackendIds((prev) => new Set(prev).add(id));
      setEntries((prev) => prev.map((entry) => {
        if (entry.id !== id) return entry;
        const current = { ...entry.backend, env: fromPairs(entry.envPairs) };
        if (submitted && canonicalCatalog(current) !== canonicalCatalog(submitted)) {
          const imported = Object.fromEntries(
            Object.entries(res.backend!.models ?? {}).filter(([modelId]) => !(modelId in submitted.models)),
          );
          return {
            ...entry,
            backend: { ...entry.backend, models: { ...imported, ...entry.backend.models } },
          };
        }
        return { ...entry, backend: res.backend!, envPairs: toPairs(res.backend!.env) };
      }));
    }
    if (res.catalogEtag) setCatalogEtag(res.catalogEtag);
    if (res.check_error && isProviderSetupErrorCode(res.check_error.code)) {
      setAdvancedBackendId(id);
    }
    if (res.check_error?.code === "backend_catalog_changed") {
      setStaleCatalog(true);
      void refetch();
    }
  };

  const reconcileProviderCatalog = async (id: string, before: Backend, res: CatalogETagged<RefreshProviderResponse>) => {
    const fresh = await refetch();
    if (!fresh.data || !res.catalogEtag || fresh.data.catalogEtag !== res.catalogEtag) {
      setStaleCatalog(true);
      return;
    }
    const committed = fresh.data.backends[id];
    if (!committed) {
      setStaleCatalog(true);
      return;
    }
    setCatalogEtag(res.catalogEtag);
    setSavedBackends((prev) => ({ ...prev, [id]: committed }));
    const added = Object.fromEntries(Object.entries(committed.models).filter(([modelId]) => !(modelId in before.models)));
    setEntries((prev) => prev.map((entry) => entry.id === id
      ? { ...entry, backend: { ...entry.backend, models: { ...added, ...entry.backend.models } } }
      : entry));
  };

  const refreshSavedBackend = async (id: string, backend: Backend) => {
    setRefreshing(id);
    setRefreshMessages((prev) => ({ ...prev, [id]: "" }));
    try {
      const res = await refreshProvider.mutateAsync({ backendId: id, modelId: backend.default_model, catalogEtag });
      updateAfterProviderAction(id, undefined, res);
      await reconcileProviderCatalog(id, backend, res);
    } catch (e) {
      setRefreshMessages((prev) => ({ ...prev, [id]: `Refresh failed: ${configErrorMessage(e)}` }));
      if ((e as { body?: { error?: { code?: string } } })?.body?.error?.code === "backend_catalog_changed") {
        setStaleCatalog(true);
        await refetch();
      }
    } finally {
      setRefreshing(null);
    }
  };

  const saveAndCheckBackend = async (id: string, backend: Backend) => {
    const submitted: Backend = { ...backend, default: id === defaultId, env: fromPairs(entries.find((entry) => entry.id === id)?.envPairs ?? []) };
    setRefreshing(id);
    setRefreshMessages((prev) => ({ ...prev, [id]: "" }));
    try {
      const res = await refreshProvider.mutateAsync({ backendId: id, backend: submitted, catalogEtag });
      updateAfterProviderAction(id, submitted, res);
      await reconcileProviderCatalog(id, submitted, res);
    } catch (e) {
      setRefreshMessages((prev) => ({ ...prev, [id]: `Save & check failed: ${configErrorMessage(e)}` }));
      if ((e as { body?: { error?: { code?: string } } })?.body?.error?.code === "backend_catalog_changed") {
        setStaleCatalog(true);
        await refetch();
      }
    } finally {
      setRefreshing(null);
    }
  };

  if (isLoading) return <p data-ui="config-editor" data-state="loading" data-variant="backends">Loading backends…</p>;

  return (
    <div className="config-editor backends-editor" data-ui="config-editor" data-state={error ? "error" : entries.length === 0 ? "empty" : undefined} data-variant="backends">
      <SettingsHeader eyebrow="Backends" title="Configured backends">
        <button type="button" className="ad-button-primary" onClick={() => setAddOpen(true)}>Add backend</button>
      </SettingsHeader>

      <AddBackendDialog
        open={addOpen}
        existingIds={entries.map((e) => e.id)}
        onCancel={() => setAddOpen(false)}
        onCreated={handleCreated}
      />

      {notice && <p className="config-notice" role="status">{notice}</p>}

      {entries.length === 0 && (
        <p className="config-empty">No backends configured. Add one to get started.</p>
      )}

      {requestedBackendId && data && (!data.backends[requestedBackendId] || !entries.some((entry) => entry.id === requestedBackendId)) && (
        <p className="config-notice" role="status">
          Backend “{requestedBackendId}” is unavailable. Choose a configured backend below or add it again.
        </p>
      )}

      {entries.map(({ id, backend, envPairs }) => {
        const capability = modelCapability(data?.backend_support, backend.type);
        const saved = savedBackends[id];
        const targetDirty = !!saved && canonicalCatalog({ ...backend, default: id === defaultId, env: fromPairs(envPairs) }) !== canonicalCatalog(saved);
        const canSaveAndCheck = !!saved && saved.type === backend.type && Boolean(saved.default) === (id === defaultId);
        const advancedOpen = advancedBackendId === id;
        return (
        <div
          key={id}
          ref={(element) => {
            if (element) backendCards.current.set(id, element);
            else backendCards.current.delete(id);
          }}
          className="backend-card"
          data-slot="item"
          tabIndex={focusedBackendId === id ? -1 : undefined}
          role="group"
          aria-label={`Backend ${backend.name}`}
        >
          <div className="backend-card-header">
            <label className="backend-default-label">
              <input
                type="radio"
                name="default-backend"
                checked={id === defaultId}
                onChange={() => setDefaultId(id)}
              />
              Default
            </label>
            <input
              value={backend.name}
              placeholder="Backend name"
              onChange={(e) => updateBackend(id, { name: e.target.value })}
              className="backend-name-input"
            />
            <select
              value={backend.type}
              onChange={(e) => {
                const type = e.target.value as Backend["type"];
                // Provider mode belongs only to Claude/Codex (TS-03.R54).
                updateBackend(id, hasProviderSource(type) ? { type } : { type, provider_mode: undefined });
              }}
              className="backend-type-select"
            >
              {BACKEND_TYPE_OPTIONS.map((t) => (
                <option key={t} value={t}>
                  {BACKEND_TYPE_LABELS[t]}
                </option>
              ))}
            </select>
            {credentials[id] && credChip(credentials[id])}
            <button type="button" className="config-text-action-danger ad-button-danger" onClick={() => removeBackend(id)}>
              Remove
            </button>
          </div>

          {hasProviderSource(backend.type) && (
            <ProviderSection
              backendId={id}
              backend={backend}
              executable={envPairs.find((p) => p.key.trim() === providerExecutableKey(backend.type))?.value ?? ""}
              onModeChange={(mode) => updateBackend(id, { provider_mode: mode })}
              onExecutableChange={(value) => updateEntry(id, { envPairs: setEnvPair(envPairs, providerExecutableKey(backend.type), value) })}
              runtime={savedBackendIds.has(id) ? data?.provider_runtimes?.[id]?.[backend.default_model] : undefined}
              targetDirty={targetDirty}
              canSaveAndCheck={canSaveAndCheck}
              catalogStale={staleCatalog}
              advancedOpen={advancedOpen}
              refreshing={refreshing === id}
              otherRefreshing={refreshing !== null && refreshing !== id}
              actionMessage={refreshMessages[id] || null}
              actionLabel={targetDirty ? "Save & check" : "Refresh provider"}
              onAction={() => {
                if (!targetDirty) {
                  const savedBackend = savedBackends[id];
                  if (savedBackend) void refreshSavedBackend(id, savedBackend);
                } else if (canSaveAndCheck) {
                  void saveAndCheckBackend(id, backend);
                }
              }}
            />
          )}

          <details className="backend-advanced-section" open={advancedOpen} onToggle={(event) => {
            if (event.currentTarget.open) setAdvancedBackendId(id);
            else if (advancedBackendId === id) setAdvancedBackendId(null);
          }}>
            <summary>Advanced</summary>
          <details className="backend-env-section">
            <summary>Backend env ({envPairs.length})</summary>
            <EnvEditor
              pairs={envPairs}
              onChange={(next) => updateEntry(id, { envPairs: next })}
            />
          </details>

          {backend.type === "codex-acp" && (
            <label className="backend-autosync-label" title="On dashboard startup, add newly available Codex models from ~/.codex/models_cache.json. Existing entries and the default are never changed.">
              <input
                type="checkbox"
                checked={backend.autosync_models ?? false}
                onChange={(e) => updateBackend(id, { autosync_models: e.target.checked })}
              />
              Auto-sync models from Codex on startup
            </label>
          )}

          {backend.type === "claude-acp" && (
            <label className="backend-autosync-label" title="At the next dashboard start, add the models named in your user-level Claude settings (~/.claude/settings.json). Existing entries and the default are never changed.">
              <input
                type="checkbox"
                checked={backend.autosync_models ?? false}
                onChange={(e) => updateBackend(id, { autosync_models: e.target.checked })}
              />
              Import configured models from Claude on startup
            </label>
          )}

          {(backend.type === "claude-acp" || backend.type === "codex-acp") && (
            <ConfigSourcePanel
              backendId={id}
              backendType={backend.type}
              persisted={savedBackendIds.has(id)}
              onConnected={() => void syncBackendFromServer(id)}
            />
          )}

          <div className="backend-models-section">
            <div className="backend-models-header">
              <strong className="config-group-label">Models</strong>
            </div>
            {(capability.effortUnknown || capability.fastUnknown) && (
              <p className="config-notice model-capability-note" role="status">
                {[
                  capability.effortUnknown && "Effort support could not be loaded.",
                  capability.fastUnknown && "Fast support could not be loaded.",
                ].filter(Boolean).join(" ")} New declarations for those options are unavailable until they can be checked. Existing declarations can still be cleared.
                <button type="button" className="btn-link" onClick={() => void refetch()}>Retry</button>
              </p>
            )}
            {Object.entries(backend.models ?? {}).map(([modelId, model]) => (
              <ModelRow
                key={modelId}
                modelId={modelId}
                model={model}
                isDefault={backend.default_model === modelId}
                radioGroup={`default-model-${id}`}
                capability={capability}
                showBlankModel={showBlankModels}
                onSetDefault={() => updateBackend(id, { default_model: modelId })}
                onChange={(updatedModel) => {
                  updateBackend(id, {
                    models: { ...(backend.models ?? {}), [modelId]: updatedModel },
                  });
                }}
                onRemove={() => {
                  const next = { ...(backend.models ?? {}) };
                  delete next[modelId];
                  updateBackend(id, { models: next });
                }}
              />
            ))}
            <button
              type="button"
              className="btn-link"
              onClick={() => {
                const newId = `model-${Date.now()}`;
                const newModel: Model = { name: "New model", model: "" };
                updateBackend(id, {
                  models: { ...(backend.models ?? {}), [newId]: newModel },
                  default_model: Object.keys(backend.models ?? {}).length === 0 ? newId : backend.default_model,
                });
              }}
            >
              + Add model
            </button>
          </div>
          </details>
        </div>
        );
      })}

      {error && <p className="form-error">{error}</p>}
      {staleCatalog && (
        <p className="config-notice" role="alert">
          The saved backend catalog changed elsewhere. Your draft is preserved; reload the saved catalog before writing again.
          <button type="button" className="btn-link" onClick={async () => {
            const fresh = await refetch();
            if (fresh.data) {
              seedDraft(fresh.data);
              setStaleCatalog(false);
              setError(null);
              setRefreshMessages({});
            }
          }}>Reload saved catalog</button>
        </p>
      )}

      <div className="backends-footer" data-slot="actions">
        <button type="button" className="ad-button-primary" onClick={handleSave} disabled={staleCatalog || putBackends.isPending}>
          {putBackends.isPending ? "Saving…" : "Save"}
        </button>
      </div>
    </div>
  );
}
