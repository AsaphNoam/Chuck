# Quota reset fallback investigation — 2026-10-16

Design evidence for the operator's `/design-feature` request. No product code was changed and no
credentialed provider prompt was sent. The operator confirmed the direction, then requested checks
outside the interrupted conversation. Behavior is FS-01.R48–R51/A31–A34 and FS-04.R59/A38;
the operator also approved native archived Codex probe records. The complete planned contract is
in [the ready change](../ready-changes/quiet-quota-recovery.md).

## Finding

Chuck already receives provider failure text that can contain a reset, but discards it when
normalizing terminal quota evidence. Structured reset absence is not proof that a reset instant is
unavailable. Conversely, terminal error text and native CLI display are not interchangeable: the
negotiated adapter path and the native producer must both preserve the relevant information.

The smallest recommended order is: independently confirm terminal quota; prefer applicable verified
structured reset data; otherwise recognize verified provider error formats; otherwise use a bounded
unknown-reset isolated-check policy. Extend the current recovery path throughout. A check must not
be an ordinary wake in the interrupted conversation; see the confirmed refinement below.

## Pinned adapter evidence

The installed artifacts match `scripts/release/package.json`/lock: Claude ACP 0.85.1, Claude Agent
SDK 0.3.286/native 2.1.286, ACP SDK 1.6.0, Codex ACP 2.1.1 and Codex 0.159.3. Packaging adds Chuck's
existing Codex steering patch; no new adapter patch is proposed.

Paths below are relative to `scripts/release/node_modules/` unless qualified. They are source
inspection receipts, not captured live quota failures.

| Provider path | Actual carrier | Reset information and limits |
| --- | --- | --- |
| Claude, negotiated AIR (Chuck's path) | Successful prompt response with `_meta.jetbrains.air.sessionFailure`, plus applicable failure updates | `title` takes the synthetic native assistant failure text or result text; no typed reset field. |
| Claude, legacy client without AIR | JSON-RPC `-32603` | Native result detail can be appended to `error.message`; categorical `data` carries `errorKind`, not a reset. |
| Codex, negotiated AIR (Chuck's path) | Terminal AIR failure in prompt metadata; applicable session-info updates | `title` comes from native `error.message`; native `additionalDetails` and account reset snapshots are not copied into this failure. |
| Codex, legacy client without AIR | JSON-RPC internal error for `usageLimitExceeded` | `data.message` takes `additionalDetails ?? message`; `data.codexErrorInfo` and non-null `additionalDetails` are retained. The RPC message can remain generic. |

Claude anchors: `@agentclientprotocol/claude-agent-acp/dist/acp-agent.js:2467`, `:2927–2954`,
`:4318–4327`, `:4369`, `:4729`, `:4919–4929`, `:7468–7479`;
`dist/session-failure-extension.js:85–95`, `:121–140`.
The negotiated failure is not a JSON-RPC exception: changing only RPC handling would miss it.

Codex anchors: `@agentclientprotocol/codex-acp/dist/index.js:30809–30870`, `:30928–30978`.
`readableServiceErrorMessage` (`:30035`) unwraps a narrowly validated service-error JSON string;
the failure title otherwise retains its input. Quota classification uses `codexErrorInfo`, not the
reset phrase. `account/rateLimits/updated` is consumed privately (`:30460–30462`, `:30998–31010`);
`account/rateLimits/read` is used privately (`:35246–35250`). Neither is a reset-bearing ACP quota
failure. Do not scrape `/status` or add an authenticated side client to recover it.

Claude's separate `usage_update._meta["_claude/rateLimit"]` forwards native `rate_limit_info`, but
only after `lastAssistantTotalUsage !== null` (`acp-agent.js:4919–4929`). A first-request failure can
therefore lack this event even though the native process knows the reset. Do not turn a partial
event into a complete subscription snapshot.

## Native reset semantics

### Claude

SDK `sdk.d.ts:5591–5614` declares a numeric `resetsAt` without documenting its units. The pinned
native executable supplies the missing independent evidence:

`@anthropic-ai/claude-agent-sdk-darwin-arm64/claude`, 225167728 bytes,
SHA-256 `75e3016e9d2570767b08e43a7467d4817a4f149232c169ca295f2c95fef21433`.
Its embedded unified-window schema explicitly states that `resetsAt` is Unix epoch seconds. The
header consumer reads the unified 5-hour/7-day reset headers as numbers, checks the value multiplied
by 1000 against the current millisecond clock, rounds and retains that value as `resetsAt`. Its
event serializer retains numeric `resetsAt`. Native retry code also adds a 60-second reset buffer.
This is pinned producer/consumer evidence, not a guessed unit based on a number's magnitude.

The prior blanket assertion in TS-04.R87 that there is no producer-unit receipt is now outdated as
an investigation assumption. Revise the technical contract after feature confirmation, preserving
its independent terminal-quota, current-session/turn, rejected-window and model/overage applicability
checks. Unit verification alone cannot establish that a particular observed window blocked a turn.
This receipt does not automatically establish semantics for an arbitrary installed native version.

The embedded formatter was also recovered. Byte offsets are into the hashed executable:
`iu` at 177001554, `Wyt` at 177002138, `o5r` at 176987685, quota error builder `Acn` at 183778041,
and error prefix builder `yy` at 183784298. `iu` converts epoch seconds to a local `Date` and formats
English month/day/hour/minute; it omits `:00`. It includes a date only when the reset is more than
24 elapsed hours away (or a caller forces a date), and includes the year when it differs from the
current year. Thus a time-only error can mean tomorrow, unlike Codex. For that verified format,
resolve only a unique future occurrence within the producer's 24-hour horizon; DST ambiguity and
minute-boundary/stale evidence still remain unknown, rather than blindly rolling a past time ahead.

Crucially, `Acn` calls `iu(reset, true)`: ordinary rejected quota error text includes the timezone
from `Intl.DateTimeFormat().resolvedOptions().timeZone` in parentheses. Its prefix is composed by
`yy` as a usage-limit error, followed by a reset clause when available. The native status line and
wait UI use different formatter flags and can omit the zone; their display is not the error wire
contract. Credit/organization/overage branches also differ and may contain no applicable reset.
Support the verified rejected-error grammar, not every string containing a clock time.

### Codex

The [pinned 0.159.3 native error source](https://github.com/openai/codex/blob/rust-v0.159.3/codex-rs/protocol/src/error.rs)
provides stronger text semantics: `UsageLimitReachedError` stores a UTC instant; its error event
uses the formatted message. The reset formatter converts to process-local time. For today it emits
only hour/minute and AM/PM; other dates include month, ordinal day, year and time. It omits timezone
and seconds. Some plan/credit branches emit no reset at all; generic API rate limiting is a separate
error category. Relevant source lines: 432–480, 635–774.

A parser can therefore support the pinned Codex suffix only with a verified effective provider
process timezone. The browser timezone is irrelevant. Capture the date of the provider error,
including timezone overrides; a time-only error does not mean tomorrow when it has already passed.
Midnight races, repeated daylight-saving hours, invalid local times, unsupported zones and stale or
conflicting evidence must fail closed to unknown. Minute precision requires a buffer past the
displayed minute. A generic natural-language date library or an LLM would add uncertainty here.

## Where Chuck loses the information

- `internal/runtime/session_failure.go:11–17`: `QuotaEvidence` carries only failure ID/revision.
  The decoder validates title/details/reason but returns only identity and quota classification
  (`:38–105`). The bounded reset text is discarded even on the negotiated path.
- `internal/runtime/chat.go:1805–1825` retains the failure identity for the executing root turn.
  Load replay, children, idle updates and stale generations must stay excluded.
- `internal/runtime/jsonrpc.go:24–31` and `transport.go:150–158` preserve `rpcError.Data` as raw JSON.
  `Error()` returns only the RPC message. `promptFailureMessage` (`chat.go:2875–2964`) extracts only
  known model-policy/old-Claude reasons. Both ordinary and activation prompt error branches
  (`:983–995`, `:1686–1694`) complete with synthetic `{"stopReason":"error"}`, losing RPC payload
  details from settlement. Unknown raw provider payloads must not be dumped into logs/transcripts.
- `internal/runtime/chat.go:1838–1853` consumes usage updates only for context percentage. It does
  not normalize the Claude quota metadata. `internal/server/quota.go:62–76` always persists
  `unknown_reset` (or disabled), without `ResetAt`/`DueAt`.

Text parsing must follow independent quota confirmation, never create it. The current negotiated
AIR terminal limit/error/empty-actions policy provides that gate. A reset-looking generic RPC
error or arbitrary HTTP 429 alone does not. Legacy RPC shapes are useful compatibility evidence;
disabling AIR to obtain more error data is counterproductive.

## Existing recovery is the right extension point

`internal/state/quota.go`, `quota_admission.go`, `quota_owners.go`, `quota_controls.go` and
`internal/server/quota_dispatch.go` already provide one local intent per chat, due-time queries,
bounded dispatch, revision/attempt claims, provider-delivery receipts, same-work owner authorization,
manual cancellation and crash recovery. No second scheduler, task or daemon is needed. The operator's
quiet-check refinement uses a separate provider session, without creating another Chuck agent or
conversation. Current `scheduled` validation requires both reset and due times; an unknown-reset retry
must not invent a reset to satisfy it. Technical design should distinguish retry due time from reset.

Bounded check policy, direction confirmed: one hour between checks, at most six isolated
target-model requests for the same unfinished interruption. Persist allowance across
restart/resume and renewed failure IDs; charge only a provider attempt, not busy/capacity deferral.
Do not let every failed retry create a fresh six-attempt allowance. Admit at most one overdue
attempt after restart and schedule the next from its result, rather than catching up missed hours.
New applicable future reset evidence takes priority. Preserve all existing non-quota and uncertain
delivery holds. Explicit credit/plan/spend-cap intervention errors stay manual.

This covers a five-hour window while Chuck runs, but deliberately gives up on a long weekly wait.
It can consume normal usage when access returns. A lower cap misses more resets; indefinite polling
would violate the bounded recovery goal. Exact-time extraction alone is the smaller alternative,
but leaves first-request/missing/ambiguous reset errors unscheduled.

The UI must distinguish **Will check again at** from **Reset**/**Will continue at**, show remaining
allowance and keep Cancel available. A retry without reset evidence cannot truthfully append
**when the quota reset**. Proposed new notice wording and rollout behavior are in FS-01.R50;
Settings help must explain the policy under the existing toggle. No new visual layout is needed.

## Confirmed refinement — checks stay outside the conversation

The operator approved the direction on 2026-10-16 and requested that probes validate quota outside
the interrupted conversation so failed wake-ups do not pollute its transcript. Both the Chuck
transcript and original native provider session matter: hiding Chuck events while continuing to
send wake prompts into the same native session would not satisfy this request.

The pinned adapters have no common passive quota-read route. Claude's SDK offers an experimental
structured usage read (`sdk.d.ts:3038–3056`, `:4217–4303`), but ACP exposes it through a queued
`/usage` command (`claude-agent-acp/dist/acp-agent.js:101–149`, `:1857–1870`, `:2701–2723`). It could
run in a separate session; parsing its Markdown would still be another command-text contract.
Codex's `account/rateLimits/read` remains private to its native app-server client
(`codex-acp/dist/index.js:35246–35250`); the ACP registrations at `:40361` expose no passthrough.
An invented method or `_meta` key cannot make it callable.

A passive utilization snapshot is not sufficient positive authorization: it can omit model-specific
blocking scope, round values, or precede a competing request. Existing credential probes in
`internal/backend/credcheck/{claude,codex}.go` test login/API-key validity, not available subscription
quota, and must not be reused as a false quota-success signal. A tiny successful model request in
an isolated session is stronger admission evidence, though it still cannot guarantee the larger
original conversation will fit the available allowance.

The recommended check is one fixed, minimal target-model request in a separate ACP session, through
the current selected adapter/provider path. No interrupted history, work payload or coordination
tools belong in it. Use the same account, routing, model and quota-relevant configuration, rather
than a cheaper model or different API-key route. If that equivalence cannot be established, report
checking unavailable and retain manual recovery. Do not create a local Chuck agent/transcript or
apply quota-success/owner-settlement callbacks to check events. Checks can consume provider usage.
Only a successful check followed by revalidated ordinary owner admission can start the original
continuation; only actual continuation progress can create its Chuck notice. If the small check
succeeds but the real continuation immediately reports quota without a reliable new reset, hold
for attention instead of repeating successful checks and failed wake-ups.

### Provider lifetime differences and pending decision

Claude ACP already honors `_meta.claudeCode.options` for SDK query options
(`dist/acp-agent.js:6708–6725`, `:6827–6860`), including `persistSession:false`, `tools:[]` and
`strictMcpConfig:true`. The tools override is explicit at `:6744–6749`. SDK `sdk.d.ts:1808–1815`
defines non-persistence; `:2293–2299` defines strict MCP configuration. Use an isolated, minimal
workspace without discarding the original authentication/profile. Close/dispose tears down the
query (`acp-agent.js:5392–5406`); delete additionally invokes native SDK deletion (`:5407–5414`).
This is a feasible source-checked non-persistent probe path, not a live same-account receipt.

Codex ACP `newSession` forwards only generated config, model provider and cwd to `threadStart`
(`dist/index.js:34084–34105`), without `ephemeral`. Standard ACP `NewSessionRequest` also has no
ephemeral field (`@agentclientprotocol/sdk/dist/schema/types.gen.d.ts:5012–5039`). The adapter's
private title-generation helper uses `ephemeral:true` (`codex-acp/dist/index.js:36867–36885`), but
that helper uses its own title model and is not a quota-check API. Ordinary `session/delete`
calls `threadArchive`, not permanent deletion (`:34115–34124`). Thus separate probes can keep the
original conversation clean, but leave small native archived conversations.

The local `/rename <name>` command invokes `thread/name/set` without a model request
(`:35930–35936`, `:34126–34128`); the name update marks the title explicit (`:30424–30433`).
An explicit title prevents the automatic title-model request (`:36808–36814`, `:39019–39029`).
An implementation must corroborate that naming before the real probe, rather than assume that
sending `/rename` alone has completed. Clearly identify the record as a Chuck quota check.

**Operator decision:** use isolated probes and accept the bounded native Codex archive records,
reusing the existing ACP integration. No new
adapter patch, undocumented ephemeral metadata or separate app-server client is proposed.

Native `codex exec --ephemeral` and Claude `--print --no-session-persistence` exist, but avoiding
archive records this way adds another execution/output integration. Claude `--bare` or a fresh
config directory is not a shortcut to equivalent authentication: it can skip keychain credentials
or change the provider route. A fresh `CODEX_HOME` can likewise lose the original login/profile.
The native-CLI alternative is not selected solely to hide native history. These flags do not prove
live same-account quota admission, and read-only sandboxing is not proof of a tool-free model call.

An empty ACP MCP list is not a sanitized configuration: Codex `createSessionConfig` returns its
merged existing config unchanged when the list is empty (`dist/index.js:34187–34209`). A probe's
separate adapter process can use the existing JSON `CODEX_CONFIG` path (`:40301–40343`) to disable
each known configured server while retaining provider/auth/model routing. The native
[pinned MCP configuration type](https://github.com/openai/codex/blob/rust-v0.159.3/codex-rs/config/src/mcp_types.rs#L222-L224)
explicitly defines `enabled:false` to skip server initialization. An empty temporary cwd avoids
project-local configuration, but does not itself remove user/profile servers. Never mutate the
shared profile or replace the auth home to obtain isolation; unverified configuration layers stay
unsupported. No original Chuck hook/MCP registration or coordination credential is delivered.

Cancellation, owner closure and global disable must invalidate both a pending check and its later
continuation authorization. A check result cannot clear quota itself, advance a room or stage, or
authorize an obsolete owner. Restart must not mistake an interrupted check for an original-work
delivery; planned claim/cleanup semantics are TS-10.R45–R47 and TS-02.R50. The hourly allowance is
a check allowance, not permission to send six wake-ups to the
interrupted conversation.

## Verification and remaining gates

`/tmp/claude-quota-verified-20261016.mjs` checks the executable hash, extracts its exact source
functions/byte anchors and records `/tmp/claude-quota-verified-20261016.json`. Six assertions execute
the extracted pure formatter with synthetic clocks: today, tomorrow within 24 hours, weekly,
different year and both occurrences of a daylight-saving fold. The two fold instants produce
identical text even with an IANA zone, demonstrating why a zone alone does not resolve ambiguity.
The rejected-error timezone flag is checked directly against the extracted error builder. Separate
source inspection checks the unit declaration, header conversion and event serializer. None of
these checks executes a native provider process or establishes live quota delivery.

`/tmp/quota-adapter-replay-20261016.mjs` ran successfully, with receipt
`/tmp/quota-adapter-replay-20261016.json`. It invokes actual pinned Claude failure/category and ACP
SDK error helpers with explicitly synthetic input; it asserts the relevant Codex source branches.
It proves Claude title/RPC serialization, not a real provider's choice of reset text. The synthetic
ISO timestamp in that receipt is not an observed native Claude grammar. Codex checks are static,
not execution of its error handler or native provider.

Official newer-source check on 2026-10-16: Claude ACP main
[package 0.92.0](https://github.com/agentclientprotocol/claude-agent-acp/blob/main/package.json),
[failure projection](https://github.com/agentclientprotocol/claude-agent-acp/blob/main/src/session-failure-extension.ts)
and [rate-limit forwarding](https://github.com/agentclientprotocol/claude-agent-acp/blob/main/src/acp-agent.ts)
retain the same relevant text/reset split. Codex ACP main
[package 2.2.2](https://github.com/agentclientprotocol/codex-acp/blob/main/package.json) and
[event handler](https://github.com/agentclientprotocol/codex-acp/blob/main/src/CodexEventHandler.ts)
still project native message text into terminal failures and handle rate-limit snapshots separately.
These source checks establish no upgrade-only replacement for the missing terminal reset field;
they are not live compatibility receipts.

Before claiming provider support, capture a real terminal ACP quota response/update for each
supported packaged/installed producer and perform a bounded continuation receipt under A28/A33.
Use supplied or naturally occurring quota failures, rather than exhausting an account for a test.
Parser tests must be grounded in producer evidence and include DST, timezone overrides, midnight,
rounding, renewed failures and owner/restart/cancellation branches; handcrafted date strings alone
cannot prove the upstream contract. No live receipt was taken in this investigation.
