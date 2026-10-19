# Enforce process and terminal-driver ownership

**State:** In progress
**Why:** Direct request on 2026-10-19 to formalize the systemic refactor proposals and unsafe control fixes.
**Execution unit:** ASW-3; use `/work enforce-process-and-driver-ownership`.
**Owned findings:** ASRA-4, ASRB-1, ASRB-4, ASRB-5, ASRB-6, ASRB-7, ASRC-3, ASRC-4, ASRC-7.
**Implementation model:** difficult — Codex Sol.
**Relevant requirements:** FS-01.R54/A36, FS-07.R25/A11/A12, TS-01.R41, TS-02.R54,
TS-04.R102, TS-05.R28, TS-12.R11 (planned); existing FS-01.R17–R21/R34,
FS-07.R1/R3/R12–R17, TS-12.R5 and INV §1/§2/§4/§5/§6/§9/§10/§11/§12/§14/§15/§16/§17.

## Outcome

A successful launch has a provable controllable execution and one generation owner. Stop refuses
unknown/mismatched process proof; external terminal exits release ownership, and terminal Resume
retains the actual native conversation discovered by SessionStart.

## Included work

Extend current owned-process helpers and TerminalDriver result with actual birth/PID/dedicated
PGID, unique agent/generation machine handle and exit observation. Validate capabilities on the
returned handle, not installation or optional fields. Existing registry launch/resume gains an
explicit launching→committed/exited generation claim; an exit cannot be overwritten by start commit.

xterm retains direct owned spawn/wait. tmux/iTerm2 use one hidden short-lived Chuck bootstrap that
reads a bounded owner-only launch record. Shell exec in a new owned view preserves the dedicated
foreground group/TTY; the helper validates it, reports
nonce-bound OS process proof and execs the exact composed argv/env/cwd. Secrets never go in external
command text or persisted driver IDs. Invoke the hidden command through the current executable,
including packaged installations. Parent corroborates birth/group/TTY independently of the nonce.
Native handshake/foreground semantics must be verified; statically unsupported operations refuse
before teardown, while dynamic target-start failure uses existing truthful rollback. Retain supported iTerm
functionality through this owned path rather than globally removing the driver. Monitor external
process identity at the specified bounded interval and use existing generation teardown on exit.

Persist process proof forward-only and corroborate immediately before destructive signals/group
termination. Unknown legacy proof is an actionable refusal, not permission to kill/prune/start
another process. Preserve the live orphan record until exact death/control is established; restore
FS-01.R20's done/process-exited outcome for confirmed dead rows in startup and periodic sweeps.
No backfill guesses from row time or claim of atomic OS identity-and-signal on Darwin.

Update running and durable native session ID together at authenticated SessionStart. Preserve the
same-generation native ID before row deletion; missing old identity refuses native Resume rather
than silently fresh-starting. Reuse owned group termination and bounded pipe drain for project setup
cancellation (A4); project readiness/resource-mode fixes stay in ASF-5's existing seam.

## How we will know it works

FS-01.A36/FS-07.A11 prove exit-before-commit, late old exit, reused PID, missing legacy proof,
descendant termination, exact environment and same-title unique handles; terminal SessionStart
survives Stop/crash/restart and chat switch. Setup cancellation/deadline with inherited stdout
descriptors returns within the execution/termination/drain bounds.

FS-07.A12 requires real tmux against an existing server and real iTerm2: two-agent hook/env isolation,
native session identity, foreground/Cancel/Stop, viewer detach versus CLI exit and crash→control.
Installation and fake Tab results do not prove these. Test installed helper discovery, replay/
symlink/bounds rejection, cancelled handshake and OS-proof mismatch. Clean files on every failure.
Port original regressions, run focused race/subprocess checks and TS-06 closure, then independent
substantive architecture review. No destructive control test may target an unowned user process.

## Waiting on

None for implementation. Real-platform/provider receipts remain required before claiming each
supported driver contract; unavailable native proof must be recorded and operations fail safely.
No process/auth/home migration or new long-lived supervisor. Coordinate registry/journal attachment
with ASW-1 and runtime turn changes with ASF-1.
