import { getHeldPrompt, withdrawPrompt } from "../api/client";
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
  const text = message?.text ?? "";
  const attachments = message?.attachments ?? [];
  const hadAttachments = attachments.length > 0;
  useHeldStore.getState().release(agentId);
  if (hadAttachments) await readHeldRecovery(agentId);
  else if (text) restoreChatDraft(agentId, text);
}

// Reads the recovery offer after a completed withdrawal. A failed read is shown
// with a retry that only reads again; the withdrawal is never repeated (INV §8).
export async function readHeldRecovery(agentId: string): Promise<void> {
  try {
    const snapshot = await getHeldPrompt(agentId);
    useHeldStore.getState().setRecovery(agentId, snapshot.recovery ?? null);
  } catch {
    useHeldStore.getState().markRecoveryUnread(agentId);
  }
}
