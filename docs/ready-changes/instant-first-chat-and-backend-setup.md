# Start a first chat with Claude or Codex

**State:** Waiting to start
**Why:** The operator asked for a working first chat immediately after installation, one clear
backend setup action, and direct Settings links on provider setup errors. The operator approved
one-click bundle selection when an installed CLI is missing and explicit native configuration linking.
**Relevant requirements:** FS-04.R62–R65, FS-09.R84–R85, FS-10.R27, TS-02.R66, TS-03.R87–R89,
TS-04.R112, TS-08.R165–R167, INV §1, §2, §3, §5, §8, §10–§17, §20, §23–§24.

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
