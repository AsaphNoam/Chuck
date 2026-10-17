import { useEffect, useMemo, useRef, useState } from "react";
import { sendAnnotations, validateAnnotations } from "../../api/client";
import type { AgentState } from "../../api/types";
import { annotationBatch } from "../../lib/annotations";
import { useAnnotationStore } from "../../store/annotationStore";

export type AnnotationTarget = "self" | "agent" | "new";

export type AnnotationDelivery = ReturnType<typeof useAnnotationDelivery>;

// useAnnotationDelivery is the one validate/send/launch/retry state machine for a
// source's batch (TS-08.R135). Each entry supplies its own hydrated projection:
// the desktop reads the agent store, the phone its connection. Its transient
// state — target, recipient, pending request, error, launch progress — lives
// with whoever keeps it mounted, so moving the tray between presentations does
// not reset a send in flight or a launched recipient awaiting retry
// (TS-08.R129, R134). Drafts stay in annotationStore.
export function useAnnotationDelivery({ sourceId, sourceActive, source, recipients, connected = true, onLaunchDelivered }: {
  sourceId: string;
  sourceActive: boolean;
  source: AgentState | undefined;
  /** Running chat agents other than the source. */
  recipients: AgentState[];
  /** False while the phone is disconnected: nothing new is sent (FS-13.R29). */
  connected?: boolean;
  /** Runs once a launched agent acknowledged delivery; the launch never navigates. */
  onLaunchDelivered?: (agentId: string) => void;
}) {
  const drafts = useAnnotationStore((state) => state.bySource[sourceId]);
  const overall = useAnnotationStore((state) => state.overallBySource[sourceId] ?? "");
  const discardSent = useAnnotationStore((state) => state.discardSent);
  const [chosenTarget, setTarget] = useState<AnnotationTarget>("self");
  // An inactive source cannot receive its own batch (FS-13.R15), so fall back
  // to an available target rather than leaving a disabled choice selected.
  const target = chosenTarget === "self" && !sourceActive
    ? (recipients.length > 0 || !source ? "agent" : "new")
    : chosenTarget;
  const [recipientId, setRecipientId] = useState("");
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showLaunch, setShowLaunch] = useState(false);
  const [launchedAgentId, setLaunchedAgentId] = useState<string | null>(null);
  // The agent the last acknowledged batch went to; cleared by the next capture.
  const [sentTo, setSentTo] = useState<string | null>(null);
  const deliveredRef = useRef(onLaunchDelivered);
  deliveredRef.current = onLaunchDelivered;

  // A different source starts clean, and a reply for the previous source cannot
  // write into this one or navigate; unmounting the owner invalidates the same
  // way (INV §1). Each callback keeps the source token it was rendered with,
  // which stays live only while that source's owner is mounted.
  const token = useMemo(() => ({ sourceId }), [sourceId]);
  const live = useRef<object | null>(null);
  useEffect(() => {
    live.current = token;
    setTarget("self");
    setRecipientId("");
    setSending(false);
    setError(null);
    setShowLaunch(false);
    setLaunchedAgentId(null);
    setSentTo(null);
    return () => { live.current = null; };
  }, [token]);
  const current = () => live.current === token;
  const count = drafts?.length ?? 0;
  useEffect(() => {
    if (count > 0) setSentTo(null);
  }, [count]);

  const batch = (agentId?: string) => annotationBatch(drafts ?? [], overall, agentId);

  // An acknowledged batch is cleared even after its owner went away, since it
  // was really sent; only UI state and navigation belong to the live owner.
  const deliver = async (agentId: string | undefined, launched: boolean) => {
    const sent = drafts ?? [];
    setSending(true);
    setError(null);
    try {
      await sendAnnotations(sourceId, batch(agentId));
      discardSent(sourceId, sent);
      if (!current()) return;
      setLaunchedAgentId(null);
      setSentTo(agentId ?? sourceId);
      if (launched) deliveredRef.current?.(agentId!);
    } catch (err) {
      if (!current()) return;
      const reason = err instanceof Error ? err.message : "Sending annotations failed.";
      setError(launched ? `Agent ${agentId} launched, but annotation delivery failed. ${reason} Retry to send to this agent.` : reason);
    } finally {
      if (current()) setSending(false);
    }
  };

  const send = async () => {
    setError(null);
    if (!connected) {
      setError("Reconnect to send annotations.");
      return;
    }
    if (target === "new") {
      if (launchedAgentId) {
        await deliver(launchedAgentId, true);
      } else if (!source) {
        setError("The source agent is no longer available for a prefilled launch.");
      } else {
        // An invalid batch never opens the launch step (FS-13.R9, R30).
        setSending(true);
        try {
          await validateAnnotations(sourceId, batch());
          if (current()) setShowLaunch(true);
        } catch (err) {
          if (current()) setError(err instanceof Error ? err.message : "Could not validate annotations.");
        } finally {
          if (current()) setSending(false);
        }
      }
      return;
    }
    const agentId = target === "agent" ? recipientId : undefined;
    if (target === "agent" && !agentId) {
      setError("Choose a running chat agent.");
      return;
    }
    await deliver(agentId, false);
  };

  // The launched agent is recorded before delivery, so a failed send retries to
  // it without launching another (FS-13.R9, R30). A launch that finishes after
  // its source changed or its owner unmounted sends nothing: the drafts stay.
  const onLaunched = (agentId: string) => {
    if (!current()) return;
    setShowLaunch(false);
    setLaunchedAgentId(agentId);
    void deliver(agentId, true);
  };

  return { sourceId, sourceActive, source, recipients, connected, target, setTarget, recipientId, setRecipientId, sending, error, showLaunch, setShowLaunch, launchedAgentId, sentTo, send, onLaunched };
}
