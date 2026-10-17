import React, { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";
import { useAgentStore } from "../../store/agentStore";
import { useAnnotationStore } from "../../store/annotationStore";
import { AgentStage, type PanelSlots } from "./AgentStage";
import { TranscriptView } from "./TranscriptView";
import type { FileLink } from "./renderers/filePath";
import { MenuPortalContext, PointerContextMenu } from "../ui/PointerContextMenu";

let reads: string[] = [];
let releaseSend: (() => void) | null = null;
let sends = 0;
const server = setupServer(
  http.get("/api/sessions/:id/file", ({ request }) => {
    const path = new URL(request.url).searchParams.get("path") ?? "";
    reads.push(path);
    return HttpResponse.json({
      agent_id: "a1", path, size: 20, mod_time: "2026-10-17T09:00:00Z", line_count: 3,
      content: path.endsWith(".md") ? "# Notes 原文\n\nSee [other](other.md).\n" : "package main\n\nfunc main() {}\n",
      truncated: false, language: path.endsWith(".md") ? "markdown" : "go",
    });
  }),
  http.post("/api/sessions/:id/annotations", async () => {
    sends += 1;
    await new Promise<void>((resolve) => { releaseSend = resolve; });
    return HttpResponse.json({ ok: true });
  }),
);
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => {
  cleanup();
  reads = [];
  sends = 0;
  releaseSend = null;
  useAnnotationStore.setState({ bySource: {}, overallBySource: {}, editedAt: {}, collapsedBySource: {} });
  useAgentStore.setState({ agents: {} });
  server.resetHandlers();
});
afterAll(() => server.close());

const events = [{ kind: "assistant_text", seq: 7, text: "Reply" }];

// The full agent page's composition: the stage around a card holding the
// transcript, with the route's open file owned above both (FS-03.R54).
function Page({ initial }: { initial: FileLink | null }) {
  const [file, setFile] = useState<FileLink | null>(initial);
  const [slots, setSlots] = useState<PanelSlots | undefined>(undefined);
  return (
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: 0 } } })}>
      <AgentStage onSlots={setSlots}>
        <div className="chat-card">
          <TranscriptView agentId="a1" events={events} sourceActive annotationsEnabled openFile={file} onOpenFile={setFile} panelSlots={slots} />
        </div>
      </AgentStage>
    </QueryClientProvider>
  );
}

function seedDraft() {
  useAnnotationStore.setState({
    bySource: { a1: [{ seq: 7, excerpt: "Reply", instruction: "keep this" }] },
    overallBySource: {}, editedAt: { a1: Date.now() }, collapsedBySource: {},
  });
}

// FS-03.A73–A75, TS-08.R127/R129/R130.
describe("reader workspace", () => {
  it("moves the same read and tray into Focus file and back without reading again", async () => {
    seedDraft();
    render(<Page initial={{ path: "docs/notes.md" }} />);
    await screen.findByRole("heading", { name: /Notes/ });
    expect(screen.getByRole("separator", { name: "Resize file and conversation" })).toHaveAttribute("aria-valuenow");
    fireEvent.click(screen.getByRole("button", { name: "Source", exact: true } as never));

    fireEvent.click(screen.getByRole("button", { name: "Focus file" }));
    const dialog = await screen.findByRole("dialog", { name: "Focus file" });
    // Same snapshot, same display mode, same single tray with its instruction.
    expect(within(dialog).getByRole("button", { name: "Source" })).toHaveAttribute("aria-pressed", "true");
    expect(within(dialog).getByLabelText("Instruction")).toHaveValue("keep this");
    expect(document.querySelectorAll(".annotation-tray")).toHaveLength(1);
    expect(document.querySelectorAll(".file-viewer")).toHaveLength(1);
    expect(screen.queryByRole("separator")).toBeNull();
    expect(reads).toEqual(["docs/notes.md"]);

    fireEvent.click(within(dialog).getByRole("button", { name: "Return to chat" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
    expect(screen.getByRole("button", { name: "Source" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByLabelText("Instruction")).toHaveValue("keep this");
    expect(reads).toEqual(["docs/notes.md"]);
  });

  it("replaces the file inside focus, reloads only on request, and closes it there", async () => {
    render(<Page initial={{ path: "docs/notes.md" }} />);
    await screen.findByRole("heading", { name: /Notes/ });
    fireEvent.click(screen.getByRole("button", { name: "Focus file" }));
    const dialog = await screen.findByRole("dialog", { name: "Focus file" });
    fireEvent.click(within(dialog).getByRole("button", { name: "other" }));
    await waitFor(() => expect(reads).toEqual(["docs/notes.md", "docs/other.md"]));
    expect(screen.getByRole("dialog", { name: "Focus file" })).toBeInTheDocument();
    fireEvent.click(within(dialog).getByRole("button", { name: "Reload" }));
    await waitFor(() => expect(reads).toHaveLength(3));

    fireEvent.click(within(dialog).getByRole("button", { name: "Close file" }));
    await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
    expect(document.querySelector(".file-viewer")).toBeNull();
    expect(reads).toHaveLength(3);
  });

  it("keeps a send in flight across focus and return", async () => {
    seedDraft();
    useAgentStore.setState({ agents: {} });
    render(<Page initial={{ path: "main.go" }} />);
    await screen.findByText(/package/);
    fireEvent.click(screen.getByRole("button", { name: "Send annotations" }));
    await waitFor(() => expect(sends).toBe(1));
    expect(screen.getByRole("button", { name: "Sending…" })).toBeDisabled();

    fireEvent.click(screen.getByRole("button", { name: "Focus file" }));
    const dialog = await screen.findByRole("dialog", { name: "Focus file" });
    expect(within(dialog).getByRole("button", { name: "Sending…" })).toBeDisabled();
    await act(async () => releaseSend?.());
    await waitFor(() => expect(useAnnotationStore.getState().bySource.a1 ?? []).toHaveLength(0));
    // The tray leaves with its last draft; the file stays open.
    expect(within(dialog).queryByRole("complementary", { name: "Pending annotations" })).toBeNull();
    expect(within(dialog).getByText(/package/)).toBeInTheDocument();
    expect(sends).toBe(1);
  });

  it("uses English reader controls and keeps the file's own text", async () => {
    render(<Page initial={{ path: "docs/notes.md" }} />);
    const heading = await screen.findByRole("heading", { name: /Notes/ });
    expect(heading).toHaveTextContent("Notes 原文");
    fireEvent.click(screen.getByRole("button", { name: "Focus file" }));
    const dialog = await screen.findByRole("dialog", { name: "Focus file" });
    const labels = within(dialog).getAllByRole("button").map((button) => button.textContent ?? "");
    expect(labels).toEqual(expect.arrayContaining(["Rendered", "Source", "Reload", "Return to chat", "Close file"]));
    for (const label of labels) expect(label).not.toMatch(/[㐀-鿿]/);
  });
});

describe("PointerContextMenu destination", () => {
  it("mounts inside a provided modal element and on the body otherwise", () => {
    const host = document.createElement("div");
    document.body.append(host);
    const menu = { x: 1, y: 2, actions: [{ label: "Copy", select: () => {} }] };
    const { unmount } = render(<MenuPortalContext.Provider value={host}><PointerContextMenu menu={menu} onClose={() => {}} /></MenuPortalContext.Provider>);
    expect(host.querySelector(".context-menu")).not.toBeNull();
    unmount();
    render(<PointerContextMenu menu={menu} onClose={() => {}} />);
    expect(document.body.querySelector(":scope > .context-menu")).not.toBeNull();
    host.remove();
  });
});
