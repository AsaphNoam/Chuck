import type { QueryClient } from "@tanstack/react-query";
import { dismissPromptRecovery, getHeldPrompt, getPromptRecoveryStatus, withdrawPrompt, type WithdrawnPrompt } from "../api/client";
import { chatDraftRevision, discardChatDraft, getChatDraft, restoreChatDraft } from "../components/chat/drafts";
import { chatDraftRequestId, clearPendingAttachmentSend, removeAcceptedAttachments, restorePendingAttachmentSend } from "../components/chat/attachments";
import { clearRecoveryAttempt, recoveryAttempts } from "../components/chat/recoveryAttempts";
import { useHeldStore } from "../store/heldStore";

// Two surfaces offer withdrawing the queued follow-up — the composer (FS-03.R48)
// and the transcript's pending tail (TS-08.R56) — so the server call and the
// client release live in one helper rather than drifting apart in two components
// (INV §2). The client state is released only after the server confirms, so a
// failed withdraw leaves the message visibly pending instead of silently
// disappearing from a chat that is still going to send it (INV §8).
// A withdrawn text message returns to the composer; an attachment-bearing one
// becomes the server's recovery offer instead (FS-03.R84, R90).
// Overlapping withdrawals of one agent's hold, from either surface, share one
// call so the held text is restored exactly once (INV §11).
const withdrawing = new Map<string, Promise<void>>();
// Only outstanding reads are retained. A completed mutation invalidates reads
// that began before it, so a late uncertainty snapshot cannot undo success.
const reads = new Set<{ agentId: string; stale: boolean }>();

export function invalidateHeldReads(agentId: string) {
  for (const read of reads) if (read.agentId === agentId) read.stale = true;
}

export async function refreshInputQueries(client: QueryClient, agentId: string) {
  invalidateHeldReads(agentId);
  const filters = { queryKey: ["held", agentId] };
  await client.cancelQueries(filters);
  await client.invalidateQueries(filters);
}

// A different latest offer says nothing about this exact receipt. Clear the
// browser retry and its submitted draft only on evidence for that receipt.
export async function settleRecoveryLocally(agentId: string, requestId: string) {
  try { clearRecoveryAttempt(agentId, requestId); } catch { /* Retry storage is optional. */ }
  const pending = await restorePendingAttachmentSend(agentId);
  if (pending?.requestId !== requestId) return;
  const revision = chatDraftRevision(agentId);
  const ownsDraft = () => chatDraftRequestId(agentId) === (pending.draftRequestId || pending.requestId)
    && getChatDraft(agentId) === pending.text && chatDraftRevision(agentId) === revision;
  try {
    if (ownsDraft()) {
      discardChatDraft(agentId);
      if (pending.files.length) {
        await removeAcceptedAttachments(agentId, pending.files.map((file) => file.id), getChatDraft(agentId) !== "");
      }
    }
  } finally {
    await clearPendingAttachmentSend(agentId, requestId);
  }
}

export async function reconcileRecoverySettlement(agentId: string, visibleId?: string) {
  try {
    const pending = await restorePendingAttachmentSend(agentId);
    const attempts = new Map(recoveryAttempts(agentId));
    if (pending && !attempts.has(pending.requestId)) attempts.set(pending.requestId, null);
    attempts.delete(visibleId ?? "");
    await Promise.allSettled([...attempts].map(async ([requestId, attemptId]) => {
      const status = await getPromptRecoveryStatus(agentId, requestId);
      if (status === "settled") await settleRecoveryLocally(agentId, requestId);
      else if (status === "unavailable" && attemptId) clearRecoveryAttempt(agentId, requestId, attemptId);
    }));
  } catch { /* A failed read leaves the original receipt and retry key available. */ }
}

export async function restoreWithdrawnDraft(agentId: string, result: WithdrawnPrompt) {
  const message = result.withdrawn ? result.message : undefined;
  if (!message?.text || message.attachments?.length) return;
  const saved = restoreChatDraft(agentId, message.text);
  // Never give up the durable server copy when browser storage refused it.
  if (!saved || !message.request_id) return;
  try {
    await dismissPromptRecovery(agentId, message.request_id);
  } catch (error) {
    if ((error as { code?: string } | null)?.code !== "recovery_changed" ||
        await getPromptRecoveryStatus(agentId, message.request_id) !== "settled") throw error;
  }
}

export async function reconcileSteerRecovery(agentId: string, requestId?: string) {
  invalidateHeldReads(agentId);
  const store = useHeldStore.getState();
  if (requestId && store.recoveryByAgent[agentId]?.request_id === requestId) store.setRecovery(agentId, null);
  await readHeldRecovery(agentId);
}

export function withdrawHeldMessage(agentId: string, projection?: {
  heldId?: string;
  refresh: () => Promise<void>;
}): Promise<void> {
  const existing = withdrawing.get(agentId);
  if (existing) return existing;
  const run = withdrawOnce(agentId, projection).finally(() => withdrawing.delete(agentId));
  withdrawing.set(agentId, run);
  return run;
}

async function withdrawOnce(agentId: string, projection?: {
  heldId?: string;
  refresh: () => Promise<void>;
}): Promise<void> {
  const store = useHeldStore.getState();
  const expectedID = projection ? projection.heldId : store.heldIdByAgent[agentId];
  const result = await withdrawPrompt(agentId, expectedID);
  const message = result.withdrawn ? result.message : undefined;
  const attachments = message?.attachments ?? [];
  const hadAttachments = attachments.length > 0;
  invalidateHeldReads(agentId);
  if (!projection) useHeldStore.getState().release(agentId, expectedID);
  try {
    if (!hadAttachments) await restoreWithdrawnDraft(agentId, result);
  } finally {
    invalidateHeldReads(agentId);
    if (projection) await projection.refresh();
    else if (hadAttachments || message?.request_id) await readWithdrawnRecovery(agentId);
  }
}

// Reads the recovery offer after a confirmed withdrawal. A failed read is shown
// with a retry that only reads again; the withdrawal is never repeated (INV §8).
export function readWithdrawnRecovery(agentId: string): Promise<void> {
  return readHeldRecovery(agentId, () => true, false, true);
}

// Reads the hold, queue and recovery offer. Only a read that follows a confirmed
// withdrawal may report a failure as an unloaded withdrawn message (FS-03.R90).
export async function readHeldRecovery(agentId: string, isCurrent = () => true, includeHold = false, afterWithdrawal = false): Promise<void> {
  invalidateHeldReads(agentId);
  const read = { agentId, stale: false };
  reads.add(read);
  const holdRevision = useHeldStore.getState().holdRevisionByAgent[agentId] ?? 0;
  try {
    const snapshot = await getHeldPrompt(agentId);
    if (read.stale || !isCurrent()) return;
    // A Send accepted a hold after this read began; the snapshot cannot know
    // about it, so it leaves the local hold alone.
    if (includeHold && (useHeldStore.getState().holdRevisionByAgent[agentId] ?? 0) === holdRevision) {
      if (snapshot.text || snapshot.attachments?.length) useHeldStore.getState().hold(agentId, snapshot.text, snapshot.after_seq, snapshot.attachments, snapshot.held_id, snapshot.request_id);
      else useHeldStore.getState().release(agentId);
    }
    useHeldStore.getState().setRecovery(agentId, snapshot.recovery ?? null);
    useHeldStore.getState().setQueued(agentId, snapshot.queued ?? []);
    void reconcileRecoverySettlement(agentId, snapshot.recovery?.request_id);
  } catch {
    if (afterWithdrawal && !read.stale && isCurrent()) useHeldStore.getState().markRecoveryUnread(agentId);
    if (!read.stale && isCurrent()) void reconcileRecoverySettlement(agentId);
  } finally {
    reads.delete(read);
  }
}
