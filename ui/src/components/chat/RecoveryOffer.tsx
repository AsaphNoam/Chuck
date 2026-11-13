import { useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { dismissPromptRecovery, sendRecoveredPrompt, type HeldPrompt } from "../../api/client";
import { invalidateHeldReads } from "../../lib/heldMessage";
import { useHeldStore } from "../../store/heldStore";
import { MessageAttachments } from "./AttachmentView";

export function RecoveryOffer({ agentId, recovery, phone = false, offline = false }: {
  agentId: string;
  recovery: NonNullable<HeldPrompt["recovery"]>;
  phone?: boolean;
  offline?: boolean;
}) {
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const inFlight = useRef(false);
  const client = useQueryClient();
  const key = `chuck-recovery-send:${agentId}:${recovery.request_id}`;
  const act = async (dismiss: boolean) => {
    if (inFlight.current || offline) return;
    inFlight.current = true;
    setBusy(true);
    setError("");
    try {
      if (dismiss) {
        await dismissPromptRecovery(agentId, recovery.request_id);
      } else {
        let id = localStorage.getItem(key);
        if (!id) {
          id = crypto.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`;
          localStorage.setItem(key, id);
        }
        await sendRecoveredPrompt(agentId, recovery.text, (recovery.attachments ?? []).map((item) => item.id), id);
      }
      localStorage.removeItem(key);
      if (!phone) {
        invalidateHeldReads(agentId);
        const store = useHeldStore.getState();
        if (store.recoveryByAgent[agentId]?.request_id === recovery.request_id) store.setRecovery(agentId, null);
      }
      await client.invalidateQueries({ queryKey: ["held", agentId] });
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : phone ? String(reason) : dismiss ? "Could not dismiss this message." : "Could not restore this message.");
    } finally {
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
