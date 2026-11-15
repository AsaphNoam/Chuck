import { useEffect, useRef, useState } from "react";
import { QUEUED_FOR_ASSIGNMENT_NOTICE, sendPrompt, steerPrompt, unacceptedInputDelivery, type PromptResult, type SteerResult } from "../../api/client";
import { chatDraftRevision, discardChatDraft, getChatDraft, republishChatDraft } from "./drafts";
import {
  attachmentErrorForStorage, chatDraftRequestId, clearPendingAttachmentSend, isDefiniteSendRefusal,
  persistPendingAttachmentSend, removeAcceptedAttachments, restoreAttachments, restorePendingAttachmentSend,
  subscribePendingSendCleared, type DraftAttachment, type PendingAttachmentSend,
} from "./attachments";

function sameAttachmentSet(left: DraftAttachment[], right: DraftAttachment[]) {
  return left.length === right.length && left.every((file) => right.some((candidate) => candidate.id === file.id));
}

type Projection = {
  beforeSend?: () => number;
  acceptPrompt?: (result: PromptResult, pending: PendingAttachmentSend, beforeSeq: number) => void;
  acceptSteer: (result: SteerResult, heldId?: string) => Promise<void>;
  refreshRecovery: (heldId?: string) => void;
  settled?: () => void;
};

export function useInputDelivery(agentId: string, projection: Projection, view: {
  phone?: boolean;
  clearDraft: () => void;
  setError: (value: string | null) => void;
  setNotice: (value: string | null) => void;
  setAttachmentError: (value: string | null) => void;
}) {
  const [retryState, setRetryState] = useState<{ agentId: string; receipt: PendingAttachmentSend } | null>(null);
  const retry = retryState?.agentId === agentId ? retryState.receipt : null;
  const setRetry = (receipt: PendingAttachmentSend | null) => setRetryState(receipt ? { agentId, receipt } : null);
  const [inFlight, setInFlight] = useState<Record<string, true>>({});
  const active = useRef(new Set<string>());
  const currentAgent = useRef(agentId);
  currentAgent.current = agentId;
  const { setAttachmentError } = view;
  useEffect(() => {
    let current = true;
    let cleared: string | undefined;
    const initialRevision = chatDraftRevision(agentId);
    const unsubscribe = subscribePendingSendCleared(agentId, (requestId) => {
      cleared = requestId;
      setRetryState((previous) => previous?.agentId === agentId && previous.receipt.requestId === requestId ? null : previous);
    });
    void restorePendingAttachmentSend(agentId).then(async (pending) => {
      if (!current || !pending) return;
      const files = await restoreAttachments(agentId);
      if (!current || cleared === pending.requestId || active.current.has(agentId)) return;
      const revision = chatDraftRevision(agentId);
      const sameDraft = getChatDraft(agentId) === pending.text && sameAttachmentSet(pending.files, files) &&
        (!pending.draftRequestId || pending.draftRequestId === chatDraftRequestId(agentId));
      setRetryState({ agentId, receipt: { ...pending, revision: revision === initialRevision && sameDraft ? revision : -1 } });
    }).catch(() => { if (current) setAttachmentError("A previous send could not be restored from browser storage."); });
    return () => { current = false; unsubscribe(); };
  }, [agentId, setAttachmentError]);

  const matchingRetry = (text: string, files: DraftAttachment[]) =>
    retry && retry.text === text && sameAttachmentSet(retry.files, files) &&
      (!retry.draftRequestId || retry.draftRequestId === chatDraftRequestId(agentId)) ? retry : null;

  const deliver = async (text: string, files: DraftAttachment[], steer = false, options: {
    retry?: PendingAttachmentSend;
    held?: { id: string; requestId?: string };
  } = {}) => {
    if (active.current.has(agentId)) {
      view.setNotice("Wait for the current send to finish.");
      return;
    }
    const match = options.held ? null : matchingRetry(text, files);
    if ((options.retry ?? match)?.retryable === false) return;
    const draftRequestId = chatDraftRequestId(agentId);
    const pending = options.retry ?? (!steer || view.phone ? match : null) ?? {
      text, files, steer,
      requestId: options.held?.requestId || (!options.held && draftRequestId) || crypto.randomUUID(),
      draftRequestId,
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
    setInFlight((running) => ({ ...running, [agentId]: true }));
    view.setError(null);
    view.setNotice(null);
    if (!view.phone && !pending.steer && !persist) view.clearDraft();
    if (persist) {
      setRetry(pending);
      try { await persistPendingAttachmentSend(agentId, pending); } catch { storageError(); }
    }
    const restoreHiddenDraft = () => {
      if (!view.phone && !pending.steer && !persist) republishChatDraft(agentId, pending.text, pending.revision);
    };
    const retainForRecovery = async () => {
      if (!promoting) {
        const receipt = { ...pending, retryable: false };
        if (ownsView()) setRetry(receipt);
        try { await persistPendingAttachmentSend(agentId, receipt); } catch { if (hasFiles) storageError(); }
        restoreHiddenDraft();
      }
      projection.refreshRecovery(options.held?.id);
    };
    try {
      const requestFiles = pending.files.map(({ name, blob }) => ({ name, blob }));
      let prompt: PromptResult | undefined;
      let steered: SteerResult | undefined;
      if (pending.steer) {
        steered = await steerPrompt(agentId, promoting ? "" : pending.text, requestFiles, options.held ? options.held.requestId : pending.requestId, options.held?.id);
      } else {
        prompt = await sendPrompt(agentId, pending.text, requestFiles, pending.requestId);
      }
      const status = (prompt ?? steered)?.attachment_delivery;
      const unsuccessful = unacceptedInputDelivery((prompt ?? steered)!);
      if (unsuccessful === "released" || unsuccessful === "unknown") {
        error(unsuccessful === "released"
          ? "This message was released for recovery. Restore it from the conversation before sending again."
          : "Delivery could not be confirmed. Check the conversation for the recovered message.");
        await retainForRecovery();
        return;
      }
      if (unsuccessful === "failed" || (hasFiles && status !== "submitted" && status !== "held" && status !== "queued")) {
        restoreHiddenDraft();
        error("Delivery could not be confirmed. The draft remains available to retry.");
        return;
      }
      if (prompt) projection.acceptPrompt?.(prompt, pending, beforeSeq);
      if (steered) await projection.acceptSteer(steered, options.held?.id);
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
        else if (steered) view.setNotice(steered.outcome === "steered"
          ? "Delivered into the running turn."
          : "That turn had already ended — sent as a new turn.");
      }
      if (prompt?.delivery === "queued") projection.refreshRecovery();
      if (persist) void clearPendingAttachmentSend(agentId, pending.requestId).catch(() =>
        storageError("Message was sent, but its retry receipt could not be cleared from browser storage."));
    } catch (reason) {
      const message = reason instanceof Error && reason.message ? reason.message : view.phone ? String(reason) : "the agent may have stopped";
      const code = (reason as { code?: string } | null)?.code;
      const uncertain = code === "delivery_uncertain"
        || (!view.phone && /delivery[_ ]uncertain|may already have reached/i.test(message));
      if (uncertain) {
        await retainForRecovery();
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
        if (code === "held_changed") projection.refreshRecovery();
        error(`Could not steer — ${message}.`);
      } else if (chatDraftRevision(agentId) === pending.revision) {
        restoreHiddenDraft();
        error(`Failed to send — ${message}. Your message was restored.`);
      }
    } finally {
      active.current.delete(agentId);
      setInFlight(({ [agentId]: _, ...running }) => running);
      projection.settled?.();
    }
  };

  return { deliver, matchingRetry, retry, sending: agentId in inFlight };
}
