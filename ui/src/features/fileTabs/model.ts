// The saved file-tab model shared by every reader surface (FS-03.R101–R106,
// TS-08.R140/R141/R144). It is pure: the store owns persistence and the reader
// hosts own file contents, so nothing here can read a file or grant access to one.

export type DisplayMode = "rendered" | "formatted" | "source";
/** A reading location: the first visible source line or rendered top-level
 *  block, and how far its top sits below the body's top edge (TS-08.R130). */
export type ReadingAnchor = { key: number; offset: number };
export type Anchors = Record<DisplayMode, ReadingAnchor | null>;

export type OwnerKind = "conversation" | "room";
/** A conversation (desktop live/archive and phone alike) or a room id. */
export type OwnerKey = `${OwnerKind}:${string}`;

/** One open file. `source` is the room's retained-source id and is present only
 *  for room tabs; `line` is the file's last cited line. */
export type FileTab = { id: string; path: string; line?: number; source?: string };
export type TabView = { rendered: boolean; anchors: Anchors };
export type OwnerTabs = { tabs: FileTab[]; active: string | null };
export type Owners = Record<string, OwnerTabs>;
export type FileTarget = { path: string; line?: number; source?: string };

export const MAX_TABS_PER_OWNER = 50;
export const MAX_SAVED_TABS = 500;
/** 2 MiB of UTF-16 localStorage (TS-08.R144). */
export const MAX_SAVED_UNITS = 1_048_576;
export const STORAGE_KEY = "chuck.file-tabs.v1";

export const emptyAnchors = (): Anchors => ({ rendered: null, formatted: null, source: null });
export const defaultView = (): TabView => ({ rendered: true, anchors: emptyAnchors() });

export function ownerKey(kind: OwnerKind, id: string): OwnerKey {
  return `${kind}:${id}`;
}

// pathKey compares path spellings, not files: `./a//b.md` and `a/b.md` are one
// tab, while a symlink or an absolute spelling of the same file may stay a second
// tab (FS-03.R101). `..` is kept because collapsing it could change which file
// the server reads.
export function pathKey(path: string): string {
  const parts = path.split("/").filter((segment) => segment !== "" && segment !== ".");
  return `${path.startsWith("/") ? "/" : ""}${parts.join("/")}`;
}

function sameFile(tab: FileTab, target: FileTarget) {
  return (tab.source ?? "") === (target.source ?? "") && pathKey(tab.path) === pathKey(target.path);
}

export function findTab(owner: OwnerTabs | undefined, target: FileTarget): FileTab | undefined {
  return owner?.tabs.find((tab) => sameFile(tab, target));
}

export function totalTabs(owners: Owners): number {
  return Object.values(owners).reduce((sum, owner) => sum + owner.tabs.length, 0);
}

export type OpenResult =
  | { owners: Owners; tab: FileTab; added: boolean }
  | { refused: "owner" | "total" };

// openTab selects an existing tab for the same file or admits a new one at the
// end. Limits are checked before anything is added, and nothing is ever evicted
// to make room (FS-03.R105).
export function openTab(owners: Owners, key: string, target: FileTarget, newId: string): OpenResult {
  const owner = owners[key] ?? { tabs: [], active: null };
  const existing = findTab(owner, target);
  if (existing) {
    const tab = target.line !== undefined && target.line !== existing.line ? { ...existing, line: target.line } : existing;
    const tabs = tab === existing ? owner.tabs : owner.tabs.map((t) => (t.id === tab.id ? tab : t));
    return { owners: { ...owners, [key]: { tabs, active: tab.id } }, tab, added: false };
  }
  if (owner.tabs.length >= MAX_TABS_PER_OWNER) return { refused: "owner" };
  if (totalTabs(owners) >= MAX_SAVED_TABS) return { refused: "total" };
  const tab: FileTab = { id: newId, path: target.path };
  if (target.line !== undefined) tab.line = target.line;
  if (target.source !== undefined) tab.source = target.source;
  return { owners: { ...owners, [key]: { tabs: [...owner.tabs, tab], active: tab.id } }, tab, added: true };
}

export function selectTab(owners: Owners, key: string, id: string): Owners {
  const owner = owners[key];
  if (!owner || owner.active === id || !owner.tabs.some((tab) => tab.id === id)) return owners;
  return { ...owners, [key]: { ...owner, active: id } };
}

// closeTab selects the right neighbour of a closed active tab, or its left one at
// the end; a background close leaves the active file alone. The last close ends
// the owner's record (FS-03.R102, TS-08.R143).
export function closeTab(owners: Owners, key: string, id: string): Owners {
  const owner = owners[key];
  const index = owner?.tabs.findIndex((tab) => tab.id === id) ?? -1;
  if (!owner || index < 0) return owners;
  const tabs = owner.tabs.filter((tab) => tab.id !== id);
  if (tabs.length === 0) {
    const { [key]: _closed, ...rest } = owners;
    return rest;
  }
  const active = owner.active !== id ? owner.active : (tabs[index] ?? tabs[index - 1]).id;
  return { ...owners, [key]: { tabs, active } };
}

export function moveTab(owners: Owners, key: string, id: string, to: number): Owners {
  const owner = owners[key];
  const from = owner?.tabs.findIndex((tab) => tab.id === id) ?? -1;
  if (!owner || from < 0) return owners;
  const target = Math.max(0, Math.min(owner.tabs.length - 1, to));
  if (target === from) return owners;
  const tabs = [...owner.tabs];
  const [tab] = tabs.splice(from, 1);
  tabs.splice(target, 0, tab);
  return { ...owners, [key]: { ...owner, tabs } };
}

// resolveTab records the normalized path a read returned for the tab that read
// it. When that spelling names a file another tab already has, the existing tab
// keeps its identity and state and the new duplicate merges away, taking the
// selection with it only when it was selected (TS-08.R142).
export function resolveTab(owners: Owners, key: string, id: string, path: string): { owners: Owners; kept: string | null } {
  const owner = owners[key];
  const tab = owner?.tabs.find((t) => t.id === id);
  if (!owner || !tab) return { owners, kept: null };
  if (tab.path === path) return { owners, kept: id };
  const duplicate = owner.tabs.find((t) => t.id !== id && sameFile(t, { path, source: tab.source }));
  if (duplicate) {
    const tabs = owner.tabs.filter((t) => t.id !== id);
    return { owners: { ...owners, [key]: { tabs, active: owner.active === id ? duplicate.id : owner.active } }, kept: duplicate.id };
  }
  return { owners: { ...owners, [key]: { ...owner, tabs: owner.tabs.map((t) => (t.id === id ? { ...t, path } : t)) } }, kept: id };
}

// --- Saved record codec (TS-08.R141) ---------------------------------------

type SavedTab = FileTab & TabView;
type SavedRecord = { v: 1; owners: Record<string, { tabs: SavedTab[]; active: string | null }> };

export function encodeTabs(owners: Owners, views: ReadonlyMap<string, TabView>): string {
  const record: SavedRecord = { v: 1, owners: {} };
  for (const [key, owner] of Object.entries(owners)) {
    record.owners[key] = {
      active: owner.active,
      tabs: owner.tabs.map((tab) => ({ ...tab, ...(views.get(tab.id) ?? defaultView()) })),
    };
  }
  return JSON.stringify(record);
}

export type Decoded = { owners: Owners; views: Map<string, TabView> };

/** Thrown for saved data that cannot be used; its message is shown to the reader. */
export class SavedTabsError extends Error {}

const ANCHOR_LIMIT = 1e7;
const OWNER_KEY = /^(conversation|room):.+$/;
const MODES: DisplayMode[] = ["rendered", "formatted", "source"];

// decodeTabs validates everything before it is used: saved data names files to
// read again through the surface's own authority and can never supply a loader
// or a room source a route did not already have (TS-08.R141/R144).
export function decodeTabs(raw: string | null): Decoded {
  const empty: Decoded = { owners: {}, views: new Map() };
  if (raw === null) return empty;
  if (raw.length > MAX_SAVED_UNITS) throw new SavedTabsError("Saved file tabs are larger than this browser keeps.");
  let parsed: unknown;
  try { parsed = JSON.parse(raw); }
  catch { throw new SavedTabsError("Saved file tabs could not be read."); }
  if (!isObject(parsed) || parsed.v !== 1 || !isObject(parsed.owners)) throw new SavedTabsError("Saved file tabs are from an unknown version.");
  const entries = Object.entries(parsed.owners);
  let total = 0;
  for (const [, owner] of entries) {
    if (!isObject(owner) || !Array.isArray(owner.tabs)) throw invalid();
    total += owner.tabs.length;
    if (owner.tabs.length > MAX_TABS_PER_OWNER || total > MAX_SAVED_TABS) throw invalid();
  }
  const ids = new Set<string>();
  for (const [key, owner] of entries as [string, { tabs: unknown[]; active: unknown }][]) {
    if (!OWNER_KEY.test(key) || owner.tabs.length === 0) throw invalid();
    const room = key.startsWith("room:");
    const tabs: FileTab[] = [];
    for (const saved of owner.tabs) {
      if (!isObject(saved) || !nonEmpty(saved.id) || ids.has(saved.id) || !nonEmpty(saved.path)) throw invalid();
      if (saved.line !== undefined && !(Number.isInteger(saved.line) && (saved.line as number) > 0)) throw invalid();
      if (room ? !nonEmpty(saved.source) : saved.source !== undefined) throw invalid();
      if (typeof saved.rendered !== "boolean" || !isObject(saved.anchors)) throw invalid();
      const anchors = emptyAnchors();
      for (const mode of MODES) {
        const anchor = saved.anchors[mode];
        if (anchor === null || anchor === undefined) continue;
        if (!isObject(anchor) || !bounded(anchor.key) || !bounded(anchor.offset)) throw invalid();
        anchors[mode] = { key: anchor.key as number, offset: anchor.offset as number };
      }
      ids.add(saved.id);
      const tab: FileTab = { id: saved.id, path: saved.path };
      if (saved.line !== undefined) tab.line = saved.line as number;
      if (room) tab.source = saved.source as string;
      tabs.push(tab);
      empty.views.set(tab.id, { rendered: saved.rendered, anchors });
    }
    const active = owner.active;
    if (active !== null && !(typeof active === "string" && tabs.some((tab) => tab.id === active))) throw invalid();
    empty.owners[key] = { tabs, active: active ?? tabs[0].id };
  }
  return empty;
}

function invalid() {
  return new SavedTabsError("Saved file tabs were damaged.");
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function nonEmpty(value: unknown): value is string {
  return typeof value === "string" && value.length > 0;
}

function bounded(value: unknown) {
  return typeof value === "number" && Number.isFinite(value) && Math.abs(value) <= ANCHOR_LIMIT;
}
