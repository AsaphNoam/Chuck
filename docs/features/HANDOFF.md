# Chuck — Implementation handoff

Settled history through v0.22.0 preparation is archived in
[HANDOFF-through-2026-11-06](../archive/state/HANDOFF-through-2026-11-06.md).
Use [AGENT-WORKFLOW.md](AGENT-WORKFLOW.md) for role rules; archives are not live queues.

## Current position

- **Locally tagged:** v0.22.0 at `62675131`, version confirmed on 2026-11-06; unpushed.
  [Notes](../archive/state/RELEASE-v0.22.0-notes.md).
  Latest published release remains [v0.21.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.21.0).
- **Architecture:** combined audit and all four repairs are complete; independent preparation
  signoff at `6dfae975` covers `be766a50` with no new findings.
  [Receipt](../archive/reviews/architecture/v0.21.0-next.json). Release gate passes.
  Focused quota-flow audit on 2026-11-07 at `8d9cb623` found three additional Must-fix and two
  Worth-fixing defects; NESTED-QUOTA-1 remains independently owned below. The new findings block
  current-candidate release; the earlier gate result above describes v0.22.0 preparation only.
  [Quota report](../archive/reviews/architecture/v0.22.0-next-quota.md) /
  [bounded receipt](../archive/reviews/architecture/v0.22.0-next.json); this focused audit is not
  combined next-release signoff.
- **Verification:** full `make test` (both Go variants) and 1172 UI tests pass at `84f65de9`,
  reused for unchanged product inputs. Release preparation passes package/spec checks and
  `make dist VERSION=0.22.0`; embed bytes are unchanged. Logs:
  `/private/tmp/chuck-release-{dist-final,knowledge,specs}.log`.
  Final tagged-candidate `make build VERSION=0.22.0` passes with exact `62675131` metadata
  and `sqlite_fts5` (`/private/tmp/chuck-release-build-tag.log`).
- **Publication:** not authorized or pushed. Supported arm64 `macos-15` assembly, fresh archive/
  install checks and clean-clone CI remain owed on publication; prior main CI failure was repaired.
  Native/provider/device gates below remain owed.
- **Field causes:** EFFORT-BUDGET awaits the affected worker's manifest/trace; no cause or fix is
  established. Frequent-429 and unchecked permission-save reports need affected server evidence.
- **Future work:** direct-action migration stays paused on its transport blocker; passive quota
  snapshots stay TBD. Project chat search is finished and available for independent review.
- **Checks:** unset `CHUCK_RUNTIME_ROOT`, use `GOFLAGS=-p=1` and a writable isolated Go cache.
  Full suites need loopback-capable execution.

## Active change

No implementation is active. Project chat search finished on 2026-11-06; see the review queue.
Unrelated `docs/ideas.md` edits remain preserved.

Verification for product inputs through `59c15236`: `make test` passes both Go variants,
`make build` and `make embed` pass, and all 1180 UI tests pass. The additional untagged fallback
priority/paging regression passes after the full run. Unset `CHUCK_RUNTIME_ROOT`, use
`GOFLAGS=-p=1 GOCACHE=/private/tmp/chuck-search-go-cache`; full suites need approved loopback
access. Logs: `/private/tmp/chuck-search-{final-go2,final-build2,embed,full-ui,fallback-final}.log`.
All 23 real-browser checks pass at 1024/1440 across all four appearances, including collapsed
turns, later merged deltas, cross-event phrases, annotations, attachments, code, direct reload/back,
keyboard entry and stable reading during a new fake-provider reply. Report/screenshots:
`/private/tmp/chuck-search-browser4/`; deterministic matrix: `/private/tmp/chuck-search-matrix/`.
Error/metadata/unavailable browser branches use controlled read responses; real tagged/untagged
SQLite and HTTP tests cover those storage contracts. No native-provider or performance claim.
Temporary browser fixtures and the Vite server are stopped.

Release v0.22.0 remains verified and annotated locally in `/private/tmp/chuck-release-next`.
Publication is not authorized; no push is part of this implementation.

## Acceptance gates still owed

- **File freshness/editing (FS-03.A86–A88):** actual Chromium desktop live/archive/room/Focus
  journeys and the real phone bundle at 360/390/430px pass. Phone SSE/tracked-file admission was
  fixture-controlled; physical-device, real paired reconnect and non-Chromium gates remain open.

- **FS-03.A85 native default browser:** JXA lookup ran but this environment has no configured
  default web browser. On a normal Mac, right-click a self-contained HTML file and select
  **Open in browser**; confirm the original file opens in the default HTTPS browser (even when
  `.html` is associated with an editor). HTTP dispatch, retained sources, remote refusal, native
  argv/timeout/error handling and in-app live/archive/room rendering are automated evidence only.

- **GOAL-LED evidence (FS-14 §6):** A61 run-start-to-notification browser journey; A64 attachment
  variant and live desktop/phone composer check; A67 fake-provider invocation-reason capture; A68
  bounded packaged Claude/Codex automatic-replacement probe (credentials).
- **QUOTA-RPC live receipts (FS-01.A37/A39):** with credentials under a real quota refusal,
  record one bounded Claude and one Codex terminal-RPC receipt (category value, wording) and the
  resulting automatic continuation at reset plus one minute. Fixtures are captured/source-backed only.

- **Pipeline worktree packaged probe (TS-09.R65):** with credentials, start an isolated run from
  the packaged app and confirm a Claude and a Codex stage agent actually run (and resume) in the
  run checkout; fake-provider capture and real-Git fixtures pass.
- **BROWSE-LAT native picker:** JXA activation probe took 0.10s versus AppleScript's 2.29s,
  and a real JXA panel returned `User canceled. (-128)`. Computer use could not target the
  osascript panel. Still verify selection returns a usable absolute path, the panel appears
  over the browser, and click-to-visible-panel timing; activation-only timing is not panel latency.

- **BORING-NIGHT rendered evidence (FS-12.A42):** Settings/Appearance, live switch, reload
  persistence, the project dashboard with busy/idle/error cards, the agent screen, a Go code
  block, a diff fence and a Mermaid diagram were rendered in Chromium (review 2026-10-31). An
  xterm terminal (selection uses `--ad-action-secondary` `#b3d2ea` under `#cedbec` text),
  overlays/permission prompt and the phone Theme select are not yet rendered.

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

## Changelog

- **2026-11-06 — Project chat search finished:** compact project-heading search reuses SQLite
  FTS5 with partial words and running/stopped/archived priority. Exact source references open the
  shared live/archive renderer at the real passage; older rows reconstruct only after byte equality
  is proved, otherwise normal opening stays available. Migration/repair remain transactional and
  source reads/provenance/previews are bounded. FS-05.R46–R53/A28–A32, TS-02.R63–R64,
  TS-03.R82–R84 and TS-08.R156–R157 are implemented. Closure evidence is above; review is available.

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

## Review queue

- **Project chat search and matching-passage navigation:** available for independent `/review`.
  Unit starts after `41edd4ed`, with substantive commits `d950c437`, `80d80257`, `62865d1f` and
  `59c15236`; the closure also strengthens the untagged fallback paging test. Requirements:
  FS-05.R46–R53/A28–A32; TS-02.R63–R64; TS-03.R82–R84; TS-08.R156–R157.
  Sweep provenance/digest/source equality, optional-map bounds and repair, SQL scope/ranking before
  paging, shared UTF-8 source reveal and reading-mode lifetime. Verification and browser receipts
  are recorded above. This is one unit; checkpoint/closure commits do not create separate units.

## Investigations awaiting evidence

### EFFORT-BUDGET — worker infers a work budget from its own configuration (2026-11-05)

**Report (verbatim):** “agents in a pipeline read the implementation manifest, saw their reasoning
was set to low, and decided to self limit on an imaginary budget, which made them fail the task
because 'the budget ran out'”. No run/agent identity, timestamp, manifest path, version or logs
were supplied. Investigated source at `47268443` on macOS and read-only local Chuck state and
recorded conversations. No matching incident was identified; the report's “implementation
manifest” must not be assumed to mean Chuck's runtime `ExecutionManifest`.
**Clarification (verbatim):** “The worker one the one that saw its own config and made up it's
own additional budget”. The reported initial failure is the worker's self-imposed constraint;
coordinator acceptance and subsequent recovery exhaustion have not been established.

**Expected behavior / classification:** FS-14.R97–R99 authorizes the work needed to meet the goal,
requires the owner to evaluate failed/blocked child evidence, and reserves human attention for
a real unavailable decision, permission, checkpoint or exhausted operation. FS-18.R15/R21 and
TS-11.R24 define runtime facts and requested-delivery metadata; TS-09.R68 explicitly leaves
semantic completion judgment with the owner. Low effort does not change those obligations.
The reported premature stop would violate that completion contract. **Diagnosis: undetermined.**
The absence of an explicit warning against this particular inference does not establish a prompt
defect or justify adding standing context. The earlier probable guidance finding, mandatory fix
classification and model recommendation are withdrawn. No code defect, specification gap or
missing diagnostic at the actual failure point has been demonstrated from the available evidence.

**Evidence and limits:** no effort-to-work-budget conversion was found in the inspected source.
Actual recovery is `pipeline.recovery.used/limit`, default three
(`internal/state/pipeline_decisions.go:27,229–244`), charged on repeat/backward/recovery
transitions (`internal/pipeline/decisions.go:67–87`), independently of effort or report prose.
`internal/state/pipeline_tasks.go:317–388` validates report authority/structure, as specified;
acceptance does not certify the truth of a claimed blocker. The existing
`TestRecoveryExhaustionPausesAndResumeRenewsTheAllowance` and
`TestQuestionPausesAndTheAnswerContinuesTheRun` pass (focused `go test ./internal/pipeline`,
2026-11-05). They verify genuine recovery/pause mechanics, not the reported model behavior.
No behavioral reproduction or skipped regression is claimed. Next evidence needed is the affected
worker's exact assignment, config/manifest tool result, and recorded explanation/report of the
stop. Inspect those before choosing a remedy; determine whether an instruction, misleading
metadata presentation, actual provider limit, or unsupported model inference supplied the alleged
budget. Any proposed change needs a bounded comparison with independently checkable completion
evidence; prompt delivery alone cannot establish improvement. History separates manifest recording
(`90f27319`, expanded `0f89e683`) from goal-led
recovery (`fb410fdb`); it does not establish when the reported inference began. No product code,
specs, live user state or unrelated dirty files changed.

## Investigations with fixable findings

### NESTED-QUOTA — Add source 3 did not automatically continue (2026-11-07)

**Report (verbatim):**

> an agent looked into a single event where the chuck quota continuation didn't work, this is the explanation it gave:
>
> Chuck did not schedule continuation because this failure arrived as a tool result, while the main conversation ended normally.
>
> In “Add source 3” (a_e73e01), I verified:
>
> Event 800: a tool failed with the provider’s spend-limit error.
> Event 801: the assistant repeated that error as ordinary text.
> Event 802: the turn finished with stop_reason: end_turn.
> Chuck’s recovery logic requires a recognized quota failure from the main provider session. It does not schedule recovery from errors embedded in tool output or assistant text. Consequently, this turn never entered quota recovery for 7:21pm.
>
> My earlier spend-cap explanation was incomplete: the current code accepts a verified reset time even when the message mentions a spend limit. The issue here is how the nested failure reached Chuck, not the monthly-limit wording.
>
> Your later “Keep going” successfully restarted work. This exposes a recovery gap for nested provider failures; I made no changes.

**Correction (2026-11-07):** the initial works-as-specified conclusion at `8a8d7fab` is withdrawn.
The operator's screenshot prompted checking native provider history, which proves that event 801
was a root provider-generated API error, not an ordinary assistant repetition of child output.
The normalized transcript alone could not establish its origin. A protocol `end_turn` does not
prove the model successfully completed its turn or the user's task.

**Classification / confidence:** root quota refusal flattened into successful-looking completion
is **confirmed** from native and Chuck logs; exact adapter/runtime normalization cause is
**probable**, pending the historical raw ACP carrier or an equivalent replay. FS-01.R40 and
TS-04.R86 require provider terminal quota state and unsuccessful completion despite misleading
`end_turn`. FS-01.R55/TS-04.R105's transcript/child exclusions remain correct, but do not explain
away this independent root refusal. FS-01.R57 allows monthly-spend wording beside a valid future
reset. No new policy for restarting a parent solely from child evidence is needed for this incident.

**Evidence:** read-only `read_agent_transcript(a_e73e01, since_seq:790/799)` on 2026-11-07
(snapshot through 987) identifies **Add source 3**, Hoid project `hoid-20261029t181502z`.
The bounded transcript shows foreground `think` (`subagent_type:general-purpose`,
`run_in_background:false`), nested assessor activity, then the failed tool result:

> Agent terminated early due to an API error: You've hit your monthly spend limit · raise it at https://claude.ai/settings/usage?from=cc_cli_limit_message · your session limit resets 7:20pm (Asia/Jerusalem) (error type rate_limit, HTTP 429, request id req_011CgmvuwtPYAkJmNTSKQBUv, model sent to the API: claude-opus-5-5)

Read-only local `/Users/mcnoam/.chuck/sessions/a_e73e01/transcript.ndjson` independently confirms
exact seq 800/801/802 at `2026-11-06T15:43:01Z`, with seq 801 `assistant_text` and seq 802
`turn_end` carrying `stop_reason:end_turn` and no quota evidence. `Keep going` later resumes work.

**Decisive native evidence:**
`/Users/mcnoam/.claude/projects/-Users-mcnoam-Projects-Hoid/10fcd54e-100b-458c-a93a-355101fd81d7.jsonl:456`
records `isSidechain:false`, `model:<synthetic>`, `isApiErrorMessage:true`,
`apiError:usage_limit_reached`, `error:rate_limit`, HTTP 429, root request
`req_011Cgmvuya5hGc3c4BRmns1J`, timestamp `2026-11-06T15:43:01.882Z`, and the exact displayed
quota sentence. Its structured `quotaLimits` reports rejected `five_hour` and
`resetsAt:1793985600`. The child log
`10fcd54e-100b-458c-a93a-355101fd81d7/subagents/agent-a87a85219cf3999a7.jsonl:21`
records a separate rejected request `req_011CgmvuwtPYAkJmNTSKQBUv` at `15:43:01.485Z`;
root line 455 contains that child's failed tool result. Thus both child and root failed.
The root synthetic message has zero input/output token usage; treating it as generated explanatory
prose was unsupported. Incident CLI version is `2.1.296`; exact historical adapter artifact,
Chuck version and Auto continue configuration are not yet established. No raw ACP capture or
historical quota-database inspection is claimed; exact dispatch eligibility remains unverified.

**Code evidence at `4255da86`:** `internal/runtime/chat.go:onNotification` routes child frames away
before root failure capture. `internal/runtime/session_failure.go:decodeSessionFailureRecord`
requires a typed AIR session-failure envelope; failed-tool text is not such an envelope.
`internal/runtime/quota.go:completePromptTurn` obtains quota evidence from root failure metadata or
`rpcQuotaRefusal` on the terminal prompt error, never transcript prose.
`internal/server/quota.go:settleQuotaTurn` creates an episode only with non-nil quota evidence.
An explicit terminal Claude `data.errorKind:rate_limit` is independently excluded by the classifier;
native usage-limit failures can also carry `error:rate_limit`, so raw native and ACP fields must not
be conflated. No proof yet establishes the actual terminal ACP category/result for this incident.
`6bbf4c3f` introduced the terminal-RPC fallback with the same text/child exclusions;
`945ce4ca` repaired reset resolution, not evidence scope. Existing regression sources
`TestQuotaNotificationIgnoredForReplayAndChildScope` and
`TestTerminalRPCQuotaRefusalSharedByActivationNotProse` protect that boundary, but do not prove
normalization of this native synthetic root error. No tests run or added; no equivalent wire replay
has been established. Only this state record changed; product code, specifications and unrelated
`docs/ideas.md` edits remain untouched.

**Bounded independent adapter inspection:** current global installation at
`/Users/mcnoam/.local/lib/node_modules/@agentclientprotocol/claude-agent-acp` declares 0.59.0
with SDK 0.3.207; repository release inputs pin 0.85.1/0.3.286. This is not proof of the
historical executable. In installed `dist/acp-agent.js:2330–2337`, root `message.error` is
remembered as `lastAssistantError`; ordinary content conversion at 2395–2474 does not preserve
`isApiErrorMessage`. Terminal failure at 2043–2044/2079–2080 depends on subsequent
`message.is_error`; otherwise 2125 can settle normally. This supports the normalization hypothesis,
but does not establish the incident's final SDK result or wire branch. Do not repair or update the
global installation as part of this investigation.

## Review findings

### ARCH-QUOTA — interruption detection through same-work continuation (2026-11-07) — **Fix model:** medium — Codex Terra or Claude Opus.

Focused audit at `8d9cb623`; [report and probes](../archive/reviews/architecture/v0.22.0-next-quota.md)
and [receipt](../archive/reviews/architecture/v0.22.0-next.json). Existing NESTED-QUOTA-1 retains
the root-refusal detection owner; do not duplicate it. No product code or normative specs changed.

- **Must fix** — **ARCH-QUOTA-2** (fix complexity: medium).
  `internal/runtime/quota_check.go:253–269,441–442` and
  `internal/server/quota_check.go:32–43,82–103`: isolated checks do not establish the same effective
  model/account route as the interrupted process. Native-inherited Claude models are omitted when
  the probe disables settings; mutable routing files are reread but excluded from the selection
  fingerprint. A probe can report ready under another model or route and authorize the original
  conversation. FS-01.R49/R51, TS-04.R97 and TS-10.R46 require same-selection evidence (INV §2/§12). Pinned SDK
  settings resolution plus fake wire/store probes confirm both cases. Carry provable effective
  selection through the existing check boundary, or refuse when equivalence is unknown; revalidate
  it before continuation. Do not load work instructions or silently switch authentication.
- **Worth fixing** — **ARCH-QUOTA-4** (fix complexity: medium).
  `internal/server/quota_dispatch.go:16`, `quota_check.go:312–338`, and
  `task_dispatcher.go:41–52`: exact-session cleanup blocks the shared dispatcher before unrelated
  quota, task, pipeline and timed-start work. One stalled initialization delayed an independent
  due claim by 5.005 seconds; eight selected references can serialize roughly 40 seconds of timeout
  waits. TS-10.R47 and INV §16 require independent dispatch despite a failed cleanup. Keep bounded cleanup
  ownership/cooldown/cursor, but perform provider I/O outside the shared admission sweep. Verify an
  unrelated due record advances while cleanup is held. This is bounded delay, not permanent starvation.
- **Worth fixing** — **ARCH-QUOTA-5** (fix complexity: medium).
  `internal/server/quota_dispatch.go:106–108` and `quota_check.go:219–221`: a configuration read error
  after the durable claim silently returns, leaving an unstarted `claimed` row with no reason and
  no future due-query eligibility. Repairing configuration does not restore progress until restart
  or manual control. The store probe faults only that reread and confirms the orphaned claim.
  FS-01.R44, TS-10.R40/R43 and INV §7/§9 govern truthful failure/recovery. Explicitly defer or hold the exact
  unstarted claim on policy-read failure and test recovery without restarting. Trigger requires a
  transient read failure or malformed external config edit; ordinary atomic settings saves are safe.

### NESTED-QUOTA — root usage-limit refusal loses quota recovery (investigation 2026-11-07) — **Fix model:** medium — Codex Terra or Claude Opus.

- **Must fix** — **NESTED-QUOTA-1** (confirmed symptom; probable integration normalization cause;
  fix complexity: medium). `internal/runtime/quota.go:completePromptTurn`, Claude adapter
  native-error normalization and `rpcQuotaRefusal`: after a foreground child fails, the root's
  own synthetic `usage_limit_reached` refusal is rendered as assistant text and completes
  `end_turn` without quota evidence (native root line 456; Chuck seq 801–802 above). The requested
  evaluation stops without entering recovery. This is root provider evidence, not a request to
  promote arbitrary tool text to root quota. FS-01.R40/R55/R57, TS-04.R86/R105/R107 and INV §11
  govern the boundary. Trace/replay the actual native-to-ACP terminal mapping; preserve the
  provider-owned usage-limit signal through the supported adapter boundary and reuse ordinary
  quota settlement/reset scheduling. Verify root rejection even after child failure, ordinary
  assistant/tool lookalikes, transient throttling, ownership, and reset-plus-one-minute behavior.
  Do not indiscriminately reclassify every `rate_limit` or scrape stored conversations at runtime.
  No separate observability defect is established: retained native history resolves the error's
  origin; exact wire mapping remains a fix-validation prerequisite.

## Blocked on human

- **File reader full UI closure:** permission requested through the asynchronous question to
  scope the pre-existing withdrawal test assertion to its exact error message (one test line).
  The four requested product changes are committed and independently reviewable. No unrelated
  test correction has been made without approval (repository instruction on separate fixes).

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

Prepared notes: [v0.22.0](../archive/state/RELEASE-v0.22.0-notes.md).
Latest published notes: [v0.21.0](../archive/state/RELEASE-v0.21.0-notes.md).
