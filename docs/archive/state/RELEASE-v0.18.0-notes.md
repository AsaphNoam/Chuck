# Chuck v0.18.0

## Changes

- Switch between a project's running agents from the breadcrumb, and move between Think Tank
  members and the room with compact chat navigation.
- Preview images, format JSON and copy paths in the file reader.
- Phone navigation starts with collapsed groups, offers Jump to latest as messages arrive, and
  uses clearer Send and runtime controls.
- Choose the default permission policy in Settings; newly created tasks can place their launched
  agents into a project group.
- Conversation events share a durable journal, ordered control delivery and visible history
  recovery boundaries. Native model selection and continuation preserve the effective session
  configuration. Process and terminal control require proof of ownership.
- Fix live effort/fast setting stalls and cancellation being mistaken for successful quota recovery.
- Keep busy/idle status owned by the active turn, preserve open permission gates, and settle
  permission requests in publication order.

## Release scope and exceptions

This release includes committed main through `ad08b386`, after v0.17.0. The concurrent runtime
fix was committed during release preparation and is included. Uncommitted ready-change edits in
the operator checkout are excluded and preserved.

On 2026-10-20 the operator explicitly authorized release and push despite pending fix items and
approved skipping the architecture review for this cut. No architecture review was performed or
claimed. The CI exception matches only v0.18.0; later releases retain the normal gate.

Open review findings, the waiting workflow review and core reliability audits, and the manual
provider/device/terminal acceptance gates remain in the live handoff. Automated success does not
close them. In particular, live Claude/Codex adoption and login/chat, real tmux/iTerm2 lifecycle,
and paired-phone receipts remain unverified.

The shipped operating-chuck package already includes this range's task-group guidance. Its
existing lifecycle/configuration guidance remains applicable; no additional package edit or
component version bump is required.

## Verification and publication

- Distribution build and binary metadata inspection pass: version 0.18.0, darwin/arm64,
  `sqlite_fts5` enabled. UI tests pass (99 files, 980 tests).
- Specification checks, release-workflow YAML parsing, whitespace checks and all eight
  architectural-gate regression tests pass. The candidate gate itself refuses the absent receipt;
  publication uses the explicit version-specific waiver above.
- Both Go test variants and static checks pass. The operator confirmed v0.18.0.
  Tagging and GitHub publication remain pending.
- A concurrent FTS5 closure run hit the previously recorded quota-dispatch transcript flake;
  the full release run passed, and five focused FTS5 repeats passed. The failure remains recorded
  in the live handoff; no test repair is claimed by this release.
