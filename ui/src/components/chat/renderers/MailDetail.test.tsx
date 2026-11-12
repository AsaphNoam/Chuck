import React from "react";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import type { TranscriptEvent } from "../../../api/types";
import { NoticeRow } from "./NoticeRow";
import { ToolCall } from "./ToolCall";

afterEach(cleanup);

const message = { from_name: "Alice", to_name: "Bob", subject: "Plan", body: "Read **this** now", created_at: "2026-11-12T10:00:00Z", wake: true };
const notice = (mail: Record<string, unknown>) => ({ kind: "notice", origin: "chuck_mail", severity: "info", title: "Mail from Alice", mail }) as TranscriptEvent;
const open = () => fireEvent.click(screen.getByRole("button", { name: /Mail/ }));

describe("mail detail", () => {
  it("received: collapsed hides the body; expanded shows sender, subject, waking and bold Markdown", () => {
    const { container } = render(<NoticeRow event={notice({ direction: "received", messages: [message], truncated: false })} />);
    expect(screen.queryByText("Plan")).toBeNull();
    open();
    expect(screen.getByText("From Alice")).toBeTruthy();
    expect(screen.getByText("Plan")).toBeTruthy();
    expect(screen.getByText("waking")).toBeTruthy();
    expect(container.querySelector(".mail-body strong")?.textContent).toBe("this");
  });

  it("a received notice without mail has no disclosure", () => {
    render(<NoticeRow event={{ kind: "notice", origin: "chuck_mail", title: "Mail" } as TranscriptEvent} />);
    expect(screen.queryByRole("button")).toBeNull();
  });

  it.each([
    ["an older saved notice", undefined],
    ["a notice with mail detail", { direction: "received", messages: [{ ...message, from_name: "Dashboard user", subject: "Annotations" }], truncated: false }],
  ])("shows the annotation sender once in %s", (_label, mail) => {
    render(<NoticeRow event={{
      kind: "notice",
      origin: "chuck_mail",
      severity: "info",
      title: "Mail from Dashboard user",
      description: "Dashboard user: Annotations",
      mail,
    } as TranscriptEvent} />);
    const summary = screen.getByRole("note").textContent ?? "";
    expect(summary).toContain("Annotations");
    expect(summary.match(/Dashboard user/g)).toHaveLength(1);
  });

  it("preserves sender and subject associations in a mixed batch", () => {
    render(<NoticeRow event={{ kind: "notice", origin: "chuck_mail", title: "Mail from Alice, Bob", description: "Alice: Plan\nBob: Review" } as TranscriptEvent} />);
    expect(screen.getByRole("note").textContent).toContain("Alice: Plan\nBob: Review");
  });

  it("sent: shows the recipient, deferred, the error and the truncated note on the tool line", () => {
    const call = { kind: "tool_call", name: "mcp__chuck-messaging__send_message", mail: { direction: "sent", messages: [{ ...message, wake: false }], error: { code: "unknown_recipient", message: "No such agent" }, truncated: true } } as TranscriptEvent;
    render(<ToolCall event={call} />);
    open();
    expect(screen.getByText("To Bob")).toBeTruthy();
    expect(screen.getByText("deferred")).toBeTruthy();
    expect(screen.getByText("unknown_recipient: No such agent")).toBeTruthy();
    expect(screen.getByText("Some message bodies were omitted to fit the record.")).toBeTruthy();
  });

  it("read: an empty result says so", () => {
    render(<NoticeRow event={notice({ direction: "read", messages: [], truncated: false })} />);
    open();
    expect(screen.getByText("No messages returned.")).toBeTruthy();
  });
});
