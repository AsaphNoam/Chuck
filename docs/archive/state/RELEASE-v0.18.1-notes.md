# Chuck v0.18.1

Status: preparation; version and publication approval pending.
Previous release: v0.18.0. This is a fixes-only patch release.

## What improves

- Conversation repair preserves complete saved data when a transcript is missing, damaged or
  interrupted during repair, including the native conversation identity and session settings.
- Task waiting, resumed assignments and pipeline controllers retain the context and ownership
  needed to continue after cleanup or restart. Quota recovery checks eligibility before spending
  a check, compensates failed room starts and progresses fairly through cleanup.
- Lost or uncertain input responses preserve explicit recovery instead of silently replaying
  messages; annotation retries and held-message withdrawal keep their exact identity.
- Launch and Resume wait for owned-checkout recreation/setup. Resource repair preserves
  restrictive owner permissions. CLI selection and partial-failure reporting remain truthful.
- Stream overflow recovers with a fresh snapshot. Transcript scans, retained runtime state and
  task-result outputs have tighter bounds.
- Offline reindex refuses to proceed when the dashboard PID file cannot be read or decoded.

The shipped `operating-chuck` coordination reference already includes the resumed-assignment
`observed_changes` guidance. The release-range review found no further agent-facing package
correction necessary. README now identifies reindex as offline repair. No scheduled-task feature
is included; the concurrent planning edits remain outside the candidate.

## Verification and limits

The clean candidate's distribution build and all 995 UI tests pass; generated UI files match
the committed copy. `make vet` passes. Full Go closure and architecture-receipt closure remain
in progress. The first no-tag Go run hit the already-known bootstrap download-start timing
failure; focused repeats passed twice and failed once. No installer/test change or waiver is
claimed. Final closure must resolve the failed check before tagging.
The same run also failed the restart-recovery fixture's immediate-release assertion while its
reporting turn was still live; TS-10.R19/R48 require deferred cleanup in that condition. The
fixture does not actually remove the runtime registry as a restart would. No product regression
or fixture repair is claimed; its full-suite retry is still pending.
The existing architecture review is reused with exact verified fix/affected-coverage entries;
no second architectural review is required for its finding repair.

Credentialed Claude/Codex adoption, physical-phone behavior, real terminal-driver ownership and
other explicit acceptance receipts remain owed in the live handoff. Automated/fake-provider
checks do not establish those behaviors. Private runtime archive assembly and fresh-install
verification belong to release CI after an authorized push.

The review and release are scoped to committed source. Publication must explicitly authorize
pushing all currently unpushed main commits and the new version tag.
