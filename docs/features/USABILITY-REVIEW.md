# Chuck — Usability Review Protocol

**Verification protocol for scoped or broad behavior-driven usability review.** It derives
expected behavior from feature-spec acceptance criteria, builds the real binary, starts fresh state,
and drives selected journeys through the actual UI. It complements the static diff review in
[`AGENT-WORKFLOW.md`](AGENT-WORKFLOW.md) §7; it is not a third product-spec set.

> Claude Code reaches this via the `/usability-review` skill; Codex via
> `.agents/skills/usability-review`. Both land here. The workflow governs role behavior; feature
> specs govern expected product behavior; this file governs the verification method.

**Prime directive: user-impact findings are confirmed against the running app. Static sweeps produce
risk leads, not product findings, until reproduced or routed to the §8 code-review path.**

---

## 1. Scope, bar, and severity

**The bar:** *a first-time or daily user hits this.* If an observed normal-use expectation is not
covered by an FS requirement/acceptance item, note that the requirement may be missing alongside the
finding so the fix can decide whether behavior or the specification must change.

**Severity taxonomy** (every finding gets exactly one):

| Severity | Meaning |
|---|---|
| **BLOCKER** | A core journey cannot be completed (crash, dead UI, wrongly failed gate, unusable surface). |
| **MAJOR** | The journey completes but with data loss, wrong/stale information, or a dead-end the user can't self-diagnose. |
| **MINOR** | Friction: unclear copy, missing feedback, surprising-but-recoverable behavior. |
| **POLISH** | Cosmetic. Report only when trivially fixable. |

When recording a finding: BLOCKER/MAJOR → **Must fix**; MINOR → **Worth fixing**; POLISH is worth
recording only when the fix is obvious and tiny.

**Non-goals** (do not spend tokens here): code-quality opinions, naming/style, performance
micro-audits, redesign proposals, or unverified feature ideas. The §8 diff review and
`INVARIANTS.md` own code-level classes; this review owns *experienced behavior*.

---

## 2. Ground rules — evidence discipline

Use [BROWSER-SETUP.md](BROWSER-SETUP.md) for the real build, isolated home, fixtures, and browser.
Select only journeys and state variants relevant to the requested FS acceptance IDs or surface.
An explicitly broad review covers the matrix; a focused review does not require every fixture,
static sweep, provider, or search build. Exercise the untagged fallback when search behavior is in
scope. Real-CLI branches without credentials are `SKIPPED(reason)`.

Every finding includes fixture, steps, expected/observed result, and evidence. Confirm normal-use
impact in the running product; source-only risks route to code review. Do not change product code
or specifications.

---

## 3. The journey matrix

The methodical top-to-bottom core. Each journey is a **charter**: a self-contained brief with an
entry fixture, ordered steps, expected observations, and known risks. Where a journey
touches collections, exercise the **state variants**: *empty / first item / many items / stale
(restart) items*. The fresh-install escape was structurally "the empty variant was never exercised" —
treat the empty variant as mandatory, not optional.

| # | Journey | Fixture | What must be observed |
|---|---|---|---|
| J1 | **Install & first paint** | `fresh/` | Build succeeds via the real build line; server starts; UI loads in a browser with **zero console errors** and a **styled** shell (computed styles on key elements, not just DOM presence). This journey covers the historical build and missing-style escapes; J2 covers CLI-credential variance. |
| J2 | **Onboarding wizard end-to-end** | `fresh/` | Every branch: no CLI installed → clear guidance; CLI installed but not logged in → failed check with actionable detail; logged in → pass (ENV-DEPENDENT). Back/skip/validate paths. Resulting config files on disk are sane and re-readable. |
| J3 | **First agent launch + chat round-trip** | `seeded/` + fakeacp | Create agent, send a message, streamed response renders, card status transitions (idle → busy → idle) match reality. Diagram rendering (FS-03.R37): with the `diagram_stream` fakeacp scenario, have the agent reply with a fenced `mermaid` block, confirm it stays a code block while streaming and becomes a themed diagram once the fence closes, toggle it back to source, reload and reopen the archived transcript to confirm the same result, and check it under Core and Sky & Grove. With the `diagram_injection` scenario, confirm the script element and the HTML label reach the page as inert text with no request. File links (FS-03.R51-R55): have the agent reply with a Markdown link to a repository file carrying a `:line` anchor, and confirm activating it opens the file viewer docked left of the transcript instead of navigating to the dashboard, that the cited line is marked, that a Markdown file toggles between rendered and source, that a reload reopens the same file from `?file=`, that Close clears it, that narrowing the window moves the viewer to the transcript-width form without losing the open file, and that a link to a path outside the working directory refuses with a stated reason while the transcript stays readable. Repeat one link from an expanded dashboard chat pane and confirm it opens the agent screen with the file showing rather than a panel inside the pane. |
| J4 | **Permission prompt flow** | `seeded/` + fakeacp | Prompt appears; approve, deny, and timeout each leave UI and server state consistent; no double-fire, no stuck prompt. |
| J5 | **Grid & layout** | `seeded/` | Verify each card shows the configured project title (not its durable id) and every context meter, including zero, visibly identifies its percentage. Right-click a stopped card and Resume it; the menu must not offer Resume while it is running. Reorder, density, groups, collapse; persists across page reload **and** server restart. Delete an agent that is in the saved order — grid stays sane. Empty grid renders a real empty state. Once active-first ordering ships (FS-02.R45): in a group holding both running and stopped agents, confirm running cards precede stopped ones with manual order kept inside each block, stop a running agent and watch only that card move across the boundary with no reload, then drag inside a block and confirm cards in the other block hold position and the new order survives a reload; finally drop across the running/stopped boundary and confirm the displayed and reloaded order do not change. Once expandable chat panes ship (FS-02.R46–R52): expand a chat card and confirm the pane holds a fixed computed height, that streaming a long reply scrolls only that pane's transcript while the page and grid scroll positions stay put, and that the collapsed cards sharing its row keep their own height and position; click Send, decide a permission request, accept an `@` completion, drag a text selection, and right-click inside the transcript, confirming none of them collapses the pane or opens the card menu; then cycle focus between panes with `Ctrl+Alt+ArrowDown`/`ArrowUp` and confirm the target composer takes focus and is scrolled into view. Once automatic pane opening ships (FS-02.R61): with the dashboard on screen, drive a chat agent into a permission request and confirm its pane opens by itself, that only the rows below it move and no card changes column, and that reloading with that agent still waiting opens nothing; then with four panes already open, drive a fifth agent into a permission request and confirm the evicted pane's disappearance is visible rather than silent and its unsent draft returns when it is expanded again. |
| J6 | **Terminal runtime** | `seeded/` | Launch terminal agent (xterm.js path), type, resize, detach/reattach; output intact, keystrokes not eaten. |
| J7 | **Stop / resume / switch** | `lived-in/` + fakeacp | Each verb preserves identity, model, system prompt, add_dirs — observed from the UI and the running process, not the code (INVARIANTS §2 as behavior). |
| J8 | **Archive & search** | `lived-in/` | On the FTS5 build **and** the untagged fallback build: empty archive, one session, many sessions; search returns the right sessions; resume from archive works. |
| J9 | **Settings & config editing** | `seeded/` | Every form: edit, save, reload — saved values round-trip; seeded collections are merge-preserved, never replaced (INVARIANTS §3 as behavior); invalid input surfaces an error, never a silent no-op. |
| J10 | **Multi-agent + messaging** | `seeded/` + fakeacp ×2 | Two agents; send_message, nudge wakes the recipient, unread badges appear and clear correctly. |
| J11 | **Failure & recovery** | `seeded/` + fakeacp | Kill the server mid-session → UI shows disconnect, reconnects with accurate state (SSE reset, INVARIANTS §1). Kill the agent process → card reflects the crash. Submit garbage into every form. Stop a non-running agent. |
| J12 | **Restart durability** | state left by J3–J10 | Restart the server; everything the UI showed before (agents, layout, archive, unread state) is still true after. |
| J13 | **Annotate & assign** | `lived-in/` + fakeacp ×2 | Select diff lines and a transcript event in both live and archived views; edit/reload the pending tray; send to self, another chat agent, and a prefilled new launch; verify cards, mail, nudging, and archive search. |
| J14 | **Configurable pipelines** | `seeded/` + fakeacp | Create/edit a four-stage template; start runs with mixed/reversed stage runtimes; observe sequential result/quiescence routing, approval/blocked recovery, repair-loop bounds, shared-workspace confirmation, notifications, restart recovery, run deletion, and links to every retained stage transcript. Exercise Create with Chucky as an ordinary builder and verify exact Save/Start proposals require fresh one-time confirmation after edits. Once the split surface ships (FS-14.R35–R45): land on Runs, confirm Templates shows no run history and Runs shows no editor, start a run through Setup/Runtimes/Review and arrive on its own page, read a repair loop as repeated compact timeline entries in execution order, open a stage agent and a delegated agent's card, and confirm a shared link to the old `?run=` URL still resolves. Repeat with maximum-shape template/run fixtures and 121 retained runs: the focused editor mounts one stage form, primary run state/actions remain ahead of secondary detail, More runs reaches an exact end, and background refresh preserves rendered content. Run the route/dialog/disclosure/live-entry sequence once normally and once with reduced motion, confirming the latter removes entrance/transform motion without losing state or control. With a 32-stage draft proposed, confirm the offer lists collapsed to its kind, title, stage count, and age with the template library still the surface's first content; expand it to the exact payload and collapse it again; reject it and find it under Declined with its decline time and a Delete; delete it; and confirm a reload returns every remaining proposal to collapsed. |
| J15 | **Dependent work** | `seeded/` + fakeacp | Empty Tasks view; create a task through the form; arm a second on the first and confirm the row says what it waits on in recognisable terms; satisfy the prerequisite and watch `armed → running` with no reload; cancel; fail a prerequisite and confirm the dependent parks with an actionable reason; retry an `interrupted` task and re-arm a parked one; delete; restart the server and confirm armed work is still armed and running work becomes `interrupted`; submit garbage into the create form. |
| J16 | **Task attention & budget** | `seeded/` + fakeacp | On a scoped project dashboard: the needs-attention count reads zero, increments live when work parks, links through to the filtered Tasks view, and decrements when resolved. In Settings → Tasks the concurrency budget shows its default, refuses a non-positive value without persisting it, round-trips across reload, and merges rather than replaces the rest of the config; queue more work than the budget and confirm only the budget runs while the rest wait visibly. Right-click the projects canvas background — including the shell's padding frame on every edge and corner — and confirm **New project**, while a card right-click still opens the card menu (FS-02.A24).  Worktree projects (FS-19, FS-02.R60): right-click a repo-backed project card and confirm **New worktree project** is offered, that a project outside a Git repository offers none, and that the same action is on the scoped project header. Create one and confirm the pre-filled branch tracks the title until edited, the base is the repository's default, the new card appears without a manual refresh showing its branch, and an agent launched there runs inside the new checkout. Delete that checkout from a terminal and relaunch: it is recreated on the same branch and the start says so. Archive the fork and confirm the dialog defaults to keeping the checkout, states the branch survives either way, and discloses uncommitted work; accept and confirm the directory is gone while the branch remains. |
| J17 | **Browser-local drafts** | `seeded/` + fakeacp ×2 | Type unsent text in two chats; confirm each survives navigation and a full page reload and restores only in its own chat; send one and confirm it clears and never resurrects after navigation or reload; stop an agent and confirm no draft leaks. This is the manual reload/navigation check FS-03.A19 names. |
| J18 | **Think Tanks** (FS-21) | `seeded/` + fakeacp ×2 | Empty list; create a room, save it as a template, load and repair the template, and confirm loading names which existing agents it reuses or creates. Run turns; Pause during a turn and confirm the pause is acknowledged and lands after that turn; message a paused room; fail a turn and Retry it, confirming the ended room no longer lists that turn as unfinished; reach the allowance end; restart the server mid-room; open two rooms independently; delete. Where the fake cannot drive room tools, script the turns and record the branch as script-driven. |
| J19 | **Project Files** (FS-11.R11–R16) | `lived-in/` | Folder selector, search and type filter on an empty, a normal and a many-file project; unreadable and removed files; a symlink and an outside-root path refuse with a reason; Markdown (with a table and fenced code) renders at full width, CSV and binary files show their own forms; HTML opens sandboxed in a tab; Copy path; Finder reveal (native, ACCEPTANCE-GATES). |
| J20 | **Phone remote control** (FS-20) | `seeded/` + fake tailnet, 390px phone UA | Pair, approve, decline and revoke, confirming a declined code and a wrong code say different things; agent list, chat, permission decision, hold/steer/cancel, quota, server restart while paired, iPhone install gate. A physical device is an ACCEPTANCE-GATES item, not this journey. |
| J21 | **Operator CLI** (FS-22) | `seeded/` and a home with no dashboard | With no dashboard running, every command exits with the documented code and never reads another home's dashboard; agent launch/send/transcript/rename/archive/restore; archive search; task and pipeline flows end to end; unknown names, stray positional arguments and API refusals print person-readable errors with a non-zero exit; `--yes` guards. |
| J22 | **Subscription quota** (FS-01.R40) | `seeded/` + quota fixtures | Complete, partial, null, unavailable, empty and failed-with/without-prior-values quotas at 1440px and 390px; expanding the quota on the agent page keeps the transcript usable at 900px height; a disabled-continuation banner can be dismissed; a passed reset and the Codex shape. |
| J23 | **Message actions** (FS-01.R60–R62, FS-22.R13) | `seeded/` + fakeacp | Edit the last message twice in a row; Fork from here; a disabled fork names its reason; queue and withdraw; steer; cancel. Repeat Edit on each packaged adapter where credentials allow (ACCEPTANCE-GATES). |

**Charter shape** (what the orchestrator hands each journey subagent — nothing more):

```
JOURNEY: J5 Grid & layout
FIXTURE: <path>   PORT: <port>   BINARY: <path>   FAKEACP: <env recipe or n/a>
STEPS:
  1. <step> → EXPECT <observable>
  2. ...
KNOWN RISKS: <the 1–3 invariant classes / sweep hits to watch here>
REPORT: per the format in USABILITY-REVIEW.md §5 — step results and findings only.
```

**Maintenance rule:** this matrix is only as good as its coverage. See §7.

---

## 4. Targeted static leads

Use static inspection only when it helps the scoped journey: server JSON versus UI/mock shape,
CSS wiring, external-CLI assumptions, null handling, mutation error feedback, or copy versus action.
Do not run whole-tree sweeps for a focused pass. A lead becomes a usability finding after observed
impact; route source-only defects through code review.

---

## 5. Execution and evidence

Establish the scoped FS acceptance IDs and journeys first. Delegate independent journeys only when
useful; give each owner its entry fixture, port, binary, backend recipe, steps, and expected results.
Parallel journeys need separate state copies and ports. Restart durability deliberately reuses its
predecessor's state and runs after it. The main reviewer owns finding validation and deduplication.

Use the shared browser setup. Record per-step `PASS | FAIL | BLOCKED(reason) | SKIPPED(reason)`.
A finding needs severity, location, reproducible steps, expected/observed outcome, and evidence path.
Keep passing steps compact; capture screenshots where they establish a claim rather than at every
click. Include keyboard, supported widths/appearances, and reduced motion when the scoped behavior
depends on them.

Reproduce material failures and independently confirm blockers. If confirmation fails, label the
claim unconfirmed. Bound retries by their value: one reproduction and confirmation normally suffice;
investigate a harness problem only while a specific next check can resolve it. Otherwise report the
blocked checks and continue unaffected work. Do not reject useful evidence for report formatting.

---

## 6. Reporting & handoff

Record confirmed **Must fix** / **Worth fixing** findings in the handoff using workflow §7, with
journey IDs and evidence. A possible systemic class remains a candidate for fix to validate; this
role does not edit specifications or invariants.

Keep a run summary when breadth or retained evidence warrants one; a narrow pass may record its
checks and evidence directly in the handoff. Distinguish passed from unexercised checks. The human
update states scope, material findings, and blocked/skipped coverage, linking a report if retained.
Follow workflow §§5–6 for scoped commits and closure. Reports may be included when needed to retain
the review's evidence. Push only with explicit authorization.

---

## 7. Maintenance — the matrix must track the product

When an FS adds or changes a user-facing acceptance item, the same change adds or extends a
journey charter here when browser verification is appropriate. Compare the selected charters with their relevant A-items; inspect the full index only for a broad
coverage review. Record uncovered normal-use surfaces without silently expanding a scoped pass.
