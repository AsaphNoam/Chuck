# Model/provider continuity — investigation and closure

Historical diagnosis from 2026-10-14, closed locally by `b3c8d9f6` and `a015cfe5`.
The original findings below describe pre-fix behavior; they are no longer an open fix unit.

## Closure evidence

- C1–C3: canonical transcript projection retains human inputs, tool/diff evidence and folded
  assistant deltas. Streaming retains bounded UTF-8 text, reserves newest context first, and labels
  deterministic older excerpts honestly. The configured heuristic budget has an 8,000-token ceiling.
- C4: rejected native load fails visibly without session/new; real fresh contexts reset runtime and
  durable usage, including reindex. Successful native loads keep the requested ID if no echo arrives.
- C5: one bounded history bridge is stored separately from the frozen persona and replayable in
  session metadata. Wire/persistence tests cover immediate Stop/Resume, model change, completed
  target turn, fake compaction notice, cold server resume, clone and rollback; no live adoption claim.
- C6: current explicit model choice is tracked independently of source inheritance provenance;
  switch → Stop → Resume carries the requested model at the wire boundary.
- C7: OpenCode/OpenHands model-on-resume and primer delivery are unverified capabilities. Handoffs
  needing those routes refuse before teardown, following normal target validation; native restoration
  of the same identity remains available. This is conservative policy, not proof of universal failure.
- C8: switch claims the lifecycle before reading and deriving a partial target. The independent
  database-barrier regression fails against the old ordering and passes after the fix.
- C9: Resume refuses identity-changing backend/model overrides before side effects. Restore first,
  then use Switch runtime for the supported native/primer policy.

Original skipped C1–C5 assertions were enabled and their pre-fix failures confirmed; temporary
switch/resume overlays also confirmed C6/C8/C9 regressions reject the old implementation. Focused
regressions, lifecycle `-race` tests, tagged build and serial `make test` (untagged + sqlite_fts5)
pass. Parallel closure variants exposed an existing mail-receipt assertion race; ten isolated runs
and both serial full-suite variants pass. No mail delivery code changed. Live Claude/Codex provider
adoption and effective-model receipts remain gates in HANDOFF.md, as does OpenCode/OpenHands' existing
provider verification gate. The stopped-agent image issue is an independent open unit.

## Original investigation

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

