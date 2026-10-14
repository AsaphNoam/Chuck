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
- **Recovered investigation — stopped-agent image refusal (2026-10-14):** the finding and skipped
  reproduction existed only on `claude/awesome-gates-nxg67z`, outside the shared main fix queue and
  v0.16.0 cut. Recovered locally under **Review findings**; publication still requires authorization.
- **Release:** `v0.16.0` published 2026-10-14 (Latest): release CI run 38593397312 passed and
  the GitHub Release carries the archive, `install.sh` and `manifest.json`; body set from
  [`RELEASE-v0.16.0-notes.md`](../archive/state/RELEASE-v0.16.0-notes.md). Manual provider gates
  below remain owed.
- **Main CI UI test races fixed locally (CI unconfirmed):** `NewAgentModal` tests now wait for an
  enabled Launch and mock `/api/config`; the phone successive-gap test has a 20 s timeout. Full
  UI suite passes locally; the next `main` push should confirm CI green. Range `v0.15.1..v0.16.0`: the shipped
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
  violates a credential test's source-build assumption.
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


### Recovered investigation — stopped agent refuses an image (2026-10-14) — **Fix model:** medium — Codex Terra or Claude Opus.

**Report (verbatim):** “Trying to send a message with an attached image when an agent is stopped
gives an error that this provider doesn't support images.” No version, provider or logs supplied;
the refusal is client-side, so nothing would be logged. Original investigation read current main
at v0.15.1. Recovered from commit `1d4db58bd685e1c458d35bf1a8fe9ae1d0c4e53b` on remote branch
`claude/awesome-gates-nxg67z`. Its finding and skipped reproduction were absent from main at
`c80dd5fd`; this is an independent missing investigation, not another copy of the model/provider
continuity unit. No product behavior is changed by this recovery.

- **Must fix** — **Confirmed code defect and specification coverage gap; fix complexity medium.**
  Stopped agents whose saved capability snapshot predates image negotiation project `image_input`
  as false. `image_input` joined the snapshot in `395c1a8` (2026-10-13);
  `state.DecodeRuntimeCapabilities` decodes an absent key as false, and both
  `AttachmentCapabilitiesFor` projections (`internal/state/manager.go` and
  `internal/server/launch.go`) publish that value. The desktop Composer and phone AgentScreen
  reject attached images before Send can reach the otherwise-correct stopped-agent wake path.
  Normal-use trigger: upgrade and attach an image to an agent not run since; manual Resume clears
  the stale capability. FS-03.R83 and TS-04.R90 require wake-time revalidation. TS-01.R35/TS-03.R63
  need a specified unknown-advertisement state rather than making old rows look like a genuine
  provider refusal. INV §1/§11/§17. Suggested fix: preserve unknown versus a provider-advertised
  false, let unknown capability reach server wake-time revalidation, and retain explicit refusal
  for real false. Un-skip `internal/server/attachment_wake_repro_test.go`
  `TestStoppedAgentWithPreImageSnapshotDoesNotRefuseImages`; add desktop/phone coverage for unknown
  capability and genuine refusal. The recovered reproduction fails on the false projection while
  its direct image wake succeeds with 202. Original author recorded it as reproduced; recovery
  independently re-ran the unskipped test against current main through a temporary overlay and
  observed the same projection failure with no wake-202 failure. The retained test stays skipped;
  focused checks pass untagged and with `sqlite_fts5`, and spec/diff checks pass.

**Cloud-work audit scope (2026-10-14):** Checked this cloud checkout's worktree, branches, reflog and
unreachable commits; all five non-main remote branches; and all ten GitHub pull requests (all
merged; none open or closed-unmerged). Besides the archived continuity and image investigations, no additional
undelivered work was found in this scope. The August review/fix branches and July Learning Atlas
branch have patch-equivalent changes on main. The July Skip setup branch's behavior is present as
Set up later, with current spec and tests. Other chats' private/unpushed cloud workspaces and
account-wide conversation history are not accessible to this session; those are not certified clear.
Recovery is local-only until authorized publication. The continuity unit is now closed locally.
**Next:** `/fix` this stopped-agent image unit independently.

Closed reviews through the `v0.16.0` cut are in the archive above.

## Changelog

- **2026-10-14 — context continuity fix (INV §1/§2/§3/§4/§5/§7/§8/§11/§12/§16/§17):** closed
  C1–C9 with canonical bounded primers, durable conversation bridges, explicit model provenance,
  atomic partial-switch composition, fail-closed restore and honest provider capability/refusal.
  `make test` passes both variants serially; tagged build and focused lifecycle race checks pass.
  Live provider verification, the stopped-agent image unit and the mail-test assertion race remain.

## Blocked on human

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

See [`RELEASE-v0.16.0-notes.md`](../archive/state/RELEASE-v0.16.0-notes.md),
[`RELEASE-v0.15.1-notes.md`](../archive/state/RELEASE-v0.15.1-notes.md) (supersedes the
never-published v0.14.0 and v0.15.0 notes) and
[`RELEASE-v0.13.0-notes.md`](../archive/state/RELEASE-v0.13.0-notes.md).
