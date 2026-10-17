import { describe, expect, it } from "vitest";
import type { TranscriptEvent } from "../../api/types";
import { groupTranscriptRows } from "./toolRun";

const event = (seq: number, kind: string, data: Record<string, unknown>) => ({ seq, kind, type: kind, ...data }) as TranscriptEvent;

describe("groupTranscriptRows", () => {
  // FS-03.A11: a permission-gated call's answered request no longer splits its run.
  it("keeps an approved tool's result inside its run when the permission sits between them", () => {
    const rows = groupTranscriptRows([
      event(1, "tool_call", { tool_call_id: "t1", name: "Edit" }),
      event(2, "permission_request", { tool_call_id: "t1", name: "Edit HANDOFF.md", resolved: "approve" }),
      event(3, "tool_result", { tool_call_id: "t1", status: "completed", content: [{ type: "content", content: { type: "text", text: "ok" } }] }),
    ]);
    expect(rows).toHaveLength(1);
    expect(rows[0]).toMatchObject({ kind: "tool-run" });
    expect(rows[0].kind === "tool-run" && rows[0].events.map((item) => item.seq)).toEqual([1, 2, 3]);
  });

  it("leaves an unanswered or unrelated permission outside the run", () => {
    const pending = groupTranscriptRows([
      event(1, "tool_call", { tool_call_id: "t1", name: "Edit" }),
      event(2, "permission_request", { tool_call_id: "t1", name: "Edit HANDOFF.md" }),
    ]);
    expect(pending.map((row) => row.kind)).toEqual(["tool-run", "event"]);
    const unrelated = groupTranscriptRows([
      event(1, "tool_call", { tool_call_id: "t1", name: "Edit" }),
      event(2, "permission_request", { tool_call_id: "t9", name: "Other", resolved: "approve" }),
    ]);
    expect(unrelated.map((row) => row.kind)).toEqual(["tool-run", "event"]);
  });
});
