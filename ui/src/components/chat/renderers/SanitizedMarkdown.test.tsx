import React from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";

// Counts Markdown parses: remark-gfm is attached once per parse.
const parses = vi.hoisted(() => ({ count: 0 }));
vi.mock("remark-gfm", async (load) => {
  const actual = (await load()) as { default: (...args: unknown[]) => unknown };
  return {
    default: function counted(this: unknown, ...args: unknown[]) {
      parses.count += 1;
      return actual.default.apply(this, args);
    },
  };
});

import { SanitizedMarkdown } from "./SanitizedMarkdown";

afterEach(() => {
  cleanup();
  parses.count = 0;
});

// TAB-BLANK-2 (FS-03.R12, INV §16): a streamed delta re-renders every settled
// message in the transcript; their unchanged Markdown is not parsed again.
describe("SanitizedMarkdown reuse", () => {
  it("parses again only when its own text changes", () => {
    const onOpenFile = () => undefined;
    const { rerender } = render(<SanitizedMarkdown text="**settled** reply" onOpenFile={onOpenFile} />);
    const first = parses.count;
    expect(first).toBeGreaterThan(0);

    rerender(<SanitizedMarkdown text="**settled** reply" onOpenFile={() => undefined} />);
    expect(parses.count).toBe(first);
    expect(screen.getByText("settled").tagName).toBe("STRONG");

    rerender(<SanitizedMarkdown text="**settled** reply, edited" onOpenFile={onOpenFile} />);
    expect(parses.count).toBeGreaterThan(first);
    expect(screen.getByText(/reply, edited/)).toBeInTheDocument();
  });

  it("rebuilds when the same text is shown for another file or without a viewer", () => {
    const onOpenFile = () => undefined;
    const text = "[notes](./notes.md)";
    const { rerender } = render(<SanitizedMarkdown text={text} sourcePath="a/README.md" onOpenFile={onOpenFile} />);
    expect(screen.getByText("notes").getAttribute("data-file-path")).toBe("a/notes.md");

    rerender(<SanitizedMarkdown text={text} sourcePath="b/README.md" onOpenFile={onOpenFile} />);
    expect(screen.getByText("notes").getAttribute("data-file-path")).toBe("b/notes.md");
    expect(screen.getByText("notes").tagName).toBe("BUTTON");

    rerender(<SanitizedMarkdown text={text} sourcePath="b/README.md" />);
    expect(screen.getByText("notes").tagName).toBe("SPAN");
  });
});
