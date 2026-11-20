import { createContext, useContext, useEffect, useRef, useState } from "react";
import { acknowledgeEdit, editLastMessage, getEditStatus, getTranscript, type EditResult } from "../../api/client";
import type { AgentState, TranscriptEvent } from "../../api/types";
import { useAgentStore } from "../../store/agentStore";
import { useTranscriptStore } from "../../store/transcriptStore";
import { AutoGrowTextarea, Button } from "../ui";
import { MessageAttachments } from "./AttachmentView";

/** Which message is open for Edit, owned by the TranscriptView that shows it. */
export const MessageEditContext = createContext<{ seq: number | null; close: () => void }>({ seq: null, close: () => undefined });

const CONFLICT = "This conversation changed. Reload the latest messages to edit again.";

type Phase = "idle" | "sending" | "waiting" | "unknown";

function errorCode(err: unknown) {
  return (err as { code?: string } | null)?.code ?? "";
}

function errorStatus(err: unknown) {
  return (err as { status?: number } | null)?.status;
}

function errorText(err: unknown) {
  return err instanceof Error ? err.message : String(err);
}

/**
 * MessageEditor replaces the latest sent message's body with an inline editor
 * (FS-01.R61, TS-08.R171). Save posts one idempotent request; the editor stays
 * until the revised conversation is read back, or reports why the edit did not
 * apply. The ordinary composer draft is never touched.
 */
export function MessageEditor({ agentId, event }: { agentId: string; event: TranscriptEvent }) {
  const { close } = useContext(MessageEditContext);
  const seq = event.seq as number;
  // An earlier edit of this message that did not apply seeds the editor. Its
  // offer ends only when a resubmission is accepted (or on Discard), so Cancel
  // or a reload never loses the text (FS-01.R61).
  const recovered = useAgentStore((state) => {
    const offer = state.agents[agentId]?.edit_recovery;
    return offer?.message_seq === seq ? offer : undefined;
  });
  const [text, setText] = useState(recovered?.text ?? String(event.text ?? ""));
  const [phase, setPhase] = useState<Phase>("idle");
  const [error, setError] = useState<string | null>(recovered ? recovered.reason || "The earlier edit did not apply." : null);
  const recoveredId = useRef(recovered?.request_id);
  const [conflict, setConflict] = useState(false);
  const requestId = useRef<string | null>(null);
  const savedRevision = useRef(0);
  const alive = useRef(true);
  const revision = useAgentStore((state) => state.agents[agentId]?.history_revision ?? 0);
  const operation = useAgentStore((state) => state.agents[agentId]?.edit_operation);
  const setTranscript = useTranscriptStore((state) => state.setTranscript);
  const revisionRead = useTranscriptStore((state) => state.revisionRead[agentId] ?? 0);

  useEffect(() => { alive.current = true; return () => { alive.current = false; }; }, []);

  const apply = (result: EditResult) => {
    if (!alive.current) return;
    if (result.phase === "aborted") {
      // The offer stays until a later save is accepted, surviving a reload.
      recoveredId.current = result.request_id;
      requestId.current = null;
      setPhase("idle");
      setError(result.reason || "The edit did not apply.");
    } else {
      const superseded = recoveredId.current;
      if (superseded && superseded !== result.request_id) {
        recoveredId.current = undefined;
        void acknowledgeEdit(agentId, superseded).catch(() => undefined);
      }
      setError(null);
      setPhase("waiting");
    }
  };

  const checkStatus = async () => {
    if (!requestId.current) return;
    setPhase("sending");
    setError(null);
    try {
      apply(await getEditStatus(agentId, requestId.current));
    } catch (err) {
      if (!alive.current) return;
      if (errorStatus(err) === 404) {
        setPhase("idle");
        setError("The edit was not received. You can save again.");
        return;
      }
      setPhase("unknown");
      setError(`Could not confirm whether the edit was saved: ${errorText(err)}`);
    }
  };

  const save = async () => {
    if (phase !== "idle" || !text.trim()) return;
    if (!requestId.current) requestId.current = crypto.randomUUID();
    savedRevision.current = revision;
    setPhase("sending");
    setError(null);
    setConflict(false);
    try {
      apply(await editLastMessage(agentId, { request_id: requestId.current, message_seq: seq, expected_revision: revision, text }));
    } catch (err) {
      if (!alive.current) return;
      const code = errorCode(err);
      if (code === "edit_conflict" || code === "edit_unavailable") {
        requestId.current = null;
        setPhase("idle");
        if (code === "edit_conflict") setConflict(true);
        else setError(errorText(err));
        return;
      }
      // The outcome is unknown: ask before allowing another Save.
      setPhase("unknown");
      void checkStatus();
    }
  };

  const reload = async () => {
    try {
      const transcript = await getTranscript(agentId);
      setTranscript(transcript.agent_id, transcript.events);
      if (alive.current) setConflict(false);
    } catch (err) {
      if (alive.current) setError(`Could not reload the conversation: ${errorText(err)}`);
    }
  };

  // While waiting, the snapshot names only an open edit; once ours is no
  // longer open, read its result: an abort keeps the text with its reason,
  // and a committed edit whose revision event was missed refetches history.
  const ownOpen = operation?.request_id === requestId.current;
  useEffect(() => {
    if (phase !== "waiting" || ownOpen) return undefined;
    const timer = window.setTimeout(() => {
      const id = requestId.current;
      if (!id) return;
      getEditStatus(agentId, id).then((result) => {
        if (!alive.current) return;
        if (result.phase === "aborted") apply(result);
        else if (result.phase === "committed") void reload().then(close);
      }).catch(() => undefined);
    }, 1500);
    return () => window.clearTimeout(timer);
  }, [phase, ownOpen, agentId]);

  // The revised conversation has been read back after its revision event.
  useEffect(() => {
    if (phase === "waiting" && revisionRead > savedRevision.current) close();
  }, [phase, revisionRead, close]);

  const working = phase === "sending" || phase === "waiting";
  return (
    <article
      className="message user-message message-editor"
      data-ui="transcript"
      data-variant="user"
      onKeyDown={(keyboard) => { if (keyboard.key === "Escape" && !working) { keyboard.stopPropagation(); close(); } }}
    >
      <AutoGrowTextarea
        className="message-editor-input"
        aria-label="Edit message"
        autoFocus
        maxHeight="40vh"
        value={text}
        disabled={working}
        onChange={(change) => setText(change.target.value)}
      />
      <MessageAttachments agentId={agentId} attachments={event.attachments ?? []} />
      <p className="message-editor-note">Saving replaces this message and its reply. Tool actions and file changes already made stay in effect.</p>
      {phase === "waiting" && <p className="message-editor-status" role="status">Editing…</p>}
      {conflict && <p className="message-editor-error form-error" role="alert">{CONFLICT}</p>}
      {error && <p className="message-editor-error form-error" role="alert">{error}</p>}
      <div className="message-editor-actions">
        <Button type="button" variant="primary" disabled={!text.trim() || phase !== "idle"} onClick={() => void save()}>Save and regenerate</Button>
        {phase === "unknown" && <Button type="button" onClick={() => void checkStatus()}>Check status</Button>}
        {conflict && <Button type="button" onClick={() => void reload()}>Reload messages</Button>}
        <Button type="button" disabled={working} onClick={close}>Cancel</Button>
      </div>
    </article>
  );
}

/**
 * EditRecoveryNotice offers back an edit that did not apply, from the agent
 * snapshot rather than client memory, so it survives a reload (FS-01.R61).
 * Edit again reopens the editor, which revalidates the selected message.
 */
export function EditRecoveryNotice({ agentId, recovery, canEdit, onEdit }: {
  agentId: string;
  recovery: NonNullable<AgentState["edit_recovery"]>;
  canEdit: boolean;
  onEdit: () => void;
}) {
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const discard = async () => {
    setBusy(true);
    setError("");
    try {
      await acknowledgeEdit(agentId, recovery.request_id);
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="transcript-held" data-ui="transcript" data-variant="held">
      <p className="transcript-held-label">Your edit was not applied{recovery.reason ? `: ${recovery.reason}` : "."}</p>
      {recovery.text && <p className="transcript-held-text">{recovery.text}</p>}
      {!canEdit && <p className="transcript-held-label">This message can no longer be edited; copy the text if you still need it.</p>}
      <div className="transcript-held-actions">
        {canEdit && <button type="button" className="ad-button-primary" disabled={busy} onClick={onEdit}>Edit again</button>}
        <button type="button" className="ad-button-secondary" disabled={busy} onClick={() => void discard()}>Discard</button>
      </div>
      {error && <p className="transcript-held-error" role="alert">{error}</p>}
    </div>
  );
}
