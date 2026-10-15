import { create } from "zustand";
import type { AgentState } from "../api/types";

interface AgentStoreState {
  agents: Record<string, AgentState>;
  /** observed indexes each agent by when this client last applied an update for it.
   *  `updated_at` is a millisecond wall clock, so a burst of updates can share one
   *  value and leave a consumer sorting by it with no order at all; this is the
   *  order the client actually saw them in (TS-08.R49). It lives beside `agents`
   *  so every path that drops an agent drops its index with it (INV §1). */
  observed: Record<string, number>;
  observedSeq: number;
  /** order is the manual card order as `layout.json` stores it (FS-02.R78). An
   *  update or a hydration burst neither appends to it nor prunes it, so a load writes
   *  nothing back and an agent with no stored position takes the grid's deterministic
   *  fallback instead of its arrival order; only a removal drops an id. */
  order: string[];
  hydrating: boolean;
  hydrated: boolean;
  applyStateUpdate: (agent: AgentState) => void;
  hydrateBegin: () => void;
  hydrateComplete: (seenIds: string[]) => void;
  removeAgent: (id: string) => void;
  setOrder: (order: string[]) => void;
  clearLastSentAt: (id: string, sentAt: string) => void;
}

export const useAgentStore = create<AgentStoreState>((set) => ({
  agents: {},
  observed: {},
  observedSeq: 0,
  order: [],
  hydrating: false,
  hydrated: false,
  applyStateUpdate: (agent) =>
    set((state) => {
      const observedSeq = state.observedSeq + 1;
      return {
        agents: { ...state.agents, [agent.agent_id]: agent },
        observed: { ...state.observed, [agent.agent_id]: observedSeq },
        observedSeq,
      };
    }),
  hydrateBegin: () => set({ hydrating: true }),
  hydrateComplete: (seenIds) =>
    set((state) => {
      const seen = new Set(seenIds);
      const agents = Object.fromEntries(Object.entries(state.agents).filter(([id]) => seen.has(id)));
      const observed = Object.fromEntries(Object.entries(state.observed).filter(([id]) => seen.has(id)));
      return {
        agents,
        observed,
        hydrating: false,
        hydrated: true,
      };
    }),
  removeAgent: (id) =>
    set((state) => {
      const { [id]: _removed, ...agents } = state.agents;
      const { [id]: _removedSeq, ...observed } = state.observed;
      return { agents, observed, order: state.order.filter((item) => item !== id) };
    }),
  setOrder: (order) => set({ order: order ?? [] }),
  clearLastSentAt: (id, sentAt) =>
    set((state) => {
      const agent = state.agents[id];
      if (!agent || agent.last_sent_at !== sentAt) return state;
      return { agents: { ...state.agents, [id]: { ...agent, last_sent_at: undefined } } };
    }),
}));
