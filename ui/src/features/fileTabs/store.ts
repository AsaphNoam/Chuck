import { create } from "zustand";
import {
  MAX_SAVED_UNITS, STORAGE_KEY, SavedTabsError, closeTab, decodeTabs, defaultView, encodeTabs, moveTab, openTab,
  resolveTab, selectTab, type FileTab, type FileTarget, type Owners, type TabView,
} from "./model";

// The browser's saved file tabs (FS-03.R103, TS-08.R141). Owners, order and the
// active tab are reactive state for the tab strips; each tab's display mode and
// reading anchors change on every scroll, so they live outside React state and
// are only read when a tab's reader is created. Nothing here holds file bytes.

type FileTabsState = { owners: Owners; notice: string | null };

export const useFileTabsStore = create<FileTabsState>(() => ({ owners: {}, notice: null }));

const views = new Map<string, TabView>();
const SAVE_DELAY_MS = 400;
let saveTimer: ReturnType<typeof setTimeout> | undefined;
let saveFailing = false;

export const STORAGE_NOTICE = "File tabs can't be saved in this browser, so changes may reset after closing or reloading the page.";

function report(notice: string) {
  useFileTabsStore.setState({ notice });
}

function storage(): Storage | null {
  try { return globalThis.localStorage ?? null; }
  catch { return null; }
}

// load restores the saved record once per page. Damaged or unknown data is not
// used: the reader starts empty and says why instead of blocking the page.
export function loadFileTabs() {
  views.clear();
  let raw: string | null = null;
  try { raw = storage()?.getItem(STORAGE_KEY) ?? null; }
  catch {
    saveFailing = true;
    useFileTabsStore.setState({ owners: {}, notice: STORAGE_NOTICE });
    return;
  }
  try {
    const decoded = decodeTabs(raw);
    decoded.views.forEach((view, id) => views.set(id, view));
    useFileTabsStore.setState({ owners: decoded.owners, notice: null });
  } catch (error) {
    const reason = error instanceof SavedTabsError ? error.message : "Saved file tabs could not be read.";
    useFileTabsStore.setState({ owners: {}, notice: `${reason} They were reset; open files again to continue.` });
  }
}

function save(owners = useFileTabsStore.getState().owners): boolean {
  clearTimeout(saveTimer);
  saveTimer = undefined;
  const text = encodeTabs(owners, views);
  if (text.length > MAX_SAVED_UNITS) {
    report(STORAGE_NOTICE);
    return false;
  }
  try {
    const target = storage();
    if (!target) throw new Error("no storage");
    if (Object.keys(owners).length === 0) target.removeItem(STORAGE_KEY);
    else target.setItem(STORAGE_KEY, text);
    saveFailing = false;
    return true;
  } catch {
    // One notice for a run of failures, not one per scroll (TS-08.R141).
    if (!saveFailing) report(STORAGE_NOTICE);
    saveFailing = true;
    return false;
  }
}

function commit(owners: Owners) {
  useFileTabsStore.setState({ owners });
  save(owners);
}

export function flushFileTabs() {
  if (saveTimer !== undefined) save();
}

const REFUSED = {
  conversation: "This conversation already has 50 open files. Close a tab to open another.",
  room: "This room already has 50 open files. Close a tab to open another.",
  total: "This browser already keeps 500 file tabs. Close a tab here or in another conversation to open another.",
  bytes: "Saved file tabs are full in this browser. Close a tab to open another.",
};

function newTabId() {
  return typeof crypto?.randomUUID === "function" ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

export const fileTabs = {
  /** Opens or selects a file; null when a limit refused it (FS-03.R105). */
  open(key: string, target: FileTarget): { tab: FileTab; added: boolean } | null {
    const result = openTab(useFileTabsStore.getState().owners, key, target, newTabId());
    if ("refused" in result) {
      report(result.refused === "total" ? REFUSED.total : key.startsWith("room:") ? REFUSED.room : REFUSED.conversation);
      return null;
    }
    if (result.added && encodeTabs(result.owners, views).length > MAX_SAVED_UNITS) {
      report(REFUSED.bytes);
      return null;
    }
    commit(result.owners);
    return { tab: result.tab, added: result.added };
  },
  select(key: string, id: string) {
    const owners = useFileTabsStore.getState().owners;
    const next = selectTab(owners, key, id);
    if (next !== owners) commit(next);
  },
  close(key: string, id: string) {
    const owners = useFileTabsStore.getState().owners;
    const next = closeTab(owners, key, id);
    if (next === owners) return;
    views.delete(id);
    commit(next);
  },
  move(key: string, id: string, to: number) {
    const owners = useFileTabsStore.getState().owners;
    const next = moveTab(owners, key, id, to);
    if (next !== owners) commit(next);
  },
  /** Records a read's normalized path; returns the tab that now names the file. */
  resolve(key: string, id: string, path: string): string | null {
    const owners = useFileTabsStore.getState().owners;
    const { owners: next, kept } = resolveTab(owners, key, id, path);
    if (kept !== id) views.delete(id);
    if (next !== owners) commit(next);
    return kept;
  },
  /** An explicit conversation/room deletion ends its saved set (FS-03.R103). */
  dropOwner(key: string) {
    const owners = useFileTabsStore.getState().owners;
    const owner = owners[key];
    if (!owner) return;
    owner.tabs.forEach((tab) => views.delete(tab.id));
    const { [key]: _dropped, ...rest } = owners;
    commit(rest);
  },
  view(id: string): TabView {
    const view = views.get(id) ?? defaultView();
    return { rendered: view.rendered, anchors: { ...view.anchors } };
  },
  // Scroll changes are coalesced; switching tabs and leaving flush them.
  setView(id: string, view: TabView) {
    if (!isOpen(id)) return;
    views.set(id, { rendered: view.rendered, anchors: { ...view.anchors } });
    if (saveTimer === undefined) saveTimer = setTimeout(() => save(), SAVE_DELAY_MS);
  },
  dismissNotice() {
    useFileTabsStore.setState({ notice: null });
  },
};

function isOpen(id: string) {
  return Object.values(useFileTabsStore.getState().owners).some((owner) => owner.tabs.some((tab) => tab.id === id));
}

loadFileTabs();
if (typeof window !== "undefined") window.addEventListener("pagehide", flushFileTabs);
