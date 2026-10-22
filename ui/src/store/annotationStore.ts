import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { AnnotationDraft } from "../api/types";

const maxDrafts = 20;
// Retention for stored trays (FS-13.R16). Nothing on the server owns this state,
// so without a bound an abandoned tray lives forever and the trays of every
// agent a browser ever opened keep consuming the shared localStorage quota —
// once it is full, zustand's persist throws on every setState.
const maxSources = 20;
const maxTrayAgeMs = 30 * 24 * 60 * 60 * 1000;

interface PersistedTrays {
  bySource: Record<string, AnnotationDraft[]>;
  overallBySource: Record<string, string>;
  editedAt: Record<string, number>;
  // Whether the docked tray is reduced to its strip (FS-13.R21). It rides this
  // record rather than a second browser key so it inherits the tray's expiry,
  // cap, and delete-with-agent path instead of outliving the drafts it belongs
  // to. Only the docked form exposes the control; the overlay ignores the flag.
  collapsedBySource: Record<string, boolean>;
  commandBySource: Record<string, string>;
  launchedBySource: Record<string, string>;
  targetBySource: Record<string, string>;
  recipientBySource: Record<string, string>;
}

interface AnnotationStoreState extends PersistedTrays {
  add: (sourceId: string, draft: AnnotationDraft) => boolean;
  updateInstruction: (sourceId: string, index: number, instruction: string) => void;
  remove: (sourceId: string, index: number) => void;
  discard: (sourceId: string) => void;
  discardSent: (sourceId: string, sent: AnnotationDraft[]) => void;
  setOverall: (sourceId: string, instruction: string) => void;
  setCollapsed: (sourceId: string, collapsed: boolean) => void;
  ensureCommand: (sourceId: string) => string;
  resetCommand: (sourceId: string) => void;
  setLaunched: (sourceId: string, agentId: string | null) => void;
  setDeliveryTarget: (sourceId: string, target: string, recipientId: string) => void;
}

// pruneTrays applies FS-13.R16 to what a reload restored: drop empty trays, drop
// trays untouched for maxTrayAgeMs, and keep only the most recently edited
// maxSources. A tray stored by an older build carries no timestamp; treat it as
// touched now so an upgrade never discards a draft the user is still working on.
export function pruneTrays(persisted: Partial<PersistedTrays>, now = Date.now()): PersistedTrays {
  const stored = persisted.bySource ?? {};
  const editedAt = persisted.editedAt ?? {};
  const keep = Object.keys(stored)
    .filter((id) => (stored[id] ?? []).length > 0 && now - (editedAt[id] ?? now) <= maxTrayAgeMs)
    .sort((a, b) => (editedAt[b] ?? now) - (editedAt[a] ?? now))
    .slice(0, maxSources);
  const next: PersistedTrays = { bySource: {}, overallBySource: {}, editedAt: {}, collapsedBySource: {}, commandBySource: {}, launchedBySource: {}, targetBySource: {}, recipientBySource: {} };
  for (const id of keep) {
    next.bySource[id] = stored[id];
    next.editedAt[id] = editedAt[id] ?? now;
    const overall = persisted.overallBySource?.[id];
    if (overall) next.overallBySource[id] = overall;
    if (persisted.collapsedBySource?.[id]) next.collapsedBySource[id] = true;
    if (persisted.commandBySource?.[id]) next.commandBySource[id] = persisted.commandBySource[id];
    if (persisted.launchedBySource?.[id]) next.launchedBySource[id] = persisted.launchedBySource[id];
    if (persisted.targetBySource?.[id]) next.targetBySource[id] = persisted.targetBySource[id];
    if (persisted.recipientBySource?.[id]) next.recipientBySource[id] = persisted.recipientBySource[id];
  }
  return next;
}

export const useAnnotationStore = create<AnnotationStoreState>()(
  persist(
    (set, get) => ({
      bySource: {},
      overallBySource: {},
      editedAt: {},
      collapsedBySource: {},
      commandBySource: {},
      launchedBySource: {},
      targetBySource: {},
      recipientBySource: {},
      add: (sourceId, draft) => {
        const current = get().bySource[sourceId] ?? [];
        if (current.length >= maxDrafts) return false;
        set({ bySource: { ...get().bySource, [sourceId]: [...current, draft] }, editedAt: touch(get().editedAt, sourceId), commandBySource: without(get().commandBySource, sourceId) });
        return true;
      },
      updateInstruction: (sourceId, index, instruction) =>
        set((state) => ({
          bySource: {
            ...state.bySource,
            [sourceId]: (state.bySource[sourceId] ?? []).map((draft, i) => i === index ? { ...draft, instruction } : draft),
          },
          editedAt: touch(state.editedAt, sourceId),
          commandBySource: without(state.commandBySource, sourceId),
        })),
      // Removing the last draft empties the tray, so it ends the whole tray
      // record the same way discarding does (FS-13.R21): a collapsed flag left
      // behind would collapse the next tray this source opens.
      remove: (sourceId, index) =>
        set((state) => {
          const remaining = (state.bySource[sourceId] ?? []).filter((_, i) => i !== index);
          if (remaining.length === 0) return dropSource(state, sourceId);
          return { bySource: { ...state.bySource, [sourceId]: remaining }, editedAt: touch(state.editedAt, sourceId), commandBySource: without(state.commandBySource, sourceId) };
        }),
      discard: (sourceId) => set((state) => dropSource(state, sourceId)),
      // An acknowledged send clears only the drafts it submitted (by identity;
      // sending disables edits), so a capture made while it was pending stays.
      // The overall instruction went with the batch.
      discardSent: (sourceId, sent) =>
        set((state) => {
          const remaining = (state.bySource[sourceId] ?? []).filter((draft) => !sent.includes(draft));
          if (remaining.length === 0) return dropSource(state, sourceId);
          const { [sourceId]: _overall, ...overallBySource } = state.overallBySource;
          return { bySource: { ...state.bySource, [sourceId]: remaining }, overallBySource, commandBySource: without(state.commandBySource, sourceId), launchedBySource: without(state.launchedBySource, sourceId) };
        }),
      setOverall: (sourceId, overall) =>
        set((state) => ({ overallBySource: { ...state.overallBySource, [sourceId]: overall }, editedAt: touch(state.editedAt, sourceId), commandBySource: without(state.commandBySource, sourceId) })),
      // Collapsing hides drafts; it does not edit them, so it deliberately
      // leaves editedAt alone and lets the tray expire on its real last edit.
      setCollapsed: (sourceId, collapsed) =>
        set((state) => ({ collapsedBySource: { ...state.collapsedBySource, [sourceId]: collapsed } })),
      ensureCommand: (sourceId) => {
        const existing = get().commandBySource[sourceId];
        if (existing) return existing;
        const command = typeof crypto?.randomUUID === "function" ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`;
        set((state) => ({ commandBySource: { ...state.commandBySource, [sourceId]: command } }));
        return command;
      },
      resetCommand: (sourceId) => set((state) => ({ commandBySource: without(state.commandBySource, sourceId) })),
      setLaunched: (sourceId, agentId) => set((state) => ({ launchedBySource: agentId ? { ...state.launchedBySource, [sourceId]: agentId } : without(state.launchedBySource, sourceId) })),
      setDeliveryTarget: (sourceId, target, recipientId) => set((state) => ({ targetBySource: { ...state.targetBySource, [sourceId]: target }, recipientBySource: recipientId ? { ...state.recipientBySource, [sourceId]: recipientId } : without(state.recipientBySource, sourceId) })),
    }),
    {
      name: "chuck-annotation-tray",
      partialize: (state) => ({ bySource: state.bySource, overallBySource: state.overallBySource, editedAt: state.editedAt, collapsedBySource: state.collapsedBySource, commandBySource: state.commandBySource, launchedBySource: state.launchedBySource, targetBySource: state.targetBySource, recipientBySource: state.recipientBySource }),
      merge: (persisted, current) => ({ ...current, ...pruneTrays((persisted ?? {}) as Partial<PersistedTrays>) }),
    },
  ),
);

// dropSource ends one source's tray. Every per-source record goes together so
// nothing — drafts, overall instruction, timestamp, collapsed flag — outlives
// the tray it belongs to (FS-13.R16, FS-13.R21).
function dropSource(state: PersistedTrays, sourceId: string): PersistedTrays {
  const { [sourceId]: _drafts, ...bySource } = state.bySource;
  const { [sourceId]: _overall, ...overallBySource } = state.overallBySource;
  const { [sourceId]: _edited, ...editedAt } = state.editedAt;
  const { [sourceId]: _collapsed, ...collapsedBySource } = state.collapsedBySource;
  const { [sourceId]: _command, ...commandBySource } = state.commandBySource;
  const { [sourceId]: _launched, ...launchedBySource } = state.launchedBySource;
  const { [sourceId]: _target, ...targetBySource } = state.targetBySource;
  const { [sourceId]: _recipient, ...recipientBySource } = state.recipientBySource;
  return { bySource, overallBySource, editedAt, collapsedBySource, commandBySource, launchedBySource, targetBySource, recipientBySource };
}

function without<T>(values: Record<string, T>, sourceId: string) {
  const { [sourceId]: _removed, ...rest } = values;
  return rest;
}

function touch(editedAt: Record<string, number>, sourceId: string) {
  return { ...editedAt, [sourceId]: Date.now() };
}
