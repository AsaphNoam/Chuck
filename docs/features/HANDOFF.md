# Chuck — Implementation handoff

Settled history through v0.20.0 is archived in
[HANDOFF-through-2026-10-25](../archive/state/HANDOFF-through-2026-10-25.md) and earlier
[HANDOFF-through-2026-10-24](../archive/state/HANDOFF-through-2026-10-24.md) and
[HANDOFF-through-2026-10-23](../archive/state/HANDOFF-through-2026-10-23.md).
Use [AGENT-WORKFLOW.md](AGENT-WORKFLOW.md) for role rules; archived snapshots are not live queues.

## Current position

- **Published:** [v0.20.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.20.0),
  2026-10-25, tag at `a774d128` (saved file-viewer tabs, project-scoped groups/selectors,
  skip-permissions default, annotation focus). Release CI 40232088716 and main CI 40232088269
  passed; downloaded archive size and SHA256 match the manifest.
  [Notes](../archive/state/RELEASE-v0.20.0-notes.md).
- **Review state:** review unit MMD-VIEW (file-viewer Mermaid sizing) is available; memory fix unit MEM has one open **Worth fixing**
  finding (MEM-4), blocked on a decision below; MEM-1/2/3/5/6 are fixed. Delegation-guidance fix unit
  DELMSG has one open **Worth fixing** finding (DELMSG-1). The v0.20.0 architectural review
  ([receipt](../archive/reviews/architecture/v0.19.0-next.json)) is closed; ARCH-v020-1 fixed
  and verified. Provider-failure fix unit PERR is closed (PERR-1–4 fixed 2026-10-26); the
  frequent-429 field cause remains unmeasured (needs correlated request history if it recurs).
- **Permission-save investigation:** PERM-SAVE has one open **Worth fixing** diagnostic finding;
  the reported unchecked toggle's field cause remains undetermined pending affected server evidence.
- **Claude effort investigation:** CEF has one open **Worth fixing** finding: new Claude
  backends omit `xhigh` from their starter declarations. Sync's refusal to update effort
  metadata is specified behavior, not an import/parser defect.
- **Compact Think Tank navigation:** TTNAV-1/2/3 are closed by fix commit `1cf8fc9d`; the
  originating unit is complete. The only outstanding spec-lint failure remains the unrelated
  DELMSG-1 invariant tag noted below.
- **Checkout:** `main`, repository `AsaphNoam/Chuck`, pushed through the v0.20.0 release record.
  Clean release verification uses a detached worktree at `/private/tmp/chuck-release-<version>`.
  The primary checkout carries a pre-existing generated UI index edit; preserve it.
- **Future work:** direct-action migration remains paused on its transport blocker; passive
  quota snapshots remain TBD.
- **Checks:** unset `CHUCK_RUNTIME_ROOT`; use `GOFLAGS=-p=1` and a writable isolated Go cache.
  Native/device/credentialed limitations remain in the owed gates.

## Active change

**Running background-work indicator** —
[`running-background-work-indicator`](../ready-changes/running-background-work-indicator.md),
**resumable**. Operator decision 2026-10-29: option (a). Each popover row is named by its
related tool call's adapter title and shows state and elapsed time. The jump only locates where
the command started; no output is shown or copied (TS-04.R64). Real transcripts show no output
reaches the tool call: Codex results are `content: null` and Claude returns only a
"running in background, output in <tmp file>" notice. The operator also wants tool detail gone
from transcripts; that broader change is being defined in `docs/ideas.md`.

- **Done (slice 1):** the runtime publishes each generation's open task count into
  `status.background_running` (migration 63; `agentState.applyLive` on every status write,
  `republishLive` after each task spawn/terminal update), `AgentState.background_running`
  projects only with a running row, and the desktop `AgentCard` shows **N running**
  (TS-04.R104, TS-02.R59, FS-02.R83/A64 now shipped). Tests:
  `TestBackgroundRunningCountIsPublished`, `TestBackgroundRunningProjectsOnlyWhileRunning`,
  `AgentCard.test.tsx`. Checks on this tree: `go test ./...`, `-tags sqlite_fts5`, focused
  `-race` task tests, `ui` `npm test` (1060) and `npm run build`, `make build` passed.
  `make check-specs` fails only on the committed DELMSG-1 line lacking an INV tag (not this
  change).
- **Next (after the decision):** FS-03.R107/TS-08.R146 — remove `BackgroundTaskList`, add the
  composer `RunningTasks` indicator/popover from `collectTasks`, phone render check, FS-03.A83's
  fake-ACP scenario, `make embed`, the 1024/1440px render check.

## Acceptance gates still owed

- **FS-03.A80–A82 / FS-20.A28 / FS-21.A42 (file tabs):** physical-phone touch reordering and
  reconnect, non-Chromium engines, and rendered proof of the 500-total/2 MiB limits and the
  storage-unavailable notice (unit-tested only) remain owed.

- **ASF-2 native/device receipts:** deterministic fake-ACP wire counts, HTTP/CLI/component checks,
  and isolated desktop Chromium withdrawal/unknown-recovery journeys establish the repaired input
  boundaries. They do not establish credentialed Claude/Codex consumption timing or a physical
  phone's background/reconnect behavior. Keep those existing native/phone gates open.
- **Earlier carried checks:** phone component proof for definite-refusal Retry, live Claude
  receipt for late tool input, and rendered verification of the withdraw failed-recovery-read
  tail notice remain owed.
- **ASF-1 live setting receipt:** credentialed mid-turn effort/fast application remains unverified.
- **Test limits:** the concurrent closure hit the known quota-dispatch FTS5 transcript flake;
  full suites and five focused repeats passed. The two runtime cancellation tests that failed
  under `-race` were repaired by ARCH-v019-2 (`-race -count=20` passes).
- **Timed starts on a real Mac:** native sleep through a start time and a time-zone change
  across restart are proven only by store/fake-clock tests; no native macOS receipt exists.

- **FS-07.A12 — real tmux/iTerm2 ownership receipts (ASW-3):** neither tmux nor iTerm2 is
  installed on the implementing machine. On a host with both: two terminal agents against an
  already-running tmux server and a real iTerm2 launch; verify each CLI's environment/hook identity,
  native SessionStart → Stop → Resume, Cancel, Stop, viewer detach versus CLI exit, descendants and
  dashboard crash → Stop/Switch. Fake tmux/osascript fixtures (`drivers_test.go`) and the real-PTY
  bootstrap tests do not close this. Also unverified: iTerm2 shells that do not lead their own
  foreground group would now refuse launch (fail-closed) rather than run.

- **FS-16.A33 — task reporting under real provider policy:** run a credentialed sequential
  workflow (Claude `/work` → Codex `$review` → Claude `/fix` as tasks); each assignee must read
  its task and record `report_task_result` without supervisor prompting, and a late native
  background continuation must leave the card idle (FS-01.R52).

- **FS-20.A18 — phone group-header New agent:** independent phone-entry browser rendering
  passes, but the stubbed API journey does not prove accepted launches propagate to desktop and
  another phone. The two-device 390px journey for this entry point (including offline disable) is
  owed.

- **FS-16.A34 — live Chucky group placement:** ask Chucky to delegate work into a new group and
  confirm the launched agent lands in it; fake runtime and tool tests do not claim real-provider
  adoption.

- **Mobile annotation parity independent rendered review (workflow §14.4):** the review's
  `phone-render.mjs --annotation-check` attempt failed to bind `::1:5199` with `EPERM`.
  Approval-capable independent browser execution remains required; the implementer's stubbed
  browser journey does not close this review gate.

- **FS-13.A19–A21 / FS-20.A27 (mobile annotation parity):** paired iPhone and Android native
  long-press selection inside events and tracked files, then Annotate, with the keyboard open
  over the tray (delivery/errors reachable). The stubbed `phone-render.mjs --annotation-check`
  journey uses programmatic ranges and a short viewport, so it cannot close these.

- **FS-03.A71–A75 (workflow §14.4):** independent real-browser review remains partial. The
  2026-10-19 formats review successfully ran the approved loopback harness independently:
  `file-reader-journey.mjs` passed 981/981 checks across live/archive at 1024/1440/2137px in all
  appearances, including slow Reload and the boundary regression sweep. The existing room viewer,
  dashboard/phone navigation and native transcript anchoring still need independent receipts.
  The fixer's earlier `file-reader-journey.mjs` (with a
  review-regression journey: boundary sweep, slow Reload, tab leave/return, agent change) passes
  963/963 post-fix, and its boundary sweep crashes the pre-fix stage (evidence in shared
  resources `roomier-file-reader-review-2026-10-17/render-postfix/`). Check live/archive/room
  surfaces, the dashboard overlay and unchanged phone/dashboard
  navigation across widths/appearances, including slow Reload, resizing across the ~992px
  docking boundary, late diagrams, nested launch failure/retry, focus return and transcript
  visible-event/follow continuity through width reflow (native anchoring is unproven there).

- **FS-06.A29 / TS-04.R94:** bounded credentialed Claude and Codex receipt that `stop_agent`
  waits for Approve once under each provider's most permissive mode. **FS-06.A30:** real-browser
  desktop and paired-phone peer Stop journeys (reload while pending, Deny, Approve, later Resume).

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

- **FS-01.A28/A31/A33/A34:** bounded packaged/installed Claude/Codex terminal-quota receipts:
  real reset text/`_claude/rateLimit` wire, actual continuation, and isolated checks proving
  same account/model, Claude non-persistence, Codex `/rename` + archive, the dotted
  `CODEX_CONFIG` `mcp_servers.<name>.enabled:false` overlay and no auto-title turn. Parsers follow
  pinned formatter source; fake-provider/browser proof is not live delivery evidence.
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
  the `_meta.systemPrompt` request, not live adoption). ASR-I below adds a bounded paired
  performance experiment; no native-versus-Chuck quality or latency comparison has run.
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

- **MEM-4 live transcript bound:** the only full-history reader of `agentState.transcript` is
  `Runtime.Transcript`/`Registry.Transcript`, which has no product caller (12 test call sites in
  `internal/runtime` and `internal/server`); notice dedupe needs notices only. Runtime cannot read
  the durable log (`internal/transcript` imports `runtime`). Recommended: retain notices only,
  drop `Transcript` from the `Runtime` interface, and move tests to subscription recorders or
  durable reads (medium; ~12 test rewrites). Alternative: keep full history only for
  non-durable generations (smaller, but leaves a test-only mode in product code).

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

## Review queue

- **GROUP-SKILL** (2026-10-29, UI): group-background New agent seeds the editable Group
  field without leaking into later openings; `@` offers advertised commands alongside files
  and room participants, while `/` and the existing `#` open commands. FS-02.R77/A58,
  FS-03.R33/A17. Group regression failed before the fix and all 87 related focused tests
  pass; composer (51) and Think Tank page (32) pass. Full UI suite: 1068 passed, 1 skipped;
  UI styles, presentation contracts, type check and production build pass. The worktree's
  dependency symlink caused TS2742; local dependency copies resolve it. Both edited feature
  specs pass scoped checks; full spec check retains the pre-existing DELMSG-1 missing INV tag
  failure. Merged into main; combined-checkout production build and `make -o ui embed` pass.
  Generated index remains with the pre-existing dirty embedded UI owned by the concurrent
  pipelines work; no pipelines source or handoff edits were committed with this unit.

- **BGRUN-COUNT** (2026-10-27, small, Go + UI): live running background-task count —
  migration 63, `internal/runtime/{chat,tasks}.go` status writes, `internal/state` projection,
  `AgentCard` badge.

- **MMD-VIEW** (2026-10-26, small, UI CSS only): file-viewer Mermaid diagrams fill the reader width
  and center instead of collapsing to 300×150 — `ui/src/styles/features/agent.css`.

## Review findings

### PERM-SAVE — Permission checkbox investigation (2026-10-29) — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

**Report, verbatim:** “the skip permissions prompts checkbox is broken, clicking on it pops
'saving' for a fraction of a second and the checkbox stays empty”. Affected version, server URL,
browser and request/response logs were not supplied. Investigated checkout `052147af`, macOS arm64.
No server answered at `127.0.0.1:4317`, including an unsandboxed read attempt; no Chuck browser tab
was available and native inspection was unavailable because the Mac was locked. The installed CLI
shim points at 0.15.1, while repository `bin/chuck` records `65aa72a4`; neither identifies the
reported server. The retained `~/.chuck/dashboard.log` ends October 24 and cannot explain this click.

**Expected / attribution:** FS-04.R60/A39 require the toggle to persist both directions and explain
failed saves. The current implementation includes `181012ac`'s PDEF-R1 repair: optional boolean
decode/merge in `internal/server/config_handlers.go:869,977`, disk persistence, and next-launch
refresh. Existing component tests (2) and
`TestPutConfigSkipPermissionsRoundTripsAndPreservesOmission` pass on this checkout. Thus a current
server persistence regression is **undetermined**, not confirmed. Before `181012ac`, a server could
acknowledge this field with HTTP 200 while leaving it unchanged, which matches the reported symptom;
an outdated server is a hypothesis only. The onboarding cache shape was checked but its gate is
not mounted on the Settings route, so it is not an explanation for this report.

- **Worth fixing** — PERM-SAVE-1, **confirmed diagnostic defect; field attribution undetermined**
  (FS-04.R60/A39; INV §8/§11/§17). At
  `ui/src/features/settings/PermissionPolicyEditor.tsx:26`, a successful PUT response is installed
  without checking that `saved.skip_permissions` equals the submitted value. A server returning the
  unchanged policy makes Saving disappear and resets the checkbox without a readable error.
  `usePutConfig` (`ui/src/api/config.ts:295–306`) treats any HTTP 2xx JSON response as success.
  Existing component fixtures always merge the requested value, masking this actual pre-PDEF-R1
  response shape. Fix complexity: trivial/easy. Detect an acknowledged-but-unapplied policy at the
  existing mutation seam, retain the server's value, and surface the mismatch so a future report
  distinguishes an ignored write from a failed request. This diagnostic repair alone does not
  establish or repair the cause in the affected server.

  **Reproduction / verification:** added one component case using the historical real handler's
  unchanged-policy HTTP 200 shape. It reaches the settled, enabled, unchecked state and fails on
  the missing `alert`; retained as `it.skip` in `PermissionPolicyEditor.test.tsx` with this finding
  ID for the fix session. No product code or specification changed. Focused component suite passes
  with the reproduction skipped (2 passed, 1 skipped); the real server persistence/next-launch test
  passes (`env -u CHUCK_RUNTIME_ROOT GOFLAGS=-p=1`, isolated writable cache).
  Next field evidence: affected server version and the click's PUT plus subsequent GET response,
  distinguishing old server, save rejection, or later overwrite. No correlated evidence exists
  here to justify a separate persistence fix or claim that the setting actually changed.
  `git diff --check` passes. `make check-specs` fails only on the pre-existing DELMSG-1
  finding lacking an INV tag; this investigation does not alter that unit.

### CEF — Missing Claude extra-high effort investigation (2026-10-29) — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

**Report, verbatim:** “claude models don't have all efforts when adding / syncing the backend
(extra is missing)”. No affected model, installed CLI/adapter version, application version, or logs
were supplied. Investigated checkout `052147af`, macOS; no credentialed provider session was run.

- **Worth fixing** — CEF-1, **confirmed stale starter declaration / specification coverage gap**
  (INV §10; FS-09.R35–R38/R46, FS-04.R40): `internal/config/seed.go:81–84` declares only
  `low, medium, high, max` for all four Claude family aliases. `StarterBackend` at line 128
  reuses this template for item-scoped backend creation (`internal/server/backend_create.go:73`),
  so adding a Claude backend deterministically omits `xhigh`. This list dates to `ba1dd648`
  (2026-08-02). Current [Claude Code model documentation](https://code.claude.com/docs/en/model-config#adjust-effort-level)
  lists `xhigh` for newer models, including current first-party family generations, but not
  Opus/Sonnet 4.6; provider aliases, installed versions, and organization caps can differ.
  The picker renders exactly the configured list (FS-09.R37), so this is missing catalog metadata,
  not a UI filter or a global effort-vocabulary restriction. R46 specifies alias identity but
  leaves its starter effort list uncovered. Fix complexity: trivial/easy. Update the documented
  starter effort declaration and shared seed at this existing seam, with a focused assertion for
  `StarterBackend`/fresh seed; retain saved catalogs and defaults. Do not infer capabilities for
  arbitrary imported selectors or silently rewrite user-owned effort lists. Live support on the
  reporter's installed adapter/model remains **undetermined**.

**Sync classification: works as specified.** FS-09.R38/R45/R47 explicitly prohibit inferred Claude
effort discovery and overwriting existing entries. `ReadClaudeConfiguredModels`
(`internal/config/claudemodels.go:82`) gives newly imported selectors an empty effort list;
`addModels` (`internal/config/modelautosync.go:47`) skips existing entries, and startup,
source-connect import, and explicit refresh share that rule. Consequently refresh preserves an
existing four-level list and imports additional selectors without any levels. Automatic effort
discovery/update is a product change outside this finding; it needs a separate specification
decision about capability evidence and preservation of manual declarations. For an existing
compatible model, Settings → Backends → model → Effort levels accepts a manual `xhigh` declaration
(FS-09.R35/R37; `ui/src/features/settings/ModelRow.tsx:127`).

**Verification:** source trace establishes the omission and add-only behavior;
`go test ./internal/config -run 'Test(ReadClaudeConfiguredModels|SyncClaudeModels|SeedClaudeFamilyAliases)' -count=1`
passed with `CHUCK_RUNTIME_ROOT` unset, `GOFLAGS=-p=1`, and an isolated writable cache.
`make check-specs` reports the pre-existing DELMSG-1 missing invariant tag. No product or
specification changes, provider calls, or changes to the pre-existing generated UI edit.

### DELMSG — Verbose delegation investigation (2026-10-27) — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

**Report, verbatim:** “Chuck agents send exceedingly verbose messages to each other, telling
"launch agent of this model, do this thing" takes the thing and gives excessive instructions,
even if it's "have it run this skill"”. No affected agent, version, or log was supplied with this
report. Investigated checkout `fd738a70`, macOS arm64, and retained local October 17–19 transcripts;
those examples establish the symptom in earlier real use, not its frequency or every affected model.

- **Worth fixing** — DELMSG-1, **confirmed specification/guidance gap; probable contributor to
  observed model behavior**: operating guidance emphasizes assignment completeness without an
  equally clear default of concise communication and discretion over necessary detail. FS-18.R16 and
  `internal/config/seed.go:147` ask Chucky for relevant context and clear completion criteria;
  `AGENTS.md:94–101` also asks for deliverables and verification criteria. Those requirements can
  be satisfied through existing skills and shared context; spelling everything out is unnecessary.
  The shared guidance in
  `internal/server/launch.go:482–486` and
  `internal/agentknowledge/operating-chuck/{SKILL.md,references/coordinate-work.md}` does not fill
  that gap. FS-18.R3/R15/R16 assign detailed procedures to skills, but do not explicitly constrain
  outgoing task/mail elaboration; this is a coverage finding, not a transport-contract violation.
  Normal trigger: ask an agent to launch a specified model and run an existing skill. It can expand
  that into duplicated workflow, checks, closure instructions and runtime choices, wasting context
  and adding opportunities to drift from the requested scope. Fix complexity: trivial/easy.

  **Captured evidence:** `/Users/mcnoam/.chuck/sessions/a_c8e292/transcript.ndjson:2` requests
  work → review → fix skills for the file reader. Its `create_task` arguments at lines 133, 199
  and 327 contain 893, 1,074 and 964 instruction characters, repeating verification, commits,
  handoff and skill procedures. At line 526 the user asks why those overhead instructions were
  added; lines 1132 and 1280 repeat the concern. The native rollout
  `/Users/mcnoam/.chuck/codex/sessions/2026/10/17/rollout-2026-10-17T18-28-03-01a14a7a-2bcc-75a3-8b6d-bf3da86fabcf.jsonl:3,6`
  records both the Chucky mandate and delegation instructions actually supplied to that Codex
  0.161.0 session. Its later explanation names those instructions, but self-explanation does not
  prove model causality. A second example, `sessions/a_2d36f5/transcript.ndjson:1,109` under the
  same Chuck home, carries explicit pipeline guidance saying skills need no additional
  instructions, yet authors a 643-character `$work` assignment with procedural/output reminders.
  Some added scope/spec pointers are useful; character counts alone do not establish excess.

  **Boundary and history:** the three October 17 task rows match their recipients' first prompts
  exactly; the October 19 example matches its task instruction plus the current 255-character
  reporting suffix. `internal/messaging/task_tools.go:351–355` preserves the authored instruction;
  `internal/server/task_dispatcher.go:319–322` and
  `internal/runtime/activation_kinds.go:25–31` add only the specified FS-16.R49 reporting reminder.
  Mail likewise stores the supplied body (`internal/messaging/tools.go:211`). The extra procedure
  is already present in the sender's tool call. `807fef1b` (2026-10-03) introduced the current
  context/completion mandate while dropping the operator's earlier explicit “be concise”; this
  is a wording change, not a proven onset. `4ddedff1` (2026-10-17) subsequently added the host
  reporting reminder and verbatim skill-prefix guidance; earlier authored reporting instructions
  must not be mistaken for duplication of a host reminder that did not yet exist.

  **User direction (2026-10-28):** make Chuck's skills smarter and generally more concise. Explain
  the value of delegation, give agents room to exercise judgment, and avoid forcing explanations
  or exhaustive instructions when they do not help. This supersedes the original recommendation
  to prescribe the contents of a skill assignment.

  **Suggested fix and evidence:** align FS-18/TS-11 and the existing shared operating guidance
  around concise communication by default, with detail proportional to the recipient's needs,
  uncertainty and consequences. Explain why delegation matters: it can improve quality, speed or
  use of context while keeping responsibility clear. Let agents judge how much context an assignee
  needs and trust applicable skills and shared understanding. Routine delegation needs no mandatory
  rationale, checklist, message template or procedural restatement. This concision principle also
  covers ordinary messages and user-facing explanations, while retaining necessary evidence,
  material uncertainty and the host reporting contract. Extend FS-18.A12's pinned Claude/Codex
  scenarios to check both a simple skill request and work that needs richer context, judging useful
  communication and scope fidelity rather than compliance with a fixed format or length.
  Source-content checks cannot prove model compliance. Package refresh follows FS-18.R8; a
  seed-only edit would also leave frozen
  existing personas unchanged (FS-01.R3). No hard character cap or new coordination mechanism is
  needed. No product code/specification changed or synthetic model-output test added in this
  investigation. Verification: retained task/recipient prompt comparisons above, `git diff --check`,
  and `go test ./internal/server -run '^TestALaunchSpecTaskStartsItselfAndBecomesRunning$' -count=1`
  passed (unset `CHUCK_RUNTIME_ROOT`, `GOFLAGS=-p=1`, isolated writable `GOCACHE`; loopback-permitted
  retry after the sandbox denied the test listener). No fresh provider/model-adherence evaluation
  was run; the transport test proves instruction preservation only.
  The 2026-10-28 direction-only update passed `git diff --check`; it changes no product behavior.

### MEM — Memory retention investigation (2026-10-26) — **Fix model:** medium — Codex Terra or Claude Opus.

**Report, verbatim:** “I have a suspicion that Chuck is leaking memory, do a thorough
investigation and see if you can find anything”. No workload, measured growth, affected process,
version, or diagnostic log was supplied. Investigated checkout `43c214d1` (v0.20.0 plus release
records), macOS arm64. This establishes code defects, not which defect caused the reported suspicion.
The optional question about the affected process/workload had no answer at closure.

- **Worth fixing** — MEM-4, confirmed: live chat keeps an unbounded duplicate of all events.
  `internal/runtime/chat.go:295,2318–2322` retains every committed event in `agentState.transcript`;
  completed turns never trim it. This is growth for the lifetime of a running generation, distinct
  from post-Stop leakage. `Transcript` at `:1469–1479` additionally copies the event slice on reads.
  A disposable runtime exercise of 20 completed turns × 100 × 8 KiB deltas retained 2,020 events
  and 16,408,840 payload bytes. These are payload counts, not heap/RSS measurements. Browser reads
  already use durable transcripts; notice deduplication also scans this slice, so simply deleting
  it would break another consumer. INV §16; TS-01.R5/TS-02 transcript authority. Bound the runtime
  projection and preserve transcript-read and notice-idempotence contracts through the existing
  durable authority. Fix complexity: medium. Current retention seam is `69ef5951`, but full-history
  accumulation existed earlier; no exact regression release is asserted. Open on the decision
  under **Decisions needing input**.

Field attribution of the user's suspected growth remains undetermined (no heap/browser snapshot
or workload was supplied); disposable probes remain under `/private/tmp/chuck-memory-*`.

## Changelog

- **2026-10-29 — Fixed compact Think Tank navigation (INV §§8/13/17):** closed TTNAV-1/2/3
  from review `addb1f12` for the existing `compact-think-tank-chat-navigation` unit. Fix commit
  `1cf8fc9d` constrains long unavailable names, reserves active-link width to prevent flex reflow,
  and makes project-qualified labels unique against literal names. Focused UI tests pass 52/52;
  full UI suite passes 1061 with 1 skipped; style/presentation checks and UI build pass. The
  supplied Chromium harness passes 108/108 across all three appearances, 1024/1600px desktop,
  360px narrow layout and a 320–600px wrapping sweep. `git diff --check` passes. `make check-specs`
  remains blocked only by pre-existing DELMSG-1 missing INV tag. No other TTNAV findings remain.

- **2026-10-29 — Reviewed compact Think Tank navigation:** recorded TTNAV-1/2 (**Must fix**)
  and TTNAV-3 (**Worth fixing**) against the same originating unit; prior dead-CSS finding remains
  closed. Fresh focused tests/styles pass; rendered receipts reproduce the three scoped issues.
  Fixes belong to the next stage. Only handoff state changed; other queues and the dirty generated
  UI index are preserved.

- **2026-10-29 — Reconciled interrupted compact Think Tank navigation work:** originating
  [`compact-think-tank-chat-navigation`](../ready-changes/compact-think-tank-chat-navigation.md)
  is already finished (FS-21.R57/A40, TS-08.R126). Design `11245b90`; substantive implementation
  range `467bf836^..467bf836`; existing review `d46bd2a7` and dead-CSS fix `c85f94fb` are separate
  closure, with finished status retained by `88adeecd`. The implementation checkpoint records
  UI 975/975, Go tests/build, UI build and 42 Chromium checks; review records independent 42/42
  rendered checks, and fix closure records UI 976/976, style/build/spec passes. Browser receipts
  are historical: their `/tmp` artifacts are no longer present. At `02b919bc`, focused
  `ThinkTankTab.test.tsx`/`ChatPanel.test.tsx` pass 51/51, `npm run check:styles` passes 47/47
  plus presentation contracts, and `git diff --check` passes. `make check-specs` still fails only
  on the pre-existing DELMSG-1 missing invariant tag. No product/spec change, replacement unit
  or reopened review queue; subsequent pipeline stages own review/fix. Active background-work
  implementation and the dirty generated UI index are preserved.

- **2026-10-29 — Designed summary-only tool activity:** ready change
  [`summary-only-tool-activity`](../ready-changes/summary-only-tool-activity.md): one-line
  tool labels (runtime description, `Ran <program>`, short title, `Used a tool`), trimmed
  failure reasons, no argument/output bodies (FS-03.R108/A84, TS-08.R147). Waiting to start.

- **2026-10-29 — Annotation assignment design ready:** operator confirmed the default-on
  source-transcript checkbox and **New agent** terminology. FS-13.R35–R36/A25–A26,
  TS-03.R75, TS-02.R60 and TS-08.R148 define the waiting change
  [`optional-annotation-source-recording`](../ready-changes/optional-annotation-source-recording.md).
  No product code changed; existing active work remains active. `git diff --check` and twin-skill
  comparison pass; `make check-specs` remains blocked by the pre-existing DELMSG-1 line missing
  an invariant tag.

- **2026-10-27 — Running background-work count (FS-02.R83, TS-04.R104, TS-02.R59):** desktop
  agent cards show **N running** from a live status column the runtime republishes on each task
  lifecycle update; stopped/archived agents project zero. The composer popover (FS-03.R107) is
  paused on the no-output decision.

- **2026-10-26 — Designed running background-work indicator:** ready change
  [`running-background-work-indicator`](../ready-changes/running-background-work-indicator.md)
  replaces the transcript Background tasks list with a desktop composer **N running**
  indicator/popover and a desktop card count (FS-03.R107/A83, FS-02.R83/A64, TS-08.R146,
  TS-04.R104, TS-02.R59). Waiting to start.

- **2026-10-26 — File-viewer Mermaid sizing (FS-03.R37):** the viewer's shrink-wrapped
  (`fit-content`) `pre` left Mermaid's `width="100%"` SVG no definite width, so it fell back to
  300×150 at the left edge. A `pre:has(.mermaid-diagram)` now takes full width and the figure
  centers the SVG. `check:styles` passed; a Chromium layout probe measured 300×150 before and
  full width after.

- **2026-10-26 — Provider-failure fixes (INV §8/§11/§15/§16):** PERR-1 a prompt result carrying a
  current error-severity AIR failure (e.g. Codex exhausted 429 retries) ends `error` with one
  visible transcript error and error status, not `end_turn`; PERR-2 it settles unsuccessful
  (claimed continuation holds `non_quota_failure`) and confirms no inline mail; PERR-3 one safe
  correlated `provider turn failed` log (TS-04.R85); PERR-4 new/forked Codex sessions are named
  via local `/rename` before the first prompt, removing the background title-model turn
  (new TS-04.R103). TS-04.R86 extended. Each regression failed before its fix; `make test`
  (untagged + FTS5) and `make build` passed on the fix tree. No generic Chuck
  retry loop exists; the field 429 frequency cause stays undetermined (cross-conversation bursts
  and quota-recovery waves remain documented, unpaced risks).

- **2026-10-26 — Provider-failure investigation:** confirmed silent terminal non-quota failures
  and false success settlement; recorded PERR-1/2 plus missing terminal-result diagnostics PERR-3
  and one skipped reproduction. Expanded comparison adds PERR-4's unused background title-model
  request and records cross-conversation burst/recovery/mail patterns. No generic request-retry loop
  or increased retry setting found; frequent-429 cause remains undetermined without correlated
  request history. Product behavior unchanged.

- **2026-10-26 — Memory fixes (INV §4/§16):** MEM-1 revoking a launch token closes its MCP
  transport sessions (TS-04.R7); MEM-2 archive pages release history with their last reader and
  fence late fetches; MEM-3 journal close flushes a stopped turn's pending search text (TS-02.R16);
  MEM-5 draft-index eviction drops cached text (FS-03); MEM-6 ended children release open
  reasoning spans (TS-04.R63). Each regression failed before its fix. MEM-4 open on a decision.

- **2026-10-26 — Memory investigation:** confirmed MCP/session, archive, stopped-turn index,
  live transcript, draft-cache and child-span retention findings; recorded one MEM fix unit and
  three skipped reproductions. Product behavior unchanged; actual field attribution remains open.

- **2026-10-25 — v0.20.0 published:** `main` and annotated tag `v0.20.0` at `a774d128` were
  pushed (29 commits) on the operator's instruction. Untagged and FTS5 Go suites, full UI suite
  (103 files/1055 tests), `make dist VERSION=0.20.0` and the architectural release gate passed.
  Release CI 40232088716 and main CI 40232088269 passed; archive size/SHA256 match the manifest.
  Settled history moved to the 2026-10-25 archive. Manual gates remain owed.

## Blocked on human

- **File reader full UI closure:** permission requested through the asynchronous question to
  scope the pre-existing withdrawal test assertion to its exact error message (one test line).
  The four requested product changes are committed and independently reviewable. No unrelated
  test correction has been made without approval (repository instruction on separate fixes).

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

Latest published notes: [v0.20.0](../archive/state/RELEASE-v0.20.0-notes.md).
Earlier publication/campaign records remain in the archived handoff.
