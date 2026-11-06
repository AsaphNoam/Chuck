import assert from "node:assert/strict";
import { chromium } from "playwright";

// Run against a loopback Vite server; fresh browser context never touches user drafts.
const base = process.env.DRAFT_TEST_URL ?? "http://127.0.0.1:5173";
const browser = await chromium.launch();
try {
  const context = await browser.newContext();
  const page = await context.newPage();
  await page.goto(base);
  const admitted = await page.evaluate(async () => {
    const store = await import("/src/features/fileTabs/drafts.ts");
    await store.restoreDrafts();
    let writes = 0;
    const transaction = IDBDatabase.prototype.transaction;
    IDBDatabase.prototype.transaction = function (...args) { if (args[1] === "readwrite") writes++; return transaction.apply(this, args); };
    for (let index = 0; index < 50; index++) {
      const identity = { owner: "conversation:browser-recovery", source: "retained", path: `file-${index}.md` };
      const draft = { ...identity, version: 1, key: store.draftKey(identity), lineage: `edit-${index}`, generation: 1, text: `exact\r\n${index}\n雪`, baseRevision: `revision-${index}`, displayMode: "source" };
      if (!store.putDraft(draft)) throw new Error(`Unexpected refusal ${index}`);
      if (index === 0) for (let generation = 2; generation <= 1000; generation++) store.putDraft({ ...draft, generation });
    }
    await store.flushDrafts();
    const drafts = store.listDrafts("conversation:browser-recovery");
    return { count: drafts.length, durable: drafts.every(store.draftRecoveryAvailable), unsafe: store.draftPersistenceUnsafe(), writes };
  });
  assert.deepEqual(admitted, { count: 50, durable: true, unsafe: false, writes: 50 });
  await page.reload();
  const recovered = await page.evaluate(async () => {
    const store = await import("/src/features/fileTabs/drafts.ts");
    await store.restoreDrafts();
    const drafts = store.listDrafts("conversation:browser-recovery");
    const first = drafts.find((draft) => draft.path === "file-0.md");
    const extra = { ...first, path: "overflow", key: store.draftKey({ ...first, path: "overflow" }) };
    const refused = !store.putDraft(extra);
    await store.flushDrafts();
    return { count: drafts.length, text: first.text, base: first.baseRevision, source: first.source, refused, after: store.listDrafts(first.owner).length, overflow: !!store.getDraft(extra) };
  });
  assert.deepEqual(recovered, { count: 50, text: "exact\r\n0\n雪", base: "revision-0", source: "retained", refused: true, after: 50, overflow: false });
  // A second page must recheck the durable budget, even with a stale in-memory inventory.
  const stale = await context.newPage();
  await stale.goto(base);
  const enforced = await stale.evaluate(async () => {
    const store = await import("/src/features/fileTabs/drafts.ts");
    const identity = { owner: "room:other-tab", path: "overflow" };
    const draft = { ...identity, version: 1, key: store.draftKey(identity), lineage: "stale-page", generation: 1, text: "retained memory only", baseRevision: "r", displayMode: "source" };
    store.putDraft(draft); await store.flushDrafts();
    return { durable: store.draftRecoveryAvailable(draft), unsafe: store.draftPersistenceUnsafe() };
  });
  assert.deepEqual(enforced, { durable: false, unsafe: true });
  await page.reload();
  const count = await page.evaluate(async () => { const store = await import("/src/features/fileTabs/drafts.ts"); await store.restoreDrafts(); return store.listDrafts("conversation:browser-recovery").length; });
  assert.equal(count, 50);
  // Separate JS realms share actual IndexedDB: page A's local lineage cannot
  // authorize deleting/replacing the record page B has since acknowledged.
  for (const replacement of ["lineage", "generation"]) for (const action of ["discard", "save", "save-with-newer-text", "write"]) {
    const shared = await browser.newContext();
    const a = await shared.newPage(), b = await shared.newPage();
    await a.goto(base);
    await a.evaluate(async () => {
      const store = await import("/src/features/fileTabs/drafts.ts");
      const identity = { owner: "conversation:cross-page", path: "shared.md" };
      store.putDraft({ ...identity, version: 1, key: store.draftKey(identity), lineage: "old", generation: 1, text: "old text", baseRevision: "r1", displayMode: "source" });
      await store.flushDrafts();
    });
    await b.goto(base);
    await b.evaluate(async (replacement) => {
      const store = await import("/src/features/fileTabs/drafts.ts");
      await store.restoreDrafts();
      const old = store.listDrafts("conversation:cross-page")[0];
      if (replacement === "lineage") await store.discardDraft(old, old.lineage);
      const newer = { ...old, lineage: replacement === "lineage" ? "new" : "old", generation: 2, text: "newer durable work" };
      store.putDraft(newer); await store.flushDrafts();
      if (!store.draftRecoveryAvailable(newer)) throw new Error("B never acknowledged recovery");
    }, replacement);
    await a.evaluate(async (action) => {
      const store = await import("/src/features/fileTabs/drafts.ts");
      const old = store.listDrafts("conversation:cross-page")[0];
      if (action === "discard") await store.discardDraft(old, old.lineage);
      else if (action === "save") await store.acknowledgeDraft(old, "r2");
      else {
        store.putDraft({ ...old, generation: 2, text: "stale page typing" });
        await store.flushDrafts();
        if (action === "save-with-newer-text") await store.acknowledgeDraft(old, "r2");
      }
    }, action);
    await b.reload();
    const retained = await b.evaluate(async () => {
      const store = await import("/src/features/fileTabs/drafts.ts");
      await store.restoreDrafts();
      return store.listDrafts("conversation:cross-page").map(({ lineage, text, baseRevision }) => ({ lineage, text, baseRevision }));
    });
    assert.deepEqual(retained, [{ lineage: replacement === "lineage" ? "new" : "old", text: "newer durable work", baseRevision: "r1" }], `${replacement}: ${action}`);
    await shared.close();
  }
  console.log("PASS: stale cross-page discard, save acknowledgement and writes preserve newer durable lineage after reload.");
  console.log("PASS: real IndexedDB acknowledged 50 drafts; 1000 rapid generations coalesced to one transaction; reload retained exact CRLF/UTF-8, base/source identity; draft 51 refused without eviction; stale-page transaction enforced origin capacity.");
} finally { await browser.close(); }
