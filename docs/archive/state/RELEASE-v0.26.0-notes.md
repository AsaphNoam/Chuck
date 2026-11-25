# Chuck v0.26.0

- Start a first chat with Claude or Codex straight from setup. Chuck reports a provider as ready
  only after its native sign-in check passes, keeps repair drafts, and opens the dashboard
  directly after an interactive install.
- Sent annotations link back to their source in the chat, including quotes across paragraphs and
  folded diff lines.
- One set of type roles across desktop and phone interfaces.
- On a phone, pull up at the newest message to refresh the chat.
- Tasks explain when they are waiting for a free runtime slot.
- Fixes: Stop and Archive settle quota continuation when the client leaves early; Dismiss works
  on a quota banner once its continuation is cancelled; the Files type picker stays open; the
  queued annotation tail settles after delivery.
- The AgentDeck compatibility shims and cutover guide are retired. The README is rewritten for
  newcomers.
