import React from "react";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DiffBlock } from "./DiffBlock";

vi.mock("react-diff-viewer-continued", () => ({
  default: ({ onLineNumberClick }: { onLineNumberClick: (lineId: string) => void }) => (
    <>
      <button type="button" onClick={() => onLineNumberClick("L-2")}>Old line 2</button>
      {/* The real unified viewer passes an added row's id with an undefined prefix. */}
      <button type="button" onClick={() => onLineNumberClick("undefined-1")}>Added line 1</button>
    </>
  ),
}));

afterEach(cleanup);

describe("DiffBlock", () => {
  it("captures a line selected with the diff viewer's line-id format", () => {
    const onAnnotate = vi.fn();
    render(
      <DiffBlock
        event={{ seq: 7, path: "main.go", old_text: "first\nsecond\nthird", new_text: "replacement" }}
        onAnnotate={onAnnotate}
      />,
    );

    expect(screen.getByText("Click line numbers to select a range.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Old line 2" }));
    fireEvent.click(screen.getByRole("button", { name: "Annotate lines 2–2" }));

    expect(onAnnotate).toHaveBeenCalledWith({
      seq: 7,
      path: "main.go",
      side: "old",
      start_line: 2,
      end_line: 2,
      excerpt: "second",
      instruction: "",
    });
  });

  it("anchors an added line on the new side", () => {
    const onAnnotate = vi.fn();
    render(
      <DiffBlock
        event={{ seq: 7, path: "main.go", old_text: "first", new_text: "replacement\nsecond" }}
        onAnnotate={onAnnotate}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Added line 1" }));
    fireEvent.click(screen.getByRole("button", { name: "Annotate lines 1–1" }));

    expect(onAnnotate).toHaveBeenCalledWith(expect.objectContaining({ side: "new", start_line: 1, end_line: 1, excerpt: "replacement" }));
  });
});

// FS-05.A20 (R37) — the diff's file heading opens the same viewer while the
// diff keeps its own content and line-selection behavior.
describe("the diff heading as a file-viewer entry point", () => {
  it("opens the diff's file and keeps line selection working", () => {
    const onOpenFile = vi.fn();
    const onAnnotate = vi.fn();
    render(
      <DiffBlock
        event={{ seq: 7, path: "main.go", old_text: "first\nsecond\nthird", new_text: "replacement" }}
        onAnnotate={onAnnotate}
        onOpenFile={onOpenFile}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "main.go" }));
    expect(onOpenFile).toHaveBeenCalledWith({ path: "main.go" });

    fireEvent.click(screen.getByRole("button", { name: "Old line 2" }));
    fireEvent.click(screen.getByRole("button", { name: "Annotate lines 2–2" }));
    expect(onAnnotate).toHaveBeenCalledWith(expect.objectContaining({ seq: 7, path: "main.go", start_line: 2 }));
  });
});
