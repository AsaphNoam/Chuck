import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { disconnect, livenessMs, start, useConnection } from "./connection";
import { checkPaired, MacUnreachableError, PhoneAPIError } from "./api";

// FS-20.A22/A24/A25, TS-13.R27–R28 — the phone connection owner with a fake
// clock, fake EventSources and a stubbed health read.
vi.mock("./api", async (importOriginal) => ({ ...(await importOriginal<typeof import("./api")>()), checkPaired: vi.fn() }));
const health = vi.mocked(checkPaired);

class FakeEventSource {
  static all: FakeEventSource[] = [];
  onopen: (() => void) | null = null;
  onerror: (() => void) | null = null;
  closed = false;
  private listeners = new Map<string, Array<(event: Event) => void>>();
  constructor(public url: string) {
    FakeEventSource.all.push(this);
  }
  addEventListener(type: string, listener: (event: Event) => void) {
    this.listeners.set(type, [...(this.listeners.get(type) ?? []), listener]);
  }
  removeEventListener() {}
  close() {
    this.closed = true;
  }
  emit(type: string, body: unknown = {}) {
    for (const listener of this.listeners.get(type) ?? []) listener({ data: JSON.stringify(body) } as MessageEvent<string>);
  }
  hydrate(agents: Record<string, unknown>[] = []) {
    this.onopen?.();
    for (const agent of agents) this.emit("state_update", { data: agent });
    this.emit("state_update", { agent_id: "__hydrated__", data: { hydrated: true } });
  }
}

const streams = () => FakeEventSource.all;
const last = () => streams()[streams().length - 1];
const link = () => useConnection.getState().link;
let hidden = false;

function setHidden(value: boolean) {
  hidden = value;
  document.dispatchEvent(new Event("visibilitychange"));
}

// Let the awaited health read settle under fake timers.
const settle = () => vi.advanceTimersByTimeAsync(0);

beforeEach(() => {
  vi.useFakeTimers();
  FakeEventSource.all = [];
  hidden = false;
  Object.defineProperty(document, "visibilityState", { configurable: true, get: () => (hidden ? "hidden" : "visible") });
  (globalThis as unknown as { EventSource: unknown }).EventSource = FakeEventSource;
  health.mockReset();
  health.mockResolvedValue({ status: "ok" });
  useConnection.setState({ link: "checking", unreachableSince: null, revision: 0, agents: {}, transcriptRev: {} });
});

afterEach(() => {
  disconnect();
  vi.useRealTimers();
});

async function connected() {
  start();
  await settle();
  last().hydrate([{ agent_id: "a1", state: "idle" }]);
  expect(link()).toBe("connected");
}

describe("phone connection recovery", () => {
  it("stays checking until the first snapshot hydrates, then coalesces it into one revision", async () => {
    start();
    await settle();
    expect(link()).toBe("checking");
    const before = useConnection.getState().revision;
    last().hydrate([{ agent_id: "a1" }, { agent_id: "a2" }, { agent_id: "a3" }]);
    expect(link()).toBe("connected");
    expect(useConnection.getState().revision).toBe(before + 1);
    expect(Object.keys(useConnection.getState().agents)).toEqual(["a1", "a2", "a3"]);
  });

  it("keeps a healthy stream on repeated return and network signals and pings never refresh views", async () => {
    await connected();
    const revision = useConnection.getState().revision;
    for (let i = 0; i < 5; i++) {
      await vi.advanceTimersByTimeAsync(10_000);
      last().emit("ping");
      setHidden(true);
      setHidden(false);
      window.dispatchEvent(new Event("pageshow"));
      window.dispatchEvent(new Event("online"));
    }
    await settle();
    expect(streams()).toHaveLength(1);
    expect(health).toHaveBeenCalledTimes(1);
    expect(useConnection.getState().revision).toBe(revision);
    expect(link()).toBe("connected");
  });

  it("recovers a silently stalled visible stream through a new snapshot", async () => {
    await connected();
    const stalled = last();
    await vi.advanceTimersByTimeAsync(livenessMs);
    expect(stalled.closed).toBe(true);
    expect(link()).toBe("reconnecting");
    await settle();
    expect(streams()).toHaveLength(2);
    // A late ping on the discarded stream cannot stand in for hydration.
    stalled.emit("ping");
    stalled.hydrate([{ agent_id: "stale" }]);
    expect(link()).toBe("reconnecting");
    last().hydrate([{ agent_id: "a2" }]);
    expect(link()).toBe("connected");
    expect(Object.keys(useConnection.getState().agents)).toEqual(["a2"]);
  });

  it("replaces a stream whose pings stopped while the page was hidden when it returns", async () => {
    await connected();
    setHidden(true);
    await vi.advanceTimersByTimeAsync(5 * 60_000);
    expect(streams()).toHaveLength(1);
    expect(streams()[0].closed).toBe(false); // a healthy hidden stream may stay open
    expect(health).toHaveBeenCalledTimes(1);
    setHidden(false);
    window.dispatchEvent(new Event("pageshow"));
    await settle();
    expect(streams()[0].closed).toBe(true);
    expect(streams()).toHaveLength(2);
    expect(health).toHaveBeenCalledTimes(2);
  });

  it("fails an opening or hydration that exceeds its budget, and pings cannot extend hydration", async () => {
    start();
    await settle();
    const opening = last();
    await vi.advanceTimersByTimeAsync(livenessMs);
    expect(opening.closed).toBe(true);
    await vi.advanceTimersByTimeAsync(5_000);
    const hydrating = last();
    expect(hydrating).not.toBe(opening);
    hydrating.onopen?.();
    for (let i = 0; i < 3; i++) {
      await vi.advanceTimersByTimeAsync(10_000);
      hydrating.emit("ping");
    }
    expect(hydrating.closed).toBe(true);
    expect(link()).toBe("checking");
  });

  it("backs off 5, 10, 20, 40 then 60 seconds during an outage and closes failed sources", async () => {
    await connected();
    health.mockRejectedValue(new MacUnreachableError("down"));
    last().onerror?.();
    expect(streams()[0].closed).toBe(true);
    await settle();
    expect(link()).toBe("unreachable");
    expect(health).toHaveBeenCalledTimes(2); // the first loss is retried at once
    const gaps = [5_000, 10_000, 20_000, 40_000, 60_000, 60_000];
    for (const gap of gaps) {
      const calls = health.mock.calls.length;
      await vi.advanceTimersByTimeAsync(gap - 1);
      expect(health).toHaveBeenCalledTimes(calls);
      await vi.advanceTimersByTimeAsync(1);
      expect(health).toHaveBeenCalledTimes(calls + 1);
    }
    expect(streams()).toHaveLength(1);
    health.mockResolvedValue({ status: "ok" });
    await vi.advanceTimersByTimeAsync(60_000);
    expect(link()).toBe("reconnecting");
    last().hydrate();
    expect(link()).toBe("connected");
  });

  it("makes no attempts while hidden and advances one retry per genuine return", async () => {
    health.mockRejectedValue(new MacUnreachableError("down"));
    start();
    await settle();
    expect(health).toHaveBeenCalledTimes(1);
    setHidden(true);
    await vi.advanceTimersByTimeAsync(10 * 60_000);
    expect(health).toHaveBeenCalledTimes(1);
    setHidden(false);
    window.dispatchEvent(new Event("pageshow"));
    window.dispatchEvent(new Event("online"));
    await settle();
    expect(health).toHaveBeenCalledTimes(2);
    // Backoff was preserved: the next scheduled retry is the second step.
    window.dispatchEvent(new Event("online"));
    await vi.advanceTimersByTimeAsync(9_999);
    expect(health).toHaveBeenCalledTimes(2);
    await vi.advanceTimersByTimeAsync(1);
    expect(health).toHaveBeenCalledTimes(3);
    // Going offline then online is a genuine transition and may advance once.
    window.dispatchEvent(new Event("offline"));
    window.dispatchEvent(new Event("online"));
    await settle();
    expect(health).toHaveBeenCalledTimes(4);
  });

  it("serializes overlapping return signals into one recovery", async () => {
    let answer: (value: unknown) => void = () => undefined;
    await connected();
    last().onerror?.();
    health.mockImplementation(() => new Promise((resolve) => { answer = resolve; }));
    setHidden(true);
    setHidden(false);
    window.dispatchEvent(new Event("pageshow"));
    window.dispatchEvent(new Event("online"));
    await settle();
    answer({ status: "ok" });
    await settle();
    expect(streams()).toHaveLength(2);
  });

  it("closes a not-yet-hydrated stream and aborts the health read when hidden", async () => {
    let signal: AbortSignal | undefined;
    health.mockImplementation((s) => { signal = s; return new Promise(() => undefined); });
    start();
    await settle();
    setHidden(true);
    expect(signal?.aborted).toBe(true);
    health.mockResolvedValue({ status: "ok" });
    setHidden(false);
    await settle();
    const opening = last();
    setHidden(true);
    expect(opening.closed).toBe(true);
    opening.onerror?.();
    await vi.advanceTimersByTimeAsync(120_000);
    expect(streams()).toHaveLength(1);
  });

  it("times out a health read after five seconds as unreachable", async () => {
    health.mockImplementation((signal) => new Promise((_, reject) => signal?.addEventListener("abort", () => reject(new MacUnreachableError("x")))));
    start();
    await vi.advanceTimersByTimeAsync(5_000);
    expect(link()).toBe("unreachable");
  });

  it("stops recovery and shows unpaired when pairing was revoked", async () => {
    await connected();
    health.mockRejectedValue(new PhoneAPIError(401, "remote_unpaired", "pair"));
    setHidden(true);
    last().onerror?.();
    setHidden(false);
    await settle();
    expect(link()).toBe("unpaired");
    expect(streams()[0].closed).toBe(true);
    await vi.advanceTimersByTimeAsync(10 * 60_000);
    window.dispatchEvent(new Event("online"));
    expect(health).toHaveBeenCalledTimes(2);
    expect(streams()).toHaveLength(1);
  });

  it("releases its stream and timers on pagehide and recovers on page-cache restore", async () => {
    await connected();
    window.dispatchEvent(new Event("pagehide"));
    expect(streams()[0].closed).toBe(true);
    window.dispatchEvent(new Event("pageshow"));
    await settle();
    expect(streams()).toHaveLength(2);
  });

  it("gates actions from pagehide through a delayed health read, opening and partial hydration", async () => {
    await connected();
    window.dispatchEvent(new Event("pagehide"));
    expect(link()).toBe("reconnecting");
    let answer: (value: { status: string }) => void = () => undefined;
    health.mockImplementationOnce(() => new Promise((resolve) => (answer = resolve)));
    window.dispatchEvent(new Event("pageshow"));
    await vi.advanceTimersByTimeAsync(4_000);
    expect(link()).toBe("reconnecting");
    answer({ status: "ok" });
    await settle();
    const replacement = last();
    expect(streams()).toHaveLength(2);
    expect(link()).toBe("reconnecting");
    replacement.onopen?.();
    replacement.emit("state_update", { data: { agent_id: "a1", state: "idle" } });
    expect(link()).toBe("reconnecting");
    replacement.emit("state_update", { agent_id: "__hydrated__", data: { hydrated: true } });
    expect(link()).toBe("connected");
  });

  it("ignores callbacks from a replaced or torn-down source", async () => {
    await connected();
    const old = last();
    disconnect();
    old.onerror?.();
    old.emit("state_update", { data: { agent_id: "late" } });
    await vi.advanceTimersByTimeAsync(120_000);
    expect(health).toHaveBeenCalledTimes(1);
    expect(useConnection.getState().agents.late).toBeUndefined();
  });
});
