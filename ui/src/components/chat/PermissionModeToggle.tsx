import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getSessionModes, setSessionMode, type SessionModes } from "../../api/client";

// The live session's own permission modes as one choice (FS-03.R88). Only the
// provider's advertisement is offered, so the control is absent when there is
// none. A turn can move the mode itself (Claude leaving plan mode), so every
// agent state change rereads it; the previous answer stays visible meanwhile.
export function PermissionModeToggle({ agentId, agentState }: { agentId: string; agentState: string }) {
  const queryClient = useQueryClient();
  const queryKey = ["session-modes", agentId, agentState];
  const modes = useQuery({ queryKey, queryFn: () => getSessionModes(agentId), retry: false, placeholderData: (previous: SessionModes | undefined) => previous });
  const [error, setError] = useState<string | null>(null);
  const change = useMutation({
    mutationFn: (mode: string) => setSessionMode(agentId, mode),
    onMutate: () => setError(null),
    onSuccess: (applied) => queryClient.setQueryData(queryKey, { current: applied.current ?? "", options: applied.options ?? [] }),
    onError: (err) => {
      setError(err instanceof Error ? err.message : String(err));
      void queryClient.invalidateQueries({ queryKey: ["session-modes", agentId] });
    },
  });
  const options = modes.data?.options ?? [];
  if (options.length === 0) return null;
  const current = change.isPending ? change.variables : modes.data?.current;
  return (
    <fieldset className="chat-permission-modes" disabled={change.isPending}>
      <legend>Permissions</legend>
      <div className="chat-permission-mode-options">
        {options.map((option) => (
          <label key={option.id} className="chat-permission-mode" title={option.description || undefined}>
            <input type="radio" name={`permission-mode-${agentId}`} value={option.id} checked={option.id === current} onChange={() => change.mutate(option.id)} />
            {option.name}
          </label>
        ))}
      </div>
      {error && <p className="form-error" role="alert">{error}</p>}
    </fieldset>
  );
}
