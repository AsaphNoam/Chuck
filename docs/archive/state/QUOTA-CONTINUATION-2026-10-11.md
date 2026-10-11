# Quota continuation — verification 2026-10-11

Substantive range: `9409787` through `8011691`. Governing requirements are FS-01.R40–R45/A23–A27,
FS-04.R53/A33, TS-02.R45, TS-03.R60, TS-04.R86–R88 and TS-10.R39–R44, with the named task,
pipeline and Think Tank owner exceptions. FS-01.A28 remains a separate live-provider gate.

## Delivered behavior

AIR terminal error/limit failures without retry/new-session actions establish a quota interruption
only for the exact root executing turn. One durable bounded record preserves its source and current
owner. Ordinary/task/room continuation uses existing activation, lifecycle and owner authority;
failed room submissions remain failed history, with no publication or allowance charge. Source
quota blocks automatic successors and preserves unread inline mail.

The default-on live policy and explicit cancellation targets reach desktop and authorized phone
clients. Manual controls cancel only the captured pending episode after the owner action succeeds;
refused actions preserve recovery. Startup fences uncertain provider delivery. Notice and success
receipts repair the exact continuation without replaying a provider turn. The transcript attributes
the exact continuation sentence to Chuck once.

A stopped borrowed task first reserves a woke runtime slot under the ordinary task budget. Full
capacity defers without a prompt, assignment change or task start-attempt charge; existing created/
woke reservations keep their slots. Normal Stop/group release still refuse lifecycle contention.

## Automated closure

- `GOCACHE=/tmp/chuck-go-cache make test`: spec lint and both Go variants pass after final product
  edits. Final server suites pass in 128.887s untagged and 129.285s with `sqlite_fts5`.
- `go vet ./...`: passes.
- Focused `-race` runtime/state/server quota, session-failure and Stop/switch/wake/group paths pass.
- UI: 86 files, 782 tests and 41 style/presentation contract checks pass.
- `make dist`: UI build and generated embed pass; final `make build` produces the FTS5-tagged
  binary including the capacity refinement. Embedded output was generated, never hand-edited.
- New fixtures prove concurrent shared-backend failures retain independent records and list/detail/
  SSE projections; known-reset live/stopped same-conversation dispatch; task ownership across
  quota/recovery; room opening/discussion/closing/judge authority; manual/global cancellation;
  notice repair and exact-tuple blocking; runtime budget deferral with zero provider effects.

Initial closure found a quota-lock regression that delayed Stop/group conflict responses; immediate
conflict behavior was restored and the existing tests pass. A provider-version probe timed out under
simultaneous full variants; its isolated recheck and final sequential closure pass. No tests were
removed, skipped or weakened to obtain closure.

## Rendered browser proof

41 Chromium checks pass, with screenshots visually inspected:

| Journey | Checks | Evidence |
| --- | ---: | --- |
| 1024px desktop and actual 390px phone layout in Core, Sky & Grove and Studio | 23 | `/tmp/chuck-quota-render-final/` |
| Embedded desktop/authenticated phone: explicit turn cancellation, typed unknown reset, actual process Stop with retained phone transcript, saved-off policy | 8 | `/tmp/chuck-quota-journey-stopped/` |
| Embedded desktop reload, authenticated phone reload and desktop Archive: exact-one notice, distinct Chuck badge, retained sequence and origin | 10 | `/tmp/chuck-quota-notice-journey-final/` |

Reproduction scripts are `ui/scripts/quota-render.mjs`, `quota-journey.mjs` and
`quota-notice-journey.mjs`. Embedded journeys use the isolated `TestGroupBrowserFixture`; the
known-reset mode is explicitly `CHUCK_QUOTA_BROWSER_KNOWN=1`. Layout fixtures exclude service-worker
coverage. Phone archival preserves FS-20.R39's existing “Archived on the Mac” screen; transcript
parity is checked before archive, while desktop Archive retains the transcript.

## Provider boundary

Claude ACP 0.85.1 and Codex ACP 2.1.1 terminal wire shapes are fixture-tested. Claude SDK 0.3.286's
numeric reset declaration does not verify its epoch unit, and Codex forwards no typed reset.
Neither supported path currently supplies an applicable timestamp with verified units. Actual
provider interruptions therefore show **Reset time unknown** and remain unscheduled. No text date
parser, magnitude inference, provider-login client, hidden usage prompt or adapter patch was added.

Known-reset dispatch and notice proof uses state-owned fixture evidence. It does not satisfy
FS-01.A28's packaged/installed credentialed provider reset, provenance or delivery acceptance.
