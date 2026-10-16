import type { AgentStatus, TranscriptEvent } from "../api/types";

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

/** peerLabel names an agent with its project and stable id. */
export function peerLabel(peer: PeerIdentity): string {
  return `${peer.name || peer.agent_id} · ${peer.project} · ${peer.agent_id}`;
}

/** A host request is decidable only while its requester waits on it: the
 *  runtime holds the requester in waiting_input until the decision, and a
 *  request from an ended generation never resolves on its own (TS-04.R94). */
export function hostRequestEnded(event: TranscriptEvent, status: AgentStatus | undefined): boolean {
  return event.origin === PEER_STOP_ORIGIN && status !== undefined && status !== "waiting_input";
}
