import { useState } from "react";
import { decidePermission } from "../../../api/client";
import type { PermissionResolution, TranscriptEvent } from "../../../api/types";
import { hostRequestEnded, PEER_STOP_CONSEQUENCE, peerLabel, peerStopRequest } from "../../../lib/peerStop";
import { useAgentStore } from "../../../store/agentStore";
import { useTranscriptStore } from "../../../store/transcriptStore";

export function PermissionPrompt({ agentId, event }: { agentId: string; event: TranscriptEvent }) {
  const resolve = useTranscriptStore((state) => state.resolvePermission);
  const status = useAgentStore((state) => state.agents[agentId]?.state);
  const toolCallId = String(event.tool_call_id ?? "");
  const peerStop = peerStopRequest(event);
  // A host request outlives nothing: once its requester stopped, it can no
  // longer be decided and replays as cancelled (TS-04.R94).
  const resolved = (event.resolved as PermissionResolution | undefined) ?? (hostRequestEnded(event, status) ? "cancelled" : undefined);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const decide = async (decision: "approve" | "deny") => {
    setError(null);
    setBusy(true);
    try {
      await decidePermission(agentId, toolCallId, decision);
      resolve(agentId, toolCallId, decision);
    } catch {
      setError("Failed to send decision — the agent may have stopped.");
    } finally {
      setBusy(false);
    }
  };
  const label = peerStop
    ? `Stop ${peerStop.target.name || peerStop.target.agent_id}`
    : String(event.name ?? event.tool ?? "Permission required");
  if (resolved) {
    const presentation = resolutionPresentation[resolved];
    return (
      <article
        className="permission-prompt resolved"
        data-ui="permission-prompt"
        data-state={resolved === "approve" ? "approved" : resolved === "deny" ? "denied" : resolved === "cancelled" ? "cancelled" : "timed-out"}
      >
        <strong data-slot="title">{label}</strong>
        <span className="permission-chip" data-slot="resolution">
          {peerStop && resolved === "approve" ? "Approved once" : presentation.label}
        </span>
      </article>
    );
  }
  return (
    <article className="permission-prompt" data-ui="permission-prompt" data-state={error ? "error" : "pending"}>
      <span className="permission-symbol" aria-hidden="true">!</span>
      <strong data-slot="title">{label}</strong>
      {peerStop ? (
        <>
          <dl data-slot="peers">
            <dt>Requested by</dt>
            <dd>{peerLabel(peerStop.requester)}</dd>
            <dt>Agent to stop</dt>
            <dd>{peerLabel(peerStop.target)}</dd>
          </dl>
          <p data-slot="reason">{peerStop.reason}</p>
          <p data-slot="consequence">{PEER_STOP_CONSEQUENCE}</p>
        </>
      ) : (
        <p data-slot="reason">{String(event.reason ?? "")}</p>
      )}
      {error && <p className="permission-error" data-slot="error">{error}</p>}
      {/* The primary Approve sits last, after the safer Deny (FS-12.R61). */}
      <div data-slot="actions">
        <button type="button" className="ad-button-secondary" disabled={busy} onClick={() => void decide("deny")}>Deny</button>
        <button type="button" className="ad-button-primary" disabled={busy} onClick={() => void decide("approve")}>
          {peerStop ? "Approve once" : "Approve"}
        </button>
      </div>
    </article>
  );
}

const resolutionPresentation: Record<PermissionResolution, { label: string }> = {
  approve: { label: "Approved" },
  deny: { label: "Denied" },
  cancelled: { label: "Cancelled" },
  timeout: { label: "Timed out" },
};
