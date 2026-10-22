import type { AnnotationBatch, AnnotationDraft, TranscriptEvent } from "../api/types";

// Annotation capture limits shared by every surface that builds a draft
// (FS-13.R2). Two components clipped excerpts with their own copy of this
// function and had already drifted from the server's marker by two runes, so
// this is the one helper both use; the server's `clipAnnotationExcerpt`
// (`internal/server/sessions.go`) stays authoritative and re-clips on send.
export const annotationExcerptLimit = 2000;
export const annotationClippedMarker = "\n… [excerpt clipped]";

export function clipAnnotationExcerpt(value: string, limit = annotationExcerptLimit) {
  const chars = [...value];
  if (chars.length <= limit) return value;
  const marker = [...annotationClippedMarker];
  return chars.slice(0, limit - marker.length).join("") + annotationClippedMarker;
}

// Event capture shared by the desktop context menu, the Think Tank room and the
// phone's Annotate action (TS-08.R133): which events can be annotated, the
// selection that lies inside one event, and the draft built from either.
export function canAnnotate(event: TranscriptEvent) {
  const kind = String(event.kind ?? event.type ?? "");
  return event.seq != null && !["session_meta", "permission_resolved", "turn_end", "annotation"].includes(kind);
}

// The highlighted text, but only when the highlight lives inside the event's host.
export function selectionWithin(host: HTMLElement): string | null {
  const selection = typeof window.getSelection === "function" ? window.getSelection() : null;
  if (!selection || selection.isCollapsed || selection.rangeCount === 0) return null;
  if (!host.contains(selection.getRangeAt(0).commonAncestorContainer)) return null;
  const text = selection.toString();
  return text.trim() ? text : null;
}

// eventDraft captures the selected text when there is one, otherwise the whole
// event. Copy keeps the exact selection; the excerpt stays trimmed (FS-03.R63).
export function eventDraft(event: TranscriptEvent, selected?: string | null): AnnotationDraft {
  if (selected) return { seq: Number(event.seq), excerpt: clipAnnotationExcerpt(selected.trim()), instruction: "" };
  const raw = event.text ?? event.delta ?? event.new_text ?? event.content ?? JSON.stringify(event, null, 2);
  const excerpt = clipAnnotationExcerpt(typeof raw === "string" ? raw : JSON.stringify(raw, null, 2));
  return { seq: Number(event.seq), excerpt, instruction: "" };
}

// annotationAnchor is the label a pending draft shows for where it points
// (FS-13.R22): file and line range for diff lines, otherwise the event.
export function annotationAnchor(draft: AnnotationDraft) {
  if (draft.anchor_kind === "file") {
    if (!draft.start_line) return `File ${draft.path}`;
    const range = draft.end_line && draft.end_line !== draft.start_line ? `–${draft.end_line}` : "";
    return `File ${draft.path}:${draft.start_line}${range}`;
  }
  if (!draft.path) return `Event ${draft.seq}`;
  const range = draft.end_line && draft.end_line !== draft.start_line ? `–${draft.end_line}` : "";
  return `${draft.path}:${draft.start_line}${range}`;
}

// annotationBatch is the one request body every tray sends (FS-13.R4): the
// desktop tray and the phone form differ only in presentation (FS-20.R32).
export function annotationBatch(drafts: AnnotationDraft[], overall: string, agentId?: string, commandId?: string): AnnotationBatch {
  return {
    ...(commandId ? { command_id: commandId } : {}),
    annotations: drafts,
    overall_instruction: overall || undefined,
    target: agentId ? { kind: "agent", agent_id: agentId } : { kind: "self" },
  };
}

// The first line of the machine annotation block `runtime.FormatAnnotationBlock`
// writes (`internal/runtime/event.go`). It is the only part of that format the
// client borrows: the transcript store recognizes a self-targeted send's prompt
// by this prefix (FS-13.R23) instead of respelling the block's layout, which
// would drift the moment either side changed. `internal/server` pins the pair by
// reading this constant and asserting the emitted block still starts with it.
export const annotationBlockSentinel = "[Chuck annotations]";
// Blocks written before the rename carry this line; append-only transcripts keep
// them, so recognition accepts both spellings permanently (FS-13.R24).
const legacyAnnotationBlockSentinel = "[AgentDeck annotations]";

export function isAnnotationBlock(text: string) {
  return text.startsWith(annotationBlockSentinel) || text.startsWith(legacyAnnotationBlockSentinel);
}
