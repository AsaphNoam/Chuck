import { describe, expect, it, vi } from "vitest";

describe("bounded recoverable drafts", () => {
  it("refuses the 51st draft without evicting existing work and guards failed persistence", async () => {
    vi.resetModules();
    const store = await import("./drafts");
    const draft = (index: number) => ({ version: 1 as const, owner: "conversation:capacity", path: `file-${index}`, key: store.draftKey({ owner: "conversation:capacity", path: `file-${index}` }), lineage: `lineage-${index}`, generation: 1, text: `owned-${index}`, baseRevision: "base", displayMode: "source" as const });
    for (let index = 0; index < 50; index++) expect(store.putDraft(draft(index))).toBe(true);
    expect(store.putDraft(draft(50))).toBe(false);
    await store.flushDrafts();
    expect(store.listDrafts("conversation:capacity")).toHaveLength(50);
    expect(store.getDraft(draft(0))?.text).toBe("owned-0");
    expect(store.getDraft(draft(50))).toBeUndefined();
    expect(store.draftRecoveryAvailable(draft(0))).toBe(false);
    expect(store.draftPersistenceUnsafe()).toBe(true);
  });
  it("ignores a saved reply from a discarded edit lineage at the same path", async () => {
    vi.resetModules();
    const store = await import("./drafts");
    const identity = { owner: "room:lineage", source: "retained-a", path: "a.md" };
    const old = { ...identity, version: 1 as const, key: store.draftKey(identity), lineage: "old", generation: 1, text: "old draft", baseRevision: "r1", displayMode: "source" as const };
    store.putDraft(old);
    await store.discardDraft(identity, old.lineage);
    const replacement = { ...old, lineage: "new", text: "new draft" };
    store.putDraft(replacement);
    await store.acknowledgeDraft(old, "r2");
    expect(store.getDraft(identity)).toEqual(replacement);
  });
});
