import type { QueuedPrompt } from "../../api/client";
import { MessageAttachments } from "./AttachmentView";

// Person input retained for the agent's assignment but not yet delivered. Read from the
// prompt endpoint, never appended to the transcript; on desktop it sits at the transcript's
// end beside a held message, as pending input (FS-03.R48, TS-09.R69/R71).
export function QueuedInput({ agentId, queued, phone = false }: { agentId: string; queued: QueuedPrompt[]; phone?: boolean }) {
  if (!queued.length) return null;
  const items = queued.map((item) => <li key={item.request_id}>{item.text && <p className={phone ? undefined : "transcript-held-text"}>{item.text}</p>}{item.attachments?.length > 0 && <MessageAttachments agentId={agentId} attachments={item.attachments} />}</li>);
  if (phone) {
    return <section className="phone-card" aria-label="Queued for this assignment">
      <p className="phone-card-kicker">Queued for the assignment’s next turn</p>
      <ul>{items}</ul>
    </section>;
  }
  return <section className="transcript-held" data-ui="transcript" data-variant="queued" aria-label="Queued for this assignment">
    <p className="transcript-held-label">Queued for the assignment’s next turn</p>
    <ul className="transcript-queued-list">{items}</ul>
  </section>;
}
