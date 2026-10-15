# Usability review run — 2026-10-15

## Scope and outcome

- **Scope:** full top-to-bottom J1–J17 journey regression and S1–S5 static sweeps.
- **Baseline:** `317706c8` on `main`.
- **Status:** in progress — journeys run in small batches; rows below are filled as each batch
  returns.

## Setup

- **Builds:** `make embed && make build` (`sqlite_fts5`) plus an untagged `go build` for the J8
  fallback pass; fake ACP built from `internal/runtime/testdata/fakeacp` and placed on `PATH` as
  `claude-agent-acp`, so the real `chuck dashboard start` binary drives it with no provider
  credentials.
- **Fixtures:** isolated `CHUCK_HOME` copies under `/tmp/chuck-ur-2026-10-15/homes/`: `fresh`
  (empty), `seeded` (onboarded; Review Workspace project on a scratch Git repo; Claude models
  mapped to fake scenarios), `lived-in` (seeded plus three stopped and two archived agents with
  transcripts). One home and loopback port per journey.
- **Browser rung:** 1 — Playwright driving Chromium, 1280 × 800 unless noted.
- **Repo changes:** none to product code or specifications.

## Journey matrix

| Journey | Result | Observed coverage and remaining branch | Evidence |
|---|---|---|---|
| J1 Install & first paint | PASS | Tagged build, fresh home, styled shell (Instrument Sans, themed background, styled buttons), zero console/page errors; config files sane, `layout.order` is `[]`. | [first paint](usability-review-2026-10-15-evidence/J1-1-first.png) |
| J2 Onboarding wizard | PARTIAL | On this machine the real Claude CLI is found even with a stripped `PATH`, so the wizard only appears with a fake `HOME` as well. Missing-CLI guidance, empty-field validation, Escape/outside-click guard, Set up later, project → config → launch, restart without re-showing the wizard passed. Logged-out/logged-in real-CLI branches SKIPPED (no credentials used). No Back control exists. | [OpenCode guidance](usability-review-2026-10-15-evidence/J1-2-OpenCode.png), [relative path](usability-review-2026-10-15-evidence/J1-2-e-relative.png) |
| J3 First launch + chat | FAIL | Launch, streamed reply, idle status, Mermaid streaming/completed/source toggle/reload/archive/Sky & Grove, inert injection with no requests, file links (line mark, rendered/source, `?file=` reload, Close, absolute path per FS-03.R64, missing-file refusal) passed. Live view duplicates every sent message. Busy state too brief to observe with the fake. | [duplicate](usability-review-2026-10-15-evidence/V3-live-dup.png), [reload scroll](usability-review-2026-10-15-evidence/J3-diag-reload.png), [Sky & Grove](usability-review-2026-10-15-evidence/J3-diag-sky.png), [file viewer](usability-review-2026-10-15-evidence/J3-file-main.png) |
| J4 Permission flow | PASS | Prompt + card “waiting”, approve, deny, reload while pending (one prompt), and timeout (server `PERMISSION_TIMEOUT=45s`) each resolved exactly once and left the agent usable. | [prompt](usability-review-2026-10-15-evidence/J4-1-prompt.png), [approved](usability-review-2026-10-15-evidence/J4-2-approved.png), [timeout](usability-review-2026-10-15-evidence/J4-4-timeout.png) |
| J5 Grid & layout | PASS | Project title and zero meter, Resume only on stopped cards, drag/density/groups/collapse across reload and restart, archive of an ordered agent, empty project state, active-first ordering (live move on stop, in-block drag, refused cross-boundary drop), chat panes (fixed height, internal scroll, no collapse on Send/selection/right-click, Ctrl+Alt+Arrow focus cycling), and automatic pane opening with least-recently-used eviction restoring the draft. A load-time layout save is recorded below. | [cross-boundary drop](usability-review-2026-10-15-evidence/J5-5c-stop-over-run.png), [pane](usability-review-2026-10-15-evidence/J5-6b.png), [eviction](usability-review-2026-10-15-evidence/J5-7c-after.png) |
| J6 Terminal runtime | PASS | Default PTY driver (tmux absent, not required) with a local `claude` stub: render, exact typing/backspace, resize, navigate/reload reattach, Stop/Resume, missing-CLI launch error. Stopped terminal page finding below. Real provider CLI not exercised. | [resize](usability-review-2026-10-15-evidence/J6-3.png), [stopped](usability-review-2026-10-15-evidence/J6-5e.png) |
| J7 Stop / resume / switch | PASS | Resume, Stop → Resume and Switch (model and cross-backend) kept id, transcript, model, system-prompt marker, cwd and add_dirs, observed in the fake peer's load dumps; stopped-agent composer wake; repeated/concurrent stop/resume consistent with one process. Project edits do not reach existing agents (frozen at launch per FS-01). A stopped agent's model cannot be changed without resuming it (disabled with hover reason). | [switched](usability-review-2026-10-15-evidence/J7-switched.png), [resumed](usability-review-2026-10-15-evidence/J7-resumed1.png), [menu](usability-review-2026-10-15-evidence/J7-ctx.png) |
| J8 Archive & search | PARTIAL | Tagged and untagged: empty state, both archived agents listed, metadata/name search, no-results, read-only transcript, Restore back to the project. Tagged: transcript search with snippets and 27 archived agents (pagination past 50 not reached). Untagged states metadata-only scope. Missing/bogus archived session shows a blank page (finding). | [search](usability-review-2026-10-15-evidence/J8-t-search-pelican.png), [fallback](usability-review-2026-10-15-evidence/J8-u-search-pelican.png), [restored](usability-review-2026-10-15-evidence/J8-t-restored.png), [bogus id](usability-review-2026-10-15-evidence/J8-t-err-bogus-id.png) |
| J9 Settings & config | PASS with finding | All seven tabs round-trip; seeded backends/models and projects merge-preserved; most invalid input refused visibly; Core/Sky & Grove/Studio apply live and persist; role delete-in-use confirms and agent keeps working; config-source panel and Remote render on a fresh home (`devices: []`). Blank backend display name persists on edit (finding). | [Studio](usability-review-2026-10-15-evidence/J9-app-studio-dash.png), [role in use](usability-review-2026-10-15-evidence/J9-role-delete-inuse.png), [blank name](usability-review-2026-10-15-evidence/J9-be-emptyname.png) |
| J10 Multi-agent + messaging | FAIL | Driven through the real `/mcp` `send_message` with an agent token (no UI path exists). Waking mail woke a stopped recipient exactly once; mail to a busy agent waited and was delivered once after the turn; restart rendered correct Mail N badges. Live Mail N / Sent never appear after a send (finding). The mail-triggered turn has no visible cause in the recipient transcript. No human inbox surface exists. | [cards](usability-review-2026-10-15-evidence/J10-3-cards-unread.png), [SSE capture](usability-review-2026-10-15-evidence/V10-sse-after-send.txt), [recipient](usability-review-2026-10-15-evidence/J10-4-D-transcript.png) |
| J11 Failure & recovery | PARTIAL | Server kill → “Reconnecting” within 4 s; send while down fails visibly and restores the draft; automatic reconnect with accurate stopped state and catch-up of changes made while disconnected; process kill reflected in under 1 s; crash survives reload and composer wake recovers; garbage in agent/project/role forms, archive search operators and a 100 KB composer paste handled. Pipeline template save with invalid ids returns 500 (finding). Reconnect does not heal the duplicate bubbles. Tasks has no create form. | [reconnecting](usability-review-2026-10-15-evidence/J11-1b-down4s.png), [send while down](usability-review-2026-10-15-evidence/J11-1c-send-down.png), [crash](usability-review-2026-10-15-evidence/J11-3a-crash-page.png) |
| J13 Annotate & assign | FAIL | Live diff-line, whole-event and text-range annotations, tray edit/reload, archived-source annotation, send to self (one “Annotations assigned” card, no duplicate prompt), send to another agent, prefilled new launch with cancel preserving the tray, and archive search for annotation text passed. New-task launch happens before delivery is validated (finding). | [self](usability-review-2026-10-15-evidence/J13-3-self.png), [orphan launch](usability-review-2026-10-15-evidence/J13-7-a.png), [archived default](usability-review-2026-10-15-evidence/J13-2-sendself.png) |
| J14 Configurable pipelines | PARTIAL | Runs/Templates split and empty states, four-stage template save/reload with one mounted stage form, start with mixed runtimes to the run page, run badge consistent with stage/attempt (the 2026-09-26 QUEUED finding is fixed), legacy `?run=` redirect, gone-run page, Stop and Delete passed. Restart leaves the run stuck (finding). Stage completion, routing, repair loop, approvals and delegated cards BLOCKED: the fake peer never calls `report_task_result`. Builder proposals SKIPPED (live model). Under reduced motion the Start-run dialog still fades (opacity only). | [run page](usability-review-2026-10-15-evidence/J14-3-runpage.png), [after restart](usability-review-2026-10-15-evidence/J14-6-after-restart.png) |
| J17 Browser-local drafts | PASS | Two drafts isolated through navigation and reload; sent draft cleared and never returned; stopping an agent leaked nothing and kept its own draft. | [drafts](usability-review-2026-10-15-evidence/J17-5-A.png), [stopped](usability-review-2026-10-15-evidence/J17-7-Bstopped.png) |

## Static sweeps

S1–S5 ran read-only and produced risk leads only; S2 found no undefined class. Leads carried into
journeys: possible `null` collections in Think Tank participants, config-source lists, remote
devices and attachment ids (S1); bare `tmux` calls without an install check (S3, J6); swallowed
archived-transcript load and post-reconnect refetch errors (S5, J8/J11).

**Matrix coverage (§7):** Think Tanks (FS-21), phone remote control (FS-20), project resources
(FS-11), native config federation (FS-08) and skin switching (FS-12) have no journey charter of
their own; they may need charters or are covered only by dedicated passes.

## Findings

```text
SEVERITY: MAJOR
WHERE: J3 step 2 (seeded home, port 4650), FS-03 transcript
REPRO: Open any chat agent, type "hello unique-xyz", press Enter, wait for the reply.
EXPECTED: One user bubble, then the reply.
OBSERVED: Two `article.user-message` bubbles with that text (one before the reply, one after the turn divider); the server transcript holds one `user_text`; a reload shows one.
EVIDENCE: usability-review-2026-10-15-evidence/V3-live-dup.png
```

```text
SEVERITY: MINOR
WHERE: J3 step 3 (agent with two diagram turns)
REPRO: Hard-reload the agent page.
EXPECTED: Transcript opens at the latest content.
OBSERVED: Stops mid-transcript with "Jump to latest" shown; the newest reply is below the fold (diagram height settles after the initial scroll).
EVIDENCE: usability-review-2026-10-15-evidence/J3-diag-reload.png
```

```text
SEVERITY: MINOR
WHERE: J2 Backend step, OpenCode/OpenHands with no adapter
REPRO: Select OpenCode or OpenHands, Check again.
EXPECTED: Actionable guidance like Claude/Codex (install link or command).
OBSERVED: "The OpenCode adapter is not installed. Install it, then check again." — no link or command.
EVIDENCE: usability-review-2026-10-15-evidence/J1-2-OpenCode.png
```

```text
SEVERITY: MINOR
WHERE: J2 Project step, FS-04.R30
REPRO: Title "Smoke", working directory "relative/path" (nonexistent), Create project.
EXPECTED: The cwd_not_found warning beside the field.
OBSERVED: Wizard advances silently; API returned 201 with warnings [cwd_not_found] and stored the relative path.
EVIDENCE: usability-review-2026-10-15-evidence/J1-2-e-relative.png
```

```text
SEVERITY: MINOR
WHERE: J6 step 5 (seeded home, port 4606), FS-07
REPRO: Launch a terminal agent, Stop it from the card menu, open its agent page (Terminal tab).
EXPECTED: A stopped state with a way to resume (or the prior scrollback).
OBSERVED: Terminal shows "[connection error]" and "[closed]"; console 404 on the terminal WebSocket; footer still says "Terminal agents receive input in the terminal tab."; no Resume on the page.
EVIDENCE: usability-review-2026-10-15-evidence/J6-5e.png
```

```text
SEVERITY: MINOR
WHERE: J5 step 3 (seeded/lived-in homes, ports 4605 and 4651), FS-02.R14
REPRO: Load /project/review with no interaction; watch requests.
EXPECTED: No layout write without a user action.
OBSERVED: Every page load sends PUT /api/layout ~0.4 s after the GET (four of four loads). In J5's seven-agent run the first loads also reshuffled card order and one load-time write re-added an archived agent id an earlier drag had pruned; the reshuffle did not reproduce on the three-agent lived-in home.
EVIDENCE: Playwright request capture (work/V5/order.cjs); J5 server.log GET/PUT pairs
```

```text
SEVERITY: MAJOR
WHERE: J10 steps 2/3 (homes J10 port 4610; replayed on V10 port 4652), FS-06.R13
REPRO: Two running chat agents; from A's MCP session call send_message to B (wake:false or wake:true); watch /api/events and B's card.
EXPECTED: B's next state_update carries unread_messages, card shows Mail N; A shows Sent for two seconds.
OBSERVED: No state_update for either agent after the send; GET /api/sessions shows B unread_messages null while GET /api/sessions/B/messages reports unread_count 1. Badges stay absent or stale (A read Mail 1 with 3 unread) until a server restart. The 2026-09-26 run saw these badges live.
EVIDENCE: usability-review-2026-10-15-evidence/V10-sse-after-send.txt, J10-3-cards-unread.png
```

```text
SEVERITY: MINOR
WHERE: J10 step 2/4 (V10 port 4652), FS-06.R31–R32 (possible missing requirement)
REPRO: send_message wake:true to an idle agent; open its agent page / transcript.
EXPECTED: The person supervising can see that peer mail started this turn (sender/subject).
OBSERVED: Transcript holds only the assistant reply and turn_end — an unexplained turn; the mail is visible nowhere in the UI.
EVIDENCE: usability-review-2026-10-15-evidence/J10-4-D-transcript.png
```

```text
SEVERITY: MINOR
WHERE: J8 step 6 (tagged 4608 and untagged 4618), FS-05
REPRO: Open /archive/bogus-id, or an archived agent whose transcript file was removed.
EXPECTED: A visible not-found / transcript-unavailable message.
OBSERVED: Normal-looking "Archived session · read-only" header, empty body, active Resume/Restore; the 404 is swallowed (ArchiveAgentPage `.catch(() => settle(id))`). Resume then shows "no such agent".
EVIDENCE: usability-review-2026-10-15-evidence/J8-t-err-bogus-id.png, J8-t-err-a_528c5a.png
```

```text
SEVERITY: MINOR
WHERE: J9 step 3 (port 4609), FS-09
REPRO: Settings → Backends, clear the Codex display name, Save, reload.
EXPECTED: "Display name is required" (as backend creation enforces); nothing persisted.
OBSERVED: Saves silently; backends.json has codex "name": "".
EVIDENCE: usability-review-2026-10-15-evidence/J9-be-emptyname.png; homes/J9/backends.json
```

```text
SEVERITY: MAJOR
WHERE: J14 step 6 (home J14 port 4614; replayed on port 4654), FS-14.R14/R48/R66
REPRO: Start a run; while stage 1 awaits its result, restart the dashboard; open the run page / GET /api/pipeline-runs/<id>.
EXPECTED: The run pauses, the page says the stage agent was stopped by the restart, withholds Open agent, and offers Retry.
OBSERVED: Stage task is `interrupted` but the run stays `running` with empty attention_reason; Continue/Retry/Replace all ineligible, Open agent still offered; only Stop run remains — a dead end.
EVIDENCE: usability-review-2026-10-15-evidence/J14-6-after-restart.png
```

```text
SEVERITY: MAJOR
WHERE: J13 steps 5/7 (home J13, port 4613), FS-13.R9/R11
REPRO: Tray with a draft whose instruction is empty (or delivery blocked), target New task → Continue to launch → Launch.
EXPECTED: Validation before launch, or a clear "agent launched, delivery failed" with a retry that does not launch again.
OBSERVED: POST /api/sessions 201 then POST …/annotations 422 3 ms later; the new agent has no annotations; tray shows only the validation text (or "Failed to fetch"); "Continue to launch" again creates another agent. After navigating away nothing indicates the orphan exists.
EVIDENCE: usability-review-2026-10-15-evidence/J13-7-a.png, J13-7-b.png; J13 server.log lines 667–668
```

```text
SEVERITY: MINOR
WHERE: J11 step 5 (port 4611; replayed on 4653), FS-14
REPRO: Pipelines → new template with id "BAD ID" (or stage id "BAD"); Validate, then Save template.
EXPECTED: Save refused with the same diagnostics Validate shows (4xx), or disabled while invalid.
OBSERVED: Validate lists the slug errors, Save stays enabled, POST /api/pipelines returns 500 "pipeline operation failed".
EVIDENCE: usability-review-2026-10-15-evidence/J11-5-pipeline-saved.png; curl 500 replay
```

```text
SEVERITY: MINOR
WHERE: J11 step 3 (port 4611), FS-03 error states
REPRO: Agent on the crash scenario, send a message; also kill an agent's process.
EXPECTED: An understandable error that says what failed and how to continue.
OBSERVED: Transcript shows "process exited" / "Failed"; header and card say "Stopped" while the API state is `error`; card carries no error text; nothing says sending a message restarts the agent.
EVIDENCE: usability-review-2026-10-15-evidence/J11-3a-crash-page.png, J11-3b-crash-card.png
```

```text
SEVERITY: MINOR
WHERE: J13 step 2 (archived Delta, port 4613), FS-13.R15
REPRO: Annotate in an archived agent, click Send annotations with the default target.
EXPECTED: An available target is selected (current agent is unavailable).
OBSERVED: The disabled "Current agent" radio stays checked; Send returns 409 and shows "the source agent is not running".
EVIDENCE: usability-review-2026-10-15-evidence/J13-2-sendself.png
```

Not promoted: a stale backend-check result after switching type (reproduced once only); the
700 px narrow file-viewer check (1024 px is the specified desktop floor); a possible hidden
“Timed out” label (unconfirmed); POLISH items (no wizard Back, Launch step returns to dashboard
instead of opening the agent, phantom last line number in the file viewer). Batch 2: no explicit eviction notice (FS-02 accepts
silent least-recently-used eviction with draft retention); a possibly clipped New Agent dialog with
Advanced open (scrolling unverified); stop actions reachable only by right-click; “finished” toast on
a user-initiated stop; console 404 on the pending-prompt fetch for stopped agents; two toasts per
permission request; empty-project panel wording. Batch 3: untagged no-results copy not tied to
the metadata-only note; no upper bound on task concurrency or role prompt size; raw regex in the
role id error; agent id instead of name in the role-in-use dialog. Settings → Projects shows the
`cwd_not_found` warning that the onboarding wizard omits (J2 finding stands). Batch 4: agent
header flips to "Stopped" while merely disconnected; console 409 on `/modes` for stopped agents;
new stage not selected after "+" in the template editor; no confirmation on run Delete; no link
to the agent a tray launch created; low-contrast diff "Annotate lines" trigger; a person-path
`POST /api/tasks/<id>/result` on a stage task (API-only, low confidence). Annotation recipients
also show an unexplained turn (same as the J10 finding). **Spec drift:** FS-14.R44 names a
three-step Setup/Runtimes/Review dialog while a later FS-14 requirement and the product use
Setup/Review with a Customize runtimes disclosure.

## Not exercised
