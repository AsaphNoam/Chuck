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
const server = setupServer(
  http.get("/api/roles", () => HttpResponse.json({})),
  http.get("/api/sessions/a1/transcript", () => {
    transcriptReads++;
    return HttpResponse.json({ agent_id: "a1", events: [
      { agent_id: "a1", seq: 1, type: "assistant_text", ts: "t", data: { text: latestText } },
    ], has_more: false, latest_assistant: latestText });
  }),
  http.get("/api/sessions/a1/prompt", () => { promptReads++; return HttpResponse.json({ agent_id: "a1", text: "" }); }),
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
  useConnection.setState({ link: "checking", unreachableSince: null, revision: 0, agents: {}, transcriptRev: {} });
});
afterEach(() => {
  cleanup();
  disconnect();
  vi.unstubAllGlobals();
});
afterAll(() => server.close());

describe("phone streamed transcript refresh", () => {
  it("bounds transcript and prompt reads during a message burst and renders the final fetched reply", async () => {
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
    queryClient.clear();
  });
});
