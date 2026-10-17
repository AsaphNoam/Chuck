import { describe, expect, it } from "vitest";
import type { TranscriptEvent } from "../../api/types";
import { groupTranscriptRows } from "./toolRun";

const event = (seq: number, kind: string, data: Record<string, unknown>) => ({ seq, kind, type: kind, ...data }) as TranscriptEvent;

describe("groupTranscriptRows", () => {
  // TOOLRUN-01 (HANDOFF review findings): a resolved permission between a tool
  // call and its result ends the run, so the approved tool's result renders
  // outside its collapsed "Ran n tools" row on desktop and phone. Un-skip with
  // the fix.
  it.skip("keeps an approved tool's result inside its run when the permission sits between them", () => {
    const rows = groupTranscriptRows([
      event(1, "tool_call", { tool_call_id: "t1", name: "Edit" }),
      event(2, "permission_request", { tool_call_id: "t1", name: "Edit HANDOFF.md", resolved: "approve" }),
      event(3, "tool_result", { tool_call_id: "t1", status: "completed", content: [{ type: "content", content: { type: "text", text: "ok" } }] }),
    ]);
    expect(rows.filter((row) => row.kind === "event" && row.event.kind === "tool_result")).toEqual([]);
  });
});
