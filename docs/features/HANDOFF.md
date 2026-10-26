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
- **Review state:** no open review units; memory fix unit MEM has one open **Worth fixing**
  finding (MEM-4), blocked on a decision below; MEM-1/2/3/5/6 are fixed. The v0.20.0 architectural review
  ([receipt](../archive/reviews/architecture/v0.19.0-next.json)) is closed; ARCH-v020-1 fixed
  and verified. Provider-failure investigation unit PERR has four open findings below;
  terminal rate-limit failures are silently classified as successful turns, and ordinary new Codex
  chats permit an unused background title-model call. Product fixes are pending.
- **Checkout:** `main`, repository `AsaphNoam/Chuck`, pushed through the v0.20.0 release record.
  Clean release verification uses a detached worktree at `/private/tmp/chuck-release-<version>`.
  The primary checkout carries a pre-existing generated UI index edit; preserve it.
- **Future work:** direct-action migration remains paused on its transport blocker; passive
  quota snapshots remain TBD.
- **Checks:** unset `CHUCK_RUNTIME_ROOT`; use `GOFLAGS=-p=1` and a writable isolated Go cache.
  Native/device/credentialed limitations remain in the owed gates.

## Active change

None.

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

## Review findings

### PERR — Silent provider failures and request amplification investigation (2026-10-26) — **Fix model:** medium — Codex Terra or Claude Opus.

**Report, verbatim:** “Chuck accepted user messages and started Codex turns, but provider errors
left them unanswered. Four consecutive turns failed with ‘exceeded retry limit, last status: 429
Too Many Requests,’ with last_agent_message: null. Chuck recorded these as turn_end with
stop_reason: end_turn despite providerError. Check error visibility and recovery so failed turns
don’t appear silently completed, chuck might be spamming the providers, I'm getting 429s too
frequently. logs below:”

```json
{"timestamp":"2026-10-25T15:36:31.161Z","ordinal":65,"type":"event_msg","payload":{"type":"task_complete","turn_id":"01a173b4-cbf9-70e1-a31b-420b0b43ec05","last_agent_message":null,"error":{"message":"exceeded retry limit, last status: 429 Too Many Requests","codex_error_info":{"response_too_many_failed_attempts":{"http_status_code":429}}},"started_at":1792942590,"completed_at":1792942591,"duration_ms":168}}
```

Affected Chuck version, machine, provider source/configuration and full four-turn timeline were
not supplied. One native completion excerpt was supplied; no ACP capture or Chuck normalized
events accompanied it. Investigated macOS arm64 checkout `1d484440`, pinned Codex ACP 2.1.1 /
Codex 0.159.3. The local dashboard log had no matching turn id/error. The four findings below
classify as code defects (PERR-1/2), an observability/specification gap (PERR-3), and unnecessary
provider traffic with missing ordinary-chat policy (PERR-4).
The adapter also uses `providerError` as a file-change-report unavailability reason
(`dist/index.js:39539,39909`; `internal/runtime/filereport.go` records it). That reason alone
does not classify a turn outcome; the native error and AIR terminal failure are the evidence here.

**Follow-up report, verbatim:** “Check if chuck has some patterns that are likely to cause these
429 rejections, this doesn't happen to me when I use the codex app or cli”. Expanded read-only
audit compared launch/configuration, adapter-generated turns and automatic work admission. An
optional question about simultaneous agents and equal model/Fast settings remained unanswered
at the time of recording; the findings do not assume either answer.

- **Must fix** — PERR-1, confirmed: terminal non-quota provider errors disappear as normal
  completion. The pinned adapter's `sessionFailureKind` maps structured HTTP 429 to
  `rate_limited`, whose `SESSION_FAILURE_POLICY` is `category:limit`, `severity:error`,
  `actions:[retry]`; `terminalFailurePromptResponse` deliberately returns `stopReason:end_turn`
  plus that AIR failure (local pinned producer:
  `scripts/release/node_modules/@agentclientprotocol/codex-acp/dist/index.js:30062,30944,39993`).
  Chuck validates the failure but retains only its quota boolean/text, dropping severity/category
  (`internal/runtime/session_failure.go:29–35,100–106`). `mapPromptResult` and
  `completePromptTurn` override success only for account quota
  (`internal/runtime/acpmap.go:593–606`, `quota.go:112–170`). Thus a terminal 429, overload or
  other non-quota failure emits no error, ends as `end_turn`, and writes idle/Stop
  (`chat.go:2632–2648`). `ui/src/components/chat/turnActivity.ts:145–161` correctly labels
  `error` as Failed, but receives no such outcome or reason here. FS-03.R8/R74; INV §8/§11.
  Fix complexity: medium. Preserve terminal error semantics and bounded human-readable reason at
  the shared completion seam without treating active retry warnings, recovered revisions,
  cancellation or arbitrary 429 text as quota. Cover ordinary and activation turns with the
  existing fake peer; verify error event, error turn end, status and live/archive/phone rendering.

- **Must fix** — PERR-2, confirmed: the same terminal failure falsely confirms recovery and
  delivery. `internal/runtime/quota.go:175–198` computes `successful:true` and calls quota
  settlement; `internal/server/quota.go:16–45` then resolves a manual quota episode or confirms
  its claimed continuation rather than holding `non_quota_failure`. `validInlineReceipt`
  (`chat.go:1240–1262`) also accepts the raw `end_turn` despite terminal failure metadata, so
  inline mail can be marked read without a successful provider completion. FS-01.R44,
  TS-01.R32 and TS-04.R88; INV §15. Fix complexity: medium. Derive success/receipt decisions
  from the validated terminal outcome, including current failure revisions, rather than fixing
  only the displayed stop reason. In particular, changing `td.StopReason` alone leaves the
  raw-result inline-mail predicate wrong. Add actual-store regressions for a quota continuation
  that fails with a non-quota error and mail remaining unread; do not introduce an automatic
  retry of uncertain input. Durable recovery/mail consequences are code-proven; the reproduction
  below directly confirms the erroneous success flag, not a complete server recovery journey.

- **Worth fixing** — PERR-3, confirmed observability/specification gap: successful RPC envelopes
  containing terminal failure metadata produce neither a visible diagnostic nor a correlated
  server failure log. `completePromptTurn` (`quota.go:103–203`) has no such log; the existing
  `promptFailureMessage` diagnostic is entered only for RPC errors (`chat.go:1118–1131`), while
  the supported adapter returns this failure as an RPC result. TS-04.R85 specifies correlation
  only for failed RPCs, leaving this semantic-failure carrier uncovered; INV §8. Without agent/backend,
  generation/executing-turn, operation and bounded typed failure correlation, the supplied native
  turn cannot be reliably matched to a user send, held successor, activation or quota continuation.
  Fix complexity: trivial/easy. Extend the existing diagnostic seam/spec to terminal result
  failures with safe identifiers/category, never prompt text or raw provider data (TS-04.R12),
  and verify one diagnostic per terminal outcome. This would make the next frequency report
  attributable; it does not by itself measure the provider's internal HTTP retry count.

- **Worth fixing** — PERR-4, confirmed unnecessary provider traffic: ordinary new Codex chats
  permit the adapter's background title-model turn even though Chuck owns its displayed agent name
  and drops the corresponding native title update. The pinned adapter initializes new sessions
  with `sessionTitleSource:unset` (`dist/index.js:37755`); after the first successful prompt,
  `TitleGenerator.onTurnCompleted` starts a fire-and-forget ephemeral thread and `runTurn` on
  hardcoded `gpt-5.6-luna` (`:36785,36832–36888,39890–39897`). This is one additional model turn
  per eligible new conversation, can overlap the next user turn, and is not retried by that helper.
  Ordinary `Start`/`applySessionConfig`/`sessionNewParams` never explicitly names the native
  conversation (`internal/runtime/chat.go:472–540,3173–3210,3504–3537`); title-only
  `session_info_update` falls through the root mapper without a product consumer. By contrast,
  isolated quota checks already suppress title generation with a provider-local `/rename` and
  wait for the exact title update (`quota_check.go:295–322`). INV §16/§12; TS-04.R77's
  provider-dependency audit applies, but the ordinary-chat title-traffic contract is a spec gap.
  Fix complexity: trivial/easy. Extend the existing explicit-name seam to ordinary new Codex
  sessions before their first model prompt, or use another producer-verified suppression option;
  preserve native resume and failure visibility, and verify no extra model turn. Source proves
  eligibility for the extra request, not that its request contributed to the reported 429s.
  It cannot independently explain four successive failures in one already-established conversation.

**Request-volume conclusion:** undetermined field cause; no generic Chuck retry storm found.
Ordinary and activation paths each issue one `session/prompt` (`chat.go:1118,1841`), and
`transport.go:198–234` writes once without retry. An existing held user message can immediately
start a distinct turn after this failure (`chat.go:1280–1340`, `quota.go:201–202`); there is at
most one hold, and the supplied evidence does not show one. Transient 429 is explicitly not account
quota (FS-01.R40, TS-04.R86), so lack of a quota block alone is not a quota-contract defect.
Automatic quota recovery is intentional extra traffic: known reset plus 60 seconds or unknown-reset
hourly isolated checks, maximum six per chain (`internal/state/quota.go:19,55–58`,
`quota_checks.go:68–74,118–135`). Several chats can independently check the same provider; current
TS-10.R45 explicitly does not coalesce guessed accounts. Mail wake attempt ownership and ordinary
turn-end dispatch contain no generic failed-model replay (`messaging_loops.go:187–263`,
`task_dispatcher.go:563–578`). Quota probes already reject any terminal failure (`quota_check.go:330–355`).
Provider-internal retries, automatic title traffic, parallel conversations/account users and the
actual four request timestamps are unmeasured. The single 168 ms completion duration is not an
HTTP request count. Do not infer an account reset or add a guessed cooldown/retry loop from it.

**Expanded traffic/configuration comparison:**

**Strongest conditional risk: bursts across conversations.** Admission is per agent/work owner,
  not a shared provider request-rate gate. Tasks have eight start workers and default capacity 10
  (`internal/server/task_dispatcher.go:18–21,115–150`, `internal/config/seed.go:14`); separate mail
  and quota pools each have eight slots (`server.go:343–345`, `messaging_loops.go:54–80`). These
  are admission bounds, not proof of 24 simultaneous model requests or a per-account allowance.
  Task capacity excludes manually launched agents (`config/types.go:147–150`); running-recipient
  mail can start independent reasoning turns. No shared provider 429 cooldown/pacing seam was
  found. Several agents can therefore make requests and undergo native retries concurrently.
  The documented mail budget is 50 per turn and resets on every activation, with no cross-turn
  loop detector (FS-06.R12/R28 and §6). Reciprocal waking messages can sustain additional turns;
  no such exchange is established for this incident. These are documented behavior/risk patterns,
  not newly established violations or authorization to add a new scheduler/rate limiter.

**Recovery can synchronize work.** Multiple quota episodes with the same reset all become due at
  reset plus 60 seconds; unknown-reset probes are independent per conversation (one per hour,
  at most six each). Successful probes can immediately schedule their respective continuations
  (`state/quota_checks.go:118–135`). The existing limits bound this traffic, but do not pace all
  traffic for one provider/account. No evidence shows that a recovery wave preceded these 429s.

**Explicit model/config differs from native defaults.** Chuck applies its saved model, effort
  and requested Fast setting to the ACP session, and the adapter passes these to each turn
  (`chat.go:3504–3537`, adapter `:34338–34355,36721`). Backend model `fast:true` declares support;
  it does not prove a failed session enabled Fast. Local saved Codex selection is Installed,
  `gpt-6.1-sol`, default medium effort; personal TOML is the same model with xhigh/default tier.
  This does not support a claim that Chuck necessarily uses greater reasoning effort.

**Distinct client and process/profile.** Chuck uses ACP 2.1.1 → `CODEX_PATH app-server`, default
  identity `@agentclientprotocol/codex-acp`, `experimentalApi:true`, `requestAttestation:false`
  (adapter `:27299–27311,33598–33613`); direct app/CLI identity differs. Personal setup/auth is
  mirrored into private `CODEX_HOME=<chuck-home>/codex` before each child spawn, while runtime
  state/caches are separate (`config/codexprofile.go:15–52,98–110`, `server/launch.go:780`).
  No credentials were read. No evidence establishes different accounts, routing or server-side
  rate limits for that identity. Do not spoof a native client identity as a proposed fix.

**No increased retry setting found.** Chuck adds no request/stream retry-limit or websocket
  override. Safe local searches found no such override or provider/base-URL/header configuration
  in personal/mirrored TOML, nor any backend/model environment keys. Ambient dashboard environment
  may differ from the shell and was not proven identical. The release bundle pins Codex 0.159.3,
  while local saved Installed selection and all 89 inspected rollout metadata records use 0.161.0;
  do not blame a bundled-version mismatch without the failed session's actual selection.

**Evidence limit.** Local private-home native logs cover Oct 14–24; all 89 native rollouts and a
  sanitized exact-error scan of the 25,882-row log store contained no matching reported retry-limit/
  Too-Many-Requests error. A substring `429` search also matches random ids/cookies and is not
  rate-limit evidence. The supplied incident is Oct 25, so local saved settings are comparative
  evidence only. No model request or workload replay was performed. An equal-account/model/tier,
  one-agent comparison through ACP versus direct CLI, with sanitized request-id/status/error-code/
  retry-after correlation, remains the discriminating next receipt if failures persist.

Official references checked Oct 26: [Codex configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference)
documents native HTTP/stream retry settings; [rate-limit guidance](https://developers.openai.com/api/docs/guides/rate-limits)
describes burst/concurrent traffic and warns that unsuccessful API requests consume rate allowance.
This general guidance does not establish the exact ChatGPT-plan limit or cause of this incident.

**Evidence/verification:** `TestTerminalProviderRateLimitInvestigation` uses the existing fake ACP
and the independently inspected pinned terminal failure shape. Running it before marking it skipped
failed on all four observations: `end_turn`, absent transcript error, idle status and
`successful:true` recovery settlement. The fake emits partial text; this proves terminal error
handling even with output, not a credentialed reproduction of the field's null response.
The test is committed skipped for PERR-1/2; no product code/specification changes or live provider
requests were made. The shared settlement logic already existed at `43a565e7` (2026-10-11), after
the adapter pin at `19fb9a28` (2026-10-06); the first affected release is not independently proven.
Closure: focused runtime reproduction compiles/skips, `make check-specs`, and `git diff --check`.

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
