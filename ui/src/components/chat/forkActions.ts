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
  return forkThroughTurn(events, event, clone);
}

const UNFINISHED_TURN = "This turn is still running; Fork is available once it completes.";

/**
 * forkThroughTurn resolves Fork from here for any root text message: the
 * clone ends at the completed root turn containing it (FS-01.R60). The
 * server's turn-end `fork` marker and the agent's clone affordance decide
 * availability (TS-03.R86); the client only locates the turn.
 */
export function forkThroughTurn(events: TranscriptEvent[], event: TranscriptEvent, clone?: ForkAffordance): ForkAction | null {
  if (!isRoot(event) || typeof event.seq !== "number") return null;
  const seq = event.seq;
  const turnEnd = events.find((item) => isRoot(item) && kind(item) === "turn_end" && typeof item.seq === "number" && item.seq > seq);
  const action: ForkAction = { label: "Fork from here", available: false, throughSeq: turnEnd?.seq as number | undefined };
  if (!turnEnd) return { ...action, reason: UNFINISHED_TURN };
  if (clone && !clone.available) return { ...action, reason: clone.reason };
  const fork = turnEnd.fork as ForkAffordance | undefined;
  if (!fork) return { ...action, reason: "This turn has no fork information yet." };
  return fork.available ? { ...action, available: true } : { ...action, reason: fork.reason };
}

/** One shared message action, driving both the hover row and the context menu (TS-08.R171). */
export type MessageActions = { copy: string; fork: ForkAction | null; edit: ForkAction | null };

/**
 * messageActionsFor gives a root person or assistant text message its Copy
 * and Fork from here actions (FS-01.R60, R62). `edit` is the shipped
 * clone-based Edit (FS-01.R58), offered only in the context menu until the
 * current-conversation Edit (FS-01.R61) replaces it.
 */
export function messageActionsFor(events: TranscriptEvent[], event: TranscriptEvent, clone?: ForkAffordance): MessageActions | null {
  const k = kind(event);
  if (!isRoot(event) || typeof event.seq !== "number" || (k !== "user_text" && k !== "assistant_text")) return null;
  return {
    copy: String(event.text ?? event.delta ?? ""),
    fork: forkThroughTurn(events, event, clone),
    edit: k === "user_text" ? forkActionFor(events, event, clone) : null,
  };
}
