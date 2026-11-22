# Chuck handoff — settled through v0.25.0 (archived 2026-11-22)

Non-authoritative history moved out of the live handoff when v0.25.0 was released.

## Position before the v0.25.0 release

- **Release:** [v0.24.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.24.0) published
  on 2026-11-16 at `4d511e1d`. The local build and 1,293 UI tests passed.
  [Release CI](https://github.com/AsaphNoam/Chuck/actions/runs/43819855461) passed assembly,
  transaction/bootstrap and fresh installation. The published archive's size and SHA-256 match
  the manifest, and the installer is attached.
- **Architecture:** v0.24.0 shipped under the
  [v0.24.0-only waiver](../archive/state/RELEASE-v0.24.0-notes.md). The owed combined review of
  `v0.23.0..d26e5fe2` is now recorded in the
  [receipt](../archive/reviews/architecture/v0.23.0-next.json); it did not gate v0.24.0. Its fixes
  are recorded there. The operator resolved EXEC-6c by retaining the existing shrink-only
  test-fixture exceptions in TS-01.R50. The next-release source assessment at `96c0b4a5`
  is recorded in [v0.24.0-next.json](../archive/reviews/architecture/v0.24.0-next.json).
  Its ARCH-v025 findings are fixed and recorded there; QUOTA-DISMISS stays open below as
  Worth fixing. The release review is complete: on 2026-11-22 a fresh reviewer covered
  `2997aadb`, `04305b5d`, `a644d024` and the `45a5c819` preparation with no new finding.
- **Verification:** [Main CI](https://github.com/AsaphNoam/Chuck/actions/runs/43819859225)
  was still running when release CI completed. The prior waiting-input UI fixture failure had a
  local repair. Native, provider and device gates in [ACCEPTANCE-GATES.md](ACCEPTANCE-GATES.md)
  remain owed.
- **Installed runtime:** managed current was v0.20.0 at audit; no install or running-dashboard
  update is part of this release preparation. Global PATH adapters remain unpatched.
- **Checks:** unset CHUCK_RUNTIME_ROOT; use GOFLAGS=-p=1 and a writable isolated GOCACHE.
  Full Go suites need loopback-capable execution.

## Resolved decisions settled by v0.25.0

- Inline task delivery and guidance cleanup (FS-16.R60–R61, FS-18.R22–R23) keeps native-command
  handling on fresh launches and attributed content on continuations. Implemented 2026-11-21;
  review repairs finished at `10d01e1b`.
- Project Files starts with all shared folders and offers a folder selector. HTML opens in a
  sandboxed viewer in a new browser tab. The shared folder is desktop-only and symbolic links
  cannot be followed. Shipped 2026-11-20 (FS-11.R11–R16).
- Message action design is ready in
  [message-hover-actions-and-current-conversation-edit.md](../ready-changes/message-hover-actions-and-current-conversation-edit.md)
  (FS-01.R60–R62/A42–A44). The last sent message has Copy/Edit/Fork; other messages have
  Copy/Fork as top-right hover icons. Fresh-context design review found no material findings
  on 2026-11-20; the change is waiting to start.
- WITHDRAW-SEND-3 and STEER-HOLD-4: show assignment-queued input once, labelled as queued for
  the assignment's next turn. Implemented under FS-03.R48 (2026-11-20): it sits at the
  transcript tail, the empty composer hides Steer when only assignment-queued input is pending,
  and runtime-held input keeps its Withdraw action.
- EXEC-6c: TS-01.R50 permits the 11 existing test-only store writers under the shrink-only
  `fixtureOnlyWriters` list. The archived architecture receipt records closure at `596821bf`.
- AUTH-TRUST-2: the assignment heading now says “Why this stage visit started.”


## Finished review units

- **Browse project files** (FS-11.R11–R16, FS-03.R119–R120, TS-03.R93–R94, TS-05.R30–R31,
  TS-08.R172–R174): reviewed `8cba163c^..f2239ad0` at `f2239ad0`. **Finished** on 2026-11-20
  by the PROJECT-FILES fix commit. PROJECT-FILES-7's inventory and remaining owner, focus and
  menu-claim gaps are in `docs/ideas.md` as “One owner rule for the file reader, its menus and
  focus”. The 1–4 repairs are proven in jsdom and API tests, not re-run in real Chromium. Native
  Finder selection remains owed in ACCEPTANCE-GATES.

- **Message hover actions and current-conversation Edit** (FS-01.R60–R62, FS-22.R13,
  TS-01.R51, TS-02.R69–R70, TS-03.R90–R91, TS-04.R113, TS-08.R171): reviewed at
  `b2986ebf` for `99e4672b`, `1bfa4d28`, `b16c722f` and `b2986ebf`. **Finished** on
  2026-11-20 by the EDIT-REVIEW fix commits (`e31222f9`..closure). EDIT-REVIEW-4, EDIT-EDGE-4
  and the inventory's remaining gaps (room ownership, dispatch failure, Stop before dispatch)
  are in `docs/ideas.md` as “One commit point for an Edit's history, session and replacement”.
  Packaged live provider receipts, including two successive edits per adapter, remain owed in
  ACCEPTANCE-GATES.

- **Inline task assignments and concise guidance** (FS-16.R60–R61, FS-18.R22–R23, TS-01.R52,
  TS-10.R63–R65, TS-11.R25–R26): reviewed `c872c240` and `7965a17c` at `7965a17c`.
  **Finished** on 2026-11-21 by `10d01e1b` and closure. INLINE-TASK-5–7's
  inventory and remaining guidance/payload-bound gaps are in `docs/ideas.md` as
  “One rule for task execution across continuations”. The packaged Claude/Codex gate remains owed.


## Closed investigation

### TASK-PROMPT — reporting paragraph after a skill invocation (investigated 2026-11-21)

**Report (verbatim):** “I specifically told you to only trigger the instruction skill and yet
you still wrote an entire wall of text”. The annotation quotes transcript event 1 of
`a_06bd8f`: `/fix` followed by “This is your Chuck task” and the reporting/coordination paragraph.
The request was to launch one Claude Opus Medium agent at a time with only the fix skill.
No separate diagnostic log or exact installed version was supplied.

**Confirmed — works as specified, not a code defect.** The durable task
`tk_e67d15ecd4cbe7df` stores `instruction: "/fix"`, `backend: "claude"`, `model: "opus"`,
`effort: "medium"`. The orchestrator supplied no extra assignment prose. Chuck appends it:
`Registry.StartLaunchedTask` (`internal/runtime/registry_message.go:22`) calls
`LaunchedTaskPrompt` (`internal/runtime/activation_kinds.go:32`), which preserves the instruction
and adds the shared `taskReportContract`. FS-16.R49 explicitly requires this paragraph and A33
requires it in the first prompt. `TestALaunchSpecTaskStartsItselfAndBecomesRunning`
(`internal/server/task_dispatch_test.go`) covers the delivered instruction and reporting contract;
it was inspected, not rerun. The annotation itself is the live reproduction. History traces the
contract to `4ddedff1` and the expanded coordination guidance to `a674e6bc`.

No investigation fix finding: the shipped behavior matched FS-16.R49 and INV §21. The approved
replacement (FS-16.R60, FS-18.R22) was implemented on 2026-11-21 and removed the appended
paragraph; see the “Inline task assignments” review unit.

