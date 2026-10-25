# Chuck — Handoff archive through 2026-10-25 (v0.20.0)

Settled state moved out of the live handoff when v0.20.0 was released. Non-authoritative history;
see the live handoff and specifications for current state.

## Superseded current position (v0.19.0)

- **Published:** [v0.19.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.19.0),
  2026-10-24, tag at `308ad23f` (timed and recurring task starts). Release CI 40104625973 and
  main CI 40104624461 passed; downloaded archive size and SHA256 match the manifest.
  [Notes](../archive/state/RELEASE-v0.19.0-notes.md).
- **Review state:** the v0.19.0 architecture review
  ([receipt](../archive/reviews/architecture/v0.18.1-next.json)) is closed: ARCH-v019-1/2
  fixed and verified. Project-scoped groups/selectors review is closed; PSG-R1/2 repaired.
  Saved file-viewer tabs review FVT-R1–R4 is closed with verified fixes.
  Skip-permissions default review is closed: PDEF-R1 repaired the Settings policy write and
  refreshed the saved default for new launches.
  Annotation font/focus review is closed: AIF-R1 reveals a collapsed tray on capture and focuses
  the new instruction. Workflow verification cleanup shipped unreviewed at the operator's
  choice; its review is now closed with no findings.
  The v0.20.0 architectural review is complete at `bc9cc82e`; ARCH-v020-1 is fixed and verified
  ([receipt](../archive/reviews/architecture/v0.19.0-next.json)).
- **Checkout:** `main`, repository `AsaphNoam/Chuck`, pushed through `308ad23f`. Clean release
  verification uses a detached worktree at `/private/tmp/chuck-release-<version>`.
- **Future work:** saved file-viewer tabs are implemented and the FVT-R1–R4 review is closed.
  Direct-action migration remains paused on its transport blocker; passive quota
  snapshots remain TBD.
- **Checks:** unset `CHUCK_RUNTIME_ROOT`; use `GOFLAGS=-p=1` and a writable isolated Go cache.
  Native/device/credentialed limitations remain in the owed gates.

## Superseded active change (v0.20.0 release preparation)

- Release v0.20.0: version and push authorized by the operator's keep-going instruction.
  Prepared candidate `bc9cc82e`; independent architectural review complete.
  ARCH-v020-1 fixed by `92930083`, embedded by `b8034e83`; exact closure is recorded in
  [the existing receipt](../archive/reviews/architecture/v0.19.0-next.json), with no review rerun.
- Verification: focused regressions fail before/pass after; full UI 103 files/1055 tests,
  untagged and `sqlite_fts5` Go suites and final `make dist VERSION=0.20.0` pass.
  Earlier overloaded Go runs failed timing gates and were stopped; no pass is inferred from them.
  Final logs: `/private/tmp/chuck-v020-{ui-final,dist-final,go-final}.log`.
  Environment: unset `CHUCK_RUNTIME_ROOT`, `GOFLAGS=-p=1`,
  `GOCACHE=/private/tmp/chuck-pdef-fix-cache`.
- `check-architectural-review.mjs --release-tag v0.20.0`: ready at `a774d128`.
  Annotated tag `v0.20.0` created locally at `a774d128`; **unpushed**. The agent's push of
  `main` (28 commits ahead of `origin/main`) and the tag was blocked by the session permission
  classifier — needs the operator to push or allow it.
- Preserve the primary checkout's pre-existing generated UI index edit.
  Next: push `main` then `v0.20.0`, verify release CI and attached archive/checksum/manifest,
  archive settled handoff and record release.

## Settled changelog

- **2026-10-25 — ARCH-v020-1 fixed (INV §8/§10/§17):** empty conversation/room/phone Files owners show the existing saved-tab refusal/recovery notice. Existing populated/focused readers keep their notice placement. Both regressions fail against pre-fix product files; focused 38-test and full 1055-test UI suites pass, as does final distribution build. Original architectural review reused with exact repair/embed commits and focused evidence in its receipt. Full release Go verification remains pending only for the FTS5 variant.

- **2026-10-25 — Architectural review for v0.20.0:** reviewed committed preparation `bc9cc82e`
  across saved work, execution, admission, supervision, information, control, bounds and installed
  artifacts. ARCH-v020-1 confirmed missing visible saved-tab refusal/recovery on empty owners;
  one localized fix unit recorded. Existing native/provider/device gates retained. Codec/limiter
  probe passed; UI/style/build knowledge evidence reused. Go failure/build delay stays open with
  the release owner; no full-suite/distribution pass is inferred.

- **2026-10-25 — PDEF-R1 fixed (INV §10 wiring; §§3/4/7/11/17 checked):** the configuration API
  merges optional `skip_permissions`; new launches read the saved global policy before resource
  registration. Both toggle directions, omission preservation, response/disk values and actual
  next-launch policy are covered by `TestPutConfigSkipPermissionsRoundTripsAndPreservesOmission`.
  The regression failed before the handler repair and again with the stale launch snapshot;
  focused manual approval/denial and frozen resume/switch tests pass. Existing manual-policy
  fixtures now save their explicit policy. Final `make test` (no-tag and FTS5 suites), `make build`
  and diff checks pass for this scope with `CHUCK_RUNTIME_ROOT` unset, `GOFLAGS=-p=1`,
  `GOCACHE=/private/tmp/chuck-pdef-fix-cache` and approved loopback access. Logs:
  `/private/tmp/chuck-pdef-fix-{test,build}.log`. No new native/provider claim. Unit closed;
  unrelated generated UI index edit preserved.

- **2026-10-25 — AIF-R1 fixed (INV §10 wiring; §§8/13/17 checked):** adding a captured draft
  clears only its source's collapsed flag in the same store update. A frame after the new row mounts,
  focus returns to its instruction if focus is unchanged or fell to `body` while the docked stage
  reports its new expanded state; it does not override an intentional focus elsewhere. The component regression
  preserves another source's collapse setting. Corrected Chromium journey captures a real transcript
  event from the context menu, asserts visible/focused before typing, then uses `page.keyboard.type`
  without locator focus; it passes at `/private/tmp/chuck-annotation-collapsed-fix3.log`. The earlier
  `/private/tmp/chuck-annotation-collapsed-fix.log` is superseded because `pressSequentially` focused
  its locator and did not prove capture focus. Pre-fix reproduction:
  `/private/tmp/chuck-annotation-collapsed-review.log`. Focused test, `make check-specs`, full UI
  suite (102 files/1053 tests), `npm run build`, and `git diff --check` pass. The first final UI run
  had two CardGrid composer-focus failures; that unchanged file passes 41/41 on focused rerun and
  the full suite passes on retry. Logs: `/private/tmp/aif-npm-test-final.log`,
  `/private/tmp/aif-npm-test-retry.log`, `/private/tmp/aif-check-specs-final.log`, and
  `/private/tmp/aif-npm-build-final.log`. Native macOS keyboard focus remains unverified; browser
  API/SSE fixtures do not claim live-provider behavior. Unit closed after parent review.
  Unrelated generated UI index edit preserved.

- **2026-10-25 — Workflow verification cleanup reviewed (`2fd50115`):** no findings; unit
  closed. **Fix model:** none required (no open fixes). The scoped completion matrix and
  recorded-check reuse match TS-06.R5; release still requires full Go/UI evidence and a fresh
  distribution build. Both release launchers agree. CI and Makefile paths reach presentation
  validation through `prebuild`; focused tests run Vitest directly as TS-08.R17 requires.
  Fresh review checks at `c2391868` on this unit's unchanged tooling: `make check-specs` passes;
  `cd ui && npm run check:styles` passes Stylelint, 47 script tests and the presentation audit;
  `npm test -- src/presentation/VisualMatrix.test.tsx --reporter=dot` passes 8/8. Logs:
  `/private/tmp/chuck-workflow-review-spec.log` and `/private/tmp/chuck-workflow-review-focused.log`.
  The new gate regression fails against the pre-change checker for its obsolete `pretest`
  requirement (`/private/tmp/agentdeck-before-proof.dDx39N/result.log`), establishing defect
  discrimination under TS-06.R35/INV §17. Invariant sweep: §§10/17 apply and pass; no applicable
  changed surface for §§1–9/11–16. No local-choice decision or rendered-product verification
  was needed. Full product/build results from implementation are not claimed as fresh review
  runs. Concurrent permission-fix edits and the unrelated generated UI index are preserved.

- **2026-10-25 — Annotation input font and focus reviewed:** one Must-fix finding, AIF-R1,
  closed above. Expanded agent/room typing and computed typography pass independent Chromium
  checks in all three appearances; native macOS keyboard focus remains unverified.

- **2026-10-25 — Skip-permissions default reviewed:** one Must-fix finding, PDEF-R1: the
  documented Settings opt-out is silently ignored by the API. A real-handler reproduction
  confirms the unchanged saved/effective policy. Seed inheritance, saved preferences and frozen
  sessions otherwise match the requirements. Same review unit remains open for the localized fix.

- **2026-10-25 — Saved file-viewer tabs fixes (INV §1/§2/§10):** FVT-R1–R4 are closed.
  Delayed normalization preserves descriptors without rewriting Back on conversation and room
  routes; cited URLs replay their jump on remount and Forward; phone nested links preserve the
  citation under tracked-file validation; tab focus survives ArrowRight → Enter → ArrowLeft.
  Old-behavior regressions fail for R1 conversation/room (`/tmp/fvt-r1-old2.log`), R2 remount/
  Forward (`/tmp/fvt-r2-old.log`), R3 phone (`/tmp/fvt-r3-old.log`) and R4 Chromium
  (`/tmp/fvt-r4-old/report.json`). Final focused UI tests pass 67/67
  (`/tmp/fvt-final-focused.log`); full UI suite passes 102 files / 1052 tests
  (`/tmp/fvt-final-ui-tests.log`); build, including Stylelint and presentation checks, passes
  (`/tmp/fvt-final-ui-build.log`); `make check-specs` passes (`/tmp/fvt-final-specs.log`);
  Chromium main journey passes 24/24 (`/tmp/fvt-browser-final2.log`). Physical-phone touch and
  reconnect, non-Chromium engines, and rendered 500-total/2 MiB/storage-failure gates remain owed.
  Pre-existing embedded index edit preserved; no generated embed.

- **2026-10-25 — Project-scoped groups/selectors fixes (INV §1/§9/§10):** PSG-R1 flushes
  pending layout on navigation and orders saves before reopening reads; PSG-R2 resets outside
  annotation recipients for fresh agent/room batches without remounting delivery ownership.
  Controlled regressions failed before repairs and pass after; full UI suite passes 102 files /
  1048 tests (`/tmp/psg-ui-test.log`); UI build/style/presentation checks, `make check-specs`
  and diff checks pass (`/tmp/psg-ui-build.log`, `/tmp/psg-specs-final.log`). Browser/device
  gates above remain open. Existing dirty
  embedded index preserved; generated embed is not part of this closure.

- **2026-10-25 — Saved file-viewer tabs implemented:** desktop conversations (live/archive,
  Focus file), rooms and phone Files share one tabbed reader with browser-saved tabs.

- **2026-10-25 — Project-scoped groups/selectors reviewed:** two Must-fix findings: collapse
  persistence on rapid navigation and annotation scope reset on a fresh batch. Independent
  26-check browser journey passes; targeted browser reproductions confirm the gaps. Room,
  desktop and phone rendering checked in Core and Sky & Grove. Same unit remains open for fixes.

- **2026-10-24 — Default permissions:** fresh installs now skip permission prompts for No
  persona and all seeded personas through the inherited global policy. Saved settings and
  explicit overrides survive. Both Go variants, build, spec and diff checks pass; ready for review.

- **2026-10-24 — v0.19.0 published:** the operator approved the version and asked to release;
  `main` and tag `v0.19.0` at `308ad23f` were pushed (23 commits). Release and main CI passed;
  archive checksum/size match. The shipped operator guidance covers start times and schedules.
  Settled history moved to the 2026-10-24 archive. Manual gates remain owed.

- **2026-10-24 — Project-scoped groups and selectors:** typed group names are kept, group
  suggestions and collapse follow the project, and annotation/room pickers start at the current
  project with Include other projects. Browser journey, UI and Go suites pass; ready for review.
- **2026-10-24 — Design ready:** saved file-viewer tabs
  are waiting to start: FS-03.R101–R106/A80–A82, FS-20.R56/A28, FS-21.R59/A42 and
  TS-08.R140–R145. The operator confirmed browser/device persistence, existing phone access/screens,
  desktop rooms, 50/500 caps without eviction and Back hiding without deleting tabs. No product
  code changed. Spec lint, design-feature twin comparison and `git diff --check` pass; rendered
  and actual-device checks are implementation gates, not claimed by this design-only unit.
- **2026-10-23 — Workflow verification cleanup:** scoped local checks, removed duplicate UI
  style runs, allowed reuse of unchanged passing evidence, and clarified one completion pass per
  fix unit plus compact test logs. Focused command/build and spec/launcher checks pass; waiting
  for review. Existing product findings and manual gates are unchanged.

- **2026-10-23 — Design in progress:** file-viewer tabs recorded under Ideas being defined.
  The requested open/close/reorder behavior needs scope confirmation for duplicate files,
  reader lifetime and supported surfaces before technical design. No product code changed.

