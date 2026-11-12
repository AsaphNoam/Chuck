import type { AgentStatus, TranscriptEvent } from "../api/types";
import { agentName, projectTitle } from "./labels";

/** Origin of the host-owned peer Stop approval (FS-03.R91, TS-03.R71). */
export const PEER_STOP_ORIGIN = "chuck_peer_stop";

export interface PeerIdentity {
  agent_id: string;
  name: string;
  project: string;
}

export interface PeerStopRequest {
  requester: PeerIdentity;
  target: PeerIdentity;
  reason: string;
}

export const PEER_STOP_CONSEQUENCE =
  "Stop ends that agent's process. Its history is kept and it can be woken later. This is not Cancel.";

function identity(value: unknown): PeerIdentity | null {
  if (!value || typeof value !== "object") return null;
  const v = value as Record<string, unknown>;
  if (typeof v.agent_id !== "string") return null;
  return { agent_id: v.agent_id, name: String(v.name ?? ""), project: String(v.project ?? "") };
}

/** peerStopRequest reads the server-derived identities of a host peer Stop
 *  request, or null for an ordinary provider permission. */
export function peerStopRequest(event: TranscriptEvent): PeerStopRequest | null {
  if (event.origin !== PEER_STOP_ORIGIN || !event.peer_stop || typeof event.peer_stop !== "object") return null;
  const subject = event.peer_stop as Record<string, unknown>;
  const requester = identity(subject.requester);
  const target = identity(subject.target);
  if (!requester || !target) return null;
  return { requester, target, reason: String(event.reason ?? "") };
}

/** peerLabel names an agent with its project title. Stop ends one specific process, so when another
 *  known agent in the same project shares the name, the stable id is appended under the shared
 *  duplicate-name rule (FS-02.R69, FS-06.R41); otherwise internal ids are never shown. */
export function peerLabel(
  peer: PeerIdentity,
  projects?: Parameters<typeof projectTitle>[0],
  others: readonly { agent_id: string; name?: string; project?: string }[] = [],
): string {
  const name = agentName(peer.name);
  const clash = others.some((other) => other.agent_id !== peer.agent_id
    && (other.project ?? "") === peer.project && agentName(other.name) === name);
  const label = clash ? `${name} (${peer.agent_id})` : name;
  const project = projectTitle(projects, peer.project);
  return project ? `${label} · ${project}` : label;
}

/** Agents a peer label must stay distinct from: every known agent plus both request parties. */
export function peerLabelPeers(request: PeerStopRequest, agents: Readonly<Record<string, { agent_id: string; name?: string; project?: string }>>) {
  return [...Object.values(agents), request.requester, request.target];
}

/** A host request is decidable only while its requester waits on it: the
 *  runtime holds the requester in waiting_input until the decision, and a
 *  request from an ended generation never resolves on its own (TS-04.R94). */
export function hostRequestEnded(event: TranscriptEvent, status: AgentStatus | undefined): boolean {
  return event.origin === PEER_STOP_ORIGIN && status !== undefined && status !== "waiting_input";
}
