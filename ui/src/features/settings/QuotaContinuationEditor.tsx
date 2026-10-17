import { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { configErrorMessage, QUERY_KEYS, useConfig, usePutConfig } from "../../api/config";
import type { Config } from "../../schemas/config";
import { SettingsHeader } from "./SettingsHeader";

export function QuotaContinuationEditor() {
  const config = useConfig();
  const putConfig = usePutConfig();
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | undefined>();
  const enabled = config.data?.auto_continue ?? true;

  const save = (next: boolean) => {
    if (putConfig.isPending || config.isError) return;
    const previous = queryClient.getQueryData<Config>(QUERY_KEYS.config);
    queryClient.setQueryData<Config>(QUERY_KEYS.config, (current) => current ? { ...current, auto_continue: next } : current);
    setError(undefined);
    putConfig.mutate(
        { auto_continue: next },
        {
          onError: (reason: unknown) => {
          const savedValue = (reason as { body?: { auto_continue?: unknown } })?.body?.auto_continue;
          if (typeof savedValue === "boolean") {
            queryClient.setQueryData<Config>(QUERY_KEYS.config, (current) => current
              ? { ...current, auto_continue: savedValue }
              : previous ? { ...previous, auto_continue: savedValue } : current);
            setError(`Auto continue: ${configErrorMessage(reason)}`);
          } else {
            queryClient.setQueryData(QUERY_KEYS.config, previous);
            setError(`Auto continue was not saved: ${configErrorMessage(reason)}`);
          }
        },
        onSuccess: (saved) => queryClient.setQueryData(QUERY_KEYS.config, saved),
        onSettled: () => { void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.config }); },
      },
    );
  };

  return (
    <section className="config-editor" data-ui="config-editor">
      <SettingsHeader
        eyebrow="Quota recovery"
        title="Automatic continuation"
        description="Continue quota-interrupted chats, including task, pipeline, and Think Tank agents. When the provider reports a reliable reset, Chuck continues a minute after it. Otherwise Chuck checks access hourly, up to six times, in a separate session outside the conversation; checks use normal provider usage. If checks aren’t supported, continue manually."
      >
        {putConfig.isPending && <span role="status">Saving…</span>}
      </SettingsHeader>
      {config.isError && <p className="form-error" role="alert">Auto continue could not be loaded: {configErrorMessage(config.error)}</p>}
      {error && <p className="form-error" role="alert">{error}</p>}
      <label className="toggle-row toggle-row-primary">
        <input
          type="checkbox"
          aria-label="Auto continue after quota reset"
          checked={enabled}
          disabled={config.isLoading || config.isError || putConfig.isPending}
          onChange={(event) => save(event.target.checked)}
        />
        Auto continue after quota reset
      </label>
    </section>
  );
}
