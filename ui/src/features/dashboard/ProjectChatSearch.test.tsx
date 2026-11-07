import { act, cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ProjectChatSearch } from "./ProjectChatSearch";
import { parseSearchReference, useSearchReference } from "./searchReference";

function Location() { const location = useLocation(); return <output>{location.pathname}{location.search}</output>; }
const row = { agent_id: "chat", name: "Authentication discussion", active: true, archived: false, snippet: "authentication", location_status: "reference", search_ref: { document_id: "turn:7", content_sha256: "a".repeat(64), start_byte: 12, end_byte: 26 } };
let originalScrollIntoView: PropertyDescriptor | undefined;
afterEach(() => {
  cleanup(); vi.restoreAllMocks(); vi.useRealTimers();
  if (originalScrollIntoView) Object.defineProperty(HTMLElement.prototype, "scrollIntoView", originalScrollIntoView);
  else delete (HTMLElement.prototype as HTMLElement & { scrollIntoView?: () => void }).scrollIntoView;
  originalScrollIntoView = undefined;
});

describe("project chat search", () => {
  // FS-05.R49: a response that ignores cancellation must not overwrite newer input.
  it("debounces requests, rejects stale responses, and opens the selected source reference with Enter", async () => {
    vi.useFakeTimers();
    let finishOld!: (response: Response) => void;
    const fetcher = vi.spyOn(globalThis, "fetch").mockImplementationOnce(() => new Promise((resolve) => { finishOld = resolve; })).mockResolvedValueOnce(new Response(JSON.stringify({ results: [row], total: 1, search_mode: "full_text" })));
    render(<MemoryRouter><ProjectChatSearch project="project-a" /><Location /></MemoryRouter>);
    const field = screen.getByRole("combobox", { name: "Search chats" });
    fireEvent.change(field, { target: { value: "old" } });
    await act(async () => { vi.advanceTimersByTime(249); });
    expect(fetcher).not.toHaveBeenCalled();
    await act(async () => { vi.advanceTimersByTime(1); });
    fireEvent.change(field, { target: { value: "auth" } });
    await act(async () => { vi.advanceTimersByTime(250); });
    expect(screen.getByText("Authentication discussion")).toBeInTheDocument();
    await act(async () => { finishOld(new Response(JSON.stringify({ results: [{ ...row, name: "Stale" }], total: 1, search_mode: "full_text" }))); });
    expect(screen.queryByText("Stale")).toBeNull();
    const url = String(fetcher.mock.calls[1][0]);
    expect(url).toContain("/api/archive/projects/project-a?");
    expect(url).toContain("view=project_search");
    fireEvent.keyDown(field, { key: "Enter" });
    expect(screen.getByRole("status").textContent).toContain("/agent/chat?search_doc=turn%3A7&search_rev=");
  });

  it("retains a failed query for retry and Escape dismisses without clearing it", async () => {
    vi.useFakeTimers();
    vi.spyOn(globalThis, "fetch").mockRejectedValueOnce(new Error("Read failed")).mockResolvedValueOnce(new Response(JSON.stringify({ results: [], total: 0, search_mode: "metadata" })));
    render(<MemoryRouter><ProjectChatSearch project="a" /></MemoryRouter>);
    const field = screen.getByRole("combobox");
    fireEvent.change(field, { target: { value: "auth" } });
    await act(async () => { vi.advanceTimersByTime(250); });
    expect(screen.getByRole("alert")).toHaveTextContent("Read failed");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    await act(async () => { vi.advanceTimersByTime(250); });
    expect(screen.getByText("No matching chats")).toBeInTheDocument();
    fireEvent.keyDown(field, { key: "Escape" });
    expect(screen.queryByText("No matching chats")).toBeNull();
    expect(field).toHaveValue("auth");
  });

  it("refreshes a shifted page boundary instead of duplicating or omitting chats", async () => {
    vi.useFakeTimers();
    const first = Array.from({ length: 20 }, (_, index) => ({ ...row, agent_id: `chat-${index}`, name: `Chat ${index}` }));
    const promoted = { ...row, agent_id: "promoted", name: "New running match" };
    const fetcher = vi.spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(new Response(JSON.stringify({ results: first, total: 30, search_mode: "full_text" })))
      .mockResolvedValueOnce(new Response(JSON.stringify({ results: [first[18], first[19]], total: 31, search_mode: "full_text" })))
      .mockResolvedValueOnce(new Response(JSON.stringify({ results: [first[18]], total: 31, search_mode: "full_text" })))
      .mockResolvedValueOnce(new Response(JSON.stringify({ results: [promoted, ...first.slice(0, 19)], total: 31, search_mode: "full_text" })));
    render(<MemoryRouter><ProjectChatSearch project="a" /></MemoryRouter>);
    fireEvent.change(screen.getByRole("combobox"), { target: { value: "auth" } });
    await act(async () => { vi.advanceTimersByTime(250); });
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: "Load more" })); });
    expect(screen.getByText("New running match")).toBeInTheDocument();
    expect(screen.getAllByRole("option")).toHaveLength(20);
    expect(String(fetcher.mock.calls[1][0])).toContain("offset=20");
    expect(String(fetcher.mock.calls[3][0])).toContain("offset=0");
  });

  it("scrolls the selected option into view while keeping keyboard focus in the field", async () => {
    vi.useFakeTimers();
    const scrollIntoView = vi.fn();
    originalScrollIntoView = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "scrollIntoView");
    Object.defineProperty(HTMLElement.prototype, "scrollIntoView", { configurable: true, value: scrollIntoView });
    const results = Array.from({ length: 20 }, (_, index) => ({ ...row, agent_id: `chat-${index}`, name: `Chat ${index}` }));
    vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify({ results, total: 20, search_mode: "full_text" })));
    render(<MemoryRouter><ProjectChatSearch project="a" /></MemoryRouter>);
    const field = screen.getByRole("combobox", { name: "Search chats" });
    field.focus();
    fireEvent.change(field, { target: { value: "auth" } });
    await act(async () => { vi.advanceTimersByTime(250); });
    for (let index = 0; index < 15; index++) fireEvent.keyDown(field, { key: "ArrowDown" });
    expect(scrollIntoView).toHaveBeenLastCalledWith({ block: "nearest" });
    expect(scrollIntoView.mock.contexts.at(-1)).toBe(screen.getByRole("option", { name: /Chat 15/ }));
    expect(document.activeElement).toBe(field);
    expect(screen.getByRole("option", { name: /Chat 15/ })).toHaveAttribute("aria-selected", "true");
    fireEvent.keyDown(field, { key: "ArrowDown" });
    fireEvent.keyDown(field, { key: "ArrowUp" });
    expect(screen.getByRole("option", { name: /Chat 15/ })).toHaveAttribute("aria-selected", "true");
    expect(document.activeElement).toBe(field);
  });
});

describe("durable search reference", () => {
  it("rejects malformed and unsafe byte coordinates", () => {
    const base = `search_doc=turn%3A7&search_rev=${"a".repeat(64)}&search_start=12&search_end=26`;
    expect(parseSearchReference(new URLSearchParams(base))).toEqual(row.search_ref);
    expect(parseSearchReference(new URLSearchParams(base.replace("search_end=26", "search_end=12")))).toBeNull();
    expect(parseSearchReference(new URLSearchParams(base.replace("search_start=12", "search_start=9007199254740992")))).toBeNull();
  });

  it("keeps an unavailable chat readable and removes only search params for normal opening", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify({ status: "unavailable", reason: "source_changed", ranges: [] })));
    function Reader() { const search = useSearchReference("chat", true, false); return <>{search.notice}<Location /></>; }
    render(<MemoryRouter initialEntries={[`/agent/chat?tab=transcript&search_doc=turn%3A7&search_rev=${"a".repeat(64)}&search_start=12&search_end=26`]}><Reader /></MemoryRouter>);
    fireEvent.click(await screen.findByRole("button", { name: "Open chat normally" }));
    expect(screen.getByRole("status")).toHaveTextContent("/agent/chat?tab=transcript");
  });
});
import React from "react";
