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
- **Review state:** no open review units; memory investigation MEM has six open findings
  (MEM-1–MEM-6), grouped as one fix unit below. The v0.20.0 architectural review
  ([receipt](../archive/reviews/architecture/v0.19.0-next.json)) is closed; ARCH-v020-1 fixed
  and verified.
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

### MEM — Memory retention investigation (2026-10-26) — **Fix model:** medium — Codex Terra or Claude Opus.

**Report, verbatim:** “I have a suspicion that Chuck is leaking memory, do a thorough
investigation and see if you can find anything”. No workload, measured growth, affected process,
version, or diagnostic log was supplied. Investigated checkout `43c214d1` (v0.20.0 plus release
records), macOS arm64. This establishes code defects, not which defect caused the reported suspicion.
The optional question about the affected process/workload had no answer at closure.

- **Must fix** — MEM-1, confirmed: MCP sessions outlive their revoked launch registration.
  `internal/messaging/messaging.go:324–331` creates the stateful SDK HTTP handler with nil options;
  SDK v1.8.0 explicitly gives zero `SessionTimeout` an unlimited idle lifetime. `Revoke` at
  `:352–356` deletes only Chuck's token mapping. A client exit, Stop, or reconnect that does not send
  MCP DELETE leaves its SDK session and reader goroutine rooted indefinitely. A disposable probe
  through the production handler initialized 100 sessions, completed all HTTP requests and revoked
  all tokens: 100 sessions and 100 additional goroutines remained after GC. Explicit MCP DELETE
  returned both counts to zero. TS-04.R6–R7; INV §4/§16. Bind transport cleanup to the exact launch
  identity and bound orphan/idle sessions; preserve other launches and active tool calls. Fix
  complexity: medium. `internal/messaging/session_teardown_investigation_test.go` reproduces one
  retained session after Revoke; proved failing enabled, committed skipped. Handler construction and
  Revoke date to `fe1f1b6a`; current SDK adoption is `7c95fa91`; prior SDK behavior was not tested.

- **Must fix** — MEM-2, confirmed: closed archive readers retain full transcripts.
  `ui/src/features/archive/ArchiveAgentPage.tsx:47–64` installs each full history in
  `useTranscriptStore.byAgent`; unmount only changes a local flag. It does not join the existing
  reference-counted reader cleanup in `ui/src/api/sse.ts:140–157`, and a fetch resolving after
  unmount still calls `setTranscript`. Browsing distinct archived conversations accumulates full
  histories for the tab lifetime. Ten mounted/unmounted production archive pages retained ten
  1,000,000-character histories (10,000,000 payload characters). INV §4/§16. Use owner-aware release
  and stale-fetch fencing, preserving any other reader of the same conversation. Fix complexity:
  medium. Added a skipped production-page unmount regression to
  `ui/src/features/archive/ArchiveAgentPage.test.tsx`; it fails enabled. Missing teardown predates
  the current file tabs; `fbdbd1bf` added the flag without fencing the store write.

- **Must fix** — MEM-3, confirmed: Stop during a turn strands the search-index accumulator.
  `internal/index/indexer.go:200,425–430` appends searchable text until a turn/content flush.
  `internal/runtime/chat.go:1123–1126,2136–2144` bypasses turn completion on intentional Stop and
  closes persistence without flushing/releasing the process-wide indexer's pending content.
  Runtime ownership ends correctly, but the last turn's text remains keyed by the stopped agent;
  archiving does not release it. Five distinct real fake-ACP Start → prompt → stream → Stop
  lifecycles retained 5,242,915 bytes across five stopped agents. TS-02.R16; INV §4/§16. Flush or
  safely release at the final generation boundary without losing durable search coverage or racing
  a successor generation. Fix complexity: medium. The skipped
  `internal/index/stop_retention_investigation_test.go` waits for writer closure and asserts zero
  retained text; enabled it fails with 65,543 bytes retained. Normal turn completion/crash paths
  do flush. The accumulator design dates to `5f33a075`; exact first affected release is unproven.

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
  accumulation existed earlier; no exact regression release is asserted.

- **Worth fixing** — MEM-5, confirmed: text-draft eviction leaves full text in a global cache.
  `ui/src/components/chat/drafts.ts:19,56–57,90–94` stores text in module-global `live` without
  joining the 20-draft eviction in `draftIndex.ts:37–51`. Creating drafts for more agent IDs
  releases persisted entries but retains their text for the tab lifetime; restoring a held message
  can resurrect discarded text. Production draft APIs reproduce this with 25 IDs: the oldest
  stored draft is absent, yet restoring “withdrawn” yields “withdrawn\n\nlarge-original-0”.
  FS-03.R36; TS-01.R17; INV §16. Integrate live-text eviction with the existing index cleanup,
  preserving in-flight send revision fences. Fix complexity: trivial/easy. Introduced by `0c081037`.

- **Worth fixing** — MEM-6, confirmed: ended child reasoning spans escape child retention limits.
  `internal/runtime/activity.go:53–66` retains one open span per activity.
  `internal/runtime/subagent.go:178–183,195–205` retires completed children and caps their tombstones
  but never clears their reasoning spans. The lifecycle update closes the parent's span or returns
  through the child fast path, so a child whose last ordinary output was reasoning leaves its own
  entry behind. Production notification dispatch followed by root `finishTurn` retained 500 spans
  after 500 completed children while the child map correctly retained only 64 tombstones.
  TS-04.R63; INV §4/§16. Clear child span ownership at terminal/eviction boundaries, retaining late
  output attribution. Fix complexity: trivial/easy. Small per-child metadata, much lower impact
  than full transcript/text retention. Child-slot cleanup is `65de12ce`; no native-provider run.

**Verification and limits:** Three focused regressions were demonstrated failing, then left skipped
as §12 requires; no product code or specifications changed. Archive suite: 8 passed, 1 skipped;
MCP and Stop regression selections pass with their intended skips. Disposable probes live under
`/private/tmp/chuck-memory-probe`, `/private/tmp/chuck-memory-ui`, and
`/private/tmp/chuck-memory-*-investigation_test.go`; they are supporting local evidence, not committed
test dependencies. Go probes unset `CHUCK_RUNTIME_ROOT`, use `GOFLAGS=-p=1` and
`GOCACHE=/private/tmp/chuck-memory-gocache`; the UI probe has two passing cases. Source review found
bounded bus/subscriber queues, proper ordinary SSE unsubscribe, live-chat last-reader cleanup,
short-lived reconciliation tails, journal reclamation, bounded reasoning text/tool inputs/child
tombstones, and file object-URL/terminal listener disposal. These are inspected paths, not an
exhaustive proof of absence of leaks. Disk transcript/log growth is not itself a RAM leak.

No normal Chuck server was identified running against `~/.chuck`; its log ends with shutdown on
2026-10-24. Three long-lived September disposable AgentDeck servers were observed at about
15–16 MiB resident each; preserved, not restarted or stopped. A single snapshot does not establish
a growth rate. No production heap/browser snapshot or credentialed provider session was captured,
and source has no built-in heap/goroutine profiling surface; attributing the user's actual growth
remains undetermined. Existing generated UI index edit was preserved.

## Changelog

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
