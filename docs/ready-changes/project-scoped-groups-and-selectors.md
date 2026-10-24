# Project-scoped groups and selectors

**State:** Waiting to start
**Why:** Operator `/design-feature`, 2026-10-24: confusing group entry and cross-project choices;
scope confirmed with an explicit request to keep the change small.
**Relevant requirements:** FS-02.R80–R82/A61–A63; FS-13.R34/A24; FS-21.R58/A41;
TS-08.R139 (supersedes R118's global/create clauses); INV §1, §2, §8, §10, §11.

## Outcome
Type a group name normally; see only the current project's groups and independent collapse
choices. Annotation/Think Tank selectors show the current project first, with an explicit
Include other projects control.

## Included work
Reuse GroupPicker, existing form/mutation paths and agent-store projections. Clear group on
launch-project change. Extend layout persistence with project-specific collapse overrides,
using legacy choices as initial defaults. Preserve all memberships and conversations.
No new group entity, framework, permission policy, or change to global search/navigation.

Source inspection: GroupPicker resets its active option to Ungrouped on typing and discards
pending text on blur; all four desktop/phone callers collect global labels. Desktop collapse
keys only by label; phone already keys by project. AnnotationTray/AnnotationForm and
ThinkTankSetupDialog intentionally list across projects today; project dashboards already filter.

## How we will know it works
Focused picker/form, selector and layout round-trip tests plus one two-project desktop/phone
browser pass covering the cited acceptance criteria. Reuse existing tests/journeys; no broad
product redesign or security audit.

## Waiting on
Nothing. Design only; implementation has not started.
