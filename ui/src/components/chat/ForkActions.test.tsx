import React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import * as client from "../../api/client";
import type { TranscriptEvent } from "../../api/types";
import { useAgentStore } from "../../store/agentStore";
import { clearChatDraftCacheForTests, getChatDraft } from "./drafts";
import { forkActionFor } from "./forkActions";
import { TranscriptView } from "./TranscriptView";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  useAgentStore.setState({ agents: {} });
  clearChatDraftCacheForTests();
});

const ok = { available: true };
// Three completed turns; the server marks each root turn end (TS-03.R86).
const events: TranscriptEvent[] = [
  { kind: "user_text", seq: 1, text: "first" },
  { kind: "assistant_text", seq: 2, text: "reply one" },
  { kind: "turn_end", seq: 3, stop_reason: "end_turn", fork: { available: false, reason: "No fork point." } },
  { kind: "user_text", seq: 4, text: "second" },
  { kind: "tool_call", seq: 5, name: "Read" },
  { kind: "assistant_text", seq: 6, text: "reply two" },
  { kind: "turn_end", seq: 7, stop_reason: "end_turn", fork: ok },
  { kind: "user_text", seq: 8, text: "third" },
  { kind: "assistant_text", seq: 9, text: "reply three" },
  { kind: "turn_end", seq: 10, stop_reason: "end_turn", fork: ok },
];

describe("forkActionFor (FS-01.R58)", () => {
  it("resolves Edit to the previous turn and Fork from here to the reply's own turn", () => {
    expect(forkActionFor(events, events[7], ok)).toMatchObject({ label: "Edit", available: true, throughSeq: 7, composeFromSeq: 8 });
    expect(forkActionFor(events, events[5], ok)).toMatchObject({ label: "Fork from here", available: true, throughSeq: 7 });
  });

  it("carries every unavailable reason and offers nothing on other events", () => {
    expect(forkActionFor(events, events[0], ok)).toMatchObject({ available: false, reason: "The first message has no earlier turn to fork from." });
    expect(forkActionFor(events, events[3], ok)).toMatchObject({ label: "Edit", available: false, reason: "No fork point." });
    expect(forkActionFor(events, events[7], { available: false, reason: "Busy." })).toMatchObject({ available: false, reason: "Busy." });
    expect(forkActionFor(events, events[4], ok)).toBeNull();
    expect(forkActionFor(events, { ...events[5], activity_id: "child" }, ok)).toBeNull();
  });
});

function renderChat(forkActionsEnabled = true, transcript = events) {
  useAgentStore.setState({ agents: { a1: { agent_id: "a1", clone: ok } as never } });
  const queries = new QueryClient({ defaultOptions: { queries: { retry: 0 } } });
  return render(
    <QueryClientProvider client={queries}>
      <MemoryRouter initialEntries={["/agent/a1"]}>
        <Routes>
          <Route path="/agent/a1" element={<TranscriptView agentId="a1" events={transcript} forkActionsEnabled={forkActionsEnabled} />} />
          <Route path="/agent/:id" element={<p>opened new agent</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function rightClick(container: HTMLElement, seq: number) {
  fireEvent.contextMenu(container.querySelector(`[data-seq="${seq}"]`) as HTMLElement, { clientX: 10, clientY: 10 });
}

describe("Edit and Fork from here in the desktop transcript (TS-08.R162)", () => {
  it("Edit confirms with the file caveat, clones once and seeds the unsent text", async () => {
    const clone = vi.spyOn(client, "cloneAgentAt").mockResolvedValue({ agent: { agent_id: "a2" }, forked_from_seq: 7, compose_text: "third" });
    const { container } = renderChat();
    rightClick(container, 8);
    fireEvent.click(screen.getByRole("button", { name: "Edit" }));
    expect(screen.getByText(/Files changed since this point are not restored/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Create agent and edit" }));
    await waitFor(() => expect(screen.getByText("opened new agent")).toBeInTheDocument());
    expect(clone).toHaveBeenCalledTimes(1);
    expect(clone).toHaveBeenCalledWith("a1", { through_seq: 7, compose_from_seq: 8 });
    expect(getChatDraft("a2")).toBe("third");
  });

  it("cancelling Fork from here sends nothing", () => {
    const clone = vi.spyOn(client, "cloneAgentAt");
    const { container } = renderChat();
    rightClick(container, 9);
    fireEvent.click(screen.getByRole("button", { name: "Fork from here" }));
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(clone).not.toHaveBeenCalled();
  });

  it("shows an unavailable action disabled with the server's reason", () => {
    const { container } = renderChat();
    rightClick(container, 4);
    const edit = screen.getByRole("button", { name: "Edit" });
    expect(edit).toBeDisabled();
    expect(edit).toHaveAttribute("aria-description", "No fork point.");
  });

  it("disables Edit on a message sent while the agent was working, before anything is sent", () => {
    const clone = vi.spyOn(client, "cloneAgentAt");
    // "also" was injected into the running third turn as an ordinary root message.
    const injected: TranscriptEvent[] = [
      ...events.slice(0, 8),
      { kind: "user_text", seq: 9, text: "also" },
      { kind: "assistant_text", seq: 10, text: "reply three" },
      { kind: "turn_end", seq: 11, stop_reason: "end_turn", fork: ok },
    ];
    const { container } = renderChat(true, injected);
    rightClick(container, 9);
    const edit = screen.getByRole("button", { name: "Edit" });
    expect(edit).toBeDisabled();
    expect(edit).toHaveAttribute("aria-description", "This message was sent while the agent was working; edit the message that began that turn.");
    fireEvent.click(edit);
    expect(clone).not.toHaveBeenCalled();
    expect(forkActionFor(injected, injected[7], ok)).toMatchObject({ available: true, composeFromSeq: 8 });
  });

  it("offers neither action where the surface does not enable them", () => {
    const { container } = renderChat(false);
    rightClick(container, 9);
    expect(screen.queryByRole("button", { name: "Fork from here" })).toBeNull();
  });
});

it("keeps a wire turn end's fork affordance through normalization", async () => {
  const { normalizeEvent } = await import("../../store/transcriptStore");
  const event = normalizeEvent({ type: "turn_end", seq: 3, data: { stop_reason: "end_turn" }, fork: { available: false, reason: "No fork point." } });
  expect(event).toMatchObject({ kind: "turn_end", stop_reason: "end_turn", fork: { available: false, reason: "No fork point." } });
});
