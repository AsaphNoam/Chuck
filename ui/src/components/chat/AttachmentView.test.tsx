// FS-03.A63: owned image thumbnails, modal inspection and corrupt-image recovery.
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MessageAttachments } from "./AttachmentView";

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("owned attachment previews", () => {
  it("loads an owned image into an inline thumbnail and reuses its object URL in the dialog", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(new Blob(["png"], { type: "image/png" }), { status: 200, headers: { "Content-Type": "image/png" } }));
    vi.stubGlobal("fetch", fetchMock);
    vi.stubGlobal("URL", { ...URL, createObjectURL: vi.fn(() => "blob:owned-preview"), revokeObjectURL: vi.fn() });
    render(<MessageAttachments agentId="agent/one" attachments={[{ id: "att-1", name: "photo.png", mime_type: "image/png", size: 3, sha256: "hash", kind: "image" }]} />);

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    const preview = await screen.findByRole("img", { name: "photo.png" });
    expect(preview).toHaveAttribute("src", "blob:owned-preview");
    const opener = screen.getByRole("button", { name: "Open image photo.png" });
    fireEvent.click(opener);
    expect(await screen.findByRole("dialog")).toBeInTheDocument();
    expect(within(screen.getByRole("dialog")).getByRole("img", { name: "photo.png" })).toHaveAttribute("src", "blob:owned-preview");
    expect(fetchMock).toHaveBeenCalledWith("/api/sessions/agent%2Fone/attachments/att-1", expect.objectContaining({ signal: expect.any(AbortSignal) }));
    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" });
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    await waitFor(() => expect(opener).toHaveFocus());
  });

  it("keeps a failed owned image labeled with the server error and retry action", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(null, { status: 404, statusText: "Not Found" })));
    render(<MessageAttachments agentId="agent-1" attachments={[{ id: "att-1", name: "missing.png", mime_type: "image/png", size: 3, sha256: "hash", kind: "image" }]} />);
    expect(await screen.findByText(/Image unavailable: 404 Not Found/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry image" })).toBeInTheDocument();
  });
});
import React from "react";
