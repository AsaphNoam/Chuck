import { useEffect, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { getTranscript, restoreAgent, resumeAgent } from "../../api/client";
import { useProjects } from "../../api/config";
import { AgentBreadcrumb, AgentIdentity } from "../../components/chat/ChatPanel";
import { AgentStage, type PanelSlots } from "../../components/chat/AgentStage";
import { useTranscriptStore } from "../../store/transcriptStore";
import { useAgentStore } from "../../store/agentStore";
import { TranscriptView } from "../../components/chat/TranscriptView";
import type { FileLink } from "../../components/chat/renderers/filePath";
import { fileLinkFromParams, writeFileLinkParams } from "../../lib/fileLinkParams";

export function ArchiveAgentPage() {
  const { id = "" } = useParams();
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const events = useTranscriptStore((state) => state.byAgent[id] ?? []);
  const setTranscript = useTranscriptStore((state) => state.setTranscript);
  const settle = useTranscriptStore((state) => state.settle);
  const agent = useAgentStore((state) => state.agents[id]);
  const [resuming, setResuming] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [panelSlots, setPanelSlots] = useState<PanelSlots | undefined>(undefined);
  // Resume and runtime switch append newer session_meta boundaries, and Resume
  // uses the most recent one. Reading the first record would show the original
  // backend/model beside Resume for a switched session (FS-05.R31/A15).
  const metadata = latestSessionMeta(events);
  // An archived session reads from the working directory it recorded, so its file
  // links open the same viewer through the same address (FS-03.R54/R55).
  const openFile = fileLinkFromParams(params);
  const openFileInViewer = (link: FileLink | null, options?: { replace?: boolean }) =>
    setParams((current) => writeFileLinkParams(current, link), options);
  const archivedName = textField(metadata?.name) || "Archived session";
  const project = textField(metadata?.project);
  const backend = textField(metadata?.backend);
  const model = textField(metadata?.model);
  const effort = textField(metadata?.effort);
  const fast = metadata?.fast === true;
  const createdAt = textField(metadata?.created_at);
  const { data: projects } = useProjects();
  const projectTitle = (project && projects?.[project]?.title) || project;

  // The read outcome is surfaced, never swallowed: an unknown id offers no
  // Resume/Restore, and a known session without a readable transcript says so
  // (FS-05.R42, INV §7/§8).
  const [load, setLoad] = useState<TranscriptLoad>({ state: "loading" });
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let current = true;
    setLoad({ state: "loading" });
    getTranscript(id, true)
      .then((r) => {
        setTranscript(r.agent_id, r.events);
        if (current) setLoad({ state: r.events.length ? "ready" : "empty" });
      })
      .catch((err: unknown) => {
        settle(id);
        if (!current) return;
        if ((err as { status?: number }).status === 404) setLoad({ state: "unknown" });
        else setLoad({ state: "failed", message: err instanceof Error ? err.message : "Could not load the transcript" });
      });
    return () => {
      current = false;
    };
  }, [id, attempt, setTranscript, settle]);
  const unknown = load.state === "unknown";

  const doResume = async () => {
    setResuming(true);
    setError(null);
    try {
      await resumeAgent(id);
      navigate(`/agent/${id}`);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Resume failed");
      setResuming(false);
    }
  };

  const doRestore = async () => {
    setResuming(true);
    setError(null);
    try {
      await restoreAgent(id);
      navigate(`/project/${agent?.project ?? project}`);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Restore failed");
      setResuming(false);
    }
  };

  return (
    <section className="chat-panel" data-ui="agent-workspace" data-state="archived" data-file-open={openFile ? "true" : undefined} data-variant="chat">
      <AgentBreadcrumb backTarget="/archive" backLabel="Back to Archive" project={projectTitle} name={archivedName} agentId={id} />
      <AgentStage source={id} onSlots={setPanelSlots}>
      <div className="chat-card">
      <header className="chat-header" data-slot="header">
        <AgentIdentity
          name={unknown ? "Session not found" : archivedName}
          state="archived"
          status={(
            <span className="archive-readonly-label">
              {unknown ? "Not found" : "Archived · read-only"}{createdAt && <> · <time dateTime={createdAt}>{formatTimestamp(createdAt)}</time></>}
            </span>
          )}
          details={[projectTitle, backend, model, effort, (backend || model) && (fast ? "Fast mode" : "Normal speed")]}
        />
        {!unknown && (
          <button
            type="button"
            className="resume-btn ad-button-primary"
            disabled={resuming}
            onClick={() => void (agent?.archived ? doRestore() : doResume())}
          >
            {resuming ? (agent?.archived ? "Restoring…" : "Resuming…") : agent?.archived ? "Restore" : "Resume"}
          </button>
        )}
      </header>
      {error ? <p className="archive-error">{error}</p> : <LoadNotice load={load} onRetry={() => setAttempt((n) => n + 1)} />}
      <div data-slot="content">{!unknown && <TranscriptView agentId={id} events={events} sourceActive={false} annotationsEnabled={agent?.interface !== "terminal"} openFile={openFile} onOpenFile={openFileInViewer} inlineDiffs panelSlots={panelSlots} />}</div>
      {/* No Composer — read-only view */}
      <div />
      </div>
      </AgentStage>
    </section>
  );
}

type TranscriptLoad = { state: "loading" | "ready" | "empty" | "unknown" } | { state: "failed"; message: string };

function LoadNotice({ load, onRetry }: { load: TranscriptLoad; onRetry: () => void }) {
  if (load.state === "unknown") return <p className="archive-error" role="alert">No session with this id exists. It may have been deleted.</p>;
  if (load.state === "empty") return <p className="archive-error" role="status">This session's transcript is unavailable — no recorded history was found.</p>;
  if (load.state === "failed") {
    return (
      <p className="archive-error" role="alert">
        Could not load the transcript: {load.message}{" "}
        <button type="button" className="ad-button-secondary" onClick={onRetry}>Retry</button>
      </p>
    );
  }
  return null;
}

// latestSessionMeta returns the newest session-metadata boundary in a transcript
// — the identity a Resume would actually restore.
export function latestSessionMeta<T extends { kind?: unknown; type?: unknown }>(events: T[]): T | undefined {
  for (let index = events.length - 1; index >= 0; index -= 1) {
    if ((events[index].kind ?? events[index].type) === "session_meta") return events[index];
  }
  return undefined;
}

function textField(value: unknown) {
  return typeof value === "string" ? value : "";
}

function formatTimestamp(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}
