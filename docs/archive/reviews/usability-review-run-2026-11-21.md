# Usability review run — 2026-11-21

## Scope and outcome

- **Scope:** full top-to-bottom J1–J17 regression, S1–S5 static sweeps, and four uncovered-surface
  charters derived from acceptance items: X1 Think Tanks (FS-21), X2 Project Files (FS-11), X3 context
  links (FS-15), X4 phone remote control (FS-20), X5 message actions/turn control (FS-01, FS-03),
  X6 quota display (FS-09), X7 operator CLI (FS-22).
- **Baseline:** `10d01e1b` on `main`, built in a detached worktree (`/tmp/chuck-ur-2026-11-21/src`)
  so the working tree's uncommitted state was untouched.
- **Outcome:** every journey driven. No BLOCKER. Five MAJOR (one unconfirmed), about forty MINOR.
  Of the 2026-10-15 findings, these are fixed: duplicate chat bubbles, live mail badges, mail-turn
  explanation, restart-paused pipeline dead end, the tray launching before validation, the pipeline
  500, blank archive page, crash shown as "Stopped", blank backend name, wizard relative-path warning,
  load-time layout write and reshuffle, stopped terminal page, silent worktree recreation, attention
  count on delete, cancelled-task signal wait, project ids in the task form, reload landing
  mid-transcript after diagrams, and the archived-source default target. Still present: no
  confirmation on run/task Delete, generic parked reason, raw field keys in task errors, Start-run
  dialog fade under reduced motion, OpenCode/OpenHands guidance without an install command.
- **Top 5:** (1) a person-authored stage success is accepted and wedges the pipeline run; (2) run
  timeline shows no per-attempt stage/delegated agent cards, so earlier attempts' transcripts are
  unreachable from the run; (3) added (`+`) diff lines cannot be annotated; (4) `README.md:12`-style
  file links render dead; (5) one Claude→Codex switch returned 500 silently and left the agent
  half-switched (single occurrence).
- **Interruptions:** seven batches hit a usage limit once and resumed from their saved state.

## Setup

- **Builds:** `npm ci && npm run build`, embed copy, `make build` (`sqlite_fts5`) plus untagged
  `go build`; fake ACP built from `internal/runtime/testdata/fakeacp` on `PATH` as `claude-agent-acp`
  and `codex-acp`. X4 used a `-tags "sqlite_fts5 dev"` build from the same snapshot with the fake
  tailnet (the shipped binary cannot join a tailnet). J6 used a local `claude` stub.
- **Fixtures:** `fresh` (empty), `seeded` (onboarded; project `review` on a scratch Git repo; scenarios
  mapped per model: sonnet stream_text, opus permission, haiku diagram_stream, fable hold_turn),
  `lived-in` (seeded plus Alpha/Bravo/Charlie stopped, Delta/Echo archived, two turns each). One home
  copy and port per journey.
- **Browser rung:** 1 — Playwright Chromium, 1280×800 unless noted; 390px phone viewport for X4/X6.
- **Repo changes:** none to product code or specifications.

## Journey matrix

| Journey | Result | Coverage and remaining branch | Evidence |
|---|---|---|---|
| J1 Install & first paint | PASS | Fresh home, zero console errors, styled shell, `layout.order` `[]`. With a working Claude CLI the wizard never appears and a placeholder "My App" project exists (J2 finding). | [first paint](usability-review-2026-11-21-evidence/B1-J1-1-first.png) |
| J2 Onboarding | PARTIAL | Fake HOME: missing CLI, Codex sign-in, OpenCode/OpenHands, stubbed logged-out/logged-in Claude, validation, Escape guard, Set up later, relative path warning, Launch, restart. Real-provider branches SKIPPED (no credentials). Findings below. | [after launch](usability-review-2026-11-21-evidence/B1-J2-10-afterlaunch.png) |
| J3 First launch + chat | FAIL | Launch, streaming, one bubble live and after reload, busy/idle, Mermaid streaming/toggle/reload/archive/Sky & Grove, inert injection, file links (line mark, rendered/source, `?file=`, Close, absolute/`../`, missing file, pane link), tools, scroll pinning, queued input, Cancel, Copy, archive. Bare `name:line` links dead. Edit/Fork BLOCKED here (covered in X5). Thinking too brief to observe. | [dead link](usability-review-2026-11-21-evidence/B2-J3-14-links2.png) |
| J4 Permission flow | PASS with findings | Prompt, reload while pending, approve (double-click once), deny, 45 s timeout, cancel. Toast duplication findings. Stop-while-pending inconclusive. | [denied](usability-review-2026-11-21-evidence/B2-J4-8-denied.png) |
| J5 Grid & layout | PASS | Titles, meters incl. zero, Resume only when stopped, reorder/density/groups/collapse across reload and restart, archive of ordered agent, empty state, active-first incl. refused cross-boundary drop, chat panes, Ctrl+Alt+Arrow, auto-open with eviction and draft return. | [eviction](usability-review-2026-11-21-evidence/B3-J5-16-evict.png) |
| J6 Terminal runtime | PASS with findings | Stub CLI: render, typing/backspace, PTY resize, reattach, Stop/Resume, stopped page. Shrink re-fit and silent Resume refusal. | [shrink](usability-review-2026-11-21-evidence/B3-J6-3c-shrunk-1100.png) |
| J7 Stop / resume / switch | PASS with finding | Resume, model and backend switches (dump-verified cwd/add_dirs/system prompt), stop idempotence, rename, composer wake, clone, restart identity. One unexplained 500. | [switch 500](usability-review-2026-11-21-evidence/B4-J7-6-switch-codex-post.png) |
| J8 Archive & search | PASS with findings | Both builds: empty, one, 60 with paging, transcript (tagged) and metadata (untagged) search, read-only view, Restore, bogus id, missing transcript. | [archive](usability-review-2026-11-21-evidence/B4-J8t-1-archive.png) |
| J9 Settings & config | PASS with findings | Seven tabs round-trip and merge-preserve; appearances live and persisted; Remote panel; config-source link/unlink. Override apply and role delete-in-use not reached. | [model error](usability-review-2026-11-21-evidence/B1-J9-3-emptymodel.png) |
| J10 Messaging | PASS | Real `/mcp` send, wake/no-wake, live Mail/Sent badges, read clearing, Markdown mail detail, failed-send reason, "Mail from" notice, stopped-recipient wake. Busy recipient not exercised. | [mail detail](usability-review-2026-11-21-evidence/B5-J10-5-B-expanded.png) |
| J11 Failure & recovery | PASS with findings | Server kill/reconnect, send while down, crash and kill -9 → Error with restart hint, garbage input (API and UI), stop non-running. | [reconnected](usability-review-2026-11-21-evidence/B4-J11-1d-reconnected.png) |
| J12 Restart durability | PASS | Homes left by J5, J7, J10, J14, J15, X1 restarted twice: project page, Tasks and Archive text identical, no page errors. | [before](usability-review-2026-11-21-evidence/J12-J5-a_project_review.png), [after](usability-review-2026-11-21-evidence/J12-J5-b_project_review.png) |
| J13 Annotate & assign | FAIL | Whole-event, text-range and removed-line annotations, tray persistence, self/other/new-launch delivery (validation before launch), archive search, archived default target. Added-line annotation impossible. | [line select](usability-review-2026-11-21-evidence/B5-J13-3a-line-selected.png) |
| J14 Pipelines | FAIL | Runs/Templates split, four-stage template, CLI/UI starts, recovery controls, restart pause, Stop/Delete, legacy URL, 123-run ledger, reduced motion, 32-stage proposal lifecycle. Routing/approval/repair loop BLOCKED (fake never reports results); max-shape and background refresh SKIPPED. | [wedged](usability-review-2026-11-21-evidence/B6-J14-5-wedged-run.png), [no cards](usability-review-2026-11-21-evidence/B6-J14-9-attempt1-open.png) |
| J15 Dependent work | PASS with findings | Every charter step incl. live armed→running, failure park, re-arm, restart interrupted, retry, garbage. | [armed](usability-review-2026-11-21-evidence/B7-J15-3a-armed.png) |
| J16 Attention, budget, worktrees | PASS | Live count up/down, link-through, budget refusal/round-trip/merge, budget-1 queueing, canvas right-click everywhere, worktree offer/create/recreate (toast)/archive. | [recreated](usability-review-2026-11-21-evidence/B7-J16-7a-relaunch-800.png) |
| J17 Browser-local drafts | PASS | Two isolated drafts across navigation/reload, sent draft cleared, fresh context empty, stop leaks nothing. | — |
| X1 Think Tanks | PASS with findings | Create, template save/load/repair, failed-turn Retry, pause/message, allowance end, restart, independent openings, delete (turns script-driven; the fake never calls room tools). Judge, attachments, raise-limit SKIPPED. | [ended](usability-review-2026-11-21-evidence/B8-X1-19-ended.png) |
| X2 Project Files | PASS with finding | Folder selector, search, type filter, empty/unreadable/removed, symlink and outside-root refusal, Markdown/CSV/binary, HTML sandbox tab, Copy path. Finder reveal SKIPPED. | [readme](usability-review-2026-11-21-evidence/B8-X2-10-readme.png) |
| X3 Context links | PASS | 15 internal hrefs resolve; room/agent/task/archive cross-links; missing targets show clear pages. FS-15 MCP A-items not exercised. | — |
| X4 Phone remote | PASS with findings | Dev-build fake tailnet, Android UA at 390px: pair/approve/decline/revoke, agents, chat, permission, hold/steer/cancel, quota, restart, iPhone install gate. Physical device not used. | [declined](usability-review-2026-11-21-evidence/B9-X4-18-declined.png) |
| X5 Message actions | PASS | Edit last (twice), Fork from here, disabled fork reason, queue/withdraw, steer, cancel. | [edit](usability-review-2026-11-21-evidence/B9-X5-5-edit-after.png) |
| X6 Quota display | PASS with finding | Complete, partial, null, unavailable, empty, failed-with/without prior values, unsupported adapter, at 1440 and 390. Codex shape and passed reset not reached. | [stale](usability-review-2026-11-21-evidence/B9-X6-7-stale-after-failure-desktop.png) |
| X7 Operator CLI | PASS with findings | No-dashboard errors (exit 3), agent launch/send/transcript/rename/archive/restore, archive search, task/project errors, `--yes` guards. Pipeline/task flows only probed. | terminal only |

## Static sweeps

Risk leads only, none promoted without reproduction: `tmux send-keys -l` without `--` (text starting
with `-`; tmux absent here), empty command name in runtime start errors, swallowed Edit
acknowledge/status-poll errors (`MessageEditor.tsx`), `pipeline-state-stopping` unstyled, task
`outputs` omitempty guarded only by the schema. S2 found no undefined class.

**Matrix coverage (§7):** X1–X7 have no §3 charter row; FS-21, FS-11, FS-20, FS-22, quota and message
actions need charters so the next run does not derive them again.

## Findings

```text
SEVERITY: MAJOR
WHERE: J14 recovery (home B6/J14, port 4760), FS-14.A42/R67
REPRO: Start a dedicated-stage run; wait for "owner not reporting"; `chuck task result <stage task> --outcome success --text done`.
EXPECTED: Person-authored stage success refused.
OBSERVED: Accepted; stage task finished/SUCCESS, run stays paused "owner not reporting"; Continue/Retry/Replace all ineligible — only Stop run remains.
EVIDENCE: usability-review-2026-11-21-evidence/B6-J14-5-wedged-run.png
```

```text
SEVERITY: MAJOR
WHERE: J14 run page (run with Replace attempts 1 and 2), FS-14.R39/A17
REPRO: Open the run page after a Replace and expand both timeline entries.
EXPECTED: Each attempt shows its stage-agent card plus delegated-agent cards linking to live or archived transcripts.
OBSERVED: No cards on any entry; only the run-level Open agent for the current owner. Attempt 1's agent is unreachable from the run.
EVIDENCE: usability-review-2026-11-21-evidence/B6-J14-9-attempt1-open.png, B6-J14-9-run-B.png
```

```text
SEVERITY: MAJOR
WHERE: J13 step 1 (home B5/J13, port 4751), FS-13.R1/R2
REPRO: Open a transcript diff; click the line number of an added "+" line.
EXPECTED: Line selected, "Annotate lines" appears.
OBSERVED: Nothing. In unified mode react-diff-viewer passes "undefined-N" for added rows (renderLine with null prefix), which DiffBlock's /^[LR]-\d+$/ rejects; new-side anchors cannot be created.
EVIDENCE: usability-review-2026-11-21-evidence/B5-J13-3a-line-selected.png; ui/src/components/chat/renderers/DiffBlock.tsx:17
```

```text
SEVERITY: MAJOR
WHERE: J3 file links (home B2/J3, port 4720), FS-03.A34/R51
REPRO: Agent replies with [x](README.md:3); click it.
EXPECTED: A file link opening the viewer at line 3.
OBSERVED: Rendered as <a> without href; click does nothing. README.md, ./README.md:3 and src/main.go:2 work.
EVIDENCE: usability-review-2026-11-21-evidence/B2-J3-14-links2.png
```

```text
SEVERITY: MAJOR (unconfirmed — one occurrence, three retries succeeded)
WHERE: J7 switch (home B4/J7, port 4740), FS-01/FS-09 switch
REPRO: Right-click a running opus agent → Switch runtime → Codex → GPT-5.5.
EXPECTED: Success, or the dialog shows the error.
OBSERVED: POST /switch-runtime 500 after 3.1 s with no server error line; dialog stayed open silently; record already codex/gpt-5.5 and Stopped while the card still read claude opus.
EVIDENCE: usability-review-2026-11-21-evidence/B4-J7-6-switch-codex-post.png; J7.log line 231
```

MINOR findings (each reproduced once and confirmed once unless marked) are listed with repro in the
handoff entry for this review; their evidence files are in the evidence directory under the batch
prefix (B1 J1/J2/J9, B2 J3/J4/J17, B3 J5/J6, B4 J7/J8/J11, B5 J10/J13, B6 J14, B7 J15/J16, B8 X1–X3,
B9 X4–X7).

Not promoted: a fixture-made shared-folder permission error (X2), a task assigned to an agent already
busy in a held room turn (X3), the 4317 default port cross-talk beyond CLI reads (X7 recorded as
MINOR), POLISH items (eviction silent but spec-permitted, "No running agents" on an empty project,
TIMED OUT chip colour, two Withdraw buttons, phone quota disclosure without arrow, Jump-to-latest over
the phone composer, empty wizard DETAILS, colour swatch preselection, folder selector narrowing,
"Event 1" labels, batched mail notice separator, CLI placeholder help text and raw JSON output,
proposals above the template library, Customize runtimes below the fold).
