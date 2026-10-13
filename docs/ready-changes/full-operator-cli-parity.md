# Full operator CLI parity

**State:** In progress
**Why:** `/design-feature` request on 2026-10-11: the CLI is outdated relative to chat;
add all relevant capabilities. Full nonvisual web coverage and the concrete CLI
input/output/consent/local-service contract were confirmed on 2026-10-12.
**Relevant requirements:** FS-22.R1–R12/A1–A9; TS-01.R38; TS-03.R64–R69; TS-06.R34.
**Invariants:** INV §1/§2/§4/§7/§8/§9/§10/§11/§14/§15/§16/§17.

## Outcome

Inspect and control conversations, agents, projects/groups/worktrees, archive, pipelines, tasks,
Think Tanks and configuration/federation/remote administration from a terminal or script.
FS-22.R8 is the required operation inventory. Preserve existing commands, with readable new
output, explicit JSON, file/stdin input, clear accepted/queued outcomes and accurate history follow.

## Included work

Extend Cobra and one shared existing-loopback REST/SSE client; keep lifecycle, state, permissions,
owner/revision/ETag checks and receipts in their current server services. `--yes` acknowledges
deletes/bulk Stop/Archive and federation bind/detach, without replacing the existing separate
consents. Exits are 0 success/accepted, 2 local input, 3 unavailable service/provider, 1 other failure;
JSON success uses server objects on stdout and errors use JSON stderr. Legacy defaults stay intact.
Complete CLI help, examples and relevant operator references with the implementation.

No new database, provider integration, HTTP surface, runtime, scheduler, authentication or remote
transport. Exclude visual UI/layout, directory picker, embedded PTY/TUI, browser drafts, phone
self-device/push operations, unshipped quota probes and paused direct-action/MCP migration.
Attachment CLI work reuses the separately approved FS-03.R81–R86/TS-03.R62–R63 seams once shipped.

### Verified seams and limitations

`internal/cli/root.go`, `launch.go`, `resume.go` and `pipeline.go` establish existing syntax and
compatibility. `internal/server/routes.go` plus feature-owned handlers already expose the needed
operator actions; collection reads supply role/project/backend show without a new route.
Pipeline compact assignment syntax omits effort/fast; existing full request shapes supply them.
`internal/server/sse.go` is global snapshot/hydrated then live, with fresh hydration on reconnect;
there is no Last-Event-ID replay. Follow uses per-agent durable transcript seq and subscribe-before-
read reconciliation; ephemeral runtime activity cannot be recovered as transcript history.
Federation binding retains preview-issued consent; checkout deletion and stale revisions retain
their existing guards. Operator CLI authority remains TS-05.R2–R3; no scoped-agent tool transport.

## How we will know it works

FS-22.A1–A9 cover old-command compatibility, exact operation wiring, real-server fake-provider
conversation and management journeys, script bytes/JSON/errors/exits, follow races and interruption,
consents/ownership/stale targets, pipeline/task/room control, attachments and redacted administration.
TS-06.R34 defines independent wire/process evidence and the applicable final closure matrix.
No CLI test substitutes for existing live-provider or tailnet receipts.

## Waiting on

The completed [chat images and file attachments](chat-images-and-file-attachments.md) unit supplies
the approved multipart, receipt, storage and read contracts. This CLI change reuses those seams;
no dependency prevents completion and no unresolved product/technical decision remains.
