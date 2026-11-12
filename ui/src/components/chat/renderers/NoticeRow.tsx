import type { TranscriptEvent } from "../../../api/types";
import { Badge } from "../../ui";
import { MailDisclosure, mailOf } from "./MailDetail";

// A runtime advisory (FS-03.R68, TS-08.R86): one compact row, never a message
// bubble. The server already bounds the text and maps unknown severity to info.
export function NoticeRow({ event }: { event: TranscriptEvent }) {
  const warning = event.severity === "warning";
  const chuckOrigin = event.origin === "chuck";
  const title = String(event.title ?? "");
  let description = String(event.description ?? "");
  if (event.origin === "chuck_mail" && title.startsWith("Mail from ") && !description.includes("\n")) {
    const prefix = `${title.slice("Mail from ".length)}: `;
    if (description.startsWith(prefix)) description = description.slice(prefix.length);
  }
  const mail = event.origin === "chuck_mail" ? mailOf(event) : undefined;
  return (
    <div className="notice-row" role="note">
      <Badge variant={warning ? "warning" : "neutral"}>{chuckOrigin ? "Chuck" : warning ? "warning" : "info"}</Badge>
      <span className="notice-title">{title}</span>
      {description ? <span className="notice-description">{description}</span> : null}
      {mail ? <div className="notice-mail"><MailDisclosure mail={mail} /></div> : null}
    </div>
  );
}
