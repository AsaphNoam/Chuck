# Invariant incident history through 2026-11-13

The "Paid for by" evidence that stood under each class of
[`INVARIANTS.md`](../../features/INVARIANTS.md) until 2026-11-13. It is retained history, not a
rule: the live file keeps each rule, its canonical patterns and its guard.

## 1. Crossing a boundary must reset or republish derived state

Paid for by:
- `ui/src/api/sse.ts` — `lastPing`/`hydrationIds`/`lastAgentSeq` reset only on first connect, not
  on `EventSource` auto-reconnect: the watchdog reaped fresh connections with a stale clock and
  stale snapshot rows survived indefinitely.
- `internal/server/messaging_loops.go` — nudger cooldown keyed by `agent_id` alone survived
  stop/relaunch; `check_messages`/janitor mutated read state without republishing the agent, so
  unread badges went stale.
- `internal/server/reconcile.go` — stale-sweep overwrote a switched-to-terminal agent's preview
  with pre-switch chat text.
- `ui/src/components/.../FilesTab.tsx`, `CommandsTab.tsx` — one-shot fetch-on-mount snapshots that
  also answer-raced across agent switches (fixed with a per-agent request token).
- `ui/src/store/annotationStore.ts` — a per-agent persisted draft tray with no cleanup on any
  lifecycle boundary: a deleted agent's drafts survived forever and could resurface against a reused
  id, and growth across agents was unbounded against the localStorage quota. Browser-local state
  needs the same boundary handling as server-derived state, plus its own retention bound when no
  server event can be relied on (drop on the delete event; expire and cap on rehydration).
- `handlePutBackends` wrote the config and left the config-derived `onboardingCache` stale. A write
  is a boundary too: invalidate derived caches in the handler that writes their source.

## 2. Parallel paths that build "the same thing" must share one helper

Paid for by:
- launch vs resume vs switch each rebuilding LaunchSpec: `SkipPerms`/`AddDirs` dropped on
  resume/switch (`internal/server/resume.go`, `switch.go`); terminal launch silently dropped
  model/system-prompt/add_dirs/MCP registration.
- `session/new` vs `session/load` params in `internal/runtime/chat.go`: `load` omitted
  `cwd`/`mcpServers`, then later `model`/`systemPrompt` — resumed agents silently lost registration
  or kept the old model.
- Live transcript append coalesced streamed assistant deltas, but full transcript replay did not,
  so Archive and resume split one assistant reply into many bubbles. Fix: both paths use the same
  `appendRenderedEvent` reducer (`ui/src/store/transcriptStore.ts`). The phone conversation later
  forked the same projection and rendered one streamed reply as 29 rows.
- Creation was the only writer that checked inherited pipeline closure; admission, retry, re-arm and
  wait wake each re-derived (or skipped) the rule, so work that was already ready and watches that
  fired later escaped a stage completion or a Stop. Fix: one `taskRunOpenClause` fragment every
  writer appends to its own statement.
- The same drift, each fixed by one helper now in the registry: the live-pidfile refusal existed
  only on the foreground dashboard path; slug validation only on POST (a path traversal); the
  annotation excerpt clip and its type were declared three times and had already diverged; the
  task row's shape was re-spelled in five statements; every pipeline control path re-derived the
  run cursor; one notification payload was built inline.

## 3. Persisted fields never receive one-shot data; forms merge, never replace

Paid for by:
- `internal/server/switch.go` — backend-switch primer concatenated into `spec.SystemPrompt`, which
  is also persisted into the frozen `sessions.system_prompt`: the primer stacked on every
  subsequent switch. Fix: a runtime-only suffix field the DB write never sees.
- `ui/src/features/onboarding/steps/BackendStep.tsx` — twice: an untouched "Continue" wholesale-PUT
  a single synthesized backend over the seeded multi-model map; then the Validate button raced the
  initial `/api/backends` fetch and clobbered it again. Fix: merge-preserve + gate the handler on
  query readiness (`isLoading`).

## 4. Create/teardown symmetry: one teardown function, generation-scoped, on every exit path

Paid for by:
- `handleLaunch`/`handleResume` failure paths hand-rolled different cleanup subsets, leaking
  tokens/MCP/hook-settings → unified `teardownAgentRegistration` (`internal/server/launch.go`).
- crash path (`chat.onExit`) only called `registry.forget`, leaving a crashed agent's identity
  spoofable → `Registry.SetExitHook`/`handleAgentExit`.
- teardown keyed by `agent_id` alone let a late crash-teardown of launch N delete launch N+1's
  artifacts during a switch window (a reproducible flake, pinned by
  `TestSwitchRuntimeKeepsTargetRegistration`).
- `handleSwitchRuntime` cleaned OLD artifacts *after* NEW registration (wiping the fresh token),
  and its rollback covered only the final failure branch.
- `exec` discards the EXIT trap along with the shell image, so in `scripts/release/install.sh` the
  process that created the piped bootstrap's temporary file could never be the one to remove it, and
  every `curl | bash` install leaked it. Teardown belongs to the last process that still needs the
  artifact — here the lock-holding child, guarded to the installer's own `mktemp` name so a real
  `bash install.sh` cannot delete its own input.
- Stop on an agent the registry didn't own silently deleted the DB row and orphaned the live
  process → `runtime.ReapOrphan`: confirm PID liveness and signal before clearing state ("not
  owned" ≠ "not running"). Also: 404 means "no identity row", never "not currently running" —
  lifecycle verbs are idempotent for known entities.

## 5. Check-then-act needs an atomic claim or a generation token

Paid for by:
- `internal/runtime/permission.go` — approve/deny/timeout raced on `pending[toolCallID]` under
  separate locks; losers still emitted `EvPermissionResolved` → `takePending` +
  `ErrPermissionAlreadyResolved`.
- cancel SIGINT escalation rechecked a `turnActive` bool, so a cancel armed for turn A could SIGINT
  turn B started inside the grace window → capture and recheck `turnSeq`.
- `internal/bus/bus.go` + `internal/server/sse.go` — `Snapshot()` then `Subscribe()` as two locked
  calls lost any event published in the gap → `SubscribeWithSnapshot`.
- notification edge detection (`Manager.Touch` skipping `writeMu`; read-prev/write-snapshot under
  separate locks) double-fired or missed done/waiting_input toasts.
- `internal/state/messages.go` — "current turn budget" read via `ORDER BY rowid DESC` picked a
  stale row after resume reset the in-process `turnSeq`: prune all other rows for the key in the
  same transaction as the reset; never trust an in-process counter across restarts.

## 7. Read paths must not swallow errors or amplify damage

Paid for by (this one recurred **four times** as the same literal mistake):
- `rows.Err()` unchecked after scan loops — `ListInactiveSessions` (`internal/state/session.go`),
  `queryTrackedFiles`/`queryTrackedCommands` (`internal/server/files_commands.go`), and again as
  residue in `internal/state/migrate.go`. `rows.Close()` is cleanup; **`rows.Err()` is the only
  iteration-failure signal.** A mid-iteration failure otherwise silently truncates the list.
- `bufio.Scanner` in the transcript reader aborted the *entire* transcript on one oversized
  (>8 MiB) line (`ErrTooLong`) — could 500 `/transcript` and block resume permanently. Skip the
  record, keep the stream.
- `chuck reindex` wiped ALL agents' index up front, then aborted wholesale on the first
  unreadable transcript — the repair tool left the archive worse than before. Per-entity isolation.
- reindex flush logic lost the final partial turn when a transcript had one finished turn plus a
  crash mid-later-turn.

## 8. User-facing surfaces get parsed, bounded, in-vocabulary data — and errors always surface

Paid for by:
- `internal/server/reconcile.go` wrote the raw NDJSON transcript line (unbounded JSON envelope)
  into `status.detail`, corrupting every idle card preview each ~30s sweep → parse tail events,
  clip to ~120 runes (`lastAssistantPreview`); never overwrite vocabulary fields (`last_trace`)
  with out-of-vocabulary values.
- bare `void switchRuntime(...)`/`void rename(...)` in `ui/src/components/grid/CardContextMenu.tsx`
  swallowed structured server errors → extract the `{error:{code,message}}` envelope in
  `ui/src/api/client.ts`, `.catch → pushError` on every mutation.
- DELETE UIs implement the 409 confirm+force-retry loop the API contract requires.
- Pipeline **Continue** was disabled until continuation text existed without saying what it needed,
  while **Continue** and **Retry stage** were offered together without explaining that Retry creates
  a fresh agent with bounded prior summaries (`ui/src/features/pipelines/RunBrowser.tsx`).
- A successful project create/update stored a `cwd_not_found` warning and immediately closed the
  editor that owned it. The editor now remains open on warnings and a successful create switches to
  edit mode so acknowledgement cannot accidentally create a duplicate project.

## 10. Ship the wiring; kill the drift

Paid for by:
- Clone context-menu action stayed a disabled stub with an obsolete phase tooltip after the feature
  shipped; only a holistic pass caught it. When closing a change, check both the governing FS
  requirements and acceptance items for unowned promises.
- A blocked pipeline pause offered **Open agent** beside **Continue**, on a pause where answering
  in that chat was a dead end.
- Docs drift: README omitted `install.sh` defaults/prereqs; `MAP.md` described the messaging MCP as
  stdio after HTTP shipped; `architecture-flow.md` showed terminal→bus parity that didn't exist.
- `.claude/skills` vs `.agents/skills` twin drift ("Codex **or** Codex" — since fixed).

A capability advertised with no way to select it is §20's *offered*, not this class.

## 11. Cross-boundary serialization and protocol contracts preserve meaning

Paid for by:
- `layoutFromConfig`/`toConfig` (`internal/server/handlers.go`) built `Order` via
  `append([]string(nil), l.Order...)` → fresh install served `order: null` →
  `CardGrid.tsx`/`agentStore.ts` called `.filter`/`.includes` on it → TypeError, dead dashboard on
  first launch (`353e940`). The MSW fixtures returned `order: []`, so no UI test could see it.
- A syntactically valid hand-edited `backends.json` with a missing `backends` map or `models:null`
  reached `Object.entries` and replaced the whole dashboard with its error boundary. Fix: validate
  required nested collections on read, fall back server-side, and retain `?? {}` at the first UI
  consumer (`internal/config/backends.go`, `ui/src/features/settings/BackendsEditor.tsx`).
- A successful ACP `session/load` response without an echoed `sessionId` was mistaken for failure,
  so Codex resume fell through to `session/new` and abandoned its conversation history. On load
  success the requested id stays authoritative; only a non-empty echoed id replaces it.
- `errors.Is` broke when a sentinel was synthesized as another concrete `*rpcError`; typed nils
  make the same mistake less obvious. A status-bearing streaming event with no status once
  defaulted to `completed` and prematurely closed tool calls.

## 12. External-CLI invocations must tolerate version and environment variance

Paid for by:
- `internal/backend/credcheck/claude.go` ran `claude auth status --no-color`; older Claude builds
  don't have `--no-color`, so a logged-in user failed the onboarding credential check (`353e940`).
  Fix: retry without the flag on `unknown option` (`runClaudeAuthStatus`).
- Older Git accepted an unknown `--path-format=absolute` option, echoed it on stdout, and exited
  zero, so a worktree repository anchor became `--path-format=absolute\n.git`. Detect the echo,
  retry without the optional flag, and normalize the fallback locally.
- The ACP handshake once logged an incompatible protocol version and proceeded. Version gates fail
  and run the normal shutdown path; they never warn and continue with an unsupported peer (§24).

## 13. The test suite cannot see CSS or layout

Paid for by:
- The first-run onboarding wizard referenced `.dialog-overlay`/`.wizard-*`/`.form-field` that were
  never defined → unstyled soup on first launch (`353e940`).
- **Regression (2026-07-09 usability review):** the same class re-escaped onto the entire
  Settings/config surface — `.settings-tabs*`, `.config-*`, `.backend-card`, `.model-row`,
  `.env-row`, `.string-list`, `.btn-danger/-link/-sm` referenced, defined nowhere; tabs render as
  default buttons and the Backends editor's controls overlap.
- **Geometry is equally invisible.** The projects background menu listened on a content-height
  element, so most of the canvas opened nothing while the test right-clicked a heading inside it.
  Its fix (`min-block-size: 100%` on a grid with no `align-content`) then stretched every project
  card to the bottom of the viewport. Mermaid diagrams rendered black and oversized behind a mocked
  renderer, and the phone transcript split one streamed reply into 29 rows. A contract about where
  something is, how large it is, or what it covers needs a real-browser assertion; a parsed-CSS
  check is only a proxy.

## 14. A loopback bind is not a browser or filesystem security boundary

Paid for by: the 2026-07-11 security review — unauthenticated dashboard API exposed to DNS
rebinding, terminal WS accepting any origin (`InsecureSkipVerify` with no outer check), `/mcp` and
the WS route mounted outside the API middleware, and a world-readable home tree
(`TestDNSRebindingHostRejected`, `TestCrossOriginRequestRejected`, `TestHomeTreeIsOwnerOnly`,
`TestStateDBIsOwnerOnly`, `TestTranscriptIsOwnerOnly`).

## 15. Commit local truth before releasing external side effects

Paid for by:
- Permission approve/deny and timeout answered the ACP peer before writing the resolved-but-active
  status. A fast peer completed the turn and wrote `idle`, then the late local write overwrote it
  back to `busy`. Fix: claim once, publish status and `permission_resolved`, then release the peer
  (`internal/runtime/permission.go`).
- Cancel released a pending permission without emitting `permission_resolved`, so the live UI and
  durable transcript kept a dead Approve/Deny action after the peer had continued. Fix: persist the
  terminal decision before responding.
- annotate-and-assign delivered reserved mail and fired its nudge before appending the source
  annotation event, so a failed append returned 500 after delivery and the preserved-tray retry
  inserted a second copy for the recipient to act on twice (`internal/server/sessions.go`). Fix:
  validate the target and compose its payload, append the durable event, and only then deliver —
  the delivery is deferred into a closure precisely so the ordering cannot drift back.

## 16. Bound work and memory where they enter the system

Paid for by:
- The mail-activation sweep selected every pending row and started one goroutine per row every two
  seconds. It now selects `messaging.ActivationBatch` oldest rows and shares one in-flight cap across
  sweeps, because a stopped-recipient wake can outlive the ticker (`internal/server/messaging_loops.go`).
- Transcript reconciliation kept a permanent second copy of every event and copied the growing
  array on every live delta. `rawByAgent` is now a short-lived append tail that exists only while an
  authoritative fetch is in flight and is cleared on settle, last-surface removal, or agent removal
  (`ui/src/store/transcriptStore.ts`).
- A joining tab restarted the shared SSE stream for every tab while dead ports accumulated for the
  worker's lifetime. The worker now replays one bounded retained snapshot to the joining port,
  releases each port on goodbye/pagehide, and closes the stream with the last port
  (`ui/src/api/sse-shared-worker.ts`).
- Worktree setup used `CombinedOutput`, buffering arbitrary command output before clipping its
  stored tail to 64 KiB. `setupOutputTail` enforces the byte bound while stdout/stderr are produced
  and preserves a valid UTF-8 boundary (`internal/server/worktree.go`).
- The FTS content buffer was capped at 1 MiB by silently dropping older transcript text. A bound
  states what it discards and tells the consumer that it did.

## 17. Tests must prove their contract independently

Paid for by:
- The retry-classification guard compared one hand-maintained classifier table with another list
  that omitted the same pipeline refusals. It passed while emitted errors had no declared class;
  the replacement derives emitted literals from producer files and enumerates forwarded dynamic
  pipeline codes (`internal/messaging/tool_result_contract_test.go`).
- A mail-activation test asserted a condition that could not fail, leaving its stated guarantee
  unproven until the assertion was made against the actual durable rows and provider prompts.
- The role-prompt migration's unreadable-file case exercised decode corruption instead, then
  skipped the whole case as root. Corrupt content, a real read-I/O failure, and write failure now
  have separate fixtures (`internal/config/config_test.go`).
- Release CI passed `-tags sqlite_fts5` only to packages with no tagged implementation and never
  inspected the shipped binary. The gate now tests the FTS packages and checks the packaged
  executable's build metadata (`.github/workflows/release.yml`).
- The server once marshaled an empty collection as `null` while its MSW double returned `[]`, so
  every UI test passed against a payload the server did not produce (§11).
- `fakeacp` accepted every `session/new` member Chuck sent, while the pinned ACP decoder strips
  anything outside its closed schema. The fixture therefore confirmed emission and was read as
  confirming delivery. The fake now decodes through the same member set, and the params check
  enumerates that set from the protocol schema and its backends from the adapter registry rather
  than from `sessionNewParams` (`internal/runtime/acp_session_schema_test.go`, TS-04.R54).
- The same shape of escape recurred across six field reports: a fixture authored from the code
  under test instead of from the real counterpart. `fakeacp` emitted an invented `{used,window}`
  usage result and an invented `sessionId` on load; phone transcript fixtures used whole-text
  events where the wire sends deltas; the remote-gateway annotation test posted a body without the
  `command_id` the bundled client always adds; the permission-save mock merged the requested value
  into every response; the Mermaid module was mocked out of its own sanitizer test.
- A manually tracked schema-version constant gets a guard test asserting equality with the
  migrations slice.

## 18. A waiting state names who can end it, and something notices when they cannot

Paid for by:
- A 23-hour pipeline run spent about nineteen hours in two stalls after its work was done. The
  stage agent stopped after a refused report; `OnTurnEnd` was a no-op without an accepted report,
  `internal/pipeline` had no sweep or timer, and the notification categories excluded the state by
  enumeration, so `await_result` with a silent agent rendered exactly like a run being worked.
- `OnTurnEnd`/`OnExit` read the run outside `runLock`, lost the revision compare-and-swap inside it
  and returned `nil`, which could park a run at `await_quiescence` with no attention reason, no
  notification and no log.
- A Think Tank participant that may not leave had `reply` as its only accepted discussion move, so
  settled participants produced filler until their allowance ran out, and one that ended its turn
  without submitting held the whole room until the operator intervened. Fix: a `pass` disposition
  for every participant, and a full quiet rotation ends the discussion.
- A root tool result arriving after the owned prompt settled set the agent `busy` again; only
  `finishTurn` restored `idle`, and no further turn end was coming (`internal/runtime/chat.go`).
- A person's prompt to an assignee whose task was waiting could not reach it: each attempt was
  refused, woke one empty continuation turn, and the assignee went back to waiting (§19).

## 19. Accepted input has one durable owner; a refusal changes nothing

Paid for by:
- A validated pipeline proposal was returned only to the MCP caller and persisted nowhere; the
  Pipelines page rebuilt proposals from tool-result content in one browser-remembered transcript,
  and the real adapter recorded that result as `content:null`.
- The waiting-assignee prompt path prepared the task continuation and then returned 409 before any
  receipt was written: the text was kept nowhere while the refusal had already woken a turn
  (`internal/server/attachments.go`). Waking mail on the same path was retained and delivered.
- A chat answer to a paused pipeline owner was marked submitted before provider delivery and left
  no durable user event; the successor visit was published before its input receipt was bound, and
  a bind failure fell through to ordinary standalone delivery after the run had advanced.
- Withdrawing queued text restored it to the composer while its `released` receipt stayed offered
  as a separate recovered message; a transient `unknown` snapshot outlived the Steer that
  submitted it. The draft, the held store and the receipt each claimed the same text.
- Answering a blocked stage agent in its own chat let it do a full turn of work whose report was
  then refused, because the run had already accepted that agent's `blocked` report.
- The backend-switch primer dropped every accepted human prompt, and the cross-backend bridge lived
  only in a runtime-only prompt field, so stopping the target before its first prompt lost the
  pre-switch conversation.
- A sent prompt rendered twice when its sequenced server event beat the HTTP response, because the
  unsequenced local echo was reconciled in one arrival order only.

## 20. Requested, offered and effective are different facts

Paid for by:
- BR-1: Chuck persisted and displayed the requested Codex model and effort for about ten weeks
  while the adapter dropped the field. The `model` member was asserted from a reading of the
  adapter's internal parser, accepted by the fake, and read by no provider. A claim that a provider
  *honors* a value needs a complete trace from the wire call to the provider-side read, or a
  credentialed run that observed the effect. An adapter-local readback is neither: an ACP
  `configOptions.model.currentValue` is adapter configuration evidence, not provider execution
  evidence, and a stale one produced a false "delivery is broken" finding (TS-04.R54).
- A Claude-to-Claude model switch recorded the new model while `session/load` resumed the
  transcript's previous one.
- A rejected `session/load` fell through to `session/new` and returned an ordinary successful
  Resume; a same-backend switch could report `history_handoff:native_resume` over a fresh session,
  with the old context-usage reading attached.
- A terminal provider failure carried inside a successful prompt result ended as `end_turn`: the
  status went idle, quota settlement recorded `successful:true`, and inline mail was marked read.
  Three consumers each derived success from the raw result, so correcting the displayed stop
  reason alone would have left the other two wrong.
- Model catalogs were imported from the user's personal CLI cache while prompts ran on the
  release-pinned runtime, so a selectable model failed at prompt time (Codex Astra; Claude 5.5).
  All four adapters claimed `CanSwitchModelOnResume` while model application existed for two, and
  capabilities once advertised a `tmux` driver that no API or UI surface could select.
- The paired-phone allowlist for annotation routes was a hand-kept field list that did not admit
  the `command_id` the bundled client always sends; the starter Claude effort list was a seed
  constant that lagged the provider's levels.
- A capability snapshot written before image negotiation decoded its absent key as `false`, so a
  stopped agent refused images as if the provider had; the context meter read an invented usage
  shape and showed 0% for every real turn.
- The permission toggle installed any 2xx response without comparing the saved value to the one
  submitted, so an acknowledged-but-unapplied write looked like a checkbox that would not stay set.

## 21. Agent-facing instructions and refusals are protocol

Paid for by:
- Every stage assignment said to report "exactly once" and that the one call ended the agent's
  part, while the shared retry vocabulary classified several refusals of that call as retryable.
  The agent obeyed the sentence it re-read each turn; the same result was accepted as soon as a
  person told it to try again (§18's nineteen hours).
- `stale_assignment`, retry class `never`, was returned for three conditions, one of them the
  run's own current attempt whose report had already been accepted. The shared vocabulary already
  had `already_reported` for that case.
- A task aimed at a stopped pipeline agent was refused as "No agent matches" for an agent that
  existed and that the same caller had just shared context with.
- Think Tank guidance told participants a turn was "a maximum, not a quota" while the tool accepted
  only `reply` from those who could not leave.
- Launched tasks carried no instruction to report a result; existing-agent activation said read and
  execute but not report, and the three assignee routes composed their text separately (§2).
- A `/review` task instruction was forwarded unchanged, the provider entered its native review
  mode, and that child's approval policy refused the task-control calls the assignment required.

## 22. Failure points leave a bounded, correlated trace

Paid for by:
- No refusal of `report_pipeline_stage_result` was logged anywhere, so two pipeline reports could
  be traced only through the agent's own transcript. The fields that were missing (caller and
  attempt agent/generation, pending action, code) were exactly those that separated the three
  conditions one refusal code conflated (§21).
- `rpcError.Error` returned only `Message` and discarded `code` and `data`
  (`internal/runtime/jsonrpc.go`). The same loss hid the cause of a clone's first-message failure
  and reduced a provider's "requires a newer version" reason to `Internal error`. A denied tool
  call's reason in `rawOutput` was likewise never decoded, leaving a failure with null content.
- A graceful dashboard shutdown recorded neither signal nor caller, so a freeze report could not
  separate server churn, browser churn and an external stop.
- A refused file read logged its path without the query and its status without the refusal code.
- `ErrorBoundary` only called `console.error` and nothing handled `window` errors, so a dead
  dashboard arrived as one bare message with no stack, route or build.
- Queue, withdrawal, Steer and paused-run delivery transitions kept only the receipt's latest state,
  so a recovery card could not be attributed to the transition that produced it.
- The prompt fingerprint covered only the frozen persona prompt, not the runtime-only suffix or
  knowledge revisions, so an archived task could not show which instructions accompanied it.

## 23. A reusable name is not an identity

Paid for by (each first recorded under the class named):
- Teardown keyed by `agent_id` alone let a late crash-teardown of launch N delete launch N+1's
  artifacts (§4); the nudger cooldown keyed the same way survived stop and relaunch, and a deleted
  agent's annotation drafts could resurface against a reused id (§1).
- Orphan reapers and both reconcilers trusted `kill(pid, 0)` and could SIGKILL a reused PID (§9).
- The in-process `turnSeq` restarted at zero on resume, so "current turn budget" picked a stale row
  (§5); a cancel armed for turn A could SIGINT turn B.
- `Revoke` deleted Chuck's token mapping while the MCP SDK session and its reader goroutine lived
  on: 100 revoked launches left 100 sessions and 100 goroutines (`internal/messaging/messaging.go`).
- A rejected `session/load` produced a new provider session under the same agent, which kept the
  old session's context-usage reading and was reported as the resumed conversation (§20, §24).
- `stale_assignment` answered three different mismatches — another agent, another generation, and
  the current attempt already reported — with one code (§21).
- Switch read the agent's identity and merged the request before taking the lifecycle claim, so a
  second switch finishing in the gap was overwritten from the stale read.

## 24. A fallback is a state change: fail closed, or make it visible, bounded and reversible

Paid for by:
- A rejected `session/load` fell through to `session/new` and returned an ordinary successful
  Resume: the model had no memory of the conversation while the dashboard still showed it.
- TS-03.R7 demoted a tab to its own direct event stream "for the rest of the session" when the
  shared stream did not open in time. Any restart longer than about 30 seconds demoted every open
  tab; five demoted tabs plus the worker exhausted the browser's six-connection pool and every
  request queued forever. Fix: demote only on evidence that the worker itself is dead, and keep
  reconnecting through an outage (`ui/src/api/sse.ts`).
- The SPA fallback answered a missing `/assets/*.css` with 200 app HTML, and the phone service
  worker cached whatever came back and never retried, so one bad response outlived the server
  restart that fixed it (`internal/server/spa.go`, `ui/public/remote-sw.js`).
- `CREATE VIRTUAL TABLE IF NOT EXISTS` froze a degraded FTS fallback forever (§9).
- A status-bearing event with no status defaulted to `completed` (§11); a capability snapshot
  with no image key decoded as `false` (§20).
- The ACP handshake logged an incompatible protocol version and proceeded (§12).
- Permission re-resolution on a role-read error must refuse, never fall back to the permissive
  global default (`resolveSkip`).

