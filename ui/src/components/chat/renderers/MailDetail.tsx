import { useState } from "react";
import type { MailRecord, TranscriptEvent } from "../../../api/types";
import { SanitizedMarkdown } from "./SanitizedMarkdown";

// mailOf reads the optional mail detail a Chuck mail notice carries; the same
// record expands in a received notice row and in a send/read tool line
// (FS-06.R45, TS-08.R161).
export function mailOf(event: TranscriptEvent): MailRecord | undefined {
  const mail = event.mail;
  if (mail == null || typeof mail !== "object") return undefined;
  const record = mail as Partial<MailRecord>;
  if (record.direction !== "received" && record.direction !== "sent" && record.direction !== "read") return undefined;
  return { direction: record.direction, messages: record.messages ?? [], error: record.error, truncated: Boolean(record.truncated) };
}

export function MailDisclosure({ mail }: { mail: MailRecord }) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button type="button" className="tool-toggle" aria-expanded={open} onClick={() => setOpen((value) => !value)}>
        {open ? "▾" : "▸"} Mail
      </button>
      {open && <MailDetail mail={mail} />}
    </>
  );
}

export function MailDetail({ mail }: { mail: MailRecord }) {
  const sent = mail.direction === "sent";
  return (
    <div className="mail-detail" data-direction={mail.direction}>
      {mail.error && <p className="mail-error">{mail.error.code}: {mail.error.message}</p>}
      {mail.messages.length === 0 && mail.direction === "read" && !mail.error && <p className="mail-note">No messages returned.</p>}
      {mail.messages.map((message, index) => {
        const name = sent ? message.to_name : message.from_name;
        const address = sent ? message.to_address : message.from_address;
        const party = name || address || "";
        return (
          <article className="mail-message" key={message.message_id ?? index}>
            <p className="mail-meta">
              {party && <span className="mail-party">{sent ? "To" : "From"} {party}</span>}
              <span className="mail-subject">{message.subject}</span>
              <span className="mail-wake">{message.wake ? "waking" : "deferred"}</span>
              {message.created_at && <time dateTime={message.created_at}>{new Date(message.created_at).toLocaleString()}</time>}
            </p>
            {message.body && <div className="mail-body"><SanitizedMarkdown text={message.body} /></div>}
          </article>
        );
      })}
      {mail.truncated && <p className="mail-note">Some message bodies were omitted to fit the record.</p>}
    </div>
  );
}
