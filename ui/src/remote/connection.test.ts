import { afterEach, expect, it, vi } from "vitest";
import { classify, disconnect, useConnection } from "./connection";

afterEach(() => {
  disconnect();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

// Mobile outage investigation 2026-10-12, HANDOFF finding MOBILE-OFFLINE-1.
// A lost tailnet route can leave fetch pending rather than immediately rejecting.
it.skip("marks the Mac unreachable when its health request never answers", async () => {
  vi.useFakeTimers();
  useConnection.setState({ link: "checking", unreachableSince: null });
  const fetch = vi.fn((_url: string, init?: RequestInit) => new Promise<Response>((_resolve, reject) => {
    const signal = init?.signal;
    if (signal?.aborted) reject(new DOMException("Aborted", "AbortError"));
    else signal?.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")), { once: true });
  }));
  vi.stubGlobal("fetch", fetch);

  void classify();
  await vi.advanceTimersByTimeAsync(30_000);

  expect(fetch).toHaveBeenCalledOnce();
  expect(useConnection.getState().link).toBe("unreachable");
  expect(useConnection.getState().unreachableSince).not.toBeNull();
});
