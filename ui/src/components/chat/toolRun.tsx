import { useState, type ReactNode } from "react";
import type { TranscriptEvent } from "../../api/types";
import { resultsByCall } from "./toolLine";

// Consecutive tool calls and their results fold into one collapsed "Ran N
// tools" row. The desktop transcript and the phone conversation share this
// grouping. A result never renders a row: each call's line folds it in by
// tool_call_id (FS-20.R13, FS-03.R108, INV §2).

// A run's results come from the whole list, so a call whose result lands after
// an unanswered permission or another row still reads its outcome.
export type TranscriptRow = { kind: "event"; event: TranscriptEvent } | { kind: "tool-run"; events: TranscriptEvent[]; results: Map<string, TranscriptEvent> };

export function groupTranscriptRows(events: TranscriptEvent[]): TranscriptRow[] {
  const rows: TranscriptRow[] = [];
  const results = resultsByCall(events);
  for (let index = 0; index < events.length;) {
    const event = events[index];
    if (!isToolEvent(event)) {
      rows.push({ kind: "event", event });
      index++;
      continue;
    }
    const run: TranscriptEvent[] = [];
    while (index < events.length && (isToolEvent(events[index]) || isResolvedPermissionFor(run, events[index]))) run.push(events[index++]);
    if (run.some((item) => kindOf(item) === "tool_call")) rows.push({ kind: "tool-run", events: run, results });
    else run.filter((item) => kindOf(item) !== "tool_result").forEach((item) => rows.push({ kind: "event", event: item }));
  }
  return rows;
}

export function ToolRun({ events, results, renderEvent }: { events: TranscriptEvent[]; results: Map<string, TranscriptEvent>; renderEvent: (event: TranscriptEvent, index: number, result?: TranscriptEvent) => ReactNode }) {
  const [open, setOpen] = useState(false);
  const count = events.filter((event) => kindOf(event) === "tool_call").length;
  return (
    <section className="tool-run" data-ui="tool-run" data-state={open ? "expanded" : "collapsed"}>
      <button type="button" className="tool-toggle" data-slot="trigger" aria-label={`Ran ${count} tool${count === 1 ? "" : "s"}`} aria-expanded={open} onClick={() => setOpen((value) => !value)}>
        {open ? "▾" : "▸"} Ran {count} tool{count === 1 ? "" : "s"}
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
