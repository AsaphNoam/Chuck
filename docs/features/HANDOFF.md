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

### Usability review 2026-10-15 — top-to-bottom J1–J17

Run report: [`usability-review-run-2026-10-15`](../archive/reviews/usability-review-run-2026-10-15.md)
(fixtures, repro steps, evidence). Fake-ACP, real tagged/untagged binaries, Playwright Chromium.
**Fix model:** medium — Codex Terra or Claude Opus.

- **Must fix** — **J3 duplicate sent message in live chat:** every send from an agent page or dashboard
  pane renders a second `article.user-message` after the reply/turn divider; the server transcript
  holds one `user_text`, a reload shows one, and SSE reconnect does not heal it. Live and reloaded
  transcripts disagree on the most common action (FS-03 transcript). Test: send → reply → exactly
  one user bubble in the live store.
- **Must fix** — **J10 Mail N / Sent never update live:** an MCP `send_message` (waking or deferred)
  publishes no `state_update` for recipient or sender; `GET /api/sessions` shows `unread_messages`
  null while `/messages` reports unread; badges appear only after a server restart. The 2026-09-26
  run saw them live — likely regression (FS-06.R13/R14, A9). Test: send → recipient state_update
  carries `unread_messages`, sender carries `last_sent_at`.
- **Must fix** — **J14 restart leaves a pipeline run stuck:** after a dashboard restart the stage task
  is `interrupted` but the run stays `running` with empty `attention_reason`; Continue/Retry/Replace
  all ineligible and Open agent still shown, so only Stop remains (FS-14.R14/R48/R66). Test: restart
  with a stage awaiting its result → paused run, Retry eligible, Open agent withheld, restart cause
  stated.
- **Must fix** — **J13 tray “New task” orphans agents:** `AnnotationTray` launches the agent
  (`POST /api/sessions` 201) before the batch is validated/delivered; an empty instruction (422) or
  a failed delivery leaves an agent with no annotations, the tray shows only the validation text or
  “Failed to fetch”, and “Continue to launch” launches another (FS-13.R9/R11; static lead
  `AnnotationTray.tsx` ~L62; no invariant class). Test: invalid batch → no launch; delivery failure → “agent launched,
  delivery failed” and retry delivers to the same agent.
- **Worth fixing** — **J5/J12 card order unstable:** agents with no stored layout position reorder across
  server restarts (and J5/J7 saw it across loads), and every page load issues an unprompted
  `PUT /api/layout`, once re-adding a pruned archived id (FS-02.R14). Test: deterministic fallback
  order; no write on load.
- **Worth fixing** — **J10/J13 mail-triggered turns have no visible cause:** waking peer mail or tray
  delivery to another agent produces an assistant turn with no event in the recipient transcript
  naming the sender or content. No requirement covers recipient visibility (FS-06.R31–R32, FS-13.R7);
  the fix should decide behavior or specification.
- **Worth fixing** — **J3 reload lands mid-transcript:** hard-reloading an agent with diagram turns stops
  above the latest reply with “Jump to latest” shown (diagram height settles after initial scroll).
- **Worth fixing** — **J6 stopped terminal page:** a stopped terminal agent’s Terminal tab shows
  “[connection error] [closed]” (WebSocket 404) with no stopped state or Resume (FS-07).
- **Worth fixing** — **J11 crash state unclear:** a crashed or killed agent reads “Stopped” on header and
  card while the API says `error`; the card carries no error text and nothing says a new message
  restarts it.
- **Worth fixing** — **J8 missing archived session renders blank:** `/archive/<unknown>` or an archived
  agent without its transcript shows a normal header, empty body and active Resume/Restore; the 404
  is swallowed at `ArchiveAgentPage.tsx` ~L43 (FS-05) (no invariant class).
- **Worth fixing** — **J11 invalid pipeline template save returns 500:** Validate lists slug errors but
  Save stays enabled and `POST /api/pipelines` answers 500 “pipeline operation failed” (FS-14).
  Test: invalid ids → 4xx validation with the same diagnostics.
- **Worth fixing** — **J9 blank backend name persists:** clearing a backend display name in Settings →
  Backends saves `"name": ""`; backend creation already requires it (FS-09).
- **Worth fixing** — **J2 wizard hides `cwd_not_found`:** the onboarding Project step advances silently
  on a relative/nonexistent working directory although the API returns the warning (FS-04.R30).
- **Worth fixing** — **J2 OpenCode/OpenHands missing-adapter guidance:** only “Install it, then check
  again” with no link or command, unlike Claude/Codex.
- **Worth fixing** — **J13 archived tray default target:** from an archived source the disabled
  “Current agent” stays selected; Send returns 409 “the source agent is not running” (FS-13.R15).
- **Worth fixing** — **J16 worktree recreation unreported:** relaunch after the checkout was deleted
  recreates it silently; the launch response’s `worktree.recreated` is never read by the UI
  (FS-19.R7).
- **Worth fixing** — **J16 attention count on delete:** deleting a parked task leaves the dashboard
  needs-attention count stale until reload; cancel/create update live (FS-16.R14).
- **Worth fixing** — **J15 cancelled task shows a live wait:** a cancelled signal-armed task still lists
  “signal … (waiting)” in its detail.

Spec drift for the fix pass: FS-14.R44 names a three-step Setup/Runtimes/Review dialog; a later FS-14
requirement and the product use Setup/Review with a Customize runtimes disclosure. Matrix gap:
Think Tanks, phone remote control, project resources and config-source binding have no journey.

### Usability review 2026-10-14 — stopped-agent image wake (FS-03.A64)

Run report: [`usability-review-run-2026-10-14`](../archive/reviews/usability-review-run-2026-10-14.md).
**Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

- **Worth fixing — J7/A64 stale Retry after a definite image refusal:** `ui/src/components/chat/Composer.tsx`
  ~L140–168 sets the unconfirmed-send `retry` state for every failed send, so a stopped agent whose
  fresh peer refuses an image (422, nothing consumed) shows “Previous message was not confirmed.
  Retry previous send” beside the refusal. Retry repeats the same 422 with no visible change and the
  notice survives reload. It misstates delivery and invites a dead loop (FS-03.A64 “visibly
  refuses”). Test: a definite 4xx refusal keeps the draft and error but offers no unconfirmed-send
  Retry.

The recovered stopped-agent image implementation unit itself remains closed; live
provider/device acceptance gates and unrelated known risks above remain open.

Closed reviews through the `v0.16.0` cut are in the archive above.

## Changelog

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
