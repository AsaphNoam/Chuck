import type { TranscriptEvent } from "../../../api/types";
import { MarkedSourceText } from "../searchPassage";

export function AnnotationCard({ event }: { event: TranscriptEvent }) {
  const annotations = Array.isArray(event.annotations) ? event.annotations as Array<Record<string, unknown>> : [];
  const target = event.target as Record<string, unknown> | undefined;
  return (
    <article className="annotation-card" data-ui="transcript" data-variant="annotation">
      <header><strong>Annotations assigned</strong><span>{target?.kind === "self" ? "Current agent" : target?.agent_id ? `Agent ${String(target.agent_id)}` : "Agent"}</span></header>
      {annotations.map((annotation, index) => (
        <section key={index}>
          <p>{annotation.anchor_kind === "file"
            ? `File ${String(annotation.path ?? "")}${annotation.start_line ? `:${String(annotation.start_line)}${annotation.end_line && annotation.end_line !== annotation.start_line ? `–${String(annotation.end_line)}` : ""}` : ""}`
            : annotation.path ? `${String(annotation.path)}:${String(annotation.start_line ?? "")}${annotation.end_line && annotation.end_line !== annotation.start_line ? `–${String(annotation.end_line)}` : ""}` : `Event ${String(annotation.seq ?? "")}`}</p>
          <blockquote><MarkedSourceText text={String(annotation.excerpt ?? "")} partIndex={index*2} /></blockquote>
          <p><strong>Instruction:</strong> <MarkedSourceText text={String(annotation.instruction ?? "")} partIndex={index*2+1} /></p>
        </section>
      ))}
      {event.overall_instruction ? <p className="annotation-card-overall"><strong>Overall:</strong> <MarkedSourceText text={String(event.overall_instruction)} partIndex={annotations.length*2} /></p> : null}
    </article>
  );
}
