# Isolated rendered verification

Use the working tree's real UI for the scoped FS/TS acceptance items. This setup supports design,
UX, and usability review; it does not require a full journey matrix or every fixture.

- Use `make dist` when UI source or embedded assets changed, or `make build` with current
  embedded assets (`sqlite_fts5` enabled). Do not run
  asset generation concurrently with tests that use embedded assets.
- Start the server loopback-only on a free port with an isolated, review-owned temporary
  `CHUCK_HOME`; never use the operator's live home. Preserve state needed by the scoped check.
- Create only needed fixtures: empty for first run, seeded for controls, lived-in for history.
  For deterministic live behavior, build `internal/runtime/testdata/fakeacp` using the existing
  integration-test recipe and register it in the isolated config. Real-provider checks need the
  actual provider and credentials.
- Drive an available real browser. Retain evidence of material outcomes (screenshots, console
  errors, or observed state). Select supported widths, appearances, keyboard, and motion states
  from affected requirements; static rendering cannot pass visual interaction.
- Use an execution owner able to request required permissions. Follow normal approval for blocked
  browser or loopback execution; never bypass denied approval. Record unverified checks.
- Stop review-owned processes; retain evidence needed for findings or review scope.

Report `PASS`, `FAIL`, `BLOCKED(reason)`, or `SKIPPED(reason)`. API/component checks may provide
partial evidence without a browser; browser-dependent claims remain blocked.
