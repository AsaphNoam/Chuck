import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { cloneAgentAt } from "../../api/client";
import { useUiStore } from "../../store/uiStore";
import { ConfirmDialog } from "../ui";
import type { ForkAction } from "./forkActions";

/**
 * Confirms Fork from here and calls the clone route once, following Clone's
 * navigation (FS-01.R58, TS-08.R162).
 */
export function ForkConfirmDialog({ agentId, action, onClose }: { agentId: string; action: ForkAction | null; onClose: () => void }) {
  const [pending, setPending] = useState(false);
  const navigate = useNavigate();
  const pushError = useUiStore((state) => state.pushError);
  const confirm = () => {
    if (!action?.throughSeq || pending) return;
    setPending(true);
    cloneAgentAt(agentId, { through_seq: action.throughSeq })
      .then((result) => {
        onClose();
        navigate(`/agent/${result.agent.agent_id}`);
      })
      .catch((err) => pushError(`${action.label} failed`, err instanceof Error ? err.message : String(err)))
      .finally(() => setPending(false));
  };
  return (
    <ConfirmDialog
      open={action !== null}
      title="Fork from here?"
      confirmLabel="Create agent"
      pending={pending}
      onCancel={onClose}
      onConfirm={confirm}
    >
      <p>
        This creates a new agent whose conversation ends at this point; this conversation is unchanged.
        Files changed since this point are not restored.
      </p>
    </ConfirmDialog>
  );
}
