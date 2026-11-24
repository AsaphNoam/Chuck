import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import type { AgentState } from "../../api/types";
import { readHeldRecovery } from "../../lib/heldMessage";
import { useHeldStore } from "../../store/heldStore";
import { matchesPromptDelivery, useTranscriptStore } from "../../store/transcriptStore";

// Every live desktop chat surface — the agent screen and the expanded dashboard
// pane — hydrates and settles the held follow-up the same way (INV §2, §10).
export function useHeldProjection(id: string, agent: Pick<AgentState, "running" | "state" | "interface"> | undefined) {
  const events = useTranscriptStore((state) => state.byAgent[id]);
  const held = useHeldStore((state) => state.byAgent[id]);
  const heldRequestId = useHeldStore((state) => state.requestIdByAgent[id]);
  const heldAttachments = useHeldStore((state) => state.attachmentsByAgent[id]);
  const heldAfterSeq = useHeldStore((state) => state.afterSeqByAgent[id] ?? 0);
  const releaseHeld = useHeldStore((state) => state.release);

  useQuery({
    queryKey: ["held", id, "desktop", agent?.running, agent?.state],
    queryFn: async ({ signal }) => {
      await readHeldRecovery(id, () => !signal.aborted, true);
      return true;
    },
    enabled: agent?.interface === "chat",
    initialData: false,
    initialDataUpdatedAt: 0,
    refetchOnMount: "always",
  });

  // A held message stops being pending the moment the server actually sends it,
  // which is when it enters the durable transcript as an ordinary user prompt.
  // Matching the sequenced server event rather than the local echo is what keeps
  // the pending tail from clearing before delivery (FS-03.R48, TS-08.R56).
  useEffect(() => {
    if (!held && !heldAttachments?.length) return;
    // A held annotation batch's prompt is quieted under FS-13.R23; its card
    // records where that prompt landed instead.
    const delivered = (events ?? []).some((event) => matchesPromptDelivery(event, held ?? "", heldRequestId, heldAfterSeq));
    if (delivered) releaseHeld(id);
  }, [events, held, heldAttachments, heldRequestId, heldAfterSeq, id, releaseHeld]);
}
