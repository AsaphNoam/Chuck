import { useEffect, useRef, useState } from "react";
import { useLocation, useNavigationType, useSearchParams } from "react-router-dom";
import { findTab, type FileTab, type FileTarget, type OwnerTabs } from "./model";
import { fileTabs, useFileTabsStore } from "./store";
import { useUiStore } from "../../store/uiStore";

// A route's saved file tabs and its address (TS-08.R143). The tab set is the
// owner's saved record; the address names only the active tab, so a file URL
// still opens its file, a reload restores the set, and Back to a file-free
// address hides the reader without closing a tab.

export type FileTabKeys = { file: string; line: string; source?: string };
export const CONVERSATION_FILE_KEYS: FileTabKeys = { file: "file", line: "fileLine" };
export const ROOM_FILE_KEYS: FileTabKeys = { file: "file", line: "line", source: "source" };

/** Marks an address this hook wrote, so it is not read back as a new request. */
const PROJECTED = "chuckFileTabs";

export type FileTabRoute = ReturnType<typeof useFileTabRoute>;

export function readFileTarget(params: URLSearchParams, keys: FileTabKeys): FileTarget | null {
  const path = params.get(keys.file);
  if (!path) return null;
  const target: FileTarget = { path };
  const line = Number(params.get(keys.line));
  if (Number.isInteger(line) && line > 0) target.line = line;
  if (keys.source) target.source = params.get(keys.source) ?? "";
  return target;
}

export function writeFileTarget(params: URLSearchParams, keys: FileTabKeys, tab: FileTarget | null | undefined): URLSearchParams {
  const next = new URLSearchParams(params);
  next.delete(keys.file);
  next.delete(keys.line);
  if (keys.source) next.delete(keys.source);
  if (tab) {
    next.set(keys.file, tab.path);
    if (keys.source) next.set(keys.source, tab.source ?? "");
    if (tab.line) next.set(keys.line, String(tab.line));
  }
  return next;
}

const activeOf = (owner: OwnerTabs | undefined) => owner?.tabs.find((tab) => tab.id === owner.active);

export function useFileTabRoute(owner: string, keys: FileTabKeys, options: { onOpen?: () => void } = {}) {
  const [params, setParams] = useSearchParams();
  const location = useLocation();
  const navigation = useNavigationType();
  const record = useFileTabsStore((state) => state.owners[owner]);
  // Hidden by Back to this owner's file-free address; it belongs to this route
  // instance and ends when the owner changes.
  const [hidden, setHidden] = useState(false);
  // A navigation command per tab: changing it shows the tab's cited line once.
  const [jumps, setJumps] = useState<Record<string, number>>({});
  const seenOwner = useRef<string | null>(null);
  // Nothing is shown until this owner's address has been applied, so an
  // explicit file URL never first mounts, and reads, the saved active tab.
  const [applied, setApplied] = useState<string | null>(null);
  const onOpenRef = useRef(options.onOpen);
  onOpenRef.current = options.onOpen;

  const project = (tab: FileTab | null | undefined, push = false) =>
    setParams((current) => writeFileTarget(current, keys, tab), { replace: !push, state: { [PROJECTED]: true } });
  const jump = (id: string) => setJumps((current) => ({ ...current, [id]: (current[id] ?? 0) + 1 }));

  useEffect(() => {
    const fresh = seenOwner.current !== owner;
    seenOwner.current = owner;
    if (fresh) setHidden(false);
    setApplied(owner);
    const projected = Boolean((location.state as Record<string, unknown> | null)?.[PROJECTED]);
    if (!fresh && navigation !== "POP" && projected) return;
    const target = readFileTarget(params, keys);
    const saved = useFileTabsStore.getState().owners[owner];
    // A room file URL must name its retained source; none is borrowed (TS-08.R143).
    if (target && keys.source && !target.source) {
      useUiStore.getState().pushError("That file link does not say which participant's workspace it is from.");
      project(activeOf(saved));
      return;
    }
    if (target) {
      const before = findTab(saved, target);
      const opened = fileTabs.open(owner, target);
      if (!opened) {
        project(activeOf(useFileTabsStore.getState().owners[owner]));
        return;
      }
      setHidden(false);
      if (target.line && (opened.added || before?.line !== target.line)) jump(opened.tab.id);
      return;
    }
    if (!fresh && navigation === "POP") {
      setHidden(true);
      return;
    }
    const restored = activeOf(saved);
    if (restored) {
      setHidden(false);
      project(restored);
    }
    // Only the address and owner decide; the handlers below write both.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.key, owner]);

  const urlHasFile = params.get(keys.file) !== null;
  const shown = applied === owner && Boolean(record?.active) && !hidden;

  return {
    owner,
    tabs: record?.tabs ?? [],
    active: shown ? activeOf(record) ?? null : null,
    shown,
    jumps,
    /** An explicit open from a link, a Files row or a diff. */
    open(target: FileTarget) {
      const opened = fileTabs.open(owner, target);
      if (!opened) return;
      onOpenRef.current?.();
      if (target.line) jump(opened.tab.id);
      setHidden(false);
      // The first open from a closed reader is the one history entry (FS-03.R106).
      project(opened.tab, !urlHasFile || hidden);
    },
    select(id: string) {
      fileTabs.select(owner, id);
      project(activeOf(useFileTabsStore.getState().owners[owner]));
    },
    close(id: string) {
      const wasActive = useFileTabsStore.getState().owners[owner]?.active === id;
      fileTabs.close(owner, id);
      if (wasActive) project(activeOf(useFileTabsStore.getState().owners[owner]));
    },
    move(id: string, to: number) {
      fileTabs.move(owner, id, to);
    },
    resolved(id: string, path: string) {
      const kept = fileTabs.resolve(owner, id, path);
      const current = useFileTabsStore.getState().owners[owner];
      if (kept && current?.active === kept) project(activeOf(current));
    },
  };
}
