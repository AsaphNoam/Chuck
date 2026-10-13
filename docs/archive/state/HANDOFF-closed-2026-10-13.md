# Chuck — closed handoff entries (2026-10-12 to 2026-10-13)

**Archived snapshot (2026-10-13).** Non-authoritative; the live file is
[`HANDOFF.md`](../../features/HANDOFF.md). Moved verbatim from **Current position** to keep the
session-start header inside its budget.

- **Closed — blank phone entry on local dev (2026-10-13):** rebuilt the current dev server
  as `/private/tmp/chuck-remote-recovery` and gracefully restarted the pre-fix process;
  PID 18990 serves loopback port 4317 and the existing tailnet address. Live Chromium
  phone-emulation checks show Android pairing and iPhone browser Home Screen instructions;
  all requested JavaScript, CSS and font assets return 200 with no page errors.
  Receipt/screenshots: `/private/tmp/chuck-remote-recovery-{receipt.json,Android.png,iPhone.png}`.
  The fix was already in `0de23db`; no product code/specification change or new review unit.
  Physical-phone reload/pairing remains the operator's next step. Other review units unchanged.
  Changelog: loaded the existing derived-asset refresh fix into the live dev process (INV §1/§10).
- **Closed — mobile CSS missing in dev mode (2026-10-12):** dev-only phone asset graph refresh,
  404s for missing `/assets/` paths, and service-worker invalid-cache recovery implemented. The
  reproductions were confirmed failing before the fix, then unskipped and passed; five worker
  regressions, 800 UI tests, `make test` (both Go variants), `make build`, and `make dist` pass.
  On-device cache recovery remains unverified. Changelog: fixed derived asset state, error amplification, and dev wiring
  drift (INV §1/§7/§10).
- **Reviewed and closed — phone group memory and long-press agent actions (2026-10-13):**
  `17b73ed`, FS-20.R47–R48/A19, TS-08.R121. No findings; 14 focused phone tests and 46
  presentation/style checks pass. Nine independent Chromium journeys verify retained group
  choices, touch and keyboard actions, refusals, nested confirmation, stopped availability and
  restored focus at 360/390/430px in all three appearances. Review evidence below; fixtures
  make no provider or real-tailnet claim. Other review units remain unchanged.
- **Reviewed and closed — phone chat drafts and Theme selection (2026-10-13):** `79cbbc3`,
  FS-20.R49–R50/A20–A21, TS-08.R122, TS-13.R25. The 2026-10-12 review reproduced accepted
  Send cleanup missing a composer reopened while delivery was pending. Theme selection and the
  narrow paired-phone config route passed review. Independent text-only AgentScreen regressions
  now verify delayed Send and Steer leave/reopen/accept cleanup through the shared draft seam,
  including newer-draft protection. Changelog: synchronized accepted draft cleanup across
  reopened composers (INV §1). The other review units are unchanged.

## Active-change closure notes (moved 2026-10-13)

CLI extends existing session APIs without a runtime registration, new storage or lifecycle owner.
INV §6: persistence/full launch use server composition; fan-out/drain, mailbox eligibility, turn
boundaries, reconciliation and hooks retain server ownership; capabilities use server projections.
Observers close only their requests/streams, join bounded workers and remove failed output
temporaries. No new registration, mailbox or lifecycle store. Loopback test fixtures require
loopback-enabled execution permission and fail rather than silently skip when unavailable.

Chat images and file attachments is implemented and independently reviewed; its fixes are pending above.
Final closure: full attachment/state/runtime/transcript and focused API/security/race checks pass.
Forward migration 51 preserves checkpoint-50 receipts; pre-wire recovery remains separate from
ambiguous post-wire outcomes. Incoming output images obey 10-image/25-MiB decoded per-frame limits
before storage and bounded event fanout. Session reads and state snapshots share the same attachment
capability projection. Rendered findings in modal dismissal, sent thumbnails and shared draft
lifecycle were repaired and verified. The existing orchestrator-instructions test race reproduced
on baseline `5fd573c` 2/10 times; a test-only lifecycle-drain wait now passes 10/10.
Final shared matrix passes in `/private/tmp/chuck-attachments-closure-test.log`; final embedded
build passes in `/private/tmp/chuck-attachments-final-dist.log`. The preceding concurrent run's
two-second Codex credential fixture timeout did not reproduce in ten focused current or baseline
runs; the complete rerun without concurrent build load passes. Credential code is unchanged.
