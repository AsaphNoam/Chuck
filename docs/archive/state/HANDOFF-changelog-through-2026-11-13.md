# Chuck — handoff changelog through 2026-11-13

The changelog section the live handoff carried until 2026-11-13. The handoff no longer keeps
one: commit messages and Git history record what changed (workflow §4).


- **2026-11-13 — Fix:** closed CR-A8-1–8 and PIPE-CHAT-1/-3. Paused
  answers and Resume instructions commit with the dispatch they authorize and follow owner
  replacement (INV §2/§5/§11/§15/§19/§24). Controller holds survive restart, task reports return
  specific refusals, and the dispatcher recovers interrupted owners, missed binds and settled
  turns (INV §5/§8/§18/§21/§22). Both Go variants, focused race tests, both builds and 1,263 UI
  tests passed. Fake ACP verified the full reply and attachment and transcript replay.

- **2026-11-13 — Design:** [first-chat and backend setup](../ready-changes/instant-first-chat-and-backend-setup.md)
  is waiting to start. The operator approved one-click bundle selection and explicit native
  configuration linking, and requested direct links from provider setup errors to backend Settings.

- **2026-11-13 — Review:** CR-A10 durable state transitions at `8faec36f`: five findings recorded,
  one Must fix (a cancelled quota wait re-armed by a failed turn, probed) and four Worth fixing,
  with a transition-rule recommendation for quota intents, tasks and pipeline runs. Thirteen
  charter leads settled. **Fix model:** medium — Codex Terra or Claude Opus. Invariant sweep
  applied classes 1/2/3/5/7/10/15/18/19/20/22/23/24; classes 4/6/8/9/11/12/13/14/16/17/21 had
  no applicable surface.
- **2026-11-13 — Fix:** closed CR-A9-1–8 at `0da7c581`. Quota stops settle requested Pause or
  End at the room boundary, and settled quota intents release their room hold (INV §10/§15/§18/§20).
  Setup failures retain Retry launch without spinning, and ended rooms expose judge holds and Resume
  (INV §8/§16/§18/§22). Reserved launches use the lifecycle claim and retry incomplete identities
  (INV §2/§4/§5/§23/§24). Startup projects recovered rooms to their runs and pipeline Stop preserves
  an existing room end record (INV §1/§15/§18). Both Go variants, the Go build, focused race checks,
  the UI build and 1,261 UI tests passed. In Chromium at 1024px, keyboard Resume cleared a seeded
  hold and started the fake-provider judge while the discussion stayed ended. The live quota gate
  remains above. CR-A8 and its linked PIPE-CHAT findings remain open.

- **2026-11-13 — Review:** CR-A9 reviewed at `8bcd53ed`. Eight findings recorded, four Must fix,
  six confirmed by throwaway probes: End requested before a quota stop strands a room for good,
  Resume on a failed setup spins the sweep, a hold on an ended room blocks the judge unseen, and a
  Stop or Archive inside a room launch is accepted and breaks the member. The unit stays open at
  medium — Codex Terra or Claude Opus.

- **2026-11-13 — Review:** CR-A8 reviewed at `35f33a07`. Eight findings recorded, five Must fix;
  three confirmed by throwaway probes (a restart destroying a held controller action, six report
  refusals answered as `already_reported`, an answer lost on owner replacement). The unit stays
  open at difficult — Codex Sol. PIPE-CHAT-1 and -3 are fixed with CR-A8-3, -4 and -7.

- **2026-11-13 — Design:** drafted first-chat setup with automatic Claude/Codex readiness and one
  provider action in Settings. Scope confirmation is pending.

- **2026-11-13 — Fix:** closed PIPE-CHAT-2 and -4; PIPE-CHAT-1 and -3 stay open on CR-A8, so
  the unit stays open. A failed prompt read (e.g. hydrating a stopped chat) no longer claims a
  withdrawal; only reads after a confirmed withdrawal can, and a failed retry shows a new outcome
  (INV §8/§11). Chat input that continues a paused run now logs request/agent/run/visit, surface
  and delivery without text (INV §15). Both reproductions failed before the fix. Spec deltas:
  FS-03.R90/A70, TS-02.R55. At this fix's tree, `make test` (both Go variants), `make build`,
  UI `npm test` (1246 passed) and UI build pass; focused `-race` of the paused-run answer tests
  passes.

- **2026-11-13 — Fix:** closed MAIL-DETAIL-1–5 and its unit. Sent/read records now stay with
  their calls when a turn completes (INV §1/§2). Records attach only to the one call the run's order
  proves, never FIFO; ambiguous parallel records stay standalone. Codex `args.tool` needs a
  messaging `args.server` (INV §11/§17). Mail detail bounds header and error fields before bodies,
  holding 64 KiB for refused sends (INV §16). TS-08.R161 now matches TS-14.R10, and a capture test
  proves room activity drops mail detail (INV §10). The Mail button is 44px on coarse pointers
  (INV §13). Its emulated-phone Chromium check measured 44×44, tapped, and stayed compact on
  desktop. New tests failed before the fix. Spec deltas: TS-08.R161, TS-04.R108. At this fix's
  tree, `make test` passes both Go variants. UI `npm test` passes 1244 tests with 2 skipped. UI
  build and `make build` pass. A first `make test` run failed only
  `TestDeferredAssignmentResumesOriginalAgent/retry_failure` ("lifecycle transition already in
  progress"). That lifecycle test is outside this diff; it passed 5/5 alone and on the full rerun.
  Watch it for flakiness. Owed: end-to-end re-render and physical phone tap (A33 gate above).

- **2026-11-13 — Review planning:** queued two subsystem reviews, CR-A8 (pipeline pause, answer and
  recovery) and CR-A9 (Think Tank lifecycle and quota join), chosen from the October audit's scope
  and fix history plus fresh Sonnet code maps at `e8a8a128`. Charters carry ten and eleven
  unreproduced leads; four were checked against source by the planner. A third candidate, one
  transition rule per durable state machine, is a design topic for after these reviews and is not
  queued. No product code, specification or finding changed.

- **2026-11-13 — Design:** five single-owner refactors from a review of 60 days of invariant
  evidence and fix history are waiting to start as separate ready changes: typed turn outcome
  (FS-03.R117/A92, TS-01.R42), request-id prompt echo (FS-03.R118/A93, TS-08.R163), shared
  desktop/phone input delivery (TS-08.R164), shared resume start and teardown in Switch
  (TS-01.R43), single-owner turn state (TS-01.R44, after R42). All planned; TS-08 is now Partial.
  Operator decisions: five changes not one; the phone gains no local echo; delivery records for
  the launch assignment prompt and stall nudge are out of scope. No product code changed.
  `make check-specs`, twin-skill comparison and `git diff --check` pass. Not traced: whether the
  Think Tank launch path holds a lifecycle claim.

- **2026-11-12 — Fix:** closed the QUOTA-FORK review (QUOTA-FORK-1–7; INV
  §2/§5/§7/§8/§10/§11/§16/§17). One Go decoder reads the SDK's Claude usage shape
  (`model_scoped` array, unknown members); an undecodable success is a failed read keeping
  earlier windows; `model_scoped` windows count by family. The Claude adapter patch forwards only
  window members (new pinned hash, still `+chuck.2`, unreleased). Only admission from `scheduled`
  pre-checks the snapshot. Edit on a mid-turn message is disabled with its reason (API refuses
  likewise); copied turns say so. First-read failure wording, phone summary 44px, A40/A41/A49/A52
  coverage stated. Both Go variants (`go test ./...`, `-tags sqlite_fts5`), `make check-specs`,
  `make build`, 1,240 UI tests and the UI build pass; one earlier full run timed out
  `TestQuotaStoppedBorrowedTaskDefersUntilRuntimeCapacity` (bypasses the changed dispatcher;
  5/5 alone and green in the rerun — load-sensitive, watch it); the patched
  bundle contract tests pass against the pinned 0.85.1 input. Not run: `make dist`, hosted CI.

- **2026-11-12 — Fix:** closed ANNOTATION-MAIL-HEADER-1 (INV §8/§17). Single-message mail
  notices show the sender once; batch attribution and expansion remain intact. The saved and
  expandable regression cases failed before repair and pass afterward. `make check-specs`, UI
  tests (1,230 passed, 2 skipped), and UI build pass.

- **2026-11-12 — Work:** shipped expandable agent mail (FS-06.R45/A33, TS-04.R108, TS-08.R161;
  INV §2/§8/§10/§11/§13/§15/§16). Received inline mail keeps a clipped copy in its notice; sends
  (including refusals) and reads append a display-only notice to the caller's live chat
  transcript; the UI attaches them to the matching tool line. Agent-facing projections are
  unchanged (notice metadata disposition). `make test`, 1,220 UI tests and the UI build pass.

- **2026-11-12 — Fix:** closed the Think Tank templates review (TT-TEMPLATE-1–10; INV
  §1/§3/§5/§7/§8/§9/§10/§11/§14/§15/§16/§17). ACP mapping strips echoed
  `participant_instructions` before transcript/history/SSE. Migration 73 stores template version
  and replaces source ids in intent with a request fingerprint (TS-14.R30–R32 updated). Room
  save reads roster only and accepts members switched to Terminal. UI fences late template loads
  (edits and blank), retains save command ids for lost-response retry, flags unsupported saved
  effort/fast, pages templates on demand, and counts instruction runes. `make test`, 1,211 UI
  tests and the UI build pass. TT-5 is exercised at the state layer, not the switch route.

- **2026-11-12 — Fix:** closed the no-raw-ids review (UI-IDS-1–6; INV §2/§7/§8/§10/§16/§17).
  Annotation cards name another-agent recipients or read *Another agent*; non-Git start/fork
  errors and unknown-project diagnostics name the title, not the slug. Prerequisite options and
  peer Stop rows reuse the duplicate-name rule (FS-06.R41 now says so). Push resolves titles only
  for notes it sends, with "an unavailable project" fallback (FS-20.R19/R33). Breadcrumb harness
  asserts no id text (258 Chromium checks pass); phone/run tests seed `/api/projects`. `make
  test`, 1,204 UI tests and the UI build pass. Unrelated `/api/roles` handler gaps remain in
  pipeline and phone-app tests.

- **2026-11-12 — Design:** native quota snapshot and fork from a message are ready as
  [native-quota-snapshot-and-message-fork](../ready-changes/native-quota-snapshot-and-message-fork.md)
  (FS-09.R83/A52 replacing R81/A50, FS-01.R58–R59/A40–A41, TS-04.R109–R111, TS-03.R85–R86,
  TS-02.R65, TS-08.R162). The operator rejected a full move off ACP and approved, for these two
  items only, a typed quota-read patch per packaged adapter and the adapters' existing
  `_meta.jetbrains.air.fork` key; Edit always forks; quota recovery may use the snapshot with the
  shipped behavior as fallback. Pinned-bundle evidence is in the change file; Claude's idle
  usage read and both live receipts remain acceptance gates. Amended the same day at the
  operator's direction: one shared in-memory snapshot per backend quota scope, read automatically
  and shown by every chat of that backend, with no Refresh control or route.

- **2026-11-12 — Design:** agent mail visibility is ready as
  expandable-agent-mail-activity
  (FS-06.R45/A33, TS-04.R108, TS-08.R161, FS-03.R108 pointer; TS-08 is now Partial). The operator
  chose to keep received bodies with the transcript, expand the notice and mail tool lines in
  place, leave agent-facing views unchanged, render bodies as Markdown, and have Chuck record
  sent and read mail itself. Evidence from real transcripts: Codex saves null mail tool results,
  and Claude saves empty send arguments unless a permission request records them.

- **2026-11-12 — Review:** internal-ID display (`178425ba`) remains open for UI-IDS-1–6.
  Five focused component files passed 143 tests with the fixture gaps above. Presentation checks
  and focused Go remote home/push tests passed. Chromium's populated pipeline fixture passed 120
  checks at 1024px and 1440px in Core,
  Sky & Grove and Studio. The breadcrumb harness stopped on its obsolete id-chip assertion.
  Invariant sweep applied classes 2/7/8/10/11/13/14/16/17. Classes 1/3/4/5/6/9/12/15 had no
  applicable surface. The peer Stop identity contract needs reconciliation, as recorded above.

- **2026-11-12 — Fix:** closed the v0.23.0 main-CI annotation replay race (RELEASE-023-CI-1;
  INV §17 tests prove their contract independently). The self-annotation fixture now holds
  `taskSweepMu` across admission and synthetic restart/replay, releasing it before explicit
  dispatch, so asynchronous dispatch cannot consume the input first; replay safety assertion and
  product receipt safety unchanged. Focused test 30/30, `-race` 10/10; `go vet` and both Go
  variants of `./internal/server` pass. A new hosted main CI run must confirm.

- **2026-11-11 — Work:** finished [Think Tank templates](../specs/features/FS-21-think-tanks.md)
  (FS-21.R60–R63/A43–A45, TS-14.R30–R33). `make test`, `make build`, full UI `npm test`
  (1201) and UI build pass; legacy v47 migration fixture gained the members table every real v47
  database has. No rendered browser pass.

- **2026-11-11 — Fix:** closed the v0.23.0 main-CI UI failure (RELEASE-023-CI-2; INV §17
  tests prove their contract independently). CardGrid cases rendered, clicked or delivered a
  waiting transition before the grid's layout read settled; a late read overwrote clicked panes
  or reseeded the transition (correct FS-02.R61 behavior). Cases now wait for the seeded
  layout's gap marker; product unchanged. Base fails 7/42 with a 300 ms delayed layout read,
  fix passes 42/42 delayed and normal; full `npm test` 1195/1195 twice; `make check-specs` ok.
  UI build not rerun (test-only). A new hosted main CI run must confirm.

- **2026-11-11 — Work:** removed internal ids from user-facing text (operator request):
  breadcrumb id chip, run/room/proposal/template code chips, `name || id` and `(id)` / `· id`
  suffixes, project-slug fallbacks (now title, blank while loading, or *Unavailable project*),
  phone `role@project` titles and push bodies. Specs: FS-02.R25/R69/A13, FS-03.R98, FS-12.R61/R70,
  FS-14 builder panel, FS-20.R12/R19, TS-08.R60/R91. `make test`, full `npm test` and UI build;
  CardGrid waiting-input flake (RELEASE-023-CI-2) is independent.

- **2026-11-11 — Design:** Think Tank templates were made
  ready to start. Confirmed project-independent snapshots that load fresh launch settings into the
  creation modal, editable participants, and optional room-specific instructions
  (FS-21.R60–R63/A43–A45, TS-14.R30–R33).
- **2026-11-11 — Fix:** closed desktop chat composer editing (COMPOSER-LIST-1–2; INV §8
  parsed user-facing data / §10 wiring drift). Shift+Enter no longer numbers lines
  inside a fence opened on an unordered item, and an item after an ATX heading continues from
  any start number; paragraph fallback unchanged. Both regressions fail before the fix.
  `make check-specs`, 69 composer/editing tests and the UI build passed; full `npm test` failed
  only the pre-existing CardGrid load case (RELEASE-023-CI-2, also fails without this fix). Both
  desktop entry points share `editNumberedList`; no rendered Chromium rerun.
- **2026-11-11 — Review:** desktop chat composer editing remains open for COMPOSER-LIST-1–2.
  Specification checks, 67 focused tests and the existing Chromium keyboard/pointer journey
  passed. A separate Chromium probe reproduced both failures in full agent chat and the expanded
  dashboard (`/private/tmp/chuck-composer-review-probe.mjs`, fixture API). The app's Markdown
  parser independently confirmed both block contexts. Tests use explicit text/selection and
  request/transcript expectations. No pending local choices. Invariant sweep applied classes
  1/2/3/5/8/10/13/16/17. Classes 4/6/7/9/11/12/14/15 had no applicable surface.
- **2026-11-11 — Review:** closed pipeline stage work status (`c947386b`, FS-14.R39/A17 and
  TS-08.R149) with no findings. Independent Chromium runs exercised the routed fixture at
  1024px and 1440px in all four appearances, including reduced motion and live/archive links.
  Fixture updates confirmed waiting → running → finished in place in Core at 1024px.
  Specification checks and all 10 run-browser tests passed. No fix model is needed.
- **2026-11-09 — Implementation:** finished
  [desktop composer editing](../ready-changes/desktop-chat-composer-editing.md).
  Specification checks, 1,193 UI tests, the production UI build and both rendered Chromium
  journeys passed. Added the invariant classifications required by the spec check to the two
  open CI findings.
