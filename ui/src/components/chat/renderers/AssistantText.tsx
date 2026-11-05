import type { TranscriptEvent } from "../../../api/types";
import { SanitizedMarkdown } from "./SanitizedMarkdown";
import type { FileLink } from "./filePath";

export function AssistantText({ event, onOpenFile, agentId, onOpenInBrowser }: { event: TranscriptEvent; onOpenFile?: (link: FileLink) => void; agentId?: string; onOpenInBrowser?: (path: string) => Promise<unknown> }) {
  const text = String(event.text ?? event.delta ?? "");
  return (
    <article className="message assistant-message" data-ui="transcript" data-variant="assistant">
      <SanitizedMarkdown text={text} onOpenFile={onOpenFile} agentId={agentId} onOpenInBrowser={onOpenInBrowser} />
    </article>
  );
}
