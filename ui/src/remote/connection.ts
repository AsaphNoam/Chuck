import { create } from "zustand";
import type { AgentState, RuntimeActivity, TranscriptEvent } from "../api/types";
import { openTurnKey } from "../components/chat/turnActivity";
import { useAnnotationStore } from "../store/annotationStore";
import { discardChatDraft } from "../components/chat/drafts";
import { discardAttachmentDraft } from "../components/chat/attachments";
import { useReasoningStore } from "../store/reasoningStore";
import { checkPaired, PhoneAPIError } from "./api";

// The phone's view of its link to the Mac (FS-20 §3). A plain EventSource with
// the device cookie; the desktop SharedWorker is not used (TS-13.R14).

export type PhoneLink = "checking" | "connected" | "reconnecting" | "unreachable" | "unpaired";

interface ConnectionState {
  link: PhoneLink;
  /** When the Mac stopped answering; set while unreachable (FS-20.R23). */
  unreachableSince: number | null;
  /** Bumped on every event that can change Home, for debounced refetches. */
  revision: number;
  /** Every agent, hydrated from the stream's state_update snapshot. */
  agents: Record<string, AgentState>;
  /** Per-agent counter bumped when its transcript grows. */
  transcriptRev: Record<string, number>;
  setLink: (link: PhoneLink) => void;
  bump: () => void;
}

export const useConnection = create<ConnectionState>((set) => ({
  link: "checking",
  unreachableSince: null,
  revision: 0,
  agents: {},
  transcriptRev: {},
  setLink: (link) =>
    set((state) => ({
      link,
      unreachableSince:
        link === "unreachable" ? (state.unreachableSince ?? Date.now()) : link === "connected" ? null : state.unreachableSince,
    })),
  bump: () => set((state) => ({ revision: state.revision + 1 })),
}));

const PAIRED_KEY = "chuck.paired";
export const rememberPaired = (paired: boolean) =>
  paired ? localStorage.setItem(PAIRED_KEY, "1") : localStorage.removeItem(PAIRED_KEY);
export const wasPaired = () => localStorage.getItem(PAIRED_KEY) === "1";

/** Events that can change what Home shows. */
const homeEvents = ["state_update", "task_update", "pipeline_update", "notification"];

// Connection recovery (FS-20.R51–R53, TS-13.R27–R28). One owner per page: at
// most one source, one health read and one deadline (liveness, opening,
// hydration or retry). The server pings every 10 s; a hydrated stream with no
// ping for livenessMs is stale. Every callback checks its generation, so work
// from a replaced attempt can neither reopen access nor overwrite newer state.
export const livenessMs = 25_000;
const healthTimeoutMs = 5_000;
export const retryDelaysMs = [5_000, 10_000, 20_000, 40_000, 60_000];

let source: EventSource | null = null;
let generation = 0;
let started = false;
/** An attempt is between its health read and its completed hydration. */
let attempting = false;
let hydrated = false;
let hydratingAgents: Record<string, AgentState> | null = null;
let lastPing = 0;
let timer: number | undefined;
let health: AbortController | null = null;
/** Consecutive attempts that failed to hydrate; reset only by hydration. */
let failures = 0;
let retryPending = false;
/** A genuine hidden/offline → visible/online transition may advance one retry. */
let mayAdvance = false;

const visible = () => document.visibilityState !== "hidden";

function setTimer(ms: number, gen: number, fire: () => void) {
  window.clearTimeout(timer);
  timer = window.setTimeout(() => {
    timer = undefined;
    if (gen === generation && started && visible()) fire();
  }, ms);
}

function clearTimer() {
  window.clearTimeout(timer);
  timer = undefined;
}

function closeSource() {
  source?.close();
  source = null;
  hydrated = false;
  hydratingAgents = null;
}

/** start begins (or, once paired, restarts) the page's connection owner. */
export function start() {
  if (!started) {
    started = true;
    failures = 0;
    document.addEventListener("visibilitychange", onVisibility);
    window.addEventListener("pageshow", onReturn);
    window.addEventListener("online", onReturn);
    window.addEventListener("offline", onOffline);
    window.addEventListener("pagehide", onPageHide);
  }
  if (!attempting && !hydrated) void attempt();
  return disconnect;
}

/** disconnect releases the owner's timers, source and health read (teardown, unpair). */
export function disconnect() {
  started = false;
  generation++;
  attempting = false;
  retryPending = false;
  clearTimer();
  health?.abort();
  health = null;
  closeSource();
  document.removeEventListener("visibilitychange", onVisibility);
  window.removeEventListener("pageshow", onReturn);
  window.removeEventListener("online", onReturn);
  window.removeEventListener("offline", onOffline);
  window.removeEventListener("pagehide", onPageHide);
}

function onVisibility() {
  if (visible()) onReturn();
  else pause(false);
}

function onOffline() {
  mayAdvance = true;
}

function onPageHide() {
  pause(true);
}

// Hidden: stop checks and retries, keep backoff and cached state. A healthy
// hydrated stream may stay open; anything else is closed (TS-13.R28).
function pause(closeHealthy: boolean) {
  mayAdvance = true;
  clearTimer();
  if (attempting) {
    generation++;
    attempting = false;
    health?.abort();
    health = null;
    retryPending = true;
  }
  if (closeHealthy || !hydrated) closeSource();
}

/** onReturn is the one liveness check for visible, pageshow and online signals. */
function onReturn() {
  if (!started || !visible()) return;
  const advance = mayAdvance;
  mayAdvance = false;
  if (attempting) return; // an opening or hydration keeps its remaining deadline
  if (source && hydrated) {
    const left = livenessMs - (Date.now() - lastPing);
    if (left > 0) armLiveness(left);
    else lost();
    return;
  }
  if (retryPending && timer !== undefined && !advance) return;
  void attempt();
}

function armLiveness(ms = livenessMs) {
  setTimer(ms, generation, () => {
    if (Date.now() - lastPing >= livenessMs) lost();
    else armLiveness(livenessMs - (Date.now() - lastPing));
  });
}

/** lost replaces a stream that was live; the first attempt runs at once. */
function lost() {
  closeSource();
  if (useConnection.getState().link === "connected") useConnection.getState().setLink("reconnecting");
  void attempt();
}

/** failed ends an attempt that did not hydrate and schedules the next one. */
function failed(gen: number) {
  if (gen !== generation) return;
  attempting = false;
  health = null;
  closeSource();
  failures++;
  retryPending = true;
  if (visible()) setTimer(retryDelaysMs[Math.min(failures, retryDelaysMs.length) - 1], gen, () => void attempt());
}

// One serialized attempt: the pairing/reachability read, then a new stream
// that must hydrate. Health success never marks the phone connected.
async function attempt() {
  if (!started || attempting || !visible()) return;
  const gen = ++generation;
  attempting = true;
  retryPending = false;
  clearTimer();
  closeSource();
  const { setLink } = useConnection.getState();
  const abort = new AbortController();
  health = abort;
  const timeout = window.setTimeout(() => abort.abort(), healthTimeoutMs);
  try {
    await checkPaired(abort.signal);
  } catch (error) {
    if (gen !== generation) return;
    if (error instanceof PhoneAPIError && error.status === 401) {
      disconnect();
      setLink("unpaired");
      return;
    }
    setLink("unreachable");
    failed(gen);
    return;
  } finally {
    window.clearTimeout(timeout);
  }
  if (gen !== generation) return;
  health = null;
  if (useConnection.getState().link === "unreachable") setLink("reconnecting");
  open(gen);
}

function open(gen: number) {
  const { setLink, bump } = useConnection.getState();
  const stream = new EventSource("/api/events", { withCredentials: true });
  source = stream;
  hydratingAgents = {};
  setTimer(livenessMs, gen, () => failed(gen)); // opening budget
  stream.onopen = () => {
    if (gen !== generation) return;
    hydratingAgents = {};
    hydrated = false;
    // Live-only reasoning and its live choices have no seq to recover (TS-08.R103).
    // No thought is admitted again until hydration ends and each open window
    // has re-read past what the disconnect may have missed.
    useReasoningStore.getState().clearAll();
    admitting = false;
    if (useConnection.getState().link === "connected") setLink("reconnecting");
    // Hydration budget: pings cannot extend it; only completed hydration clears it.
    setTimer(livenessMs, gen, () => failed(gen));
  };
  stream.onerror = () => {
    if (gen !== generation) return;
    // Close so the server's retry: 2000 cannot run a competing native loop.
    if (attempting) return failed(gen);
    closeSource();
    clearTimer();
    if (useConnection.getState().link === "connected") setLink("reconnecting");
    if (visible()) void attempt();
  };
  // A ping only records receipt; the liveness deadline rearms from it on expiry.
  stream.addEventListener("ping", () => {
    if (gen === generation) lastPing = Date.now();
  });
  for (const type of homeEvents) stream.addEventListener(type, () => {
    // A snapshot's rows coalesce into the one bump after hydration (TS-13.R29).
    if (gen === generation && !hydratingAgents) bump();
  });
  stream.addEventListener("state_update", (event) => {
    if (gen !== generation) return;
    const envelope = parse(event);
    // The server marks hydration on the envelope: {agent_id:"__hydrated__", data:{hydrated:true}}.
    if (envelope?.agent_id === "__hydrated__") {
      const agents = hydratingAgents ?? {};
      hydratingAgents = null;
      hydrated = true;
      attempting = false;
      failures = 0;
      lastPing = Date.now();
      useConnection.setState((state) => {
        const transcriptRev = Object.fromEntries(Object.entries(state.transcriptRev).filter(([id]) => agents[id]));
        for (const id of openTranscripts.keys()) {
          transcriptRev[id] = (transcriptRev[id] ?? 0) + 1;
          fenceRev.set(id, transcriptRev[id]);
        }
        return { agents, transcriptRev };
      });
      admitting = true;
      if (visible()) armLiveness();
      else clearTimer();
      setLink("connected");
      bump(); // catch up only after the authoritative snapshot is complete.
      return;
    }
    const agent = envelope?.data as AgentState | undefined;
    if (!agent?.agent_id) return;
    if (agent.removed) {
      discardChatDraft(agent.agent_id);
      void discardAttachmentDraft(agent.agent_id).catch(() => undefined);
    }
    if (hydratingAgents) {
      if (agent.removed) delete hydratingAgents[agent.agent_id];
      else hydratingAgents[agent.agent_id] = agent;
      return;
    }
    // A deleted agent can no longer be an annotation source (FS-13.R16).
    if (agent.removed) useAnnotationStore.getState().discard(agent.agent_id);
    useConnection.setState((state) => {
      const agents = { ...state.agents };
      if (agent.removed) delete agents[agent.agent_id];
      else agents[agent.agent_id] = agent;
      return { agents };
    });
  });
  stream.addEventListener("runtime_activity", admitReasoning);
  stream.addEventListener("new_message", (event) => {
    if (gen !== generation || hydratingAgents) return;
    const id = parse(event)?.agent_id;
    if (!id) return;
    useConnection.setState((state) => ({ transcriptRev: { ...state.transcriptRev, [id]: (state.transcriptRev[id] ?? 0) + 1 } }));
  });
}

// Open conversations admit live reasoning, as the desktop's open agents do
// (TS-08.R104). Each reads its current folded window on arrival.
// A read returns null unless it holds an exact (not placeholder) window read,
// with the revision that read answered. Leaving the conversation drops its
// thoughts and live choices with it (TS-08.R102).
export interface OpenTranscript {
  events: TranscriptEvent[];
  /** The leading turn's carried key (carryLeadKey). */
  lead: string;
  rev: number;
}

const openTranscripts = new Map<string, () => OpenTranscript | null>();
// The revision each open window must reach after the latest hydration.
const fenceRev = new Map<string, number>();
let admitting = false;

export function watchReasoning(agentId: string, read: () => OpenTranscript | null): () => void {
  openTranscripts.set(agentId, read);
  return () => {
    if (openTranscripts.get(agentId) !== read) return;
    openTranscripts.delete(agentId);
    fenceRev.delete(agentId);
    useReasoningStore.getState().discard(agentId);
  };
}

// The phone's window slides as the conversation grows, so a span anchors after
// the last seq the screen had seen rather than at a list position.
function admitReasoning(event: Event) {
  const activity = parse(event)?.data as RuntimeActivity | undefined;
  const read = activity?.agent_id ? openTranscripts.get(activity.agent_id) : undefined;
  const view = admitting ? read?.() : null;
  if (!activity || !view || view.rev < (fenceRev.get(activity.agent_id) ?? 0)) return;
  const last = [...view.events].reverse().find((item) => typeof item.seq === "number")?.seq ?? 0;
  useReasoningStore.getState().append(activity, Number(last), openTurnKey(view.events, view.lead));
}

function parse(event: Event): { agent_id?: string; data?: unknown } | null {
  try {
    return JSON.parse((event as MessageEvent<string>).data) as { agent_id?: string; data?: unknown };
  } catch {
    return null;
  }
}
