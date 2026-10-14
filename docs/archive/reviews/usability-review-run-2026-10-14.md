# Usability review run — 2026-10-14

## Scope and outcome

- **Scope:** J1 first paint on a fresh home, and the stopped-agent image wake from FS-03.A64
  (desktop wake composer: unknown capability accepted, unknown capability refused by the fresh
  peer, saved explicit refusal). No scope was named; this is the change that landed last.
- **Baseline:** `8f8d868e` on `main`.
- **Outcome:** no BLOCKER or MAJOR. One confirmed **MINOR / Worth fixing** (stale unconfirmed-send
  Retry after a definite image refusal). One unconfirmed risk lead.
- **Evidence:** [screenshots](usability-review-2026-10-14-evidence/).

## Setup

- **Build:** `make embed && make build` (`sqlite_fts5`), run as `chuck dashboard start --port 4711`
  with `CHUCK_HOME` in a review-owned temp dir and `CHUCK_RUNTIME_ROOT` unset.
- **Agent backend:** the fake ACP peer built as `claude-agent-acp` first on `PATH` (the source build
  launches that name), with backend env `FAKEACP_SCENARIO=stream_text` and, for the accepting
  variant, `FAKEACP_IMAGE_INPUT=1`.
- **Legacy snapshot:** agent stopped, server stopped, `sessions.runtime_capabilities_json` rewritten
  to the pre-image shape (no `image_input`), server restarted. The session read then omitted
  `image_input`, as TS-03.R63 requires.
- **Browser rung:** 1 — Playwright with the cached Chromium, 1400 px wide desktop.

## Journey matrix

| Journey | Result | Observed coverage | Evidence |
|---|---|---|---|
| J1 Install & first paint | PASS | Fresh home, tagged build, styled shell (Instrument Sans, themed background), zero console errors, **Local link**. | [first paint](usability-review-2026-10-14-evidence/J1-first-paint.png) |
| A64 unknown → fresh peer accepts | PASS | Send enabled with image attached on a stopped agent; one POST woke it (202), reply streamed, draft cleared, image shown in the user message; reload shows the same single turn. Repeated three times (after an earlier turn, after restart with legacy snapshot, and on a never-prompted agent). | [after send](usability-review-2026-10-14-evidence/A-3-after.png), [reload](usability-review-2026-10-14-evidence/A-4-reload-tall.png), [fresh agent](usability-review-2026-10-14-evidence/D-3-after.png) |
| A64 unknown → fresh peer refuses | FAIL (MINOR) | Refused with 422 before consumption; reason visible; text and image kept; transcript empty. The composer also claims the message was not confirmed and offers a Retry that repeats the refusal. | [refused](usability-review-2026-10-14-evidence/G-1-refused.png), [after Retry](usability-review-2026-10-14-evidence/G-2-retry.png) |
| A64 saved explicit refusal | PASS | `image_input:false` saved after the refused wake; Send disabled with "This runtime does not accept image input…"; text and image survive reload; removing the image re-enables Send. | [blocked](usability-review-2026-10-14-evidence/F-1-blocked.png), [reload](usability-review-2026-10-14-evidence/F-2-reload.png) |
| A64 phone composer | SKIPPED | Phone stopped agents offer Resume, not a wake composer (handoff Active change); not exercised. | — |

## Findings

```
SEVERITY: MINOR
WHERE: A64 refusal (fixture home-a, port 4711, fake peer without image support)
REPRO: stop agent with legacy snapshot → restart → attach PNG + text → Send → click Retry previous send → reload
EXPECTED: definite refusal reads as final; no Retry of the identical refused request
OBSERVED: "Previous message was not confirmed. Retry previous send" beside the refusal; Retry POSTs 422 again with no visible change; notice persists after reload
EVIDENCE: usability-review-2026-10-14-evidence/G-1-refused.png, G-2-retry.png
```

## Risk leads (not findings)

- **Unconfirmed live duplicate on first wake.** In the first accepting run the live transcript showed
  the prompt text, then the reply, then the prompt again with its image below the reply; reload
  showed the correct single turn. Three repeats (including a never-prompted agent under the same
  legacy setup) rendered correctly. Not reproduced, so not a finding. Evidence:
  [A-3-after](usability-review-2026-10-14-evidence/A-3-after.png).
- Opening a stopped agent logs a browser console error for `GET /api/sessions/{id}/prompt` → 404
  when nothing is held. Not user-visible.
