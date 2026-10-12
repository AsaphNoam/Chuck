# Chuck — Implementation handoff

**Live agent state.** Settled state through 2026-10-11 is archived in
[`HANDOFF-through-2026-10-11`](../archive/state/HANDOFF-through-2026-10-11.md).
Follow [`AGENT-WORKFLOW.md`](AGENT-WORKFLOW.md).

## Current position

- **Release:** `v0.14.0` tagged locally 2026-10-11 at the release commit, **unpushed**; pushing
  `main` and the tag needs explicit user authorization, then publication verification (workflow
  §16.6). `v0.13.0` was published 2026-10-10. The user waived the open review units below for
  release; that is not completed review. Manual provider gates below remain owed.
- **Repository:** `AsaphNoam/Chuck`; branch `main`.
- **Investigated — mobile CSS missing, fix open (2026-10-12):** dev-mode UI rebuilds can
  leave the phone static allowlist behind; the service worker permanently caches the resulting
  stylesheet error. Three confirmed defects and skipped reproductions are recorded under Review
  findings. The user's authorized dev restart completed; the tailnet is On and current CSS returns
  200 `text/css`. Phone cache recovery and on-device styling remain unverified; product code unchanged.
- **Closed mobile outage investigation — 2026-10-12:** the user confirmed that an unrelated
  terminal crash stopped Chuck. The separate embedded tailnet device going offline while the
  Mac stayed connected is expected. Investigation `e0489b5` is closed at the user's request;
  incidental timeout/diagnostic findings are withdrawn from the fix queue, not implemented,
  and the skipped reproduction test is removed. No product or runtime changes.
- **Awaiting review — mobile agent header, composer and model switch:** `c3b3c70`, `73cc75b`,
  `f15ee50`; FS-20.R43, TS-08.R111. New agent alone uses the centered PhoneSheet. The back
  chevron sits beside the persona eyebrow above the agent name; context usage sits between
  identity and tabs; Stop/Resume is a sibling of the tablist. The composer encloses a borderless
  textbox (no blue typing outline) and Send at bottom right. Model/effort is a right-aligned,
  bordered, raised button above the composer that opens the Switch runtime sheet. Final state:
  73 phone tests, 41 presentation/style checks, spec check and `make dist` pass; full UI suite
  790/791 with only the NewAgentModal flake below. `ui/scripts/phone-render.mjs --details-check`
  passes all three appearances at 360/390/430px; receipts `/tmp/mobile-model-20261011-*.png`.
- **Awaiting review — phone group memory and long-press agent actions:** `17b73ed`,
  FS-20.R47–R48/A19, TS-08.R121. Project-scoped page-session group choices and a long-press
  action sheet over shared phone management forms. Receipts: shared project resources
  `mobile-actions-20261011/`. Fixtures make no provider claim.
- **Reviewed — phone chat drafts and Theme selection, fix open:** `79cbbc3`,
  FS-20.R49–R50/A20–A21, TS-08.R122, TS-13.R25. The 2026-10-12 review reproduced accepted
  Send cleanup missing a composer reopened while delivery was pending (see Review findings).
  Theme selection and the narrow paired-phone config route passed review. This same unit remains
  open for the draft fix; the other available review units are unchanged.
- **Design awaiting scope confirmation — CLI parity:** `docs/ideas.md` proposes full nonvisual
  web parity with composable terminal input/output. No requirements or active change yet.
- **Design prepared/TBD — subscription quota views:** FS-09.R80–R81/A49–A50,
  TS-04.R89/TS-03.R61. Full passive paired snapshots remain TBD for Claude and Codex; evidence is
  in [`ideas.md`](../ideas.md). Not a ready unit. Do not implement a hidden prompt, command
  parser, login side client or new adapter patch.
- **Paused work:** `migrate-internal-actions-from-mcp.md` remains paused on its transport blocker.
- **Source-build Go tests:** run with `CHUCK_RUNTIME_ROOT` unset; an inherited installed bundle
  violates a credential test's source-build assumption.
- **Known flaky checks:** `internal/server` `TestOrdinaryStageAgentStopPausesPipelineRun`
  intermittently returns 409 “a resume is already in progress” (pre-existing at `74c8e84`).
  `ui` `NewAgentModal.test.tsx` fails 1–2 cases intermittently on unchanged `d3c9307`
  (observed 2026-10-11). Neither is investigated.

## Active change

None. The v0.14.0 release is cut; the next change starts from a review unit above or a new request.

## Acceptance gates still owed

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
- CLI parity product scope (see Current position).

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

### Mobile CSS missing in dev mode — investigation 2026-10-12 — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Report verbatim: “the css isn't loading on mobile”. Follow-up: “It's running on dev mode, not an
installed release, go ahead and try restarting”. Screenshot: Android phone app at 07:35, showing
the Chuck companion Home, Connected to your Mac and agent attention data, with default serif text,
buttons and oversized mark. Attachment `1-1000192461.jpg` was provided; no logs accompanied the
report. Exact running commit/browser version and whether reload helps were not supplied.
Classified as a code defect against FS-20.R10 and TS-08.R73/R111, TS-13.R14.

- **Must fix — confirmed** — Dev phone asset allowlist freezes before UI rebuild.
  `internal/server/spa.go:47,57–59` snapshots `phoneAssetGraph` when the handler starts, while
  `internal/server/static_dev.go:31–32` serves a changing `os.DirFS("ui/dist")`. Rebuilding Vite
  changes HTML and hashed CSS/JS names; the same running handler serves the new HTML but rejects
  its existing assets as 404 `remote_route_not_available`. This breaks the dev build's documented
  rebuild-without-restart behavior and TS-13.R14's embedded phone entry contract (INV §1/§10).
  `TestPhoneStylesheetAfterDevRebuild` failed with new HTML 200 and its CSS 404. Fix difficulty:
  trivial/easy; refresh the dev asset graph at the existing static seam, retaining the desktop
  asset exclusion. The frozen graph was introduced by `680775d` (2026-09-29).
- **Must fix — confirmed** — Failed stylesheet responses poison the persistent shell cache.
  `ui/public/remote-sw.js:5,32–42` caches every asset response without checking status or MIME
  type, then serves the cache hit without attempting the network. A 404 JSON CSS response remains
  unusable after the server recovers; restarting the Mac alone cannot repair the phone cache.
  This violates FS-20.R10 and the usable shell of TS-08.R73 (INV §1). Running the actual worker in
  `ui/scripts/remote-sw.test.mjs` reproduced the failure: a second CSS request returned the cached
  404 JSON, despite the network being ready to return valid CSS. The same test reproduced cached
  200 HTML at the CSS URL. Fix difficulty: trivial/easy; cache only valid assets and repair existing
  invalid cached responses, with recovery for already affected phones that preserves pairing.
  This unconditional cache behavior dates to the initial phone worker in `edf9d77`; the cache name
  changed at `a7157c1`, but validation/recovery was not added.
- **Worth fixing — confirmed** — Missing asset requests become successful app HTML.
  `internal/server/spa.go:60–62` applies SPA fallback to missing `/assets/*.css` and JavaScript
  files. After a rebuild removes old hashes, an asset request gets 200 `text/html`, which browsers
  cannot use as CSS and the worker caches. This hides broken delivery as success in remote request
  logs (TS-13.R14; INV §7). `TestPhoneMissingStylesheetDoesNotServeAppHTML` failed with 200
  `text/html` instead of 404. Fix difficulty: trivial/easy; return 404 for unavailable static asset
  URLs while preserving app-route fallback and the listener security boundary.

Field-cause confidence: **probable**, supported by the actual `~/.chuck/dashboard.log`:
2026-10-11 22:33:40 +03:00 recorded `/assets/remote-CJjGcYOz.css` 404 alongside rejected rebuilt
JS; 2026-10-12 07:21:46–49 recorded current `remote-BEZY269w.js` and `appearance-BqEL4BPf.js`
200, with no new request for the unchanged CSS hash before the 07:35 screenshot. Phone Cache
Storage was not directly inspected, so the field cache contents are not confirmed. Before and after
the authorized restart, a direct tailnet fetch of that CSS returned 200 `text/css`, 80,301 bytes.
The dev process was gracefully stopped and restarted using `go run -tags dev ./cmd/chuck dashboard
stop` then `dashboard start --detach`; the new PID was 60404 at loopback port 4317, and `/api/remote`
reported On with the existing Android pairing retained. No site data was cleared.

Verification: all four reproduction cases failed unskipped on existing product code, then were
marked skipped for fix. Focused static Go tests passed with two skips; the worker tests have two
skips. Commands: `env -u CHUCK_RUNTIME_ROOT go test ./internal/server -run
'TestPhoneStylesheetAfterDevRebuild|TestPhoneMissingStylesheetDoesNotServeAppHTML|TestStaticEntriesStayOnTheirListener'
-count=1`; `node --test ui/scripts/remote-sw.test.mjs`. Product code, specifications and generated
assets were not changed. Fix should unskip these tests, preserve desktop asset exclusion, verify an
affected cache recovers without deleting pairing cookies, and complete the applicable TS-06 closure.
Applicable invariant classes: §1/§7/§10/§14/§17; §13 selector coverage is not the cause (the current
built CSS includes phone styles). Other trigger classes have no applicable investigation surface.

### Phone chat drafts and Theme selection — `79cbbc3` — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Reviewed 2026-10-12; fix open.

- **Must fix** — Accepted Send leaves submitted text in a reopened composer (INV §1).
  `ui/src/remote/AgentScreen.tsx:207,413–423` restores the draft only on mount and clears local
  text only in the component that submitted it. Send, navigate away while delivery is pending,
  then reopen the same agent before acceptance: the reopened composer restores the submitted
  draft, but acceptance clears only storage and the old unmounted component. The visible text
  remains available to send again. This violates FS-03.R36 and FS-20.R49/A20's accepted-draft
  cleanup. Synchronize the mounted composer with accepted cleanup through the shared draft seam,
  preserving revision protection for newer text. Add a delayed Send/Steer leave/reopen/accept
  regression case. Independent Chromium reproduction retained `submitted before navigation`
  in the textbox after the storage entry became absent; evidence:
  `/tmp/review-phone-boundary-20261012.mjs` and `/tmp/review-phone-boundary-20261012.png`.

Review evidence: 66 focused UI tests passed (AgentScreen, PhoneSettings, PhoneApp and drafts);
the two focused remote appearance/field-filter server tests passed. Existing PhoneApp fixtures
emit missing `/api/roles` MSW warnings without failing. The working tree's actual remote entry
passed `phone-render.mjs --parity-check` with isolated Chromium at 360/390/430px in all three
appearances; inspected screenshots show a labelled, readable Theme control, adequate touch height
and no horizontal overflow. Receipts: `/tmp/review-phone-parity-20261012*.png`. Fixtures make no
live-provider or real-tailnet claim. Shared draft/config reuse, partial config merge, finite skin
selection and query ownership are sound local choices; none needs a human decision.

Invariant sweep: §1 has the finding above. Applicable §2, §3, §8, §10, §11, §13, §14, §15,
§16 and §17 have no additional findings. §4, §5, §6, §7, §9 and §12 have no applicable surface
in this unit. Product code and specifications were unchanged.

The mobile outage investigation was closed by the user on 2026-10-12 (see Current position).
Closed 2026-10-11 reviews are in the archive above.

## Blocked on human

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

See [`RELEASE-v0.14.0-notes.md`](../archive/state/RELEASE-v0.14.0-notes.md) (unpublished until
pushed) and [`RELEASE-v0.13.0-notes.md`](../archive/state/RELEASE-v0.13.0-notes.md).
