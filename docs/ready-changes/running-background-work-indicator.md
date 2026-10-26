# Show running background work instead of a Background tasks list

**State:** Waiting to start
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

## How we will know it works
FS-03.A83 and FS-02.A64. TS-04.R104 runtime count tests, TS-02.R59 migration/projection tests,
and TS-08.R146's desktop render check. Run TS-06 UI/Go checks and `make embed`.

## Waiting on
Nothing.
