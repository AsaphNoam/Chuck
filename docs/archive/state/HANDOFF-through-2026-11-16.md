# Chuck — settled handoff state through v0.24.0 preparation

- **Match sent prompts by request ID:** reviewed 2026-11-15 at `fb245d40` (FS-03.R118/A93,
  TS-08.R163). PROMPT-1–4 repaired on main at `de246a42`.
- [CR-A8](../../features/CORE-RELIABILITY-REVIEW-A8.md), pipeline pause, answer and recovery:
  findings closed, including PIPE-CHAT-1 and PIPE-CHAT-3.
- [CR-A9](../../features/CORE-RELIABILITY-REVIEW-A9.md), Think Tank lifecycle and quota:
  eight findings closed at `0da7c581`. CR-A9-1 and -5 shared quota hold release; CR-A9-2
  and -3 shared the rule for which holds a room could carry in `setup` and `ended`.

[v0.24.0 notes](RELEASE-v0.24.0-notes.md) record the version-specific waiver. EXEC-6c and
the combined architectural review remain open in the [live handoff](../../features/HANDOFF.md).

## Previous release

- [v0.23.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.23.0) published from annotated
  tag at `f3f34986` after workflow `42450625734` passed architecture, assembly,
  transaction/bootstrap and fresh installation. GitHub's archive metadata matched its manifest.
  The v0.22.0 baseline tag was pushed with explicit approval and its workflow `42450809571`
  was cancelled. See [v0.23.0 notes](RELEASE-v0.23.0-notes.md).
- The v0.23.0 [architecture receipt](../reviews/architecture/v0.22.0-next.json) covered
  release preparation and closed the combined source audit and quota findings.
- Main CI retry `42450626202` passed both Go variants, specs, gate tests and vet. One of 1,181
  UI tests failed while opening the waiting-input pane. The fixture was repaired locally.
