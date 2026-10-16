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
export async function withdrawHeldMessage(agentId: string): Promise<void> {
  const store = useHeldStore.getState();
  const text = store.byAgent[agentId] ?? "";
  const hadAttachments = (store.attachmentsByAgent[agentId] ?? []).length > 0;
  await withdrawPrompt(agentId);
  useHeldStore.getState().release(agentId);
  if (hadAttachments) {
    const snapshot = await getHeldPrompt(agentId).catch(() => null);
    if (snapshot) useHeldStore.getState().setRecovery(agentId, snapshot.recovery ?? null);
  } else if (text) {
    restoreChatDraft(agentId, text);
  }
}
