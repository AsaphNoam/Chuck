import { create } from "zustand";
import type { NotificationPayload } from "../api/types";
import { notificationAgentId } from "../lib/agentConversation";

export interface ToastItem {
  id: string;
  title: string;
  body?: string;
  // "notice" is client-side, success-adjacent feedback from a mutation response
  // (e.g. a recreated worktree checkout, FS-19.R7).
  type: NotificationPayload["notification_type"] | "error" | "notice";
  // The agent whose conversation the toast opens; absent on error toasts.
  agentId?: string;
  /** Lifetime before it fades out; the default is 6s. */
  durationMs?: number;
}

/** Send/Steer outcome notices are brief confirmations, not standing state (FS-03.R50). */
export const OUTCOME_TOAST_MS = 3_000;

const ATTENTION_TOASTS = new Set<ToastItem["type"]>(["waiting_input", "permission_required"]);

interface UiStoreState {
  density: { perRow: number; gap: number };
  groupLayout: Record<string, { collapsed: boolean }>;
  /** Per-project collapse overrides; a missing entry falls back to groupLayout (FS-02.R82). */
  projectGroupLayout: Record<string, Record<string, { collapsed: boolean }>>;
  connection: "connecting" | "open" | "reconnecting" | "down";
  contextMenu: { agentId: string; x: number; y: number } | null;
  toasts: ToastItem[];
  setDensity: (density: { perRow: number; gap: number }) => void;
  setGroupLayout: (groups: Record<string, { collapsed: boolean }>, projectGroups?: Record<string, Record<string, { collapsed: boolean }>>) => void;
  /** Toggles one project's choice, or the shared choice where no project scopes the grid. */
  toggleGroupCollapsed: (group: string, project?: string) => void;
  setConnection: (connection: UiStoreState["connection"]) => void;
  openContextMenu: (agentId: string, x: number, y: number) => void;
  closeContextMenu: () => void;
  pushToast: (notification: NotificationPayload) => void;
  /** Drops an agent's waiting/permission toasts once it no longer waits. */
  dismissAttention: (agentId: string) => void;
  pushError: (title: string, body?: string) => void;
  pushNotice: (title: string, body?: string, agentId?: string, durationMs?: number) => void;
  dismissToast: (id: string) => void;
}

/** A project's own choice wins; otherwise the legacy shared choice applies (FS-02.R82). */
export function groupCollapsed(state: Pick<UiStoreState, "groupLayout" | "projectGroupLayout">, group: string, project?: string): boolean {
  return (project ? state.projectGroupLayout[project]?.[group]?.collapsed : undefined) ?? state.groupLayout[group]?.collapsed ?? false;
}

export const useUiStore = create<UiStoreState>((set) => ({
  density: { perRow: 3, gap: 16 },
  groupLayout: {},
  projectGroupLayout: {},
  connection: "connecting",
  contextMenu: null,
  toasts: [],
  setDensity: (density) => set({ density }),
  setGroupLayout: (groupLayout, projectGroupLayout = {}) => set({ groupLayout, projectGroupLayout }),
  toggleGroupCollapsed: (group, project) =>
    set((state) => {
      const collapsed = !groupCollapsed(state, group, project);
      if (!project) return { groupLayout: { ...state.groupLayout, [group]: { collapsed } } };
      return {
        projectGroupLayout: {
          ...state.projectGroupLayout,
          [project]: { ...state.projectGroupLayout[project], [group]: { collapsed } },
        },
      };
    }),
  setConnection: (connection) => set({ connection }),
  openContextMenu: (agentId, x, y) => set({ contextMenu: { agentId, x, y } }),
  closeContextMenu: () => set({ contextMenu: null }),
  pushToast: (notification) =>
    set((state) => {
      const id = `${notification.agent_id}-${notification.notification_type}-${notification.ts}`;
      const agentId = notificationAgentId(notification);
      const type = notification.notification_type;
      // A permission request also moves its agent into waiting_input; the
      // person sees one toast for it, the one naming the permission.
      if (type === "waiting_input" && agentId && state.toasts.some((toast) => toast.agentId === agentId && toast.type === "permission_required")) return {};
      return {
        toasts: [
          ...state.toasts.filter((toast) => toast.id !== id && !(type === "permission_required" && toast.agentId === agentId && toast.type === "waiting_input")),
          { id, title: notification.title, body: notification.body, type, agentId },
        ].slice(-4),
      };
    }),
  dismissAttention: (agentId) =>
    set((state) => {
      const toasts = state.toasts.filter((toast) => !(toast.agentId === agentId && ATTENTION_TOASTS.has(toast.type)));
      return toasts.length === state.toasts.length ? {} : { toasts };
    }),
  pushError: (title, body) =>
    set((state) => {
      const toast: ToastItem = {
        id: `error-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        title,
        body,
        type: "error",
      };
      return { toasts: [...state.toasts, toast].slice(-4) };
    }),
  pushNotice: (title, body, agentId, durationMs) =>
    set((state) => {
      const toast: ToastItem = {
        id: `notice-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        title,
        body,
        type: "notice",
        agentId,
        durationMs,
      };
      return { toasts: [...state.toasts, toast].slice(-4) };
    }),
  dismissToast: (id) => set((state) => ({ toasts: state.toasts.filter((toast) => toast.id !== id) })),
}));
