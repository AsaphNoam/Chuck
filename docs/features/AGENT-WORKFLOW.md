# Chuck work workflow

This is the shared way Claude Code and Codex make changes, review work, and leave the repository
easy to resume. The feature and technical specifications say what the product must do; this document
says how agents work with them.

## 1. Start with the current state

1. Use the injected handoff header, or read **Current position** and **Active change** in [HANDOFF.md](HANDOFF.md). Open the selected queue and change by name. For edits, inspect Git status, staged changes and the existing diff in affected paths. Preserve other people's work.
2. Use the unit the human names. For a queue-owning role, an explicit command with no unit selects one eligible unit from its own queue. Say which one. Other queues and chronology do not gate it. Companions inherit the active task. For a standalone critique or test with no identifiable scope, ask which surface or behavior to examine.
3. Read only the governing FS/TS items, relevant surrounding requirements, and the procedure named by the launcher. For implementation, review, repair or investigation, scan the [invariant trigger index](INVARIANTS.md#trigger-index) and read each matching class in full. New interfaces, runtimes and drivers must satisfy the full INV §6 checklist. The specs define the contract.
4. Keep one originating unit through review and repair. Administrative commits, including fix closure, handoff edits and release records, never create review units. Use [§5](#5-commit-and-resume-safely) when changing repository state and [§6](#6-human-update) at exit. Companions reuse the parent’s scope and already-loaded context.

## 2. Make a change

For `/work`, resume the selected handoff unit or take a named or available **Waiting to start** change from `docs/ready-changes/`; move it into the active handoff. If none is available, report that. Use [§3](#3-when-to-ask-the-user) for unresolved decisions.

Implement large changes in independently verifiable slices. Record the next slices and dependencies in the handoff. Update the governing [specification](#10-keep-specifications-accurate) before changing behavior or architecture; a restoration of specified behavior needs no spec change. Extend an existing seam where it satisfies the requirement. Use [write-tests](#17-write-purposeful-tests) when new evidence or materially changed tests are needed. For material presentation changes use [design](#14-design-ui); for a changed task flow or consequential interaction use [UX](#15-shape-and-test-the-experience).

Run focused checks while iterating. At each verified checkpoint, inspect the scoped diff, update the [handoff](#4-keep-the-handoff-useful), commit, and continue while the selected unit has actionable work. On completion, add one substantive unit to the review queue alongside existing units.

### Completion checks

Run the applicable closure once after the final relevant edit (TS-06.R5). Run `git diff --check` for all changes.

| Changed inputs | Completion |
|---|---|
| Go product, dependencies or build | `make test` (both Go variants, includes spec checks) and `make build`. Include affected UI checks and focused race checks for concurrency changes. |
| UI only | `make check-specs`, then `cd ui && npm test && npm run build`. Add focused Go integration evidence for a changed Go-facing contract. |
| Docs, workflow or verification tooling | `make check-specs` and relevant link, syntax or focused tooling/claim checks. |

`make dist` rebuilds UI, embedded assets and binary; omit the superseded build steps. Generated embed changes alone do not trigger Go suites. For focused UI tests use `npm test -- <filter>`. Capture verbose logs to a temporary file and inspect the summary or failure.

Reuse a passing check across work, fix and release only when source, dependencies, flags and environment still match. Record command, result and revision or scope in the verifying commit. CI still checks a clean clone. Keep a failing lifecycle, concurrency or recovery test open in the live findings until its cause is explained and repaired; a passing rerun cannot close it.

## 3. When to ask the user

Ask when an unresolved product, security, privacy, retention or compatibility choice changes the contract, an action needs credentials or destructive/irreversible authority, requirements conflict, or required checks remain blocked after reasonable attempts. Record a blocking decision and its consequence in **Blocked on human**. Make reversible local choices within the contract; flag consequential assumptions for review. A local note cannot narrow a promised requirement.

## 4. Keep the handoff useful

At each verified slice and before an intentional interruption, record the selected unit, completed work, next action, and checks passed or owed. On resumption check that state and the current diff before repeating work. Queue entries name the unit, commit or range, requirements and outstanding decisions. Git retains history. Remove resolved state and keep outstanding acceptance checks in [ACCEPTANCE-GATES.md](ACCEPTANCE-GATES.md).

A failed acceptance or provider check belongs in live **Review findings** under [§7](#finding-and-fix-routing). Reconcile every Must fix in a run report before archiving. If a conflicting writer or separate workspace prevents recording it on main, identify where the evidence is and what remains blocked. An archived report alone cannot put a finding in the fix queue.

## 5. Commit and resume safely

Commit each verified slice directly on `main`, including its scoped state and applicable checks. Stage only owned changes. State-only roles commit the handoff and artifacts permitted by their role. An empty queue takes no commit. Push only with authorization covering what will be published.

Continue until the selected unit is complete, the user redirects, or an actual blocker or usage limit intervenes. Keep unfinished edits and leave the handoff resumable; list unverified work explicitly.

## 6. Human update

Tell the operator the outcome, any material risk or verification limit, and the next action and owner when one remains. Include a fix routing recommendation when another agent needs to take the unit.

## 7. Review work

Review another agent's substantive change. Use the named range or an eligible review unit from the handoff. Read its requirements before the diff. Check both directions: promised behavior actually works, and shipped user-visible behavior and architecture are specified. Verify normal-use defects and unnecessary parallel mechanisms at the relevant boundary. Tag findings with matching invariant classes. Challenge materially changed tests against TS-06.R35 and INV §17. Material visual changes require independent [rendered inspection](#144-inspect-the-rendered-result).

### Finding and fix routing

For each finding record location, reachable trigger, consequence, governing requirement when available, and a distinguishing check or suggested repair. **Must fix** covers a likely normal-use failure, data loss or a requirement violation. **Worth fixing** covers other material improvements. Complexity is independent of severity:

- **Trivial/easy**: a local repair at an existing seam, with clear expected behavior and low state or concurrency risk.
- **Medium**: a coordinated repair with known ownership. A fix that changes how concurrent writers preserve user data is at least medium, even when it takes one line.
- **Difficult**: new or uncertain ownership/lifecycle machinery, risky data migration, external protocol uncertainty or cross-system architecture.

Choose the unit's band by its hardest open fix. Write one **Fix model** recommendation using the [model routing table](MODEL-ROUTING.md) in the unit heading and repeat it in the human update. Reassess when its open findings change.

If clean, close this unit and commit its review state. If **all** repairs are trivial/easy, perform [§8](#8-fix-review-findings) in-session and close with verified fixes and handoff in one commit; name the findings and checks there. Otherwise record the **whole unit** in live **Review findings** for `/fix`. Commit only the handoff and leave product code and specs untouched. If prior repairs and current failures share an unresolved ownership or invariant problem, attach an inventory finding with the common cause.

## 8. Fix review findings

Take the named scope or one available findings unit; handle Must fix first. The unit retains its [finding and fix routing](#finding-and-fix-routing). Confirm each finding at the real entry point. Unskip and confirm a committed reproduction where available. For a false or already-fixed finding, remove it with the evidence. Otherwise repair the cause, examine sibling paths that share it and use [write-tests](#17-write-purposeful-tests) for meaningful regression evidence. Update the [spec](#10-keep-specifications-accurate) when behavior changes or coverage was missing.

Run focused checks per repair and [completion checks](#completion-checks) once per unit. For inventory findings, repair urgent loss or misdelivery first, inventory the writers, entry points and divergent rule, and attach remaining same-cause work to one idea for feature design. Other defects remain independent findings.

Use the [independent fix check](#independent-fix-check) when applicable. Remove verified findings and close only their originating unit. Record finding IDs, applicable invariants and checks in the repair commit. A blocked finding stays open. For architectural findings follow the [receipt and release gate](#receipt-and-release-gate): commit the verified repair before recording its exact commit and evidence in the original receipt.

### Independent fix check

Before closing a medium or difficult unit, give a fresh reviewer the finding, requirement and diff, without the fixer's justification. Use a model at the unit’s band or stronger. Check the real boundary, one adjacent consumer/restart path and whether the regression test can fail. Repair confirmed defects; record disputed ones with evidence. If an independent reviewer is unavailable, leave the check owed and the unit open. This check runs once within the unit, without another full suite or review cycle. Architectural receipt findings use §18's verified closure.

## 9. Usability reviews

Usability reviews do not change product code or specifications. Narrow the requested journey or surface to its FS acceptance items, then use the matching charters in [`USABILITY-REVIEW.md`](USABILITY-REVIEW.md); a scoped review does not inherit the full matrix. Confirm user-impact findings in the real product and record them using §7's **Must fix** / **Worth fixing** format. Mark blocked browser or credentialed checks explicitly. Follow §§5–6 for closure; retain a run report when its scope or evidence warrants one. Push only with explicit authorization.

## 10. Keep specifications accurate

Specifications describe shipped behavior and architecture. Requirement IDs are append-only: do not renumber or silently change their meaning. Mark unshipped items `(planned)`; record an explicit deviation when shipped code and a specification temporarily differ.

Tests, commits, and review findings should cite a requirement ID only when that link helps someone find the rule being checked. The specification checker verifies the mechanics of these links, but people still need to judge whether the text and code agree.

Build and fix sessions edit specifications; a review repairing a trivial/easy unit under §7 is a fix session for that repair. Review sessions otherwise report missing or incorrect specification coverage; usability reviews report observed behavior and do not edit specifications.

## 11. Design a feature

`/design-feature` turns one new or recorded idea into work that is ready to implement. It changes
specifications and planning documents, not product code.

Make the lightweight `/ux` trigger check in §15 while shaping behavior. A full UX pass accompanies
this role only when the feature changes an established task flow or introduces an unfamiliar or
consequential decision, ambiguous state, recovery path, long-running operation, or AI uncertainty.
When it runs, apply its task framing before presenting the proposed behavior in step 4. Put durable
results into the planned FS requirements and acceptance items, not into a separate UX artifact. If
the trigger does not pass, continue without recording a UX ceremony or rationale.

1. Use the idea named by the user from any `docs/ideas.md` section, including `Known things to
   improve`. A new prompt can supply the idea directly. Clarify an ambiguous name.
   If none is named, automatic selection is limited to the available or resumable ideas: choose any
   resumable entry under `Ideas being defined` or available entry under `New ideas`; the explicit
   role invocation authorizes that selection. Idea age and other role queues do not constrain the
   choice. Ignore template examples. Ask for an idea if both queues are empty.
2. Move a newly selected idea to `Ideas being defined`, then work with the user to understand its
   outcome, scope, exclusions, edge cases, compatibility, and acceptance criteria.
3. Draft the feature specification first. Extend an existing FS when it already owns the capability;
   create a new one only for a distinct capability. Mark every unshipped requirement `(planned)`.
4. Explain the proposed feature behavior in the conversation before asking for confirmation: state
   what changes, when it happens, where its data lives, what users and agents/API clients observe,
   lifecycle/retention effects, exclusions, and each remaining product decision. Do not ask a
   person to confirm an unspecified “behavior.” Then wait for their confirmation before designing
   the architecture.
5. Draft the matching technical specification using the relevant code, pinned interfaces and
   invariants. Verify an assumed limitation before proposing a workaround. Apply [lifecycle
   design](#lifecycle-design) where relevant. Explain the options and wait for material unresolved
   decisions reserved by [§3](#3-when-to-ask-the-user), including destructive migrations and
   externally visible protocol changes. Choose reversible implementation details within scope.

6. When both specifications are complete, create a descriptive file in `docs/ready-changes/` with
   the exact requirements and acceptance evidence, remove the source idea, and run the documentation
   checks. Do not make the change active in `HANDOFF.md`; implementation has not started. Commit the
   specifications, the ready change, the removed idea, and the state file (§5) together.

If a material decision remains unresolved, leave the idea under `Ideas being defined` and do not
call it ready. Keep partial specifications honest and record what is needed from the user.

### Lifecycle design

When durable state, waits, or effect ownership changes, record the affected transitions in the
change file. Use a state/event table for interacting transitions; give the next state, durable
owner, and observable result. Include only reachable events;
consider restart, Stop/cancel, owner replacement, failed input, quota/timeout, and late callbacks.
An unresolved reachable transition is an open question. When this exposes actual ownership or
recovery risk, have a fresh-context capable subagent review the planned items under §13 before
step 6. Revise within this design unit; retain only unresolved findings in the handoff. A local
state addition without such risk does not require an independent review.

## 12. Investigate a reported bug

Capture the report as given, its entry point, environment, version and logs or absence of logs. Ask for a report if none was supplied. Read the requirements governing the symptom and check history for a repair since the reported revision. If it works as specified, explain the mismatch and route a requested improvement to the ideas list. Record a requirement gap separately from any code defect.

Trace the evidence that distinguishes plausible causes. Try the reported path and environment where available. Label conclusions **confirmed**, **probable** or **undetermined**. Distinguish reproducing a defect from establishing that it caused this incident. If missing diagnostics prevent attribution, record the missing observation and question it would answer.

Keep product and spec files read-only. A reproducible failure may be committed as a skipped regression referencing its finding. Use [finding and fix routing](#finding-and-fix-routing) for one investigation unit plus confidence labels. Commit the handoff and any skipped reproduction.

## 13. Review a design

`/review-design` independently checks another author's waiting change and its cited planned FS/TS items before implementation.
Use the named change, or choose one waiting change when none is named. Follow §1; read the cited
requirements, adjacent shipped contract, and matching invariant classes. Change no product code,
specifications, or change file.

Check three things:
- **Scope:** each requirement serves the confirmed outcome, a real report, or a binding invariant.
  Unneeded modes, abstraction, and hypothetical edge-case machinery are findings.
- **Extension:** reuse an existing capability, seam, route, or storage shape where it fits. A new
  mechanism must justify its contract and maintenance cost.
- **Evidence:** verify claimed capabilities and limitations against the actual code and pinned
  tool. Research newer releases or alternatives when a claimed limitation drives a workaround.
  Check existing guards before claiming a failure.

For lifecycle designs (§11), inspect each reachable transition for state, owner, observable result,
failure, concurrency, and recovery. Also check planned tags, consistency with shipped requirements,
observable acceptance, and reserved human decisions.

Every finding states who encounters what consequence and when, with supporting evidence. Record
real findings in the handoff's **Review findings** using §7; page-only defects belong in **Design
consistency notes** and do not block implementation. Open Must-fix design findings block starting
that change. A round without user-visible or unbuildable consequences closes the review; do not
request another pass for proofreading. Follow §§5–6 for the state commit and human update.

## 14. Design UI

`/design` runs explicitly or accompanies material visual work: new screens, redesigns, meaningful
composition/styling, motion, polish, or critique. Data wiring, behavior tests, copy-only edits, and
style-preserving fixes do not trigger it. It adds judgment to authorized work; it never selects or
enlarges a change. Standalone critique edits no repository files. Implementation follows §§1–6
and §10.

### 14.1 Ground the direction

Read the affected FS/TS requirement IDs, relevant FS-12/TS-08 rules, `ui/AGENTS.md`, affected code,
and one rendered incumbent surface. Inherit the shipped visual language unless redesign is requested.

For a narrow extension, state the intended improvement and important states briefly in the working
plan. For a new screen or redesign, also capture the operator's task, reading order, density, focal
point, product-specific hierarchy, motion purpose, and commitments to preserve. Explore two or three
compositions only when materially different structures remain open. Keep direction in the existing
plan; durable behavior belongs in FS and presentation architecture in TS-08. Critique uses the
requirements and incumbent as its direction without creating a document.

### 14.2 Compose before decorating

Use real content ranges, including long names, dense work, errors, and empty states. Hierarchy should
express sequence, ownership, state, urgency, dependency, or scope. Keep frequent controls and
technical content clear; typography, spacing, alignment, and copy carry most of the finish.

Extend TS-08's tokens, primitives, feature composition, and presentation hooks. Preserve Core and
the declared skins. Do not introduce a framework, theme provider, dependency, token system, or
screenshot baseline merely to carry out this workflow.

### 14.3 Make motion earn its place

Use motion for feedback, spatial continuity, or state comprehension. Frequent and keyboard-driven
actions normally stay instant. Prefer narrow CSS transform/opacity transitions; routine motion is
usually 100–250ms and under 300ms unless the interaction warrants more.

Preserve reading position, make repeated input interruptible, gate hover to capable pointers, and
provide reduced motion. Lifecycle animation belongs to newly observed transitions, never initial
render, reload, refetch, or history replay.

### 14.4 Inspect the rendered result

Material visual changes require the working tree's real UI in a browser; use
[`BROWSER-SETUP.md`](BROWSER-SETUP.md) for isolated setup. The deterministic matrix supports,
but does not replace, inspection of the affected product surface. Choose supported widths from
the affected requirements; include the desktop floor for layout changes and relevant skins when
shared presentation changes.

Check the affected content/state range, keyboard and focus flow, hover feedback, hierarchy,
contrast, clipping, overflow, and stacking. Observe motion at normal speed, repeated/interrupted
input, and reduced motion. Source or static fixtures alone cannot establish these claims.

Name evidence and user consequence. In implementation, repair material in-scope findings and
re-render; stop when the direction and material issues are satisfied. Use independent rendered
critique for a substantial new/redesigned surface when it adds value, not automatically for local
edits. A §7 review records findings under §7; standalone critique reports in conversation only.
If the browser cannot run, leave the visual claim unverified.

## 15. Shape and test the experience

`/ux` runs explicitly or during feature design/implementation when an established flow changes or
there is an unfamiliar/consequential decision, ambiguous state, recovery, long operation, or AI
uncertainty. Ordinary familiar additions, internal wiring, styling, and specification-preserving
fixes do not trigger it. It does not select work or authorize implementation.

### 15.1 Start with the human task

Read the relevant FS acceptance IDs and adjacent shipped experience. Frame the operator, entry
state, intended result, critical decisions, completion signal, and material consequence/recovery
branch. Default to Chuck's experienced internal operators: speed, density, keyboard flow,
predictability, and control. Do not invent novice needs or research evidence.

Walk the risky steps: can this operator find the action, and does feedback explain progress,
consequence, and next action? Name the likely failure before adding a control. Preserve work on
errors; disclose material scope, uncertainty, cost, and permissions; offer recovery when needed.
For CLI work, retain composability, meaningful exit status, concise help, and non-TTY behavior.
Add only mechanisms that resolve a concrete task risk.

### 15.2 Harden behavior before the contract hardens

Use the frame before feature confirmation to resolve hidden prerequisites, unclear states, unsafe
consequences, and dead ends. Put durable behavior and task-based acceptance in FS; keep transient
reasoning in the plan/conversation. Confirm material assumptions about operator habits at the
existing human checkpoints; ask only for decisions those checkpoints reserve.

### 15.3 Exercise the task in the real product

Validate in the real product when interaction, timing, state, recovery, or unresolved risk could
change the judgment; explicit standalone critique also requires this. Reuse
[`BROWSER-SETUP.md`](BROWSER-SETUP.md). Run the primary task and material branch, not a product
tour. Acceptance tests suffice for ordinary behavior outside these risks.

A finding states task/context, observed friction and evidence, consequence, and a proportionate
improvement. Static hypotheses remain risks, not experienced findings. In build/fix work, repair
in-scope findings under the normal specification rules and repeat the affected task. In review,
record findings under §7. Standalone critique reports in conversation and edits no repository
files. If the real path cannot run, state what remains unverified. Stop once the scoped task,
state/consequence comprehension, and material recovery branch work.

### 15.4 Work with `/design` without splitting the experience

UX owns task sequence, language, discoverability, state, consequence, and recovery; design makes
them perceptible through hierarchy, composition, typography, color, and motion. When both trigger,
share one task frame and rendered pass. Record each problem once; visual polish cannot replace a
missing step or recovery path. `/usability-review` owns broad acceptance-journey regression.

## 16. Cut a release

1. On clean `main`, identify the highest merged release tag and the commits since it. Read the live review queue, findings, human blockers and [owed gates](ACCEPTANCE-GATES.md). Report open units for a human decision; send open Must fix findings to §8. Read the relevant release requirements in FS-10 and TS-06.R13–R22.
2. Compare agent-facing changes in the range with the embedded `internal/agentknowledge/operating-chuck/` package. Update affected guidance under FS-18 and TS-11.R1/R8, README install/command claims and release component pins where needed. Record when the range changes nothing agent-facing. Keep product fixes in their own unit.
3. Use the version the human specified, or propose patch for fixes/internal changes and minor for new/changed capability. Obtain version confirmation before tagging. Run missing or invalidated Go/UI checks under [completion checks](#completion-checks), then `make dist VERSION=<version>`. Hosted CI assembles the Darwin archive and private Node runtime.
4. Commit verified preparation. Obtain or reuse the candidate's [architectural review](#18-review-architecture-for-a-reliable-development-hub), then run the [receipt and release gate](#receipt-and-release-gate) before tagging. Recheck after any candidate change.
5. Create an annotated `vX.Y.Z` tag with message `Release vX.Y.Z`. Report the branch, count of all unpushed commits and tag. Pushing main and the tag triggers publication and requires explicit authorization for that scope. Without it, stop at the local tag.
6. After authorized publication, verify release CI, archive, checksum, manifest and installation evidence. Distinguish failed, waived or owed native/provider/device checks from passes. Record failures under §4.
7. Record actual release state and owed gates, archive settled handoff content under `docs/archive/state/HANDOFF-through-YYYY-MM-DD.md`, and keep the injected header under 8 KiB. Commit the scoped state and report what the version delivers and what remains unverified.

## 17. Write purposeful tests

Within work or fix, reuse the parent scope, evidence and closure. Standalone `/write-tests` covers only the requested behavior and follows [completion checks](#completion-checks). Read **R35 and its evidence table** in [TS-06](../specs/tech/TS-06-build-test.md) and the [independent-oracle rule](INVARIANTS.md#17-tests-must-prove-their-contract-independently). Use the smallest check that distinguishes the plausible defect at the promised boundary. Reuse adequate coverage. For regressions, demonstrate pre-fix failure when practical. Isolate user data; credentialed provider runs need their own authorization.

## 18. Review architecture for a reliable development hub

Use a fresh reviewer on the stable committed candidate after release preparation, including fixes-only releases. An explicit narrower audit can contribute bounded evidence; release completion requires combined-system coverage. Start from the previous release, its assessed capability/owner map and valid evidence, or the [core audit map](CORE-RELIABILITY-AUDIT-PLAN.md) for the first run. Read relevant requirements and pinned runtime facts.

Assess recoverable work, faithful delivery, progress, intended execution/workspace, truthful results and control, and operating cost. For each high-consequence scenario identify the real stimulus, promised response, owner and evidence. Trace changed assumptions through unchanged callers, schedulers, repair paths and installed artifacts. Probe consequential uncertainty; measure performance claims under representative workloads and distinguish local, fake and provider evidence.

Apply [finding and fix routing](#finding-and-fix-routing) to live findings. Preserve existing owners; group new cross-system findings as one repair unit. Commit only the handoff, receipt and necessary bounded evidence. Reference every release-blocking finding in the receipt. A completed review can have open findings; §8 closes the repairs in that same receipt.

### Receipt and release gate

Before tagging run `node scripts/check-architectural-review.mjs --release-tag vX.Y.Z` on the committed candidate. Resolve rejected provenance or open findings before proceeding.

Use one JSON receipt at `docs/archive/reviews/architecture/<previous-tag>-next.json` (or
`initial-next.json` without an earlier release). The gate derives the highest merged `vX.Y.Z` tag,
excluding `--release-tag` itself in tag CI. Keep the original reviewed commit after fixes. Example:

```json
{
  "schema": 1,
  "previous_release": "v0.17.0",
  "reviewed_commit": "<full candidate commit ID>",
  "review_complete": true,
  "reviewer": "<independent reviewer>",
  "coverage": ["Saved work: authority, scenario, evidence and limits; other core areas likewise"],
  "findings": [{"id": "ARCH-1", "severity": "must_fix"}],
  "updates": []
}
```

`previous_release` is `null` for the initial case. `coverage` retains the current capability/owner
map, assessed joins, evidence and limits; the gate checks presence, not semantic quality. Findings
are references, with severity `must_fix` or `worth_fixing`; their full reports/live state stay in
the handoff. All Must-fix entries need verified closure before release.

After §8 repairs, the fixer appends an update with `kind: "fix"`, exact full `commits`, the original
`finding_ids`, a `closes` subset, `verified: true`, and concise `evidence` of focused regression and
affected-path checks. Multiple repair commits can share one evidence entry. No architectural
rerun is required to write it. Partial fixes can leave `closes` empty. Unrelated substantive work
uses `kind: "coverage"` with its exact commits and focused independent affected-coverage evidence;
it cannot close findings or be silently admitted as a fix.

Only the receipt directory, live handoff, owed-gates file, and specifically named
release-note/handoff archive files are administrative allowances. Every other post-review commit needs exact verified update coverage.
The gate reads committed candidate files, checks prior-release/review ancestry and rejects open
live Must-fix findings. Committing the receipt or later finding closure does not invalidate itself.
It does not infer fix intent from commit subjects or broadly allow all files a fix happened to touch.

**No review loop:** architectural review → findings → fix/verification → recorded closure → release.
The gate never demands a second architectural review solely because those fixes occurred. A later
release has its own review. Owed native/provider/device gates retain their existing policy and
visibility; an actually failed gate remains a finding. Publishing authorization stays under §§5/16.
