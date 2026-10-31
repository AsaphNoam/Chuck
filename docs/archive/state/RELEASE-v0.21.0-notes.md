# Chuck v0.21.0

## Changes

- Pipelines can be goal-led. The run's owner chooses the next step (including repeating a step,
  or going back from Fix to Review), finishes only with evidence for every required check, and
  asks one concrete question when it needs you. Interrupted, unstartable or stalled owners are
  resumed or replaced automatically within a bounded allowance; when that runs out the run pauses
  recoverably and you can Resume or send instructions. Messages to a waiting owner are queued and
  delivered with its next turn.
- A pipeline run can start in its own new Git worktree and branch. Every agent the run creates
  works in that checkout; when the run ends, Chuck offers to remove it, and keeps it by default.
- The pipelines workspace has a refreshed layout across appearances.
- New **Boring Night** dark skin.
- Tool activity shows as one-line summaries. Running background commands appear beside desktop
  composers and as a count on agent cards, with Stop.
- Launching into a group prefills its defaults, and the composer autocompletes skill triggers.
  File and command suggestions no longer wait on each other.
- Annotations can be sent without recording a source transcript entry, on desktop and phone.
- Phone Home is simpler, the Sky & Grove wallpaper is calmer, and Cancel now sits in the composer.
- Quota interruptions are recognized from more provider error forms, and reset times are read
  from more wordings, so automatic continuation applies more often.
- Claude models offer the extra-high reasoning effort. Directory Browse opens faster. Permission
  saves that did not apply now say why. Terminal provider failures are shown instead of failing
  silently, and Codex no longer spends a background turn on titles.
- Fixes: compact Think Tank navigation, full-width Mermaid diagrams in the file viewer, lower
  memory retention for long conversations, and several pipeline, input-replay and agent-tool
  session repairs from the release architecture review.

## Verification and limits

Release verification and the independent architectural review are recorded in the live handoff
and `docs/archive/reviews/architecture/v0.20.0-next.json`.

Credentialed Claude/Codex receipts for goal-led recovery, run worktrees and quota continuation,
native phone/device journeys and the remaining rendered checks stay owed as listed in the handoff.

## Installation

This release supports Apple-silicon macOS. The archive includes Chuck's private Node and adapter
runtime. Provider selection defaults to Installed; Bundle is an explicit Settings choice.
The release is unsigned and not notarized, so macOS may require unidentified-developer approval.
Updates are explicit and preserve Chuck user data; restart the dashboard to use the new runtime.
