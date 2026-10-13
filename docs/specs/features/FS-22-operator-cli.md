# FS-22 — Operator CLI

**Status:** Current
**Code:** `internal/cli/`, `cmd/chuck/`, `internal/server/` · **Journeys:** —
**Absorbed:** —

## 1. Purpose

An experienced operator can inspect and control Chuck from a terminal or script without opening
the web interface. This spec owns the CLI surface and its observable input/output contract;
the existing feature specs continue to own each operation's semantics. It does not give agents
new scoped tools or replace the terminal-interface agent surface in FS-07.

The full nonvisual scope and CLI contract in R2–R4 were confirmed on 2026-10-12.

## 2. Behavior

- **R1 — Additive operator commands.** Preserve existing `chuck <role>@<project>`
  launch/automatic-resume behavior, `resume`, dashboard, auth, update, release, reindex and pipeline
  commands, including default output and exit behavior. New command families expose the inventory
  below, with operation-specific help, input shape and one practical example. Legacy launch forms
  retain explicit roles; No persona is `default` under FS-01.R46/FS-04.R54. A new
  `agent launch --role <role> --project <project>` accepts all supported launch settings and JSON
  output, always creating an agent; explicit `agent resume <id>` resumes one. No implicit current
  agent/project or display-name matching is introduced. Stable opaque ids and project/role ids
  identify targets; group labels remain project-scoped strings.
- **R2 — Composable results.** New commands default to concise readable output naming
  the target, resulting state and any actionable failure or warning. `--json` emits one server
  success object, array or JSON value on stdout, preserving fields and collection shapes;
  streaming produces one complete JSON object per line. An HTTP/local rejection produces no
  success value:
  diagnostics go to stderr, with one `{error:{code,message},http_status?:number}` JSON object there
  in JSON mode; human diagnostics never mix into JSON stderr. Commands with no JSON response body
  return `{}` in JSON mode after any requested file output succeeds. New commands exit `0`
  for a successful read or accepted action, `2` for local usage/input errors, `3` when the dashboard
  is unreachable or an existing provider-unavailable error applies, and `1` for other failures.
  A bulk operation with any failed member prints all member outcomes and exits `1` even if HTTP
  succeeded. Legacy commands keep their current defaults; opting into a new JSON mode applies
  this contract. Output never describes request acceptance as completed model work.
- **R3 — Explicit input.** Text-bearing commands accept exactly one of `--text <text>`,
  `--text-file <path>` or `--text-file -` for stdin. Preserve multiline text, Unicode and trailing
  newlines; do not shell-expand content. Structured operations accept `--request-file <path|->` containing
  one JSON object in the existing operation's shape, with documented common flags as shortcuts.
  Conflicting flag/JSON values, malformed JSON, missing required input, unreadable files and
  over-limit input fail before mutation. Do not silently read stdin, open an editor, prompt for
  missing values or invent defaults different from the operation's own contract. Existing pipeline
  `--input name=value` and continuation `--input` text retain their established meanings.
- **R4 — Preserve consequences and consent.** New commands never auto-start the
  dashboard; an unreachable dashboard names `chuck dashboard start` as the repair. Operations
  requiring web confirmation require explicit `--yes` in the CLI, failing with the consequence
  and repair if absent, in a TTY as well as a script. This covers deletes, group/project Stop or
  Archive, and federation bind/detach. It does not add confirmation to ordinary Send, per-agent
  Cancel/Stop, Clone, or reversible field edits. `--yes` never supplies separate existing consents:
  checkout deletion, dirty-checkout consent, shared-workspace acknowledgement and force semantics
  remain individually explicit. Permission approval identifies the exact pending tool call.
  Federation binding still requires the preview-issued, digest-bound consent token; `--yes` is
  an operator acknowledgement and cannot mint or replace that token.
  Required revision, ETag, expected-limit, opening-attempt and retry-request identifiers remain
  caller-visible; stale actions fail rather than silently rereading and overriding them.
  `backend list --etag-output <path>` explicitly saves the catalog response ETag to an owner-only
  new file while retaining the unchanged server JSON on stdout; it refuses an existing file.
- **R5 — Conversation control.** Agent commands preserve FS-03's Send/Steer,
  held-message replacement/withdrawal, capability gating, permission decisions and background-task
  control; FS-01's lifecycle/runtime changes and FS-01.R43's explicit quota-cancellation target.
  Ordinary Cancel submits `{target:"turn"}`; cancelling quota continuation requires
  `{target:"quota_continuation",episode_id,revision}` under TS-03.R60. Repeating cancellation
  against the same cancelled episode is a no-op; a stale episode/revision fails with conflict.
  Print actual `sent`/`held` and `steered`/`new_turn` outcomes. A stopped chat follows the same
  wake behavior as the web. Owned task/pipeline/room work retains its owner gates; CLI input is
  not a way around them. Status includes available context usage, runtime capabilities,
  pending permissions, held/recovered messages and quota-continuation state, preserving unknown
  values. Native children remain nested activity, not independent Chuck agents.
  `agent status <id>` combines the existing session detail, hydrated state snapshot, held-message
  read and a bounded 500-event permission window. Pending permissions retain exact tool/child
  identities; a window with earlier history explicitly reports incomplete permission coverage.
- **R6 — Inspect and follow.** `agent transcript <id>` reads durable history with
  explicit `--since-seq`; `--follow` follows that agent until interruption. Durable events remain
  ordered and duplicate-free across initial load, live delivery, gaps and reconnect; report
  unreadable/missing history rather than implying an empty conversation. Ephemeral thinking is
  labeled live-only and is never added to retained CLI history. Unknown event types remain
  inspectable. Connection gaps and reconnect progress go to stderr; durable catch-up precedes
  resumed live output. Ctrl-C closes the observer, exits `0`, and neither cancels nor stops work.
  Files, provider commands, native child events, background-task events, mail and room results
  can be inspected through their existing reads; command discovery does not execute a command.
- **R7 — Attachment parity.** Expose repeatable `--attach <path>` for Send/Steer under
  FS-03.R81–R86 and their approved attachment limits, native-image capability and receipt rules.
  Attachment-only Send is valid; empty Steer keeps its existing consume-held meaning. Inspect,
  explicitly resubmit or dismiss an owned recovery message without changing a different hold.
  Owned-byte and explicit local-image reads support explicit file output rather than writing
  binary bytes to a terminal; refuse overwriting an existing destination without `--overwrite`.
  Reading transcript metadata never fetches image bytes or opens a browser automatically.
  No CLI draft store, independent upload lifecycle or retention policy is introduced.
- **R8 — Complete management parity.** The inventory is the required coverage,
  not a generic arbitrary-HTTP escape hatch. Every command inherits the cited capability's
  validation, project/owner scope, permissions, warnings, retention and partial-failure behavior.
  Complex request shapes remain available through R3's JSON input, including pipeline participant
  and stage assignments with effort/fast, task prerequisites and Think Tank setup.

| Family | Required operations | Owning behavior |
|---|---|---|
| `status` | local service health and capabilities | FS-00, FS-09 |
| `agent` | list, show, status, launch, resume, stop, cancel, rename, identity/group edit, clone, archive, restore, switch-runtime, session-config; send, held, withdraw, steer, approve, deny, background-task-stop; transcript/follow, files, commands, file-search, available-commands, file, attachments/image reads, messages, think-tank-results, annotate | FS-01, FS-02, FS-03, FS-05, FS-06, FS-13 |
| `project`, `group`, `worktree` | project list/show/create/edit/delete/archive/restore; project-scoped named-group Stop/Archive; worktree fork/status, including title/branch/base and setup warnings | FS-02, FS-04, FS-05, FS-19 |
| `archive` | search with query, project and active filters, explicit limit/offset; project archive detail, retained agent/room discovery | FS-05, FS-21 |
| `pipeline template`, `pipeline proposal`, `pipeline run` | template list/show/validate/create/update/delete; proposal list/decline/delete, save/start via the template/run operations; run list/show/start/continue/retry/replace/repair-cleanup/stop/delete | FS-14 |
| `task`, `signal` | project-scoped task list/show/create/cancel/result/retry/rearm/delete; fire project signal | FS-16 |
| `think-tank` | list/show/create/delete; entries/activity/files/commands/source-file reads; send with participant mentions, annotate; pause/resume/end/retry, participant turn-limit edits | FS-21, FS-13 |
| `role`, `backend`, `config`, `config-source` | role list/show/create/edit/delete; backend list/show/create/catalog replace/provider refresh; config show/edit; federation list/preview/bind/refresh/detach | FS-04, FS-08, FS-09 |
| `remote` | show/configure enablement and keep-awake; create pairing code, allow/decline pending pairing; device list/rename/revoke | FS-20 |

## 3. States & transitions

- **R9 — Server-owned truth.** Commands use the running local Chuck service. They
  introduce no second database, transcript, queue, provider client, runtime session or automatic
  scheduler. Accepted messages and mutations live in their existing Chuck-owned storage under
  the same lifecycle and deletion rules. Unsent input remains in the caller's file/stdin; stopping
  the CLI does not preserve it as a browser draft. Remote pairing codes remain expiring secrets
  shown only by the explicit pairing operation. Config/source reads retain existing redaction.
- **R10 — Multi-step work is honest.** Starting an agent then annotating it, or saving
  a template then starting a run, remains explicit operations sharing the existing server seams.
  Failure of the later operation does not claim the earlier operation was rolled back. Annotation
  JSON names the existing anchor/excerpt/instruction shapes and self/agent destination; it is not
  a browser selection tray. Think Tank deletion leaves participant agents/history intact. Group
  results identify each member's actual outcome, including stopped-but-not-archived failures.

## 4. Edge cases & errors

- **R11 — Bounded and repairable.** File/stdin input, response bodies and stream frames
  have finite documented limits. Large lists/history use existing bounded queries and pagination;
  no hidden unbounded fetch-all. Text display treats external content as data, escaping terminal
  control sequences without altering machine-readable content. Errors identify the target and
  server reason when safe; no secrets, raw credential files or provider error dumps are exposed.
  A transport failure after a mutation may leave its result unknown; report that uncertainty,
  print the caller's request id only where the operation supports one, and do not automatically
  issue a second mutation. Existing
  idempotency receipts may be explicitly retried with identical request ids and input.
- **R12 — Exclusions.** No browser layout/density/order/theme presentation commands,
  selection tray or browser drafts, embedded PTY/TUI chat editor, push subscription or phone
  self-device operations, provider-native credential API, raw MCP/hook producer writes, context
  capability minting or planned `chuck action` transport. Native directory-picker and layout
  reads/writes are presentation-only and excluded. Settings may still edit the existing
  appearance preference as data. TBD provider quota reads and detached federation import remain
  excluded. Existing API authorization and phone allowlists are unchanged; CLI uses local operator
  authority under TS-05.R2–R3, not agent-token or paired-phone authority.

## 5. Acceptance criteria

- **A1** — Run a built binary against an isolated loopback service: legacy launch,
  automatic/explicit resume, dashboard, auth, maintenance and pipeline fixtures retain existing
  default output/exit behavior. Exercise new help and JSON launch/resume without browser opening.
  *Verify:* CLI compatibility tests and built-binary smoke.
- **A2** — From a listed agent id, Send during a busy turn, inspect/replace/withdraw
  the hold, Steer, and Cancel show exact delivery outcomes; stopped Send wakes normally. Unsupported
  Steer and owned room/task/pipeline restrictions return failures. Approve/Deny identify a native
  child's pending permission, stale decisions fail, and background Stop waits for a terminal
  runtime update before reporting stopped. *Verify:* CLI-to-real-server fake-provider journey.
- **A3** — Read then follow a transcript containing child/tool/diff/unknown events.
  Inject output between stream subscription and initial read, drop/reconnect, and introduce a seq
  gap: every durable event appears once in order. Thinking remains live-only; missing/corrupt
  history fails visibly. Ctrl-C/broken output pipe cleans up the observer without mutating the
  agent. *Verify:* controlled SSE/REST fixture and process-level interruption tests.
- **A4** — Every R8 operation maps to its intended HTTP method/path and independently
  supplied request/query/header fixture; no omitted family or generic proxy satisfies coverage.
  Include list pagination, annotations to self/another/newly launched agent, native command
  discovery, project-isolated group labels, worktree setup warnings and room source reads.
  *Verify:* reviewed operation inventory plus wire-level CLI tests.
- **A5** — Long Unicode/multiline stdin and files reach the server unchanged. Invalid,
  conflicting, oversized or missing input performs no mutation. Successful JSON retains unknown
  fields; flat and structured errors go only to stderr with the required exit; mixed group results
  retain successes/failures and exit nonzero. Untrusted escape sequences cannot control the
  terminal. *Verify:* independent stdin/stdout/stderr/exit and negative-input fixtures.
- **A6** — Delete, group/project bulk actions and federation binding refuse absent
  confirmation in TTY and non-TTY. `--yes` does not grant force, dirty checkout deletion, workspace
  sharing or stale revision/ETag/limit permission. Room deletion retains participants, task controls
  refuse room-stage ownership, and cancelled quota continuation targets only its exact episode.
  *Verify:* real-server negative/consent/concurrency fixtures and CLI no-request assertions.
- **A7** — Use the CLI to save/start/list/continue/retry/replace/repair/stop a pipeline,
  create/arm/result/retry a task, and create/send/pause/resume/end/retry a Think Tank. Full effort/fast
  assignments, stable command ids, attempt ids and stale revisions survive the wire. Existing
  server receipts prevent duplicate start on an explicit identical retry; ambiguous transport
  failure is never auto-replayed. *Verify:* bounded real-server fake-provider journeys.
- **A8** — Attach images/files via paths and attachment-only Send, inspect/withdraw
  a hold, resubmit/dismiss a recovered copy, and explicitly save retained bytes. Verify native image
  versus ordinary-file input, server limits, lost-response identical retry, source-file changes
  after acceptance, unsupported Steer and destination overwrite refusal. *Verify:* FS-03.A62–A65
  reuse plus CLI multipart/binary-output fixtures; no new provider-support claim without receipts.
- **A9** — Inspect/edit roles, project settings, backend catalog, federation and remote
  configuration from the CLI. Stale ETags fail, immutable config keys fail, preview consent remains
  required, responses stay redacted, allow/decline names a pending pairing and device revoke is
  explicit. *Verify:* isolated-home real-server administration fixtures; existing live tailnet/
  provider gates remain separate.

## 6. Deviations & open decisions

No unresolved product or technical decision remains. Attachment commands reuse the completed
FS-03.R81–R86/TS-03.R62–R63 server seams. Fake-provider CLI evidence does not satisfy separate
live-provider or tailnet gates. Status explicitly reports incomplete historical permission coverage
when its bounded window cannot cover earlier events.

## 7. Traceability

Entry points: `internal/cli/root.go`, `launch.go`, `resume.go`, `pipeline.go`.
Canonical operator inventory: `internal/server/routes.go`; original request shapes and service
semantics remain in their feature-owned handlers. Stream evidence: `internal/server/sse.go` emits
snapshot/hydrated then live events; transcript recovery uses the per-agent transcript sequence,
not global bus ids. `internal/cli/operator_client.go`, `operator_output.go`, `operator_agent.go`,
`operator_agent_status.go`, `operator_follow.go`, `operator_attachments.go`,
`operator_management.go` and `operator_workflows.go` implement the shared client and families.
Their wire/negative/observer tests plus `operator_contract_test.go`, `operator_pipeline_test.go`
and `operator_process_test.go` cover A1–A9. The built-binary process journey exercises the real
loopback server with fake ACP, exact permission IDs, explicit multipart retry after response loss,
Ctrl-C/EPIPE, legacy auto-resume, stale guards and redacted administration.
Operator examples and bounds: `docs/operator-cli.md` and `README.md`.
