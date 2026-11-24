# Start a first chat with Claude or Codex

**State:** Finished — packaged provider acceptance gates owed
**Why:** The operator asked for a working first chat immediately after installation, one clear
backend setup action, and direct Settings links on provider setup errors. The operator approved
one-click bundle selection when an installed CLI is missing and explicit native configuration linking.
**Relevant requirements:** FS-04.R62–R65, FS-09.R84–R85, FS-10.R27, TS-02.R66, TS-03.R87–R89,
TS-04.R15/R112, TS-08.R165–R167, INV §1, §2, §3, §5, §8, §10–§17, §20, §23–§24.

## Outcome

Fresh interactive installs open Chuck directly. Claude and Codex are checked independently using
existing sign-in. As soon as either is ready, pick a project folder if needed and start chatting.
Each provider in Settings has one primary refresh or save-and-check action. Provider setup errors
open the affected backend in a new tab and leave the current form ready to resume.

## Included work

- Replace the installer Claude-only prompt and fresh Claude/Codex wizard path with first-chat
  setup. Use the installed executable by default. Keep explicit bundle recovery and native
  linking. Existing homes and the explicit setup for other providers keep their normal behavior.
- Reuse provider resolution, credential probes, scoped model import, backend validation and the
  catalog lock. Preserve unrelated drafts and give a truthful result after a partial failure.
- Reuse the Settings Backends editor for targeted desktop repair. Keep configuration local to
  the Mac and preserve the first-run or New Agent form while Settings is open.

## How we will know it works

- FS-04.A41–A44 and FS-10.A15 cover fresh installs, both ready-provider paths, the bundle/sign-in
  recovery, preserving setup through repair, and avoiding duplicate work on retry.
- FS-09.A53–A54 cover checking with unsaved edits, concurrent saves, partial failures, and
  keyboard navigation to the intended backend and back to the original form.
- Run deterministic tests against the real server/client boundary and the rendered desktop flow.
  Verify a first chat with existing native sign-in for each packaged provider and a packaged
  missing-CLI recovery through the bundle.

## Implementation direction

The first view serves the operator's first chat: provider outcomes arrive independently above
one project selection and **Start chat** action. Keep the existing onboarding dialog, typography,
semantic status tokens and folder picker; show repair beside its provider and keep advanced
configuration subordinate in Settings. No new motion is needed. A check is readiness evidence,
not a promise of model access. Preserve selection, folder and confirmed create/launch results
through repair; an uncertain response offers recovery instead of automatically replaying it.

Implementation slices: target-scoped provider/config API; first-chat and Settings composition;
installer startup; integrated checks and rendered recovery. The parent owns integration and
verification, with separate backend and frontend ownership during implementation.

## Verification receipt — 2026-11-24

Implemented in `406251af`, `01d8573e`, `be225a99` and `f95ce4d8`.
Final-source `make test` passes both Go configurations; `make dist` and the final Go-only
`make build` pass. The full UI suite passes 1,423 tests across 132 files. Focused provider
concurrency race checks and native-status process-boundary regressions pass; the latter failed
before the repair. Unknown or malformed native authentication output no longer reports ready.

Rendered Core and Sky desktop flows cover independent readiness, targeted keyboard repair,
draft preservation, partial completion retry without duplicate creation, Settings save/check,
explicit native-link failure and return-to-form recovery without automatic launch. Evidence is
in the shared project resources under `first-chat-browser/` (`report.json`,
`settings-report.json`, `repaired-error-report.json` and screenshots). These use real production
server routes with fake providers/ACP and isolated homes, not live native authentication.
The final repaired-error rendering routes the final built static UI over the existing isolated
server APIs, without replaying completed setup operations.

First-chat project suggestions conservatively exclude currently missing owned-worktree checkouts;
ordinary launch retains its existing recreation path. Packaged native authentication, executable
resolution and model access remain unverified; the explicit acceptance gates are recorded in
[ACCEPTANCE-GATES.md](../features/ACCEPTANCE-GATES.md).
