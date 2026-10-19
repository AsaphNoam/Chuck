import { AutoGrowTextarea, ConfirmDialog, VisuallyHidden } from "../components/ui";
import { useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  cancelTurn,
  archiveAgent,
  cloneAgent,
  decidePermission,
  getHeldPrompt,
  dismissPromptRecovery,
  getTrackedFiles,
  renameAgent,
  getTranscriptWindow,
  resumeAgent,
  sendPrompt,
  sendRecoveredPrompt,
  setSessionConfig,
  steerPrompt,
  stopAgent,
  switchRuntime,
  withdrawPrompt,
  updateAgentIdentity,
} from "../api/client";
import type { AgentState, AnnotationDraft, AttachmentDescriptor, TranscriptEvent } from "../api/types";
import { useAnnotationStore } from "../store/annotationStore";
import { useReasoningStore } from "../store/reasoningStore";
import { foldTranscript, normalizeEvent } from "../store/transcriptStore";
import { WorkingIndicator } from "../components/chat/WorkingIndicator";
import { AssistantText } from "../components/chat/renderers/AssistantText";
import { DiffBlock } from "../components/chat/renderers/DiffBlock";
import { AnnotationCard } from "../components/chat/renderers/AnnotationCard";
import { FileViewerView, useFileReader } from "../components/chat/FileViewer";
import { canAnnotate, eventDraft, selectionWithin } from "../lib/annotations";
import { NoticeRow } from "../components/chat/renderers/NoticeRow";
import { ToolCall } from "../components/chat/renderers/ToolCall";
import { ToolResult } from "../components/chat/renderers/ToolResult";
import { groupTranscriptRows, ToolRun } from "../components/chat/toolRun";
import { hideFileChanges, markBackgrounded, nestActivities, withReasoning, type ChildNode } from "../components/chat/runtimeActivity";
import { carryLeadKey } from "../components/chat/turnActivity";
import { ChildActivity } from "../components/chat/renderers/ChildActivity";
import { ThinkingDisclosure } from "../components/chat/renderers/ThinkingDisclosure";
import { TurnList, useFocusReturn, useTurnChoices } from "../components/chat/TurnList";
import { PhoneAnnotationButton, PhoneAnnotations, PhoneAnnotationSent, usePhoneAnnotationDelivery } from "./AnnotationForm";
import { useConnection, watchReasoning, type OpenTranscript } from "./connection";
import { getRuntimeOptions } from "./api";
import { navigate } from "./router";
import { PhoneIcon } from "./PhoneIcon";
import { PhoneSheet } from "./PhoneSheet";
import { recallPlace, rememberPlace } from "./scrollMemory";
import { bumpChatDraftRevision, chatDraftRevision, discardChatDraft, getChatDraft, restoreChatDraft, setChatDraft, subscribeChatDraft } from "../components/chat/drafts";
import { addFiles, attachmentErrorForStorage, chatDraftRequestId, clearPendingAttachmentSend, hasPersistedAttachmentDraft, isDefiniteSendRefusal, isPreviewImage, persistAttachments, persistPendingAttachmentSend, removeAcceptedAttachments, restoreAttachments, restorePendingAttachmentSend, subscribeAttachmentDraft, touchAttachmentDraft, type DraftAttachment } from "../components/chat/attachments";
import { AssistantImageRow, DraftAttachments, MessageAttachments } from "../components/chat/AttachmentView";
import { ContextBar } from "../components/grid/ContextBar";
import { GroupPicker } from "../components/ui/GroupPicker";
import { useRoles } from "../api/config";
import { personaIdentityLabel } from "../lib/personas";
import type { RoleResponse } from "../schemas/role";
import { decodeQuotaState } from "../lib/quota";
import { QuotaBanner } from "../components/agent/QuotaBanner";
import { PEER_STOP_CONSEQUENCE, peerLabel, peerStopRequest } from "../lib/peerStop";

// The phone reads a bounded window and keeps at most EARLIER_PAGES older ones
// (FS-20.R13).
const TRANSCRIPT_PAGE = 750;
const EARLIER_PAGES = 3;
const EARLIER_EVENTS = TRANSCRIPT_PAGE * EARLIER_PAGES;
const EARLIER_BYTES = 3 * 1024 * 1024;
const errorText = (error: unknown) => (error instanceof Error ? error.message : String(error));
const isAtLatest = () => window.scrollY + window.innerHeight >= document.documentElement.scrollHeight - 48;

function retainEarlier(events: TranscriptEvent[]) {
  const unique = [...new Map(events.map((event) => [event.seq, event])).values()].sort((a, b) => (a.seq ?? 0) - (b.seq ?? 0));
  const encoder = new TextEncoder();
  let bytes = 0;
  let start = unique.length;
  while (start > 0 && unique.length - start < EARLIER_EVENTS) {
    const size = encoder.encode(JSON.stringify(unique[start - 1])).byteLength;
    if (bytes + size > EARLIER_BYTES && start < unique.length) break;
    bytes += size;
    start--;
  }
  return unique.slice(start);
}

export function agentTitle(agent: Pick<AgentState, "name" | "role" | "project">, role?: Partial<RoleResponse>) {
  return agent.name || `${personaIdentityLabel(agent.role, role)}@${agent.project}`;
}

/** requestSummary shows the command or the file a permission is for (FS-20.R12). */
function requestSummary(event: TranscriptEvent): string {
  const args = (event.args ?? {}) as Record<string, unknown>;
  const pick = (...keys: string[]) => keys.map((key) => args[key]).find((value) => typeof value === "string") as string | undefined;
  const command = pick("command", "cmd");
  if (command) return command;
  const path = pick("file_path", "path", "notebook_path");
  if (path) return `${String(event.name ?? "Edit")} ${path}`;
  const text = JSON.stringify(event.args ?? {});
  return text === "{}" ? "" : text;
}

const clip = (text: string, max: number) => (text.length > max ? `${text.slice(0, max)}…` : text);

function sameAttachmentSet(left: DraftAttachment[], right: DraftAttachment[]) {
  return left.length === right.length && left.every((file) => right.some((candidate) => candidate.id === file.id));
}

function PhoneRecovery({ agentId, recovery, offline }: { agentId: string; recovery: { text: string; request_id: string; attachments: AttachmentDescriptor[] }; offline: boolean }) {
  const client = useQueryClient();
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const key = `chuck-recovery-send:${agentId}:${recovery.request_id}`;
  const act = async (dismiss: boolean) => {
    setBusy(true); setError("");
    try {
      if (dismiss) await dismissPromptRecovery(agentId, recovery.request_id);
      else {
        let id = localStorage.getItem(key);
        if (!id) { id = crypto.randomUUID(); localStorage.setItem(key, id); }
        await sendRecoveredPrompt(agentId, recovery.text, recovery.attachments.map((item) => item.id), id);
      }
      localStorage.removeItem(key);
      await client.invalidateQueries({ queryKey: ["held", agentId] });
    } catch (reason) { setError(errorText(reason)); }
    finally { setBusy(false); }
  };
  return <section className="phone-card" aria-label="Message available to restore">
    <p className="phone-card-kicker">Message not delivered — choose whether to send it again</p>
    {recovery.text && <p>{recovery.text}</p>}
    <MessageAttachments agentId={agentId} attachments={recovery.attachments} />
    <div className="phone-actions"><button type="button" disabled={busy || offline} onClick={() => void act(false)}>Send recovered message</button><button type="button" disabled={busy || offline} onClick={() => void act(true)}>Dismiss</button></div>
    {error && <p className="phone-error" role="alert">{error}</p>}
  </section>;
}

function PermissionCard({ agent, event, latest, disabled, onSettled }: {
  agent: AgentState;
  event: TranscriptEvent;
  latest: string;
  disabled: boolean;
  onSettled: () => void;
}) {
  const { data: roles } = useRoles();
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const decide = async (decision: "approve" | "deny") => {
    setBusy(true);
    setError(null);
    try {
      await decidePermission(agent.agent_id, String(event.tool_call_id), decision);
    } catch (err) {
      // The first decision wins; a later one learns what happened (FS-20.R24).
      setError(/already resolved/i.test(errorText(err)) ? "Already answered on the Mac or another phone." : errorText(err));
    } finally {
      setBusy(false);
      onSettled();
    }
  };
  const peerStop = peerStopRequest(event);
  const summary = peerStop ? "" : requestSummary(event);
  return (
    <section className="phone-card phone-decision" aria-label="Permission request">
      <p className="phone-card-kicker">{agentTitle(agent, roles?.[agent.role])} needs permission</p>
      <h2>{peerStop ? `Stop ${peerStop.target.name || peerStop.target.agent_id}` : String(event.name ?? "Tool")}</h2>
      {peerStop && (
        <dl className="phone-peer-stop">
          <dt>Requested by</dt>
          <dd>{peerLabel(peerStop.requester)}</dd>
          <dt>Agent to stop</dt>
          <dd>{peerLabel(peerStop.target)}</dd>
        </dl>
      )}
      {summary && <pre className="phone-pre">{clip(summary, 1200)}</pre>}
      {event.reason ? <p>{String(event.reason)}</p> : null}
      {peerStop && <p className="phone-meta">{PEER_STOP_CONSEQUENCE}</p>}
      {latest && (
        <blockquote className="phone-quote">
          <span>Latest message</span>
          {clip(latest, 600)}
        </blockquote>
      )}
      {error && <p className="phone-error">{error}</p>}
      <div className="phone-actions phone-permission-actions">
        <button type="button" disabled={disabled || busy} onClick={() => void decide("deny")}>
          Deny
        </button>
        <button type="button" className="ad-button-primary" disabled={disabled || busy} onClick={() => void decide("approve")}>
          {peerStop ? "Approve once" : "Approve"}
        </button>
      </div>
    </section>
  );
}

function EventRow({ agentId, event, onAnnotate }: { agentId: string; event: TranscriptEvent; onAnnotate: (draft: AnnotationDraft) => void }) {
  switch (event.kind) {
    case "user_text":
      return <div className="phone-user">{String(event.text ?? "")}<MessageAttachments agentId={agentId} attachments={event.attachments ?? []} /></div>;
    case "assistant_text":
      return <AssistantText event={event} agentId={agentId} />;
    case "assistant_image":
      return <div className="assistant-message"><AssistantImageRow agentId={agentId} attachment={event.attachment} path={typeof event.path === "string" ? event.path : undefined} name={typeof event.name === "string" ? event.name : undefined} error={typeof event.error === "string" ? event.error : undefined} /></div>;
    case "tool_call":
      return <ToolCall event={event} />;
    case "tool_result":
      return <ToolResult event={event} />;
    case "diff":
      return <DiffBlock event={event} onAnnotate={onAnnotate} selectHint="Tap line numbers to select a range." />;
    case "permission_request":
      return event.resolved ? <p className="phone-meta">{String(event.name ?? "Permission")} · {String(event.resolved)}</p> : null;
    case "error":
      return <p className="phone-error">{String(event.message ?? event.text ?? "The turn failed.")}</p>;
    case "notice":
      return <NoticeRow event={event} />;
    case "reasoning":
      return <ThinkingDisclosure text={String(event.text ?? "")} activityId={event.activity_id} />;
    case "annotation":
      return <AnnotationCard event={event} />;
    default:
      return null;
  }
}

// Rows the phone draws with text worth pointing at; recorded annotation cards
// and status-only rows are not capture sources (FS-13.R28, R31).
const annotatableKinds = new Set(["user_text", "assistant_text", "tool_call", "tool_result", "diff", "error", "notice", "reasoning"]);

// AnnotatableEvent adds the quiet Annotate action: it takes the selection inside
// this event, read on press because a tap can collapse it before click, and
// otherwise the whole event (TS-08.R133). Native long-press selection, link taps
// and scrolling are left alone.
function AnnotatableEvent({ agentId, event, onAnnotate }: { agentId: string; event: TranscriptEvent; onAnnotate: (draft: AnnotationDraft) => void }) {
  const host = useRef<HTMLDivElement>(null);
  const pressed = useRef<string | null | undefined>(undefined);
  const row = <EventRow agentId={agentId} event={event} onAnnotate={onAnnotate} />;
  if (!canAnnotate(event) || !annotatableKinds.has(String(event.kind))) return row;
  const selected = () => (host.current ? selectionWithin(host.current) : null);
  return (
    <div ref={host} className="phone-annotatable">
      {row}
      <button
        type="button"
        className="phone-annotate-event"
        onPointerDown={() => { pressed.current = selected(); }}
        onClick={(click) => {
          const text = pressed.current !== undefined ? pressed.current : selected();
          pressed.current = undefined;
          // A tap does not focus a button on phones; the tray returns focus here.
          click.currentTarget.focus();
          onAnnotate(eventDraft(event, text));
        }}
      >
        Annotate
      </button>
    </div>
  );
}

// renderRows is the phone's row renderer inside the shared turn list: tool runs,
// nested native children and ordinary rows, each one list item (FS-03.R77).
function renderRows(agentId: string, list: TranscriptEvent[], onAnnotate: (draft: AnnotationDraft) => void, ancestry: string[], depth: number): ReactNode {
  const nested = (events: TranscriptEvent[], path: string[], level: number) => renderRows(agentId, events, onAnnotate, path, level);
  return groupTranscriptRows(list).map((row, index) => {
    if (row.kind === "tool-run") {
      return (
        <div className="phone-message" role="listitem" key={`run:${row.events[0].seq ?? index}`} data-place={`run:${row.events[0].seq ?? index}`}>
          <ToolRun events={row.events} renderEvent={(event) => <AnnotatableEvent key={`${event.seq}:${event.kind}`} agentId={agentId} event={event} onAnnotate={onAnnotate} />} />
        </div>
      );
    }
    const event = row.event;
    return (
      <div className="phone-message" role="listitem" key={`${event.seq ?? event.message_id ?? index}:${event.kind}`} data-place={`${event.seq ?? event.message_id ?? index}:${event.kind}`}>
        {event.kind === "activity" ? (
          <ChildActivity node={event.node as ChildNode} ancestry={ancestry} depth={depth} renderEvents={nested} />
        ) : (
          <AnnotatableEvent agentId={agentId} event={event} onAnnotate={onAnnotate} />
        )}
      </div>
    );
  });
}

// The phone stores a span's anchor as the last seq it had seen; place it before
// the first later row of the current window.
function slotAfter(events: TranscriptEvent[], seq: number) {
  const index = events.findIndex((event) => typeof event.seq === "number" && event.seq > seq);
  return index < 0 ? events.length : index;
}

export function AgentScreen({ agentId }: { agentId: string }) {
  const { data: roles } = useRoles();
  const agent = useConnection((state) => state.agents[agentId]);
  const rev = useConnection((state) => state.transcriptRev[agentId] ?? 0);
  const link = useConnection((state) => state.link);
  const offline = link !== "connected";
  const client = useQueryClient();
  const [text, setText] = useState(() => getChatDraft(agentId));
  const [attachments, setAttachments] = useState<DraftAttachment[]>([]);
  const [attachmentsReady, setAttachmentsReady] = useState(() => !hasPersistedAttachmentDraft(agentId));
  const [attachmentError, setAttachmentError] = useState<string | null>(null);
  const [draggingFiles, setDraggingFiles] = useState(false);
  const [retryPhoneSend, setRetryPhoneSend] = useState<{ text: string; files: DraftAttachment[]; requestId: string; steer: boolean; revision: number; retryable?: boolean } | null>(null);
  const attachmentPicker = useRef<HTMLInputElement>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [quotaCancelError, setQuotaCancelError] = useState<string | undefined>();
  const [stopOpen, setStopOpen] = useState(false);
  const [runtimeOpen, setRuntimeOpen] = useState(false);
  const runtimeOptions = useQuery({ queryKey: ["runtime-options"], queryFn: getRuntimeOptions });
  const [tab, setTab] = useState<"chat" | "files" | "manage">("chat");
  const [rename, setRename] = useState("");
  const [filePath, setFilePath] = useState<string | null>(null);
  const [diffSeq, setDiffSeq] = useState<number | null>(null);
  const chat = agent?.interface !== "terminal";
  const files = useQuery({ queryKey: ["tracked-files", agentId, rev], queryFn: () => getTrackedFiles(agentId), enabled: tab === "files" });
  // The shared reader keeps the tracked path as its read key, so Reload asks for
  // exactly that path even when the response names a normalized one (TS-13.R31).
  const fileReader = useFileReader({ agentId, link: filePath ? { path: filePath } : null });
  // Open diff reads exactly the requested event, wherever it falls (FS-20.R37).
  const focusedDiff = useQuery({
    queryKey: ["focused-diff", agentId, diffSeq],
    queryFn: async () => (await getTranscriptWindow(agentId, { limit: 1, beforeSeq: diffSeq! + 1 })).events.find((event) => event.seq === diffSeq) ?? null,
    enabled: diffSeq !== null,
  });
  const diffCard = useRef<HTMLElement | null>(null);
  useEffect(() => {
    if (tab === "chat" && focusedDiff.data) diffCard.current?.scrollIntoView?.({ block: "start" });
  }, [tab, focusedDiff.data]);
  const addAnnotation = useAnnotationStore((state) => state.add);
  // One tray owner above the Chat/Files branches (TS-08.R134).
  const delivery = usePhoneAnnotationDelivery(agentId, agent);
  const [trayOpen, setTrayOpen] = useState(false);
  const [focusNewest, setFocusNewest] = useState(0);

  useEffect(() => {
    let current = true;
    setText(getChatDraft(agentId));
    setAttachments([]);
    const needsAttachmentRestore = hasPersistedAttachmentDraft(agentId);
    setAttachmentsReady(!needsAttachmentRestore);
    setAttachmentError(null);
    if (needsAttachmentRestore) {
      void restoreAttachments(agentId).then((restored) => { if (current) { setAttachments(restored); setAttachmentsReady(true); } }).catch(() => { if (current) setAttachmentError("Saved attachments could not be restored. Reload before sending to avoid losing them."); });
    }
    const initialRevision = chatDraftRevision(agentId);
    void restorePendingAttachmentSend(agentId).then(async (pending) => {
      if (current && pending) {
        const liveFiles = await restoreAttachments(agentId);
        const revision = chatDraftRevision(agentId);
        const sameFiles = pending.files.length === liveFiles.length && pending.files.every((file) => liveFiles.some((live) => live.id === file.id));
        const sameDraft = getChatDraft(agentId) === pending.text && sameFiles;
        setRetryPhoneSend({ ...pending, revision: revision === initialRevision && sameDraft ? revision : -1 });
      }
    }).catch(() => { if (current) setAttachmentError("A previous attachment send could not be restored from browser storage."); });
    return () => { current = false; };
  }, [agentId]);

  useEffect(() => subscribeChatDraft(agentId, setText), [agentId]);
  useEffect(() => subscribeAttachmentDraft(agentId, setAttachments), [agentId]);

  const saveAttachments = (next: DraftAttachment[]) => {
    setAttachments(next);
    setAttachmentError(null);
    if (!touchAttachmentDraft(agentId, next, text !== "")) setAttachmentError(attachmentErrorForStorage());
    bumpChatDraftRevision(agentId);
    void persistAttachments(agentId, next).catch(() => setAttachmentError(attachmentErrorForStorage()));
  };
  const acceptPhoneFiles = (files: FileList | File[], source: "picker" | "paste" | "drop") => {
    const result = addFiles(attachments, files, source, agent?.attachment_capabilities);
    if (result.files !== attachments) saveAttachments(result.files);
    setAttachmentError(result.errors[0] ?? null);
  };

  // Older windows cover seqs before `anchor`; while they are shown, the live
  // window reads from the anchor so the two stay contiguous.
  const [earlier, setEarlier] = useState<{ anchor: number; events: TranscriptEvent[]; pages: number; hasMore: boolean; recoveredLiveStart?: number; recoveryPages?: number } | null>(null);
  const [loadingEarlier, setLoadingEarlier] = useState(false);
  const [gapRetry, setGapRetry] = useState(0);
  const loadingEarlierRef = useRef(false);
  const scrollAnchor = useRef<{ height: number; top: number } | null>(null);
  const generation = useRef(0);

  const transcript = useQuery({
    queryKey: ["transcript", agentId, rev, earlier?.anchor],
    queryFn: () => getTranscriptWindow(agentId, { limit: TRANSCRIPT_PAGE, sinceSeq: earlier?.anchor }),
    enabled: chat,
    placeholderData: (prev) => prev?.agent_id === agentId ? prev : undefined,
  });
  const liveStart = transcript.data?.events[0]?.seq;
  const gap = !!earlier && !transcript.isPlaceholderData && !!transcript.data?.has_more && earlier.recoveredLiveStart !== liveStart;
  const held = useQuery({
    queryKey: ["held", agentId, rev, agent?.state],
    queryFn: () => getHeldPrompt(agentId).catch(() => null),
    enabled: chat && !!agent,
  });

  // Fold the joined windows as the desktop does, so streamed deltas coalesce
  // into one message even across a window edge (TS-08.R73).
  const events = useMemo(() => {
    const joined = [...(earlier?.events ?? []), ...(gap ? [] : (transcript.data?.events ?? []))];
    return foldTranscript([...new Map(joined.map((event) => [event.seq, event])).values()].sort((a, b) => (a.seq ?? 0) - (b.seq ?? 0)));
  }, [earlier, gap, transcript.data]);
  // Live reasoning arrives on the same authenticated stream as the desktop's and
  // shares its bounded store (TS-08.R104). The phone's window slides as the
  // conversation grows, so a span anchors after the last seq it saw rather than
  // at a list position. Only an exact window read owns a thought: none before the
  // first read, nor while a newer read replaces placeholder data (TS-08.R103).
  // The leading turn keeps its identity as the window slides (TS-08.R102).
  const reasoning = useReasoningStore((state) => state.byAgent[agentId]?.spans);
  const knownKeys = useRef(new Map<number, string>());
  const atStart = earlier ? !earlier.hasMore : !transcript.data?.has_more;
  const lead = useMemo(() => {
    const carried = carryLeadKey(events, atStart, knownKeys.current);
    knownKeys.current = carried.known;
    return carried.lead;
  }, [events, atStart]);
  const viewRef = useRef<OpenTranscript | null>(null);
  viewRef.current = transcript.data && !transcript.isPlaceholderData ? { events, lead, rev } : null;
  useEffect(() => watchReasoning(agentId, () => viewRef.current), [agentId]);
  const rows = useMemo(() => {
    const spans = reasoning?.map((span) => ({ ...span, anchor: slotAfter(events, span.anchor) }));
    return nestActivities(hideFileChanges(withReasoning(markBackgrounded(events), spans, lead)));
  }, [events, reasoning, lead]);
  const choices = useTurnChoices(agentId);
  const listRef = useRef<HTMLDivElement>(null);
  const trackFocus = useFocusReturn(listRef);
  // The server derives both from the whole session, so they hold when the
  // request or the reply falls before the window.
  const pendingEvent = transcript.data?.pending_permission;
  const pending = pendingEvent ? normalizeEvent(pendingEvent) : undefined;
  const latest = transcript.data?.latest_assistant ?? "";
  const refresh = () => void client.invalidateQueries({ queryKey: ["transcript", agentId] });
  const olderAvailable = earlier ? earlier.hasMore : !!transcript.data?.has_more;

  const showEarlier = async () => {
    if (loadingEarlierRef.current) return;
    if (gap) {
      setGapRetry((value) => value + 1);
      return;
    }
    if ((earlier?.pages ?? 0) >= EARLIER_PAGES) return;
    const anchor = earlier?.anchor ?? transcript.data?.events[0]?.seq;
    const before = earlier?.events[0]?.seq ?? anchor;
    if (!anchor || !before) return;
    const requestGeneration = generation.current;
    loadingEarlierRef.current = true;
    setLoadingEarlier(true);
    scrollAnchor.current = { height: document.documentElement.scrollHeight, top: window.scrollY };
    try {
      const page = await getTranscriptWindow(agentId, { limit: TRANSCRIPT_PAGE, beforeSeq: before });
      if (requestGeneration !== generation.current) return;
      setEarlier((prev) => ({
        anchor,
        events: retainEarlier([...page.events, ...(prev?.events ?? []), ...(!prev ? transcript.data?.events.slice(0, 1) ?? [] : [])]),
        pages: (prev?.pages ?? 0) + 1,
        hasMore: page.has_more,
      }));
      setError(null);
    } catch (err) {
      if (requestGeneration !== generation.current) return;
      scrollAnchor.current = null;
      setError(errorText(err));
    } finally {
      if (requestGeneration !== generation.current) return;
      loadingEarlierRef.current = false;
      setLoadingEarlier(false);
    }
  };

  useLayoutEffect(() => {
    if (!scrollAnchor.current) return;
    const { height, top } = scrollAnchor.current;
    scrollAnchor.current = null;
    window.scrollTo(0, top + document.documentElement.scrollHeight - height);
  }, [earlier?.events]);

  // Reopening a chat returns to the message that was at the top of the screen, or
  // to the newest message when it was left there or is new (FS-20.R54).
  const placePath = `/agent/${agentId}`;
  const initialPlace = useRef(recallPlace(placePath));
  const placeRestored = useRef(false);
  const lastY = useRef(0);
  useLayoutEffect(() => {
    if (placeRestored.current || !transcript.data || transcript.isPlaceholderData || !listRef.current) return;
    placeRestored.current = true;
    const place = initialPlace.current;
    const top = document.querySelector(".phone-header")?.getBoundingClientRect().bottom ?? 0;
    if (place && "row" in place) {
      const row = listRef.current.querySelector(`[data-place="${CSS.escape(place.row)}"]`) ?? listRef.current;
      window.scrollTo(0, window.scrollY + row.getBoundingClientRect().top - (row === listRef.current ? top : place.offset));
    } else {
      window.scrollTo(0, document.documentElement.scrollHeight);
    }
    lastY.current = window.scrollY;
  }, [rows, tab, transcript.data, transcript.isPlaceholderData]);

  // A refreshed window (new messages, a recovered connection) keeps the message
  // the person was reading where it was on screen (FS-20.R52, TS-13.R29).
  useLayoutEffect(() => {
    const place = recallPlace(placePath);
    if (!placeRestored.current || !listRef.current || !place || !("row" in place)) return;
    const row = listRef.current.querySelector(`[data-place="${CSS.escape(place.row)}"]`);
    const shift = row ? row.getBoundingClientRect().top - place.offset : 0;
    if (shift) window.scrollTo(0, window.scrollY + shift);
  }, [rows, placePath]);

  const [awayFromLatest, setAwayFromLatest] = useState(false);
  // A live reply can push the newest message off screen without a scroll event.
  useEffect(() => { setAwayFromLatest(!isAtLatest()); }, [rows]);
  useEffect(() => {
    let frame = 0;
    const save = () => {
      frame = 0;
      const atLatest = isAtLatest();
      setAwayFromLatest(!atLatest);
      const list = listRef.current;
      if (!placeRestored.current || !list) return;
      if (atLatest) return rememberPlace(placePath, { newest: true });
      const top = document.querySelector(".phone-header")?.getBoundingClientRect().bottom ?? 0;
      const row = [...list.querySelectorAll<HTMLElement>("[data-place]")].find((item) => item.getBoundingClientRect().bottom > top);
      if (row) rememberPlace(placePath, { row: row.dataset.place!, offset: row.getBoundingClientRect().top });
    };
    const onScroll = () => { if (!frame) frame = requestAnimationFrame(save); };
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => { cancelAnimationFrame(frame); window.removeEventListener("scroll", onScroll); };
  }, [placePath]);

  useEffect(() => {
    lastY.current = window.scrollY;
    const onScroll = () => {
      const y = window.scrollY;
      if (y < lastY.current && y <= 200 && (gap || olderAvailable)) void showEarlier();
      lastY.current = y;
    };
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, [gap, olderAvailable, earlier, transcript.data]);

  useEffect(() => {
    if (!gap || loadingEarlierRef.current || !earlier || !liveStart) return;
    if (earlier.recoveredLiveStart && earlier.recoveredLiveStart !== liveStart) {
      setEarlier({ ...earlier, events: earlier.events.filter((event) => (event.seq ?? 0) <= earlier.anchor), recoveredLiveStart: undefined, recoveryPages: 0 });
      return;
    }
    if ((earlier.recoveryPages ?? 0) >= EARLIER_PAGES) {
      setEarlier(null);
      setError("New activity exceeded the phone's retained history. The phone returned to the latest messages.");
      return;
    }
    const recover = async () => {
      const requestGeneration = generation.current;
      loadingEarlierRef.current = true;
      setLoadingEarlier(true);
      try {
        const bridge = earlier.events.filter((event) => (event.seq ?? 0) > earlier.anchor);
        const beforeSeq = bridge[0]?.seq ?? liveStart;
        const page = await getTranscriptWindow(agentId, { limit: TRANSCRIPT_PAGE, beforeSeq });
        if (requestGeneration !== generation.current) return;
        setEarlier((prev) => prev && ({
          ...prev,
          events: retainEarlier([...prev.events, ...page.events.filter((event) => (event.seq ?? 0) >= prev.anchor)]),
          recoveryPages: (prev.recoveryPages ?? 0) + 1,
          recoveredLiveStart: (page.events[0]?.seq ?? 0) <= prev.anchor ? liveStart : prev.recoveredLiveStart,
        }));
        setError(null);
      } catch (err) {
        if (requestGeneration !== generation.current) return;
        setError(errorText(err));
      } finally {
        if (requestGeneration !== generation.current) return;
        loadingEarlierRef.current = false;
        setLoadingEarlier(false);
      }
    };
    void recover();
  }, [agentId, gap, gapRetry, earlier, liveStart]);

  useEffect(() => {
    generation.current++;
    setEarlier(null);
    setGapRetry(0);
    setLoadingEarlier(false);
    setError(null);
    loadingEarlierRef.current = false;
    scrollAnchor.current = null;
    setTrayOpen(false);
    setFocusNewest(0);
  }, [agentId]);

  if (!agent) return <p className="phone-empty">This agent is not on the Mac any more. <button type="button" className="phone-link ad-button-secondary" onClick={() => navigate("/")}>Home</button></p>;
  // Archived from either device: the screen stops offering work (FS-20.R39).
  if (agent.archived) {
    return (
      <div className="phone-agent">
        <header className="phone-agent-header"><h1>{agentTitle(agent, roles?.[agent.role])}</h1></header>
        <p className="phone-empty">Archived on the Mac. Restore is available on the desktop.</p>
        <div className="phone-actions">
          <button type="button" onClick={() => navigate(`/project/${encodeURIComponent(agent.project)}`)}>Project</button>
          <button type="button" onClick={() => navigate("/")}>Home</button>
        </div>
      </div>
    );
  }

  // Every refused action shows the desktop's reason and keeps the typed text
  // (FS-20.R27); only a delivered message clears the composer.
  const act = async (fn: () => Promise<unknown>, clears = false, after?: (result: unknown) => void | Promise<void>) => {
    const submittedRevision = chatDraftRevision(agentId);
    setBusy(true);
    setError(null);
    setNotice(null);
    try {
      const result = await fn();
      if (clears && chatDraftRevision(agentId) === submittedRevision) {
        discardChatDraft(agentId);
        setText("");
      }
      await after?.(result);
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
      refresh();
      void client.invalidateQueries({ queryKey: ["held", agentId] });
    }
  };
  const finishPhoneAttachmentSend = async (result: unknown, selected: DraftAttachment[], promptText: string, requestId: string, steeringMessage: boolean, submittedRevision: number) => {
    const response = result && typeof result === "object" ? result as { attachment_delivery?: string; delivery?: string } : {};
    if (selected.length && response.attachment_delivery !== "submitted" && response.attachment_delivery !== "held") {
      setError(response.attachment_delivery === "released"
        ? "This message was released for recovery. Restore it from the conversation before sending again."
        : response.attachment_delivery === "failed"
          ? "The message was not delivered. Its attachments remain here; retry the same send."
          : "Delivery could not be confirmed. Its attachments remain available; retry uses the same request receipt.");
      const retryable = response.attachment_delivery !== "released";
      const pendingRetry = { text: promptText, files: selected, requestId, steer: steeringMessage, revision: submittedRevision, retryable };
      setRetryPhoneSend(pendingRetry);
      if (!retryable) void persistPendingAttachmentSend(agentId, pendingRetry).catch(() => setAttachmentError(attachmentErrorForStorage()));
      return;
    }
    if (response.delivery === "held" || response.attachment_delivery === "held") setNotice("Held — it will be sent when the current turn ends.");
    if (chatDraftRevision(agentId) === submittedRevision) {
      discardChatDraft(agentId);
      setText("");
    }
    try { await removeAcceptedAttachments(agentId, selected.map((file) => file.id), getChatDraft(agentId) !== ""); }
    catch { setAttachmentError("Message was accepted, but attachment cleanup could not be saved. Remove any sent files before sending again."); }
    setRetryPhoneSend(null);
    void clearPendingAttachmentSend(agentId, requestId).catch(() => setAttachmentError("Message was sent, but its retry receipt could not be cleared from browser storage."));
  };
  const sendPhoneMessage = (steeringMessage: boolean, requestId?: string, expectedRevision?: number, retryFiles?: DraftAttachment[], retryText?: string) => {
    const matchingRetry = !requestId && !retryFiles && retryPhoneSend?.retryable !== false && retryPhoneSend && retryPhoneSend.text === text && sameAttachmentSet(retryPhoneSend.files, attachments) ? retryPhoneSend : null;
    const messageText = retryText ?? matchingRetry?.text ?? text;
    const selected = retryFiles ?? matchingRetry?.files ?? [...attachments];
    const request = requestId ?? matchingRetry?.requestId ?? (chatDraftRequestId(agentId) || crypto.randomUUID());
    const submittedRevision = expectedRevision ?? matchingRetry?.revision ?? chatDraftRevision(agentId);
    if (!attachmentsReady) { setAttachmentError("Restoring attachments… wait before sending."); return; }
    if (!messageText.trim() && !selected.length) return;
    if (!requestId && retryPhoneSend?.retryable === false && retryPhoneSend.text === messageText && sameAttachmentSet(retryPhoneSend.files, selected)) return;
    const unsupportedImage = selected.some((file) => isPreviewImage(file) && agent?.attachment_capabilities?.image_input === false);
    const unsupportedFile = selected.some((file) => !isPreviewImage(file) && agent?.attachment_capabilities?.files === false);
    if (unsupportedImage || unsupportedFile) {
      setError(unsupportedImage
        ? "This runtime does not accept image input. Remove the image or choose a runtime that supports it."
        : "This runtime cannot read uploaded files. Remove the file or choose a runtime that supports it.");
      return;
    }
    if (!selected.length) {
      void act(
        () => (matchingRetry?.steer ?? steeringMessage) ? steerPrompt(agentId, messageText) : sendPrompt(agentId, messageText),
        true,
        (result) => {
          if (result && typeof result === "object" && "delivery" in result && result.delivery === "held") setNotice("Held — it will be sent when the current turn ends.");
        },
      );
      return;
    }
    setRetryPhoneSend({ text: messageText, files: selected, requestId: request, steer: steeringMessage, revision: submittedRevision });
    void (async () => {
      try { await persistPendingAttachmentSend(agentId, { text: messageText, files: selected, requestId: request, steer: steeringMessage, revision: submittedRevision }); }
      catch { setAttachmentError(attachmentErrorForStorage()); }
      await act(
        () => ((matchingRetry?.steer ?? steeringMessage)
          ? steerPrompt(agentId, messageText, selected.map(({ name, blob }) => ({ name, blob })), request)
          : sendPrompt(agentId, messageText, selected.map(({ name, blob }) => ({ name, blob })), request)
        ).catch(async (err: unknown) => {
          if (isDefiniteSendRefusal(err)) {
            setRetryPhoneSend(null);
            try { await clearPendingAttachmentSend(agentId, request); } catch { setAttachmentError(attachmentErrorForStorage()); }
          }
          throw err;
        }),
        false,
        (result) => finishPhoneAttachmentSend(result, selected, messageText, request, matchingRetry?.steer ?? steeringMessage, submittedRevision),
      );
    })();
  };
  const isBusy = agent.state === "busy" || agent.state === "waiting_input";
  const quota = decodeQuotaState(agent.quota);
  const cancelQuotaContinuation = async () => {
    if (!quota || busy || offline) return;
    setBusy(true);
    setQuotaCancelError(undefined);
    try {
      await cancelTurn(agentId, {
        target: "quota_continuation",
        episode_id: quota.episode_id,
        revision: quota.revision,
      });
    } catch (err) {
      setQuotaCancelError(errorText(err));
    } finally {
      setBusy(false);
      refresh();
    }
  };
  // A capture opens the tray at its new instruction.
  const annotate = (draft: AnnotationDraft) => {
    if (!addAnnotation(agentId, draft)) {
      setError("At most 20 annotations can wait at once. Send or discard them first.");
      return;
    }
    setTrayOpen(true);
    setFocusNewest((value) => value + 1);
  };
  const closeTray = () => { setTrayOpen(false); setFocusNewest(0); };
  const heldText = held.data && typeof held.data === "object" && "text" in held.data ? String((held.data as { text?: string }).text ?? "") : "";
  const heldAttachments = held.data && typeof held.data === "object" && Array.isArray((held.data as { attachments?: unknown }).attachments) ? (held.data as { attachments: AttachmentDescriptor[] }).attachments : [];
  const recovery = held.data && typeof held.data === "object" && "recovery" in held.data ? (held.data as { recovery?: { text: string; request_id: string; attachments: AttachmentDescriptor[] } }).recovery : undefined;
  const releaseBlocked = retryPhoneSend?.retryable === false && retryPhoneSend.text === text && sameAttachmentSet(retryPhoneSend.files, attachments);
  const stateLabel = agent.running ? agent.state.replace("_", " ") : "Stopped";
  const runtimeLabel = [agent.backend, agent.model].filter(Boolean).join(" · ");
  // The effort button steps through the live model's efforts in order and wraps;
  // "model default" counts as the model's default effort.
  const liveModel = runtimeOptions.data?.backends?.find((item) => item.id === agent.backend)?.models.find((item) => item.id === agent.model);
  const efforts = agent.interface === "chat" ? liveModel?.efforts ?? [] : [];
  const currentEffort = agent.effort || liveModel?.default_effort || "";
  const nextEffort = efforts[(efforts.indexOf(currentEffort) + 1) % Math.max(efforts.length, 1)] ?? "";
  const roleLabel = personaIdentityLabel(agent.role, roles?.[agent.role]);
  const roleBackend = [roleLabel, agent.backend].filter(Boolean).join(" · ");

  return (
    <div className="phone-agent">
      <header className="phone-agent-header">
        <div className="phone-agent-title">
          <button type="button" className="phone-agent-back ad-button-secondary" aria-label={`Back to ${agent.project}`} onClick={() => navigate(`/project/${encodeURIComponent(agent.project)}`)}>
            <PhoneIcon name="back" size={19} />
          </button>
          <span className="phone-eyebrow">{roleBackend}</span>
        </div>
        <div className="phone-agent-identity">
          <h1>{agentTitle(agent, roles?.[agent.role])}</h1>
          <span className="phone-status" data-tone={agent.running ? agent.state : "stopped"}>
            {stateLabel}
          </span>
        </div>
        <p className="phone-agent-runtime">{agent.model}{agent.effort ? ` · ${agent.effort} effort` : ""}</p>
      </header>
      <ContextBar detailed value={agent.context_pct} used={agent.context_used} size={agent.context_size} />
      <QuotaBanner value={quota} cancelPending={busy} cancelDisabled={offline} cancelError={quotaCancelError} onCancel={() => void cancelQuotaContinuation()} />
      <div className="phone-agent-tabs">
        <div className="phone-actions phone-tabs" role="tablist" aria-label="Agent views">
          {(["chat", "files", "manage"] as const).map((name) => <button key={name} type="button" role="tab" aria-selected={tab === name} className={tab === name ? "active" : undefined} onClick={() => setTab(name)}>{name[0].toUpperCase() + name.slice(1)}{name === "files" && files.data?.files.length ? <span className="phone-tab-count">{files.data.files.length}</span> : null}</button>)}
        </div>
        {agent.running ? (
          <button type="button" className="ad-button-danger" disabled={offline || busy} onClick={() => setStopOpen(true)}>Stop</button>
        ) : (
          <button type="button" className="ad-button-secondary" disabled={offline || busy} onClick={() => void act(() => resumeAgent(agentId))}>Resume</button>
        )}
      </div>
      {chat && tab !== "manage" && <PhoneAnnotationButton sourceId={agentId} onOpen={() => setTrayOpen(true)} />}
      {tab !== "manage" && <PhoneAnnotationSent delivery={delivery} />}
      <PhoneAnnotations delivery={delivery} open={trayOpen} focusNewest={focusNewest} onClose={closeTray} />
      {!agent.running && tab !== "manage" && error && <p className="phone-error" role="alert">{error}</p>}
      {tab === "files" && <section className="phone-section phone-files" aria-label="Files">
        <h2>Files</h2>
        {files.isError ? <p className="phone-error">{errorText(files.error)}</p> : <ul className="phone-list">
          {(files.data?.files ?? []).map((tracked) => <li key={tracked.path}>
            <div className="phone-row">
              <span className="phone-row-title">{tracked.path}</span>
              <span className="phone-row-meta">{tracked.edit_count} edits · {new Date(tracked.last_ts).toLocaleString()}</span>
              <div className="phone-actions phone-file-actions">
                {tracked.has_diff && tracked.diff_refs[0] && <button type="button" onClick={() => { setDiffSeq(tracked.diff_refs[0].seq); setTab("chat"); }}>Open diff</button>}
                <button type="button" aria-expanded={filePath === tracked.path} onClick={() => setFilePath(filePath === tracked.path ? null : tracked.path)}>{filePath === tracked.path ? "Close" : "Open file"}</button>
              </div>
              {filePath === tracked.path && <section className="phone-file-preview" aria-label="File content">
                <FileViewerView reader={fileReader} onAnnotateSelection={chat ? annotate : undefined} />
              </section>}
            </div>
          </li>)}
        </ul>}
      </section>}
      {tab === "manage" && <AgentManagement agent={agent} offline={offline} busy={busy} act={act} rename={rename} setRename={setRename} error={error} />}
      {tab === "chat" && <div className="phone-conversation">
      {diffSeq !== null && (
        <section ref={diffCard} className="phone-card" aria-label="Requested diff">
          <div className="phone-actions">
            <strong>Diff</strong>
            <button type="button" onClick={() => setDiffSeq(null)}>Close</button>
          </div>
          {focusedDiff.isError ? <p className="phone-error">{errorText(focusedDiff.error)}</p> : focusedDiff.data ? <EventRow agentId={agentId} event={normalizeEvent(focusedDiff.data)} onAnnotate={annotate} /> : focusedDiff.data === null ? <p className="phone-meta">This diff is no longer in the conversation.</p> : <p className="phone-meta">Loading…</p>}
        </section>
      )}
      {!chat ? (
        <p className="phone-empty">This is a terminal agent. Its terminal is on the Mac; the phone shows its status only.</p>
      ) : (
        <>
          {loadingEarlier && <p className="phone-meta" role="status">Loading earlier messages…</p>}
          {olderAvailable && (earlier?.pages ?? 0) >= EARLIER_PAGES && (
            <p className="phone-meta">Earlier messages are outside the phone's retained history.</p>
          )}
          <div className="phone-transcript phone-transcript-bubbles" role="list" aria-label="Conversation" ref={listRef} onFocus={trackFocus}>
            <TurnList agentId={agentId} events={rows} lead={lead} choices={choices} renderEvents={(list) => renderRows(agentId, list, annotate, [], 1)} />
            {!offline && agent.running && agent.state === "busy" && <WorkingIndicator />}
          </div>
          {pending && <PermissionCard agent={agent} event={pending} latest={latest} disabled={offline} onSettled={refresh} />}
          {(heldText || heldAttachments.length > 0) && (
            <div className="phone-card">
              <p className="phone-card-kicker">Held until the current turn ends</p>
              {heldText && <p>{heldText}</p>}
              <MessageAttachments agentId={agentId} attachments={heldAttachments} />
              <div className="phone-actions">
                {agent.steering_available && isBusy && (
                  <button type="button" disabled={offline || busy} onClick={() => void act(() => steerPrompt(agentId, ""))}>
                    Send now
                  </button>
                )}
                <button type="button" disabled={offline || busy} onClick={() => void act(() => withdrawPrompt(agentId), false, () => { if (heldText && heldAttachments.length === 0) restoreChatDraft(agentId, heldText); })}>
                  Withdraw
                </button>
              </div>
            </div>
          )}
          {recovery && <PhoneRecovery agentId={agentId} recovery={recovery} offline={offline} />}
          {awayFromLatest && <button type="button" className="phone-jump-latest ad-button-secondary" onClick={() => window.scrollTo(0, document.documentElement.scrollHeight)}>Jump to latest</button>}
          {agent.running && (
            <div className="phone-runtime-controls">
              <button type="button" className="phone-model-switch" aria-label={`Switch model, current ${runtimeLabel}`} disabled={offline || busy} onClick={() => setRuntimeOpen(true)}>
                {runtimeLabel}
                <PhoneIcon name="down" size={12} />
              </button>
              {efforts.length > 0 && (
                <button type="button" className="phone-model-switch phone-effort-cycle" aria-label={`Effort ${currentEffort || "model default"}; change to ${nextEffort}`} disabled={offline || busy} onClick={() => void act(() => setSessionConfig(agentId, { effort: nextEffort }))}>
                  {currentEffort || "default"} effort
                </button>
              )}
            </div>
          )}
          {runtimeOpen && (
            <PhoneSheet title="Switch runtime" onClose={() => setRuntimeOpen(false)}>
              <AgentManagement agent={agent} offline={offline} busy={busy} act={(fn, clears, after) => void act(fn, clears, (result) => { after?.(result); setRuntimeOpen(false); })} rename={rename} setRename={setRename} error={error} action="runtime" />
            </PhoneSheet>
          )}
          {agent.running && (
            <form
              className={`phone-composer phone-composer-box${draggingFiles ? " is-file-drop-target" : ""}`}
              onDragEnter={(event) => { if ([...event.dataTransfer.types].includes("Files")) { event.preventDefault(); setDraggingFiles(true); } }}
              onDragOver={(event) => { if ([...event.dataTransfer.types].includes("Files")) event.preventDefault(); }}
              onDragLeave={(event) => { if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setDraggingFiles(false); }}
              onDrop={(event) => { if (!event.dataTransfer.files.length) return; event.preventDefault(); setDraggingFiles(false); acceptPhoneFiles(event.dataTransfer.files, "drop"); }}
              onSubmit={(event) => {
                event.preventDefault();
                sendPhoneMessage(false);
              }}
            >
              <input ref={attachmentPicker} className="ad-visually-hidden" type="file" multiple aria-label="Choose files to attach" onChange={(event) => { if (event.target.files) acceptPhoneFiles(event.target.files, "picker"); event.target.value = ""; }} />
              <DraftAttachments files={attachments} onRemove={(id) => saveAttachments(attachments.filter((file) => file.id !== id))} disabled={busy} />
              <label className="phone-field">
                <VisuallyHidden>{agent.state === "waiting_input" && !pending ? "Reply" : "Message"}</VisuallyHidden>
                <AutoGrowTextarea rows={3} maxHeight="40vh" value={text} placeholder={offline ? "Reconnect to send a message" : "Ask for changes…"} onChange={(event) => { setText(event.target.value); setChatDraft(agentId, event.target.value); }} onPaste={(event) => {
                  const images = [...event.clipboardData.items].filter((item) => item.kind === "file" && item.type.startsWith("image/")).map((item) => item.getAsFile()).filter((file): file is File => !!file);
                  if (images.length) { acceptPhoneFiles(images, "paste"); if (!event.clipboardData.getData("text/plain")) event.preventDefault(); }
                }} />
              </label>
              {error && <p className="phone-error">{error}</p>}
              {notice && <p className="phone-meta">{notice}</p>}
              {attachmentError && <p className="phone-error" role="alert">{attachmentError}</p>}
                {!attachmentsReady && !attachmentError && <p className="phone-meta" role="status">Restoring attachments…</p>}
                {retryPhoneSend && <p className="phone-meta" role="status">Previous message was not confirmed. {retryPhoneSend.retryable === false ? "Use the recovered message above to decide whether to send it again." : <button type="button" disabled={!attachmentsReady || busy || offline} onClick={() => sendPhoneMessage(retryPhoneSend.steer, retryPhoneSend.requestId, retryPhoneSend.revision, retryPhoneSend.files, retryPhoneSend.text)}>Retry previous send</button>}</p>}
                {attachments.some((file) => isPreviewImage(file) && agent.attachment_capabilities?.image_input === false) && <p className="phone-error">This runtime does not accept image input. Remove the image or switch runtime.</p>}
                {attachments.some((file) => !isPreviewImage(file) && agent.attachment_capabilities?.files === false) && <p className="phone-error">This runtime cannot read uploaded files. Remove the file or switch runtime.</p>}
              <div className="phone-actions phone-composer-actions">
                <button type="button" disabled={offline || busy} onClick={() => attachmentPicker.current?.click()}>Attach</button>
                {agent.steering_available && isBusy && (
                  <button type="button" disabled={!attachmentsReady || offline || busy || releaseBlocked || (!text.trim() && attachments.length === 0) || attachments.some((file) => isPreviewImage(file) ? agent.attachment_capabilities?.image_input === false : agent.attachment_capabilities?.files === false)} onClick={() => sendPhoneMessage(true)}>
                    Steer
                  </button>
                )}
                <button type="submit" className="phone-send-icon ad-button-icon-primary" aria-label="Send" title="Send" disabled={!attachmentsReady || offline || busy || releaseBlocked || (!text.trim() && attachments.length === 0) || attachments.some((file) => isPreviewImage(file) ? agent.attachment_capabilities?.image_input === false : agent.attachment_capabilities?.files === false)}>
                  <PhoneIcon name="up" size={14} />
                </button>
              </div>
            </form>
          )}
        </>
      )}
      <div className="phone-actions phone-controls">
        {agent.running && isBusy && (
          <button type="button" disabled={offline || busy} onClick={() => void act(() => cancelTurn(agentId, { target: "turn" }))}>
            Cancel turn
          </button>
        )}
      </div>
      </div>}
      <ConfirmDialog
        open={stopOpen}
        title="Stop this agent?"
        confirmLabel="Stop"
        destructive
        pending={busy}
        confirmDisabled={offline}
        onCancel={() => setStopOpen(false)}
        onConfirm={() => void act(() => stopAgent(agentId), false, () => setStopOpen(false))}
      >
        <p>Stop this agent on your Mac? You can still view its transcript and recorded output.</p>
        {error && <p className="phone-error" role="alert">{error}</p>}
      </ConfirmDialog>
    </div>
  );
}

export function AgentManagement({ agent, offline, busy, act, rename, setRename, error, action }: { agent: AgentState; offline: boolean; busy: boolean; act: (fn: () => Promise<unknown>, clears?: boolean, after?: (result: unknown) => void) => void; rename: string; setRename: (value: string) => void; error: string | null; action?: "rename" | "group" | "runtime" }) {
  const { data: roles } = useRoles();
  const options = useQuery({ queryKey: ["runtime-options"], queryFn: getRuntimeOptions });
  const [runtime, setRuntime] = useState({ backend: agent.backend, model: agent.model, effort: agent.effort ?? "" });
  const allAgents = useConnection((state) => state.agents);
  const groups = useMemo(() => Object.values(allAgents).filter((item) => !item.archived).map((item) => item.group?.trim()).filter((label): label is string => !!label), [allAgents]);
  const [group, setGroup] = useState(agent.group ?? "");
  const [archiveConfirm, setArchiveConfirm] = useState(false);
  // A runtime change from either device restarts the switch draft from the
  // agent's live runtime (FS-20.R17).
  useEffect(() => setRuntime({ backend: agent.backend, model: agent.model, effort: agent.effort ?? "" }), [agent.backend, agent.model, agent.effort]);
  useEffect(() => setGroup(agent.group ?? ""), [agent.agent_id, agent.group]);
  const findModel = (backendID: string, modelID: string) => options.data?.backends.find((item) => item.id === backendID)?.models.find((item) => item.id === modelID);
  const backend = options.data?.backends.find((item) => item.id === runtime.backend);
  const model = findModel(runtime.backend, runtime.model);
  const liveModel = findModel(agent.backend, agent.model);
  const disable = offline || busy || !agent.running || agent.interface !== "chat";
  // Choosing a model always starts from that model's default effort.
  const selectModel = (backendID: string, modelID: string) => setRuntime({ backend: backendID, model: modelID, effort: findModel(backendID, modelID)?.default_effort ?? "" });
  const selectBackend = (backendID: string) => {
    const next = options.data?.backends.find((item) => item.id === backendID);
    selectModel(backendID, (next?.models.find((item) => item.id === next.default_model) ?? next?.models[0])?.id ?? "");
  };
  return (
    <section className="phone-section phone-manage" aria-label="Agent management">
      {(!action || action === "rename") && <form className="phone-card phone-form phone-manage-rename" onSubmit={(event) => { event.preventDefault(); if (rename.trim()) act(() => renameAgent(agent.agent_id, rename.trim()), false, () => setRename("")); }}>
        <label className="phone-field">Name<input value={rename} placeholder={agent.name || personaIdentityLabel(agent.role, roles?.[agent.role])} onChange={(event) => setRename(event.target.value)} /></label>
        <button type="submit" disabled={offline || busy || !rename.trim()}>Rename</button>
      </form>}
      {(!action || action === "group") && <form className="phone-card phone-form phone-manage-group" onSubmit={(event) => { event.preventDefault(); act(() => updateAgentIdentity(agent.agent_id, { group: group.trim() }), false, () => setGroup(group.trim())); }}>
        <label className="phone-field">Group<GroupPicker id="phone-manage-group" value={group} groups={groups} onChange={setGroup} disabled={offline || busy} /></label>
        <button type="submit" disabled={offline || busy}>Save group</button>
      </form>}
      {(!action || action === "runtime") && agent.running && agent.interface === "chat" && (
        <form className="phone-card phone-form" onSubmit={(event) => { event.preventDefault(); act(() => switchRuntime(agent.agent_id, runtime)); }}>
          <p className="phone-card-kicker">Runtime</p>
          <label className="phone-field">Backend<select value={runtime.backend} onChange={(event) => selectBackend(event.target.value)}><option value="">Choose…</option>{(options.data?.backends ?? []).map((item) => <option key={item.id} value={item.id}>{item.name || item.id}</option>)}</select></label>
          <label className="phone-field">Model<select value={runtime.model} onChange={(event) => selectModel(runtime.backend, event.target.value)}>{(backend?.models ?? []).map((item) => <option key={item.id} value={item.id}>{item.name || item.id}</option>)}</select></label>
          {(model?.efforts.length ?? 0) > 0 && <label className="phone-field">Effort<select value={runtime.effort} onChange={(event) => setRuntime({ ...runtime, effort: event.target.value })}><option value="">Model default</option>{model!.efforts.map((effort) => <option key={effort} value={effort}>{effort}</option>)}</select></label>}
          <button type="submit" disabled={disable || !runtime.backend || !runtime.model}>Switch runtime</button>
        </form>
      )}
      {!action && <>
      {agent.fast_available && <label className="phone-card phone-check"><input type="checkbox" checked={agent.fast} disabled={disable} onChange={(event) => act(() => setSessionConfig(agent.agent_id, { fast: event.target.checked }))} /> Fast mode</label>}
      {agent.running && agent.interface === "chat" && (liveModel?.efforts.length ?? 0) > 0 && <label className="phone-card phone-field">Effort<select value={agent.effort ?? ""} disabled={disable} onChange={(event) => act(() => setSessionConfig(agent.agent_id, { effort: event.target.value }))}><option value="">Model default</option>{liveModel!.efforts.map((effort) => <option key={effort} value={effort}>{effort}</option>)}</select></label>}
      <div className="phone-card phone-manage-lifecycle">
        <button type="button" disabled={offline || busy || !agent.clone?.available} title={agent.clone?.reason} onClick={() => act(() => cloneAgent(agent.agent_id), false, (result) => navigate(`/agent/${encodeURIComponent((result as { agent: { agent_id: string } }).agent.agent_id)}`))}>Clone</button>
        {agent.clone && !agent.clone.available && <p className="phone-meta">{agent.clone.reason}</p>}
        <button type="button" className="ad-button-danger" disabled={offline || busy} onClick={() => setArchiveConfirm(true)}>Archive</button>
      </div>
      </>}
      {!archiveConfirm && error && <p className="phone-error" role="alert">{error}</p>}
      <ConfirmDialog
        open={archiveConfirm}
        title="Archive this agent?"
        confirmLabel="Archive agent"
        destructive
        pending={busy}
        confirmDisabled={offline}
        onCancel={() => setArchiveConfirm(false)}
        onConfirm={() => act(() => archiveAgent(agent.agent_id), false, () => navigate(`/project/${encodeURIComponent(agent.project)}`))}
      >
        <p>Restore is available on the desktop.</p>
        {error && <p className="phone-error" role="alert">{error}</p>}
      </ConfirmDialog>
    </section>
  );
}
