# Chuck — acceptance gates still owed

Live list of native, provider, device and hosted checks that have not been run. One entry per
gate; remove an entry when its receipt exists. A gate that was run and failed is a finding in
[HANDOFF.md](HANDOFF.md#review-findings), not an entry here (workflow §4).


- **Quota snapshot (FS-09.A49/A52):** one bounded credentialed read each on packaged Claude and
  Codex, compared with the provider's own usage display (window identity, percentage, reset
  instant). Real `get_usage` with `skipBehaviors` and real `account/rateLimits/read` shapes are
  unverified. Rendered 2026-11-12 in independent Chromium (working-tree UI, real binary, isolated
  home, fake ACP): the complete state at 1024px, 1440px and the 390px phone page in Core and
  Sky & Grove showed no overflow, a visible focus ring, keyboard opening and no request on
  open. Stale, partial and unavailable states were not rendered. The phone page was loaded
  without real pairing. After QUOTA-FORK-7 the phone summary measured 390×44 in Chromium
  (isolated markup with the real stylesheets, not the full phone page). The Claude adapter now
  forwards only window members; the `model_scoped` decode is fixture-verified only.
- **Fork from a message (FS-01.A40):** one bounded credentialed fork each on packaged Claude and
  Codex proving the new provider context ends at the chosen turn; Codex must also accept the
  proof-only `session/load` of its fork. Rendered 2026-11-12 in the same run at 1280px in Core:
  enabled and disabled Edit and Fork from here, both confirmations, navigation, Edit's unsent
  text and the unchanged source passed. Keyboard-only menu use, Sky & Grove and the dashboard
  pane were not rendered. Re-render the disabled Edit on a message sent mid-turn and a copied
  turn's reason (QUOTA-FORK-2/4 are component/API-tested only).
- **Mail detail (FS-06.A33):** independent Chromium used the built UI, an isolated Go server,
  fake ACP, real MCP send/read and fixture tool-call events. The unfinished send disclosures
  rendered Markdown and a refusal at 1024px in Core. Keyboard expansion and Markdown layout
  also passed at 1024px/1600px in Sky & Grove.
  The authenticated 390px phone view expanded received and read mail. Received bodies remained
  available after mailbox expiry and archive reload. The completion detachment is repaired and
  covered by projection tests (2026-11-13). Re-render completed, archived and phone activity
  end to end. The physical paired-phone check, including tapping the 44px Mail button, remains
  owed.

- **Think Tank templates (FS-21.A43–A45, TS-14.R33):** independent Chromium on the built
  desktop bundle and isolated fake-ACP server passed load, unavailable-persona repair, long-row
  layout, declined replacement, room instruction disclosure and room save dialog at 1024px/1600px
  in all four appearances. Core also passed keyboard focus restoration, draft save without a goal,
  active/ended room save and increased-ceiling reload. A saved existing-chat row launched as a
  fresh destination chat and the source template remained intact.
  Chromium reproduced TT-TEMPLATE-3 and TT-TEMPLATE-4. The state tests cover source deletion.
  Recheck the affected rendered paths after repair and exercise A45 with fake-provider turns
  through retry, closing and synthesis plus the SSE check in TT-TEMPLATE-1. The focused Go suites
  and 78 component tests passed.

- **File freshness/editing (FS-03.A86–A88):** actual Chromium desktop live/archive/room/Focus
  journeys and the real phone bundle at 360/390/430px pass. Phone SSE/tracked-file admission was
  fixture-controlled; physical-device, real paired reconnect and non-Chromium gates remain open.

- **FS-03.A85 native default browser:** JXA lookup ran but this environment has no configured
  default web browser. On a normal Mac, right-click a self-contained HTML file and select
  **Open in browser**; confirm the original file opens in the default HTTPS browser (even when
  `.html` is associated with an editor). HTTP dispatch, retained sources, remote refusal, native
  argv/timeout/error handling and in-app live/archive/room rendering are automated evidence only.

- **GOAL-LED evidence (FS-14 §6):** A61 run-start-to-notification browser journey; A64 live
  desktop/phone composer and attachment restart checks. The paused chat answer with attachment
  reached fake ACP and durable history. The original field report of an approval refusal remains
  unidentified. A67 fake-provider invocation-reason capture; A68
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
  pinned formatter source; fake-provider/browser proof is not live delivery evidence. Include a
  Think Tank turn that reaches quota after the operator requests End.
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

