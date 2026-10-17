import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ToolResult } from "./ToolResult";

describe("ToolResult", () => {
  it("omits a successful no-payload outcome", () => {
    const { container } = render(<ToolResult event={{ kind: "tool_result", status: "completed", content: null }} />);

    expect(container.firstChild).toBeNull();
  });

  it("uses a meaningful error when an empty content field accompanies it", () => {
    render(<ToolResult event={{ kind: "tool_result", status: "failed", content: "", error: "Permission denied" }} />);

    expect(screen.getByText("Permission denied")).toBeInTheDocument();
    expect(screen.queryByText("Failed")).not.toBeInTheDocument();
  });

  it("shows the text of an ACP content envelope, keeping JSON for other blocks (FS-03.R2)", () => {
    const text = (value: string) => ({ type: "content", content: { type: "text", text: value } });
    const { container, rerender } = render(<ToolResult event={{ kind: "tool_result", status: "completed", content: [text("```console\nok\n```"), text("done")] }} />);
    expect(container.querySelector("pre")?.textContent).toBe("```console\nok\n```\ndone");

    rerender(<ToolResult event={{ kind: "tool_result", status: "completed", content: [text("a"), { type: "diff", path: "x.go" }] }} />);
    expect(container.querySelector("pre")?.textContent).toContain('"type": "diff"');
  });

  it("keeps long non-empty results expandable", () => {
    const output = "a".repeat(601);
    render(<ToolResult event={{ kind: "tool_result", status: "completed", content: output }} />);

    expect(screen.getByText("Show more")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Show more"));
    expect(screen.getByText("Show less")).toBeInTheDocument();
  });
});
