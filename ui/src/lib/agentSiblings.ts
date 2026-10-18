import type { AgentState } from "../api/types";
import { orderProjectAgents } from "../features/dashboard/projectDashboardData";

export interface AgentSibling {
  id: string;
  name: string;
  state: string;
}

/** FS-03.R98/TS-08.R137: the project's running agents plus the current one, in
 * dashboard order. A lone current agent yields none, so the breadcrumb keeps its path. */
export function deriveAgentSiblings(agents: Record<string, AgentState>, currentId: string): AgentSibling[] {
  const current = agents[currentId];
  if (!current || current.archived) return [];
  const members = orderProjectAgents(Object.values(agents).filter((agent) =>
    agent.project === current.project && !agent.archived && (agent.running || agent.agent_id === currentId)));
  if (members.length < 2) return [];
  return members.map((agent) => ({
    id: agent.agent_id,
    name: agent.name,
    // The header's state rule: a stopped agent reads as stopped unless it failed.
    state: agent.running || agent.state === "error" ? agent.state : "stopped",
  }));
}

export function sameSiblings(a: AgentSibling[], b: AgentSibling[]) {
  return a.length === b.length && a.every((entry, index) =>
    entry.id === b[index].id && entry.name === b[index].name && entry.state === b[index].state);
}
