import React from "react";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { AgentScreen } from "./AgentScreen";
import { disconnect, start, useConnection } from "./connection";
import type { AgentState } from "../api/types";

vi.mock("./api", async (importOriginal) => ({ ...(await importOriginal<typeof import("./api")>()), checkPaired: vi.fn(async () => ({ status: "ok" })) }));

class FakeEventSource {
  static current: FakeEventSource | undefined;
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  private listeners = new Map<string, Array<(event: MessageEvent<string>) => void>>();
  constructor() { FakeEventSource.current = this; }
  addEventListener(type: string, listener: (event: MessageEvent<string>) => void) {
    this.listeners.set(type, [...(this.listeners.get(type) ?? []), listener]);
  }
  removeEventListener() {}
  close() {}
  emit(type: string, value: unknown) {
    for (const listener of this.listeners.get(type) ?? []) listener({ data: JSON.stringify(value) } as MessageEvent<string>);
  }
}

const agent: AgentState = {
  agent_id: "a1", name: "", role: "implementer", project: "my-app", backend: "claude", model: "m", fast: false,
  fast_available: false, steering_available: true, interface: "chat", created_at: "", running: true,
  state: "busy", detail: "", context_pct: 0, updated_at: 0, archived: false,
};
let transcriptReads = 0;
let promptReads = 0;
let latestText = "Initial reply";
let recovery: { request_id: string; text: string; delivery: string } | undefined;
const server = setupServer(
  http.get("/api/roles", () => HttpResponse.json({})),
  http.get("/api/sessions/a1/transcript", () => {
    transcriptReads++;
    return HttpResponse.json({ agent_id: "a1", events: [
      { agent_id: "a1", seq: 1, type: "assistant_text", ts: "t", data: { text: latestText } },
    ], has_more: false, latest_assistant: latestText });
  }),
  http.get("/api/sessions/a1/prompt", () => { promptReads++; return HttpResponse.json({ agent_id: "a1", text: "", recovery }); }),
  http.get("/api/sessions/a1/files", () => HttpResponse.json({ agent_id: "a1", files: [] })),
  http.get("/api/remote/runtime-options", () => HttpResponse.json({ backends: [] })),
);

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
beforeEach(() => {
  FakeEventSource.current = undefined;
  vi.stubGlobal("EventSource", FakeEventSource as unknown as typeof EventSource);
  window.scrollTo = vi.fn();
  transcriptReads = 0;
  promptReads = 0;
  latestText = "Initial reply";
  recovery = undefined;
  useConnection.setState({ link: "checking", unreachableSince: null, revision: 0, agents: {}, transcriptRev: {} });
});
afterEach(() => {
  cleanup();
  disconnect();
  vi.unstubAllGlobals();
  server.resetHandlers();
});
afterAll(() => server.close());

// A long conversation whose latest user turn crosses the first older-page
// boundary; `revise()` commits a desktop edit that replaces seqs 150–900.
function revisableConversation(olderPage: () => Promise<void> = async () => undefined) {
  let revised = false;
  const oldTail = Array.from({ length: 750 }, (_, index) => ({
    agent_id: "a1", seq: 151 + index, type: "assistant_text", ts: "t", data: { text: index === 0 ? "Old reply tail" : "" },
  }));
  server.use(http.get("/api/sessions/a1/transcript", async ({ request }) => {
    const url = new URL(request.url);
    if (url.searchParams.has("before_seq")) {
      await olderPage();
      return HttpResponse.json({ agent_id: "a1", history_revision: 0, has_more: false, events: [
        { agent_id: "a1", seq: 150, type: "user_text", ts: "t", data: { text: "Superseded user question" } },
      ] });
    }
    if (!revised && url.searchParams.has("since_seq")) return HttpResponse.json({ agent_id: "a1", history_revision: 0, has_more: false, events: oldTail.slice(1) });
    return HttpResponse.json({ agent_id: "a1", history_revision: revised ? 1 : 0, has_more: !revised, events: revised ? [
      { agent_id: "a1", seq: 901, type: "conversation_revision", ts: "t", data: { revision: 1, replaces_from_seq: 150, replaces_through_seq: 900 } },
      { agent_id: "a1", seq: 902, type: "user_text", ts: "t", data: { text: "Revised user question" } },
      { agent_id: "a1", seq: 903, type: "assistant_text", ts: "t", data: { text: "Revised reply" } },
    ] : oldTail });
  }));
  return { revise: () => { revised = true; } };
}

async function openConversation() {
  start();
  await waitFor(() => expect(FakeEventSource.current).toBeDefined());
  const stream = FakeEventSource.current!;
  await act(async () => {
    stream.onopen?.();
    stream.emit("state_update", { data: agent });
    stream.emit("state_update", { agent_id: "__hydrated__", data: { hydrated: true } });
  });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  render(<QueryClientProvider client={queryClient}><AgentScreen agentId="a1" /></QueryClientProvider>);
  expect(await screen.findByText("Old reply tail")).toBeInTheDocument();
  return { stream, queryClient };
}

function scrollToTop() {
  Object.defineProperty(window, "scrollY", { configurable: true, value: 100 });
  act(() => window.dispatchEvent(new Event("scroll")));
  Object.defineProperty(window, "scrollY", { configurable: true, value: 0 });
  act(() => window.dispatchEvent(new Event("scroll")));
}

async function reconnectRevised(stream: FakeEventSource) {
  await act(async () => {
    stream.onopen?.();
    stream.emit("state_update", { data: { ...agent, history_revision: 1 } });
    stream.emit("state_update", { agent_id: "__hydrated__", data: { hydrated: true } });
  });
}

describe("phone revised conversation after older pages (TS-03.R91, FS-01.R61)", () => {
  it.each(["reconnect", "event"])("drops older cached pages after %s discovers a revised conversation", async (trigger) => {
    const conversation = revisableConversation();
    const { stream, queryClient } = await openConversation();
    scrollToTop();
    expect(await screen.findByText("Superseded user question")).toBeInTheDocument();
    conversation.revise();
    if (trigger === "reconnect") await reconnectRevised(stream);
    else await act(async () => {
      stream.emit("new_message", { agent_id: "a1", data: { type: "conversation_revision", seq: 901, data: { revision: 1 } } });
    });
    expect(await screen.findByText("Revised reply")).toBeInTheDocument();
    expect(screen.queryByText("Superseded user question")).not.toBeInTheDocument();
    queryClient.clear();
  });

  it("discards an older-page read that started before a reconnect revealed a revision", async () => {
    let release!: () => void;
    const gate = new Promise<void>((resolve) => { release = resolve; });
    let olderStarted = false;
    const conversation = revisableConversation(async () => { olderStarted = true; await gate; });
    const { stream, queryClient } = await openConversation();
    scrollToTop();
    await waitFor(() => expect(olderStarted).toBe(true));
    conversation.revise();
    await reconnectRevised(stream);
    expect(await screen.findByText("Revised reply")).toBeInTheDocument();
    await act(async () => { release(); await new Promise((resolve) => setTimeout(resolve, 20)); });
    expect(screen.queryByText("Superseded user question")).not.toBeInTheDocument();
    expect(screen.getByText("Revised reply")).toBeInTheDocument();
    queryClient.clear();
  });
});

describe("phone streamed transcript refresh", () => {
  it("bounds transcript and prompt reads during a message burst and renders the final fetched reply", async () => {
    recovery = { request_id: "peer-recovers", text: "Peer recovery", delivery: "released" };
    start();
    await waitFor(() => expect(FakeEventSource.current).toBeDefined());
    const stream = FakeEventSource.current!;
    await act(async () => {
      stream.onopen?.();
      stream.emit("state_update", { data: agent });
      stream.emit("state_update", { agent_id: "__hydrated__", data: { hydrated: true } });
    });

    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
    render(<QueryClientProvider client={queryClient}><AgentScreen agentId="a1" /></QueryClientProvider>);
    await waitFor(() => expect(transcriptReads).toBe(1));
    await waitFor(() => expect(promptReads).toBe(1));

    for (let i = 0; i < 5; i++) {
      await act(async () => {
        latestText = `Streamed reply ${i + 1}`;
        stream.emit("new_message", { agent_id: "a1", data: { seq: i + 1, type: "assistant_text", text: latestText } });
      });
      await new Promise((resolve) => setTimeout(resolve, 8));
    }
    await act(async () => { await new Promise((resolve) => setTimeout(resolve, 300)); });
    await waitFor(() => expect(screen.getByText("Streamed reply 5")).toBeInTheDocument());
    expect(transcriptReads).toBeLessThanOrEqual(2);
    expect(promptReads).toBeLessThanOrEqual(2);
    expect(await screen.findByText("Peer recovery")).toBeInTheDocument();
    act(() => useConnection.getState().setLink("reconnecting"));
    expect(screen.getByText("Peer recovery")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send recovered message" })).toBeDisabled();
    act(() => useConnection.getState().setLink("connected"));
    const beforeSettlement = transcriptReads;
    await act(async () => {
      recovery = undefined;
      stream.emit("input_recovery_changed", { agent_id: "a1", data: { request_id: "peer-recovers" } });
    });
    await waitFor(() => expect(screen.queryByText("Peer recovery")).not.toBeInTheDocument());
    expect(transcriptReads).toBe(beforeSettlement);
    queryClient.clear();
  });

  it("drops the replaced window and rereads the transcript from the start on a conversation revision (FS-01.R61)", async () => {
    start();
    await waitFor(() => expect(FakeEventSource.current).toBeDefined());
    const stream = FakeEventSource.current!;
    await act(async () => {
      stream.onopen?.();
      stream.emit("state_update", { data: agent });
      stream.emit("state_update", { agent_id: "__hydrated__", data: { hydrated: true } });
    });
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
    render(<QueryClientProvider client={queryClient}><AgentScreen agentId="a1" /></QueryClientProvider>);
    expect(await screen.findByText("Initial reply")).toBeInTheDocument();
    const reads = transcriptReads;
    latestText = "Revised reply";
    await act(async () => {
      stream.emit("new_message", { agent_id: "a1", data: { type: "conversation_revision", seq: 9, data: { revision: 1 } } });
    });
    expect(await screen.findByText("Revised reply")).toBeInTheDocument();
    expect(screen.queryByText("Initial reply")).not.toBeInTheDocument();
    expect(transcriptReads).toBe(reads + 1);
    queryClient.clear();
  });
});
