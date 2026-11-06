# Chuck v0.22.0

## Changes

- Edit writable text files directly in desktop conversations, archives, rooms and Focus file.
  Save updates the original file explicitly; browser-local drafts preserve unsaved edits for
  recovery. Conflicts keep your draft for review. Phone file views remain read-only.
- File readers check for changes while keeping the current snapshot readable during refresh.
  A delayed refresh cannot replace a newer saved snapshot.
- Preview self-contained HTML files in the file reader, or open a local file in your browser
  from its context menu.
- Send annotations to the current agent like ordinary chat messages, with delivery feedback
  and recovery when input is interrupted.
- Task creation uses a simpler form and exposes the fields needed to correct server refusals.
  Pipeline stage-work cards link to their agents, and automatic pane opening waits for layout.
- Think Tank setup fits its participant controls more reliably. Routine pipeline coordination
  uses task results and deferred mail to reduce unnecessary interruptions.
- Fixes preserve withdrawn drafts, clear resolved Steer recovery offers, serialize file saves
  across path aliases, and protect newer browser drafts from stale pages.

## Verification and limits

Local release verification and independent architecture signoff are recorded in the handoff
and `docs/archive/reviews/architecture/v0.21.0-next.json`. Hosted archive and installation checks
run on publication. Credentialed provider, physical-device and remaining native/browser journeys
stay owed as listed in the handoff.

## Installation

This release supports Apple-silicon macOS. The archive includes Chuck's private Node and adapter
runtime. Provider selection defaults to Installed; Bundle is an explicit Settings choice.
The release is unsigned and not notarized, so macOS may require unidentified-developer approval.
Updates are explicit and preserve Chuck user data; restart the dashboard to use the new runtime.
