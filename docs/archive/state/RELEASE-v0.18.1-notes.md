# Chuck v0.18.1

Status: [published](https://github.com/AsaphNoam/Chuck/releases/tag/v0.18.1) on 2026-10-23,
annotated tag at `dc9790575fa7c1f043b86318d1431db459104ad0` after explicit operator approval
of the version, all 23 unpushed commits and tag.
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

Both complete Go variants now pass at clean source `001ba8b5`. The distribution build and all
995 UI tests pass on identical product source; generated UI matches the committed copy.
`make vet`, spec/link/diff checks and the focused runtime-guard race test pass.

Earlier no-tag runs failed the existing bootstrap download-start deadline; the initial focused
repeats passed twice and failed once. An early run also failed immediate release while its
reporting turn was still live; TS-10.R19/R48 require deferred cleanup in that condition. The
fixture does not actually remove the runtime registry as a restart would. That server suite
passed on the full no-tag retry and in the FTS5 run. These failures are preserved as evidence,
not waived or attributed to the PID guard. The operator approved both test-only corrections
under the separate-fix rule on 2026-10-23. They now pass three focused CLI repeats, five recovery
repeats, independent assertion review and both full suites. The recovery test proves settled cleanup and retained
finished state, not immutable result/summary preservation by itself.

Source/checkpoint commits: guard repair `69082b39`; preparation `0e533e0c`. The existing
architecture receipt records exact repair/affected coverage and verified ARCH-1 closure.
REL-VERIFY-1 is also closed; no product behavior changed in its test correction. Earlier logs:
`/private/tmp/chuck-v0.18.1-{go-closure,go-retry,fts-closure,bootstrap-diagnostic,ui-closure,dist,vet}.log`.
Final logs: `/private/tmp/chuck-v0.18.1-approved-{go-full,fts-full,static,cli-focus,recovery-focus}.log`.
The existing architecture review is reused with exact verified fix/affected-coverage entries;
no second architectural review is required for its finding repair.

Credentialed Claude/Codex adoption, physical-phone behavior, real terminal-driver ownership and
other explicit acceptance receipts remain owed in the live handoff. Automated/fake-provider
checks do not establish those behaviors.

## Publication receipts

[Release CI 39959315460](https://github.com/AsaphNoam/Chuck/actions/runs/39959315460) passed
architecture closure, private runtime assembly, release/bootstrap tests, FTS5 archive verification
and fresh installation preserving the existing home. Archive, installer and manifest are attached.
The downloaded `chuck-0.18.1-darwin-arm64.tar.gz` matches manifest size `325053538` and SHA256
`7724309786110ae3de645f61562f1ab45996f36a5f789c0977b3b00f11fb33ee`.

[Main CI 39959316204](https://github.com/AsaphNoam/Chuck/actions/runs/39959316204) failed on
both original and unchanged rerun attempts at `TestCancelBeforePermissionPublicationResolvesAfterRequest`:
observed `[permission_request]`, expected request then resolution. Local full-suite success does
not waive this failure. REL-CI-1 remains an open Must-fix in the live handoff; fixture versus
runtime cause is undiagnosed. No further code changes or cancellation repair are claimed.

Concurrent uncommitted scheduling design remains excluded from the published source.
