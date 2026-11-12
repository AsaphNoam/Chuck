import { useContext, useEffect, useState, type ReactNode } from "react";
import { RevealTaskContext } from "./TurnList";
import type { TranscriptEvent } from "../../api/types";
import { resultsByCall } from "./toolLine";
import { mailOf } from "./renderers/MailDetail";

// Consecutive tool calls and their results fold into one collapsed "Ran N
// tools" row. The desktop transcript and the phone conversation share this
// grouping. A result never renders a row: each call's line folds it in by
// tool_call_id (FS-20.R13, FS-03.R108, INV §2).

// A run's results come from the whole transcript, so a call whose result lands
// after an unanswered permission or another row — even in a later turn part —
// still reads its outcome. Callers rendering a slice pass the transcript's map.
export type TranscriptRow = { kind: "event"; event: TranscriptEvent } | { kind: "tool-run"; events: TranscriptEvent[]; results: Map<string, TranscriptEvent> };

export function groupTranscriptRows(events: TranscriptEvent[], results = resultsByCall(events)): TranscriptRow[] {
  const rows: TranscriptRow[] = [];
  for (let index = 0; index < events.length;) {
    const event = events[index];
    if (!isToolEvent(event) && !isOutgoingMail(event)) {
      rows.push({ kind: "event", event });
      index++;
      continue;
    }
    const run: TranscriptEvent[] = [];
    while (index < events.length && (isToolEvent(events[index]) || isOutgoingMail(events[index]) || isResolvedPermissionFor(run, events[index]))) run.push(events[index++]);
    const attached = attachMail(run);
    if (attached.some((item) => kindOf(item) === "tool_call" || kindOf(item) === "notice")) rows.push({ kind: "tool-run", events: attached, results });
    else attached.filter((item) => kindOf(item) !== "tool_result").forEach((item) => rows.push({ kind: "event", event: item }));
  }
  return rows;
}

// A sent or read mail record joins the run of the call that produced it
// (TS-08.R161). Received mail is a prompt notice and never joins.
function isOutgoingMail(event: TranscriptEvent) {
  if (kindOf(event) !== "notice" || event.origin !== "chuck_mail") return false;
  const direction = mailOf(event)?.direction;
  return direction === "sent" || direction === "read";
}

// Each record attaches to the earliest unattached call of its messaging tool,
// so parallel calls pair in order. An unmatched record stays its own line.
function attachMail(run: TranscriptEvent[]): TranscriptEvent[] {
  const out = [...run];
  const taken = new Set<number>();
  const gone = new Set<number>();
  for (let i = 0; i < out.length; i++) {
    const notice = out[i];
    if (!isOutgoingMail(notice) || gone.has(i)) continue;
    const mail = mailOf(notice)!;
    const tool = mail.direction === "sent" ? "send_message" : "check_messages";
    const target = out.findIndex((item, at) => kindOf(item) === "tool_call" && !taken.has(at) && messagingTool(item) === tool);
    if (target < 0) continue;
    taken.add(target);
    out[target] = { ...out[target], mail };
    gone.add(i);
  }
  return out.filter((_, at) => !gone.has(at));
}

const MESSAGING_SERVERS = new Set(["chuck", "chuck-messaging", "agentdeck-messaging"]);

// The tool name of a call from the Chuck messaging server: the suffix of an
// `mcp__<server>__<tool>` or `mcp.<server>.<tool>` title or name, or Codex's
// `args.tool` (with `args.server` when present). Anything else is "".
function messagingTool(call: TranscriptEvent) {
  const args = (call.args != null && typeof call.args === "object" ? call.args : {}) as Record<string, unknown>;
  for (const text of [call.name, call.title]) {
    const match = /^mcp(?:__([^_].*?)__|\.([^.]+)\.)([A-Za-z0-9_]+)$/.exec(String(text ?? "").trim());
    if (match && MESSAGING_SERVERS.has(match[1] ?? match[2])) return match[3];
  }
  if (typeof args.tool === "string" && (args.server === undefined || MESSAGING_SERVERS.has(String(args.server)))) return args.tool;
  return "";
}

export function ToolRun({ events, results, renderEvent }: { events: TranscriptEvent[]; results: Map<string, TranscriptEvent>; renderEvent: (event: TranscriptEvent, index: number, result?: TranscriptEvent) => ReactNode }) {
  const [open, setOpen] = useState(false);
  const reveal = useContext(RevealTaskContext);
  useEffect(() => { if (reveal !== undefined && events.some((event) => event.seq === reveal.seq)) setOpen(true); }, [reveal]);
  const count = events.filter((event) => kindOf(event) === "tool_call").length;
  // A run holding only unmatched mail records has no calls to count.
  const label = count === 0 ? "Chuck mail" : `Ran ${count} tool${count === 1 ? "" : "s"}`;
  return (
    <section className="tool-run" data-ui="tool-run" data-state={open ? "expanded" : "collapsed"}>
      <button type="button" className="tool-toggle" data-slot="trigger" aria-label={label} aria-expanded={open} onClick={() => setOpen((value) => !value)}>
        {open ? "▾" : "▸"} {label}
      </button>
      {open && (
        <div className="tool-run-content" data-slot="content">
          {events.filter((event) => kindOf(event) !== "tool_result").map((event, index) => renderEvent(event, index, results.get(String(event.tool_call_id ?? ""))))}
        </div>
      )}
    </section>
  );
}

export function kindOf(event: TranscriptEvent) {
  return String(event.kind ?? event.type ?? "");
}

function isToolEvent(event: TranscriptEvent) {
  const kind = kindOf(event);
  return kind === "tool_call" || kind === "tool_result";
}

// A permission-gated call records its request between the call and its result.
// Once answered, the request belongs to that call's run instead of ending it; an
// unanswered request stays outside as the actionable prompt (FS-03.R26).
function isResolvedPermissionFor(run: TranscriptEvent[], event: TranscriptEvent) {
  if (kindOf(event) !== "permission_request" || !event.resolved) return false;
  const id = String(event.tool_call_id ?? "");
  return id !== "" && run.some((item) => kindOf(item) === "tool_call" && String(item.tool_call_id ?? "") === id);
}
