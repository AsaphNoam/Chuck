# Chuck — Implementation handoff

**Live agent state.** Settled state through 2026-10-14 is archived in
[`HANDOFF-through-2026-10-14`](../archive/state/HANDOFF-through-2026-10-14.md).
Follow [`AGENT-WORKFLOW.md`](AGENT-WORKFLOW.md).

## Current position

- **Open investigation — model/provider continuity (2026-10-14):** diagnosed gaps are grouped under
  **Review findings** below. Product code and specifications are unchanged; six failing reproduction
  assertions are retained skipped. This record was originally local-only and was absent from the
  shared fix queue and v0.16.0 release gate. Next: `/fix` this investigation unit.
- **Release:** `v0.16.0` published 2026-10-14 (Latest): release CI run 38593397312 passed and
  the GitHub Release carries the archive, `install.sh` and `manifest.json`; body set from
  [`RELEASE-v0.16.0-notes.md`](../archive/state/RELEASE-v0.16.0-notes.md). Manual provider gates
  below remain owed.
- **Main CI red on UI flakes:** CI runs 38514607452 and 38593397065 on `main` failed only in the
  UI suite: `NewAgentModal` (1–2 cases) and the phone `AgentScreen` successive-gap test, all
  listed as known flaky below. Every `main` push now risks a red CI until they are investigated. Range `v0.15.1..v0.16.0`: the shipped
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
  progress”. `ui` `NewAgentModal.test.tsx` fails 1–2 cases intermittently. `FileViewer.test.tsx`
  “scrolls to and marks a cited line” failed once in the full suite and passes alone. The phone
  successive-gap test can time out when the UI suite runs alongside browser load.
  `internal/server` `TestQuotaDispatcherContinuesSameConversationOnce/stopped` failed once
  (“reloaded transcript differs”, FTS5 variant) in the `v0.16.0` closure run; 40 isolated runs,
  30 `-race` runs and a full FTS5 package rerun pass. Range commit `9989fb88` touched ACP update
  routing; whether it relates is uninvestigated.

## Active change

None.

## Acceptance gates still owed

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

### Investigation — model/provider switching and conversation continuity (2026-10-14) — **Fix model:** difficult — Codex Sol.

**Report (verbatim):** “Chuck supports switching models and provider mid agent chat, but I believe
there are serious gaps in how well it actually does this and maintains all the relevant agent
context, history, etc. figure out what the gaps are and in which cases they affect (continuing an
agent that was stopped as well? Just switching providers or models as well?)”

**Evidence scope:** No user logs, affected version, or provider/environment details were supplied.
Investigated `main` at `d94320969dcccc57f9683a1f9b6acad3ffbade99`, following publication of v0.15.1,
in an isolated Linux source checkout without provider credentials. Two independent read-only traces
covered native providers and UI/configuration. Root verified primer and runtime paths, ran six
failing assertions through temporary unskipped overlays, and inspected provider source. Exact npm
lockfile versions/integrities were checked for Claude ACP 0.85.1, Claude SDK 0.3.286, Codex ACP 2.1.1;
Codex source was read at `rust-v0.159.3` (`01fc69f4026735edfdf6789820549727a4867b11`). OpenCode source
was current upstream `34d8ded0c58dd66808c1d60e081e8c66da12a896`, not a Chuck pin or proof of the
operator's installed version. No live inference was attempted.

- **Must fix** — **C1, confirmed code defect; fix complexity trivial/easy.** The primer discards
  every accepted human prompt: `internal/server/primer.go:117` handles assistant/tool/diff/end events
  but never `runtime.EvUserPrompt`. A short conversation such as “Use staging; never deploy” /
  “Understood” transfers only “Understood.” Goals, constraints, corrections, and task inputs therefore
  disappear on every cross-backend handoff, including different backend entries using the same CLI
  type, and any future same-backend model switch assigned primer handoff. Ordinary successful native
  resume does not use this builder. Violates FS-01.R13/R14 and INV §2/§11. Reproduced by
  `primer_continuity_investigation_test.go` / `accepted user instructions`. Fix through the existing
  canonical transcript projection, preserving human text with its role and turn boundary.
- **Must fix** — **C2, confirmed code defect; fix complexity trivial/easy.** The same alternate
  projection drops tool arguments/output and diff contents, retaining only title/status/path
  (`primer.go:127–151`); it also trims and prefixes each streamed assistant delta independently
  (`:118–123`). Even a tiny transcript within budget loses observed facts and edits, and
  `staging-` / `only.json` becomes separate framed strings rather than `staging-only.json`. “Verbatim”
  tail text is therefore inaccurate. Same primer-path scope as C1. FS-01.R14, TS-01.R22,
  INV §2/§11. Reproduced by the tool-evidence and streamed-text subtests. Reuse typed text projection
  and fold deltas before framing; preserve bounded relevant tool evidence rather than labels alone.
- **Must fix** — **C3, confirmed code defect/coverage gap; fix complexity medium.** Production's
  `defaultPrimerSummarizer` always errors (`primer.go:29–30`; installed at `server.go:339`), so older
  context is an earliest-text prefix, not a semantic summary. The final prefix truncation (`:69–83`,
  `:158–171`) spends the shared budget on older text before the advertised last six turns and can
  drop the newest turn entirely. Reproduced with the default 8,000-token estimate and long turns;
  the latest “migration failed; rollback next” is absent. History loss caused by a bounded handoff
  is expected, but retaining stale history at the expense of current state and calling it a summary
  needs an explicit corrected contract. FS-01.R14, INV §7/§16. Reserve recent context first; specify
  deterministic older-context selection, honest truncation/summary labels, UTF-8-safe clipping and
  the heuristic versus real-token bound. Stream input at the existing transcript reader seam rather
  than retaining the whole unbounded file before truncating.
- **Must fix** — **C4, confirmed code defect and observability gap; fix complexity medium.** Generic
  `session/load` RPC rejection falls through to `session/new` (`internal/runtime/chat.go:1442–1474`),
  with no transcript primer on ordinary Resume/wake. It returns a normal successful Resume while
  replacing the native ID; only a server warning describes history loss. The old visible Chuck
  transcript remains, and the new empty context reports the old token usage (`:1364`, `:1515–1518`).
  A missing/unloadable native rollout can therefore affect stopped-agent Resume, prompt/mail wake,
  server-restart continuation, automatic continuation that resumes, and same-backend model/interface/
  effort switches. A same-backend switch can even report `history_handoff:native_resume` although
  runtime fell back to a new session. Deadlines, transport closure and recognized Claude version
  incompatibility already fail closed; successful load with no echoed ID is correctly retained.
  FS-01.R10/R14/R33; INV §1/§8/§11. Reproduced with fake load rejection: fresh ID, successful return,
  stale 0.73/146,000-of-200,000 reading, no fallback metadata. The failing expected-contract test is
  `resume_continuity_investigation_test.go`; the fix may fail closed or expose an explicit recovered
  history handoff, but must not claim full restoration after replacing context. Reset usage on a
  fresh context and publish the actual handoff outcome. This generic fallback was deliberately
  warning-logged in `a434ef0a1` (2026-08-18); it is not the old missing-ID bug returning.
- **Must fix** — **C5, confirmed before the target's first prompt; later persistence undetermined;
  fix complexity difficult.** A cross-backend primer exists only in `spec.RuntimeSystemPrompt`
  (`switch.go:235–245`). Frozen snapshots intentionally exclude it, and later Resume/another native
  model switch reconstructs only the clean base prompt (`resume.go:504`, `switch.go:450`,
  `runtime/runtime.go:97–110`). If the target is stopped/restarted or switched again before sending
  its first prompt, no provider conversation has yet recorded the bridge: pinned Codex explicitly
  defers developer-context insertion until the first real turn (`core/src/session/mod.rs:1549–1555,
  :4296–4322`); pinned Claude creates an empty SDK input stream and records system-prompt snapshots
  only on the first request. The next native load succeeds but its first prompt lacks the
  pre-switch context. Reproduced at the ACP wire boundary by
  `switch_resume_continuity_investigation_test.go`, with no target prompt between switch and stop.
  FS-01.R10/R13/R14, INV §1/§3/§11. Keep a durable, separately owned handoff bridge without polluting
  or stacking primers in the frozen persona. Verify immediate stop/wake, model switch, restart,
  post-target-turn resume, and compaction. Do not assert universal loss after a completed target
  turn: Claude SDK snapshots may retain the primer (account rollout dependent, currently absent
  on Bedrock/Vertex/Foundry), and Codex persists developer messages in turn history. A previous
  investigative inference from Codex `base_instructions` precedence was rejected because Chuck
  supplies `developer_instructions`; those are different fields.
- **Worth fixing** — **C6, confirmed request-level defect; actual model reversion undetermined;
  fix complexity medium.** An explicit model/backend switch from a native-inherited federation
  launch keeps stale `launch_config.native_inherited=true` (`switch.go:471`; runtime metadata carries
  it unchanged). The first switch sends the selected model, but later ordinary Resume or an
  identity-preserving switch clears `ModelID` because of that old bit (`resume.go:322`,
  `switch.go:231`). A fake-provider scenario captured `opus` on the switch load, no model on later
  load, while identity still said `opus`. Native provider persistence can mask this; the absence
  of a requested model alone is not proof of provider reversion. FS-01.R10/R30, FS-08.R17–R19,
  TS-04.R47; INV §1/§11. Preserve source provenance while tracking the current explicit model
  choice, and pin switch → stop → resume request delivery plus the effective-provider receipt.
- **Worth fixing** — **C7, probable OpenCode model defect; OpenHands effective model/prompt
  delivery undetermined; fix complexity difficult.** All four adapters claim
  `CanSwitchModelOnResume=true` (`backend/adapter.go:195,253,311,358`), but OpenCode/OpenHands declare
  no session config IDs, and `applySessionConfig` applies models only to Claude/Codex
  (`runtime/chat.go:2952–2965`). OpenCode receives an out-of-schema top-level `model`; inspected
  current upstream load ignores it and restores the persisted selection, while advertised model
  options provide a real change route. OpenHands receives `LLM_MODEL` in its environment but loaded
  session behavior is unverified. Both also receive a top-level `systemPrompt` whose consumption is
  unverified, so their primer can be built successfully without reaching the model. This is already
  a documented live gate in TS-04.R47/R54 and FS-09.A6, not a confirmed failure on an unspecified
  installed version. INV §11/§12. Obtain versioned provider-side evidence, advertise verified model/
  prompt capability, and correct delivery or refuse unsupported switches; do not infer effective
  model from Chuck's selected identity or a permissive fake.
- **Worth fixing** — **C8, probable concurrency consequence from confirmed stale-read ordering;
  fix complexity medium.** `switch.go:52–100` reads identity/merges requested fields before taking
  the exclusive lifecycle claim (`:130`). A second switch can finish in that gap; a later partial
  request then overwrites its untouched fields from the old identity. Example: concurrent backend
  change followed by `{interface:terminal}` derived from stale Claude state can revert the backend.
  The counting archive lease is not exclusion. Applies to concurrent requests/tabs/API clients;
  no barrier reproduction was run. FS-01.R13/R16, INV §5. Re-read and derive the target while owning
  the claim; add a deterministic partial-update interleaving regression.
- **Must fix** — **C9, confirmed API handoff bypass/coverage gap; fix complexity medium.** The
  stopped-agent `/resume` handler accepts `backend`/`model` overrides (`resume.go:17–29,214–229`)
  but always carries `snap.LastSessionID` to the target (`:524`), never invokes switch's compatible-ID
  decision or primer builder, and reports no handoff kind. A backend override therefore tries the
  old provider's native ID on the new provider, then normally hits C4's empty-session fallback.
  The UI/CLI use empty-body Resume and cannot directly switch a stopped agent; ordinary UI usage
  requires Resume then Switch. The accepted API override is nevertheless reachable and has no
  equivalent history contract. FS-01.R10/R14 and INV §2/§11. Either reject unsupported resume
  identity overrides explicitly or route them through the same handoff policy as runtime switch;
  test stopped cross-backend and incompatible model overrides.

**Case matrix / boundaries:** Every current adapter advertises model-on-resume support, so current
same-backend model switches take native resume; the incompatible-model primer branch is presently
future capability handling. Cross-backend is determined by configured backend ID, not just CLI
type: two Claude entries also take primer handoff. Successful Claude/Codex native loading normally
preserves native conversation and reapplies explicit model, effort and fast settings. Frozen cwd,
persona/project prompt, added directories and Chuck permission policy are kept; current backend/
model environment and provider executable are intentionally resolved for each new process (secret
values are never snapshotted). The visible transcript remains append-only across these operations.
Live effort/fast changes use session configuration without restart; header model/backend changes
use the stop/load lifecycle. Permission mode deliberately resets on resume/switch (FS-03.R88),
historical images are not automatically resent and multimodal context is not guaranteed
(TS-04.R90), and native child activity is excluded from cross-backend primers by FS-03.R58.
Native-load preservation does not keep an executing process/background command alive through Stop.

**Verification:** Four primer assertions plus native-load fallback and switch-before-first-target-
prompt assertions failed as predicted when temporarily enabled, then were retained skipped in
three reproduction files. Additional observational fake-provider tests confirmed stale usage and
native-inheritance request omission. Focused existing lifecycle/configuration tests passed for
`internal/server` and `internal/runtime` both untagged and with `sqlite_fts5`; source runs unset
`CHUCK_RUNTIME_ROOT`. Spec lint and diff checks passed. No product/spec changes or live
provider receipt. **Next:** `/fix` this whole investigation unit; preserve the provider-dependent
confidence limits and close behavior plus missing acceptance coverage together.

**Queue correction (2026-10-14):** This investigation originally existed only in local commit
`479c6188`; it was not pushed and was absent from the shared handoff, fix queue and v0.16.0 release
range. The shared handoff at `c80dd5fd` correctly recorded no open findings visible to that checkout.
Reconciled against that newer main without reverting its fixes or release record. Compared current
provider continuity code against the investigation baseline: primer, switch, resume and adapter
logic are unchanged; chat-runtime changes only optimize update routing, so the findings remain open.

Closed reviews through the `v0.16.0` cut are in the archive above.

## Blocked on human

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

See [`RELEASE-v0.16.0-notes.md`](../archive/state/RELEASE-v0.16.0-notes.md),
[`RELEASE-v0.15.1-notes.md`](../archive/state/RELEASE-v0.15.1-notes.md) (supersedes the
never-published v0.14.0 and v0.15.0 notes) and
[`RELEASE-v0.13.0-notes.md`](../archive/state/RELEASE-v0.13.0-notes.md).
