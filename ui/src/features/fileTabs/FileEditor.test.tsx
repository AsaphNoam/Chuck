import React from "react";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { FileContent } from "../../api/types";
import { editedSource, FileEditor } from "./FileEditor";
import { draftKey, getDraft, MAX_DRAFT_TEXT_BYTES } from "./drafts";

afterEach(cleanup);
function snapshot(content: string, revision: string): FileContent {
  return { agent_id: "a", path: "notes.md", content, revision, language: "markdown", size: content.length, line_count: 1, mod_time: "", truncated: false };
}
describe("file source editor", () => {
  it("preserves untouched mixed source newline bytes while editing normalized textarea text", () => {
    expect(editedSource("one\r\ntwo\nthree\r\n", "one\nchanged\nthree\n")).toBe("one\r\nchanged\nthree\r\n");
  });
  it("retains typing across an in-flight save and advances the surviving draft base", async () => {
    const identity = { owner: "conversation:save-generation", path: "notes.md" };
    let resolve!: (value: FileContent) => void;
    const save = vi.fn(() => new Promise<FileContent>((done) => { resolve = done; }));
    const onSaved = vi.fn();
    render(<FileEditor identity={identity} initial={snapshot("base", "r1")} load={async () => snapshot("base", "r1")} save={save} onSaved={onSaved} onCancel={vi.fn()} />);
    const field = await screen.findByLabelText("File source draft");
    await waitFor(() => expect(field).not.toBeDisabled());
    fireEvent.change(field, { target: { value: "submitted\r\n" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(save).toHaveBeenCalledWith("r1", "submitted\n"));
    fireEvent.change(field, { target: { value: "newer" } });
    resolve(snapshot("submitted\n", "r2"));
    await waitFor(() => expect(getDraft(identity)?.baseRevision).toBe("r2"));
    expect(field).toHaveValue("newer");
    expect(onSaved).not.toHaveBeenCalled();
  });
  it("requires explicit reviewed-base adoption and refuses oversized edits without replacing text", async () => {
    const identity = { owner: "conversation:conflict", path: "notes.md" };
    const save = vi.fn(async () => snapshot("draft", "r3"));
    render(<FileEditor identity={identity} initial={snapshot("base", "r1")} load={async () => snapshot("external", "r2")} save={save} onSaved={vi.fn()} onCancel={vi.fn()} />);
    const field = screen.getByLabelText("File source draft");
    await waitFor(() => expect(field).not.toBeDisabled());
    fireEvent.change(field, { target: { value: "draft" } });
    fireEvent.change(field, { target: { value: "x".repeat(MAX_DRAFT_TEXT_BYTES + 1) } });
    expect(field).toHaveValue("draft");
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await screen.findByRole("button", { name: "Use this version as save base" });
    expect(save).not.toHaveBeenCalled();
    expect(getDraft(identity)?.baseRevision).toBe("r1");
    fireEvent.click(screen.getByRole("button", { name: "Use this version as save base" }));
    expect(getDraft(identity)?.baseRevision).toBe("r2");
    expect(draftKey(identity)).toBe(getDraft(identity)?.key);
  });
});
