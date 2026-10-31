import type { QueuedPrompt } from "../../api/client";
import { MessageAttachments } from "./AttachmentView";

// Person input retained for the agent's assignment but not yet delivered. Read from the
// prompt endpoint, never appended to the transcript (TS-09.R69/R71).
export function QueuedInput({ agentId, queued, phone = false }: { agentId: string; queued: QueuedPrompt[]; phone?: boolean }) {
  if (!queued.length) return null;
  return <section className={phone ? "phone-card" : "composer-queued"} aria-label="Queued for this assignment">
    <p className={phone ? "phone-card-kicker" : "composer-notice"}>Queued for this assignment — delivered with its next turn</p>
    <ul>{queued.map((item) => <li key={item.request_id}>{item.text && <span>{item.text}</span>}{item.attachments?.length > 0 && <MessageAttachments agentId={agentId} attachments={item.attachments} />}</li>)}</ul>
  </section>;
}
