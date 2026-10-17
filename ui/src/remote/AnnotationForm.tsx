import { AutoGrowTextarea } from "../components/ui";
import { useEffect, useRef } from "react";
import type { AgentState, AnnotationDraft } from "../api/types";
import { annotationAnchor } from "../lib/annotations";
import { useAnnotationStore } from "../store/annotationStore";
import { useAnnotationDelivery, type AnnotationDelivery } from "../components/chat/annotationDelivery";
import { useConnection } from "./connection";
import { useRoles } from "../api/config";
import { personaIdentityLabel } from "../lib/personas";
import { navigate } from "./router";
import { PhoneSheet } from "./PhoneSheet";
import { NewAgentForm } from "./NewAgentForm";

const noDrafts: AnnotationDraft[] = [];

/** The phone's projection for the shared delivery state machine (TS-08.R135).
 *  A batch sent to a newly launched agent opens that agent only after its
 *  delivery is acknowledged (FS-13.R30). */
export function usePhoneAnnotationDelivery(sourceId: string, agent: AgentState | undefined) {
  const agents = useConnection((state) => state.agents);
  const connected = useConnection((state) => state.link === "connected");
  const recipients = Object.values(agents).filter(
    (other) => other.agent_id !== sourceId && other.running && other.interface === "chat" && !other.archived,
  );
  return useAnnotationDelivery({
    sourceId,
    sourceActive: !!agent?.running,
    source: agent,
    recipients,
    connected,
    onLaunchDelivered: (agentId) => navigate(`/agent/${encodeURIComponent(agentId)}`),
  });
}

/** The pending-count control shown in Chat and Files (TS-08.R134). */
export function PhoneAnnotationButton({ sourceId, onOpen }: { sourceId: string; onOpen: () => void }) {
  const count = useAnnotationStore((state) => state.bySource[sourceId]?.length ?? 0);
  if (count === 0) return null;
  return (
    <button type="button" className="phone-annotate-open" aria-haspopup="dialog" onClick={onOpen}>
      Pending annotations <span className="phone-annotate-count">{count}</span>
    </button>
  );
}

/** Success names the recipient; the tray itself is gone with its drafts. */
export function PhoneAnnotationSent({ delivery }: { delivery: AnnotationDelivery }) {
  const agents = useConnection((state) => state.agents);
  const count = useAnnotationStore((state) => state.bySource[delivery.sourceId]?.length ?? 0);
  if (!delivery.sentTo || count > 0) return null;
  const name = delivery.sentTo === delivery.sourceId ? "this agent" : agents[delivery.sentTo]?.name || delivery.sentTo;
  return <p className="phone-meta" role="status">Annotations sent to {name}.</p>;
}

/** The one tray sheet and its New task launch step. The owner keeps `delivery`
 *  above the Chat/Files tabs, so closing the sheet or switching tabs keeps the
 *  target, a launched recipient and any send in flight (TS-08.R134). */
export function PhoneAnnotations({ delivery, open, focusNewest, onClose }: {
  delivery: AnnotationDelivery;
  open: boolean;
  /** Set by a capture: the newest instruction receives focus. */
  focusNewest: number;
  onClose: () => void;
}) {
  const { sourceId, source, showLaunch, setShowLaunch, onLaunched } = delivery;
  const { data: roles } = useRoles();
  const count = useAnnotationStore((state) => state.bySource[sourceId]?.length ?? 0);
  if (showLaunch && source) {
    return (
      <PhoneSheet title="New agent" centered onClose={() => setShowLaunch(false)}>
        <NewAgentForm
          project={source.project}
          initialRole={source.role}
          chooseProject
          onClose={() => setShowLaunch(false)}
          onLaunched={onLaunched}
          summary={`Launches a chat agent, then sends these annotations from ${source.name || personaIdentityLabel(source.role, roles?.[source.role])} to it.`}
        />
      </PhoneSheet>
    );
  }
  if (!open || count === 0) return null;
  return <PhoneAnnotationSheet delivery={delivery} focusNewest={focusNewest} onClose={onClose} />;
}

function PhoneAnnotationSheet({ delivery, focusNewest, onClose }: { delivery: AnnotationDelivery; focusNewest: number; onClose: () => void }) {
  const { data: roles } = useRoles();
  const { sourceId, sourceActive, source, recipients, connected, target, setTarget, recipientId, setRecipientId, sending, error, launchedAgentId, send } = delivery;
  const drafts = useAnnotationStore((state) => state.bySource[sourceId]) ?? noDrafts;
  const overall = useAnnotationStore((state) => state.overallBySource[sourceId] ?? "");
  const updateInstruction = useAnnotationStore((state) => state.updateInstruction);
  const setOverall = useAnnotationStore((state) => state.setOverall);
  const remove = useAnnotationStore((state) => state.remove);
  const discard = useAnnotationStore((state) => state.discard);
  const list = useRef<HTMLOListElement>(null);

  // A new capture brings its instruction into reach once the sheet has opened.
  useEffect(() => {
    if (!focusNewest) return undefined;
    const frame = requestAnimationFrame(() => {
      const fields = list.current?.querySelectorAll("textarea");
      fields?.[fields.length - 1]?.focus();
    });
    return () => cancelAnimationFrame(frame);
  }, [focusNewest]);

  const launched = launchedAgentId ? recipients.find((agent) => agent.agent_id === launchedAgentId)?.name || launchedAgentId : null;
  const sendLabel = sending ? "Sending…" : target === "new" ? launched ? `Retry delivery to ${launched}` : "Continue to launch" : "Send annotations";

  return (
    <PhoneSheet
      title={`Pending annotations · ${drafts.length}/20`}
      className="phone-annotate-sheet"
      onClose={onClose}
      actions={<button type="button" className="phone-annotate-discard ad-button-danger" disabled={sending} onClick={() => discard(sourceId)}>Discard</button>}
    >
      <div className="phone-annotate-body">
        <ol ref={list} className="phone-annotate-drafts">
          {drafts.map((draft, index) => (
            <li key={`${draft.seq}-${draft.path ?? ""}-${index}`} className="phone-annotate-draft">
              <div className="phone-annotate-head">
                <h3>{annotationAnchor(draft)}</h3>
                <button type="button" className="phone-link ad-button-danger" disabled={sending} onClick={() => remove(sourceId, index)}>
                  Remove
                </button>
              </div>
              <pre className="phone-pre">{draft.excerpt}</pre>
              <label className="phone-field">
                Instruction
                <AutoGrowTextarea rows={2} maxHeight="30vh" maxLength={2000} value={draft.instruction} disabled={sending} onChange={(event) => updateInstruction(sourceId, index, event.target.value)} />
              </label>
            </li>
          ))}
        </ol>
        <label className="phone-field">
          Overall instruction (optional)
          <AutoGrowTextarea rows={2} maxHeight="30vh" maxLength={2000} value={overall} disabled={sending} onChange={(event) => setOverall(sourceId, event.target.value)} />
        </label>
      </div>
      <div className="phone-annotate-footer">
        <div className="phone-actions" role="radiogroup" aria-label="Send to">
          {(
            [
              ["self", "This agent"],
              ["agent", "Another agent"],
              ["new", "New task"],
            ] as const
          ).map(([value, label]) => (
            <button
              key={value}
              type="button"
              role="radio"
              aria-checked={target === value}
              className={target === value ? "phone-target-selected" : undefined}
              disabled={sending || (value === "self" && !sourceActive) || (value === "new" && !source)}
              onClick={() => setTarget(value)}
            >
              {label}
            </button>
          ))}
        </div>
        {!sourceActive && <p className="phone-meta">This agent is stopped, so it cannot receive its own annotations.</p>}
        {target === "agent" &&
          (recipients.length > 0 ? (
            <label className="phone-field">
              Agent
              <select value={recipientId} disabled={sending} onChange={(event) => setRecipientId(event.target.value)}>
                <option value="">Choose a running chat agent</option>
                {recipients.map((other) => (
                  <option key={other.agent_id} value={other.agent_id}>
                    {other.name || `${personaIdentityLabel(other.role, roles?.[other.role])}@${other.project}`}
                  </option>
                ))}
              </select>
            </label>
          ) : (
            <p className="phone-empty">No other chat agent is running.</p>
          ))}
        {target === "new" && !launched && <p className="phone-meta">Checks the batch, then lets you review the new agent before it launches.</p>}
        {!connected && <p className="phone-meta" role="status">Reconnect to send. Your annotations stay here.</p>}
        {error && <p className="phone-error" role="alert">{error}</p>}
        <button type="button" className="ad-button-primary phone-annotate-send" disabled={sending || !connected} onClick={() => void send()}>
          {sendLabel}
        </button>
      </div>
    </PhoneSheet>
  );
}
