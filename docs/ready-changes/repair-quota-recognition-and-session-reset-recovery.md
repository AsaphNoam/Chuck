# Repair quota recognition and session-reset recovery

**State:** Waiting to start
**Why:** Operator's 2026-10-29 annotation of `a_a7e830` event 199, followed by
`/design-feature fix this feature`. Simplified by explicit operator direction on 2026-10-30:
recognize quota, dynamically extract its reported recovery date/time, continue one minute later.
**Relevant requirements:** FS-01.R55/R57/A37/A39; existing FS-01.R40–R44/R48–R51,
FS-04.R53/R59; TS-04.R105/R107 and existing R85–R88/R96–R97; TS-10.R62 and
existing R39–R47/R52; unchanged TS-02.R45/R50 and TS-03.R60/R72;
INV §2, §4, §5, §7, §8, §9, §10, §11, §12, §15, §16, §17.

## Outcome

A provider terminal quota refusal enters Chuck's existing recovery flow even when AIR failure
metadata is missing. For Claude and Codex, extract an unambiguous future reset/retry date and
time without depending on the exact surrounding sentence, then schedule the same unfinished
conversation for that instant plus one minute. This includes the incident's monthly-spend error
with a reported session reset. Attempt continuation directly; a preliminary access check is
unnecessary. If quota still blocks work, retain the existing bounded recovery/manual rules.

## Included work

- Extend the existing shared terminal settlement classifier for verified Claude/Codex RPC quota
  carriers; use it for ordinary prompts, reserved activations and isolated checks. AIR precedence,
  cancellation, turn ownership and automatic-successor suppression remain authoritative.
- Broaden the existing bounded date/time resolver beyond exact clause prefixes. Support common
  12-hour and 24-hour clocks, ISO and English month/day dates, explicit zones/offsets and the
  existing verified provider time-only conventions. Changed words or punctuation alone must not
  prevent extraction. Resolve from the provider error's arrival time; ambiguous/conflicting
  candidates, DST gaps/folds and uncertain timezone/date remain unknown.
- Feed the accepted target through existing `ResetAt`/`ResetSource`, `reset_at`, `due_at` and
  `next_action:continue`. The monthly-spend/session-reset combination authorizes an attempt,
  rather than claiming that the monthly cap itself expires at that time. Prefer applicable
  structured evidence and preserve conflict checks.
- Reuse current desktop/phone **Will continue at …**, API/SSE/CLI projection, cancellation,
  owner admission, restart and notice handling. No additional projection or presentation state.
- Retain existing local-state lifetime, raw-payload exclusions, probe cleanup/native-history
  policy, permissions, cancellation, owner exclusivity and six-check ceiling for existing
  unknown-reset checks. There is no new field, migration, reason code, timer service, task,
  agent, setting, provider client or adapter patch.

Excludes historical-error scanning/rearming, passive subscription quota dashboards, changes to
monthly caps, inferred success at a clock time, and automatic replay of uncertain provider delivery.
No LLM date extraction or arbitrary locale/date-order guessing. The separate preliminary-check
design (FS-01.R56/A38, TS-02.R61, TS-03.R76, TS-04.R106, TS-08.R150, TS-10.R61) was never
implemented and is retired; its stable requirement IDs remain as historical pointers.
This is a repair to the shipped recovery path, independent of the retained older quota change files.

## Verified boundaries and incident evidence

Read-only inspection on 2026-10-29 found:

- `a_a7e830` local transcript seq 199: provider text says monthly spend limit plus
  `your session limit resets 10pm (Asia/Jerusalem)`; seq 200 is a protocol error; seq 201
  ends `error` without quota evidence. The server log at 18:25:06 +02 records RPC `-32603`
  and data key `errorKind`, but **does not record its value**. No quota-intent row exists for
  the agent. Manual `Keep going` at 22:22:51 resumes work. Do not invent a full captured wire
  envelope or claim this proves what the missing category value was.
- `internal/runtime/quota.go` requires negotiated AIR/turn quota before reset extraction;
  `promptErrorText` preserves bounded RPC text but does not classify it. The probe in
  `quota_check.go:driveQuotaCheck` likewise requires a decoded quota failure. Reuse one bounded
  classifier across these seams rather than adding a watcher over transcript text.
- `quota_reset.go` only recognizes `· resets …`, so the named-session clause is not parsed.
  `server/quota.go` and `state/quota_checks.go` currently schedule direct continuation from
  `ResetAt`; these are the existing scheduling seams to reuse. Under the revised policy,
  that instant means the reported recovery target for an attempt, including the named session
  time in this incident, rather than a guarantee that every blocker clears.

Primary provider artifacts inspected under `scripts/release/node_modules` match the assembly pins:
Claude ACP 0.85.1, SDK 0.3.286, Codex ACP 2.1.1 and Codex native 0.159.3. This design uses available
carrier/parser seams; it assumes no missing upstream capability or replacement adapter.

- Claude `dist/acp-agent.js:7477` supplies `data.errorKind`; call sites at 4326/4369 pass the
  native error category. `dist/session-failure-extension.js:329–349` independently classifies
  `billing_error`/`account_on_hold` as quota and `rate_limit` as temporary throttling.
  The adapter's synthetic quota-error grammar supplies an independent text classification path.
  `acp-agent.js:2467` appends native detail on the legacy non-AIR path; negotiated AIR normally
  returns typed metadata at 2927–2954. Do not assume every unexpected negotiated RPC retains
  the text. Explicit non-quota classifications cannot be overridden by a matching clock phrase.
- Codex `dist/index.js:30839–30842` turns legacy `usageLimitExceeded` into an internal RPC
  error; `30968–30977` retains `data.codexErrorInfo`, `data.message` and optional details.
  Generic HTTP 429 does not establish quota.
- The pinned Claude native executable is 225167728 bytes, SHA256
  `75e3016e9d2570767b08e43a7467d4817a4f149232c169ca295f2c95fef21433`.
  At byte 183776801, `ege.five_hour` names `session limit`; `stn` at 183776988 formats
  `resetsAt` via `iu(..., true)`; monthly-spend builder `Acn` at 183778041 appends
  ` · your ${limitName} resets ${resetTime}`; prefix builder `yy` at 183784298 emits the
  quota prefix. This directly establishes the incident's named-session error grammar.
  Formatter `iu` at 177001554 and zone source `o5r` at 176987685 establish the existing
  English local date/24-hour horizon, omitted `:00`, year and IANA-zone conventions described
  in [the earlier source receipt](../features/QUOTA-RESET-FALLBACK-2026-10-16.md).
  The named session reset is a window hint, never proof that the monthly spend cap resets then.

Recheck these producer contracts if the pinned artifacts change. Source receipts establish the
observed carrier and provider date conventions. They do not restrict extraction to one exact
sentence: common alternate date/time forms and changed prose are deliberately supported by
FS-01.R57/TS-04.R107. Actual provider admission still requires the explicit live gate.

## How we will know it works

FS-01.A37/A39 own the new acceptance evidence. Use captured incident fields plus separately
source-backed carrier fixtures; label reconstructed or omitted category values honestly. Drive
the real JSON-RPC boundary without AIR metadata for both ordinary and reserved turns and probes.
Confirm the incident regression fails on the old implementation. Negative cases include matching
assistant/user/tool prose, generic protocol errors, explicit throttling, cancellation, child/replay
events, newer non-quota revisions and malformed/oversized carriers.

Parser cases cover the exact incident and alternate messages such as `Quota exhausted; available
again at 10pm (Asia/Jerusalem)` and `Usage limit reached. Retry at 22:00 (Asia/Jerusalem)`,
plus case/punctuation changes, `10:00 PM`, explicit ISO/month dates and supported time-only forms
for both providers. Expected instants come from the input and contract, not the parser under test.
Cover invalid zones, DST gaps/folds, ambiguous/current-minute dates, unrelated numeric tokens and
conflicting times. Mixed spend/session errors schedule direct continuation at target plus 60s,
with no preliminary probe. Exercise success, renewed quota with a fresh target, unchanged/past
targets, no time and delayed settlement through state, serialized projection and rendered
desktop/phone checks. No duplicate wake, notice, completed owned work or allowance reset occurs.

Reuse FS-01.A25–A27/A31–A34's owner, cancellation, restart, cleanup and uncertain-delivery
evidence where it covers unchanged paths; add focused regressions for recognition and extraction.
Run applicable TS-06 closure checks, schema/spec checks and embed after UI changes. Update the
bundled operating-chuck guidance during the ordinary release refresh. Record a bounded supported
provider receipt when credentials/quota conditions permit; synthetic fixtures do not satisfy it.

## Waiting on

No remaining product or architecture decision. Implementation has not started. Live-provider
quota/admission evidence is an implementation acceptance gate; the incident alone is not a
successful automatic-recovery receipt.
