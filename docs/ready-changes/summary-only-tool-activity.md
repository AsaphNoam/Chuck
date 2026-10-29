# Show tool activity as one-line summaries

**State:** Waiting to start
**Why:** Operator request 2026-10-29: remove detailed commands from chat transcripts. A one-line
summary, like Codex's "• Searched the web for …", is the deepest drill-down Chuck should show.
**Relevant requirements:** FS-03.R108/A84 (refines R2; keeps R26, R67), TS-08.R147 (with
R146), INV §2, §10, §13

## Outcome
Every tool call reads as one short line, such as `Run spec checks`, `Ran git`,
`Read file SKILL.md` or `Ran npm · failed: 3 tests failed`. There are no argument dumps or
output bodies anywhere, and the working indicator names what the agent is doing now.

## Included work
- A shared pure `toolLine` helper (label, state, trimmed failure reason) used by the desktop,
  phone and child transcripts, `WorkingIndicator`, and the running-indicator popover.
- Delete the `ToolCall` argument disclosure and the `ToolResult` body, with their CSS.
- Excluded: permission prompts, which keep the full request; durable events, search, archive
  data and agent-facing reads; and the unknown-event JSON fallback.

Operator decisions (2026-10-29): permission prompts keep the full command. Failures show a
trimmed one-line reason (Chuck-derived: first meaningful line, no AI). Codex commands show the
program name only. Any unmapped case falls back to a short label.

Evidence: retained transcripts show Claude shell calls usually carry `args.description`, while
Codex `exec_command`/`execute` titles are the raw command. Of 212 failed results, 143
(Codex) have `content: null` and 69 (Claude) carry raw output, such as fenced `Exit code 1`
blocks or `<tool_use_error>` text.

## How we will know it works
FS-03.A84: table-driven `toolLine` tests over real-transcript fixtures, plus desktop, phone and
child component tests and a fake-ACP rendered scenario. Run the TS-06 UI checks and `make embed`.

## Waiting on
Nothing. It can run before or after the active running-indicator change; whichever lands second
switches popover rows to `toolLine`.
