import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { cloneAgentAt } from "../../api/client";
import { useUiStore } from "../../store/uiStore";
import { ConfirmDialog } from "../ui";
import { setChatDraft } from "./drafts";
import type { ForkAction } from "./forkActions";

/**
 * Confirms Edit / Fork from here, calls the clone route once and follows
 * Clone's navigation; Edit's text waits unsent in the new composer
 * (FS-01.R58, TS-08.R162).
 */
export function ForkConfirmDialog({ agentId, action, onClose }: { agentId: string; action: ForkAction | null; onClose: () => void }) {
  const [pending, setPending] = useState(false);
  const navigate = useNavigate();
  const pushError = useUiStore((state) => state.pushError);
  const edit = action?.label === "Edit";
  const confirm = () => {
    if (!action?.throughSeq || pending) return;
    setPending(true);
    cloneAgentAt(agentId, { through_seq: action.throughSeq, compose_from_seq: action.composeFromSeq })
      .then((result) => {
        if (result.compose_text) setChatDraft(result.agent.agent_id, result.compose_text);
        onClose();
        navigate(`/agent/${result.agent.agent_id}`);
      })
      .catch((err) => pushError(`${action.label} failed`, err instanceof Error ? err.message : String(err)))
      .finally(() => setPending(false));
  };
  return (
    <ConfirmDialog
      open={action !== null}
      title={edit ? "Edit in a new agent?" : "Fork from here?"}
      confirmLabel={edit ? "Create agent and edit" : "Create agent"}
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
