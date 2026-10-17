import { create } from "zustand";
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
  afterSeqByAgent: Record<string, number>;
  attachmentsByAgent: Record<string, AttachmentDescriptor[]>;
  recoveryByAgent: Record<string, { text: string; request_id: string; attachments: AttachmentDescriptor[] }>;
  // A withdrawal succeeded but its recovery offer could not be read yet; the tail
  // says so and offers to read it again rather than showing nothing (FS-03.R90).
  recoveryUnreadByAgent: Record<string, true>;
  hold: (agentId: string, text: string, afterSeq?: number, attachments?: AttachmentDescriptor[]) => void;
  release: (agentId: string) => void;
  setRecovery: (agentId: string, recovery: { text: string; request_id: string; attachments: AttachmentDescriptor[] } | null) => void;
  markRecoveryUnread: (agentId: string) => void;
}

export const useHeldStore = create<HeldStoreState>((set) => ({
  byAgent: {},
  afterSeqByAgent: {},
  attachmentsByAgent: {},
  recoveryByAgent: {},
  recoveryUnreadByAgent: {},
  hold: (agentId, text, afterSeq = 0, attachments = []) =>
    set((state) => ({
      byAgent: { ...state.byAgent, [agentId]: text },
      afterSeqByAgent: { ...state.afterSeqByAgent, [agentId]: afterSeq },
      attachmentsByAgent: { ...state.attachmentsByAgent, [agentId]: attachments },
    })),
  release: (agentId) =>
    set((state) => {
      if (!(agentId in state.byAgent)) return state;
      const byAgent = { ...state.byAgent };
      const afterSeqByAgent = { ...state.afterSeqByAgent };
      const attachmentsByAgent = { ...state.attachmentsByAgent };
      delete byAgent[agentId];
      delete afterSeqByAgent[agentId];
      delete attachmentsByAgent[agentId];
      return { byAgent, afterSeqByAgent, attachmentsByAgent };
    }),
  setRecovery: (agentId, recovery) => set((state) => {
    const recoveryByAgent = { ...state.recoveryByAgent };
    if (recovery) recoveryByAgent[agentId] = recovery;
    else delete recoveryByAgent[agentId];
    const recoveryUnreadByAgent = { ...state.recoveryUnreadByAgent };
    delete recoveryUnreadByAgent[agentId];
    return { recoveryByAgent, recoveryUnreadByAgent };
  }),
  markRecoveryUnread: (agentId) => set((state) => ({ recoveryUnreadByAgent: { ...state.recoveryUnreadByAgent, [agentId]: true } })),
}));
