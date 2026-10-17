import { useEffect, useLayoutEffect, useRef, useState, type MouseEvent, type ReactNode } from "react";
import type { AnnotationDraft, AttachmentDescriptor, TranscriptEvent } from "../../api/types";
import { clipAnnotationExcerpt } from "../../lib/annotations";
import { copyText } from "../../lib/copyText";
import { claimWebLink } from "../../lib/linkActions";
import { ErrorBoundary } from "../ErrorBoundary";
import { Button } from "../ui";
import { AssistantText } from "./renderers/AssistantText";
import { AssistantImageRow, MessageAttachments } from "./AttachmentView";
import { DiffBlock } from "./renderers/DiffBlock";
import { PermissionPrompt } from "./renderers/PermissionPrompt";
import { ToolCall } from "./renderers/ToolCall";
import { ToolResult } from "./renderers/ToolResult";
import { groupTranscriptRows, kindOf, ToolRun } from "./toolRun";
import { TurnError } from "./renderers/TurnError";
import { AnnotationCard } from "./renderers/AnnotationCard";
import { NoticeRow } from "./renderers/NoticeRow";
import { ThinkingDisclosure } from "./renderers/ThinkingDisclosure";
import { ChildActivity } from "./renderers/ChildActivity";
import { BackgroundTaskList } from "./renderers/BackgroundTaskList";
import { collectTasks, hideFileChanges, markBackgrounded, nestActivities, withReasoning, type ChildNode } from "./runtimeActivity";
import { useReasoningStore } from "../../store/reasoningStore";
import { useTranscriptStore } from "../../store/transcriptStore";
import { projectTurns } from "./turnActivity";
import { TurnList, useFocusReturn, useTurnChoices } from "./TurnList";
import { AnnotationContextMenu, type AnnotationMenuState } from "./AnnotationContextMenu";
import { ReaderWorkspace } from "./ReaderWorkspace";
import type { PanelSlots } from "./AgentStage";
import { mergeThinkTankResults, THINK_TANK_RESULT_KIND, ThinkTankResultRow } from "./thinkTankResults";
import { useThinkTankResults } from "../../api/thinkTanks";
import type { FileLink } from "./renderers/filePath";
import { useAnnotationStore } from "../../store/annotationStore";
import { useAgentStore } from "../../store/agentStore";
import { useHeldStore } from "../../store/heldStore";
import { readHeldRecovery, withdrawHeldMessage } from "../../lib/heldMessage";
import { WorkingIndicator } from "./WorkingIndicator";
import { useUiStore } from "../../store/uiStore";
import { dismissPromptRecovery, sendRecoveredPrompt } from "../../api/client";
import { useQueryClient } from "@tanstack/react-query";

// openFile/onOpenFile are per-surface, exactly as annotationsEnabled already is
// (TS-08.R57). The agent and archived-agent screens pass the file their route
// carries and a handler that writes it back to the route; the dashboard chat pane
// passes no open file and a handler that navigates to the agent screen instead,
// which is that pane's existing route to the full surface (FS-03.R53).
// taskControl offers targeted background-task Stop; only a live session whose
// runtime negotiated it passes true, and the archive never does (FS-03.R59).
// reveal names an event the caller is about to scroll to; a fresh object per request.
export function TranscriptView({ agentId, events, sourceActive = false, annotationsEnabled = true, busy = false, openFile = null, onOpenFile, taskControl = false, reveal = null, inlineDiffs = false, panelSlots }: { agentId: string; events: TranscriptEvent[]; sourceActive?: boolean; annotationsEnabled?: boolean; busy?: boolean; openFile?: FileLink | null; onOpenFile?: (link: FileLink | null, options?: { replace?: boolean }) => void; taskControl?: boolean; reveal?: { seq: number } | null; inlineDiffs?: boolean; panelSlots?: PanelSlots }) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const atBottomRef = useRef(true);
  const lastTopRef = useRef(0);
  const [atBottom, setAtBottom] = useState(true);
  const [menu, setMenu] = useState<AnnotationMenuState | null>(null);
  const addAnnotation = useAnnotationStore((state) => state.add);
  const pushError = useUiStore((state) => state.pushError);
  // The queued follow-up renders beside the event list, never inside it: the
  // server sends no event for a message it has not delivered (TS-08.R56).
  const held = useHeldStore((state) => state.byAgent[agentId]);
  const heldAttachments = useHeldStore((state) => state.attachmentsByAgent[agentId] ?? []);
  const recovery = useHeldStore((state) => state.recoveryByAgent[agentId]);
  const recoveryUnread = useHeldStore((state) => state.recoveryUnreadByAgent[agentId]);
  const reasoning = useReasoningStore((state) => state.byAgent[agentId]?.spans);

  // Annotating is a right-click action on the event under the pointer: it captures the
  // highlighted text when there is a selection inside that event, otherwise the whole event.
  const openMenu = (mouse: MouseEvent<HTMLDivElement>, event: TranscriptEvent) => {
    if (!annotationsEnabled || !canAnnotate(event)) return;
    const selected = selectionWithin(mouse.currentTarget);
    mouse.preventDefault();
    // Copy keeps the exact selection; the annotation excerpt stays trimmed (FS-03.R63).
    const draft = selected ? { seq: Number(event.seq), excerpt: clipAnnotationExcerpt(selected.trim()), instruction: "" } : eventDraft(event);
    setMenu({
      x: mouse.clientX,
      y: mouse.clientY,
      label: selected ? "Annotate selection" : "Annotate whole event",
      copy: selected ? () => copyText(selected, pushError) : undefined,
      annotate: () => addAnnotation(agentId, draft),
      link: claimWebLink(mouse),
    });
  };

  // Only a move toward older content leaves the bottom. A row whose height
  // settles after render (a diagram, an image) grows the list, and the browser's
  // compensating scroll is not the reader scrolling away (FS-03.R3).
  const onScroll = () => {
    const el = scrollRef.current;
    if (!el) return;
    const stuck = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
    const pinned = stuck || (atBottomRef.current && el.scrollTop >= lastTopRef.current);
    lastTopRef.current = el.scrollTop;
    atBottomRef.current = pinned;
    setAtBottom(pinned);
    if (pinned && !stuck) el.scrollTop = el.scrollHeight;
  };

  useEffect(() => {
    const el = scrollRef.current;
    if (el && atBottomRef.current) el.scrollTop = el.scrollHeight;
  }, [events, busy, reasoning]);

  // Late-settling rows change no event, so follow the content's own size.
  useEffect(() => {
    const el = scrollRef.current;
    const content = contentRef.current;
    if (!el || !content || typeof ResizeObserver === "undefined") return undefined;
    const observer = new ResizeObserver(() => {
      if (atBottomRef.current) el.scrollTop = el.scrollHeight;
    });
    observer.observe(content);
    return () => observer.disconnect();
  }, []);

  const renderEvents = eventRenderer({
    agentId,
    onAnnotate: (draft) => addAnnotation(agentId, draft),
    onContextMenu: openMenu,
    onOpenFile,
  });

  const results = useThinkTankResults(agentId);
  const settled = useTranscriptStore((state) => Boolean(state.settled[agentId]));
  const merged = withReasoning(markBackgrounded(mergeThinkTankResults(events, results.data, settled)), reasoning);
  // A view without a Files tab keeps diffs inline (FS-03.R87).
  const nested = nestActivities(inlineDiffs ? merged : hideFileChanges(merged, reveal?.seq));
  const choices = useTurnChoices(agentId);
  const trackFocus = useFocusReturn(scrollRef);
  // A Files-tab Diff reveal opens the completed turn hiding that row before the
  // caller scrolls to it.
  const { setOpen } = choices;
  useLayoutEffect(() => {
    if (!reveal) return;
    const turn = projectTurns(nested).find((item) => item.completed && item.seqs.has(reveal.seq));
    if (turn) setOpen(turn.key, true);
    // Only a new reveal request reopens; later updates must not undo a close.
  }, [reveal]);

  const jumpToLatest = () => {
    const el = scrollRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  };

  return (
    <div className="transcript-wrap" data-ui="transcript">
      <div className="transcript-view" data-slot="list" ref={scrollRef} onScroll={onScroll} onFocus={trackFocus}>
        <div ref={contentRef}>
          <TurnList agentId={agentId} events={nested} choices={choices} renderEvents={(list) => renderEvents(list, [], 1)} />
          {results.isError && (
            <p className="form-error" role="status" data-ui="think-tank" data-slot="results-error">
              Think Tank synthesis could not be loaded.{" "}
              <Button type="button" variant="secondary" onClick={() => void results.refetch()}>Retry</Button>
            </p>
          )}
          <BackgroundTaskList agentId={agentId} tasks={collectTasks(events)} controllable={taskControl} />
          {busy && <WorkingIndicator />}
          {held !== undefined && <HeldMessage agentId={agentId} text={held} attachments={heldAttachments} />}
          {recovery && <RecoveryMessage agentId={agentId} recovery={recovery} />}
          {!recovery && recoveryUnread && (
            <div className="transcript-held" data-ui="transcript" data-variant="held">
              <p className="transcript-held-error" role="alert">Withdrawn, but the message and its attachments could not be loaded.</p>
              <div className="transcript-held-actions"><button type="button" className="ad-button-secondary" onClick={() => void readHeldRecovery(agentId)}>Load withdrawn message</button></div>
            </div>
          )}
        </div>
      </div>
      {!atBottom && (
        <button type="button" className="jump-to-latest ad-button-secondary" onClick={jumpToLatest}>
          Jump to latest
        </button>
      )}
      <ReaderWorkspace
        key={agentId}
        agentId={agentId}
        openFile={openFile}
        onOpenFile={onOpenFile}
        annotationsEnabled={annotationsEnabled}
        sourceActive={sourceActive}
        panelSlots={panelSlots}
        onReturnFocus={() => scrollRef.current?.focus()}
      />
      <AnnotationContextMenu menu={menu} onClose={() => setMenu(null)} />
    </div>
  );
}

function RecoveryMessage({ agentId, recovery }: { agentId: string; recovery: { text: string; request_id: string; attachments: AttachmentDescriptor[] } }) {
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const client = useQueryClient();
  const key = `chuck-recovery-send:${agentId}:${recovery.request_id}`;
  const sendId = () => {
    const existing = localStorage.getItem(key);
    if (existing) return existing;
    const id = typeof crypto.randomUUID === "function" ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(36).slice(2)}`;
    localStorage.setItem(key, id);
    return id;
  };
  const send = async () => {
    setBusy(true); setError("");
    try {
      await sendRecoveredPrompt(agentId, recovery.text, recovery.attachments.map((item) => item.id), sendId());
      localStorage.removeItem(key);
      useHeldStore.getState().setRecovery(agentId, null);
      await client.invalidateQueries({ queryKey: ["held", agentId] });
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Could not restore this message."); }
    finally { setBusy(false); }
  };
  const dismiss = async () => {
    setBusy(true); setError("");
    try {
      await dismissPromptRecovery(agentId, recovery.request_id);
      localStorage.removeItem(key);
      useHeldStore.getState().setRecovery(agentId, null);
      await client.invalidateQueries({ queryKey: ["held", agentId] });
    } catch (reason) { setError(reason instanceof Error ? reason.message : "Could not dismiss this message."); }
    finally { setBusy(false); }
  };
  return <div className="transcript-held" data-ui="transcript" data-variant="held">
    <p className="transcript-held-label">Message not delivered — choose whether to send it again</p>
    {recovery.text && <p className="transcript-held-text">{recovery.text}</p>}
    <MessageAttachments agentId={agentId} attachments={recovery.attachments} />
    <div className="transcript-held-actions"><button type="button" className="ad-button-primary" disabled={busy} onClick={() => void send()}>Send recovered message</button><button type="button" className="ad-button-secondary" disabled={busy} onClick={() => void dismiss()}>Dismiss</button></div>
    {error && <p className="transcript-held-error" role="alert">{error}</p>}
  </div>;
}

// eventRenderer is the one row renderer for the root list and every nested
// child (TS-08.R59). A Think Tank room reuses it for each attempt's retained
// activity, scoped to that attempt's source agent (TS-14.R15).
export function eventRenderer({ agentId, onAnnotate, onContextMenu, onOpenFile }: {
  agentId: string;
  onAnnotate: (draft: AnnotationDraft) => void;
  onContextMenu: (mouse: MouseEvent<HTMLDivElement>, event: TranscriptEvent) => void;
  onOpenFile?: (link: FileLink | null) => void;
}) {
  const renderEvents = (list: TranscriptEvent[], ancestry: string[], depth: number): ReactNode =>
    groupTranscriptRows(list).map((row, index) => {
      if (row.kind === "tool-run") {
        return (
          <ToolRun
            key={`run-${keyOf(row.events[0], index)}`}
            events={row.events}
            renderEvent={(event, eventIndex) => (
              <TranscriptEventFrame
                agentId={agentId}
                event={event}
                key={keyOf(event, eventIndex)}
                onAnnotate={onAnnotate}
                onContextMenu={onContextMenu}
                onOpenFile={onOpenFile}
                className="tool-run-event"
              />
            )}
          />
        );
      }
      if (kindOf(row.event) === "activity") {
        return <ChildActivity key={keyOf(row.event, index)} node={row.event.node as ChildNode} ancestry={ancestry} depth={depth} renderEvents={renderEvents} />;
      }
      return (
        <TranscriptEventFrame
          agentId={agentId}
          event={row.event}
          key={keyOf(row.event, index)}
          onAnnotate={onAnnotate}
          onContextMenu={onContextMenu}
          onOpenFile={onOpenFile}
        />
      );
    });
  return renderEvents;
}

// HeldMessage is the pending tail: the message the person submitted while the
// agent was working, which is not yet sent and not yet seen by the agent. It must
// read that way rather than as a sent message awaiting a reply — the failure mode
// is believing the agent already has it — and it carries its own withdraw
// affordance (FS-03.R48, TS-08.R56, INV §8).
function HeldMessage({ agentId, text, attachments }: { agentId: string; text: string; attachments: AttachmentDescriptor[] }) {
  const [error, setError] = useState<string | null>(null);
  return (
    <div className="transcript-held" data-ui="transcript" data-variant="held">
      <p className="transcript-held-label">Queued — sends when this turn ends</p>
      <p className="transcript-held-text">{text}</p>
      <MessageAttachments agentId={agentId} attachments={attachments} />
      <button
        type="button"
        className="transcript-held-withdraw ad-button-secondary"
        onClick={() => {
          setError(null);
          withdrawHeldMessage(agentId).catch(() => setError("Could not withdraw — it may already have been sent."));
        }}
      >
        Withdraw
      </button>
      {error && <p className="transcript-held-error" role="alert">{error}</p>}
    </div>
  );
}

function TranscriptEventFrame({ agentId, event, onAnnotate, onContextMenu, onOpenFile, className = "transcript-item" }: {
  agentId: string;
  event: TranscriptEvent;
  onAnnotate: (draft: AnnotationDraft) => void;
  onContextMenu: (mouse: MouseEvent<HTMLDivElement>, event: TranscriptEvent) => void;
  onOpenFile?: (link: FileLink | null) => void;
  className?: string;
}) {
  return (
    // data-seq lets the Files tab's "Diff" action scroll to this event
    // (present only when the event carries a runtime seq).
    <div
      className={className}
      data-slot="event"
      data-variant={variantOf(event)}
      data-seq={event.seq ?? undefined}
      onContextMenu={(mouse) => onContextMenu(mouse, event)}
    >
      <ErrorBoundary
        label="message"
        fallback={<pre className="tool-block tool-result-error">Failed to render this event.</pre>}
      >
        <TranscriptItem agentId={agentId} event={event} onAnnotate={onAnnotate} onOpenFile={onOpenFile} />
      </ErrorBoundary>
    </div>
  );
}

type TranscriptVariant = "assistant" | "user" | "tool-call" | "tool-result" | "diff" | "permission" | "error" | "turn" | "backend-switch" | "fork-boundary" | "annotation" | "notice" | "thinking" | "unknown";


function variantOf(event: TranscriptEvent): TranscriptVariant {
  const kind = String(event.kind ?? event.type ?? "");
  if (kind === "assistant_text") return "assistant";
  if (kind === "user_text") return "user";
  if (kind === "tool_call") return "tool-call";
  if (kind === "tool_result") return "tool-result";
  if (kind === "diff") return "diff";
  if (kind === "permission_request" || kind === "permission_resolved") return "permission";
  if (kind === "error") return "error";
  if (kind === "turn_end") return "turn";
  if (kind === "backend_switch") return "backend-switch";
  if (kind === "fork_boundary") return "fork-boundary";
  if (kind === "annotation") return "annotation";
  if (kind === "notice") return "notice";
  if (kind === "reasoning") return "thinking";
  return "unknown";
}

// Stable React key: prefer the runtime seq, then a local message_id, then index.
function keyOf(event: TranscriptEvent, index: number) {
  if (event.seq != null) return `s${event.seq}`;
  if (event.message_id) return `m${event.message_id}`;
  return `i${index}`;
}

function TranscriptItem({ agentId, event, onAnnotate, onOpenFile }: { agentId: string; event: TranscriptEvent; onAnnotate: (draft: AnnotationDraft) => void; onOpenFile?: (link: FileLink | null) => void }) {
  const kind = String(event.kind ?? event.type ?? "");
  if (kind === "assistant_text") return <AssistantText event={event} onOpenFile={onOpenFile} agentId={agentId} />;
  if (kind === "user_text")
    return <article className="message user-message" data-ui="transcript" data-variant="user">{String(event.text ?? "")}<MessageAttachments agentId={agentId} attachments={event.attachments ?? []} /></article>;
  if (kind === "assistant_image") return <article className="message assistant-message" data-ui="transcript" data-variant="assistant"><AssistantImageRow agentId={agentId} attachment={event.attachment} path={typeof event.path === "string" ? event.path : undefined} name={typeof event.name === "string" ? event.name : undefined} error={typeof event.error === "string" ? event.error : undefined} /></article>;
  if (kind === "permission_request") return <PermissionPrompt agentId={agentId} event={event} />;
  if (kind === "diff") return <DiffBlock event={event} onAnnotate={onAnnotate} onOpenFile={onOpenFile} />;
  if (kind === "tool_call") return <ToolCall event={event} />;
  if (kind === "tool_result") return <ToolResult event={event} />;
  if (kind === "error") return <TurnError event={event} />;
  if (kind === "annotation") return <AnnotationCard event={event} />;
  if (kind === "notice") return <NoticeRow event={event} />;
  if (kind === "reasoning") return <ThinkingDisclosure text={String(event.text ?? "")} activityId={event.activity_id} />;
  if (kind === "turn_end") return <hr className="turn-end" />;
  if (kind === THINK_TANK_RESULT_KIND) return <ThinkTankResultRow event={event} />;
  if (kind === "backend_switch") {
    const from = String(event.from ?? "");
    const to = String(event.to ?? "");
    return <div className="backend-switch-divider">{from} {"->"} {to}</div>;
  }
  if (kind === "fork_boundary") return <ForkBoundary sourceId={String(event.forked_from_agent_id ?? "")} />;
  // permission_resolved is folded into its prompt by the store; nothing to render.
  if (kind === "permission_resolved" || kind === "session_meta") return null;
  return <pre className="tool-block">{JSON.stringify(event, null, 2)}</pre>;
}

// ForkBoundary marks where a clone's copied history ends (FS-01.R36).
function ForkBoundary({ sourceId }: { sourceId: string }) {
  const source = useAgentStore((state) => state.agents[sourceId]?.name);
  return <div className="backend-switch-divider">Cloned from {source || "another agent"}</div>;
}

export function canAnnotate(event: TranscriptEvent) {
  const kind = String(event.kind ?? event.type ?? "");
  return event.seq != null && !["session_meta", "permission_resolved", "turn_end", "annotation"].includes(kind);
}

// The highlighted text, but only when the highlight lives inside the right-clicked event.
export function selectionWithin(host: HTMLElement): string | null {
  const selection = typeof window.getSelection === "function" ? window.getSelection() : null;
  if (!selection || selection.isCollapsed || selection.rangeCount === 0) return null;
  if (!host.contains(selection.getRangeAt(0).commonAncestorContainer)) return null;
  const text = selection.toString();
  return text.trim() ? text : null;
}

export function eventDraft(event: TranscriptEvent): AnnotationDraft {
  const raw = event.text ?? event.delta ?? event.new_text ?? event.content ?? JSON.stringify(event, null, 2);
  const excerpt = clipAnnotationExcerpt(typeof raw === "string" ? raw : JSON.stringify(raw, null, 2));
  return { seq: Number(event.seq), excerpt, instruction: "" };
}
