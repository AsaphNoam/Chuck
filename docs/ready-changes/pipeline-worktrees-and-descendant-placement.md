# Pipeline worktrees and descendant placement

**State:** In progress
**Why:** Operator request 2026-10-29: pipeline stage agents and every recursively spawned agent
must use the pipeline worktree and designated group. Product and technical scope approved in
conversation, including keeping the original project and offering checkout removal when a run ends.
**Relevant requirements:** FS-14.R91–R96/A57–A60, FS-19.R12/A8, TS-09.R61–R65, TS-12.R12–R17;
INV §2, INV §3, INV §4, INV §5, INV §7, INV §8, INV §9, INV §10, INV §11, INV §12,
INV §14, INV §15, INV §16, INV §17.

## Outcome
Run setup optionally creates one isolated checkout while the pipeline stays in its original
project. Standing/replacement orchestrators, dedicated coordinators, pipeline Think Tank
participants/judges and recursive workers/reviewers share that checkout and initially receive the
frozen run-name group. Manual group moves survive. Existing targets in other checkouts are refused.

## Included work
Extend durable run/task/room provenance and shared launch/resume/switch composition with a run
workspace identity; keep project configuration, resources, permissions and additional directories
under their existing authority. New worktrees have run-keyed ownership, branch/base selection,
bounded setup, crash-aware preparation, missing-checkout recovery and actionable mismatch errors.
The optional Start field, proposals/digests, CLI/UI schemas, effective agent metadata and local
workspace list/detail/remove APIs carry one contract. Existing runs/sessions retain their behavior.

After terminal cleanup, offer desktop removal with Keep as the default and explicit dirty/unknown
state consent. Keep branch, commits and conversations; retain workspace management after run
deletion. Explicit removal leaves a tombstone and blocks silent recreation on resume. Desktop and
phone expose workspace choice/status; destructive management remains desktop-only. No per-stage
checkout, new project, merge/PR automation, extra filesystem access or new MCP action.

Verified seams: `internal/state/tasks.go` transactionally inherits pipeline lineage; ordinary task
launch flows through `internal/server/task_dispatcher.go` to the shared launch composer. Pipeline
Think Tank participants/judges instead launch through `internal/server/think_tanks.go` and must join
the same resolver. `SessionSnapshot.Cwd` already freezes resume/switch cwd, but existing worktree
ownership/readiness is project-keyed; extend the shared Git/setup machinery with separate run-owned
records, not a second pipeline launcher. No provider capability workaround is needed.

## How we will know it works
FS-14.A57–A60 and FS-19.A8: independently capture effective cwd/group through stages, replacement,
two descendant generations, room participants/judge, resume/switch and project-mode targets.
Real-Git fixtures cover creation/recovery, substitution refusal, idempotent starts, setup failure,
restart, deletion/start races and explicit removal versus out-of-band loss. Check schema/CLI/proposal
agreement and retained management after run deletion. A focused desktop/phone start/detail journey
plus desktop cleanup journey verifies operator feedback; packaged Claude/Codex launch/resume probes
verify actual cwd. Run TS-06's applicable closure matrix once after final implementation edits.

## Waiting on
None. Documentation validation currently has an unrelated pre-existing DELMSG-1 invariant-tag
error in `docs/features/HANDOFF.md`; do not absorb that finding into this feature.
