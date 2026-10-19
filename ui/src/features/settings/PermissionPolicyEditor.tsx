import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { configErrorMessage, QUERY_KEYS, useConfig, usePutConfig } from "../../api/config";
import type { Config } from "../../schemas/config";
import { SettingsHeader } from "./SettingsHeader";

export function PermissionPolicyEditor() {
  const config = useConfig();
  const putConfig = usePutConfig();
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | undefined>();
  const skip = config.data?.skip_permissions ?? false;

  const save = (next: boolean) => {
    if (putConfig.isPending || config.isError) return;
    const previous = queryClient.getQueryData<Config>(QUERY_KEYS.config);
    queryClient.setQueryData<Config>(QUERY_KEYS.config, (current) => current ? { ...current, skip_permissions: next } : current);
    setError(undefined);
    putConfig.mutate(
      { skip_permissions: next },
      {
        onError: (reason: unknown) => {
          queryClient.setQueryData(QUERY_KEYS.config, previous);
          setError(`Default permissions were not saved: ${configErrorMessage(reason)}`);
        },
        onSuccess: (saved) => queryClient.setQueryData(QUERY_KEYS.config, saved),
        onSettled: () => { void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.config }); },
      },
    );
  };

  return (
    <section className="config-editor" data-ui="config-editor">
      <SettingsHeader
        eyebrow="Permissions"
        title="Default permissions"
        description="Applies to No persona and to personas that use default permissions. Changes affect new launches only."
      >
        {putConfig.isPending && <span role="status">Saving…</span>}
      </SettingsHeader>
      {config.isError && <p className="form-error" role="alert">Default permissions could not be loaded: {configErrorMessage(config.error)}</p>}
      {error && <p className="form-error" role="alert">{error}</p>}
      <label className="toggle-row toggle-row-primary">
        <input
          type="checkbox"
          aria-label="Skip permission prompts"
          checked={skip}
          disabled={config.isLoading || config.isError || putConfig.isPending}
          onChange={(event) => save(event.target.checked)}
        />
        Skip permission prompts
      </label>
    </section>
  );
}
