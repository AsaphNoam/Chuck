const indexKey = "chuck-chat-draft-index";
const textDraftKey = "chuck-chat-drafts";
const attachmentDB = "chuck-chat-attachments";
const attachmentStore = "drafts";

export type DraftIndexEntry = { editedAt: number; text: boolean; attachments: boolean; pending?: boolean; requestId: string };
export type DraftIndex = Record<string, DraftIndexEntry>;
let evictionListener: ((agentIds: string[]) => void) | null = null;

export function registerDraftEvictionListener(listener: (agentIds: string[]) => void) {
  evictionListener = listener;
  return () => { if (evictionListener === listener) evictionListener = null; };
}

export function chatDraftRequestId(agentId: string) {
  return readIndex()[agentId]?.requestId ?? "";
}

export function hasPersistedAttachmentDraft(agentId: string) {
  const entry = readIndex()[agentId];
  return entry?.attachments === true || entry?.pending === true;
}

export function noteDraftChange(agentId: string, change: { text?: boolean; attachments?: boolean; pending?: boolean }, edited = true) {
  const index = readIndex();
  const existing = index[agentId];
  const next: DraftIndexEntry = {
    editedAt: edited ? Math.max(Date.now(), ...Object.values(index).map((entry) => entry.editedAt + 1)) : (existing?.editedAt ?? Date.now()),
    text: change.text ?? existing?.text ?? false,
    attachments: change.attachments ?? existing?.attachments ?? false,
    pending: change.pending ?? existing?.pending ?? false,
    requestId: edited || !existing?.requestId ? newRequestId() : existing.requestId,
  };
  const removed = !next.text && !next.attachments && !next.pending;
  if (removed) delete index[agentId];
  else index[agentId] = next;
  const evicted = Object.entries(index)
    .sort((left, right) => right[1].editedAt - left[1].editedAt || left[0].localeCompare(right[0]))
    .slice(20)
    .map(([id]) => id);
  evicted.forEach((id) => delete index[id]);
  localStorage.setItem(indexKey, JSON.stringify(index));
  if (removed) {
    pruneTextDrafts([agentId]);
    void deleteAttachmentDrafts([agentId]).catch(() => undefined);
    evictionListener?.([agentId]);
  }
  if (evicted.length) {
    pruneTextDrafts(evicted);
    void deleteAttachmentDrafts(evicted).catch(() => undefined);
    evictionListener?.(evicted);
  }
  return next.requestId;
}

export function preserveDraftRequestId(agentId: string) {
  return chatDraftRequestId(agentId);
}

export function ensureDraftIndexed(agentId: string, draft: { text: boolean; editedAt: number }) {
  const index = readIndex();
  if (index[agentId]) {
    if (!index[agentId].text) {
      index[agentId] = { ...index[agentId], text: draft.text, editedAt: Math.max(index[agentId].editedAt, draft.editedAt) };
      localStorage.setItem(indexKey, JSON.stringify(index));
    }
    return true;
  }
  index[agentId] = { editedAt: draft.editedAt, text: draft.text, attachments: false, requestId: newRequestId() };
  const evicted = Object.entries(index)
    .sort((left, right) => right[1].editedAt - left[1].editedAt || left[0].localeCompare(right[0]))
    .slice(20)
    .map(([id]) => id);
  evicted.forEach((id) => delete index[id]);
  localStorage.setItem(indexKey, JSON.stringify(index));
  if (evicted.length) {
    pruneTextDrafts(evicted);
    void deleteAttachmentDrafts(evicted).catch(() => undefined);
    evictionListener?.(evicted);
  }
  return !evicted.includes(agentId);
}

function readIndex(): DraftIndex {
  try {
    const parsed: unknown = JSON.parse(localStorage.getItem(indexKey) ?? "{}");
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return {};
    return Object.fromEntries(Object.entries(parsed).filter((entry): entry is [string, DraftIndexEntry] => {
      const value = entry[1];
      return !!value && typeof value === "object" && Number.isFinite(value.editedAt) && typeof value.text === "boolean" && typeof value.attachments === "boolean" && (value.pending === undefined || typeof value.pending === "boolean") && typeof value.requestId === "string";
    }));
  } catch {
    return {};
  }
}

function pruneTextDrafts(ids: string[]) {
  try {
    const parsed: unknown = JSON.parse(localStorage.getItem(textDraftKey) ?? "{}");
    if (!parsed || typeof parsed !== "object" || !("drafts" in parsed)) return;
    const drafts = (parsed as { drafts: Record<string, unknown> }).drafts;
    ids.forEach((id) => delete drafts[id]);
    localStorage.setItem(textDraftKey, JSON.stringify({ drafts }));
  } catch {
    // Text drafts keep their existing best-effort storage behavior.
  }
}

async function deleteAttachmentDrafts(ids: string[]) {
  if (typeof indexedDB === "undefined") return;
  const db = await new Promise<IDBDatabase>((resolve, reject) => {
    const request = indexedDB.open(attachmentDB, 1);
    request.onupgradeneeded = () => {
      if (!request.result.objectStoreNames.contains(attachmentStore)) request.result.createObjectStore(attachmentStore);
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
  await new Promise<void>((resolve, reject) => {
    const tx = db.transaction(attachmentStore, "readwrite");
    const store = tx.objectStore(attachmentStore);
    ids.forEach((id) => { store.delete(id); store.delete(`pending:${id}`); });
    tx.oncomplete = () => { db.close(); resolve(); };
    tx.onerror = () => { db.close(); reject(tx.error); };
    tx.onabort = () => { db.close(); reject(tx.error); };
  });
}

function newRequestId() {
  return typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(36).slice(2)}`;
}
