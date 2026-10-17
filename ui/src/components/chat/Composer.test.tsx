import React from "react";
import { describe, it, expect, beforeAll, afterAll, afterEach, vi } from "vitest";
import { render, screen, fireEvent, waitFor, cleanup } from "@testing-library/react";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { Composer } from "./Composer";
import { getChatDraft } from "./drafts";
import { useHeldStore } from "../../store/heldStore";
import { withdrawHeldMessage } from "../../lib/heldMessage";
import type { AttachmentDescriptor } from "../../api/types";
import { useTranscriptStore } from "../../store/transcriptStore";
import { clearAttachmentDraftCacheForTests } from "./attachments";

// Files returned by the mock file-search, narrowed by the q param so the test can
// assert query filtering (FS-03.R30/A15).
const filesFor = (q: string) => {
  const all = ["src/app.ts", "src/util.ts", "README.md"];
  return q ? all.filter((f) => f.toLowerCase().includes(q.toLowerCase())) : all;
};

let promptBodies: string[] = [];
let attachmentRequests: string[] = [];
let failFileSearch = false;
let resolvePrompt: ((response: Response) => void) | null = null;

const server = setupServer(
  http.get("/api/sessions/:id/file-search", ({ request }) => {
    if (failFileSearch) return new HttpResponse(null, { status: 409 });
    const q = new URL(request.url).searchParams.get("q") ?? "";
    return HttpResponse.json({ agent_id: "a_1", files: filesFor(q) });
  }),
  http.get("/api/sessions/:id/available-commands", () =>
    HttpResponse.json({
      agent_id: "a_1",
      commands: [
        { name: "review", description: "Review code", input_hint: "branch" },
        { name: "$plan", description: "Plan work" },
      ],
    }),
  ),
  http.post("/api/sessions/:id/prompt", async ({ request }) => {
    if (request.headers.get("content-type")?.includes("multipart/form-data")) {
      attachmentRequests.push(request.headers.get("content-type") ?? "");
    } else promptBodies.push(await request.text());
    return HttpResponse.json({}, { status: 202 });
  }),
);

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => {
  cleanup();
  localStorage.clear();
  clearAttachmentDraftCacheForTests();
  useHeldStore.setState({ byAgent: {}, afterSeqByAgent: {}, attachmentsByAgent: {}, recoveryByAgent: {}, recoveryUnreadByAgent: {} });
  useTranscriptStore.setState({ byAgent: {}, rawByAgent: {}, pending: {} });
  server.resetHandlers();
  promptBodies = [];
  attachmentRequests = [];
  failFileSearch = false;
  resolvePrompt = null;
  vi.restoreAllMocks();
});
afterAll(() => server.close());

// type sets the textarea value and caret (jsdom moves the caret to the end on a
// value set, but we pass it explicitly so detectTrigger reads the real cursor).
function type(el: HTMLTextAreaElement, value: string) {
  fireEvent.change(el, { target: { value, selectionStart: value.length, selectionEnd: value.length } });
}

describe("Composer autocomplete", () => {
  // FS-03.A64/A65: legacy unknown input reaches wake; a real refusal keeps the draft.
  it.each([undefined, true, false])("honors image advertisement %s without inventing a refusal", async (image_input) => {
    render(<Composer agentId="a_1" busy={false} attachmentCapabilities={{ image_input, files: true, max_files: 10, max_file_bytes: 10485760, max_total_bytes: 26214400, max_pixels: 40000000 }} />);
    fireEvent.change(screen.getByLabelText("Choose files to attach"), { target: { files: [new File(["image"], "wake.png", { type: "image/png" })] } });
    const send = screen.getByRole("button", { name: "Send" });
    if (image_input === false) {
      expect(send).toBeDisabled();
      expect(screen.getByText(/This runtime does not accept image input/)).toBeInTheDocument();
      expect(screen.getByText("wake.png")).toBeInTheDocument();
      fireEvent.click(send);
      expect(attachmentRequests).toHaveLength(0);
    } else {
      expect(send).toBeEnabled();
      fireEvent.click(send);
      await waitFor(() => expect(attachmentRequests).toHaveLength(1));
    }
  });

  it("sends selected files and text together in declared order, including attachment-only sends", async () => {
    render(<Composer agentId="a_1" busy={false} />);
    const picker = screen.getByLabelText("Choose files to attach");
    fireEvent.change(picker, { target: { files: [new File(["first"], "one.png", { type: "image/png" }), new File(["second"], "two.txt", { type: "text/plain" })] } });
    type(screen.getByRole("textbox") as HTMLTextAreaElement, "look at these");
    fireEvent.click(screen.getByRole("button", { name: "Send" }));

    await waitFor(() => expect(attachmentRequests).toHaveLength(1));
    expect(attachmentRequests[0]).toContain("multipart/form-data; boundary=");
  });

  it("allows an image-only attachment message", async () => {
    render(<Composer agentId="a_1" busy={false} />);
    fireEvent.change(screen.getByLabelText("Choose files to attach"), { target: { files: [new File(["image"], "image.png", { type: "image/png" })] } });
    fireEvent.click(screen.getByRole("button", { name: "Send" }));
    await waitFor(() => expect(attachmentRequests).toHaveLength(1));
    expect(attachmentRequests[0]).toContain("multipart/form-data; boundary=");
  });

  it("keeps one live user bubble when an attached send replies before acceptance returns", async () => {
    server.use(http.post("/api/sessions/:id/prompt", () => {
      const store = useTranscriptStore.getState();
      store.appendMessage("a_1", { agent_id: "a_1", seq: 7, type: "user_text", ts: "t7", data: { text: "read this", request_id: "file-1" } });
      store.appendMessage("a_1", { agent_id: "a_1", seq: 8, type: "assistant_text", ts: "t8", data: { delta: "Done." } });
      store.appendMessage("a_1", { agent_id: "a_1", seq: 9, type: "turn_end", ts: "t9", data: { stop_reason: "end_turn" } });
      return HttpResponse.json({ accepted: true, agent_id: "a_1", delivery: "sent", after_seq: 0,
        request_id: "file-1", attachment_delivery: "submitted", attachments: [] }, { status: 202 });
    }));
    render(<Composer agentId="a_1" busy={false} />);
    fireEvent.change(screen.getByLabelText("Choose files to attach"), { target: { files: [new File(["file"], "one.txt", { type: "text/plain" })] } });
    type(screen.getByRole("textbox") as HTMLTextAreaElement, "read this");
    await waitFor(() => expect(screen.getByRole("button", { name: "Send" })).toBeEnabled());
    fireEvent.click(screen.getByRole("button", { name: "Send" }));

    await waitFor(() => expect(useTranscriptStore.getState().byAgent.a_1?.some((event) => event.kind === "turn_end")).toBe(true));
    await waitFor(() => expect((screen.getByRole("textbox") as HTMLTextAreaElement).value).toBe(""));
    expect(useTranscriptStore.getState().byAgent.a_1.filter((event) => event.kind === "user_text"))
      .toEqual([expect.objectContaining({ seq: 7, text: "read this" })]);
  });

  // FS-03.A19: browser-local drafts follow the selected chat through navigation
  // and remounts, but never cross to another agent.
  it("restores distinct drafts for their matching chat after navigation and remount", async () => {
    const view = render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "first draft");

    view.rerender(<Composer agentId="a_2" busy={false} />);
    await waitFor(() => expect(ta.value).toBe(""));
    type(ta, "second draft");

    view.rerender(<Composer agentId="a_1" busy={false} />);
    await waitFor(() => expect(ta.value).toBe("first draft"));
    view.unmount();

    render(<Composer agentId="a_2" busy={false} />);
    expect((screen.getByRole("textbox") as HTMLTextAreaElement).value).toBe("second draft");
  });

  it("clears the stored draft only after an accepted prompt or manual emptying", async () => {
    const view = render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "send this");
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(promptBodies).toHaveLength(1));
    view.unmount();

    render(<Composer agentId="a_1" busy={false} />);
    expect((screen.getByRole("textbox") as HTMLTextAreaElement).value).toBe("");
    type(screen.getByRole("textbox") as HTMLTextAreaElement, "remove this");
    type(screen.getByRole("textbox") as HTMLTextAreaElement, "");
    cleanup();

    render(<Composer agentId="a_1" busy={false} />);
    expect((screen.getByRole("textbox") as HTMLTextAreaElement).value).toBe("");
  });

  it("keeps the live composer usable when browser storage is unavailable", async () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new Error("storage unavailable"); });
    render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "send without storage");
    fireEvent.keyDown(ta, { key: "Enter" });

    await waitFor(() => expect(promptBodies).toHaveLength(1));
    expect(JSON.parse(promptBodies[0]).text).toBe("send without storage");
  });

  it("opens the file picker on `@` at a word boundary but not inside a word", async () => {
    render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;

    type(ta, "look at @src");
    expect(await screen.findByText("src/app.ts")).toBeInTheDocument();
    expect(screen.getByText("src/util.ts")).toBeInTheDocument();

    // `@` inside a word (email) must not open the picker.
    type(ta, "mail user@example.com");
    await waitFor(() => expect(screen.queryByRole("listbox")).not.toBeInTheDocument());
  });

  it("filters, navigates with arrows, and accepts a file inserting `@<path> `", async () => {
    render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;

    type(ta, "@app");
    expect(await screen.findByText("src/app.ts")).toBeInTheDocument();
    expect(screen.queryByText("src/util.ts")).not.toBeInTheDocument();

    type(ta, "@src");
    await screen.findByText("src/util.ts");
    // Highlight starts at 0 (app.ts); ArrowDown moves to util.ts, Enter accepts it.
    fireEvent.keyDown(ta, { key: "ArrowDown" });
    fireEvent.keyDown(ta, { key: "Enter" });

    await waitFor(() => expect(ta.value).toBe("@src/util.ts "));
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });

  it("dismisses the picker on Escape", async () => {
    render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "@src");
    await screen.findByText("src/app.ts");
    fireEvent.keyDown(ta, { key: "Escape" });
    await waitFor(() => expect(screen.queryByRole("listbox")).not.toBeInTheDocument());
  });

  it("submits the unchanged prompt when the picker is closed", async () => {
    render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "hello there");
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(promptBodies.length).toBe(1));
    expect(JSON.parse(promptBodies[0]).text).toBe("hello there");
    expect(ta.value).toBe("");
  });

  it("opens the command picker on `#` and inserts `/<name> ` including `/$skill`", async () => {
    render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;

    type(ta, "#");
    expect(await screen.findByText("/review")).toBeInTheDocument();
    expect(screen.getByText("/$plan")).toBeInTheDocument();

    // Filter to the codex-style skill and accept it.
    type(ta, "#plan");
    await screen.findByText("/$plan");
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(ta.value).toBe("/$plan "));
  });

  it("submits selected file and command text exactly as displayed, keeping the trailing space", async () => {
    render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;

    // Accept a file: inserts `@src/app.ts ` (with trailing space).
    type(ta, "@app");
    await screen.findByText("src/app.ts");
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(ta.value).toBe("@src/app.ts "));

    // Continue and accept a command: inserts `/$plan ` (with trailing space).
    type(ta, "@src/app.ts #plan");
    await screen.findByText("/$plan");
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(ta.value).toBe("@src/app.ts /$plan "));

    // The picker is closed after accepting, so Enter submits. The request body must
    // retain the inserted trailing space (FS-03.R31/R33, INV §1).
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(promptBodies.length).toBe(1));
    expect(JSON.parse(promptBodies[0]).text).toBe("@src/app.ts /$plan ");
  });

  // FS-03.R35/A18 — the composer of a stopped chat agent is the wake surface: it
  // submits the ordinary prompt request, and a rejected wake shows the server's
  // own error while keeping the draft.
  it("submits the ordinary prompt request for a stopped agent", async () => {
    render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "wake up");
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(promptBodies.length).toBe(1));
    expect(JSON.parse(promptBodies[0]).text).toBe("wake up");
    expect(ta.value).toBe("");
  });

  it("shows the server error and restores the draft when the wake is rejected", async () => {
    server.use(
      http.post("/api/sessions/:id/prompt", () =>
        HttpResponse.json(
          { error: { code: "agent_archived", message: "agent is archived; restore it before resuming" } },
          { status: 409 },
        ),
      ),
    );
    render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "wake up");
    fireEvent.keyDown(ta, { key: "Enter" });

    expect(await screen.findByText(/agent is archived; restore it before resuming/)).toBeInTheDocument();
    await waitFor(() => expect(ta.value).toBe("wake up"));
    cleanup();

    render(<Composer agentId="a_1" busy={false} />);
    expect((screen.getByRole("textbox") as HTMLTextAreaElement).value).toBe("wake up");
  });

  it("does not restore a rejected send into a different chat opened while it was pending", async () => {
    server.use(
      http.post("/api/sessions/:id/prompt", () => new Promise<Response>((resolve) => { resolvePrompt = resolve; })),
    );
    const view = render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "first draft");
    fireEvent.keyDown(ta, { key: "Enter" });

    view.rerender(<Composer agentId="a_2" busy={false} />);
    await waitFor(() => expect(ta.value).toBe(""));
    type(ta, "second draft");
    resolvePrompt?.(HttpResponse.json({ error: { code: "agent_archived", message: "agent is archived" } }, { status: 409 }));

    await waitFor(() => expect(ta.value).toBe("second draft"));
    view.unmount();
    render(<Composer agentId="a_1" busy={false} />);
    expect((screen.getByRole("textbox") as HTMLTextAreaElement).value).toBe("first draft");
  });

  // INV §1/§5 — a delayed send's completion is scoped to the draft generation it
  // submitted, so a newer same-agent draft typed while the request is in flight
  // is never discarded on success nor overwritten on failure.
  it("keeps a newer same-agent draft typed while a successful send is in flight", async () => {
    server.use(
      http.post("/api/sessions/:id/prompt", () => new Promise<Response>((resolve) => { resolvePrompt = resolve; })),
    );
    const view = render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "message A");
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(ta.value).toBe(""));

    // The person starts a new draft for the same chat before A's send resolves.
    type(ta, "message B");
    resolvePrompt?.(HttpResponse.json({}, { status: 202 }));
    await new Promise((resolve) => setTimeout(resolve, 0));

    // A's accepted send must not discard the newer draft B.
    expect(ta.value).toBe("message B");
    view.unmount();
    render(<Composer agentId="a_1" busy={false} />);
    expect((screen.getByRole("textbox") as HTMLTextAreaElement).value).toBe("message B");
  });

  it("keeps a newer same-agent draft typed while a rejected send is in flight", async () => {
    server.use(
      http.post("/api/sessions/:id/prompt", () => new Promise<Response>((resolve) => { resolvePrompt = resolve; })),
    );
    const view = render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "message A");
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(ta.value).toBe(""));

    type(ta, "message B");
    resolvePrompt?.(HttpResponse.json({ error: { code: "agent_archived", message: "agent is archived" } }, { status: 409 }));
    await new Promise((resolve) => setTimeout(resolve, 0));

    // A's rejection must not overwrite the newer draft B with the sent text.
    expect(ta.value).toBe("message B");
    view.unmount();
    render(<Composer agentId="a_1" busy={false} />);
    expect((screen.getByRole("textbox") as HTMLTextAreaElement).value).toBe("message B");
  });

  it("stays usable when the file source is unavailable", async () => {
    failFileSearch = true;
    render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;

    type(ta, "@src");
    // Empty/unavailable state renders and never blocks submission.
    expect(await screen.findByText("No matching files")).toBeInTheDocument();

    type(ta, "send this");
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(promptBodies.length).toBe(1));
    expect(JSON.parse(promptBodies[0]).text).toBe("send this");
  });
});

// FS-03.A31/A32/A33 — the composer half of the two deliberate actions on a busy
// agent: Send queues and is withdrawable, Steer delivers into the running turn
// and only where the live session advertises it, and a lost agent returns the
// held text without ever overwriting text typed since.
describe("Composer queued follow-up and steering", () => {
  it("holds a message sent to a busy agent instead of echoing it into the transcript", async () => {
    server.use(http.post("/api/sessions/:id/prompt", async ({ request }) => {
      promptBodies.push(await request.text());
      return HttpResponse.json({ accepted: true, agent_id: "a_1", delivery: "held" }, { status: 202 });
    }));
    render(<Composer agentId="a_1" busy />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;

    type(ta, "also check the tests");
    fireEvent.keyDown(ta, { key: "Enter" });

    await waitFor(() => expect(useHeldStore.getState().byAgent.a_1).toBe("also check the tests"));
    expect(ta.value).toBe("");
    // Send stays present on a busy agent; Cancel is still offered beside it.
    expect(screen.getByRole("button", { name: "Send" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    // FS-02.A49 — both carry the icon and a tooltip repeating the name; on the full
    // agent page Send also shows its text (FS-12.R61) while Cancel stays an icon.
    for (const name of ["Send", "Cancel"]) {
      const button = screen.getByRole("button", { name });
      expect(button).toHaveAttribute("title", name);
      expect(button.querySelector("svg")).not.toBeNull();
      expect(button.textContent).toBe(name === "Send" ? "Send" : "");
    }
  });

  it("keeps the dashboard composer's Send icon-only and without insert buttons", () => {
    render(<Composer agentId="a_dash" busy={false} variant="dashboard" />);
    expect(screen.getByRole("button", { name: "Send" }).textContent).toBe("");
    expect(screen.queryByRole("button", { name: "Reference a file" })).toBeNull();
    expect(screen.getByRole("textbox")).not.toHaveAttribute("placeholder");
  });

  // FS-12.R61 — @ and # type the existing trigger at the caret and open the
  // shared picker; a word right before the caret gets a separating space.
  it("inserts the picker trigger from the toolbar", async () => {
    render(<Composer agentId="a_insert" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "see");
    ta.setSelectionRange(3, 3);
    fireEvent.click(screen.getByRole("button", { name: "Reference a file" }));
    await waitFor(() => expect(ta.value).toBe("see @"));
    await waitFor(() => expect(document.activeElement).toBe(ta));
    expect(ta.selectionStart).toBe(5);
    fireEvent.click(screen.getByRole("button", { name: "Insert a command" }));
    await waitFor(() => expect(ta.value).toBe("see @ #"));
  });

  it("does not echo a message when an idle-looking submit is held by the server", async () => {
    server.use(http.post("/api/sessions/:id/prompt", () =>
      HttpResponse.json({ accepted: true, agent_id: "a_1", delivery: "held", after_seq: 8 }, { status: 202 })));
    render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;

    type(ta, "raced with another turn");
    fireEvent.keyDown(ta, { key: "Enter" });

    await waitFor(() => expect(useHeldStore.getState().byAgent.a_1).toBe("raced with another turn"));
    expect(useTranscriptStore.getState().byAgent.a_1 ?? []).toHaveLength(0);
  });

  // The server's sequenced event can win the race with the prompt response; the
  // durable event must remain the single rendered user message.
  it("keeps one user message when its sequenced event arrives before the prompt response", async () => {
    server.use(http.post("/api/sessions/:id/prompt", async ({ request }) => {
      const text = String((await request.json() as { text: string }).text);
      useTranscriptStore.getState().appendMessage("a_1", {
        agent_id: "a_1", seq: 7, type: "user_text", ts: "t7", data: { text },
      });
      return HttpResponse.json({ accepted: true, agent_id: "a_1", delivery: "sent", after_seq: 7 }, { status: 202 });
    }));
    render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;

    type(ta, "wait before answering");
    fireEvent.keyDown(ta, { key: "Enter" });

    await waitFor(() => {
      const events = useTranscriptStore.getState().byAgent.a_1;
      expect(events).toHaveLength(1);
      expect(events[0]).toMatchObject({ kind: "user_text", seq: 7, text: "wait before answering" });
    });
  });

  it("keeps one live user bubble when the reply ends before the prompt response", async () => {
    server.use(http.post("/api/sessions/:id/prompt", async ({ request }) => {
      const text = String((await request.json() as { text: string }).text);
      const store = useTranscriptStore.getState();
      store.appendMessage("a_1", { agent_id: "a_1", seq: 7, type: "user_text", ts: "t7", data: { text } });
      store.appendMessage("a_1", { agent_id: "a_1", seq: 8, type: "assistant_text", ts: "t8", data: { delta: "Done." } });
      store.appendMessage("a_1", { agent_id: "a_1", seq: 9, type: "turn_end", ts: "t9", data: { stop_reason: "end_turn" } });
      return HttpResponse.json({ accepted: true, agent_id: "a_1", delivery: "sent", after_seq: 0 }, { status: 202 });
    }));
    render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;

    type(ta, "finish before acknowledgement");
    fireEvent.keyDown(ta, { key: "Enter" });

    await waitFor(() => expect(ta.value).toBe(""));
    expect(useTranscriptStore.getState().byAgent.a_1.filter((event) => event.kind === "user_text"))
      .toEqual([expect.objectContaining({ seq: 7, text: "finish before acknowledgement" })]);
  });

  it("keeps a second identical prompt when it starts after the first turn", async () => {
    let turn = 0;
    server.use(http.post("/api/sessions/:id/prompt", async ({ request }) => {
      const text = String((await request.json() as { text: string }).text);
      const seq = 7 + turn++ * 3;
      const store = useTranscriptStore.getState();
      store.appendMessage("a_1", { agent_id: "a_1", seq, type: "user_text", ts: `t${seq}`, data: { text } });
      store.appendMessage("a_1", { agent_id: "a_1", seq: seq + 1, type: "assistant_text", ts: `t${seq + 1}`, data: { delta: "Done." } });
      store.appendMessage("a_1", { agent_id: "a_1", seq: seq + 2, type: "turn_end", ts: `t${seq + 2}`, data: { stop_reason: "end_turn" } });
      return HttpResponse.json({ accepted: true, agent_id: "a_1", delivery: "sent", after_seq: 0 }, { status: 202 });
    }));
    render(<Composer agentId="a_1" busy={false} />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "repeat");
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(useTranscriptStore.getState().byAgent.a_1?.filter((event) => event.kind === "turn_end")).toHaveLength(1));
    await waitFor(() => expect(screen.getByRole("button", { name: "Send" })).toBeDisabled());
    type(ta, "repeat");
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(useTranscriptStore.getState().byAgent.a_1?.filter((event) => event.kind === "turn_end")).toHaveLength(2));
    expect(useTranscriptStore.getState().byAgent.a_1.filter((event) => event.kind === "user_text").map((event) => event.seq)).toEqual([7, 10]);
  });

  it("withdraws the held message through the prompt resource", async () => {
    let withdrawn = 0;
    server.use(
      http.post("/api/sessions/:id/prompt", () =>
        HttpResponse.json({ accepted: true, agent_id: "a_1", delivery: "held" }, { status: 202 })),
      http.delete("/api/sessions/:id/prompt", () => {
        withdrawn++;
        return HttpResponse.json({ accepted: true, agent_id: "a_1", delivery: "withdrawn" });
      }),
    );
    render(<Composer agentId="a_1" busy />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "never mind");
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(useHeldStore.getState().byAgent.a_1).toBe("never mind"));

    fireEvent.click(screen.getByRole("button", { name: "Withdraw queued" }));
    await waitFor(() => expect(useHeldStore.getState().byAgent.a_1).toBeUndefined());
    expect(withdrawn).toBe(1);
    // FS-03.A70 — the withdrawn text comes back for editing.
    await waitFor(() => expect(ta.value).toBe("never mind"));
  });

  it("puts the withdrawn text ahead of a draft typed since", async () => {
    server.use(
      http.post("/api/sessions/:id/prompt", () =>
        HttpResponse.json({ accepted: true, agent_id: "a_1", delivery: "held" }, { status: 202 })),
      http.delete("/api/sessions/:id/prompt", () => HttpResponse.json({ accepted: true, agent_id: "a_1" })),
    );
    render(<Composer agentId="a_1" busy />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "first thought");
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(useHeldStore.getState().byAgent.a_1).toBe("first thought"));
    type(ta, "second thought");

    fireEvent.click(screen.getByRole("button", { name: "Withdraw queued" }));
    await waitFor(() => expect(ta.value).toBe("first thought\n\nsecond thought"));
  });

  it("turns a withdrawn attachment hold into the recovery offer instead of a draft", async () => {
    const attachment: AttachmentDescriptor = { id: "att_1", name: "shot.png", kind: "image", mime_type: "image/png", size: 4, sha256: "abcd" };
    const recovery = { text: "see image", request_id: "req_1", attachments: [attachment] };
    server.use(
      http.delete("/api/sessions/:id/prompt", () => HttpResponse.json({ accepted: true, agent_id: "a_1" })),
      http.get("/api/sessions/:id/prompt", () => HttpResponse.json({ agent_id: "a_1", text: "", after_seq: 0, recovery })),
    );
    useHeldStore.getState().hold("a_1", "see image", 0, [attachment]);
    render(<Composer agentId="a_1" busy />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;

    fireEvent.click(screen.getByRole("button", { name: "Withdraw queued" }));
    await waitFor(() => expect(useHeldStore.getState().recoveryByAgent.a_1).toEqual(recovery));
    expect(ta.value).toBe("");
  });

  it("keeps newer typed text when browser storage refuses to save it", async () => {
    server.use(
      http.post("/api/sessions/:id/prompt", () =>
        HttpResponse.json({ accepted: true, agent_id: "a_1", delivery: "held" }, { status: 202 })),
      http.delete("/api/sessions/:id/prompt", () => HttpResponse.json({ accepted: true, agent_id: "a_1" })),
    );
    render(<Composer agentId="a_1" busy />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "first thought");
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(useHeldStore.getState().byAgent.a_1).toBe("first thought"));
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new DOMException("full", "QuotaExceededError"); });
    type(ta, "unsaved thought");

    fireEvent.click(screen.getByRole("button", { name: "Withdraw queued" }));
    await waitFor(() => expect(ta.value).toBe("first thought\n\nunsaved thought"));
  });

  it("reports a recovery read that fails after the withdrawal succeeded", async () => {
    const attachment: AttachmentDescriptor = { id: "att_1", name: "shot.png", kind: "image", mime_type: "image/png", size: 4, sha256: "abcd" };
    server.use(
      http.delete("/api/sessions/:id/prompt", () => HttpResponse.json({ accepted: true, agent_id: "a_1" })),
      http.get("/api/sessions/:id/prompt", () => HttpResponse.json({ error: "down" }, { status: 503 })),
    );
    useHeldStore.getState().hold("a_1", "see image", 0, [attachment]);
    render(<Composer agentId="a_1" busy />);

    fireEvent.click(screen.getByRole("button", { name: "Withdraw queued" }));
    await waitFor(() => expect(useHeldStore.getState().recoveryUnreadByAgent.a_1).toBe(true));
    expect(useHeldStore.getState().byAgent.a_1).toBeUndefined();
    expect(screen.queryByText(/Could not withdraw/)).toBeNull();
  });

  it("restores the held text once when withdrawals overlap across both controls", async () => {
    let withdrawn = 0;
    let finish: (() => void) | null = null;
    server.use(http.delete("/api/sessions/:id/prompt", async () => {
      withdrawn++;
      await new Promise<void>((resolve) => { finish = resolve; });
      return HttpResponse.json({ accepted: true, agent_id: "a_1" });
    }));
    useHeldStore.getState().hold("a_1", "held text");
    render(<Composer agentId="a_1" busy />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;

    fireEvent.click(screen.getByRole("button", { name: "Withdraw queued" }));
    fireEvent.click(screen.getByRole("button", { name: "Withdraw queued" }));
    // The pending tail's Withdraw uses the same helper.
    const second = withdrawHeldMessage("a_1");
    await waitFor(() => expect(finish).not.toBeNull());
    finish!();
    await second;
    await waitFor(() => expect(ta.value).toBe("held text"));
    expect(withdrawn).toBe(1);
  });

  it("leaves the composer unchanged when the withdrawal is refused", async () => {
    server.use(
      http.post("/api/sessions/:id/prompt", () =>
        HttpResponse.json({ accepted: true, agent_id: "a_1", delivery: "held" }, { status: 202 })),
      http.delete("/api/sessions/:id/prompt", () => HttpResponse.json({ error: "gone" }, { status: 409 })),
    );
    render(<Composer agentId="a_1" busy />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "keep queued");
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(useHeldStore.getState().byAgent.a_1).toBe("keep queued"));

    fireEvent.click(screen.getByRole("button", { name: "Withdraw queued" }));
    expect(await screen.findByText(/Could not withdraw/)).toBeInTheDocument();
    expect(ta.value).toBe("");
    expect(useHeldStore.getState().byAgent.a_1).toBe("keep queued");
  });

  it("offers Steer only where the live session advertises it", () => {
    const view = render(<Composer agentId="a_1" busy steerable={false} />);
    expect(screen.queryByRole("button", { name: "Steer" })).not.toBeInTheDocument();
    view.unmount();

    render(<Composer agentId="a_1" busy steerable />);
    expect(screen.getByRole("button", { name: "Steer" })).toBeInTheDocument();
  });

  it("reports which of the two things the adapter did with a steered message", async () => {
    let outcome = "steered";
    server.use(http.post("/api/sessions/:id/steer", () =>
      HttpResponse.json({ accepted: true, agent_id: "a_1", outcome }, { status: 202 })));

    const view = render(<Composer agentId="a_1" busy steerable />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "use the other file");
    fireEvent.click(screen.getByRole("button", { name: "Steer" }));
    expect(await screen.findByText("Delivered into the running turn.")).toBeInTheDocument();
    expect(ta.value).toBe("");
    view.unmount();

    outcome = "new_turn";
    render(<Composer agentId="a_1" busy steerable />);
    type(screen.getByRole("textbox") as HTMLTextAreaElement, "too late");
    fireEvent.click(screen.getByRole("button", { name: "Steer" }));
    expect(await screen.findByText("That turn had already ended — sent as a new turn.")).toBeInTheDocument();
  });

  it("submits one steer while the first request is still in flight", async () => {
    let steerRequests = 0;
    let finishSteer: (() => void) | undefined;
    server.use(http.post("/api/sessions/:id/steer", async () => {
      steerRequests++;
      await new Promise<void>((resolve) => { finishSteer = resolve; });
      return HttpResponse.json({ accepted: true, agent_id: "a_1", outcome: "steered" }, { status: 202 });
    }));

    render(<Composer agentId="a_1" busy steerable />);
    type(screen.getByRole("textbox") as HTMLTextAreaElement, "use the other file");
    const steerButton = screen.getByRole("button", { name: "Steer" });
    fireEvent.click(steerButton);
    fireEvent.click(steerButton);

    await waitFor(() => expect(steerRequests).toBe(1));
    expect(steerButton).toBeDisabled();
    finishSteer?.();
    await waitFor(() => expect(steerButton).toBeEnabled());
  });

  it("steers the held message when the composer is empty, and keeps typed text on a refusal", async () => {
    let steerBodies: string[] = [];
    server.use(
      http.post("/api/sessions/:id/prompt", () =>
        HttpResponse.json({ accepted: true, agent_id: "a_1", delivery: "held" }, { status: 202 })),
      http.post("/api/sessions/:id/steer", async ({ request }) => {
        const body = await request.text();
        steerBodies.push(body);
        if (JSON.parse(body).text !== "") {
          return HttpResponse.json({ error: { code: "internal", message: "the model cannot accept that" } }, { status: 500 });
        }
        return HttpResponse.json({ accepted: true, agent_id: "a_1", outcome: "steered" }, { status: 202 });
      }),
    );
    render(<Composer agentId="a_1" busy steerable />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;

    type(ta, "queued first");
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(useHeldStore.getState().byAgent.a_1).toBe("queued first"));

    // Empty composer + a held message: Steer delivers the held one.
    fireEvent.click(screen.getByRole("button", { name: "Steer" }));
    await waitFor(() => expect(useHeldStore.getState().byAgent.a_1).toBeUndefined());
    expect(JSON.parse(steerBodies[0]).text).toBe("");

    // A refused steer leaves the composer's text exactly where it was.
    type(ta, "not acceptable");
    fireEvent.click(screen.getByRole("button", { name: "Steer" }));
    await waitFor(() => expect(screen.getByText(/Could not steer/)).toBeInTheDocument());
    expect(ta.value).toBe("not acceptable");
  });

  it("returns a held message to an empty composer when the agent stops, and discards it over typed text", async () => {
    server.use(http.post("/api/sessions/:id/prompt", () =>
      HttpResponse.json({ accepted: true, agent_id: "a_1", delivery: "held" }, { status: 202 })));

    const view = render(<Composer agentId="a_1" busy running />);
    const ta = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta, "held text");
    fireEvent.keyDown(ta, { key: "Enter" });
    await waitFor(() => expect(useHeldStore.getState().byAgent.a_1).toBe("held text"));

    view.rerender(<Composer agentId="a_1" busy={false} running={false} />);
    await waitFor(() => expect((screen.getByRole("textbox") as HTMLTextAreaElement).value).toBe("held text"));
    expect(useHeldStore.getState().byAgent.a_1).toBeUndefined();
    expect(getChatDraft("a_1")).toBe("held text");
    view.unmount();

    // Same stop, but the person has since typed: their text wins and the held
    // message is discarded rather than overwriting it.
    const second = render(<Composer agentId="a_2" busy running />);
    const ta2 = screen.getByRole("textbox") as HTMLTextAreaElement;
    type(ta2, "queue me");
    fireEvent.keyDown(ta2, { key: "Enter" });
    await waitFor(() => expect(useHeldStore.getState().byAgent.a_2).toBe("queue me"));
    type(ta2, "typed since");

    second.rerender(<Composer agentId="a_2" busy={false} running={false} />);
    await waitFor(() => expect(useHeldStore.getState().byAgent.a_2).toBeUndefined());
    expect((screen.getByRole("textbox") as HTMLTextAreaElement).value).toBe("typed since");
    expect(getChatDraft("a_2")).toBe("typed since");
  });
});
