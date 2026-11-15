import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { dismissPromptRecovery, sendRecoveredPrompt, unacceptedInputDelivery, type HeldPrompt } from "../../api/client";
import { invalidateHeldReads, readHeldRecovery, refreshInputQueries, settleRecoveryLocally } from "../../lib/heldMessage";
import { useHeldStore } from "../../store/heldStore";
import { MessageAttachments } from "./AttachmentView";
import { saveRecoveryAttempt } from "./recoveryAttempts";
import { newRequestId } from "./attachments";

export function RecoveryOffer({ agentId, recovery, phone = false, offline = false }: {
  agentId: string;
  recovery: NonNullable<HeldPrompt["recovery"]>;
  phone?: boolean;
  offline?: boolean;
}) {
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const inFlight = useRef(false);
  const attemptId = useRef<string>();
  const client = useQueryClient();
  const act = async (dismiss: boolean) => {
    if (inFlight.current || offline) return;
    inFlight.current = true;
    setBusy(true);
    setError("");
    try {
      if (dismiss) {
        await dismissPromptRecovery(agentId, recovery.request_id);
      } else {
        attemptId.current ??= newRequestId();
        try { attemptId.current = saveRecoveryAttempt(agentId, recovery.request_id, attemptId.current); } catch { /* Keep the ID for this view. */ }
        const result = await sendRecoveredPrompt(agentId, recovery, attemptId.current);
        const unsuccessful = unacceptedInputDelivery(result);
        if (unsuccessful) {
          setError(unsuccessful === "released"
            ? "This send was released for recovery. Review the current offer."
            : "Delivery could not be confirmed. Review the current offer.");
          return;
        }
      }
      if (phone) {
        await client.cancelQueries({ queryKey: ["held", agentId] });
        client.setQueriesData<HeldPrompt>({ queryKey: ["held", agentId] }, (snapshot) =>
          snapshot?.recovery?.request_id === recovery.request_id ? { ...snapshot, recovery: undefined } : snapshot);
      } else {
        invalidateHeldReads(agentId);
        const store = useHeldStore.getState();
        if (store.recoveryByAgent[agentId]?.request_id === recovery.request_id) store.setRecovery(agentId, null);
      }
      await settleRecoveryLocally(agentId, recovery.request_id).catch(() => undefined);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : phone ? String(reason) : dismiss ? "Could not dismiss this message." : "Could not restore this message.");
    } finally {
      await refreshInputQueries(client, agentId);
      if (!phone && !client.getQueryCache().findAll({ queryKey: ["held", agentId] }).some((query) => query.isActive())) {
        await readHeldRecovery(agentId, () => true, true);
      }
      inFlight.current = false;
      setBusy(false);
    }
  };
  const Container = phone ? "section" : "div";
  return <Container
    className={phone ? "phone-card" : "transcript-held"}
    aria-label={phone ? "Message available to restore" : undefined}
    data-ui={phone ? undefined : "transcript"}
    data-variant={phone ? undefined : "held"}
  >
    <p className={phone ? "phone-card-kicker" : "transcript-held-label"}>
      {recovery.delivery === "unknown" || recovery.delivery === "uncertain" || recovery.delivery === "delivery_uncertain"
        ? "This message may already have reached the agent — sending again may repeat it"
        : "Message not delivered — choose whether to send it again"}
    </p>
    {recovery.text && <p className={phone ? undefined : "transcript-held-text"}>{recovery.text}</p>}
    <MessageAttachments agentId={agentId} attachments={recovery.attachments ?? []} />
    <div className={phone ? "phone-actions" : "transcript-held-actions"}>
      <button type="button" className={phone ? undefined : "ad-button-primary"} disabled={busy || offline} onClick={() => void act(false)}>Send recovered message</button>
      <button type="button" className={phone ? undefined : "ad-button-secondary"} disabled={busy || offline} onClick={() => void act(true)}>Dismiss</button>
    </div>
    {error && <p className={phone ? "phone-error" : "transcript-held-error"} role="alert">{error}</p>}
  </Container>;
}
