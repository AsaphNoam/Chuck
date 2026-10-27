# Show running background work instead of a Background tasks list

**State:** Paused
**Why:** Operator request 2026-10-26: the transcript-tail **Background tasks** list is hard to read
or understand, but knowing whether and what is still running is useful. Claude Code (`/tasks`)
and Codex (`/ps`) both show running work in an on-demand panel rather than a transcript list.
**Relevant requirements:** FS-03.R107/A83 (with R59, R67), FS-02.R83/A64, TS-08.R146,
TS-04.R104, TS-02.R59, INV §1, §2, §10, §13

## Outcome
The desktop chat shows an **N running** indicator beside the composer only while background
commands run. It opens a popover with each command's name, elapsed time, owning subagent, a jump
to the related tool call, and per-row Stop. Desktop dashboard cards show the same count. Finished
tasks keep only their tool-call label.

## Included work
- Remove `BackgroundTaskList` from every transcript (live, dashboard pane, archive) and its CSS.
- Add the composer-side indicator/popover from `collectTasks`, reusing the existing Stop behavior.
- Add `status.background_running`, published from the runtime's `openTasks`, projected only while
  running, and shown on desktop `AgentCard`.
- Excluded: phone indicator or count, stop-all, listing subagents, and any change to background-task
  lifecycle, retention, replay or Stop semantics.

Operator decisions (2026-10-26): composer placement, desktop dashboard-card count, nothing on the
phone, no stop-all, and background commands only.

Follow-up from comparing the Claude Code desktop and Codex apps (2026-10-27):
- **Subagents outliving their turn:** no rows are added. Under FS-03.R58, `nestActivities`
  (`ui/src/components/chat/runtimeActivity.ts`) marks a child still active at its root `turn_end`
  as `disconnected`, so Chuck has no live state to show. A background subagent that the runtime
  reports as an async task already appears as an ordinary task row.
- **Output on jump (first step of the work):** before building the popover, check with a
  real Claude and Codex background command whether the related tool call receives the command's
  output or only a "started in background" result. If it holds no output, pause and return to the
  operator; do not copy output into task rows (TS-04.R64).

## How we will know it works
FS-03.A83 and FS-02.A64. TS-04.R104 runtime count tests, TS-02.R59 migration/projection tests,
and TS-08.R146's desktop render check. Run TS-06 UI/Go checks and `make embed`.

## Waiting on
Operator decision (2026-10-27): real Claude and Codex transcripts show the related tool call holds
no command output (Codex results are empty; Claude's says output goes to a temporary file). Keep
the jump as a command locator, drop it, or redesign output access — see `HANDOFF.md` **Blocked on
human**. The card count and its runtime/status plumbing shipped.
