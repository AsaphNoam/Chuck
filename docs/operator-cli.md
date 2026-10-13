# Operator CLI

The operator commands use the running loopback Chuck service. They never start
the dashboard automatically. Reads return concise text; add `--json` to keep
the server object or array on stdout. JSON errors go to stderr and use exit 2
for local input, 3 for an unavailable service, and 1 for other failures.

Structured writes read one JSON object with `--request-file path` or
`--request-file -`. Text commands accept exactly one of `--text`, `--text-file path`,
or `--text-file -`. Files and explicit stdin preserve Unicode, multiline text and
trailing newlines. A shortcut flag cannot also supply the same request-file field.
No command implicitly reads stdin or prompts for missing values. Use each operation's
`--help` for its request shape, required stable IDs, guards and an example.

The command inventory is:

```text
chuck status
chuck agent list|show|status|launch|resume|stop|cancel|rename|identity|clone|archive|restore
chuck agent switch-runtime|session-config|send|held|withdraw|steer|approve|deny
chuck agent background-task-stop|transcript|files|commands|file-search|available-commands
chuck agent file|attachments|attachment|image|messages|think-tank-results|annotate
chuck project list|show|create|edit|delete|archive|restore
chuck group stop|archive
chuck worktree fork|status
chuck archive search
chuck role list|show|create|edit|delete
chuck backend list|show|create|replace|refresh
chuck config show|edit
chuck config-source list|preview|bind|refresh|detach
chuck remote show|configure|pairing-code|allow|decline|devices|rename|revoke
chuck pipeline template list|show|validate|create|update|delete
chuck pipeline proposal list|decline|delete
chuck pipeline run list|show|start|continue|retry|replace|repair-cleanup|stop|delete
chuck task list|show|create|cancel|result|retry|rearm|delete
chuck signal fire
chuck think-tank list|show|create|delete|entries|activity|files|commands|source-file
chuck think-tank send|annotate|pause|resume|end|retry|participant-turn-limit
```

Examples:

```sh
chuck project show --project demo --json
chuck project archive --project demo --yes
chuck group stop --project demo --group workers --yes
chuck worktree fork --project demo --title 'Demo fork' --branch demo-fork
chuck backend list --json --etag-output catalog.etag
chuck backend replace --request-file backends.json --etag "$(cat catalog.etag)"
chuck config-source preview --request-file source.json
chuck config-source bind --backend-id claude --request-file bind.json --yes
chuck remote pairing-code --json
```

Deletion, project Archive, group Stop/Archive, federation bind/detach, and device
revoke require explicit `--yes`. This acknowledges the operation but does not
grant force, dirty-checkout consent, stale-revision permission, or federation
preview consent. Responses report accepted, queued, and partial outcomes using
the existing server fields.

`agent launch` always creates an agent; `agent resume AGENT_ID` resumes that exact
agent. The existing `chuck role@project` form keeps automatic resume. `agent status`
combines session detail, hydrated live state, held-message state and a bounded
500-event permission window. Check its completeness field before interpreting
absence of an earlier permission. Approve/Deny use the exact pending tool-call ID,
including native-child identity. Cancel defaults to the current turn; quota
continuation requires its explicit episode and revision.

```sh
printf 'First line\nSecond line\n' | chuck agent send AGENT_ID --text-file - --json
chuck agent send AGENT_ID --attach report.pdf --request-id receipt-1 --json
chuck agent attachment AGENT_ID ATTACHMENT_ID --output report-copy.pdf
chuck agent transcript AGENT_ID --since-seq 42 --follow --json
```

Send/Steer accept up to ten regular files, 10 MiB per file and 25 MiB total,
through the existing attachment admission and capability checks. Attachment-only
Send is valid. Downloads require an explicit output path; files are staged with
owner-only permissions, and existing destinations require `--overwrite`.
Inspect or explicitly resubmit/dismiss owned recovery messages through `held`,
`send` and `withdraw`; a receipt retry must keep the same request ID and input.

Follow subscribes before reading durable history, reconciles transcript sequences
across gaps and reconnects, and periodically checks retained history on pings.
Each durable event is printed once in order. Reasoning activity is live-only.
Ctrl-C or a closed output pipe exits successfully and leaves agent work running.
Connection progress goes to stderr. JSON follow emits one object per line.

Ordinary requests default to a 30-second `--timeout`; follow uses a 30-second
liveness/recovery window. JSON/file/stdin input is limited to 1 MiB, message text
to the existing 64 KiB admission limit, responses to 64 MiB, SSE frames to 16 MiB,
and the catch-up tail to 256 frames or 32 MiB. Use existing list pagination and
transcript cursors to bound large reads. Terminal output escapes control characters;
JSON retains the original data.

Exit 0 means a successful read or accepted action. It does not mean model work
has finished. A successful HTTP bulk response with a failed member prints every
outcome and exits 1. A lost mutation response reports an unknown result and does
not replay the mutation. Inspect server state, or explicitly retry an identical
receipt-supported request. Saving a template and starting its run are separate
operations; a later failure does not roll back the earlier success.
