# TS-12 — Worktree lifecycle

**Status:** Current
**Code:** `internal/worktree/`, `internal/server/worktree.go`, `internal/server/worktree_handlers.go`, `internal/config/worktrees.go`, `internal/state/worktrees.go`
**Absorbed:** —

## 1. Scope

The architecture behind FS-19 worktree projects: the Git execution boundary, the durable ownership
record, fork orchestration, disposable-checkout recreation, explicit deletion, and the API surface
the dashboard consumes. Out of scope: task/pipeline semantics (unchanged — they inherit the
project's cwd through the existing composition seam), multi-repo coordination, and any merge or
branch automation beyond creating the fork branch.

## 2. Design & constraints

- **R1 — One Git execution boundary.** All Git operations live in one package
  (`internal/worktree`), invoked argv-only via `exec.Command` (never a shell), rooted with
  `git -C <path>`, with bounded per-command timeouts, `stdin` closed, and `GIT_TERMINAL_PROMPT=0`
  so no operation can hang on a prompt. A missing `git` binary or an unparseable output is an
  actionable error, and parsing tolerates version variance (INV §12) by using plumbing commands
  (`rev-parse`, `symbolic-ref`, `status --porcelain`, `worktree list --porcelain`) rather than
  scraping porcelain UI output. No optional flag is load-bearing, and the fallback cannot be an
  error check: `rev-parse` answers an option it does not know by echoing that option on stdout and
  still exiting zero, so a query that prefers one (`--path-format=absolute`, added in Git 2.31)
  detects the echo, repeats the query without the flag, and makes the answer absolute locally.
  Because Git resolves the `-C` directory before printing an absolute path, the local form resolves
  symlinks too, so one repository reports one common directory however the caller's path reached it
  (R2's ownership match depends on that). Query helpers recognize only their documented no-answer exits
  (a plain non-repository directory, detached HEAD, or a missing ref); unreadable, missing, and
  corrupt repository paths preserve Git's actionable error instead of becoming an ordinary false
  or empty answer.
- **R2 — Ownership is a SQLite record.** Machine-created worktree ownership lives in
  `state.db` (TS-02.R1's writer split; a hand-editable JSON field could forge ownership and widen
  the deletion surface). A `project_worktrees` row — see §3 — exists exactly for checkouts
  Chuck created; its presence is the sole ownership test (FS-19.R4). The `project` column is a
  logical, non-cascading reference like TS-02.R25's task references: deleting a project never
  cascades here, and the deletion flow removes the row explicitly.
- **R3 — Fork ordering is crash-safe toward under-ownership.** Fork executes: validate
  (source project, repo, branch/base, target path) → create branch and worktree (the external
  effect) → write the new project file → insert the ownership row → run setup (FS-19.R3). Any
  failure before setup rolls back in reverse: delete the row, remove the project file, remove the
  worktree, delete the still-commitless branch — satisfying FS-19.R10's all-or-nothing rule. The
  ownership row is written **last**, after both the checkout and the project that owns it exist.
  Per INV §15 the record that authorizes a future deletion is never written ahead of the artifact
  it describes, so a crash anywhere in the window can only leave a checkout that is *not* recorded
  as owned — inert, conservatively treated as external, and reclaimable by hand.

  *Correction (2026-09-02, implementation):* the original ordering inserted the row before the
  project file. A crash in that window left a branch, a directory, and a row belonging to a project
  that did not exist, which no surface listed and no flow could reclaim, so the crash-safety claim
  was false for the design's own window. The row now goes last.
- **R4 — Checkout resolution is one shared helper on every start path.** A single
  `ensureWorktreeCheckout(project, cwd)` helper consults the ownership record and recreates a
  missing owned checkout from its recorded branch (FS-19.R7) before any process starts. It is
  called from `composeLaunch`, `composeResumeSpec`, and `composeSwitchSpec` — replacing the two
  currently duplicated cwd stat checks with one seam, per INV §2 and TS-01.R6/R9. Pipeline stage
  tasks inherit it through the ordinary task launch/resume dispatcher. Resume and switch keep the frozen `snap.Cwd`; the
  helper only re-materializes the directory the frozen path names, never re-derives a different
  path. Starts targeting an owned checkout share a per-project readiness claim, including starts
  arriving after recreation has created the directory but before setup finishes. Exactly one runs
  `git worktree add`; waiting starts observe setup completion and its warning before proceeding
  (INV §5). Cancelling a waiter does not cancel another caller's setup.
  Recreation reports itself in the start response; a recorded branch that no longer exists fails
  the start with an actionable error. Resume and switch thread caller cancellation through this
  work and expose the same recreation/setup warning notice as launch; pipeline continuation uses
  the same context-aware composer even though its control-plane response has no session envelope.
  A directory already present at the owned path is accepted only when its Git common directory is
  the recorded repository; a plain directory or another repository's checkout reusing the path
  fails the start with an actionable error and is neither repaired nor deleted.

  *Correction (2026-09-02, implementation):* the original text also called the helper from pipeline
  `ValidateStage`. That is the manager's read-only per-stage pre-flight, run before a run snapshot
  is committed and possibly for stages that never execute, so calling a mutating helper there would
  materialize checkouts and run setup commands for work that may not happen, and would report a
  recreation failure as a stage diagnostic instead of a start error. `ValidateStage` stays
  read-only and instead *tolerates* a currently-missing owned checkout at the project's recorded
  path, leaving the decision to the start path. The original text also promised a `status.detail`
  report; that field is the runtime's own bounded vocabulary surface (INV §8) and a start-path
  write into it would race the runtime's, so the recreation is reported in the start response only.
- **R5 — Setup runs non-interactively with launch-equivalent env.** `setup_command` runs
  as `/bin/sh -c` in the checkout with the same inherited server environment agents get, `stdin`
  closed, combined output captured with a bounded tail (64 KiB), and a 10-minute timeout. Result,
  timestamp, and captured tail are stored on the ownership row so the warning (FS-19.R3/A2)
  survives the creating request; the fork and start responses carry it too. The 64 KiB tail is a
  **storage** bound: everything crossing into a human-facing warning is clamped again at the
  presentation boundary to the last 2,000 runes, so a verbose failing build cannot push a log into
  a banner (INV §8).
- **R6 — Status queries are cheap where they are frequent.** The projects list enriches
  each project from the DB only (`worktree: {owned, branch}` — no Git subprocesses on the list
  path). `repo_backed`, needed for action visibility (FS-02.R60), comes from a bounded
  `rev-parse --is-inside-work-tree` check memoized per expanded cwd with a short TTL and
  invalidation on project edit. Expensive facts (dirty state for the archive dialog, FS-19.R8) are
  computed only on the on-demand per-project endpoint (§3); a failed or timed-out check reports
  "undeterminable", never a fabricated clean state (INV §8).
- **R7 — Deletion is gated, verified, and ordered.** Checkout deletion runs only inside
  the existing project-archive claim window (`beginProjectArchive`, TS-01.R13) or the
  project-delete path after the same claim, with all processes stopped. Before removing anything,
  the flow verifies the recorded path is the canonical `$CHUCK_HOME/worktrees/{project-id}`
  location, symlink-free, and currently registered to the recorded repository via
  `git worktree list`; any mismatch aborts with an error instead of deleting. Removal uses
  `git worktree remove --force`. The request carries the dirty-state snapshot the dialog disclosed;
  immediately before removal the server rechecks a checkout disclosed as clean and refuses if it
  gained uncommitted changes (FS-19.R8). Only then does it delete the ownership row, so a crash between the two leaves a row whose
  checkout is missing — the R4 recreation case — rather than an unrecorded owned checkout. External
  checkouts have no row and therefore no deletion path at all.
- **R8 — Owned paths follow the established filesystem rules.** The `worktrees/` root
  and `{project-id}` construction follow the project-resources pattern: slug-validated id before
  path construction, owner-only (0700) root, symlink rejection on root and leaf
  (FS-11.R8/R9, TS-02.R13). No Chuck code path ever removes a directory outside the owned
  `worktrees/` root.
- **R9 — Base detection order is fixed.** Effective base = project `base_branch` when
  set; else `origin/HEAD`'s target (via `symbolic-ref`); else the branch checked out in the
  repository's **main** worktree (the first entry of `worktree list --porcelain`); a detached HEAD
  with no `origin/HEAD` is an actionable fork error. Detection runs at use time (fork, FS-19.R2),
  never cached durably. The fallback reads the main worktree rather than the calling directory's
  own HEAD because forking a fork calls it from inside a linked worktree, where that HEAD is the
  source fork's branch — using it would stack the new branch on the source's work, which FS-19.R11
  forbids.
- **R10 — Git refs are the atomic claim.** Concurrent forks are not serialized by
  Chuck; branch-ref creation is the atomic claim (INV §5), so the second identical fork fails
  on the existing branch and rolls back per R3. Server-derived project ids make checkout-path
  collisions unreachable; the path is still pre-checked and fails closed.

**R11 — Setup cancellation contains the owned process tree and pipe drain.** Extend
the existing bounded setup-command helper with a dedicated process group, whole-group cancellation/
deadline termination, owned-child reaping and a one-second maximum post-termination pipe-drain wait.
Retain R5's ten-minute execution limit, 64 KiB captured tail, 2,000-rune warning and normal
setup-failure warning semantics; no uncancelled descendant may hold the recreation request open.
Reuse TS-01.R41's owned-command containment primitive, not a separate setup scheduler. This does
not change R4's per-project readiness authority or authorize shell/Git operations outside the
existing checkout/setup boundary.

## 3. Interfaces & data shapes

### Planned run-owned checkout specialization

- **R12 (shipped 2026-10-29) — Separate ownership, shared Git machinery.** Add a forward-only SQLite
  `pipeline_workspaces` record keyed by immutable server-generated workspace id, with unique run id
  and start request id, original project id, retained project/run display names, selection digest,
  repository common directory, resolved base commit, branch, canonical checkout path, frozen setup
  command, preparation/removal state, verified ownership flag, revision and R5's bounded setup
  result. Project/run references are logical and non-cascading. Use a distinct canonical namespace
  `$CHUCK_HOME/worktrees/pipelines/{workspace-id}/`, with R8's slug, symlink and permission rules.
  An intent row does not confer ownership: only a post-create verified record can authorize deletion.
  Reuse `internal/worktree` Git operations, setup containment, ownership checks and readiness logic;
  do not put Git subprocesses in the pipeline controller or create a second launch composer.
  Project-mode records freeze only the effective project cwd and any existing project-worktree
  ownership reference; they never claim ownership of that checkout or offer run-specific removal.
- **R13 (shipped 2026-10-29) — Preparation precedes agent admission and is idempotent.** For new-worktree mode,
  reserve the
  selected workspace intent with the run/first task transaction (TS-09.R61). A per-workspace claim
  serializes creation/recreation and removal, and every start holds a compatible readiness claim
  before even checking directory existence. Prepare validates the recorded repository and base,
  creates the new branch/worktree through R1, verifies registration/path/branch, then commits
  ownership before running the frozen setup command and marking ready. Setup failure retains R5's
  launchable-warning semantics; Git/preparation failure starts no runtime and exposes the cause
  through existing task/run attention. All waiters see setup completion. Request retries use the
  reserved identity/path/branch and existing run-start idempotency, not another allocation.
  Pre-ownership rollback removes only artifacts positively attributable to this attempt and a
  still-unchanged branch it created; collisions or altered contents are never removed. If a crash
  leaves ambiguous unrecorded artifacts, retain the intent and show an actionable preparation
  error; recovery does not guess ownership, adopt a replacement or start in the project checkout.
  Recorded owned preparation resumes bounded setup safely under the same claim after restart.
- **R14 (shipped 2026-10-29) — One resolver handles project-owned and run-owned readiness.** Extend the
  canonical checkout resolver with an optional server-owned workspace identity, retaining the
  current project path for unbound launches. Sessions persist `workspace_id` with frozen cwd and
  project; tasks obtain it from run lineage, and pipeline rooms from immutable pipeline origin.
  Resolve by indexed identity, not transcript/group lookup or an unbounded ancestor scan. The
  recorded path, actual common repository, registration and branch must agree before launch,
  recreation or removal. Recreate a missing owned checkout from its recorded branch and frozen
  setup command; missing branch or mismatched contents fail closed. A committed explicit-removal
  tombstone instead refuses launch/resume with the branch and recovery guidance. Reads failing
  are errors, never evidence of an ordinary/unowned checkout. Resume/switch keep frozen cwd and
  workspace id, including after deleting run history; metadata retention is independent of that
  history. This specializes R4's project-keyed ownership/readiness without changing ordinary paths.
- **R15 (shipped 2026-10-29) — Checkout removal is an explicit durable operation.** Offer removal only
  after a run is terminal and normal task/room cleanup has settled; dismissal defaults to keeping.
  Under the workspace claim, block new starts and refuse while any Chuck runtime/start claim using
  that checkout remains, including a resumed retained conversation or clone. Reuse existing
  generation/process identity proof; uncertainty refuses removal, and this action never stops
  unrelated work implicitly. Verify canonical namespace, symlink-free path, recorded Git common
  directory/branch and exact worktree registration. Disclose dirty/unknown state on demand; carry
  the disclosed `dirty_known`/`dirty` values and workspace revision in the confirmation, as R7 does.
  Immediately before removal recheck a checkout disclosed as clean and refuse clean-to-dirty or
  newly unknown state for fresh consent. Unknown state requires
  explicit acknowledgement, never a clean claim. Persist `removing` plus the consent revision
  before `git worktree remove --force`; do not delete the branch. Commit `removed` afterwards and
  retain the tombstone so launch cannot undo consent. Restart reconciles a missing checkout in
  `removing` to removed; a present/changed checkout requires fresh validation/consent rather than
  blind deletion. A failed removal remains inspectable and retryable, never reported as success.
- **R16 (shipped 2026-10-29) — Manage retained workspaces independently of run records.** Add local guarded
  `GET /api/pipeline-workspaces?project=...&cursor=...&limit=...`,
  `GET /api/pipeline-workspaces/{id}` and
  `POST /api/pipeline-workspaces/{id}/remove` with
  `{revision, dirty_known, dirty, acknowledge_unknown}`. The list is a DB-only bounded projection,
  reusing pipeline pagination bounds and initialized arrays; on-demand detail returns the bounded
  Git dirty state and workspace revision. Remove accepts no path or
  branch and uses R15's eligibility/verification, returning the authoritative workspace state;
  repeated removal of a removed workspace succeeds idempotently. TS-03 envelopes, local-only
  guards and TS-09 input/serialization bounds apply. List/detail include retained source labels,
  branch/path, preparing/ready/error/removing/removed state and bounded setup/error information;
  `missing` is an on-demand observation, not inferred from list I/O failures. No collection read
  runs Git for every row. Run deletion and project archive/delete retain workspace metadata and
  never cascade checkout removal. The same terminal/quiescence gates still apply after run deletion;
  the retained record stores settled eligibility before that deletion can commit. Tombstones and
  ownership survive without automatic expiry while retained sessions may reference them.
- **R17 (shipped 2026-10-29) — Compatibility and independent verification.** Existing project ownership
  rows, legacy sessions and runs decode without a run workspace id and retain current behavior;
  no destructive migration or retrospective regrouping. New runs in project mode freeze their
  chosen cwd for descendants while respecting any existing project-owned checkout recovery.
  Keep project resources, additional-directory grants and permissions scoped as before; worktree
  placement is not a sandbox and does not prevent an agent from choosing to access other permitted
  directories. Verify all launch families independently, actual Git registration, failed preparation,
  duplicate starts, setup cancellation, restart, branch/path substitution, removal/start races,
  clean-to-dirty changes, explicit removal versus out-of-band loss, and retention after run deletion.
  Use FS-14.A57–A60/FS-19.A8 as the observable oracle, not a second copy of resolver logic.

### Existing project-worktree interfaces

Migration (next version, forward-only, TS-02.R6):

```sql
CREATE TABLE project_worktrees (
  project        TEXT PRIMARY KEY,   -- immutable project id; logical reference, no cascade
  repo_path      TEXT NOT NULL,      -- expanded absolute Git common directory; stable after a source worktree is removed
  branch         TEXT NOT NULL,
  checkout_path  TEXT NOT NULL,      -- canonical $CHUCK_HOME/worktrees/{project}
  created_at     TEXT NOT NULL,
  setup_ok       INTEGER,            -- NULL until first setup run
  setup_at       TEXT,
  setup_output   TEXT                -- bounded tail of the last setup run
);
```

Endpoints (error envelope and status codes per TS-03):

- `POST /api/projects/{project}/worktree-fork` `{title, branch, base}` → `201` with the new
  project id and any setup warning; Git/validation failures are `422` with the specific reason
  (FS-19.R9/R10). Archived source projects are rejected.
- `GET /api/projects/{project}/worktree` → `{owned, repo_backed, branch, base, dirty,
  dirty_known, setup: {ok, at, output}}`; computed on demand (R6), feeds the fork form and the
  archive dialog.
- The archive and delete endpoints accept an optional `delete_checkout` boolean, honored only for
  owned checkouts (R7); its absence never deletes. When deletion is requested they also carry the
  disclosed `dirty_known` and `dirty` snapshot so R7 can detect a clean-to-dirty race.
- `GET /api/projects` gains per-project `worktree: {owned, branch}` and `repo_backed` (R6).

Project config file additions (`base_branch`, `setup_command`) are owned by FS-04.R45 and
TS-02's config authority; both are optional and absent-tolerant.

## 4. Invariants

- **INV §2 / TS-01.R6/R9** — checkout resolution joins the shared composition helpers (R4); the
  seam count goes down, not up.
- **INV §4** — the checkout deliberately opts out of create/teardown symmetry, like FS-11 project
  resources: creation has no automatic teardown on any exit path; the only teardown is the
  explicit, consented deletion flow (R7). This deviation is the FS-19.R8 product decision.
- **INV §5** — branch-ref creation is the fork's atomic claim (R10); a per-project claim
  serializes concurrent recreation of one missing checkout (R4).
- **INV §7** — status enrichment isolates per-project Git failures; one unreadable repo degrades
  that project's fields, never the list.
- **INV §12** — plumbing-only Git parsing with timeout and prompt suppression (R1).
- **INV §14** — owned paths are validated and symlink-rejected before any create or delete (R8).
- **INV §15** — ownership rows are written after the checkout exists and deleted after it is
  removed, so no crash window can authorize deleting something Chuck did not create (R3/R7).

## 5. Deviations & open decisions

- Setup uses `/bin/sh -c` with the server's inherited environment — matching what launched agents
  see — rather than the user's login shell; shell-profile-dependent tooling may need an absolute
  path in `setup_command`.
- The `repo_backed` memoization TTL is an implementation constant; it trades a short staleness
  window for a subprocess-free list path.

## 6. Traceability

- Git boundary: `internal/worktree/git.go`, `internal/worktree/git_test.go` (cases run against the
  installed Git, not a stub).
- Ownership: migration 20 in `internal/state/schema.go`, `internal/state/worktrees.go`,
  `internal/state/worktrees_test.go`.
- Owned paths: `internal/config/worktrees.go`, `internal/config/worktrees_test.go`.
- Fork, checkout resolution, setup, status, and deletion: `internal/server/worktree.go`;
  HTTP surface in `internal/server/worktree_handlers.go` and `internal/server/routes.go`.
- Start-path wiring: `composeLaunchWithOptions` (`internal/server/launch.go`),
  `composeResumeSpecWithGeneration` (`internal/server/resume.go`), `composeSwitchSpec`
  (`internal/server/switch.go`), and the read-only tolerance in `ValidateStage`
  (`internal/server/pipeline_lifecycle.go`).
- Consented deletion: `handleArchiveProjectAction` (`internal/server/archive_actions.go`) and
  `handleDeleteProject` (`internal/server/config_handlers.go`).
- Run-owned workspaces (R12–R17): migration 64 and `internal/state/pipeline_workspaces.go`
  (records, `placeRunTaskTx`, `CheckPipelineTaskTarget`); owned paths
  `PipelineWorktreePath`/`EnsurePipelineWorktreeRoot` (`internal/config/worktrees.go`); planning,
  preparation, recovery and the shared `resolveStartCheckout` seam in
  `internal/server/pipeline_workspace.go`; list/detail/remove and restart reconciliation in
  `internal/server/pipeline_workspace_handlers.go`. Anchors: `pipeline_workspace_test.go`
  (`TestPipelineRunWorkspacePlacesEveryNewAgent`, `...RefusesTargetsInOtherCheckouts`,
  `...PreparationFailures`, `...RecoveryAndTombstone`, `...SurvivesResume`,
  `...ExplicitRemoval`) and `TestPipelineThinkTankStageEndToEnd`.
- Regression anchors: `TestWorktreeForkCreatesBranchCheckoutAndProject`,
  `TestWorktreeForkOfAForkUsesTheEffectiveBase`, `TestWorktreeCheckoutRecreationAndMissingBranch`,
  `TestConcurrentCheckoutRecreationClaimsOnce`, `TestArchiveDeletesCheckoutOnlyWithConsent`,
  `TestExternalCheckoutIsNeverDeleted`, `TestWorktreeStatusAndListEnrichment`.
