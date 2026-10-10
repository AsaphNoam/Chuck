# No persona and optional awareness — verification receipt

Implementation: `d29a4f5` through `2de7167`. Governing acceptance: FS-01.A29–A30,
FS-04.A34–A36, FS-18.A16. Closure on 2026-10-10.

- `make test`: specification checks and both Go variants passed.
- `make dist`: UI/presentation checks, generated embed and tagged binary passed.
- UI suite: 82 files, 771 tests passed; 41 presentation/style contract checks passed.
- Focused race tests for frozen lifecycle composition and clone after persona edits passed.
- Real Chromium, embedded desktop and authenticated phone routes: 88 checks passed, zero page
  errors. Desktop 1024×900; phone 360/390/430×844; Core, Sky & Grove and Studio.
  Keyboard journeys covered closed-default launch, selected-persona launch, collapsed summaries,
  retained drafts, saved custom default, canonical Default hidden in Settings, persona awareness
  editing/help association, full onboarding to launch, and the development presentation matrix.
- Rendered screenshots were inspected for hierarchy, focus, density, clipping and overflow.
  No material in-scope visual/interaction finding remained.

Raw receipt and screenshots: `/tmp/chuck-persona-journey-final3/receipt.json` and its sibling PNGs.
The fixture uses isolated temporary homes, synthetic readiness and fake ACP; these receipts prove
host behavior, not credentialed provider login or model adoption of instructions. Existing pinned
native-prompt tests passed in the runtime suite; the handoff's existing live-provider gates remain.

Reproduction after `make dist`:

```sh
CHUCK_GROUP_BROWSER_READY=/tmp/persona.json CHUCK_PERSONA_BROWSER_ONBOARDING=1 \
  go test -tags sqlite_fts5 ./internal/server -run '^TestGroupBrowserFixture$' -timeout 20m -v
node ui/scripts/persona-journey.mjs /tmp/persona.json /tmp/persona-receipt
touch /tmp/persona.json.done
```

The optional fourth script argument is a loopback Vite origin for `/__visual-matrix`; its API proxy
must target the fixture's desktop URL. Synthetic missing readiness briefly opens the real wizard;
removing the fixture-owned marker lets ordinary Validate → Project → Config → Launch proceed.
No real credentials, user homes, or external providers are touched.
