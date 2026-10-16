import { ensureDraftIndexed, noteDraftChange } from "./draftIndex";

const storageKey = "chuck-chat-drafts";
const maxDrafts = 20;

type Draft = { text: string; editedAt: number };
type StoredDrafts = Record<string, Draft>;

// Per-agent edit counter, bumped on every draft mutation. An in-flight prompt
// captures it before sending so its completion can tell whether the draft it
// submitted is still current, or a newer same-agent draft has replaced it and
// must be left untouched (INV §1/§5). It lives in memory only: an in-flight
// request cannot outlive a page reload, so a fresh process starting at zero is
// correct.
const revisions: Record<string, number> = {};
const listeners = new Map<string, Set<(text: string) => void>>();

// Listeners hear drafts written outside their own composer: a clear after send,
// or a withdrawn held message coming back (FS-03.R90).
export function subscribeChatDraft(agentId: string, listener: (text: string) => void) {
  const subscribers = listeners.get(agentId) ?? new Set<(text: string) => void>();
  subscribers.add(listener);
  listeners.set(agentId, subscribers);
  return () => {
    subscribers.delete(listener);
    if (!subscribers.size) listeners.delete(agentId);
  };
}

function notifyDraft(agentId: string, text: string) {
  listeners.get(agentId)?.forEach((listener) => listener(text));
}

export function chatDraftRevision(agentId: string): number {
  return revisions[agentId] ?? 0;
}

export function bumpChatDraftRevision(agentId: string): number {
  revisions[agentId] = chatDraftRevision(agentId) + 1;
  return revisions[agentId];
}

export function getChatDraft(agentId: string): string {
  const drafts = readDrafts();
  return drafts[agentId]?.text ?? "";
}

export function setChatDraft(agentId: string, text: string) {
  try {
    noteDraftChange(agentId, { text: text !== "" });
  } catch {
    // Draft persistence remains best-effort, matching the existing text draft contract.
  }
  const drafts = readDrafts();
  if (text === "") {
    delete drafts[agentId];
  } else {
    const newest = Math.max(Date.now(), ...Object.values(drafts).map((draft) => draft.editedAt + 1));
    drafts[agentId] = { text, editedAt: newest };
  }
  bumpChatDraftRevision(agentId);
  writeDrafts(drafts);
}

export function discardChatDraft(agentId: string) {
  try {
    noteDraftChange(agentId, { text: false });
  } catch {
    // Keep the in-memory cleanup working when browser storage is unavailable.
  }
  const drafts = readDrafts();
  delete drafts[agentId];
  bumpChatDraftRevision(agentId);
  writeDrafts(drafts);
  notifyDraft(agentId, "");
}

// A withdrawn held message returns as the person's own text. Anything typed
// since stays after it, so neither is lost (FS-03.R90).
export function restoreChatDraft(agentId: string, text: string) {
  const current = getChatDraft(agentId);
  const next = current ? `${text}\n\n${current}` : text;
  setChatDraft(agentId, next);
  notifyDraft(agentId, next);
}


function readDrafts(): StoredDrafts {
  try {
    const raw = window.localStorage.getItem(storageKey);
    if (!raw) return {};
    const parsed: unknown = JSON.parse(raw);
    if (!isStoredDrafts(parsed)) return {};
    const drafts = pruneDrafts(parsed.drafts);
    let removed = false;
    Object.entries(drafts).sort((left, right) => right[1].editedAt - left[1].editedAt).forEach(([agentId, draft]) => {
      try {
        if (!ensureDraftIndexed(agentId, { text: true, editedAt: draft.editedAt })) { delete drafts[agentId]; removed = true; }
      } catch { /* Index migration must not hide a readable text draft. */ }
    });
    if (removed) writeDrafts(drafts);
    // Rehydration enforces the same retention bound as writes.
    if (Object.keys(drafts).length !== Object.keys(parsed.drafts).length) writeDrafts(drafts);
    return drafts;
  } catch {
    return {};
  }
}

function writeDrafts(drafts: StoredDrafts) {
  try {
    window.localStorage.setItem(storageKey, JSON.stringify({ drafts: pruneDrafts(drafts) }));
  } catch {
    // Browser storage is optional: persistence failure must never block typing or Send.
  }
}

function pruneDrafts(drafts: StoredDrafts): StoredDrafts {
  return Object.fromEntries(
    Object.entries(drafts)
      .sort(([leftId, left], [rightId, right]) => right.editedAt - left.editedAt || leftId.localeCompare(rightId))
      .slice(0, maxDrafts),
  );
}

function isStoredDrafts(value: unknown): value is { drafts: StoredDrafts } {
  if (!value || typeof value !== "object" || !("drafts" in value)) return false;
  const drafts = value.drafts;
  return !!drafts && typeof drafts === "object" && Object.values(drafts).every(isDraft);
}

function isDraft(value: unknown): value is Draft {
  return !!value && typeof value === "object" &&
    "text" in value && typeof value.text === "string" && value.text !== "" &&
    "editedAt" in value && typeof value.editedAt === "number" && Number.isFinite(value.editedAt);
}
