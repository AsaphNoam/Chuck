import { dismissPromptRecovery, getHeldPrompt, withdrawPrompt, type WithdrawnPrompt } from "../api/client";
import { restoreChatDraft } from "../components/chat/drafts";
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

export async function restoreWithdrawnDraft(agentId: string, result: WithdrawnPrompt) {
  const message = result.withdrawn ? result.message : undefined;
  if (!message?.text || message.attachments?.length) return;
  const saved = restoreChatDraft(agentId, message.text);
  // Never give up the durable server copy when browser storage refused it.
  if (saved && message.request_id) await dismissPromptRecovery(agentId, message.request_id);
}

export async function reconcileSteerRecovery(agentId: string, requestId?: string) {
  invalidateHeldReads(agentId);
  const store = useHeldStore.getState();
  if (requestId && store.recoveryByAgent[agentId]?.request_id === requestId) store.setRecovery(agentId, null);
  await readHeldRecovery(agentId);
}

export function withdrawHeldMessage(agentId: string): Promise<void> {
  const existing = withdrawing.get(agentId);
  if (existing) return existing;
  const run = withdrawOnce(agentId).finally(() => withdrawing.delete(agentId));
  withdrawing.set(agentId, run);
  return run;
}

async function withdrawOnce(agentId: string): Promise<void> {
  const store = useHeldStore.getState();
  const expectedID = store.heldIdByAgent[agentId];
  const result = await withdrawPrompt(agentId, expectedID);
  const message = result.withdrawn ? result.message : undefined;
  const attachments = message?.attachments ?? [];
  const hadAttachments = attachments.length > 0;
  invalidateHeldReads(agentId);
  useHeldStore.getState().release(agentId);
  try {
    if (!hadAttachments) await restoreWithdrawnDraft(agentId, result);
  } finally {
    invalidateHeldReads(agentId);
    if (hadAttachments || message?.request_id) await readHeldRecovery(agentId);
  }
}

// Reads the recovery offer after a completed withdrawal. A failed read is shown
// with a retry that only reads again; the withdrawal is never repeated (INV §8).
export async function readHeldRecovery(agentId: string, isCurrent = () => true, includeHold = false): Promise<void> {
  const read = { agentId, stale: false };
  reads.add(read);
  try {
    const snapshot = await getHeldPrompt(agentId);
    if (read.stale || !isCurrent()) return;
    if (includeHold) {
      if (snapshot.text || snapshot.attachments?.length) useHeldStore.getState().hold(agentId, snapshot.text, snapshot.after_seq, snapshot.attachments, snapshot.held_id);
      else useHeldStore.getState().release(agentId);
    }
    useHeldStore.getState().setRecovery(agentId, snapshot.recovery ?? null);
    useHeldStore.getState().setQueued(agentId, snapshot.queued ?? []);
  } catch {
    if (!read.stale && isCurrent()) useHeldStore.getState().markRecoveryUnread(agentId);
  } finally {
    reads.delete(read);
  }
}
