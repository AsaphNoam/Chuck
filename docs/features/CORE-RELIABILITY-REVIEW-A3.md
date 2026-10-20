# CR-A3 — Executable workspace and configuration

Prepared 2026-10-20 from two focused GPT-6 Luna research passes and parent integration against
`a312b597`. This maps **A3 only** from the
[core reliability priorities](CORE-RELIABILITY-AUDIT-PLAN.md). It is one integrated review for a
frontier reviewer. Research inspected requirements, current callers and existing test assertions;
it performed no audit, failure probes, test execution or product repairs.
[HANDOFF.md](HANDOFF.md#review-queue) owns live status.

## Question and responsibility

Does admitted work start in the intended checkout, with completed setup and the intended executable
configuration, usable shared resources and access? Does continuation preserve its promised frozen
inputs or refuse clearly before disrupting working development?

Trace project/source selection → checkout/readiness → composition/preflight → native process
cwd/environment/setup → continuation or rollback. Prioritize work on the wrong branch or directory,
work starting before setup finishes, unusable resources, stale configuration passing a fresh-start
gate, and an effective model/setup differing from what Chuck claims. A correct `LaunchSpec`, source
preview or directory-exists check is not sufficient evidence of the environment actually used.

A3 owns the join between workspace and executable configuration. Inspect Codex and Claude chat,
plus the supported Claude terminal path where its composition differs. Include manual, task,
pipeline, wake, Resume and Switch callers without taking over their scheduling or result logic.
Unsupported terminal/provider or source/profile combinations may correctly refuse; do not expand
compatibility as part of the review.

## Read first

Read the current handoff's Current position and Active change, this charter and related findings.
Use workflow §§3–7 for review/reporting, with this explicit subsystem scope replacing a commit-diff
scope. Read requirements before following source:

- FS-19.R1–R11 and TS-12.R1–R11: Git identity/base, ownership, fork/rollback, recreation, setup,
  consented deletion and child containment. FS-04.R45 owns project base/setup fields.
- FS-11.R1–R10 and TS-02.R13: stable opaque shared resources, usable path and retention. Read
  FS-11 §6's existing-session exception before requiring resource grants on old snapshots.
- FS-04's project/backend/role/environment composition and FS-08.R1–R9/R34–R37 with
  TS-07.R1–R10/R14–R19: source authority, consent, project-aware precedence, freshness/cache,
  frozen continuation and realizable setup. Later execution support limits govern source previews.
- FS-01.R10/R12/R14 and FS-09's model/effort/fast and provider-source contracts; TS-01.R6/R9/R40,
  TS-02.R53 and TS-07.R19 define frozen selection and purpose-aware realization. Inspect
  TS-04.R20–R21/R71/R75/R101 where native homes, authentication and selected executable cross
  the process boundary. Ordinary Resume is not an alternate model/backend-switch API.
- TS-11.R4–R5/R10 only where runtime knowledge overlays must stay outside frozen user configuration;
  A4 owns the knowledge package/content. Read the invariant trigger index and matching bodies:
  likely §§1/2/3/4/5/7/9/10/11/12/14/15/17; include others only for involved surfaces. TS-06.R35
  governs proportional evidence; adequate existing proof can justify no new test.

Specs govern behavior. This charter, source comments and old reports are navigation. Detached
configuration import and broader arbitrary-root/profile execution are not assumed shipped.
Installed/native compatibility receipts remain acceptance-gated; missing receipts are not findings.

## Product distinctions the review must preserve

- A project is the isolation unit. Agents, tasks and stages in one project share a checkout;
  sibling worktree projects have separate checkouts/branches. Forking a fork uses the effective
  base unless the human explicitly chooses otherwise. No per-agent worktree, merge automation,
  multi-repository isolation or automatic task-created project is promised. Copied `add_dirs`
  may intentionally point outside the new checkout.
- Setup runs at checkout creation/recreation, not on every launch and not in the first agent's
  prompt. Starts must wait for setup to finish. A completed failure/timeout leaves the project
  launchable with a bounded warning; demanding successful setup as a universal launch condition
  would change the product. Setup uses the server's inherited environment under `/bin/sh`, not
  a guaranteed interactive/login shell or every backend/model-specific environment layer.
- Accidental checkout loss with retained ownership permits recreation at the recorded path/branch.
  It restores committed branch content, not lost uncommitted work. Explicit consented deletion
  removes ownership; restoring that project must not undo the deletion by recreating its checkout.
  External checkouts never become owned merely because project JSON names them.
- Frozen selection preserves explicit/native model semantics, cwd, policy and other specified
  snapshot inputs. It is not an immutable copy of external native files, authentication, process
  environment values or executables. Distinguish fields deliberately recomposed at process start
  from frozen inputs; do not persist secrets or demand that native inheritance become a pinned model.

## Starting authority and lifetime map

Verify this map against the selected HEAD; the physical directory or a cached view is not an oracle.

| Fact | Authority and producer | Consumer / distinction to establish |
|---|---|---|
| Project/role/backend definitions | Config JSON; config handlers/store | Fresh launch uses current project cwd/context and selected configuration. Worktree `base_branch`/`setup_command` are project fields, not ownership proof. |
| Owned checkout | SQLite `project_worktrees` plus actual Git repository/branch/registration | Fork/recreate/delete use recorded common repository, branch and canonical checkout path. JSON/path shape alone cannot authorize deletion or branch substitution. |
| Readiness and setup result | Per-project recreation/setup coordination plus stored setup result/timestamp/output | A present directory can still be under construction. Every actual start must join the same readiness owner/result; cancellation of a waiter must not discard another owner's setup. |
| Shared resources | Opaque files under `$CHUCK_HOME/project-resources/{project-id}`; config path helper | Stable by immutable project ID, outside checkout, retained after deletion. Ensure usability before start without scanning contents or widening existing restrictive permissions. |
| Bound native source | Provider files plus durable source binding/approved roots/overrides | Fresh project-aware resolution selects supported inputs. Discovery/preview is not consent, and readable inventory is not proof the runtime can execute that setup. |
| Effective view/cache | Manager generation, redacted fingerprints/health and optional disposable mirror | Watchers accelerate updates; `ResolveFresh` gates a fresh start. Last-known-good stale material can remain inspectable without authorizing a new launch. |
| Continuation contract | SQLite session snapshot and versioned execution contract | Composers consume frozen cwd/prompt/directories/permissions and explicit/native selection; explicit refresh or identity switch follows its own new-selection seam. Legacy ambiguity follows the documented derivation/refusal policy. |
| Actual process selection and environment | Provider selector/preparation, shared env helpers and native runtime | Observe selected executable/source, cwd, model request, native setup/home, auth origin and grants. Provider env/home layers are reserved; runtime-only hooks/knowledge do not accumulate into frozen user inputs. |

An approved native source root is not an authentication/session store to adopt wholesale. Current
TS-07.R19 supports Claude's personal setup with real project cwd and Codex's personal setup
mirrored into Chuck's private native home; unproved distinct roots/profiles refuse before effects.
Do not confuse that native session home with a disposable federation display cache.

## Start and continuation paths to trace

| Path | Source entry points and lifetime question |
|---|---|
| Fork → setup → first manual/task start | `internal/server/{worktree,worktree_handlers,config_handlers,launch}.go`, `internal/worktree/git.go`, `internal/state/worktrees.go`, `internal/config/worktrees.go`. Follow branch/worktree/project/ownership ordering, rollback and readiness through the actual process start. |
| New source-dependent launch | `internal/server/{launch,config_sources,provider_runtime}.go`, `internal/configsource`, `internal/backend`. Resolve against the actual project, preflight execution support and compose through shared helpers before registration/provider effects. |
| Ordinary Resume/wake/interface switch/rollback | `internal/server/{resume,switch,contract}.go`, `internal/state/session.go`, runtime native load. Preserve frozen values, reselect required executables/environment as specified, ensure the recorded cwd/resources and refuse unsupported setup before destructive teardown. |
| Explicit refresh or model/backend switch | Same composers and source preflight | Establish which operation authorizes a new contract, what refreshes, what stays frozen, and what rollback restores. A failed fresh resolve cannot silently use the stale cache or a different provider source. |
| Pipeline/task execution | `internal/pipeline/{manager,actions}.go` → durable stage task → `internal/server/task_dispatcher.go` → `launchAgent` or ordinary continuation composer | Follow the current producer-to-composer path. Stage preflight is read-only; preparation of a stage that never executes must not create a checkout/run setup. |
| Clone/native fork or isolated quota check | Contract consumers and runtime purpose-aware preparation | Check only preservation/isolation of this charter's execution inputs. Native fork consumes its source contract; a check uses an isolated no-work projection or refuses. A4/A2 own context delivery and check authorization respectively. |
| Archive/delete → optional checkout removal → Restore/new start | `internal/server/{archive_actions,config_handlers,worktree}.go` and config/state ownership helpers | Verify exact owned repository/path and consent at deletion time, dirty-state recheck and retained branch/resources. Preserve intentional deletion and fail safely on mismatches. |

The old `LaunchStage`/`ContinueStage` executor methods and some tests remain in source but have no
production callers at this baseline (ASRF-2/ASF-3). TS-12.R4 still names that old wiring. Inspect
the live pipeline → task → dispatcher path; a passing test of the obsolete executor cannot prove
current environment delivery. Preserve the existing cleanup finding rather than creating another.

## Connected searches and useful scenarios

These are facets of one review, not separately queued checks or a required Cartesian test matrix.

| Scenario / search | Evidence that distinguishes intended execution from apparent readiness |
|---|---|
| Fork/create → setup in progress → starts before and after directory creation | Compare ownership/setup state and actual child start time/cwd. All admitted callers join readiness, see the final success or warning, and cannot launch into unfinished setup. Known ASRA-6 remains ASF-5's finding. |
| Missing owned checkout → concurrent manual and task/wake starts | Compare real Git common directory/branch, frozen cwd, one recreation/setup owner, warning and native process start. A cancelled waiter cannot cancel unrelated ownership or strand remaining starters. Setup containment is already ASW-3's responsibility. |
| Project cwd/config or native source changes after a session starts | Contrast fresh launch, ordinary frozen Resume and explicit refresh/switch. Observe actual cwd and provider selection; a missing old frozen checkout must be recreated at its recorded path or refuse, never silently jump to the new project directory. Distinguish recomposed env from frozen fields. |
| Backend-global source connected without a project → launch in a sibling worktree | Compare project-aware native resolution and actual provider cwd/setup. Admission of the current project root does not widen persistent source-root consent. A preview made for a different checkout cannot prove the new checkout's effective native layer. |
| Source becomes invalid, watcher misses an edit, or executable source becomes unavailable | Inspect last-good display separately from fresh resolution/preflight and actual process invocation. No fallback to cached effective values, another executable source or an unsupported profile after acceptance-looking setup. A failed switch keeps the working source runtime where required. |
| Resources survive rename, checkout recreation or project deletion | Compare immutable project ID/path, contents, modes, composed env/directory access and actual child access. Do not require legacy snapshots to gain unpromised frozen resource grants or treat retained resource data as disposable checkout/cache. Known ASRA-5 remains ASF-5's owner. |
| Native preparation differs between fresh, resume, fork and check | Observe consumer-decoded selection/model flags, source/profile/home, auth origin and project grants with hostile ambient override values where meaningful. Native inheritance is omission, not a substituted catalogue model; checks must not inherit work payload/cwd access under a broad shared builder. |
| Archive/delete after checkout changes or another setup/start is pending | Compare disclosed consent/dirty state, exact ownership/common directory and guards immediately before removal. Check retained branch/resources and correct subsequent Restore behavior. Failed/uncertain ownership reads must not become permission to delete external work. |
| A fork/setup or source-preparation step fails after a partial local effect | Follow reverse cleanup and retained project/setup warning according to its boundary. Distinguish fork failures before setup (rollback) from allowed completed setup failure. Refusal must not leave a false running identity or silently replace an existing working contract. |

Choose a few connected disposable-repository journeys. A useful central journey is fork/setup →
actual child start → project/source edit → Stop → lost checkout → frozen Resume, with a contrasting
fresh/refresh refusal. Add other cases only where they challenge a different authority or consumer.
Do not multiply every provider/interface/source mode/project state or write tests that merely repeat
the composition helpers' outputs as the expected answer.

## Reuse these proofs before adding any

These tests were researched, not run or certified. Inspect the assertion boundary before reuse.

| Seam | Existing starting proof and its limit |
|---|---|
| Real Git fork/base/ownership | `internal/server/worktree_projects_test.go`: `TestWorktreeForkCreatesBranchCheckoutAndProject`, `TestWorktreeForkOfAForkUsesTheEffectiveBase`. These inspect real Git/project state; they do not launch an agent or observe its cwd. |
| Setup/recreation | Same file: `TestWorktreeForkFailingSetupStillCreatesTheProject`, `TestWorktreeCheckoutRecreationAndMissingBranch`, `TestConcurrentCheckoutRecreationClaimsOnce`. Failure warning and mutually exclusive recreation do not alone prove starts wait after directory creation. |
| Explicit deletion/external protection | Same file: `TestArchiveDeletesCheckoutOnlyWithConsent`, `TestArchiveRechecksCleanCheckoutBeforeDeletion`, `TestExternalCheckoutIsNeverDeleted`. Check Restore/missing ownership separately from accidental missing-checkout recreation. |
| Resource composition/access | `internal/server/project_resources_test.go`: `TestLaunchComposesProjectResources`, `TestResumeAndSwitchCarryProjectResources`, `TestProjectCreateFailsWhenResourcesUnusable`; `internal/config/resources_test.go`: `TestEnsureProjectResourcesRejectsSymlink`. Composed env/add_dirs and directory mode do not establish actual native access. |
| Freshness/cache and execution preflight | `internal/configsource/manager_test.go`: `TestResolveFreshInvalidSourceBlocksButKeepsLastKnownGood`, `TestSweepRecoversMissedEvent`; `internal/server/source_setup_test.go`: `TestSourceSetupPreflightRefusesUnrealizableBindings`. Manager cache health and preflight refusal are distinct from live setup adoption. |
| Frozen model/policy semantics | `internal/server/contract_test.go`: `TestContractKeepsSourceModelOverrideAcrossContinuation`, `TestContractEmptyModelOverrideIsNative`, `TestContractLegacySnapshots`; `internal/server/switch_test.go`: `TestResumeAndSwitchUseFrozenSkipAndAddDirs`. The last test edits policy/directories, not project cwd. |
| Executable/env parity and switch refusal | `internal/server/provider_runtime_test.go`: `TestProviderSelectionReachesEveryLifecycleComposer`; `internal/server/launch_test.go`: `TestComposeChildEnvCodexHomeOverride`; `internal/server/switch_test.go`: `TestSwitchCodexProfileFailurePreservesWorkingRuntime`. Composer parity does not establish the native child's setup/permissions. |
| Current pipeline seam | `internal/pipeline/manager_test.go`: `TestStartCreatesOneStandingStageTaskWithoutDirectLaunch`. It proves task creation, not dispatch. Existing ASRF-2 already owns replacing obsolete executor proof with real task-dispatch coverage. |

The main question is the combined current path from readiness and source choice to actual process
environment, especially after cwd/source edits and recreation. Existing tests narrow that search;
their names alone cannot establish the join. A disposable observer process can prove cwd, inherited
env, filesystem access and invocation; independent pinned native evidence or live receipts are
needed for provider setup/config/permission adoption. No credentials or secret env values belong
in a report. Preserve existing live gates when those environments are unavailable.

## Joins and finding ownership

Refresh the [fix ownership map](AGENT-SURFACE-FIX-UNITS.md) and live handoff. Reuse ASR-A/B and
[original joins J1/J5](AGENT-SURFACE-REVIEW.md#mandatory-joins-between-reviews).

- ASF-5 is waiting at this baseline: ASRA-6 owns launch crossing unfinished setup; ASRA-5 owns
  restrictive resource permissions being widened. Corroborate their current behavior, or record a
  distinct trigger/mechanism once; do not commission another implementation of the same fixes.
- ASW-2 and ASW-3 are closed. Preserve frozen native-selection/human legacy decisions, unsupported
  root/profile refusal, process identity and setup-child containment. Unrun native/terminal gates
  remain unrun; old failing overlays are not current findings. ASRF-2/ASF-3 retains stale pipeline
  executor/wiring proof; ASW-1 retains open event/control ownership failures.
- A1 owns snapshot/history durability and repair; A2 eligibility, slots, wait/wake and retry. A3
  owns where admitted work executes and which executable configuration is realized. Inspect those
  joins without substituting composition success for actual continuation or admission evidence.
- A4 will own exact information, managed knowledge and context adoption; A5 authoritative results;
  A6 general action/observation truth; A7 sustained overhead. Inspect adjacent code when it breaks
  A3's promise, and assign each finding once rather than stopping at a function boundary.

Do not add a new workspace manager, generalized readiness framework, per-agent isolation, merge
automation, arbitrary-root/profile support, backup or config migration. A valuable architectural
remedy discovered by review is a separately designed work item, not implementation during this audit.

## Required review result

Record exact reviewed HEAD, relevant dirty work and provider/adapter source versions. Produce one
concise ledger of workspace/config authorities, frozen versus recomposed values, live callers and
connected scenarios, with actual child/native evidence and its limits. Separate findings from
hypotheses and unavailable receipts.

Record real findings under **CR-A3** in the handoff using workflow §7: current location, credible
trigger, development impact, requirement/invariant and reproducible evidence or precise source-backed
mechanism. Give the unit one Fix model recommendation based on its hardest open fix. Respect
specified warning-only failure and legacy limitations rather than inventing stronger guarantees.

Use isolated temporary Chuck/native homes, disposable repositories and loopback-only servers;
preserve user data and dirty work. Follow the review skill's allowed reporting rules; change no
product code or normative specs. Close only this unit, retaining findings and honest evidence
limits. Do not run other audit units, fixes or a release architectural-review receipt as part of A3.
