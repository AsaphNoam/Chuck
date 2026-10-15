# Chuck — Implementation handoff

**Live agent state.** Settled state through 2026-10-14 is archived in
[`HANDOFF-through-2026-10-14`](../archive/state/HANDOFF-through-2026-10-14.md).
Follow [`AGENT-WORKFLOW.md`](AGENT-WORKFLOW.md).

## Current position

- **Finished — agent context continuity (2026-10-14):** C1–C9 closed locally. Instructions,
  tool evidence and bounded recent history survive handoffs; the separate bridge survives native
  resume/restart. Native-load failures refuse replacement; unverified provider handoffs refuse
  before teardown. Full serial tests, tagged build and focused race checks pass. Live provider
  receipts remain gates below. Original evidence and closure are in
  [`INVESTIGATION-agent-continuity-2026-10-14`](../archive/state/INVESTIGATION-agent-continuity-2026-10-14.md).
- **Finished — stopped-agent image refusal (2026-10-14):** old snapshots preserve unknown image
  capability instead of advertising refusal; desktop wake reaches fresh runtime validation.
  Explicit false still blocks image Send with a visible reason and preserves the draft. Both read
  projections, desktop/phone composers, fresh acceptance and refusal have regression coverage.
  Full Go matrix passes with `GOFLAGS=-p=1`, all 850 UI tests pass, and `make dist` passes.
  Phone stopped agents retain their existing Resume flow. The originating fix unit is closed.
- **Release:** `v0.16.0` published 2026-10-14 (Latest): release CI run 38593397312 passed and
  the GitHub Release carries the archive, `install.sh` and `manifest.json`; body set from
  [`RELEASE-v0.16.0-notes.md`](../archive/state/RELEASE-v0.16.0-notes.md). Manual provider gates
  below remain owed.
- **Main CI UI test races fixed locally (CI unconfirmed):** `NewAgentModal` tests now wait for an
  enabled Launch and mock `/api/config`; the phone successive-gap test has a 20 s timeout. Full
  UI suite passes locally; remote CI confirmation remains owed. Range `v0.15.1..v0.16.0`: the shipped
  `operating-chuck` Think Tank reference now names the discussion `pass` disposition and the
  quiet-round ending; nothing else agent-facing.
- **Repository:** `AsaphNoam/Chuck`; branch `main`.
- **Known risk (unchanged, uninvestigated):** runtime `SetSessionConfig` holds `as.mu` across its
  provider call while `onNotification` takes `as.mu` for unmatched frames on the synchronous read
  loop, so an adapter notification sent before that reply could stall it.
- **Design prepared/TBD — subscription quota views:** FS-09.R80–R81/A49–A50,
  TS-04.R89/TS-03.R61. Full passive paired snapshots remain TBD for Claude and Codex; evidence is
  in [`ideas.md`](../ideas.md). Not a ready unit. Do not implement a hidden prompt, command
  parser, login side client or new adapter patch.
- **Paused work:** `migrate-internal-actions-from-mcp.md` remains paused on its transport blocker.
- **Source-build Go tests:** run with `CHUCK_RUNTIME_ROOT` unset; an inherited installed bundle
  violates a credential test's source-build assumption. Prefer `GOFLAGS=-p=1` for full closure:
  the parallel image-fix run timed out `TestCodexProberAcceptsNativeLoginWithoutAPIKey` after 2 s;
  five isolated repeats and the full serial matrix passed without code changes.
- **Known flaky checks (uninvestigated):** `internal/server`
  `TestOrdinaryStageAgentStopPausesPipelineRun` intermittently returns 409 “a resume is already in
  progress”. `FileViewer.test.tsx`
  “scrolls to and marks a cited line” failed once in the full suite and passes alone.
  `internal/server` `TestQuotaDispatcherContinuesSameConversationOnce/stopped` failed once
  (“reloaded transcript differs”, FTS5 variant) in the `v0.16.0` closure run; 40 isolated runs,
  30 `-race` runs and a full FTS5 package rerun pass. Range commit `9989fb88` touched ACP update
  routing; whether it relates is uninvestigated.
- **Mail receipt assertion race (2026-10-14):**
  `TestMailActivationStartsIdleAgentWithoutMutatingMailProvenance` waits for a fake-provider
  prompt file, which appears before the asynchronous response and inline-receipt commit.
  Concurrent closure variants failed the immediate receipt assertion; ten isolated repeats pass.
  The activation path is unchanged by the continuity fixes. Waiting for eventual receipt is
  separate test-maintenance work; no product delivery defect was observed.

## Active change

None.

## Acceptance gates still owed

- **FS-01.A7/A8 / TS-02.R48:** live Claude/Codex cross-backend handoff → immediate Stop/wake,
  explicit model → Stop/Resume, restart, completed target turn and native compaction. Regression
  wire capture and fake compaction notice prove Chuck's bridge delivery/lifetime, not effective
  live-provider context adoption. OpenCode/OpenHands handoffs now refuse before teardown pending
  verified model/primer delivery; their existing FS-09.A6 gate remains owed.

- **FS-20.A22–A24 / TS-13.R30:** paired iPhone Home Screen app and Android browser
  background-return journey (stalled stream, outage, revoked pairing). Simulated/browser proof
  passes; the device suspension cause is still unreproduced.

- **FS-03.A67 / TS-04.R92:** live Claude/Codex permission-mode receipt. Claude 0.59.0's mode
  option is source-checked only; the Codex mode shape is unverified.

- **FS-01.A28:** bounded packaged/installed Claude/Codex terminal-quota and owner-activation
  provider receipt. Current Claude SDK 0.3.286's bare numeric reset declaration has no verified
  producer-unit receipt; Codex forwards no typed reset. Both remain unknown and unscheduled.
  Known-reset state/fake-provider/browser proof is not live-provider reset or delivery evidence.
- **FS-14.A50 / FS-21.A31:** bounded packaged Claude/Codex Think Tank stage probe (start → fresh
  participants → judge → accepted output → next stage). Fake-provider E2E and the 318-check render
  pass are not live receipts. **FS-14.A49** restart/race matrix and **FS-21.A32** live Stop-cancel
  race remain planned.
- **FS-21 / TS-06.R33:** credentialed Claude/Codex Think Tank smoke covering room read/submit,
  addressed input, approval/denial, private Send/Steer, native resume, end-only judge, and retained
  exact result. Fake-provider journey and automated/race closure passed; live receipt is owed.
- **FS-10.R25/A14 / TS-02.R41:** supervised disposable-real-home cutover rehearsal with receipt.
- **FS-02.A46 / A27:** real-browser toast and macOS notification clicks; six-tab shared-stream,
  J14 real-browser, and Sky & Grove with Codex capabilities.
- **FS-14.A54:** packaged Claude/Codex probe that a standing owner/dedicated coordinator adopts
  shared orchestrator instructions; fake delivery capture is not a live receipt.
- **FS-18.A12/A13:** six manual role scenarios and role-free follow-ups on pinned Claude/Codex;
  credentialed fresh/resumed Claude native-preset adoption (pinned adapter 0.85.1 only proves
  the `_meta.systemPrompt` request, not live adoption).
- **FS-20.A1/A5/A6/A8:** real tailnet, Android, iPhone, and `pmset -g assertions`; iPhone Home
  Screen PNG touch icon is absent. The combined 390px dashboard/project/agent-management/Files/
  Commands-retired journey remains owed; fast-mode picker and Continue on approval pause remain
  outside the fake-provider coverage.
- **TS-06.R21:** credentialed Claude/Codex login/chat.
- **TS-06.R31 / FS-09.A40/A42/A46/A47/R78 / FS-10.A10–A12:** finite four-combination provider
  smoke (each provider with current Bundle and one current Installed CLI), plus Installed update →
  Refresh → model and missing Installed → Bundle → retry → Installed-with-overrides journeys.
  Claude Installed fresh launch passed 2026-10-05; the broader credentialed matrix is owed.
- **TS-06.R26/R32:** credentialed Codex 2.1.1 receipt for FS-03.A41/A42 and FS-01.A20.
- **TS-06.R32:** two-point Claude/Codex smoke on ACP 0.85.1/2.1.1 and Codex 0.159.3, one notice,
  Steer during a running command, and refused model switch where policy allows. Automated assembly,
  fake-runtime, release-CI assembly, and native-provider proofs passed; live receipts are owed.

## Decisions needing input

- Whether to standardize TS-03.R3–R4 mixed legacy error envelopes or TS-04.R3 provider model-ID
  ownership (compatibility change).
- Whether a pipeline stage paused after failed launch/resume continues withholding **Open agent**,
  matching restart recovery, or permits chat under a wider continuation contract.

## Design consistency notes

- Next presentation review: decide whether crisp asymmetric technical-surface radii are deliberate
  under FS-12.R52.
- When direct-action work resumes, align TS-01.R25/TS-03.R32 with TS-04.R40 and scope FS-17
  section 6's opening sentence to the planned boundary.
- Injected-steer lifetime edge case needs `/investigate-bug` before `/fix`; FilesTab still copies
  silently through bare `writeText`.
- The phone agent header still repeats the model as text under the name now that the model
  switch shows it above the composer.

## Review findings

### Usability review 2026-10-14 — stopped-agent image wake (FS-03.A64) — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Run report: [`usability-review-run-2026-10-14`](../archive/reviews/usability-review-run-2026-10-14.md).

- **Worth fixing** — **J7/A64 stale Retry after a definite image refusal:** `ui/src/components/chat/Composer.tsx`
  ~L140–168 sets the unconfirmed-send `retry` state for every failed send, so a stopped agent whose
  fresh peer refuses an image (422, nothing consumed) shows “Previous message was not confirmed.
  Retry previous send” beside the refusal. Retry repeats the same 422 with no visible change and the
  notice survives reload. It misstates delivery and invites a dead loop (FS-03.A64 “visibly
  refuses”). Test: a definite 4xx refusal keeps the draft and error but offers no unconfirmed-send
  Retry. (INV §8)

The recovered stopped-agent image implementation unit itself remains closed; live
provider/device acceptance gates and unrelated known risks above remain open.

Closed reviews through the `v0.16.0` cut are in the archive above.

## Changelog

- **2026-10-15 — J3/J6/J11 transcript and stopped-state fixes; review closed (INV §8/§13):** the
  transcript stays pinned to the bottom while late rows (diagrams) settle, and only scrolling toward
  older content leaves it (FS-03.R3/A68). A stopped terminal agent opens no terminal socket, says it
  is stopped and offers Resume in the header (FS-07.R24/A10). An agent stopped on error reads
  “Error” on card and header with its detail (never blank) and a restart hint (FS-02.R79/A60,
  FS-03.R89/A69). FS-14.R44 now records R80's Setup → Review supersession. This closes the
  2026-10-15 review unit. Not covered: the phone agent screen still reads “Stopped” for an errored
  agent; Think Tanks, phone remote control, project resources and config-source binding still
  have no usability journey.

- **2026-10-15 — J9/J11 save validation fixes (INV §2/§3/§8):** a malformed pipeline template id now
  reaches the shared validate-before-write path and returns 422 with Validate's diagnostics (new
  FS-14.R90/A56); update with a malformed id is 404. Catalog save refuses a blank backend display
  name through the create path's shared check, and the editor keeps the draft (new FS-09.R82/A51).
  A catalog already holding a blank name still loads. Regressions failed before the fix; focused
  Go (both variants) and UI tests pass. Full closure owed at run end.

- **2026-10-15 — J2/J8/J13/J16 UI fixes (INV §7/§8):** the onboarding Project step holds on a
  `cwd_not_found` warning with Continue anyway; OpenCode/OpenHands guidance names the `opencode`/
  `openhands` command that must be on PATH (the repo documents no install URL — supply one in
  FS-09.R27 to link it); an inactive tray source defaults to an available target; launch, resume
  and switch toast a recreated worktree checkout; the archive page distinguishes unknown id (no
  Resume/Restore), empty transcript and failed read with Retry (new FS-05.R42/A25). Each
  regression failed before its fix; focused UI tests pass. Full closure owed at run end.

- **2026-10-15 — J5/J12 card order fix (INV §1/§3):** the agent stream no longer appends to or
  prunes the stored manual order, the grid saves only a layout that differs from what it read or
  last saved, and unplaced agents sort by creation time then id (new FS-02.R78/A59). The new grid
  regression failed before the fix; focused grid/store/SSE tests and the typecheck pass. A loaded
  layout naming an unknown pane id still saves once (A32). Full closure is owed at run end.

- **2026-10-15 — J10/J13 mail-turn visibility (INV §8):** decided under new FS-06.R38: any turn that
  receives mail in its prompt records one notice row naming each sender and subject (no body), so
  peer-mail and tray-to-agent turns show their cause. The extended activation test fails without
  the notice; runtime package tests pass. Full closure is owed at the end of this fix run.

- **2026-10-15 — J15/J16 task view fixes (INV §1):** deleting a task now publishes a `task_update`
  removal so every open view, including the dashboard attention count, refetches. A finished task
  shows unmet prerequisites and unfired signals as history, without a live wait or signal control.
  Both regressions failed before the fix; focused Go and Tasks UI tests pass. Full closure is owed
  at the end of this review's fix run.

- **2026-10-15 — J13 new-task annotation delivery fix (INV §1/§8/§10/§11/§14/§15/§16/§17):** the tray now runs
  the server's annotation and mail-size validators before opening launch. Once launched, a failed
  delivery names that agent and retries to it instead of launching another; changing source agents
  resets the retry state. The bounded preflight has no transcript or mail side effects and is
  available through the same paired-phone route policy as annotation delivery. Full serial Go
  tests in both variants, all 856 UI tests, Go/UI builds, spec and diff checks pass. The October 15
  review remains open for its other findings.

- **2026-10-15 — J14 pipeline restart fix (INV §1/§2/§17):** restart task recovery interrupted a
  stage task without projecting that transition onto its run. It now uses the existing stage
  interruption seam with a restart reason. The regression failed before the fix; afterward the
  run pauses, offers an effective Retry, and the rendered page withholds Open agent and names the
  restart cause. Full serial Go tests in both variants, all 854 UI tests, Go/UI builds, spec and
  diff checks pass. The October 15 review remains open for its other findings.

- **2026-10-15 — J10 mail state fix (INV §1/§11/§17):** the send callback that refreshes recipient
  unread mail and sender Sent pulse ran only for waking messages. It now runs after every successful
  insert and passes the wake choice so deferred mail does not queue activation. A real MCP-to-bus
  test failed before the fix for deferred sends and passes for both waking and deferred sends; the
  existing read/clear check also passes. Full serial Go tests in both variants, Go build, spec and
  diff checks pass. The October 15 review remains open for its other findings.

- **2026-10-15 — J3 live sent-message fix (INV §11/§17):** a prompt response can arrive after the
  durable user event, reply, and turn divider. The composer now checks for that delivered prompt
  before adding a local echo; repeated identical sends remain distinct. The old behavior failed
  the new late-response regression. Text and attachment send regressions pass. Full Go matrix
  (serial, including `sqlite_fts5`), all 853 UI tests, Go/UI builds, and spec checks pass. The
  2026-10-15 review stays open for its other findings, including the separate J3 reload-scroll issue.

- **2026-10-14 — usability review (J1, FS-03.A64 desktop wake):** real tagged binary, fresh home,
  fake peer as `claude-agent-acp` on PATH, Chromium. J1 and the accept/explicit-refusal variants
  pass; one Worth-fixing stale Retry after refusal recorded above; phone wake not exercised.

- **2026-10-14 — stopped-agent image fix (INV §1/§2/§8/§11/§17):** confirmed the recovered
  regression failed before editing; preserved unknown image advertisements across persistence
  and shared read projections, retained explicit refusal and wake-time validation, and surfaced
  the desktop disabled-Send reason. Added serialization, fresh-wake acceptance/refusal and both
  composer regressions. `env -u CHUCK_RUNTIME_ROOT GOFLAGS=-p=1 make test`, all 850 UI tests,
  `make dist` and diff checks pass. No findings remain in this unit; existing manual gates remain.
- **2026-10-14 — context continuity fix (INV §1/§2/§3/§4/§5/§7/§8/§11/§12/§16/§17):** closed
  C1–C9 with canonical bounded primers, durable conversation bridges, explicit model provenance,
  atomic partial-switch composition, fail-closed restore and honest provider capability/refusal.
  `make test` passes both variants serially; tagged build and focused lifecycle race checks pass.
  Live provider verification and the mail-test assertion race remain.

## Blocked on human

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

See [`RELEASE-v0.16.0-notes.md`](../archive/state/RELEASE-v0.16.0-notes.md),
[`RELEASE-v0.15.1-notes.md`](../archive/state/RELEASE-v0.15.1-notes.md) (supersedes the
never-published v0.14.0 and v0.15.0 notes) and
[`RELEASE-v0.13.0-notes.md`](../archive/state/RELEASE-v0.13.0-notes.md).
