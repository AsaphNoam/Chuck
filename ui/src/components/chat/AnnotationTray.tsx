import { AutoGrowTextarea } from "../ui";
import { useState } from "react";
import { sendAnnotations, validateAnnotations } from "../../api/client";
import type { AnnotationDraft } from "../../api/types";
import { annotationAnchor, annotationBatch } from "../../lib/annotations";
import { useAgentStore } from "../../store/agentStore";
import { useAnnotationStore } from "../../store/annotationStore";
import { NewAgentModal } from "../../features/launch/NewAgentModal";
import { useRoles } from "../../api/config";
import { personaIdentityLabel } from "../../lib/personas";

export function AnnotationTray({ sourceId, sourceActive }: { sourceId: string; sourceActive: boolean }) {
  return <AnnotationTrayView delivery={useAnnotationDelivery(sourceId, sourceActive)} />;
}

export type AnnotationDelivery = ReturnType<typeof useAnnotationDelivery>;

// The tray's transient delivery state: target, recipient, pending request,
// error and launch progress. It lives with whoever keeps it mounted, so moving
// the tray between the docked column and Focus file does not reset a send in
// flight or a launched recipient awaiting retry (TS-08.R129). Drafts stay in
// annotationStore.
export function useAnnotationDelivery(sourceId: string, sourceActive: boolean) {
  const drafts = useAnnotationStore((state) => state.bySource[sourceId] ?? []);
  const overall = useAnnotationStore((state) => state.overallBySource[sourceId] ?? "");
  const discard = useAnnotationStore((state) => state.discard);
  const agents = useAgentStore((state) => state.agents);
  const source = agents[sourceId];
  const recipients = Object.values(agents).filter((agent) => agent.agent_id !== sourceId && agent.running && agent.interface === "chat");
  const [chosenTarget, setTarget] = useState<"self" | "agent" | "new">("self");
  // An inactive source cannot receive its own batch (FS-13.R15), so fall back
  // to an available target rather than leaving a disabled radio selected.
  const target = chosenTarget === "self" && !sourceActive
    ? (recipients.length > 0 || !source ? "agent" : "new")
    : chosenTarget;
  const [recipientId, setRecipientId] = useState("");
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showLaunch, setShowLaunch] = useState(false);
  const [launchedAgentId, setLaunchedAgentId] = useState<string | null>(null);

  const batch = (agentId?: string) => annotationBatch(drafts, overall, agentId);

  const send = async () => {
    setError(null);
    if (target === "new") {
      if (launchedAgentId) {
        await deliverToLaunchedAgent(launchedAgentId);
      } else if (!source) {
        setError("The source agent is no longer available for a prefilled launch.");
      } else {
        setSending(true);
        try {
          await validateAnnotations(sourceId, batch());
          setShowLaunch(true);
        } catch (err) {
          setError(err instanceof Error ? err.message : "Could not validate annotations.");
        } finally {
          setSending(false);
        }
      }
      return;
    }
    const agentId = target === "agent" ? recipientId : undefined;
    if (target === "agent" && !agentId) {
      setError("Choose a running chat agent.");
      return;
    }
    setSending(true);
    try {
      await sendAnnotations(sourceId, batch(agentId));
      discard(sourceId);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Sending annotations failed.");
    } finally {
      setSending(false);
    }
  };

  const deliverToLaunchedAgent = async (agentId: string) => {
    setSending(true);
    setError(null);
    try {
      await sendAnnotations(sourceId, batch(agentId));
      discard(sourceId);
    } catch (err) {
      const reason = err instanceof Error ? ` ${err.message}` : "";
      setError(`Agent ${agentId} launched, but annotation delivery failed.${reason} Retry to send to this agent.`);
    } finally {
      setSending(false);
    }
  };

  const onLaunched = (agentId: string) => {
    setShowLaunch(false);
    setLaunchedAgentId(agentId);
    void deliverToLaunchedAgent(agentId);
  };

  return { sourceId, sourceActive, source, recipients, target, setTarget, recipientId, setRecipientId, sending, error, showLaunch, setShowLaunch, launchedAgentId, send, onLaunched };
}

export function AnnotationTrayView({ delivery }: { delivery: AnnotationDelivery }) {
  const { sourceId, sourceActive, source, recipients, target, setTarget, recipientId, setRecipientId, sending, error, showLaunch, setShowLaunch, launchedAgentId, send, onLaunched } = delivery;
  const { data: roles } = useRoles();
  const drafts = useAnnotationStore((state) => state.bySource[sourceId] ?? []);
  const overall = useAnnotationStore((state) => state.overallBySource[sourceId] ?? "");
  const updateInstruction = useAnnotationStore((state) => state.updateInstruction);
  const remove = useAnnotationStore((state) => state.remove);
  const discard = useAnnotationStore((state) => state.discard);
  const setOverall = useAnnotationStore((state) => state.setOverall);
  const collapsed = useAnnotationStore((state) => state.collapsedBySource[sourceId] ?? false);
  const setCollapsed = useAnnotationStore((state) => state.setCollapsed);

  if (drafts.length === 0) return null;

  return (
    // Docked column or floating overlay is decided by the space around the
    // conversation, so this renders one tray in one shape either way. The collapse
    // control and the collapsed strip are likewise CSS states of that same markup:
    // the overlay hides the control, so narrowing the window can never strand a
    // person with a collapsed tray and no way to open it (FS-13.R20–R21).
    <aside className="annotation-tray" data-ui="annotation-tray" data-state={collapsed ? "collapsed" : "expanded"} aria-label="Pending annotations">
      <header className="annotation-tray-header">
        <div><strong className="annotation-tray-title">Pending annotations</strong><span>{drafts.length}/20</span></div>
        <div>
          <button
            type="button"
            className="annotation-tray-collapse ad-button-icon"
            aria-expanded={!collapsed}
            aria-label={collapsed ? "Expand pending annotations" : "Collapse pending annotations"}
            onClick={() => setCollapsed(sourceId, !collapsed)}
          >
            {collapsed ? "‹" : "›"}
          </button>
          <button type="button" className="annotation-tray-discard ad-button-danger" onClick={() => discard(sourceId)} disabled={sending}>Discard all</button>
        </div>
      </header>
      <div className="annotation-tray-body">
        <ol className="annotation-drafts">
          {drafts.map((draft, index) => <AnnotationDraftRow key={`${draft.seq}-${index}`} draft={draft} index={index} sourceId={sourceId} onRemove={remove} onUpdate={updateInstruction} disabled={sending} />)}
        </ol>
        <label className="annotation-overall">
          Overall instruction (optional)
          <AutoGrowTextarea value={overall} maxLength={2000} onChange={(event) => setOverall(sourceId, event.target.value)} disabled={sending} />
        </label>
      </div>
      <footer className="annotation-tray-footer">
        <div className="annotation-targets">
          <label><input type="radio" checked={target === "self"} disabled={!sourceActive || sending} onChange={() => setTarget("self")} /> Current agent</label>
          <label><input type="radio" checked={target === "agent"} disabled={sending} onChange={() => setTarget("agent")} /> Another agent</label>
          <label><input type="radio" checked={target === "new"} disabled={!source || sending} onChange={() => setTarget("new")} /> New task</label>
          {target === "agent" && (
            <select value={recipientId} onChange={(event) => setRecipientId(event.target.value)} disabled={sending} aria-label="Annotation recipient">
              <option value="">Choose an agent</option>
              {recipients.map((agent) => <option key={agent.agent_id} value={agent.agent_id}>{agent.name} ({personaIdentityLabel(agent.role, roles?.[agent.role])}@{agent.project})</option>)}
            </select>
          )}
        </div>
        {error && <p className="annotation-error">{error}</p>}
        <button type="button" className="annotation-send ad-button-primary" onClick={() => void send()} disabled={sending}>{sending ? "Sending…" : target === "new" ? launchedAgentId ? `Retry delivery to ${launchedAgentId}` : "Continue to launch" : "Send annotations"}</button>
      </footer>
      <NewAgentModal open={showLaunch} onClose={() => setShowLaunch(false)} initialRole={source?.role} initialProject={source?.project} onLaunched={onLaunched} />
    </aside>
  );
}

function AnnotationDraftRow({ draft, index, sourceId, onRemove, onUpdate, disabled }: {
  draft: AnnotationDraft;
  index: number;
  sourceId: string;
  onRemove: (sourceId: string, index: number) => void;
  onUpdate: (sourceId: string, index: number, instruction: string) => void;
  disabled: boolean;
}) {
  const anchor = annotationAnchor(draft);
  return (
    <li className="annotation-draft">
      {/* The anchor is what the reader scans for, so it is the row's heading
          rather than bold text sharing a line with a control (FS-13.R22). */}
      <div className="annotation-draft-head"><h3 className="annotation-draft-anchor">{anchor}</h3><button type="button" className="ad-button-danger" onClick={() => onRemove(sourceId, index)} disabled={disabled}>Remove</button></div>
      <blockquote>{draft.excerpt}</blockquote>
      <label>Instruction<AutoGrowTextarea value={draft.instruction} maxLength={2000} onChange={(event) => onUpdate(sourceId, index, event.target.value)} disabled={disabled} /></label>
    </li>
  );
}
