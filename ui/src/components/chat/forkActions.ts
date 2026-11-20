import type { AgentState, TranscriptEvent } from "../../api/types";

/** The server's turn-end fork affordance (TS-03.R86). */
export type ForkAffordance = { available: boolean; reason?: string };

/** Fork from here: clones the conversation at a completed turn. */
export type ForkAction = {
  label: "Fork from here";
  available: boolean;
  reason?: string;
  throughSeq?: number;
};

/** Edit of the latest sent message in this conversation (FS-01.R61). */
export type EditAction = { seq: number; available: boolean; reason?: string };

/** The agent fields Edit availability reads; the server decides the live gate. */
export type EditAgent = Pick<AgentState, "edit" | "runtime_capabilities">;

const INJECTED_MESSAGE = "This message was sent while the agent was working; edit the message that began that turn.";
const LATER_TURN = "The agent took another turn after this message, so editing it would hide that turn.";
const UNFINISHED_TURN ="This turn is still running; Fork is available once it completes.";

function isRoot(event: TranscriptEvent) {
  return !event.activity_id;
}

function kind(event: TranscriptEvent) {
  return event.kind ?? event.type;
}

function isRootKind(event: TranscriptEvent, type: string) {
  return isRoot(event) && kind(event) === type && typeof event.seq === "number";
}

/**
 * editActionFor offers Edit only on the latest root person message (FS-01.R61).
 * Structure is located here; live availability comes from the agent's server
 * `edit` gate and the previous turn end's fork marker.
 */
export function editActionFor(events: TranscriptEvent[], event: TranscriptEvent, agent?: EditAgent): EditAction | null {
  if (!isRootKind(event, "user_text")) return null;
  const seq = event.seq as number;
  const latest = [...events].reverse().find((item) => isRootKind(item, "user_text"));
  if (!latest || latest.seq !== seq) return null;
  const no = (reason?: string): EditAction => ({ seq, available: false, reason: reason || "Edit is not available right now." });
  if (!agent?.edit?.available) return no(agent?.edit?.reason);
  const previous = [...events].reverse().find((item) => isRootKind(item, "turn_end") && (item.seq as number) < seq);
  const floor = (previous?.seq as number | undefined) ?? 0;
  if (events.some((item) => isRootKind(item, "user_text") && (item.seq as number) > floor && (item.seq as number) < seq)) return no(INJECTED_MESSAGE);
  const endsAfter = events.filter((item) => isRootKind(item, "turn_end") && (item.seq as number) > seq).length;
  if (endsAfter === 0) return no("Wait for this turn to finish.");
  if (endsAfter > 1) return no(LATER_TURN);
  if (events.some((item) => isRootKind(item, "user_text") && (item.seq as number) < seq)) {
    const fork = previous?.fork as ForkAffordance | undefined;
    if (!fork) return no("This turn has no fork information yet.");
    if (!fork.available) return no(fork.reason);
    if (agent.runtime_capabilities?.complete_turn_fork !== true) {
      return no("This agent's adapter cannot prove a complete earlier turn; update to the supported package to edit later messages.");
    }
  }
  return { seq, available: true };
}

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
export type MessageActions = { copy: string; fork: ForkAction | null; edit: EditAction | null };

/**
 * messageActionsFor gives a root person or assistant text message its Copy
 * and Fork from here actions (FS-01.R60, R62); the latest person message also
 * gets Edit (FS-01.R61).
 */
export function messageActionsFor(events: TranscriptEvent[], event: TranscriptEvent, clone?: ForkAffordance, agent?: EditAgent): MessageActions | null {
  const k = kind(event);
  if (!isRoot(event) || typeof event.seq !== "number" || (k !== "user_text" && k !== "assistant_text")) return null;
  return {
    copy: String(event.text ?? event.delta ?? ""),
    fork: forkThroughTurn(events, event, clone),
    edit: k === "user_text" ? editActionFor(events, event, agent) : null,
  };
}
