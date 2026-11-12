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

describe("groupTranscriptRows mail records (TS-08.R161)", () => {
  const mail = (seq: number, direction: string, subject = "s") =>
    event(seq, "notice", { origin: "chuck_mail", severity: "info", title: "Mail", mail: { direction, messages: [{ subject, body: "b", wake: true }], truncated: false } });
  const call = (seq: number, id: string, name: string, extra: Record<string, unknown> = {}) => event(seq, "tool_call", { tool_call_id: id, name, ...extra });
  const run = (rows: ReturnType<typeof groupTranscriptRows>) => {
    const row = rows[0];
    if (row.kind !== "tool-run") throw new Error("not a run");
    return row.events;
  };

  it("attaches a sent record to the send_message call and a read record to check_messages", () => {
    const rows = groupTranscriptRows([
      call(1, "a", "mcp__chuck-messaging__send_message"),
      mail(2, "sent"),
      call(3, "b", "mcp.chuck-messaging.check_messages"),
      mail(4, "read"),
    ]);
    expect(rows).toHaveLength(1);
    const events = run(rows);
    expect(events.map((item) => item.seq)).toEqual([1, 3]);
    expect((events[0].mail as { direction: string }).direction).toBe("sent");
    expect((events[1].mail as { direction: string }).direction).toBe("read");
  });

  it("matches Codex args.tool and pairs parallel same-tool calls in order", () => {
    const events = run(groupTranscriptRows([
      call(1, "a", "mcp", { args: { server: "chuck-messaging", tool: "send_message" } }),
      call(2, "b", "mcp__chuck-messaging__send_message"),
      mail(3, "sent", "first"),
      mail(4, "sent", "second"),
    ]));
    expect(events.map((item) => (item.mail as { messages: { subject: string }[] }).messages[0].subject)).toEqual(["first", "second"]);
  });

  it("keeps an unmatched record as its own line in the run, and ignores other servers' tools", () => {
    const events = run(groupTranscriptRows([call(1, "a", "mcp__other__send_message"), call(2, "b", "Edit"), mail(3, "sent")]));
    expect(events.map((item) => item.seq)).toEqual([1, 2, 3]);
    expect(events[0].mail).toBeUndefined();
  });

  it("turns a record outside any run into a one-line run", () => {
    const rows = groupTranscriptRows([event(1, "assistant_text", { text: "hi" }), mail(2, "read")]);
    expect(rows.map((row) => row.kind)).toEqual(["event", "tool-run"]);
  });

  it("never pulls a received notice or an old mail-less notice into a run", () => {
    const old = event(3, "notice", { origin: "chuck_mail", severity: "info", title: "Mail" });
    const rows = groupTranscriptRows([call(1, "a", "Edit"), mail(2, "received"), old]);
    expect(rows.map((row) => row.kind)).toEqual(["tool-run", "event", "event"]);
    expect(run(rows).map((item) => item.seq)).toEqual([1]);
  });
});
