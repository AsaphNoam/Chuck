import { useContext } from "react";
import type { TranscriptEvent } from "../../../api/types";
import { AnnotationLinkContext } from "../annotationNavigation";
import { MarkedSourceText } from "../searchPassage";

/** The recipient agent id of an annotation batch assigned to another agent, for name lookup. */
export function annotationTargetId(event: TranscriptEvent): string | undefined {
  const target = event.target as Record<string, unknown> | undefined;
  return target?.kind !== "self" && typeof target?.agent_id === "string" ? target.agent_id : undefined;
}

/** recipientName is the caller's current name for the target; a retained card whose recipient is
 *  no longer known reads neutrally rather than showing its id (FS-02.R69). */
export function AnnotationCard({ event, recipientName }: { event: TranscriptEvent; recipientName?: string }) {
  const link = useContext(AnnotationLinkContext);
  const annotations = Array.isArray(event.annotations) ? event.annotations as Array<Record<string, unknown>> : [];
  const target = event.target as Record<string, unknown> | undefined;
  return (
    <article className="annotation-card" data-ui="transcript" data-variant="annotation">
      <header><strong>Annotations assigned</strong><span>{target?.kind === "self" ? "Current agent" : recipientName?.trim() || "Another agent"}</span></header>
      {annotations.map((annotation, index) => {
        const excerpt = String(annotation.excerpt ?? "");
        const side = annotation.side;
        const line = (side === "old" || side === "new") && typeof annotation.start_line === "number"
          ? { side, number: annotation.start_line } as const
          : undefined;
        const showSource = annotation.anchor_kind !== "file" && typeof annotation.seq === "number"
          ? link?.(annotation.seq, excerpt, line)
          : undefined;
        const quote = <MarkedSourceText text={excerpt} partIndex={index*2} />;
        return (
          <section key={index}>
            <p>{annotation.anchor_kind === "file"
              ? `File ${String(annotation.path ?? "")}${annotation.start_line ? `:${String(annotation.start_line)}${annotation.end_line && annotation.end_line !== annotation.start_line ? `–${String(annotation.end_line)}` : ""}` : ""}`
              : annotation.path ? `${String(annotation.path)}:${String(annotation.start_line ?? "")}${annotation.end_line && annotation.end_line !== annotation.start_line ? `–${String(annotation.end_line)}` : ""}` : `Event ${String(annotation.seq ?? "")}`}</p>
            <blockquote>{showSource ? (
              <button type="button" className="annotation-source-link" onClick={showSource} aria-label={`Jump to source: ${excerpt}`}>{quote}</button>
            ) : quote}</blockquote>
            <p><strong>Instruction:</strong> <MarkedSourceText text={String(annotation.instruction ?? "")} partIndex={index*2+1} /></p>
          </section>
        );
      })}
      {event.overall_instruction ? <p className="annotation-card-overall"><strong>Overall:</strong> <MarkedSourceText text={String(event.overall_instruction)} partIndex={annotations.length*2} /></p> : null}
    </article>
  );
}
