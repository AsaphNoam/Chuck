import React from "react";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { useTranscriptStore } from "../../store/transcriptStore";
import { ArchiveAgentPage } from "./ArchiveAgentPage";

const server = setupServer(
  http.get("/api/sessions/:id/think-tank-results", () => HttpResponse.json({
    version: 1, complete: true, results: [],
  })),
  http.get("/api/sessions/a_archive/transcript", ({ request }) => {
    expect(new URL(request.url).searchParams.get("include_meta")).toBe("true");
    return HttpResponse.json({
      agent_id: "a_archive",
      events: [
        { seq: 1, type: "session_meta", ts: "t1", data: { name: "Atlas", project: "chuck", backend: "codex", model: "gpt-5.6-sol", interface: "chat", created_at: "2026-07-24T12:30:00Z" } },
        { seq: 2, type: "assistant_text", ts: "t2", data: { delta: "Sure, " } },
        { seq: 3, type: "assistant_text", ts: "t3", data: { delta: "I'll " } },
        { seq: 4, type: "assistant_text", ts: "t4", data: { delta: "do that." } },
      ],
    });
  }),
);

function renderArchive(id: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/archive/${id}`]}>
        <Routes>
          <Route path="/archive/:id" element={<ArchiveAgentPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
beforeEach(() => useTranscriptStore.setState({ byAgent: {}, pending: {}, settled: {}, appliedSeq: {} }));
afterEach(() => {
  cleanup();
  server.resetHandlers();
});
afterAll(() => server.close());

// FS-05.R31/A15: a switched or resumed session appends newer metadata, and the
// read-only header must show the identity Resume will actually restore.
describe("ArchiveAgentPage switched session identity", () => {
  it("shows the newest recorded backend/model, not the original", async () => {
    server.use(http.get("/api/sessions/a_switched/transcript", () => HttpResponse.json({
      agent_id: "a_switched",
      events: [
        { seq: 1, type: "session_meta", ts: "t1", data: { name: "Atlas", project: "chuck", backend: "codex", model: "gpt-5.6-sol", interface: "chat", created_at: "2026-07-24T12:30:00Z" } },
        { seq: 2, type: "assistant_text", ts: "t2", data: { delta: "working" } },
        { seq: 3, type: "session_meta", ts: "t3", data: { name: "Atlas", project: "chuck", backend: "claude", model: "sonnet", interface: "chat", created_at: "2026-07-24T12:30:00Z", resumed_at: "2026-07-25T09:00:00Z" } },
      ],
    })));

    renderArchive("a_switched");

    await waitFor(() => expect(document.querySelector(".agent-heading-details")).toHaveTextContent(/^chuck · claude · sonnet · Normal speed$/));
    expect(document.querySelector(".agent-heading-details")).not.toHaveTextContent("codex");
    expect(screen.getByRole("heading", { name: "Atlas" })).toBeInTheDocument();
    expect(document.querySelector("time")?.getAttribute("datetime")).toBe("2026-07-24T12:30:00Z");
  });
});

describe("ArchiveAgentPage", () => {
  it("renders a stored assistant stream as one message", async () => {
    renderArchive("a_archive");

    expect(await screen.findByText("Sure, I'll do that.")).toBeInTheDocument();
    expect(document.querySelectorAll("article.assistant-message")).toHaveLength(1);
    expect(screen.getByRole("heading", { name: "Atlas" })).toBeInTheDocument();
    expect(document.querySelector(".agent-heading-details")).toHaveTextContent(/^chuck · codex · gpt-5\.6-sol · Normal speed$/);
    expect(screen.getByRole("link", { name: "Back to Archive" })).toHaveAttribute("href", "/archive");
    expect(screen.getByText(/Archived · read-only/)).toBeInTheDocument();
    expect(document.querySelector("time")?.getAttribute("datetime")).toBe("2026-07-24T12:30:00Z");
  });
});

// FS-21.A37, TS-14.R26: an archived judge shows its retained synthesis once,
// with its exact body and a room link, before the completing turn's end —
// not the incidental provider reply.
describe("ArchiveAgentPage judge result", () => {
  it("renders the retained synthesis at its completion anchor", async () => {
    server.use(
      http.get("/api/sessions/a_judge/transcript", () => HttpResponse.json({
        agent_id: "a_judge",
        events: [
          { seq: 1, type: "session_meta", ts: "t1", data: { name: "Judge", project: "chuck", backend: "codex", model: "m", interface: "chat", created_at: "2026-07-24T12:30:00Z" } },
          { seq: 2, type: "assistant_text", ts: "t2", data: { delta: "Submitted." } },
          { seq: 3, type: "turn_end", ts: "t3", data: { stop_reason: "end_turn" } },
        ],
      })),
      http.get("/api/sessions/a_judge/think-tank-results", () => HttpResponse.json({
        version: 1, complete: true,
        results: [{ result_id: 1, room_id: "tt_1", room_title: "Cache choice", room_available: true, entry_seq: 9,
          attempt_id: "att", body: "Exact synthesis: keep LRU", generation: "g", turn_id: "t", event_seq: 3, completed_at: "2026-10-07T00:00:00Z" }],
      })),
    );
    renderArchive("a_judge");
    expect(await screen.findByText("Exact synthesis: keep LRU")).toBeInTheDocument();
    expect(screen.getAllByText("Exact synthesis: keep LRU")).toHaveLength(1);
    expect(screen.getByRole("link", { name: "Cache choice" }).getAttribute("href")).toBe("/think-tank/tt_1");
    const result = document.querySelector("[data-slot='result']")!;
    const end = document.querySelector("hr.turn-end")!;
    expect(result.compareDocumentPosition(end) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  // TS-14.R26: an empty provider transcript (missing file) or a failed Archive
  // read still shows the retained synthesis, framed as source-unavailable.
  it.each([
    ["an empty transcript", () => HttpResponse.json({ agent_id: "a_judge", events: [] })],
    ["a failed transcript read", () => HttpResponse.json({ error: "gone" }, { status: 500 })],
  ])("shows the synthesis after %s", async (_name, transcript) => {
    server.use(
      http.get("/api/sessions/a_judge/transcript", transcript),
      http.get("/api/sessions/a_judge/think-tank-results", () => HttpResponse.json({
        version: 1, complete: true,
        results: [{ result_id: 1, room_id: "tt_1", room_title: "Cache choice", room_available: true, entry_seq: 9,
          attempt_id: "att", body: "Exact synthesis: keep LRU", generation: "g", turn_id: "t", event_seq: 3, completed_at: "2026-10-07T00:00:00Z" }],
      })),
    );
    renderArchive("a_judge");
    expect(await screen.findByText("Exact synthesis: keep LRU")).toBeInTheDocument();
    expect(screen.getByText("Its originating turn is not in this transcript.")).toBeInTheDocument();
  });
});

// FS-05.R42/A25, INV §7/§8: a failed or empty transcript read is visible, and
// an unknown id offers no Resume/Restore.
describe("ArchiveAgentPage unavailable transcript", () => {
  it("says an unknown id was not found and offers no lifecycle action", async () => {
    server.use(http.get("/api/sessions/a_bogus/transcript", () =>
      HttpResponse.json({ error: { code: "not_found", message: "no such agent: a_bogus" } }, { status: 404 })));

    renderArchive("a_bogus");

    expect(await screen.findByRole("alert")).toHaveTextContent("No session with this id exists.");
    expect(screen.getByRole("heading", { name: "Session not found" })).toBeInTheDocument();
    expect(screen.queryByText(/Archived · read-only/)).toBeNull();
    expect(screen.queryByRole("button", { name: /Resume|Restore/ })).toBeNull();
  });

  it("says a known session's transcript is unavailable and keeps Resume", async () => {
    server.use(http.get("/api/sessions/a_missing/transcript", () => HttpResponse.json({ agent_id: "a_missing", events: [] })));

    renderArchive("a_missing");

    expect(await screen.findByRole("status")).toHaveTextContent("This session's transcript is unavailable");
    expect(screen.getByRole("button", { name: "Resume" })).toBeEnabled();
  });

  it("surfaces a failed read with a retry", async () => {
    let fail = true;
    server.use(http.get("/api/sessions/a_flaky/transcript", () => fail
      ? HttpResponse.json({ error: { code: "internal", message: "disk unreadable" } }, { status: 500 })
      : HttpResponse.json({ agent_id: "a_flaky", events: [{ seq: 1, type: "assistant_text", ts: "t1", data: { delta: "Recovered." } }] })));

    renderArchive("a_flaky");

    expect(await screen.findByRole("alert")).toHaveTextContent("Could not load the transcript: disk unreadable");
    fail = false;
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(await screen.findByText("Recovered.")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).toBeNull();
  });
});
