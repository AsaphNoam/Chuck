import { createContext, useContext, useEffect, useRef, useState } from "react";
import { editLastMessage, getEditStatus, getTranscript, type EditResult } from "../../api/client";
import type { TranscriptEvent } from "../../api/types";
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
  const [text, setText] = useState(String(event.text ?? ""));
  const [phase, setPhase] = useState<Phase>("idle");
  const [error, setError] = useState<string | null>(null);
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
      requestId.current = null;
      setPhase("idle");
      setError(result.reason || "The edit did not apply.");
    } else {
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
