import React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { MemoryRouter } from "react-router-dom";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import * as client from "../../api/client";
import type { TranscriptEvent } from "../../api/types";
import { useAgentStore } from "../../store/agentStore";
import { useAnnotationStore } from "../../store/annotationStore";
import { useHeldStore } from "../../store/heldStore";
import { foldTranscript, useTranscriptStore } from "../../store/transcriptStore";
import { annotationBlockSentinel } from "../../lib/annotations";
import { TranscriptView } from "./TranscriptView";

const mswServer = setupServer();
beforeAll(() => mswServer.listen({ onUnhandledRequest: "bypass" }));
afterAll(() => mswServer.close());

afterEach(() => {
  cleanup();
  window.getSelection()?.removeAllRanges();
  useAnnotationStore.setState({ bySource: {}, overallBySource: {}, editedAt: {}, collapsedBySource: {} });
  useTranscriptStore.setState({ byAgent: {}, rawByAgent: {}, pending: {}, settled: {}, appliedSeq: {} });
  useHeldStore.setState({ byAgent: {}, afterSeqByAgent: {}, attachmentsByAgent: {}, recoveryByAgent: {}, recoveryUnreadByAgent: {} });
  useAgentStore.setState({ agents: {} });
  mswServer.resetHandlers();
});

const events = [{ kind: "assistant_text", seq: 7, text: "First line\nSecond line" }];

function renderTranscript(annotationsEnabled = true, transcriptEvents = events, busy = false) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: 0 } } });
  return render(
    <QueryClientProvider client={client}>
      <TranscriptView agentId="a1" events={transcriptEvents} sourceActive annotationsEnabled={annotationsEnabled} busy={busy} />
    </QueryClientProvider>,
  );
}

function selectWithin(node: Node) {
  const range = document.createRange();
  range.selectNodeContents(node);
  const selection = window.getSelection();
  selection?.removeAllRanges();
  selection?.addRange(range);
}

describe("TranscriptView annotation entry point", () => {
  it("shows no standing Annotate control on transcript events", () => {
    renderTranscript();

    expect(screen.queryByRole("button", { name: /annotate/i })).toBeNull();
  });

  it("captures the highlighted text when the selection is right-clicked", () => {
    const { container } = renderTranscript();
    const item = container.querySelector('[data-slot="event"]') as HTMLElement;
    const text = document.createTextNode("Second line");
    item.appendChild(text);
    selectWithin(text);

    fireEvent.contextMenu(item, { clientX: 20, clientY: 30 });
    fireEvent.click(screen.getByRole("button", { name: "Annotate selection" }));

    expect(useAnnotationStore.getState().bySource.a1).toEqual([
      { seq: 7, excerpt: "Second line", instruction: "" },
    ]);
  });

  it("trims surrounding whitespace from an annotated selection", () => {
    const { container } = renderTranscript();
    const item = container.querySelector('[data-slot="event"]') as HTMLElement;
    const text = document.createTextNode("  Second line  ");
    item.appendChild(text);
    selectWithin(text);

    fireEvent.contextMenu(item, { clientX: 20, clientY: 30 });
    fireEvent.click(screen.getByRole("button", { name: "Annotate selection" }));

    expect(useAnnotationStore.getState().bySource.a1).toEqual([
      { seq: 7, excerpt: "Second line", instruction: "" },
    ]);
  });

  it("copies the exact highlighted text without creating an annotation", () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
    const { container } = renderTranscript();
    const item = container.querySelector('[data-slot="event"]') as HTMLElement;
    const text = document.createTextNode("  Second line  ");
    item.appendChild(text);
    selectWithin(text);

    fireEvent.contextMenu(item, { clientX: 20, clientY: 30 });

    expect(screen.getByRole("button", { name: "Annotate selection" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Copy selection" }));
    expect(writeText).toHaveBeenCalledWith("  Second line  ");
    expect(useAnnotationStore.getState().bySource.a1).toBeUndefined();
  });

  it("falls back to the whole event when nothing is highlighted", () => {
    const { container } = renderTranscript();
    const item = container.querySelector('[data-slot="event"]') as HTMLElement;

    fireEvent.contextMenu(item, { clientX: 20, clientY: 30 });
    fireEvent.click(screen.getByRole("button", { name: "Annotate whole event" }));

    expect(useAnnotationStore.getState().bySource.a1).toEqual([
      { seq: 7, excerpt: "First line\nSecond line", instruction: "" },
    ]);
  });

  it("leaves the native menu alone where annotations are disabled", () => {
    const { container } = renderTranscript(false);
    const item = container.querySelector('[data-slot="event"]') as HTMLElement;

    const opened = fireEvent.contextMenu(item, { clientX: 20, clientY: 30 });

    expect(opened).toBe(true); // preventDefault was not called
    expect(screen.queryByRole("menu")).toBeNull();
  });

  it("collapses uninterrupted tool activity and omits successful no-payload outcomes", () => {
    renderTranscript(false, [
      { kind: "tool_call", seq: 1, name: "Read", args: { path: "a.ts" } },
      { kind: "tool_result", seq: 2, status: "completed", content: null },
      { kind: "tool_call", seq: 3, name: "Search" },
      { kind: "tool_result", seq: 4, status: "completed", content: "Found one match." },
    ]);

    const summary = screen.getByRole("button", { name: "Ran 2 tools" });
    expect(screen.queryByText(/Tool call:/)).toBeNull();
    expect(screen.queryByText("Completed")).toBeNull();

    fireEvent.click(summary);

    expect(screen.getAllByText(/Tool call:/)).toHaveLength(2);
    expect(screen.getByText("Found one match.")).toBeInTheDocument();
    expect(screen.queryByText("Completed")).toBeNull();
  });

  it("keeps an approved tool's permission and result inside its run (FS-03.A11)", () => {
    renderTranscript(false, [
      { kind: "tool_call", seq: 1, tool_call_id: "t1", name: "Edit" },
      { kind: "permission_request", seq: 2, tool_call_id: "t1", name: "Edit HANDOFF.md", resolved: "approve" },
      { kind: "tool_result", seq: 3, tool_call_id: "t1", status: "completed", content: "Edited." },
    ]);

    const summary = screen.getByRole("button", { name: "Ran 1 tool" });
    expect(screen.queryByText("Edited.")).toBeNull();
    expect(screen.queryByText(/Edit HANDOFF\.md/)).toBeNull();

    fireEvent.click(summary);

    expect(screen.getByText(/Edit HANDOFF\.md/)).toBeInTheDocument();
    expect(screen.getByText("Edited.")).toBeInTheDocument();
  });

  // FS-03.A66: file changes stay out of the stream; a Files-tab reveal shows only its diff.
  it("hides diffs unless one is revealed or the view keeps them inline", () => {
    const withDiffs = [
      { kind: "assistant_text", seq: 1, text: "Edited both files." },
      { kind: "diff", seq: 2, path: "a.go", old_text: "a", new_text: "b" },
      { kind: "diff", seq: 3, path: "c.go", old_text: "c", new_text: "d" },
    ];
    const client = new QueryClient({ defaultOptions: { queries: { retry: 0 } } });
    const view = (props: { reveal?: { seq: number }; inlineDiffs?: boolean }) => (
      <QueryClientProvider client={client}><TranscriptView agentId="a1" events={withDiffs} {...props} /></QueryClientProvider>
    );
    const { rerender } = render(view({}));
    expect(screen.getByText("Edited both files.")).toBeInTheDocument();
    expect(document.querySelector(".diff-block")).toBeNull();
    rerender(view({ reveal: { seq: 3 } }));
    expect([...document.querySelectorAll(".diff-block")].map((node) => node.textContent)).toEqual([expect.stringContaining("c.go")]);
    rerender(view({ inlineDiffs: true }));
    expect(document.querySelectorAll(".diff-block")).toHaveLength(2);
  });

  it("starts a new tool run after non-tool activity", () => {
    renderTranscript(false, [
      { kind: "tool_call", seq: 1, name: "Read" },
      { kind: "tool_result", seq: 2, status: "completed", content: null },
      { kind: "assistant_text", seq: 3, text: "I found the file." },
      { kind: "tool_call", seq: 4, name: "Edit" },
      { kind: "tool_result", seq: 5, status: "completed", content: null },
    ]);

    expect(screen.getAllByRole("button", { name: "Ran 1 tool" })).toHaveLength(2);
    expect(screen.getByText("I found the file.")).toBeInTheDocument();
  });

  it("shows a waiting indicator only while the agent is busy", () => {
    const { rerender } = renderTranscript(false, events, false);
    expect(screen.queryByText("Working…")).toBeNull();

    rerender(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: 0 } } })}>
        <TranscriptView agentId="a1" events={events} sourceActive annotationsEnabled={false} busy />
      </QueryClientProvider>,
    );

    expect(screen.getByText("Working…")).toBeInTheDocument();
  });

  it("keeps expanded tool calls individually annotatable", () => {
    const { container } = renderTranscript(true, [
      { kind: "tool_call", seq: 1, name: "Read" },
      { kind: "tool_result", seq: 2, status: "completed", content: null },
    ]);

    fireEvent.click(screen.getByRole("button", { name: "Ran 1 tool" }));
    fireEvent.contextMenu(container.querySelector('[data-seq="1"]') as HTMLElement, { clientX: 20, clientY: 30 });
    fireEvent.click(screen.getByRole("button", { name: "Annotate whole event" }));

    expect(useAnnotationStore.getState().bySource.a1).toMatchObject([{ seq: 1 }]);
  });
});

// FS-13.A14 (R23). Sending a batch to the current agent starts an ordinary
// prompt turn whose text is the machine block, so the transcript would otherwise
// show the same excerpts twice: once as the card, once as a wall of generated
// prose the person never wrote.
describe("TranscriptView self-annotation prompt", () => {
  const block = `${annotationBlockSentinel}\n\n1. Transcript event 7\nExcerpt:\n---\nFirst line\n---\nInstruction: tighten this\n`;
  const annotationEvent = {
    type: "annotation",
    seq: 8,
    data: { annotations: [{ seq: 7, excerpt: "First line", instruction: "tighten this" }], target: { kind: "self" } },
  };
  const promptEvent = { type: "user_text", seq: 9, data: { text: block } };

  function expectQuieted() {
    expect(screen.getByText("Annotations assigned")).toBeInTheDocument();
    expect(screen.getByText("tighten this")).toBeInTheDocument();
    expect(screen.queryByText(annotationBlockSentinel, { exact: false })).toBeNull();
  }

  it("draws the card alone when the prompt arrives live", () => {
    const append = useTranscriptStore.getState().appendMessage;
    append("a1", annotationEvent);
    append("a1", promptEvent);

    renderTranscript(true, useTranscriptStore.getState().byAgent.a1);

    expectQuieted();
  });

  // The same list on replay: a rule that lived on only one of the two paths
  // would draw an event that the next reload then removes (INV §2).
  it("draws the same list when the transcript is replayed", () => {
    renderTranscript(true, foldTranscript([annotationEvent, promptEvent]));

    expectQuieted();
    expect(foldTranscript([annotationEvent, promptEvent])).toHaveLength(1);
  });

  // FS-13.A15 (R24): transcripts recorded before the rename keep the old first
  // line and still render as a card beside new-spelling blocks.
  it("quiets a block written with the pre-rename first line", () => {
    const legacyPrompt = { ...promptEvent, data: { text: block.replace(annotationBlockSentinel, "[AgentDeck annotations]") } };
    const later = { ...annotationEvent, seq: 10 };
    const rendered = foldTranscript([annotationEvent, legacyPrompt, later, { ...promptEvent, seq: 11 }]);
    renderTranscript(true, rendered);

    expect(rendered).toHaveLength(2);
    expect(screen.getAllByText("Annotations assigned")).toHaveLength(2);
    expect(screen.queryByText("[AgentDeck annotations]", { exact: false })).toBeNull();
  });

  // The three conditions are load-bearing. A batch assigned to another agent
  // leaves the source free to keep talking, and that next message is ordinary
  // prose that must still appear.
  it("keeps a prompt that follows a batch assigned to another agent", () => {
    const assigned = { ...annotationEvent, data: { ...annotationEvent.data, target: { kind: "agent", agent_id: "a2" } } };
    renderTranscript(true, foldTranscript([assigned, { type: "user_text", seq: 9, data: { text: "carry on without me" } }]));

    expect(screen.getByText("carry on without me")).toBeInTheDocument();
  });

  it("keeps a prompt that does not immediately follow its annotation event", () => {
    const rendered = foldTranscript([
      annotationEvent,
      { type: "assistant_text", seq: 9, data: { text: "on it" } },
      { ...promptEvent, seq: 10 },
    ]);
    renderTranscript(true, rendered);

    expect(screen.getByText(annotationBlockSentinel, { exact: false })).toBeInTheDocument();
  });
});

// TS-08.R56 — the queued follow-up is a transcript-tail affordance, not a
// transcript event: it renders beside the list the fold builds, reads as
// not-yet-sent, and carries its own withdraw control.
describe("TranscriptView pending follow-up", () => {
  it("renders nothing when no message is held", () => {
    renderTranscript(true, events, true);

    expect(screen.queryByText(/^Queued/)).toBeNull();
  });

  it("shows a held message as queued and outside the folded event list", () => {
    useHeldStore.getState().hold("a1", "also update the docs");
    renderTranscript(true, events, true);

    expect(screen.getByText("Queued — sends when this turn ends")).toBeInTheDocument();
    expect(screen.getByText("also update the docs")).toBeInTheDocument();
    // It is beside the list, not an item in it — a live render and a reload
    // cannot disagree about something the fold never sees.
    expect(document.querySelector('[data-variant="held"]')?.closest(".transcript-item")).toBeNull();
    expect(useTranscriptStore.getState().byAgent.a1).toBeUndefined();
  });

  it("keeps the message visible when withdrawing it fails", async () => {
    useHeldStore.getState().hold("a1", "still queued");
    const failing = vi.spyOn(client, "withdrawPrompt").mockRejectedValue(new Error("gone"));
    renderTranscript(true, events, true);

    fireEvent.click(screen.getByRole("button", { name: "Withdraw" }));
    expect(await screen.findByText("Could not withdraw — it may already have been sent.")).toBeInTheDocument();
    expect(useHeldStore.getState().byAgent.a1).toBe("still queued");
    failing.mockRestore();
  });

  // FS-03.A70 — a completed attachment withdrawal whose recovery read fails says
  // so and reads again without withdrawing twice.
  it("offers to load a withdrawn attachment message when its recovery read fails", async () => {
    const attachment = { id: "att_1", name: "shot.png", kind: "image", mime_type: "image/png", size: 4, sha256: "abcd" } as const;
    const recovery = { text: "see image", request_id: "req_1", attachments: [attachment] };
    let withdrawn = 0;
    let reads = 0;
    mswServer.use(
      http.delete("/api/sessions/a1/prompt", () => { withdrawn++; return HttpResponse.json({ accepted: true, agent_id: "a1", withdrawn: true, message: { held_id: "hold-1", text: "see image", attachments: [attachment] } }); }),
      http.get("/api/sessions/a1/prompt", () => {
        reads++;
        return reads === 1 ? HttpResponse.json({ error: "down" }, { status: 503 }) : HttpResponse.json({ agent_id: "a1", text: "", after_seq: 0, recovery });
      }),
    );
    useHeldStore.getState().hold("a1", "see image", 0, [attachment]);
    renderTranscript(true, events, true);

    fireEvent.click(screen.getByRole("button", { name: "Withdraw" }));
    expect(await screen.findByText("Withdrawn, but the message and its attachments could not be loaded.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Load withdrawn message" }));
    expect(await screen.findByText("Message not delivered — choose whether to send it again")).toBeInTheDocument();
    expect(screen.queryByText("Withdrawn, but the message and its attachments could not be loaded.")).toBeNull();
    expect(withdrawn).toBe(1);
  });

  it("labels text recovery with uncertain delivery and accepts empty attachments", async () => {
    useHeldStore.getState().setRecovery("a1", { text: "check the deploy", request_id: "uncertain-1", delivery: "unknown" });
    renderTranscript(true, events, true);
    expect(await screen.findByText("This message may already have reached the agent — sending again may repeat it")).toBeInTheDocument();
    expect(screen.queryByText("Message not delivered — choose whether to send it again")).toBeNull();
  });
});

// FS-03.A46, FS-13.A16 (J13): the file viewer's own selection menu is reached
// through TranscriptView's shared annotate-selection entry point, and the
// draft it hands the store must carry the file's exact anchor.
describe("TranscriptView file annotation integration (J13)", () => {
  function selectBetween(start: Element, end: Element) {
    const range = document.createRange();
    range.setStart(start, 0);
    range.setEnd(end, end.childNodes.length);
    const selection = window.getSelection();
    selection?.removeAllRanges();
    selection?.addRange(range);
  }

  it("captures the file's exact anchor when a selection is annotated through the menu", async () => {
    mswServer.use(
      http.get("/api/sessions/:id/file", () =>
        HttpResponse.json({
          agent_id: "a1",
          path: "internal/state/messages.go",
          size: 30,
          mod_time: "2026-09-10T10:00:00Z",
          line_count: 3,
          content: "package state\n\nfunc Read() {}\n",
          truncated: false,
          language: "go",
        }),
      ),
    );
    const onOpenFile = vi.fn();
    const client_ = new QueryClient({ defaultOptions: { queries: { retry: 0 } } });
    render(
      <QueryClientProvider client={client_}>
        <TranscriptView
          agentId="a1"
          events={events}
          sourceActive
          annotationsEnabled
          openFile={{ path: "internal/state/messages.go" }}
          onOpenFile={onOpenFile}
        />
      </QueryClientProvider>,
    );

    await screen.findByText("internal/state/messages.go");
    const first = await waitFor(() => document.querySelector('[data-file-line="1"]') as HTMLElement);
    const second = document.querySelector('[data-file-line="2"]') as HTMLElement;
    selectBetween(first, second);
    fireEvent.contextMenu(document.querySelector(".file-viewer-body") as HTMLElement, { clientX: 5, clientY: 6 });
    fireEvent.click(await screen.findByRole("button", { name: "Annotate selection" }));

    expect(useAnnotationStore.getState().bySource.a1).toEqual([
      expect.objectContaining({ anchor_kind: "file", path: "internal/state/messages.go", start_line: 1, end_line: 2, instruction: "" }),
    ]);
  });
});

// FS-13.A16, TS-02.R38, TS-03.R49, TS-08.R80 (J13): the tray holds a
// transcript-anchored and a file-anchored draft together, survives being
// mounted fresh (what a reload restores from the persisted store), and can
// still be edited, have one draft removed, and send the exact remaining
// batch as the wire payload.
describe("TranscriptView mixed annotation tray (J13)", () => {
  function seedMixedTray() {
    useAnnotationStore.setState({
      bySource: {
        a1: [
          { seq: 7, excerpt: "transcript excerpt", instruction: "initial instruction" },
          { anchor_kind: "file", path: "src/a.go", start_line: 2, end_line: 4, excerpt: "file excerpt", instruction: "" },
        ],
      },
      overallBySource: {},
      editedAt: { a1: Date.now() },
      collapsedBySource: {},
    });
  }

  // A page reload: keep only what persist wrote to storage, then rehydrate.
  async function reloadTrays() {
    const key = useAnnotationStore.persist.getOptions().name as string;
    const saved = localStorage.getItem(key) as string;
    useAnnotationStore.setState({ bySource: {}, overallBySource: {}, editedAt: {}, collapsedBySource: {} });
    localStorage.setItem(key, saved);
    await useAnnotationStore.persist.rehydrate();
  }

  function renderMixed() {
    const client_ = new QueryClient({ defaultOptions: { queries: { retry: 0 } } });
    return render(
      <QueryClientProvider client={client_}>
        <TranscriptView agentId="a1" events={events} sourceActive annotationsEnabled />
      </QueryClientProvider>,
    );
  }

  it("preserves both drafts across a fresh mount, then edits, removes, and sends the exact remaining batch", async () => {
    seedMixedTray();
    await reloadTrays();
    renderMixed();

    // What a reload restores: both anchors are still present and readable.
    expect(screen.getByText("transcript excerpt")).toBeInTheDocument();
    expect(screen.getByText("file excerpt")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "File src/a.go:2–4" })).toBeInTheDocument();

    // Edit the file draft's instruction.
    const fileRow = screen.getByRole("heading", { name: "File src/a.go:2–4" }).closest(".annotation-draft") as HTMLElement;
    fireEvent.change(within(fileRow).getByLabelText("Instruction"), { target: { value: "tighten this excerpt" } });
    expect(useAnnotationStore.getState().bySource.a1[1].instruction).toBe("tighten this excerpt");

    // Remove the transcript draft, leaving only the edited file draft.
    const transcriptRow = screen.getByText("transcript excerpt").closest(".annotation-draft") as HTMLElement;
    fireEvent.click(within(transcriptRow).getByRole("button", { name: "Remove" }));
    expect(useAnnotationStore.getState().bySource.a1).toHaveLength(1);

    let sentBody: unknown = null;
    mswServer.use(
      http.post("/api/sessions/a1/annotations", async ({ request }) => {
        sentBody = await request.json();
        return HttpResponse.json({ accepted: true, seq: 99 }, { status: 202 });
      }),
    );

    fireEvent.click(screen.getByRole("button", { name: "Send annotations" }));

    await waitFor(() => expect(useAnnotationStore.getState().bySource.a1).toBeUndefined());
    expect(sentBody).toMatchObject({
      annotations: [
        { anchor_kind: "file", path: "src/a.go", start_line: 2, end_line: 4, excerpt: "file excerpt", instruction: "tighten this excerpt" },
      ],
      target: { kind: "self" },
    });
    expect((sentBody as { command_id?: string }).command_id).toEqual(expect.any(String));
  });

  // A file annotation persisted with an anchor the server would reject (a
  // start_line with no end_line) must stay in the tray rather than being
  // silently dropped on the next mount — only Remove or Send should end it.
  it("preserves a persisted file annotation with an invalid anchor rather than dropping it", async () => {
    useAnnotationStore.setState({
      bySource: {
        a1: [
          { anchor_kind: "file", path: "src/broken.go", start_line: 5, excerpt: "broken anchor", instruction: "note" },
        ],
      },
      overallBySource: {},
      editedAt: { a1: Date.now() },
      collapsedBySource: {},
    });
    await reloadTrays();
    renderMixed();

    expect(screen.getByText("broken anchor")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "File src/broken.go:5" })).toBeInTheDocument();
    expect(useAnnotationStore.getState().bySource.a1).toHaveLength(1);
  });
});

// FS-03.A49, TS-08.R86: notices render as compact rows in their original order,
// with unknown severity read as info, through the same fold as replay.
describe("TranscriptView notices", () => {
  it("renders each severity as a compact row between the surrounding messages", () => {
    const raw = [
      { agent_id: "a1", seq: 1, type: "assistant_text", ts: "", data: { delta: "Before." } },
      { agent_id: "a1", seq: 2, type: "notice", ts: "", data: { severity: "info", title: "Context compacted" } },
      { agent_id: "a1", seq: 3, type: "notice", ts: "", data: { severity: "warning", title: "Usage limit", description: "90% used." } },
      { agent_id: "a1", seq: 4, type: "notice", ts: "", data: { severity: "error", title: "Hook failed" } },
      { agent_id: "a1", seq: 5, type: "assistant_text", ts: "", data: { delta: "After." } },
    ];
    const { container } = renderTranscript(false, foldTranscript(raw));
    const variants = [...container.querySelectorAll('[data-slot="event"]')].map((node) => node.getAttribute("data-variant"));
    expect(variants).toEqual(["assistant", "notice", "notice", "notice", "assistant"]);
    const notes = screen.getAllByRole("note");
    expect(notes.map((note) => note.textContent)).toEqual([
      "infoContext compacted",
      "warningUsage limit90% used.",
      "infoHook failed",
    ]);
    expect(within(notes[1]).getByText("warning").closest('[data-ui="badge"]')).toHaveAttribute("data-variant", "warning");
    expect(within(notes[0]).getByText("info").closest('[data-ui="badge"]')).toHaveAttribute("data-variant", "neutral");
    expect(container.querySelector('[data-variant="notice"] .message')).toBeNull();
  });
});

describe("TranscriptView quiet completed turns (FS-03.A55–A57)", () => {
  const turn = (base: number, answer: string): TranscriptEvent[] => [
    { kind: "user_text", seq: base, text: `Ask ${base}` },
    { kind: "assistant_text", seq: base + 1, text: `Working ${base}` },
    { kind: "tool_call", seq: base + 2, tool_call_id: `t${base}`, name: "Read" },
    { kind: "tool_result", seq: base + 3, tool_call_id: `t${base}`, status: "completed", content: "out" },
    { kind: "assistant_text", seq: base + 4, text: answer },
    { kind: "turn_end", seq: base + 5, stop_reason: "end_turn" },
  ];

  it("leaves input and response visible behind one keyboard-operable control per turn", () => {
    renderTranscript(true, turn(1, "Final one"));
    expect(screen.getByText("Ask 1")).toBeInTheDocument();
    expect(screen.getByText("Final one")).toBeInTheDocument();
    expect(screen.queryByText("Working 1")).toBeNull();
    expect(screen.queryByRole("button", { name: /Ran 1 tool/ })).toBeNull();

    const control = screen.getByRole("button", { name: /Show activity/ });
    expect(control).toHaveAttribute("aria-expanded", "false");
    fireEvent.click(control);
    expect(control).toHaveTextContent("Hide activity");
    const region = document.getElementById(control.getAttribute("aria-controls")!.split(" ")[0])!;
    expect(within(region).getByText("Working 1")).toBeInTheDocument();
    expect(within(region).getByRole("button", { name: /Ran 1 tool/ })).toBeInTheDocument();
    // Order is preserved: the hidden rows sit before the response.
    const variants = [...document.querySelectorAll('[data-slot="event"]')].map((node) => node.getAttribute("data-variant"));
    expect(variants).toEqual(["user", "assistant", "assistant", "turn"]);
    fireEvent.click(control);
    expect(screen.queryByText("Working 1")).toBeNull();
  });

  it("keeps an older reopened turn open when a newer turn completes", () => {
    const client = new QueryClient();
    const view = (list: TranscriptEvent[]) => (
      <QueryClientProvider client={client}><TranscriptView agentId="a1" events={list} /></QueryClientProvider>
    );
    const first = turn(1, "Final one");
    const { rerender } = render(view([...first, ...turn(10, "Final two").slice(0, 3)]));
    fireEvent.click(screen.getByRole("button", { name: /Show activity/ }));
    // The running turn is not collapsed.
    expect(screen.getByText("Working 10")).toBeInTheDocument();
    rerender(view([...first, ...turn(10, "Final two")]));
    expect(screen.getByText("Working 1")).toBeInTheDocument();
    expect(screen.queryByText("Working 10")).toBeNull();
    expect(screen.getAllByRole("button", { name: /activity/ }).map((button) => button.textContent)).toEqual(["▾ Hide activity", "▸ Show activity"]);
  });

  it("moves focus from hidden activity to its control when the turn completes", () => {
    const client = new QueryClient();
    const running = turn(1, "Final one").slice(0, 4);
    const view = (list: TranscriptEvent[]) => (
      <QueryClientProvider client={client}><TranscriptView agentId="a1" events={list} /></QueryClientProvider>
    );
    const { rerender } = render(view(running));
    const tools = screen.getByRole("button", { name: /Ran 1 tool/ });
    tools.focus();
    fireEvent.focus(tools);
    rerender(view(turn(1, "Final one")));
    expect(document.activeElement).toBe(screen.getByRole("button", { name: /Show activity/ }));
  });

  it("keeps a pending approval and a partial outcome visible and opens the turn a diff reveal targets", () => {
    const cancelled: TranscriptEvent[] = [
      { kind: "user_text", seq: 1, text: "Go" },
      { kind: "diff", seq: 2, path: "a.go", new_text: "x" },
      { kind: "permission_request", seq: 3, tool_call_id: "t1", name: "Run rm" },
      { kind: "assistant_text", seq: 4, text: "Half an answer" },
      { kind: "turn_end", seq: 5, stop_reason: "cancelled" },
    ];
    const client = new QueryClient();
    const view = (reveal: { seq: number } | null) => (
      <QueryClientProvider client={client}><TranscriptView agentId="a1" events={cancelled} reveal={reveal} /></QueryClientProvider>
    );
    const { rerender, container } = render(view(null));
    expect(screen.getByText("Run rm")).toBeInTheDocument();
    expect(screen.getByText("Cancelled — response is partial")).toBeInTheDocument();
    expect(container.querySelector('[data-seq="2"]')).toBeNull();
    rerender(view({ seq: 2 }));
    expect(container.querySelector('[data-seq="2"]')).not.toBeNull();
  });

  it("does not offer an empty control for a turn with nothing to hide", () => {
    renderTranscript(true, [
      { kind: "user_text", seq: 1, text: "Hi" },
      { kind: "assistant_text", seq: 2, text: "Hello" },
      { kind: "turn_end", seq: 3, stop_reason: "end_turn" },
    ]);
    expect(screen.queryByRole("button", { name: /activity/ })).toBeNull();
  });
});

// FS-03.R3/A68: a row whose height settles after render (a Mermaid diagram, an
// image) must not leave a reader who was at the bottom above the newest reply.
describe("TranscriptView bottom follow", () => {
  let resize: (() => void) | null = null;
  beforeEach(() => {
    resize = null;
    vi.stubGlobal("ResizeObserver", class {
      constructor(callback: () => void) { resize = callback; }
      observe() {}
      unobserve() {}
      disconnect() {}
    });
  });
  afterEach(() => vi.unstubAllGlobals());

  function geometry(list: HTMLElement, height: number) {
    const box = { height, top: 0 };
    Object.defineProperty(list, "scrollHeight", { configurable: true, get: () => box.height });
    Object.defineProperty(list, "clientHeight", { configurable: true, get: () => 100 });
    Object.defineProperty(list, "scrollTop", { configurable: true, get: () => box.top, set: (value: number) => { box.top = Math.min(value, box.height - 100); } });
    list.scrollTop = height;
    fireEvent.scroll(list);
    return box;
  }

  it("keeps the latest content in view while late rows settle", () => {
    const { container } = renderTranscript();
    const list = container.querySelector(".transcript-view") as HTMLElement;
    const box = geometry(list, 500);

    box.height = 900;
    act(() => resize?.());
    expect(list.scrollTop).toBe(800);

    // Growth the browser compensates with its own scroll (scroll anchoring)
    // is not the reader scrolling away.
    box.height = 1300;
    box.top = 850;
    fireEvent.scroll(list);
    expect(list.scrollTop).toBe(1200);
    expect(screen.queryByRole("button", { name: "Jump to latest" })).toBeNull();
  });

  it("leaves a reader who scrolled up in place until Jump to latest", () => {
    const { container } = renderTranscript();
    const list = container.querySelector(".transcript-view") as HTMLElement;
    const box = geometry(list, 500);

    list.scrollTop = 200;
    fireEvent.scroll(list);
    box.height = 900;
    act(() => resize?.());
    fireEvent.scroll(list);
    expect(list.scrollTop).toBe(200);

    fireEvent.click(screen.getByRole("button", { name: "Jump to latest" }));
    fireEvent.scroll(list);
    expect(list.scrollTop).toBe(800);
    expect(screen.queryByRole("button", { name: "Jump to latest" })).toBeNull();
  });
});

// TS-14.R26: a failed synthesis read is visible and retryable without
// disturbing provider history; Retry recovers the retained result.
describe("TranscriptView Think Tank results failure", () => {
  it("shows a retryable error and recovers", async () => {
    let fail = true;
    mswServer.use(http.get("/api/sessions/a1/think-tank-results", () => fail
      ? HttpResponse.json({ error: "down" }, { status: 500 })
      : HttpResponse.json({ version: 1, complete: true, results: [{ result_id: 1, room_id: "tt_1", room_title: "Cache choice",
          room_available: true, entry_seq: 9, attempt_id: "att", body: "Exact synthesis", generation: "g", turn_id: "t",
          event_seq: 0, completed_at: "2026-10-07T00:00:00Z" }] })));
    render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: 0 } } })}>
        <MemoryRouter><TranscriptView agentId="a1" events={events} /></MemoryRouter>
      </QueryClientProvider>,
    );
    expect(await screen.findByText("Think Tank synthesis could not be loaded.", { exact: false })).toBeInTheDocument();
    expect(screen.getByText(/First line/)).toBeInTheDocument();
    fail = false;
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(await screen.findByText("Exact synthesis")).toBeInTheDocument();
    expect(screen.queryByText("Think Tank synthesis could not be loaded.", { exact: false })).toBeNull();
  });
});
