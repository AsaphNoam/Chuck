import { afterEach, describe, expect, it, vi } from "vitest";
import { STORAGE_KEY } from "./model";
import { createReadLimiter, ReadDropped } from "./readLimiter";
import { fileTabs, flushFileTabs, loadFileTabs, STORAGE_NOTICE, useFileTabsStore } from "./store";

const KEY = "conversation:a1";
const saved = () => JSON.parse(localStorage.getItem(STORAGE_KEY) ?? "null");

afterEach(() => vi.restoreAllMocks());

describe("saved file tabs store (FS-03.R103, TS-08.R141)", () => {
  it("saves membership at once and coalesces reading positions until flushed", () => {
    const a = fileTabs.open(KEY, { path: "a.md" })!.tab;
    fileTabs.open(KEY, { path: "b.md" });
    expect(saved().owners[KEY].tabs.map((tab: { path: string }) => tab.path)).toEqual(["a.md", "b.md"]);
    fileTabs.setView(a.id, { rendered: false, anchors: { rendered: null, formatted: null, source: { key: 12, offset: 0 } } });
    expect(saved().owners[KEY].tabs[0].rendered).toBe(true);
    flushFileTabs();
    expect(saved().owners[KEY].tabs[0]).toMatchObject({ rendered: false, anchors: { source: { key: 12, offset: 0 } } });
    loadFileTabs();
    expect(fileTabs.view(a.id).anchors.source).toEqual({ key: 12, offset: 0 });
    expect(useFileTabsStore.getState().owners[KEY].active).not.toBe(a.id);
  });

  it("removes the record with the last owner and on explicit deletion", () => {
    const a = fileTabs.open(KEY, { path: "a.md" })!.tab;
    fileTabs.open("room:r", { path: "x", source: "s" });
    fileTabs.close(KEY, a.id);
    expect(saved().owners[KEY]).toBeUndefined();
    fileTabs.dropOwner("room:r");
    expect(localStorage.getItem(STORAGE_KEY)).toBeNull();
  });

  it("explains a refused admission and keeps the set", () => {
    for (let i = 0; i < 50; i += 1) fileTabs.open(KEY, { path: `f${i}` });
    expect(fileTabs.open(KEY, { path: "f50" })).toBeNull();
    expect(useFileTabsStore.getState().notice).toMatch(/Close a tab/);
    expect(useFileTabsStore.getState().owners[KEY].tabs).toHaveLength(50);
    expect(fileTabs.open(KEY, { path: "f1" })?.added).toBe(false);
  });

  it("stays usable in memory when storage refuses writes, with one notice", () => {
    vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new DOMException("full", "QuotaExceededError"); });
    fileTabs.open(KEY, { path: "a.md" });
    expect(useFileTabsStore.getState().notice).toBe(STORAGE_NOTICE);
    fileTabs.dismissNotice();
    fileTabs.open(KEY, { path: "b.md" });
    expect(useFileTabsStore.getState().notice).toBeNull();
    expect(useFileTabsStore.getState().owners[KEY].tabs).toHaveLength(2);
  });

  it("starts empty with recovery feedback from damaged saved data", () => {
    localStorage.setItem(STORAGE_KEY, "{not json");
    loadFileTabs();
    expect(useFileTabsStore.getState().owners).toEqual({});
    expect(useFileTabsStore.getState().notice).toMatch(/could not be read.*open files again/);
  });
});

describe("read limiter (TS-08.R144)", () => {
  function deferred() {
    let resolve!: (value: string) => void;
    const promise = new Promise<string>((r) => { resolve = r; });
    return { promise, resolve };
  }

  it("keeps four reads outstanding, serves the selected tab first and drops unselected opens", async () => {
    const limiter = createReadLimiter(4);
    const reads = Array.from({ length: 4 }, deferred);
    const starts: string[] = [];
    reads.forEach((read, i) => void limiter.run(`t${i}`, "open", () => { starts.push(`t${i}`); return read.promise; }));
    limiter.select("t9");
    const dropped = limiter.run("t5", "open", () => { starts.push("t5"); return Promise.resolve("x"); });
    const reload = limiter.run("t6", "reload", () => { starts.push("t6"); return Promise.resolve("x"); });
    const selected = limiter.run("t9", "open", () => { starts.push("t9"); return Promise.resolve("x"); });
    await expect(dropped).rejects.toBeInstanceOf(ReadDropped);
    expect(limiter.outstanding).toBe(4);
    reads[0].resolve("done");
    await selected;
    expect(starts).toEqual(["t0", "t1", "t2", "t3", "t9"]);
    reads[1].resolve("done");
    await reload;
    expect(starts.at(-1)).toBe("t6");
  });
});
