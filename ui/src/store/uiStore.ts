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
}

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
  pushError: (title: string, body?: string) => void;
  pushNotice: (title: string, body?: string, agentId?: string) => void;
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
      return {
        toasts: [
          ...state.toasts.filter((toast) => toast.id !== id),
          {
            id,
            title: notification.title,
            body: notification.body,
            type: notification.notification_type,
            agentId: notificationAgentId(notification),
          },
        ].slice(-4),
      };
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
  pushNotice: (title, body, agentId) =>
    set((state) => {
      const toast: ToastItem = {
        id: `notice-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        title,
        body,
        type: "notice",
        agentId,
      };
      return { toasts: [...state.toasts, toast].slice(-4) };
    }),
  dismissToast: (id) => set((state) => ({ toasts: state.toasts.filter((toast) => toast.id !== id) })),
}));
