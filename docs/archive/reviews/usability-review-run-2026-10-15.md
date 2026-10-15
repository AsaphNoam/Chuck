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

Not promoted: a stale backend-check result after switching type (reproduced once only); the
700 px narrow file-viewer check (1024 px is the specified desktop floor); a possible hidden
“Timed out” label (unconfirmed); POLISH items (no wizard Back, Launch step returns to dashboard
instead of opening the agent, phantom last line number in the file viewer). Batch 2: no explicit eviction notice (FS-02 accepts
silent least-recently-used eviction with draft retention); a possibly clipped New Agent dialog with
Advanced open (scrolling unverified); stop actions reachable only by right-click; “finished” toast on
a user-initiated stop; console 404 on the pending-prompt fetch for stopped agents; two toasts per
permission request; empty-project panel wording.

## Not exercised
