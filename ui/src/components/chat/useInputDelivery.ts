import { useRef, useState } from "react";
import { QUEUED_FOR_ASSIGNMENT_NOTICE, sendPrompt, steerPrompt, type PromptResult, type SteerResult } from "../../api/client";
import { chatDraftRevision, discardChatDraft, getChatDraft } from "./drafts";
import {
  attachmentErrorForStorage, chatDraftRequestId, clearPendingAttachmentSend, isDefiniteSendRefusal,
  persistPendingAttachmentSend, removeAcceptedAttachments, type DraftAttachment, type PendingAttachmentSend,
} from "./attachments";

export function sameAttachmentSet(left: DraftAttachment[], right: DraftAttachment[]) {
  return left.length === right.length && left.every((file) => right.some((candidate) => candidate.id === file.id));
}

type Projection = {
  beforeSend?: () => number;
  acceptPrompt?: (result: PromptResult, pending: PendingAttachmentSend, beforeSeq: number) => void;
  acceptSteer: (result: SteerResult, held: boolean) => Promise<void>;
  refreshRecovery: (releaseHold: boolean) => void;
  settled?: () => void;
};

export function useInputDelivery(agentId: string, projection: Projection, view: {
  phone?: boolean;
  clearDraft: () => void;
  restoreDraft?: (text: string) => void;
  setError: (value: string | null) => void;
  setNotice: (value: string | null) => void;
  setAttachmentError: (value: string | null) => void;
}) {
  const [retryState, setRetryState] = useState<{ agentId: string; receipt: PendingAttachmentSend } | null>(null);
  const retry = retryState?.agentId === agentId ? retryState.receipt : null;
  const setRetry = (receipt: PendingAttachmentSend | null) => setRetryState(receipt ? { agentId, receipt } : null);
  const [inFlight, setInFlight] = useState<Record<string, boolean>>({});
  const active = useRef(new Set<string>());
  const currentAgent = useRef(agentId);
  currentAgent.current = agentId;

  const matchingRetry = (text: string, files: DraftAttachment[]) =>
    retry && retry.text === text && sameAttachmentSet(retry.files, files) ? retry : null;

  const deliver = async (text: string, files: DraftAttachment[], steer = false, options: {
    retry?: PendingAttachmentSend;
    held?: { requestId?: string; text: string };
  } = {}) => {
    if (active.current.has(agentId)) return;
    const match = options.held ? null : matchingRetry(text, files);
    if ((options.retry ?? match)?.retryable === false) return;
    const pending = options.retry ?? (!steer || view.phone ? match : null) ?? {
      text, files, steer,
      requestId: options.held?.requestId || (!options.held && chatDraftRequestId(agentId)) || crypto.randomUUID(),
      revision: chatDraftRevision(agentId),
    };
    const promoting = !!options.held;
    const hasFiles = pending.files.length > 0;
    const persist = hasFiles || !!options.retry || (!!match && !promoting && (!steer || view.phone));
    const ownsView = () => currentAgent.current === agentId;
    const error = (message: string) => { if (ownsView()) view.setError(message); };
    const storageError = (message = attachmentErrorForStorage()) => { if (ownsView()) view.setAttachmentError(message); };
    const beforeSeq = projection.beforeSend?.() ?? 0;
    active.current.add(agentId);
    setInFlight((running) => ({ ...running, [agentId]: pending.steer }));
    view.setError(null);
    view.setNotice(null);
    if (!view.phone && !pending.steer && !persist) view.clearDraft();
    if (persist) {
      setRetry(pending);
      try { await persistPendingAttachmentSend(agentId, pending); } catch { storageError(); }
    }
    try {
      const requestFiles = pending.files.map(({ name, blob }) => ({ name, blob }));
      let prompt: PromptResult | undefined;
      let steered: SteerResult | undefined;
      if (pending.steer) {
        steered = await steerPrompt(agentId, promoting ? "" : pending.text, requestFiles, pending.requestId);
        await projection.acceptSteer(steered, promoting);
      } else {
        prompt = await sendPrompt(agentId, pending.text, requestFiles, pending.requestId);
        projection.acceptPrompt?.(prompt, pending, beforeSeq);
      }
      const status = (prompt ?? steered)?.attachment_delivery;
      if (hasFiles && status !== "submitted" && status !== "held" && status !== "queued") {
        error(status === "released"
          ? "This message was released for recovery. Restore it from the conversation before sending again."
          : status === "failed"
            ? `The message was not delivered. Its attachments remain ${view.phone ? "here" : "in the composer"}; retry the same send.`
            : "Delivery could not be confirmed. Its attachments remain available; retry uses the same request receipt.");
        const receipt = { ...pending, retryable: status !== "released" };
        if (ownsView()) setRetry(receipt);
        if (!receipt.retryable) void persistPendingAttachmentSend(agentId, receipt).catch(() => storageError());
        return;
      }
      if (!promoting && chatDraftRevision(agentId) === pending.revision) {
        discardChatDraft(agentId);
        if (ownsView()) view.clearDraft();
      }
      if (hasFiles) {
        try { await removeAcceptedAttachments(agentId, pending.files.map((file) => file.id), getChatDraft(agentId) !== ""); }
        catch { storageError("Message was accepted, but attachment cleanup could not be saved. Remove any sent files before sending again."); }
      }
      if (ownsView()) {
        if (persist) setRetry(null);
        if (prompt?.delivery === "queued") view.setNotice(QUEUED_FOR_ASSIGNMENT_NOTICE);
        else if (view.phone && (prompt?.delivery === "held" || status === "held")) view.setNotice("Held — it will be sent when the current turn ends.");
        else if (!view.phone && steered) view.setNotice(persist || steered.outcome === "steered"
          ? "Delivered into the running turn."
          : "That turn had already ended — sent as a new turn.");
      }
      if (prompt?.delivery === "queued") projection.refreshRecovery(false);
      if (persist) void clearPendingAttachmentSend(agentId, pending.requestId).catch(() =>
        storageError("Message was sent, but its retry receipt could not be cleared from browser storage."));
    } catch (reason) {
      const message = reason instanceof Error && reason.message ? reason.message : view.phone ? String(reason) : "the agent may have stopped";
      const uncertain = (reason as { code?: string } | null)?.code === "delivery_uncertain"
        || (!view.phone && /delivery[_ ]uncertain|may already have reached/i.test(message));
      if (uncertain) {
        projection.refreshRecovery(pending.steer || (!view.phone && !persist));
        if (persist || (view.phone && pending.steer)) {
          const receipt = { ...pending, text: options.held?.text ?? pending.text, retryable: false };
          if (ownsView()) setRetry(receipt);
          if (persist) void persistPendingAttachmentSend(agentId, receipt).catch(() => storageError());
        }
        if (!view.phone && !pending.steer && !persist && chatDraftRevision(agentId) === pending.revision) {
          view.restoreDraft?.(pending.text);
        }
        error(view.phone ? message : pending.steer
          ? "Steer delivery is uncertain — the message may have reached the agent. Do not retry Steer; use Send deliberately if needed."
          : "Send delivery is uncertain — the message may have reached the agent. Check recovery/history, then use Send deliberately if needed.");
      } else if (persist) {
        if (isDefiniteSendRefusal(reason)) {
          if (ownsView()) setRetry(null);
          try { await clearPendingAttachmentSend(agentId, pending.requestId); } catch { storageError(); }
          error(view.phone ? message : `Could not send — ${message}.`);
        } else {
          error(view.phone ? message : `Could not send — ${message}. Your message is still available to retry.`);
        }
      } else if (view.phone) {
        error(message);
      } else if (pending.steer) {
        error(`Could not steer — ${message}.`);
      } else if (chatDraftRevision(agentId) === pending.revision) {
        view.restoreDraft?.(pending.text);
        error(`Failed to send — ${message}. Your message was restored.`);
      }
    } finally {
      active.current.delete(agentId);
      setInFlight(({ [agentId]: _, ...running }) => running);
      projection.settled?.();
    }
  };

  return { deliver, matchingRetry, retry, setRetry, sending: agentId in inFlight, steering: inFlight[agentId] === true, inFlight: active };
}
