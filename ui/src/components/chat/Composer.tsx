import { AutoGrowTextarea } from "../ui";
import { SendIcon, StopIcon } from "../ui/icons";
import { useEffect, useRef, useState } from "react";
import { cancelTurn, sendPrompt, steerPrompt } from "../../api/client";
import { withdrawHeldMessage } from "../../lib/heldMessage";
import { useHeldStore } from "../../store/heldStore";
import { useTranscriptStore } from "../../store/transcriptStore";
import { bumpChatDraftRevision, chatDraftRevision, discardChatDraft, getChatDraft, setChatDraft, subscribeChatDraft } from "./drafts";
import { useAutocomplete } from "./autocomplete";
import { addFiles, attachmentErrorForStorage, chatDraftRequestId, clearPendingAttachmentSend, hasPersistedAttachmentDraft, isDefiniteSendRefusal, isPreviewImage, persistAttachments, persistPendingAttachmentSend, removeAcceptedAttachments, restoreAttachments, restorePendingAttachmentSend, subscribeAttachmentDraft, touchAttachmentDraft, type DraftAttachment } from "./attachments";
import { DraftAttachments } from "./AttachmentView";
import type { AttachmentCapabilities } from "../../api/types";

// Composer sends to one agent and owns its `@` file / `#` command picker through
// the shared autocomplete hook (FS-03.R30–R34).

function latestTranscriptSeq(agentId: string): number {
  return (useTranscriptStore.getState().byAgent[agentId] ?? []).reduce((max, event) => Math.max(max, event.seq ?? 0), 0);
}

function alreadyDelivered(agentId: string, text: string, beforeSeq: number, requestId?: string): boolean {
  return (useTranscriptStore.getState().byAgent[agentId] ?? []).some((event) =>
    event.kind === "user_text" && (requestId && event.request_id
      ? event.request_id === requestId
      : event.seq != null && event.seq > beforeSeq && event.text === text));
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
  const [steering, setSteering] = useState(false);
  const [files, setFiles] = useState<DraftAttachment[]>([]);
  const [filesReady, setFilesReady] = useState(() => !hasPersistedAttachmentDraft(agentId));
  const [attachmentError, setAttachmentError] = useState<string | null>(null);
  const [draggingFiles, setDraggingFiles] = useState(false);
  const [sending, setSending] = useState(false);
  const [retry, setRetry] = useState<{ text: string; files: DraftAttachment[]; requestId: string; steer: boolean; revision: number; retryable?: boolean } | null>(null);
  const held = useHeldStore((state) => state.byAgent[agentId]);
  const hold = useHeldStore((state) => state.hold);
  const releaseHeld = useHeldStore((state) => state.release);
  const append = useTranscriptStore((state) => state.appendMessage);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const currentAgentId = useRef(agentId);
  const steeringAgents = useRef(new Set<string>());
  const pickerRef = useRef<HTMLInputElement>(null);
  const inFlight = useRef(false);
  currentAgentId.current = agentId;
  const autocomplete = useAutocomplete({
    sourceId: agentId,
    text,
    textareaRef,
    onText: (next) => {
      setText(next);
      setChatDraft(agentId, next);
    },
  });

  useEffect(() => {
    let current = true;
    const initialRevision = chatDraftRevision(agentId);
    setText(getChatDraft(agentId));
    setError(null);
    setNotice(null);
    autocomplete.reset();
    setSteering(steeringAgents.current.has(agentId));
    setFiles([]);
    const needsAttachmentRestore = hasPersistedAttachmentDraft(agentId);
    setFilesReady(!needsAttachmentRestore);
    setAttachmentError(null);
    if (needsAttachmentRestore) {
      void restoreAttachments(agentId).then((restored) => { if (current) { setFiles(restored); setFilesReady(true); } }).catch(() => { if (current) setAttachmentError("Saved attachments could not be restored. Reload before sending to avoid losing them."); });
    }
    void restorePendingAttachmentSend(agentId).then(async (pending) => {
      if (current && pending && !inFlight.current) {
        const liveFiles = await restoreAttachments(agentId);
        const revision = chatDraftRevision(agentId);
        const sameFiles = pending.files.length === liveFiles.length && pending.files.every((file) => liveFiles.some((live) => live.id === file.id));
        const sameDraft = getChatDraft(agentId) === pending.text && sameFiles;
        setRetry({ ...pending, revision: revision === initialRevision && sameDraft ? revision : -1 });
      }
    }).catch(() => { if (current) setAttachmentError("A previous attachment send could not be restored from browser storage."); });
    return () => { current = false; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [agentId]);

  useEffect(() => subscribeChatDraft(agentId, setText), [agentId]);
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

  const onSubmitFiles = async (prompt: string, selected: DraftAttachment[], requestId: string, steeringMessage = false, expectedRevision?: number) => {
    if (inFlight.current || !filesReady) return;
    inFlight.current = true;
    const beforeSeq = latestTranscriptSeq(agentId);
    setSending(true);
    const submittedRevision = expectedRevision ?? chatDraftRevision(agentId);
    const pending = { text: prompt, files: selected, requestId, steer: steeringMessage, revision: submittedRevision };
    setRetry(pending);
    try { await persistPendingAttachmentSend(agentId, pending); } catch { setAttachmentError(attachmentErrorForStorage()); }
    setError(null);
    setNotice(null);
    try {
      const requestFiles = selected.map(({ name, blob }) => ({ name, blob }));
      const result = steeringMessage
        ? await steerPrompt(agentId, prompt, requestFiles, requestId)
        : await sendPrompt(agentId, prompt, requestFiles, requestId);
      const acceptedAttachments = "attachments" in result ? result.attachments ?? [] : [];
      const acceptedRequestId = "request_id" in result ? result.request_id : undefined;
      if (!steeringMessage && "delivery" in result && result.delivery === "held") hold(agentId, prompt, result.after_seq, acceptedAttachments);
      const attachmentStatus = "attachment_delivery" in result ? result.attachment_delivery : undefined;
      if (!steeringMessage && (!selected.length || attachmentStatus === "submitted") &&
          !alreadyDelivered(agentId, prompt, beforeSeq, acceptedRequestId ?? requestId)) {
        append(agentId, { kind: "user_text", text: prompt, attachments: acceptedAttachments, message_id: acceptedRequestId ?? requestId });
      }
      if (selected.length && attachmentStatus !== "submitted" && attachmentStatus !== "held") {
        setError(attachmentStatus === "released"
          ? "This message was released for recovery. Restore it from the conversation before sending again."
          : attachmentStatus === "failed"
            ? "The message was not delivered. Its attachments remain in the composer; retry the same send."
            : "Delivery could not be confirmed. Its attachments remain available; retry uses the same request receipt.");
        const retryable = attachmentStatus !== "released";
        const pendingRetry = { text: prompt, files: selected, requestId, steer: steeringMessage, revision: submittedRevision, retryable };
        setRetry(pendingRetry);
        if (!retryable) void persistPendingAttachmentSend(agentId, pendingRetry).catch(() => setAttachmentError(attachmentErrorForStorage()));
        return;
      }
      if (chatDraftRevision(agentId) === submittedRevision) {
        discardChatDraft(agentId);
        setText("");
      }
      if (selected.length) {
        try { await removeAcceptedAttachments(agentId, selected.map((file) => file.id), getChatDraft(agentId) !== ""); }
        catch { setAttachmentError("Message was accepted, but attachment cleanup could not be saved. Remove any sent files before sending again."); }
      }
      setRetry(null);
      void clearPendingAttachmentSend(agentId, requestId).catch(() => setAttachmentError("Message was sent, but its retry receipt could not be cleared from browser storage."));
      if (steeringMessage) setNotice("Delivered into the running turn.");
    } catch (err) {
      const reason = err instanceof Error && err.message ? err.message : "the agent may have stopped";
      if (isDefiniteSendRefusal(err)) {
        setError(`Could not send — ${reason}.`);
        setRetry(null);
        try { await clearPendingAttachmentSend(agentId, requestId); } catch { setAttachmentError(attachmentErrorForStorage()); }
        return;
      }
      setError(`Could not send — ${reason}. Your message is still available to retry.`);
      setRetry({ text: prompt, files: selected, requestId, steer: steeringMessage, revision: submittedRevision });
    } finally {
      inFlight.current = false;
      setSending(false);
    }
  };

  // Stop, crash, or archive leaves no next turn for a held message, so it comes
  // back as the person's own text — but only into an empty composer, never over
  // text typed since, which is R36's newer-draft rule applied to the same store
  // (FS-03.R49, TS-08.R56).
  useEffect(() => {
    if (running || !held) return;
    if (text === "") {
      setText(held);
      setChatDraft(agentId, held);
    }
    releaseHeld(agentId);
    // `text` is read, not depended on: this runs on the lifecycle boundary, and
    // re-running it on every keystroke would race the person's own typing.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [agentId, running, held, releaseHeld]);

  const submit = async () => {
    // Reject whitespace-only drafts (R19) but send the text exactly as displayed:
    // a picker insertion's trailing space is part of the prompt and must reach both
    // the optimistic event and POST /prompt verbatim (FS-03.R31/R33, INV §1).
    const prompt = text;
    if (!filesReady) { setAttachmentError("Restoring attachments… wait before sending."); return; }
    if (!prompt.trim() && files.length === 0) return;
    if (capabilityError) { setError(capabilityError); return; }
    if (retry?.retryable === false && retry.text === prompt && sameFileSet(retry.files, files)) return;
    if (retry?.retryable !== false && retry && retry.text === prompt && sameFileSet(retry.files, files)) {
      void onSubmitFiles(retry.text, retry.files, retry.requestId, retry.steer, retry.revision);
      return;
    }
    if (files.length) {
      void onSubmitFiles(prompt, files, chatDraftRequestId(agentId) || crypto.randomUUID());
      return;
    }
    setError(null);
    setNotice(null);
    // A busy agent queues the message, and a queued message is not in the
    // transcript until the server actually sends it — so there is nothing to
    // echo yet, and the pending tail carries it instead (FS-03.R48, TS-08.R56).
    // Capture the draft generation this send owns. If the person edits the draft
    // for this agent while the request is in flight, the completion must leave
    // that newer draft alone instead of discarding or overwriting it (INV §1/§5).
    const draftRev = chatDraftRevision(agentId);
    const beforeSeq = latestTranscriptSeq(agentId);
    setText("");
    autocomplete.clearTrigger();
    try {
      // A stopped chat agent is woken by this same request (FS-03.R35), so the
      // composer submits normally and only the reply takes longer to start.
      const result = await sendPrompt(agentId, prompt);
      // The server's answer decides whether this ran or is waiting, so the
      // pending state is rendered from what happened rather than inferred from
      // agent status (TS-03.R38).
      if (result.delivery === "held") hold(agentId, prompt, result.after_seq);
      else if (!alreadyDelivered(agentId, prompt, beforeSeq)) append(agentId, { kind: "user_text", text: prompt, message_id: `local-${Date.now()}` });
      // Only clear the draft we actually sent; a newer same-agent draft survives.
      if (chatDraftRevision(agentId) === draftRev) discardChatDraft(agentId);
    } catch (err) {
      // A newer same-agent draft typed while the request was pending wins: leave
      // it and its live composer untouched rather than restoring the sent text.
      if (chatDraftRevision(agentId) !== draftRev) return;
      // Surface the server's own typed error — a rejected wake included — and
      // restore the draft so the user can retry; the optimistic bubble stays,
      // but the error makes clear it was not delivered.
      const reason = err instanceof Error && err.message ? err.message : "the agent may have stopped";
      setChatDraft(agentId, prompt);
      // The request belongs to the chat that started it. Do not overwrite a
      // different chat the person opened while this request was in flight.
      if (currentAgentId.current === agentId) {
        setError(`Failed to send — ${reason}. Your message was restored.`);
        setText(prompt);
      }
    }
  };

  // Steer delivers into the turn that is running instead of queueing behind it.
  // With the composer empty and a message already held, it delivers that one, so
  // a person who queued and then changed their mind about waiting does not
  // retype it (FS-03.R50).
  const steer = async () => {
    const typed = text.trim() ? text : "";
    if (!typed && files.length === 0 && !held) return;
    if (!filesReady) { setAttachmentError("Restoring attachments… wait before steering."); return; }
    if (capabilityError) { setError(capabilityError); return; }
    if (steeringAgents.current.has(agentId)) return;
    const steeringAgentId = agentId;
    steeringAgents.current.add(steeringAgentId);
    setSteering(true);
    setError(null);
    setNotice(null);
    try {
      if (files.length) {
        await onSubmitFiles(typed, files, chatDraftRequestId(agentId) || crypto.randomUUID(), true);
        return;
      }
      const result = await steerPrompt(steeringAgentId, typed);
      if (typed) {
        setText("");
        autocomplete.clearTrigger();
        discardChatDraft(agentId);
      } else {
        releaseHeld(agentId);
      }
      setNotice(result.outcome === "steered"
        ? "Delivered into the running turn."
        : "That turn had already ended — sent as a new turn.");
    } catch (err) {
      // A refused message is not downgraded to a queue and not discarded: the
      // composer keeps exactly what the person typed, with the reason (INV §8).
      const reason = err instanceof Error && err.message ? err.message : "the agent may have stopped";
      setError(`Could not steer — ${reason}.`);
    } finally {
      steeringAgents.current.delete(steeringAgentId);
      if (currentAgentId.current === steeringAgentId) setSteering(false);
    }
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
    setText(next);
    setChatDraft(agentId, next);
    requestAnimationFrame(() => {
      if (!textarea) return;
      textarea.focus();
      textarea.setSelectionRange(start + token.length, start + token.length);
      autocomplete.syncTrigger(textarea);
    });
  };

  const onKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (autocomplete.onKeyDown(event)) return;
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
          onChange={(event) => {
            const next = event.target.value;
            setText(next);
            setChatDraft(agentId, next);
            autocomplete.syncTrigger(event.target);
          }}
          onKeyUp={(event) => autocomplete.syncTrigger(event.currentTarget)}
          onClick={(event) => autocomplete.syncTrigger(event.currentTarget)}
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
        <button type="button" className="ad-button-secondary composer-attach" aria-label="Attach files" title="Attach files" disabled={sending} onClick={() => pickerRef.current?.click()}>Attach</button>
        <button type="submit" className="composer-icon ad-button-icon-primary" aria-label="Send" title="Send" disabled={!filesReady || sending || (!text.trim() && files.length === 0) || !!capabilityError || (retry?.retryable === false && retry.text === text && sameFileSet(retry.files, files))}>{!variant && <span className="composer-send-label">Send</span>}<SendIcon /></button>
        {busy && steerable && (
          <button type="button" className="ad-button-secondary" disabled={steering || !!capabilityError || (retry?.retryable === false && retry.text === text && sameFileSet(retry.files, files))} onClick={() => void steer()}>Steer</button>
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
      {retry && <p className="composer-retry" role="status"><span>Previous message was not confirmed.</span> {retry.retryable === false ? <span>Use the recovered message above to decide whether to send it again.</span> : <button type="button" disabled={!filesReady || sending} onClick={() => void onSubmitFiles(retry.text, retry.files, retry.requestId, retry.steer, retry.revision)}>Retry previous send</button>}</p>}
      {draggingFiles && <p className="composer-drop-hint" aria-live="polite">Drop files to attach</p>}
      {attachmentError && <p className="composer-error" role="alert">{attachmentError}</p>}
      {notice && <p className="composer-notice" role="status">{notice}</p>}
      {(error || capabilityError) && <p className="composer-error">{error || capabilityError}</p>}
    </form>
  );
}

function sameFileSet(left: DraftAttachment[], right: DraftAttachment[]) {
  return left.length === right.length && left.every((file) => right.some((candidate) => candidate.id === file.id));
}
