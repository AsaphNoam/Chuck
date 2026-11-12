import type { TranscriptEvent } from "../../api/types";

/** The server's turn-end fork affordance (TS-03.R86). */
export type ForkAffordance = { available: boolean; reason?: string };

/** One message action that clones the conversation at a chosen turn. */
export type ForkAction = {
  label: "Edit" | "Fork from here";
  available: boolean;
  reason?: string;
  throughSeq?: number;
  composeFromSeq?: number;
};

const FIRST_MESSAGE = "The first message has no earlier turn to fork from.";
const INJECTED_MESSAGE = "This message was sent while the agent was working; edit the message that began that turn.";

function isRoot(event: TranscriptEvent) {
  return !event.activity_id;
}

function kind(event: TranscriptEvent) {
  return event.kind ?? event.type;
}

/**
 * forkActionFor resolves Edit for a person's message and Fork from here for a
 * completed turn's last reply (FS-01.R58, TS-08.R162). Eligibility comes only
 * from the server: the agent's clone affordance and the turn end's `fork`
 * marker. Any other event has no action.
 */
export function forkActionFor(events: TranscriptEvent[], event: TranscriptEvent, clone?: ForkAffordance): ForkAction | null {
  if (!isRoot(event) || typeof event.seq !== "number") return null;
  const seq = event.seq;
  const gate = (action: ForkAction, turnEnd?: TranscriptEvent): ForkAction => {
    if (clone && !clone.available) return { ...action, available: false, reason: clone.reason };
    const fork = turnEnd?.fork as ForkAffordance | undefined;
    if (!fork) return { ...action, available: false, reason: action.reason ?? "This turn has no fork information yet." };
    return fork.available ? { ...action, available: true } : { ...action, available: false, reason: fork.reason };
  };
  if (kind(event) === "user_text") {
    const previous = [...events].reverse().find((item) => isRoot(item) && kind(item) === "turn_end" && typeof item.seq === "number" && item.seq < seq);
    if (!previous) return { label: "Edit", available: false, reason: FIRST_MESSAGE };
    // Only the message that began a turn can be edited; one sent while the
    // agent was working joined that turn (TS-03.R86).
    const began = events.find((item) => isRoot(item) && kind(item) === "user_text" && typeof item.seq === "number" && item.seq > (previous.seq as number));
    if (began && began.seq !== seq) return { label: "Edit", available: false, reason: INJECTED_MESSAGE };
    return gate({ label: "Edit", available: false, throughSeq: previous.seq, composeFromSeq: seq }, previous);
  }
  if (kind(event) !== "assistant_text") return null;
  // The reply is the turn's last: nothing but more of its text follows before
  // that turn's end.
  for (const item of events) {
    if (!isRoot(item) || typeof item.seq !== "number" || item.seq <= seq) continue;
    const next = kind(item);
    if (next === "assistant_text") continue;
    if (next !== "turn_end") return null;
    return gate({ label: "Fork from here", available: false, throughSeq: item.seq }, item);
  }
  return null;
}
