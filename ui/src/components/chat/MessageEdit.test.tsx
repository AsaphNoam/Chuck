import React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import * as client from "../../api/client";
import type { AgentState, TranscriptEvent } from "../../api/types";
import { useAgentStore } from "../../store/agentStore";
import { useTranscriptStore } from "../../store/transcriptStore";
import { useUiStore } from "../../store/uiStore";
import { clearChatDraftCacheForTests, getChatDraft, setChatDraft } from "./drafts";
import { editActionFor } from "./forkActions";
import { TranscriptView } from "./TranscriptView";

const initialUi = useUiStore.getState();

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  useUiStore.setState(initialUi, true);
  useAgentStore.setState({ agents: {} });
  useTranscriptStore.setState({ byAgent: {}, revisionRead: {} });
  clearChatDraftCacheForTests();
});

const ok = { available: true };
const events: TranscriptEvent[] = [
  { kind: "user_text", seq: 1, text: "first" },
  { kind: "assistant_text", seq: 2, text: "reply one" },
  { kind: "turn_end", seq: 3, stop_reason: "end_turn", fork: ok },
  { kind: "user_text", seq: 4, text: "second" },
  { kind: "assistant_text", seq: 5, text: "reply two" },
  { kind: "turn_end", seq: 6, stop_reason: "end_turn", fork: ok },
];
const agent = (extra: Partial<AgentState> = {}) => ({
  agent_id: "a1", clone: ok, history_revision: 3, edit: ok, runtime_capabilities: { complete_turn_fork: true }, ...extra,
}) as unknown as AgentState;

function renderChat(transcript = events, state = agent()) {
  useAgentStore.setState({ agents: { a1: state } });
  const queries = new QueryClient({ defaultOptions: { queries: { retry: 0 } } });
  const ui = (list: TranscriptEvent[]) => (
    <QueryClientProvider client={queries}>
      <MemoryRouter><TranscriptView agentId="a1" events={list} forkActionsEnabled /></MemoryRouter>
    </QueryClientProvider>
  );
  const view = render(ui(transcript));
  return { ...view, rerenderEvents: (list: TranscriptEvent[]) => view.rerender(ui(list)) };
}

function row(container: HTMLElement, seq: number) {
  return within(container.querySelector(`[data-seq="${seq}"] .message-actions`) as HTMLElement);
}

const result = (extra: Partial<client.EditResult> = {}): client.EditResult => ({
  agent_id: "a1", request_id: "r", phase: "committing", revision: 4, delivery: "pending", active: "original", ...extra,
});

function openEditor(container: HTMLElement) {
  fireEvent.click(row(container, 4).getByRole("button", { name: "Edit" }));
  return screen.getByRole("textbox", { name: "Edit message" }) as HTMLTextAreaElement;
}

describe("Edit editor stays clear of Jump to latest (FS-12.R24, TS-08.R171)", () => {
  it("hides Jump to latest while the editor is open and restores it after", () => {
    const { container } = renderChat();
    const list = container.querySelector(".transcript-view") as HTMLElement;
    Object.defineProperty(list, "scrollHeight", { configurable: true, get: () => 1000 });
    Object.defineProperty(list, "clientHeight", { configurable: true, get: () => 100 });
    Object.defineProperty(list, "scrollTop", { configurable: true, writable: true, value: 900 });
    fireEvent.scroll(list);
    list.scrollTop = 100;
    fireEvent.scroll(list);
    expect(screen.getByRole("button", { name: "Jump to latest" })).toBeInTheDocument();

    openEditor(container);
    fireEvent.scroll(list);
    expect(screen.queryByRole("button", { name: "Jump to latest" })).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.getByRole("button", { name: "Jump to latest" })).toBeInTheDocument();
  });
});

describe("Edit eligibility (FS-01.R61)", () => {
  it("offers Edit only on the latest sent message", () => {
    const { container } = renderChat();
    expect(row(container, 4).getByRole("button", { name: "Edit" })).not.toHaveAttribute("aria-disabled");
    for (const seq of [1, 2, 5]) expect(row(container, seq).queryByRole("button", { name: "Edit" })).toBeNull();
  });

  const cases: Array<[string, TranscriptEvent[], Partial<AgentState>, string]> = [
    ["the server's live gate", events, { edit: { available: false, reason: "The agent is working." } }, "The agent is working."],
    ["a message sent while working", [...events.slice(0, 3), { kind: "user_text", seq: 4, text: "a" }, { kind: "user_text", seq: 5, text: "b" }, { kind: "turn_end", seq: 6, fork: ok }], {}, "This message was sent while the agent was working; edit the message that began that turn."],
    ["an unfinished turn", events.slice(0, 5), {}, "Wait for this turn to finish."],
    ["a later turn the agent started", [...events, { kind: "assistant_text", seq: 7, text: "mail arrived" }, { kind: "turn_end", seq: 8, fork: ok }], {}, "The agent took another turn after this message, so editing it would hide that turn."],
    ["an earlier turn that cannot fork", [{ ...events[0] }, events[1], { kind: "turn_end", seq: 3, fork: { available: false, reason: "No fork point." } }, ...events.slice(3)], {}, "No fork point."],
    ["an adapter that cannot prove completeness", events, { runtime_capabilities: { complete_turn_fork: false } as never }, "This agent's adapter cannot prove a complete earlier turn; update to the supported package to edit later messages."],
  ];
  it.each(cases)("explains %s without opening an editor", (_name, list, extra, reason) => {
    const { container } = renderChat(list, agent(extra));
    const seq = [...list].reverse().find((event) => event.kind === "user_text")!.seq as number;
    const edit = row(container, seq).getByRole("button", { name: "Edit" });
    expect(edit).toHaveAttribute("aria-disabled", "true");
    expect(edit).toHaveAccessibleDescription(`Edit unavailable: ${reason}`);
    fireEvent.click(edit);
    expect(screen.queryByRole("textbox", { name: "Edit message" })).toBeNull();
  });

  it("needs neither fork marker nor capability for the first message", () => {
    const first = events.slice(0, 3);
    expect(editActionFor(first, first[0], agent({ runtime_capabilities: undefined }))).toEqual({ seq: 1, available: true });
  });

  it("offers Edit in the context menu, replacing the clone-based Edit", () => {
    const { container } = renderChat();
    fireEvent.contextMenu(container.querySelector('[data-seq="4"]') as HTMLElement, { clientX: 5, clientY: 5 });
    expect(within(screen.getByRole("menu")).getAllByRole("button", { name: "Edit" })).toHaveLength(1);
    fireEvent.click(within(screen.getByRole("menu")).getByRole("button", { name: "Edit" }));
    expect(screen.getByRole("textbox", { name: "Edit message" })).toHaveValue("second");
  });
});

describe("inline editor (TS-08.R171)", () => {
  it("seeds the message, keeps the composer draft and restores on Cancel", async () => {
    setChatDraft("a1", "my draft");
    const { container } = renderChat();
    const opener = row(container, 4).getByRole("button", { name: "Edit" });
    opener.focus();
    const input = openEditor(container);
    expect(input).toHaveValue("second");
    expect(screen.getByText("Saving replaces this message and its reply. Tool actions and file changes already made stay in effect.")).toBeInTheDocument();
    fireEvent.change(input, { target: { value: "changed" } });
    expect(getChatDraft("a1")).toBe("my draft");
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.queryByRole("textbox", { name: "Edit message" })).toBeNull();
    expect(screen.getByText("second")).toBeInTheDocument();
    expect(getChatDraft("a1")).toBe("my draft");
    await waitFor(() => expect(row(container, 4).getByRole("button", { name: "Edit" })).toHaveFocus());
  });

  it("closes on Escape and disables Save for empty text", () => {
    const { container } = renderChat();
    const input = openEditor(container);
    fireEvent.change(input, { target: { value: "   " } });
    expect(screen.getByRole("button", { name: "Save and regenerate" })).toBeDisabled();
    fireEvent.keyDown(input, { key: "Escape" });
    expect(screen.queryByRole("textbox", { name: "Edit message" })).toBeNull();
  });

  it("posts exactly once with the revision and shows Editing until the revision is read back", async () => {
    let resolve!: (value: client.EditResult) => void;
    const post = vi.spyOn(client, "editLastMessage").mockReturnValue(new Promise((done) => { resolve = done; }));
    const { container } = renderChat();
    fireEvent.change(openEditor(container), { target: { value: "better" } });
    const save = screen.getByRole("button", { name: "Save and regenerate" });
    fireEvent.click(save);
    fireEvent.click(save);
    expect(post).toHaveBeenCalledTimes(1);
    const body = post.mock.calls[0][1];
    expect(post.mock.calls[0][0]).toBe("a1");
    expect(body).toMatchObject({ message_seq: 4, expected_revision: 3, text: "better" });
    expect(body.request_id).toMatch(/\S/);
    await act(async () => resolve(result()));
    expect(screen.getByText("Editing…")).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: "Edit message" })).toBeDisabled();
    act(() => useTranscriptStore.getState().markRevisionRead("a1", 4));
    await waitFor(() => expect(screen.queryByRole("textbox", { name: "Edit message" })).toBeNull());
  });

  it("closes when the edited row is replaced", () => {
    const { container, rerenderEvents } = renderChat();
    openEditor(container);
    rerenderEvents([{ kind: "user_text", seq: 7, text: "better" }]);
    expect(screen.queryByRole("textbox", { name: "Edit message" })).toBeNull();
  });

  it("keeps the text and shows the reason when the operation aborts", async () => {
    vi.spyOn(client, "editLastMessage").mockResolvedValue(result({ phase: "aborted", reason: "The agent could not rewind." }));
    const { container } = renderChat();
    fireEvent.change(openEditor(container), { target: { value: "better" } });
    fireEvent.click(screen.getByRole("button", { name: "Save and regenerate" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("The agent could not rewind.");
    expect(screen.getByRole("textbox", { name: "Edit message" })).toHaveValue("better");
    expect(screen.getByRole("button", { name: "Save and regenerate" })).toBeEnabled();
  });

  it("reads the result when an accepted edit aborts later and is no longer open", async () => {
    // The snapshot lists only open edits, so a later abort is learned from the status route.
    vi.spyOn(client, "editLastMessage").mockResolvedValue(result({ phase: "preparing" }));
    const status = vi.spyOn(client, "getEditStatus").mockResolvedValue(result({ phase: "aborted", reason: "Nothing was changed: the fork failed." }));
    const { container } = renderChat();
    fireEvent.change(openEditor(container), { target: { value: "better" } });
    fireEvent.click(screen.getByRole("button", { name: "Save and regenerate" }));
    expect(await screen.findByText("Nothing was changed: the fork failed.", {}, { timeout: 3000 })).toBeInTheDocument();
    const [, requestId] = status.mock.calls[0];
    expect(requestId).toBe((client.editLastMessage as unknown as { mock: { calls: [string, { request_id: string }][] } }).mock.calls[0][1].request_id);
    expect(screen.getByRole("textbox", { name: "Edit message" })).toHaveValue("better");
  });

  it("queries status with the same request id after a lost response before allowing another Save", async () => {
    const post = vi.spyOn(client, "editLastMessage").mockRejectedValue(new TypeError("network down"));
    const status = vi.spyOn(client, "getEditStatus").mockRejectedValueOnce(new TypeError("still down")).mockResolvedValueOnce(result({ phase: "committed" }));
    const { container } = renderChat();
    fireEvent.change(openEditor(container), { target: { value: "better" } });
    fireEvent.click(screen.getByRole("button", { name: "Save and regenerate" }));
    await waitFor(() => expect(status).toHaveBeenCalledTimes(1));
    const id = post.mock.calls[0][1].request_id;
    expect(status).toHaveBeenCalledWith("a1", id);
    expect(await screen.findByRole("alert")).toHaveTextContent("Could not confirm");
    expect(screen.getByRole("button", { name: "Save and regenerate" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Check status" }));
    await screen.findByText("Editing…");
    expect(status).toHaveBeenLastCalledWith("a1", id);
    expect(post).toHaveBeenCalledTimes(1);
  });

  it("keeps the text on a stale-revision conflict and reloads the latest messages", async () => {
    vi.spyOn(client, "editLastMessage").mockRejectedValue(Object.assign(new Error("stale"), { status: 409, code: "edit_conflict" }));
    const reload = vi.spyOn(client, "getTranscript").mockResolvedValue({ agent_id: "a1", events });
    const { container } = renderChat();
    fireEvent.change(openEditor(container), { target: { value: "better" } });
    fireEvent.click(screen.getByRole("button", { name: "Save and regenerate" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("This conversation changed. Reload the latest messages to edit again.");
    expect(screen.getByRole("textbox", { name: "Edit message" })).toHaveValue("better");
    fireEvent.click(screen.getByRole("button", { name: "Reload messages" }));
    await waitFor(() => expect(reload).toHaveBeenCalledWith("a1"));
    await waitFor(() => expect(screen.queryByRole("alert")).toBeNull());
    expect(screen.getByRole("textbox", { name: "Edit message" })).toHaveValue("better");
  });
});
