# Expandable agent mail in transcript activity

**State:** Waiting to start
**Why:** The operator asked for this on 2026-11-02: show in an agent's activity whom it mailed and
whom it received mail from, with the message content collapsed by default. Today the FS-06.R38
notice shows senders and subjects only, and the mail tool lines show nothing beyond their
one-line label.
**Relevant requirements:** FS-06.R45, FS-06.A33, FS-03.R108 (exception), TS-04.R108,
TS-08.R161, INV §2/§10

## Outcome
The person supervising an agent can expand the existing "Mail from …" notice and the
`send_message`/`check_messages` lines in the tool area. Expanded, they show each message's
sender or recipient, subject, waking or deferred, time and body as Markdown. Collapsed views look
as they do today.

## Included work
- **Received mail:** a copy of the bodies supplied inline is kept with the received notice.
- **Sent and read mail:** Chuck adds a display-only record of each send or read it handles to the
  caller's transcript. Provider tool events are not a complete source: Codex saves null results
  and Claude saves empty call arguments (see TS-04.R108). The UI attaches the record to the
  matching tool line; a record it can't match gets its own mail line.
- **Surfaces:** desktop, archived, phone and Think Tank transcripts, all through the shared
  components.
- **Not included:** any change to agent prompts, peer transcript reads, search, unread counts,
  notifications, wakes or mailbox retention; backfilling existing transcripts; terminal-agent
  transcripts.

## How we will know it works
FS-06.A33: runtime and transcript tests for the saved copies and the unchanged agent-facing
projections; UI grouping and matching tests, including parallel same-tool calls and unmatched
records; component tests for each direction; and a paired phone check as a manual gate.

## Waiting on
Nothing.
