import { AutoGrowTextarea } from "../ui";
import { SendIcon, StopIcon } from "../ui/icons";
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";
import { cancelTurn } from "../../api/client";
import { invalidateHeldReads, readHeldRecovery, reconcileSteerRecovery, withdrawHeldMessage } from "../../lib/heldMessage";
import { useHeldStore } from "../../store/heldStore";
import { RunningTasks } from "./RunningTasks";
import { matchesPromptDelivery, useTranscriptStore } from "../../store/transcriptStore";
import { bumpChatDraftRevision, getChatDraft, setChatDraft, subscribeChatDraft } from "./drafts";
import { useAutocomplete } from "./autocomplete";
import { useComposerEditing } from "./useComposerEditing";
import { addFiles, attachmentErrorForStorage, hasPersistedAttachmentDraft, isPreviewImage, persistAttachments, restoreAttachments, subscribeAttachmentDraft, touchAttachmentDraft, type DraftAttachment } from "./attachments";
import { useInputDelivery } from "./useInputDelivery";
import { DraftAttachments } from "./AttachmentView";
import type { AttachmentCapabilities } from "../../api/types";

const NO_QUEUED: never[] = [];

// Composer sends to one agent and owns its `@` file / `#` command picker through
// the shared autocomplete hook (FS-03.R30–R34).

function latestTranscriptSeq(agentId: string): number {
  return (useTranscriptStore.getState().byAgent[agentId] ?? []).reduce((max, event) => Math.max(max, event.seq ?? 0), 0);
}

function alreadyDelivered(agentId: string, text: string, beforeSeq: number, requestId?: string): boolean {
  return (useTranscriptStore.getState().byAgent[agentId] ?? []).some((event) =>
    matchesPromptDelivery(event, text, requestId, beforeSeq));
}

export function Composer({ agentId, busy, running = true, steerable = false, variant, attachmentCapabilities }: {
  agentId: string;
  busy: boolean;
  running?: boolean;
  steerable?: boolean;
  variant?: "dashboard";
  attachmentCapabilities?: AttachmentCapabilities;
}) {
  const [text, setText] = useState(() => getChatDraft(agentId));
  const [error, setError] = useState<string | null>(null);
  // Steer's outcome is best-effort by nature and the two outcomes mean different
  // things, so it is reported rather than left to be inferred (FS-03.R50).
  const [notice, setNotice] = useState<string | null>(null);
  const [files, setFiles] = useState<DraftAttachment[]>([]);
  const [filesReady, setFilesReady] = useState(() => !hasPersistedAttachmentDraft(agentId));
  const [attachmentError, setAttachmentError] = useState<string | null>(null);
  const [draggingFiles, setDraggingFiles] = useState(false);
  const held = useHeldStore((state) => state.byAgent[agentId]);
  const heldId = useHeldStore((state) => state.heldIdByAgent[agentId]);
  const heldRequestId = useHeldStore((state) => state.requestIdByAgent[agentId]);
  const queuedInput = useHeldStore((state) => state.queuedByAgent[agentId]) ?? NO_QUEUED;
  const hold = useHeldStore((state) => state.acceptHold);
  const releaseHeld = useHeldStore((state) => state.release);
  const append = useTranscriptStore((state) => state.appendMessage);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const pickerRef = useRef<HTMLInputElement>(null);
  const editing = useComposerEditing(agentId, text, setText, textareaRef);
  const resetEditing = editing.reset;
  const autocomplete = useAutocomplete({
    sourceId: agentId,
    text,
    textareaRef,
    onReplace: editing.replace,
  });
  const resetAutocomplete = autocomplete.reset;
  const replaceDraftBaseline = useCallback((next: string) => {
    resetEditing();
    resetAutocomplete();
    setText(next);
  }, [resetEditing, resetAutocomplete]);

  const { deliver, matchingRetry, retry, sending } = useInputDelivery(agentId, {
    beforeSend: () => latestTranscriptSeq(agentId),
    acceptPrompt: (result, pending, beforeSeq) => {
      const requestId = result.request_id ?? pending.requestId;
      const attachments = result.attachments ?? [];
      if (result.delivery === "held") {
        hold(agentId, pending.text, result.after_seq, attachments, result.held_id, requestId);
        if (result.request_id && !result.held_id) void readHeldRecovery(agentId, () => true, true);
      } else if (result.delivery !== "queued" && (!pending.files.length || result.attachment_delivery === "submitted")
          && !alreadyDelivered(agentId, pending.text, beforeSeq, requestId)) {
        append(agentId, { kind: "user_text", text: pending.text, attachments, message_id: requestId, echo_after_seq: beforeSeq });
      }
    },
    acceptSteer: async (result, promotedId) => {
      if (promotedId !== undefined) releaseHeld(agentId, promotedId);
      await reconcileSteerRecovery(agentId, result.request_id);
    },
    refreshRecovery: (promotedId) => {
      invalidateHeldReads(agentId);
      if (promotedId !== undefined) releaseHeld(agentId, promotedId);
      void readHeldRecovery(agentId, () => true, true);
    },
  }, {
    clearDraft: () => replaceDraftBaseline(""),
    setError, setNotice, setAttachmentError,
  });

  useLayoutEffect(() => editing.flush(autocomplete.syncTrigger));

  useLayoutEffect(() => {
    replaceDraftBaseline(getChatDraft(agentId));
    return subscribeChatDraft(agentId, replaceDraftBaseline);
  }, [agentId, replaceDraftBaseline]);

  useEffect(() => {
    let current = true;
    setError(null);
    setNotice(null);
    setFiles([]);
    const needsAttachmentRestore = hasPersistedAttachmentDraft(agentId);
    setFilesReady(!needsAttachmentRestore);
    setAttachmentError(null);
    if (needsAttachmentRestore) {
      void restoreAttachments(agentId).then((restored) => { if (current) { setFiles(restored); setFilesReady(true); } }).catch(() => { if (current) setAttachmentError("Saved attachments could not be restored. Reload before sending to avoid losing them."); });
    }
    return () => { current = false; };
  }, [agentId]);

  useEffect(() => subscribeAttachmentDraft(agentId, setFiles), [agentId]);

  const saveFiles = (next: DraftAttachment[]) => {
    setFiles(next);
    setAttachmentError(null);
    const indexed = touchAttachmentDraft(agentId, next, text !== "");
    bumpChatDraftRevision(agentId);
    if (!indexed) setAttachmentError(attachmentErrorForStorage());
    void persistAttachments(agentId, next).catch(() => setAttachmentError(attachmentErrorForStorage()));
  };

  const acceptFiles = (list: FileList | File[], source: "picker" | "paste" | "drop") => {
    const result = addFiles(files, list, source, attachmentCapabilities);
    if (result.files !== files) saveFiles(result.files);
    setAttachmentError(result.errors[0] ?? null);
  };

  const onPaste = (event: React.ClipboardEvent<HTMLTextAreaElement>) => {
    const images = [...event.clipboardData.items]
      .filter((item) => item.kind === "file" && item.type.startsWith("image/"))
      .map((item) => item.getAsFile())
      .filter((file): file is File => !!file);
    if (!images.length) return;
    acceptFiles(images, "paste");
    // Keep any available text once through the normal textarea paste behavior.
    if (!event.clipboardData.getData("text/plain")) event.preventDefault();
  };

  const capabilityError = files.some((file) => isPreviewImage(file)
    ? attachmentCapabilities?.image_input === false
    : attachmentCapabilities?.files === false)
    ? files.some((file) => isPreviewImage(file) && attachmentCapabilities?.image_input === false)
      ? "This runtime does not accept image input. Remove the image or choose a runtime that supports it."
      : "This runtime cannot read uploaded files. Remove the file or choose a runtime that supports it."
    : null;

  // A receipt-bearing hold is restored through its server recovery offer.
  // Older holds without receipts still return to an empty composer.
  useEffect(() => {
    if (running || held === undefined) return;
    if (!heldRequestId && held && text === "") {
      replaceDraftBaseline(held);
      setChatDraft(agentId, held);
    }
    releaseHeld(agentId, heldId);
    // `text` is read, not depended on: this runs on the lifecycle boundary, and
    // re-running it on every keystroke would race the person's own typing.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [agentId, running, held, heldId, heldRequestId, releaseHeld]);

  const submit = () => {
    if (!filesReady) { setAttachmentError("Restoring attachments… wait before sending."); return; }
    if (!text.trim() && files.length === 0) return;
    if (capabilityError) { setError(capabilityError); return; }
    void deliver(text, files);
  };

  const steer = () => {
    const typed = text.trim() ? text : "";
    if (!typed && files.length === 0 && !heldId) return;
    if (!filesReady) { setAttachmentError("Restoring attachments… wait before steering."); return; }
    if (capabilityError) { setError(capabilityError); return; }
    void deliver(typed, files, true, !typed && !files.length ? { held: { id: heldId, requestId: heldRequestId } } : {});
  };

  const withdraw = async () => {
    setError(null);
    setNotice(null);
    try {
      await withdrawHeldMessage(agentId);
    } catch (err) {
      const reason = err instanceof Error && err.message ? err.message : "the agent may have stopped";
      setError(`Could not withdraw — ${reason}. It may already have been sent.`);
    }
  };

  // The toolbar's @ and # type the existing trigger at the caret, then let the
  // shared picker see it exactly as if it had been typed (FS-12.R61).
  const insertTrigger = (trigger: "@" | "#") => {
    const textarea = textareaRef.current;
    const start = textarea?.selectionStart ?? text.length;
    const end = textarea?.selectionEnd ?? start;
    const token = (start > 0 && !/\s/.test(text[start - 1]) ? " " : "") + trigger;
    const next = text.slice(0, start) + token + text.slice(end);
    textarea?.focus();
    editing.replace(next, start + token.length);
  };

  const onKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (editing.isComposing(event)) return;
    if (autocomplete.onKeyDown(event)) {
      editing.breakInteraction();
      return;
    }
    if (editing.onKeyDown(event)) return;
    // Picker closed (or showing an empty state): Enter submits, Shift+Enter newlines.
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      void submit();
    }
  };

  return (
    <form className={`composer${draggingFiles ? " is-file-drop-target" : ""}`} data-ui="composer" data-variant={variant} onSubmit={(event) => { event.preventDefault(); void submit(); }}
      onDragEnter={(event) => { if ([...event.dataTransfer.types].includes("Files")) { event.preventDefault(); setDraggingFiles(true); } }}
      onDragOver={(event) => { if ([...event.dataTransfer.types].includes("Files")) event.preventDefault(); }}
      onDragLeave={(event) => { if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setDraggingFiles(false); }}
      onDrop={(event) => { if (!event.dataTransfer.files.length) return; event.preventDefault(); setDraggingFiles(false); acceptFiles(event.dataTransfer.files, "drop"); }}>
      <input ref={pickerRef} className="ad-visually-hidden" type="file" multiple aria-label="Choose files to attach" onChange={(event) => { if (event.target.files) acceptFiles(event.target.files, "picker"); event.target.value = ""; }} />
      <DraftAttachments files={files} onRemove={(id) => saveFiles(files.filter((file) => file.id !== id))} disabled={sending} />
      <div className="composer-input">
        <AutoGrowTextarea
          ref={textareaRef}
          maxHeight="40vh"
          placeholder={variant ? undefined : "Add direction to this session…"}
          value={text}
          {...editing.input}
          onKeyUp={(event) => {
            editing.input.onKeyUp(event);
            if (!editing.isComposing(event) && document.activeElement === event.currentTarget) autocomplete.syncTrigger(event.currentTarget);
          }}
          onClick={(event) => autocomplete.syncTrigger(event.currentTarget)}
          onCompositionStart={(event) => {
            editing.input.onCompositionStart(event);
            autocomplete.clearTrigger();
          }}
          onCompositionEnd={(event) => {
            editing.input.onCompositionEnd(event);
            autocomplete.syncTrigger(event.currentTarget);
          }}
          onKeyDown={onKeyDown}
          onPaste={onPaste}
        />
        {autocomplete.picker}
      </div>
      {/* Send is always present and is the safe default; Steer appears only where
          the live session advertises it, and never as a disabled control where it
          does not (FS-03.R50, FS-09.R26). The toolbar is display: contents outside
          the full agent page, so the dashboard composer keeps its geometry. */}
      <div className="composer-toolbar">
      {!variant && (
        <div className="composer-references">
          <button type="button" className="ad-button-icon" aria-label="Reference a file" title="Reference a file" onClick={() => insertTrigger("@")}>@</button>
          <button type="button" className="ad-button-icon" aria-label="Insert a command" title="Insert a command" onClick={() => insertTrigger("#")}>#</button>
        </div>
      )}
      <div className="composer-actions">
        <RunningTasks key={agentId} agentId={agentId} />
        <button type="button" className="ad-button-secondary composer-attach" aria-label="Attach files" title="Attach files" disabled={sending} onClick={() => pickerRef.current?.click()}>Attach</button>
        <button type="submit" className="composer-icon ad-button-icon-primary" aria-label="Send" title="Send" disabled={!filesReady || sending || (!text.trim() && files.length === 0) || !!capabilityError || (matchingRetry(text, files)?.retryable === false)}>{!variant && <span className="composer-send-label">Send</span>}<SendIcon /></button>
        {busy && steerable && (!queuedInput.length || heldId || text.trim() || files.length > 0) && (
          <button type="button" className="ad-button-secondary" title={sending ? "Wait for the current send to finish." : undefined} disabled={sending || !!capabilityError || (matchingRetry(text, files)?.retryable === false)} onClick={() => void steer()}>Steer</button>
        )}
        {held && (
          <button type="button" className="ad-button-secondary" onClick={() => void withdraw()}>Withdraw queued</button>
        )}
        {busy && (
          <button
            type="button"
            className="composer-icon ad-button-icon"
            aria-label="Cancel"
            title="Cancel"
            onClick={() => {
              setError(null);
              setNotice(null);
              cancelTurn(agentId, { target: "turn" }).catch(() => setError("Failed to cancel — the turn may have already finished."));
            }}
          >
            <StopIcon />
          </button>
        )}
      </div>
      </div>
      {!filesReady && <p className="composer-retry" role="status">Restoring attachments…</p>}
      {retry && <p className="composer-retry" role="status"><span>Previous message was not confirmed.</span> {retry.retryable === false ? <span>Use the recovered message above to decide whether to send it again.</span> : <button type="button" disabled={!filesReady || sending} onClick={() => void deliver(retry.text, retry.files, retry.steer, { retry })}>Retry previous send</button>}</p>}
      {draggingFiles && <p className="composer-drop-hint" aria-live="polite">Drop files to attach</p>}
      {attachmentError && <p className="composer-error" role="alert">{attachmentError}</p>}
      {notice && <p className="composer-notice" role="status">{notice}</p>}
      {(error || capabilityError) && <p className="composer-error">{error || capabilityError}</p>}
    </form>
  );
}
