import { create } from "zustand";
import type { QueuedPrompt } from "../api/client";
import type { AttachmentDescriptor } from "../api/types";

// The queued follow-up (FS-03.R48) is live state on both sides. The runtime holds
// at most one per agent and it dies with the process, so this mirror is memory
// only: a browser reload rehydrates it from the runtime's live snapshot, while a
// dashboard restart correctly clears it with the runtime process.
//
// It is deliberately not merged into the transcript event list (TS-08.R56): the
// server sends no event for a message it has not delivered, so keeping it beside
// the list is what makes it structurally impossible for a live render and a
// reload to disagree about it.
interface HeldStoreState {
  byAgent: Record<string, string>;
  heldIdByAgent: Record<string, string>;
  requestIdByAgent: Record<string, string | undefined>;
  afterSeqByAgent: Record<string, number>;
  attachmentsByAgent: Record<string, AttachmentDescriptor[]>;
  recoveryByAgent: Record<string, { text: string; request_id: string; attachments?: AttachmentDescriptor[]; delivery?: string }>;
  // A withdrawal succeeded but its recovery offer could not be read yet; the tail
  // says so and offers to read it again rather than showing nothing (FS-03.R90).
  // The value counts failed reads so a failed retry shows a new outcome.
  recoveryUnreadByAgent: Record<string, number>;
  // Counts locally accepted holds so a snapshot read begun before one cannot
  // overwrite it with an older hold or an earlier empty state (INV §23).
  // Snapshot installs and releases do not count: the last snapshot still wins
  // among reads, and a re-installed delivered hold is released again by its match.
  holdRevisionByAgent: Record<string, number>;
  acceptHold: (agentId: string, text: string, afterSeq?: number, attachments?: AttachmentDescriptor[], heldId?: string, requestId?: string) => void;
  hold: (agentId: string, text: string, afterSeq?: number, attachments?: AttachmentDescriptor[], heldId?: string, requestId?: string) => void;
  release: (agentId: string) => void;
  setRecovery: (agentId: string, recovery: { text: string; request_id: string; attachments?: AttachmentDescriptor[]; delivery?: string } | null) => void;
  markRecoveryUnread: (agentId: string) => void;
  queuedByAgent: Record<string, QueuedPrompt[]>;
  setQueued: (agentId: string, queued: QueuedPrompt[]) => void;
}

export const useHeldStore = create<HeldStoreState>((set, get) => ({
  byAgent: {},
  heldIdByAgent: {},
  requestIdByAgent: {},
  afterSeqByAgent: {},
  attachmentsByAgent: {},
  recoveryByAgent: {},
  recoveryUnreadByAgent: {},
  holdRevisionByAgent: {},
  queuedByAgent: {},
  setQueued: (agentId, queued) => set((state) => ({ queuedByAgent: { ...state.queuedByAgent, [agentId]: queued } })),
  acceptHold: (agentId, ...held) => {
    set((state) => ({ holdRevisionByAgent: { ...state.holdRevisionByAgent, [agentId]: (state.holdRevisionByAgent[agentId] ?? 0) + 1 } }));
    get().hold(agentId, ...held);
  },
  hold: (agentId, text, afterSeq = 0, attachments = [], heldId = "", requestId) =>
    set((state) => ({
      byAgent: { ...state.byAgent, [agentId]: text },
      heldIdByAgent: { ...state.heldIdByAgent, [agentId]: heldId },
      requestIdByAgent: { ...state.requestIdByAgent, [agentId]: requestId },
      afterSeqByAgent: { ...state.afterSeqByAgent, [agentId]: afterSeq },
      attachmentsByAgent: { ...state.attachmentsByAgent, [agentId]: attachments },
    })),
  release: (agentId) =>
    set((state) => {
      if (!(agentId in state.byAgent)) return state;
      const byAgent = { ...state.byAgent };
      const heldIdByAgent = { ...state.heldIdByAgent };
      const requestIdByAgent = { ...state.requestIdByAgent };
      const afterSeqByAgent = { ...state.afterSeqByAgent };
      const attachmentsByAgent = { ...state.attachmentsByAgent };
      delete byAgent[agentId];
      delete heldIdByAgent[agentId];
      delete requestIdByAgent[agentId];
      delete afterSeqByAgent[agentId];
      delete attachmentsByAgent[agentId];
      return { byAgent, heldIdByAgent, requestIdByAgent, afterSeqByAgent, attachmentsByAgent };
    }),
  setRecovery: (agentId, recovery) => set((state) => {
    const recoveryByAgent = { ...state.recoveryByAgent };
    if (recovery) recoveryByAgent[agentId] = recovery;
    else delete recoveryByAgent[agentId];
    const recoveryUnreadByAgent = { ...state.recoveryUnreadByAgent };
    delete recoveryUnreadByAgent[agentId];
    return { recoveryByAgent, recoveryUnreadByAgent };
  }),
  markRecoveryUnread: (agentId) => set((state) => ({ recoveryUnreadByAgent: { ...state.recoveryUnreadByAgent, [agentId]: (state.recoveryUnreadByAgent[agentId] ?? 0) + 1 } })),
}));
