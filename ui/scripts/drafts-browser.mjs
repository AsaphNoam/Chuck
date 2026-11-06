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
    for (let index = 0; index < 50; index++) {
      const identity = { owner: "conversation:browser-recovery", source: "retained", path: `file-${index}.md` };
      const draft = { ...identity, version: 1, key: store.draftKey(identity), lineage: `edit-${index}`, generation: 1, text: `exact\r\n${index}\n雪`, baseRevision: `revision-${index}`, displayMode: "source" };
      if (!store.putDraft(draft)) throw new Error(`Unexpected refusal ${index}`);
    }
    await store.flushDrafts();
    const drafts = store.listDrafts("conversation:browser-recovery");
    return { count: drafts.length, durable: drafts.every(store.draftRecoveryAvailable), unsafe: store.draftPersistenceUnsafe() };
  });
  assert.deepEqual(admitted, { count: 50, durable: true, unsafe: false });
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
  console.log("PASS: real IndexedDB acknowledged 50 drafts; reload retained exact CRLF/UTF-8, base/source identity; draft 51 refused without eviction; stale-page transaction enforced origin capacity.");
} finally { await browser.close(); }
