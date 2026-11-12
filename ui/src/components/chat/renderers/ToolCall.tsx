import type { TranscriptEvent } from "../../../api/types";
import { toolLine, toolLineSuffix } from "../toolLine";
import { useThoughtScope } from "../TurnList";
import { MailDisclosure, mailOf } from "./MailDetail";

// One line per call with its result folded in; the line is the deepest detail
// shown, so there are no arguments or output body (FS-03.R108). A call left
// without a result by a finished turn or child is not still running.
export function ToolCall({ event, result }: { event: TranscriptEvent; result?: TranscriptEvent }) {
  const live = useThoughtScope()?.live ?? true;
  const line = toolLine(event, result);
  const suffix = line.state === "running" && !live ? "" : toolLineSuffix(line);
  const mail = mailOf(event);
  const text = (
    <p className="tool-block tool-call" data-ui="tool-call" data-state={line.state} title={line.label}>
      <span className="tool-call-label">{line.label}</span>
      {suffix && <span className="tool-call-state"> · {suffix}</span>}
    </p>
  );
  // A matched send_message/check_messages call keeps its one-line label and
  // gains the mail disclosure (TS-08.R161).
  if (!mail) return text;
  return <div className="tool-call-mail">{text}<MailDisclosure mail={mail} /></div>;
}
