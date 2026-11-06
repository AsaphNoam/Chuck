import { act, cleanup, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { FileContent } from "../../api/types";
import { useFileFreshness } from "./useFileFreshness";

const snapshot = (observed_revision: string): FileContent => ({ observed_revision, agent_id: "a", path: "x", content: observed_revision, size: 1, mod_time: "", line_count: 1, language: "text", truncated: false });
afterEach(() => { cleanup(); vi.restoreAllMocks(); vi.useRealTimers(); });
const advance = async (ms: number) => { await act(async () => { await vi.advanceTimersByTimeAsync(ms); }); };

// TS-08.R153's observable contract: fake time drives the real hook and injected
// boundary replies distinguish quiet coalescing, gated application and teardown.
describe("selected file freshness", () => {
  it("cancels queued work while hidden and rechecks immediately on return", async () => {
    vi.useFakeTimers();
    let hidden = false;
    vi.spyOn(document, "hidden", "get").mockImplementation(() => hidden);
    const probe = vi.fn(async () => ({ path: "x", observed_revision: "one" }));
    const cancelPending = vi.fn();
    renderHook(() => useFileFreshness({ identity: "a:x", selected: true, snapshot: snapshot("one"), probe, read: async () => snapshot("one"), apply: vi.fn(), blocked: false, retry: 0, cancelPending }));
    await advance(0);
    expect(probe).toHaveBeenCalledTimes(1);
    act(() => { hidden = true; document.dispatchEvent(new Event("visibilitychange")); });
    expect(cancelPending).toHaveBeenCalledTimes(1);
    await advance(5000);
    expect(probe).toHaveBeenCalledTimes(1);
    await act(async () => { hidden = false; document.dispatchEvent(new Event("visibilitychange")); });
    expect(probe).toHaveBeenCalledTimes(2);
  });

  it("discards a refresh candidate when Save has already applied a newer snapshot", async () => {
    vi.useFakeTimers();
    const apply = vi.fn();
    const { rerender } = renderHook(({ blocked, file }) => useFileFreshness({ identity: "a:x", selected: true, snapshot: file,
      probe: async () => ({ path: "x", observed_revision: "external" }), read: async () => snapshot("external"), apply, blocked, retry: 0 }),
    { initialProps: { blocked: true, file: snapshot("one") } });
    await advance(2000);
    expect(apply).not.toHaveBeenCalled();
    rerender({ blocked: false, file: snapshot("saved") });
    expect(apply).not.toHaveBeenCalled();
  });
  it("coalesces writes for two seconds and applies only after selection ends", async () => {
    vi.useFakeTimers();
    let revision = "one";
    const probe = vi.fn(async () => ({ path: "x", observed_revision: revision }));
    const read = vi.fn(async () => snapshot(revision));
    const apply = vi.fn();
    const { result, rerender } = renderHook(({ blocked }) => useFileFreshness({ identity: "a:x", selected: true, snapshot: snapshot("one"), probe, read, apply, blocked, retry: 0 }), { initialProps: { blocked: true } });
    await advance(0);
    revision = "two";
    await advance(1000);
    expect(result.current.changing).toBe(true);
    revision = "three";
    await advance(1000);
    await advance(1000);
    expect(read).not.toHaveBeenCalled();
    await advance(1000);
    expect(read).toHaveBeenCalledTimes(1);
    expect(result.current.available).toBe(true);
    expect(apply).not.toHaveBeenCalled();
    await advance(2000);
    expect(read).toHaveBeenCalledTimes(1);
    rerender({ blocked: false });
    expect(apply).toHaveBeenCalledWith(snapshot("three"));
  });

  it("fences late replies across selection changes and backs off after failure", async () => {
    vi.useFakeTimers();
    let reject!: (error: Error) => void;
    const probe = vi.fn(() => new Promise<{ path: string; observed_revision: string }>((_resolve, fail) => { reject = fail; }));
    const apply = vi.fn();
    const { result, rerender } = renderHook(({ selected }) => useFileFreshness({ identity: "a:x", selected, snapshot: snapshot("one"), probe, read: async () => snapshot("two"), apply, blocked: false, retry: 0 }), { initialProps: { selected: true } });
    await advance(0);
    rerender({ selected: false });
    await act(async () => reject(new Error("Gone")));
    expect(result.current.stale).toBeNull();
    await advance(6000);
    expect(probe).toHaveBeenCalledTimes(1);
    rerender({ selected: true });
    await advance(0);
    await act(async () => reject(new Error("Gone")));
    expect(result.current.stale).toBe("Gone");
    await advance(4999);
    expect(probe).toHaveBeenCalledTimes(2);
    await advance(1);
    expect(probe).toHaveBeenCalledTimes(3);
    expect(apply).not.toHaveBeenCalled();
  });
});
