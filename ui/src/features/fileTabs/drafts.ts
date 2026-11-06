import { useSyncExternalStore } from "react";
import { pathKey, type DisplayMode } from "./model";

export type DraftIdentity = { owner: string; source?: string; path: string };
export type FileDraft = DraftIdentity & { version: 1; key: string; lineage: string; generation: number; text: string; baseRevision: string; displayMode: DisplayMode };
export const MAX_DRAFT_TEXT_BYTES = 512 * 1024;
export const MAX_DRAFTS = 50;
export const MAX_DRAFT_BYTES = 32 * 1024 * 1024;
export const draftKey = (id: DraftIdentity) => JSON.stringify([id.owner, id.source ?? "", pathKey(id.path)]);
const bytes = (value: string) => new TextEncoder().encode(value).length;
const records = new Map<string, FileDraft>();
const durable = new Map<string, FileDraft>();
const pendingWrites = new Map<string, { draft: FileDraft }>();
const unsafeEditors = new Set<string>();
export function markDraftUnsafe(key: string, unsafe: boolean) { if (unsafe) unsafeEditors.add(key); else unsafeEditors.delete(key); changed(); }
const listeners = new Set<() => void>();
let revision = 0;
let pending = 0;
let failed = false;
let queue: Promise<void> = Promise.resolve();
let database: Promise<IDBDatabase> | undefined;
function changed() { revision++; listeners.forEach((listener) => listener()); }
function db() {
  return database ??= new Promise<IDBDatabase>((resolve, reject) => {
    const request = indexedDB.open("chuck.file-drafts.v1", 1);
    request.onupgradeneeded = () => request.result.createObjectStore("drafts", { keyPath: "key" });
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
    request.onblocked = () => reject(new Error("Draft database is blocked"));
  });
}
function valid(value: unknown): value is FileDraft {
  if (!value || typeof value !== "object") return false;
  const d = value as FileDraft;
  return d.version === 1 && typeof d.owner === "string" && typeof d.path === "string" &&
    (d.source === undefined || typeof d.source === "string") && typeof d.lineage === "string" &&
    Number.isSafeInteger(d.generation) && d.generation >= 0 && typeof d.text === "string" &&
    bytes(d.text) <= MAX_DRAFT_TEXT_BYTES && typeof d.baseRevision === "string" && d.baseRevision.length <= 4096 &&
    ["rendered", "formatted", "source"].includes(d.displayMode) && d.key === draftKey(d);
}
let restoring: Promise<void> | undefined;
export function restoreDrafts() {
  return restoring ??= (async () => {
    try {
      const database = await db();
      const restored = await new Promise<FileDraft[]>((resolve, reject) => {
        const tx = database.transaction("drafts", "readonly");
        const result: FileDraft[] = [];
        let total = 0;
        const request = tx.objectStore("drafts").openCursor();
        request.onsuccess = () => {
          const cursor = request.result;
          if (!cursor) return;
          const value: unknown = cursor.value;
          if (!valid(value) || result.length >= MAX_DRAFTS || (total += bytes(JSON.stringify(value))) > MAX_DRAFT_BYTES) {
            tx.abort(); return;
          }
          result.push(value); cursor.continue();
        };
        tx.oncomplete = () => resolve(result);
        tx.onabort = tx.onerror = () => reject(new Error("Draft recovery could not be read safely"));
      });
      for (const value of restored) if (!records.has(value.key)) { records.set(value.key, value); durable.set(value.key, value); }
    } catch { failed = true; }
    changed();
  })();
}
function persist(operation: (store: IDBObjectStore) => void, acknowledged: () => void) {
  pending++; changed();
  queue = queue.then(async () => {
    try {
      const database = await db();
      await new Promise<void>((resolve, reject) => {
        const tx = database.transaction("drafts", "readwrite");
        operation(tx.objectStore("drafts"));
        tx.oncomplete = () => resolve();
        tx.onerror = tx.onabort = () => reject(tx.error);
      });
      acknowledged();
    } catch { failed = true; }
    finally { pending--; changed(); }
  });
  return queue;
}
export function getDraft(identity: DraftIdentity) { return records.get(draftKey(identity)); }
export function listDrafts(owner: string) { return [...records.values()].filter((draft) => draft.owner === owner); }
export function putDraft(draft: FileDraft): boolean {
  if (!valid(draft)) return false;
  const next = new Map(records); next.set(draft.key, draft);
  if (next.size > MAX_DRAFTS || [...next.values()].reduce((sum, item) => sum + bytes(JSON.stringify(item)), 0) > MAX_DRAFT_BYTES) {
    return false;
  }
  records.set(draft.key, draft); changed();
  const queued = pendingWrites.get(draft.key);
  if (queued) { queued.draft = draft; return true; }
  const write = { draft };
  pendingWrites.set(draft.key, write);
  let persisted = draft;
  void persist((store) => {
    persisted = write.draft;
    if (pendingWrites.get(draft.key) === write) pendingWrites.delete(draft.key);
    // Check the durable origin-wide budget inside the write transaction too:
    // another browser tab may have admitted drafts since this page restored.
    let count = 1;
    let total = bytes(JSON.stringify(persisted));
    const cursor = store.openCursor();
    cursor.onsuccess = () => {
      const item = cursor.result;
      if (!item) { store.put(persisted); return; }
      if (item.key !== persisted.key) {
        if (!valid(item.value)) { store.transaction.abort(); return; }
        count++; total += bytes(JSON.stringify(item.value));
        if (count > MAX_DRAFTS || total > MAX_DRAFT_BYTES) { store.transaction.abort(); return; }
      }
      item.continue();
    };
  }, () => { durable.set(persisted.key, persisted); });
  return true;
}
export function discardDraft(identity: DraftIdentity, lineage?: string) {
  const key = draftKey(identity);
  if (lineage && records.get(key)?.lineage !== lineage) return queue;
  const discarded = records.get(key);
  // Freeze an earlier queued write so edits after discard cannot move before it.
  pendingWrites.delete(key);
  return persist((store) => { store.delete(key); }, () => {
    if (records.get(key) === discarded) records.delete(key);
    durable.delete(key);
  });
}
/** Apply a save only to its surviving lineage; typing during Save retains its exact text. */
export async function acknowledgeDraft(submitted: FileDraft, savedRevision: string) {
  const current = records.get(submitted.key);
  if (!current || current.lineage !== submitted.lineage) return;
  if (current.generation === submitted.generation) return discardDraft(current, current.lineage);
  putDraft({ ...current, baseRevision: savedRevision });
  await queue;
}
export async function purgeOwnerDrafts(owner: string) { for (const draft of listDrafts(owner)) await discardDraft(draft); }
export function flushDrafts() { return queue; }
let confirmLeave: (() => Promise<boolean>) | undefined;
export function registerDraftLeaveConfirmation(callback: () => Promise<boolean>) { confirmLeave = callback; return () => { if (confirmLeave === callback) confirmLeave = undefined; }; }
export function draftRecoveryAvailable(draft: FileDraft) { return durable.get(draft.key) === draft; }
export function draftPersistenceUnsafe() { return unsafeEditors.size > 0 || pending > 0 || [...records.values()].some((draft) => !draftRecoveryAvailable(draft)); }
export async function confirmDraftLeave() {
  await flushDrafts();
  return !draftPersistenceUnsafe() || await confirmLeave?.() || false;
}
export function useDraftSummary(owner: string) {
  useSyncExternalStore((listener) => { listeners.add(listener); return () => { listeners.delete(listener); }; }, () => revision);
  return { drafts: listDrafts(owner), pending: pending > 0, failed };
}
if (typeof window !== "undefined") window.addEventListener("beforeunload", (event) => {
  if (draftPersistenceUnsafe()) { event.preventDefault(); event.returnValue = ""; }
});
