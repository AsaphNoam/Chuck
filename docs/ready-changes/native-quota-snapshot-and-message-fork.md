# Native quota snapshot and fork from a message

**State:** Waiting to start
**Why:** Direct `/design-feature` request on 2026-11-11, after a full move off ACP was assessed and
rejected as too large. Replaces two ACP Wait-list entries: passive subscription quota (2026-10-10)
and edit/fork from an earlier message (held 2026-10-14).
**Relevant requirements:** FS-09.R80, R83/A49, A52; FS-01.R58–R59/A40–A41; TS-04.R109–R111
(building on R65, R89, R96–R97); TS-03.R85–R86; TS-02.R65; TS-08.R162; INV §2, §3, §7, §9, §10,
§11, §12, §15.

## Outcome

Every chat header shows its backend's complete 5-hour and Weekly subscription quota, kept current
automatically from one shared snapshot per backend. A person can Edit a sent message or Fork from a reply, each creating a new agent that ends
at that turn. Quota-interrupted work gets a known reset and a turn-free pre-check from the same
snapshot, with today's recovery kept as the fallback.

## Included work

Three independently shippable parts; ACP stays the conversation transport.

1. **Quota snapshot** — one small typed passthrough patch per packaged adapter
   (`_chuck/quota/read`, advertised at initialize), Go normalization, and one in-memory snapshot
   per backend quota scope (backend plus launch routing) that every agent of the scope projects.
2. **Edit / Fork from here** — record a fork point per completed turn, send it as
   `_meta.jetbrains.air.fork` to a verified adapter version, prove the cut from the fork's replay,
   extend the clone route and add the two desktop message actions.
3. **Recovery use** — snapshot as the lowest-ranked reset source and as a pre-check; every
   snapshot failure returns to the shipped R96/R97 path unchanged.

Operator decisions: quota and edit/fork only (Codex steering, model/effort and system-prompt
delivery stay on the Wait-list); route through the packaged adapters, an approved exception to the
standard-ACP-only rule and to the earlier no-patch/no-side-channel bans; Edit always forks;
recovery uses approved 2026-11-12 with the current behavior as fallback; and, on 2026-11-12, one
shared snapshot per backend instead of per agent, read automatically at any session start, any
turn end and once after a reset passes, with no Refresh control. Not included: a native driver
replacing either adapter, in-place transcript rewrite, file rewind, phone or terminal fork
actions, any client-triggered read, starting a process to read, persisting the snapshot,
model-specific or credit windows in the view, polling, and letting a healthy snapshot replace
the target-model check.

## How we will know it works

FS-09.A49/A52, FS-01.A40 and A41. A41 reruns A31–A34's recovery scenarios with the snapshot
failing in every position and requires identical results. Each adapter patch ships a contract test
run at assembly. Live receipts on packaged Claude and Codex are acceptance gates, not design
evidence: one quota read each compared with the provider's own display, and one fork each proving
the provider context ends at the chosen turn.

## Verified provider boundary

Checked 2026-11-11 in `scripts/release/node_modules` (Claude ACP 0.85.1, Claude Agent SDK 0.3.286,
ACP SDK 1.6.0, Codex ACP 2.1.1, Codex 0.159.3 schema from `codex app-server generate-json-schema`).

- **Claude quota:** `Query.usage_EXPERIMENTAL_MAY_CHANGE_DO_NOT_RELY_ON_THIS_API_YET`, control
  `get_usage` (`sdk.d.ts:3054`, `:4208`, `:4219`): `rate_limits.five_hour`/`seven_day` are
  `{utilization` 0–100`, resets_at` ISO 8601`}`; `rate_limits_available:false` without a
  subscription; `skipBehaviors` avoids the transcript scan. The adapter calls it only inside a
  `/usage` turn (`dist/acp-agent.js:105-141`). Standard ACP 1.6.0 has no quota method.
- **Codex quota:** `account/rateLimits/read` needs no thread or turn; `RateLimitWindow` has
  `usedPercent`, `windowDurationMins`, `resetsAt`. The adapter keeps it private and renders
  `/status` text (`dist/index.js:35249`, `:30460`, `:36118`); it treats `resetsAt` as epoch seconds
  (`:36156`, `:36196`).
- **Fork point:** ACP `ForkSessionRequest` has no fork-point field. Both adapters read
  `_meta.jetbrains.air.fork = {version: 1, messageId, messageFingerprint?, messageOccurrence?}`
  (`claude-agent-acp/dist/fork-session.js:7-24`; `codex-acp/dist/index.js:33447-33522`) and cut
  inclusively (Claude SDK `forkSession` `upToMessageId`, `sdk.d.ts:836`; Codex `thread/fork`
  `lastTurnId`). Chunk `messageId` is standard ACP; Claude stamps it live and on replay
  (`acp-agent.js:7863-7886`), Codex on live agent deltas and replay (`index.js:30574`, `:39105`).
  Neither adapter advertises the key, hence TS-04.R110's version gate and replay proof.
- **Unverified until the live receipts:** Claude's usage read on an idle session and mid-turn, and
  whether a second Codex process is ever needed (the design does not use one). A failed read is
  already specified as stale/unavailable with the fallback path.

Codex `thread/revert` and Claude `resumeSessionAt` exist but are unused: Edit always forks.

## Waiting on

None.
