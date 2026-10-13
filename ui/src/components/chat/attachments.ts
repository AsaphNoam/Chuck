import type { AttachmentDescriptor } from "../../api/types";
import { noteDraftChange, registerDraftEvictionListener } from "./draftIndex";
export { chatDraftRequestId } from "./draftIndex";
export { hasPersistedAttachmentDraft } from "./draftIndex";

export type DraftAttachment = {
  id: string;
  name: string;
  type: string;
  size: number;
  blob: Blob;
};

const nativeImageTypes = new Set(["image/png", "image/jpeg", "image/webp", "image/gif"]);
const maxFiles = 10;
const maxFileBytes = 10 * 1024 * 1024;
const maxMessageBytes = 25 * 1024 * 1024;
const dbName = "chuck-chat-attachments";
const dbVersion = 1;
const draftsInMemory = new Map<string, DraftAttachment[]>();
const pendingInMemory = new Map<string, PendingAttachmentSend | null>();
const draftVersions = new Map<string, number>();
const writeQueues = new Map<string, Promise<void>>();
const draftListeners = new Map<string, Set<(files: DraftAttachment[]) => void>>();

export function clearAttachmentDraftCacheForTests() {
  draftsInMemory.clear();
  pendingInMemory.clear();
}

export function subscribeAttachmentDraft(agentId: string, listener: (files: DraftAttachment[]) => void) {
  const listeners = draftListeners.get(agentId) ?? new Set();
  listeners.add(listener);
  draftListeners.set(agentId, listeners);
  return () => { listeners.delete(listener); if (!listeners.size) draftListeners.delete(agentId); };
}

function setAttachmentCache(agentId: string, files: DraftAttachment[]) {
  draftsInMemory.set(agentId, files);
  draftVersions.set(agentId, (draftVersions.get(agentId) ?? 0) + 1);
  draftListeners.get(agentId)?.forEach((listener) => listener(files));
}

function forgetEvictedDrafts(agentIds: string[]) {
  for (const agentId of agentIds) {
    draftsInMemory.delete(agentId);
    pendingInMemory.delete(agentId);
    draftVersions.set(agentId, (draftVersions.get(agentId) ?? 0) + 1);
    draftListeners.get(agentId)?.forEach((listener) => listener([]));
    void queueDraftWrite(agentId, (store) => { store.delete(agentId); store.delete(`pending:${agentId}`); }).catch(() => undefined);
  }
}

registerDraftEvictionListener(forgetEvictedDrafts);

export function isPreviewImage(file: Pick<DraftAttachment, "type">) {
  return nativeImageTypes.has(file.type.toLowerCase());
}

export function addFiles(current: DraftAttachment[], files: FileList | File[], source: "picker" | "paste" | "drop" = "picker", limits?: { max_files?: number; max_file_bytes?: number; max_total_bytes?: number }) {
  const countLimit = limits?.max_files ?? maxFiles;
  const fileLimit = limits?.max_file_bytes ?? maxFileBytes;
  const totalLimit = limits?.max_total_bytes ?? maxMessageBytes;
  const accepted = [...current];
  const errors: string[] = [];
  for (const file of Array.from(files)) {
    if (source === "paste" && !file.type.toLowerCase().startsWith("image/")) continue;
    if (accepted.length >= countLimit) {
      errors.push(`A message can include up to ${countLimit} files.`);
      break;
    }
    if (file.size > fileLimit) {
      errors.push(`${file.name || "File"} is larger than ${formatLimit(fileLimit)}.`);
      continue;
    }
    if (accepted.reduce((size, item) => size + item.size, 0) + file.size > totalLimit) {
      errors.push(`${file.name || "File"} would put this message over ${formatLimit(totalLimit)}.`);
      continue;
    }
    accepted.push({ id: newId(), name: file.name || "Pasted image", type: file.type || "application/octet-stream", size: file.size, blob: file });
  }
  return { files: accepted, errors };
}

function formatLimit(bytes: number) {
  return bytes >= 1024 * 1024 ? `${Math.round(bytes / 1024 / 1024)} MiB` : `${Math.round(bytes / 1024)} KiB`;
}

export function newRequestId() {
  return newId();
}

function newId() {
  return typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

type PersistedEntry = { id: string; name: string; type: string; size: number; blob: Blob };
type PersistedDraft = { entries: PersistedEntry[] };

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    if (typeof indexedDB === "undefined") return reject(new Error("IndexedDB is unavailable"));
    const request = indexedDB.open(dbName, dbVersion);
    request.onupgradeneeded = () => {
      if (!request.result.objectStoreNames.contains("drafts")) request.result.createObjectStore("drafts");
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error ?? new Error("Could not open attachment storage"));
  });
}

function storageUnavailable(error: unknown) {
  return error instanceof Error && error.message === "IndexedDB is unavailable";
}

async function transact<T>(mode: IDBTransactionMode, operation: (store: IDBObjectStore) => IDBRequest<T>): Promise<T> {
  const db = await openDB();
  return new Promise((resolve, reject) => {
    const tx = db.transaction("drafts", mode);
    const request = operation(tx.objectStore("drafts"));
    let result: T;
    let requestFailed = false;
    request.onsuccess = () => { result = request.result; };
    request.onerror = () => { requestFailed = true; };
    tx.oncomplete = () => {
      db.close();
      if (requestFailed) reject(request.error ?? new Error("Attachment storage failed"));
      else resolve(result!);
    };
    tx.onerror = () => { db.close(); reject(tx.error ?? new Error("Attachment storage failed")); };
    tx.onabort = () => { db.close(); reject(tx.error ?? new Error("Attachment storage failed")); };
  });
}

export async function restoreAttachments(agentId: string): Promise<DraftAttachment[]> {
  const remembered = draftsInMemory.get(agentId);
  if (remembered) return remembered;
  const versionAtRead = draftVersions.get(agentId) ?? 0;
  let persisted: PersistedDraft | undefined;
  try {
    persisted = await transact<PersistedDraft | undefined>("readonly", (store) => store.get(agentId));
  } catch (error) {
    if (storageUnavailable(error)) {
      if ((draftVersions.get(agentId) ?? 0) === versionAtRead) setAttachmentCache(agentId, []);
      return [];
    }
    throw error;
  }
  const updatedWhileReading = draftsInMemory.get(agentId);
  if (updatedWhileReading) return updatedWhileReading;
  if ((draftVersions.get(agentId) ?? 0) !== versionAtRead) return draftsInMemory.get(agentId) ?? [];
  if (!persisted) {
    setAttachmentCache(agentId, []);
    return [];
  }
  if (!Array.isArray(persisted.entries) || persisted.entries.length > maxFiles) throw new Error("Saved attachments could not be restored safely.");
  let total = 0;
  const files = persisted.entries.map((entry) => {
    if (!entry || typeof entry.id !== "string" || typeof entry.name !== "string" || typeof entry.type !== "string" || !(entry.blob instanceof Blob)) {
      throw new Error("Saved attachments could not be restored safely.");
    }
    const size = entry.blob.size;
    total += size;
    if (size > maxFileBytes || total > maxMessageBytes || size !== entry.size) throw new Error("Saved attachments exceed the current size limits.");
    return { id: entry.id, name: entry.name, type: entry.type, size, blob: entry.blob };
  });
  setAttachmentCache(agentId, files);
  return files;
}

export function rememberAttachments(agentId: string, files: DraftAttachment[]) {
  setAttachmentCache(agentId, files);
}

export async function persistAttachments(agentId: string, files: DraftAttachment[]): Promise<void> {
  const version = draftVersions.get(agentId) ?? 0;
  await queueDraftWrite(agentId, (store) => {
    if ((draftVersions.get(agentId) ?? 0) !== version) return;
    if (files.length === 0) store.delete(agentId);
    else store.put({ entries: files.map(({ id, name, type, size, blob }) => ({ id, name, type, size, blob })) }, agentId);
  });
}

export function touchAttachmentDraft(agentId: string, files: DraftAttachment[], hasText = false, edited = true) {
  setAttachmentCache(agentId, files);
  try {
    noteDraftChange(agentId, { text: hasText, attachments: files.length > 0 }, edited);
    return true;
  } catch {
    return false;
  }
}

export async function removeAcceptedAttachments(agentId: string, acceptedIds: string[], hasText: boolean) {
  const restored = await restoreAttachments(agentId);
  const existing = draftsInMemory.get(agentId) ?? restored;
  const accepted = new Set(acceptedIds);
  const remaining = existing.filter((file) => !accepted.has(file.id));
  touchAttachmentDraft(agentId, remaining, hasText, false);
  await persistAttachments(agentId, remaining);
  return remaining;
}

export async function discardAttachmentDraft(agentId: string) {
  touchAttachmentDraft(agentId, [], false, false);
  pendingInMemory.set(agentId, null);
  await queueDraftWrite(agentId, (store) => { store.delete(agentId); store.delete(`pending:${agentId}`); });
}

function queueDraftWrite(agentId: string, operation: (store: IDBObjectStore) => void | IDBRequest) {
  const previous = writeQueues.get(agentId) ?? Promise.resolve();
  const next = previous.catch(() => undefined).then(async () => {
    const db = await openDB();
    await new Promise<void>((resolve, reject) => {
      const tx = db.transaction("drafts", "readwrite");
      operation(tx.objectStore("drafts"));
      tx.oncomplete = () => { db.close(); resolve(); };
      tx.onerror = () => { db.close(); reject(tx.error ?? new Error("Attachment storage failed")); };
      tx.onabort = () => { db.close(); reject(tx.error ?? new Error("Attachment storage failed")); };
    });
  });
  writeQueues.set(agentId, next);
  void next.finally(() => { if (writeQueues.get(agentId) === next) writeQueues.delete(agentId); }).catch(() => undefined);
  return next;
}

export function attachmentErrorForStorage() {
  return "Attachments are available until this page closes, but browser storage could not save them.";
}

export type AttachmentMessage = { text: string; attachments: AttachmentDescriptor[] };
export type PendingAttachmentSend = { text: string; files: DraftAttachment[]; requestId: string; steer: boolean; revision: number; retryable?: boolean };

export async function persistPendingAttachmentSend(agentId: string, pending: PendingAttachmentSend) {
  pendingInMemory.set(agentId, pending);
  await queueDraftWrite(agentId, (store) => store.put({
    ...pending,
    files: pending.files.map(({ id, name, type, size, blob }) => ({ id, name, type, size, blob })),
  }, `pending:${agentId}`));
  noteDraftChange(agentId, { pending: true }, false);
}

export async function restorePendingAttachmentSend(agentId: string): Promise<PendingAttachmentSend | null> {
  let value: PendingAttachmentSend | undefined;
  try {
    value = await transact<PendingAttachmentSend | undefined>("readonly", (store) => store.get(`pending:${agentId}`));
  } catch (error) {
    if (storageUnavailable(error)) return pendingInMemory.get(agentId) ?? null;
    throw error;
  }
  if (pendingInMemory.has(agentId)) return pendingInMemory.get(agentId) ?? null;
  if (!value) return null;
  if (typeof value.text !== "string" || typeof value.requestId !== "string" || !value.requestId || !Array.isArray(value.files) || value.files.length > maxFiles) {
    throw new Error("Saved message could not be restored safely.");
  }
  let total = 0;
  const files = value.files.map((file) => {
    if (!file || typeof file.id !== "string" || typeof file.name !== "string" || typeof file.type !== "string" || !(file.blob instanceof Blob) || file.blob.size > maxFileBytes) throw new Error("Saved message could not be restored safely.");
    total += file.blob.size;
    return { id: file.id, name: file.name, type: file.type, size: file.blob.size, blob: file.blob };
  });
  if (total > maxMessageBytes) throw new Error("Saved message could not be restored safely.");
  return { text: value.text, files, requestId: value.requestId, steer: value.steer === true, revision: value.revision, retryable: value.retryable !== false };
}

export async function clearPendingAttachmentSend(agentId: string, requestId?: string) {
  const current = pendingInMemory.get(agentId);
  if (requestId && current && current.requestId !== requestId) return;
  let cleared = !requestId;
  await queueDraftWrite(agentId, (store) => {
    if (!requestId) { store.delete(`pending:${agentId}`); return; }
    const request = store.get(`pending:${agentId}`);
    request.onsuccess = () => {
      const stored = request.result as PendingAttachmentSend | undefined;
      if (!stored || stored.requestId === requestId) { store.delete(`pending:${agentId}`); cleared = true; }
    };
  });
  if (!cleared) return;
  const latest = pendingInMemory.get(agentId);
  if (!latest || latest.requestId === requestId) pendingInMemory.set(agentId, null);
  noteDraftChange(agentId, { pending: false }, false);
}
