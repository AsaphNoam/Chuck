import React, { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { MemoryRouter, useLocation, useNavigate, type NavigateFunction } from "react-router-dom";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { FileTabsReader } from "./FileTabs";
import { TranscriptView } from "../../components/chat/TranscriptView";
import { ownerKey, STORAGE_KEY } from "./model";
import { loadFileTabs, useFileTabsStore } from "./store";
import { CONVERSATION_FILE_KEYS, ROOM_FILE_KEYS, useFileTabRoute } from "./useFileTabRoute";

// FS-03.A80–A82 (desktop conversation portions) against the real route hook,
// reader and file endpoint. Reads are counted at the HTTP boundary.

let reads: string[] = [];
let hold: Record<string, Promise<void>> = {};
let normalizedPaths: Record<string, string> = {};
const server = setupServer(
  http.get("/api/sessions/:id/file", async ({ request }) => {
    const path = new URL(request.url).searchParams.get("path") ?? "";
    reads.push(path);
    if (hold[path]) await hold[path];
    if (path === "missing.md") return HttpResponse.json({ error: { code: "not_found", message: "That file no longer exists." } }, { status: 404 });
    const name = path.replace(/\.md$/, "");
    return HttpResponse.json({
      agent_id: "a1", path: normalizedPaths[path] ?? path.replace(/^\.\//, ""), size: 20, mod_time: "2026-10-25T09:00:00Z", line_count: 40,
      content: `# Title ${name.replace(/^\.\//, "")}\n\n${Array.from({ length: 30 }, (_, i) => `Paragraph ${i}.`).join("\n\n")}\n`,
      truncated: false, language: "markdown",
    });
  }),
);
beforeAll(() => {
  server.listen({ onUnhandledRequest: "bypass" });
  Element.prototype.scrollIntoView ??= () => undefined;
});
afterEach(() => {
  cleanup();
  reads = [];
  hold = {};
  normalizedPaths = {};
  server.resetHandlers();
});
afterAll(() => server.close());

const events = [{ kind: "assistant_text", seq: 1, text: "[A](a.md) [B](b.md) [C](c.md) [Gone](missing.md) [A12](./a.md:12)" }];
let navigate: NavigateFunction;
let address = "";

function Probe() {
  navigate = useNavigate();
  const location = useLocation();
  address = location.search;
  return null;
}

function Conversation({ agentId }: { agentId: string }) {
  const tabs = useFileTabRoute(ownerKey("conversation", agentId), CONVERSATION_FILE_KEYS);
  return <><output data-testid="jumps">{JSON.stringify(tabs.jumps)}</output><TranscriptView agentId={agentId} events={events} annotationsEnabled fileTabs={tabs} onOpenFile={(link) => { if (link) tabs.open(link); }} /></>;
}

function RoomConversation() {
  const tabs = useFileTabRoute(ownerKey("room", "r1"), ROOM_FILE_KEYS);
  return <><button type="button" onClick={() => tabs.open({ path: "./a.md", source: "s1" })}>Open room file</button><output data-testid="jumps">{JSON.stringify(tabs.jumps)}</output><TranscriptView agentId="a1" events={[]} annotationsEnabled fileTabs={tabs} /></>;
}

function App({ initial = "/agent/a1", agentId = "a1" }: { initial?: string; agentId?: string }) {
  const [client] = useState(() => new QueryClient());
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[initial]}>
        <Probe />
        <Conversation agentId={agentId} />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

function RoomApp() {
  const [client] = useState(() => new QueryClient());
  return <QueryClientProvider client={client}><MemoryRouter initialEntries={["/room/r1"]}><Probe /><RoomConversation /></MemoryRouter></QueryClientProvider>;
}

const tabNames = () => screen.queryAllByRole("tab").map((tab) => tab.textContent);
const selected = () => screen.getAllByRole("tab").find((tab) => tab.getAttribute("aria-selected") === "true")?.textContent;
const link = (name: string) => within(document.querySelector(".transcript-list, .transcript") ?? document.body).getAllByRole("button", { name }).find((el) => !el.closest(".file-viewer"))!;
const heading = (name: string) => screen.findByRole("heading", { name: `Title ${name}` });

describe("file tabs on a conversation (FS-03.A80–A82)", () => {
  it("opens the right-clicked inactive HTML tab from its own retained source", async () => {
    const tabs = [{ id: "one", path: "a.md", source: "s1" }, { id: "two", path: "page.html", source: "s2" }];
    const openBrowserFor = vi.fn().mockResolvedValue({ path: "page.html" });
    const onSelect = vi.fn();
    render(<FileTabsReader tabs={tabs} active={tabs[0]} jumps={{}} onSelect={onSelect} onClose={() => {}} onMove={() => {}} contextFor={(tab) => tab.source!} openBrowserFor={openBrowserFor} renderView={(_reader, strip) => strip} />);
    fireEvent.contextMenu(screen.getByRole("tab", { name: "page.html" }));
    fireEvent.click(await screen.findByRole("button", { name: "Open in browser" }));
    expect(openBrowserFor).toHaveBeenCalledWith(tabs[1], "page.html");
    expect(onSelect).not.toHaveBeenCalled();
    expect(reads).not.toContain("page.html");
  });

  it("keeps the trail, returns from memory with its mode, and closes by the neighbour rule", async () => {
    render(<App />);
    fireEvent.click(link("A"));
    await heading("a");
    fireEvent.click(screen.getByRole("button", { name: "Source" }));
    fireEvent.click(link("B"));
    await heading("b");
    fireEvent.click(link("C"));
    await heading("c");
    expect(tabNames()).toEqual(["a.md", "b.md", "c.md"]);
    expect(reads).toEqual(["a.md", "b.md", "c.md"]);

    fireEvent.click(screen.getByRole("tab", { name: "a.md" }));
    expect(screen.getByRole("button", { name: "Source" })).toHaveAttribute("aria-pressed", "true");
    expect(reads).toHaveLength(3);
    expect(address).toBe("?file=a.md");

    // A cited line on an open file selects it rather than adding a duplicate.
    fireEvent.click(link("A12"));
    expect(tabNames()).toEqual(["a.md", "b.md", "c.md"]);
    expect(address).toBe("?file=a.md&fileLine=12");

    // Middle active closes to its right; a background close keeps the active file.
    fireEvent.click(screen.getByRole("tab", { name: "b.md" }));
    fireEvent.click(screen.getByRole("button", { name: "Close b.md" }));
    expect(selected()).toBe("c.md");
    expect(screen.getByRole("tab", { name: "c.md" })).toHaveFocus();
    fireEvent.click(screen.getByRole("button", { name: "Close a.md" }));
    expect(selected()).toBe("c.md");
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(document.querySelector(".file-viewer")).toBeNull();
    expect(address).toBe("");
    expect(localStorage.getItem(STORAGE_KEY)).toBeNull();
    expect(reads).toHaveLength(3);
  });

  it("moves tabs with keyboard-reachable controls without changing the selection or address", async () => {
    render(<App />);
    fireEvent.click(link("A"));
    fireEvent.click(link("B"));
    await heading("b");
    const before = address;
    fireEvent.click(screen.getByRole("button", { name: "Move b.md left" }));
    expect(tabNames()).toEqual(["b.md", "a.md"]);
    expect(selected()).toBe("b.md");
    expect(address).toBe(before);
    expect(screen.getByRole("button", { name: "Move b.md left" })).toBeDisabled();
    // Arrow keys move focus along the tabs; Enter selects.
    screen.getByRole("tab", { name: "b.md" }).focus();
    fireEvent.keyDown(screen.getByRole("tab", { name: "b.md" }), { key: "ArrowRight" });
    expect(screen.getByRole("tab", { name: "a.md" })).toHaveFocus();
    expect(selected()).toBe("b.md");
  });

  it("adds one history entry; Back hides the reader and Forward reopens it", async () => {
    render(<App />);
    fireEvent.click(link("A"));
    fireEvent.click(link("B"));
    await heading("b");
    fireEvent.click(screen.getByRole("tab", { name: "a.md" }));
    act(() => navigate(-1));
    await waitFor(() => expect(document.querySelector(".file-viewer")).toBeNull());
    expect(address).toBe("");
    expect(useFileTabsStore.getState().owners["conversation:a1"].tabs).toHaveLength(2);
    act(() => navigate(1));
    await heading("a");
    expect(tabNames()).toEqual(["a.md", "b.md"]);
    expect(address).toBe("?file=a.md");
  });

  it("does not project a delayed normalized path after Back hides the reader", async () => {
    let release!: () => void;
    hold["a.md"] = new Promise<void>((resolve) => { release = resolve; });
    normalizedPaths["a.md"] = "normalized/a.md";
    render(<App />);
    fireEvent.click(link("A"));
    await waitFor(() => expect(reads).toContain("a.md"));
    act(() => navigate(-1));
    await waitFor(() => expect(address).toBe(""));
    await act(async () => { release(); });
    await waitFor(() => expect(useFileTabsStore.getState().owners["conversation:a1"].tabs[0].path).toBe("normalized/a.md"));
    expect(address).toBe("");
    expect(document.querySelector(".file-viewer")).toBeNull();
  });

  it("keeps a delayed room normalization from rewriting the file-free Back address", async () => {
    let release!: () => void;
    hold["./a.md"] = new Promise<void>((resolve) => { release = resolve; });
    normalizedPaths["./a.md"] = "normalized/a.md";
    render(<RoomApp />);
    fireEvent.click(screen.getByRole("button", { name: "Open room file" }));
    await waitFor(() => expect(reads).toContain("./a.md"));
    act(() => navigate(-1));
    await waitFor(() => expect(address).toBe(""));
    await act(async () => { release(); });
    await waitFor(() => expect(useFileTabsStore.getState().owners["room:r1"].tabs[0].path).toBe("normalized/a.md"));
    expect(address).toBe("");
    expect(document.querySelector(".file-viewer")).toBeNull();
  });

  it("reapplies a cited line on remount even when the saved descriptor has that line", async () => {
    const first = render(<App initial="/agent/a1?file=./a.md&fileLine=12" />);
    await heading("a");
    first.unmount();
    const owner = useFileTabsStore.getState().owners["conversation:a1"];
    useFileTabsStore.setState({ owners: { ...useFileTabsStore.getState().owners, "conversation:a1": { ...owner, tabs: owner.tabs.map((tab) => ({ ...tab, anchors: { Rendered: 1200 } })) } } });
    reads = [];
    render(<App initial="/agent/a1?file=./a.md&fileLine=12" />);
    await heading("a");
    expect(reads).toEqual(["a.md"]);
    const tabId = useFileTabsStore.getState().owners["conversation:a1"].active!;
    expect(JSON.parse(screen.getByTestId("jumps").textContent ?? "{}")[tabId]).toBeGreaterThan(0);
  });

  it("reapplies a cited line when Forward returns to the same citation", async () => {
    render(<App />);
    fireEvent.click(link("A12"));
    await heading("a");
    act(() => navigate(-1));
    await waitFor(() => expect(address).toBe(""));
    act(() => navigate(1));
    await waitFor(() => expect(address).toBe("?file=a.md&fileLine=12"));
    const tabId = useFileTabsStore.getState().owners["conversation:a1"].active!;
    expect(JSON.parse(screen.getByTestId("jumps").textContent ?? "{}")[tabId]).toBeGreaterThan(1);
  });

  it("restores the saved set after reload, reading only the selected tab", async () => {
    const first = render(<App />);
    fireEvent.click(link("A"));
    fireEvent.click(link("B"));
    await heading("b");
    fireEvent.click(screen.getByRole("button", { name: "Source" }));
    first.unmount();
    reads = [];
    loadFileTabs();
    render(<App />);
    expect(await screen.findByText(/Paragraph 0/)).toBeInTheDocument();
    expect(tabNames()).toEqual(["a.md", "b.md"]);
    expect(selected()).toBe("b.md");
    expect(screen.getByRole("button", { name: "Source" })).toHaveAttribute("aria-pressed", "true");
    expect(address).toBe("?file=b.md");
    expect(reads).toEqual(["b.md"]);
    const saved = localStorage.getItem(STORAGE_KEY)!;
    expect(saved).not.toMatch(/Paragraph|readAt|mod_time/);
    // Another conversation keeps its own set.
    cleanup();
    render(<App initial="/agent/a2" agentId="a2" />);
    expect(document.querySelector(".file-viewer")).toBeNull();
  });

  it("lets an explicit file URL win over the saved active tab", async () => {
    const first = render(<App />);
    fireEvent.click(link("A"));
    fireEvent.click(link("B"));
    await heading("b");
    first.unmount();
    reads = [];
    render(<App initial="/agent/a1?file=c.md&fileLine=4" />);
    await heading("c");
    expect(tabNames()).toEqual(["a.md", "b.md", "c.md"]);
    expect(address).toBe("?file=c.md&fileLine=4");
    expect(reads).toEqual(["c.md"]);
  });

  it("keeps a failed read in its own closable tab", async () => {
    render(<App />);
    fireEvent.click(link("Gone"));
    expect(await screen.findByRole("alert")).toHaveTextContent(/no longer exists/);
    fireEvent.click(link("A"));
    await heading("a");
    fireEvent.click(screen.getByRole("tab", { name: "missing.md" }));
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(useFileTabsStore.getState().owners["conversation:a1"].tabs).toHaveLength(2);
  });

  it("drops a closed tab's late reply without touching the remaining tabs", async () => {
    let release!: () => void;
    hold["b.md"] = new Promise<void>((resolve) => { release = resolve; });
    render(<App />);
    fireEvent.click(link("A"));
    await heading("a");
    fireEvent.click(link("B"));
    await waitFor(() => expect(reads).toContain("b.md"));
    fireEvent.click(screen.getByRole("button", { name: "Close b.md" }));
    await act(async () => { release(); await new Promise((resolve) => setTimeout(resolve, 10)); });
    expect(tabNames()).toEqual(["a.md"]);
    expect(address).toBe("?file=a.md");
    expect(screen.getByRole("heading", { name: "Title a" })).toBeInTheDocument();
  });

  it("refuses a 51st tab without eviction or a read, and still selects open ones", async () => {
    const tabs = Array.from({ length: 50 }, (_, i) => ({ id: `t${i}`, path: `f${i}.md`, rendered: true, anchors: {} }));
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ v: 1, owners: { "conversation:a1": { tabs, active: "t0" } } }));
    loadFileTabs();
    render(<App />);
    await heading("f0");
    reads = [];
    fireEvent.click(link("A"));
    expect(await screen.findByText(/already has 50 open files/)).toBeInTheDocument();
    expect(screen.getAllByRole("tab")).toHaveLength(50);
    expect(selected()).toBe("f0.md");
    expect(reads).toEqual([]);
    expect(address).toBe("?file=f0.md");
  });
});
