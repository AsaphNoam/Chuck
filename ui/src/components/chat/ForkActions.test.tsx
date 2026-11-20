import React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import * as client from "../../api/client";
import type { TranscriptEvent } from "../../api/types";
import { useAgentStore } from "../../store/agentStore";
import { useUiStore } from "../../store/uiStore";
import { clearChatDraftCacheForTests, getChatDraft } from "./drafts";
import { forkActionFor, forkThroughTurn } from "./forkActions";
import { TranscriptView } from "./TranscriptView";

const initialUi = useUiStore.getState();

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  useUiStore.setState(initialUi, true);
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
    fireEvent.click(within(screen.getByRole("menu")).getByRole("button", { name: "Fork from here" }));
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
    expect(container.querySelector(".message-actions")).toBeNull();
  });
});

function row(container: HTMLElement, seq: number) {
  return within(container.querySelector(`[data-seq="${seq}"] .message-actions`) as HTMLElement);
}

describe("message hover actions (FS-01.R60, R62, TS-08.R171)", () => {
  it("offers Copy and Fork on every root text message and nowhere else", () => {
    const { container } = renderChat();
    for (const seq of [1, 2, 4, 6, 8, 9]) {
      expect(row(container, seq).getByRole("button", { name: "Copy" })).toBeInTheDocument();
      expect(row(container, seq).getByRole("button", { name: "Fork from here" })).toBeInTheDocument();
      expect(row(container, seq).queryByRole("button", { name: "Edit" })).toBeNull();
    }
    for (const seq of [3, 5, 7]) expect(container.querySelector(`[data-seq="${seq}"] .message-actions`)).toBeNull();
  });

  it("forks a person's message through its own completed turn, sharing the menu's descriptor", async () => {
    const clone = vi.spyOn(client, "cloneAgentAt").mockResolvedValue({ agent: { agent_id: "a2" }, forked_from_seq: 7 });
    const { container } = renderChat();
    const opener = row(container, 4).getByRole("button", { name: "Fork from here" });
    opener.focus();
    fireEvent.click(opener);
    fireEvent.click(screen.getByRole("button", { name: "Create agent" }));
    await waitFor(() => expect(screen.getByText("opened new agent")).toBeInTheDocument());
    expect(clone).toHaveBeenCalledWith("a1", { through_seq: 7, compose_from_seq: undefined });
    expect(forkThroughTurn(events, events[3], ok)).toEqual(forkThroughTurn(events, events[5], ok));
  });

  it("returns focus to the opener after dismissing Fork", async () => {
    const { container } = renderChat();
    const opener = row(container, 6).getByRole("button", { name: "Fork from here" });
    opener.focus();
    fireEvent.click(opener);
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(opener).toHaveFocus());
  });

  it("explains an unavailable Fork on pointer and keyboard without opening confirmation", () => {
    const clone = vi.spyOn(client, "cloneAgentAt");
    const running: TranscriptEvent[] = [...events.slice(0, 7), { kind: "user_text", seq: 8, text: "third" }, { kind: "assistant_text", seq: 9, text: "partial" }];
    const { container } = renderChat(true, running);
    for (const [seq, reason] of [[2, "No fork point."], [9, "This turn is still running; Fork is available once it completes."]] as const) {
      const fork = row(container, seq).getByRole("button", { name: "Fork from here" });
      expect(fork).toHaveAttribute("aria-disabled", "true");
      expect(fork).not.toBeDisabled();
      expect(fork).toHaveAccessibleDescription(`Fork from here unavailable: ${reason}`);
      fireEvent.click(fork);
    }
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(clone).not.toHaveBeenCalled();
  });

  it("copies the complete Markdown source and announces it, even where Fork is unavailable", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
    const markdown = "## Plan\n\n- one\n- `two`\n";
    const { container } = renderChat(true, [{ kind: "assistant_text", seq: 1, text: markdown }]);
    fireEvent.click(row(container, 1).getByRole("button", { name: "Copy" }));
    await waitFor(() => expect(row(container, 1).getByRole("button", { name: "Copied" })).toBeInTheDocument());
    expect(writeText).toHaveBeenCalledWith(markdown);
    expect(screen.getByText("Copied to clipboard")).toBeInTheDocument();
  });

  it("reports a denied clipboard instead of showing copied", async () => {
    const pushError = vi.fn();
    useUiStore.setState({ pushError } as never);
    Object.defineProperty(navigator, "clipboard", { value: { writeText: vi.fn().mockRejectedValue(new Error("denied")) }, configurable: true });
    const { container } = renderChat();
    fireEvent.click(row(container, 1).getByRole("button", { name: "Copy" }));
    await waitFor(() => expect(pushError).toHaveBeenCalledWith("Copy failed", "denied"));
    expect(row(container, 1).getByRole("button", { name: "Copy" })).toBeInTheDocument();
  });
});

it("keeps a wire turn end's fork affordance through normalization", async () => {
  const { normalizeEvent } = await import("../../store/transcriptStore");
  const event = normalizeEvent({ type: "turn_end", seq: 3, data: { stop_reason: "end_turn" }, fork: { available: false, reason: "No fork point." } });
  expect(event).toMatchObject({ kind: "turn_end", stop_reason: "end_turn", fork: { available: false, reason: "No fork point." } });
});
