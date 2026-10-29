# Repair quota recognition and session-reset recovery

**State:** Waiting to start
**Why:** Operator's 2026-10-29 annotation of `a_a7e830` event 199, followed by
`/design-feature fix this feature`. Product scope confirmed: “Yeah, it needs to just work as expected.”
**Relevant requirements:** FS-01.R55–R56/A37–A38; existing FS-01.R40–R44/R48–R51,
FS-04.R53/R59; TS-04.R105–R106 and existing R85–R88/R96–R97; TS-02.R61,
TS-03.R76, TS-08.R150, TS-10.R61 and existing R39–R47/R52;
INV §2, §4, §5, §7, §8, §9, §10, §11, §12, §15, §16, §17.

## Outcome

A provider terminal quota refusal enters Chuck's existing recovery flow even when AIR failure
metadata is missing. When Claude reports a monthly spend limit plus a named session reset,
Chuck schedules one isolated access check a minute after that session reset. Only successful
same-scope verification authorizes continuing the same unfinished work. A repeated spend refusal
explains the manual intervention needed instead of silently doing nothing or repeatedly waking work.

## Included work

- Extend the existing shared terminal settlement classifier for verified Claude/Codex RPC quota
  carriers; use it for ordinary prompts, reserved activations and isolated checks. AIR precedence,
  cancellation, turn ownership and automatic-successor suppression remain authoritative.
- Recognize the pinned native Claude monthly-spend/session-reset grammar, distinguishing a
  session hint from the combined blocking reset. Preserve observation-time, timezone, DST and
  precision rules. A session clock is not evidence that a monthly cap clears then.
- Add nullable `session_reset_at` to the existing recovery row and projection; keep `reset_at`
  null for this mixed case. Use existing `next_action:check`, due time, claims and allowance.
  Add `spend_limit` to the closed reason vocabulary. A repeated explicit spend refusal on the
  check stops automation even if it supplies another session time.
- Extend shared desktop/phone quota presentation, API/SSE and operator CLI decoding in lockstep.
  Show session reset and next check separately; restore committed state after reconnect/restart.
- Retain existing local-state lifetime, raw-payload exclusions, probe cleanup/native-history
  policy, permissions, cancellation, owner exclusivity and six-check ceiling. There is no
  new timer service, task, agent, setting, provider client or adapter patch.

Excludes historical-error scanning/rearming, passive subscription quota dashboards, changes to
monthly caps, inferred success at a clock time, and automatic replay of uncertain provider delivery.
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
  `ResetAt`; neither may receive the named session hint as a combined reset.

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

Recheck these producer contracts if the pinned artifacts change. Source receipts establish grammar
and carrier support; actual provider admission and continuation still require the explicit live gate.

## How we will know it works

FS-01.A37–A38 own the new acceptance evidence. Use captured incident fields plus separately
source-backed carrier fixtures; label reconstructed or omitted category values honestly. Drive
the real JSON-RPC boundary without AIR metadata for both ordinary and reserved turns and probes.
Confirm the incident regression fails on the old implementation. Negative cases include matching
assistant/user/tool prose, generic protocol errors, explicit throttling, cancellation, child/replay
events, newer non-quota revisions and malformed/oversized carriers.

Parser cases cover the exact named session clause, explicit date and time-only forms, invalid
zones, DST gaps/folds, stale/ambiguous/current-minute and conflicting windows. Mixed spend/session
errors never populate a combined reset or directly continue. Exercise both probe success and
repeated spend refusal, with unchanged/changed hint, through state, serialized projection and
rendered desktop/phone checks. One check is charged; no duplicate wake, notice, completed owned
work or replenished allowance occurs. Check supported and unsupported installed profiles honestly.

Reuse FS-01.A25–A27/A31–A34's owner, cancellation, restart, cleanup and uncertain-delivery
evidence where it covers unchanged paths; add focused regressions for the new carrier/hint.
Run applicable TS-06 closure checks, schema/spec checks and embed after UI changes. Update the
bundled operating-chuck guidance during the ordinary release refresh. Record a bounded supported
provider receipt when credentials/quota conditions permit; synthetic fixtures do not satisfy it.

## Waiting on

No remaining product or architecture decision. Implementation has not started. Live-provider
quota/admission evidence is an implementation acceptance gate; the incident alone is not a
successful automatic-recovery receipt.
