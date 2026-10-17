import { AutoGrowTextarea } from "../ui";
import type { AnnotationDraft } from "../../api/types";
import { annotationAnchor } from "../../lib/annotations";
import { useAgentStore } from "../../store/agentStore";
import { useAnnotationStore } from "../../store/annotationStore";
import { NewAgentModal } from "../../features/launch/NewAgentModal";
import { useRoles } from "../../api/config";
import { personaIdentityLabel } from "../../lib/personas";
import { useAnnotationDelivery, type AnnotationDelivery } from "./annotationDelivery";

export type { AnnotationDelivery } from "./annotationDelivery";

export function AnnotationTray({ sourceId, sourceActive }: { sourceId: string; sourceActive: boolean }) {
  return <AnnotationTrayView delivery={useDesktopAnnotationDelivery(sourceId, sourceActive)} />;
}

// The desktop's projection for the shared delivery state machine.
export function useDesktopAnnotationDelivery(sourceId: string, sourceActive: boolean) {
  const agents = useAgentStore((state) => state.agents);
  const recipients = Object.values(agents).filter((agent) => agent.agent_id !== sourceId && agent.running && agent.interface === "chat");
  return useAnnotationDelivery({ sourceId, sourceActive, source: agents[sourceId], recipients });
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
