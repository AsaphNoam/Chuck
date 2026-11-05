# File-viewer refresh and editing

**State:** Waiting to start
**Why:** Operator `/design-feature` request on 2026-11-05: viewer files stay stale during agent
work; refresh without repeated disruption and allow direct editing. Scope confirmed with a small
unobtrusive status spinner and retained readable content; optimistic save conflicts and owned-file
write authority subsequently approved.
**Relevant requirements:** FS-03.R110–R112/A86–A88; TS-03.R80–R81; TS-05.R29;
TS-08.R153–R155; INV §§1/2/4/5/8/10/11/13/14/16/17.

## Outcome

Selected previews follow settled disk changes while preserving reading context. Desktop users can
edit source and save deliberately, recover local drafts, and compare detected conflicts without
losing their edits. Phone previews refresh but remain read-only.

## Included work

- Shared selected-visible-file probing, two-second observed quiet window, retained snapshots,
  selection/edit gating and an unobtrusive spinner in the existing status area. No content overlay,
  dimming, layout shift, per-write toast or inactive-tab polling.
- Complete text editing in live/archive/room/Focus file, exact-byte conditional saves, browser-local
  draft recovery and comparison. Existing owned writable regular files may be outside the workspace;
  symlinks, hard links and unsafe metadata replacement are refused. No autosave, create/rename/delete,
  binary/truncated editing, agent messages, server draft copies or phone writes.
- Optimistic revision checks, serialized Chuck saves and atomic replacement. The operator accepted
  the residual race with noncooperating external writers; no exclusive filesystem lock is promised.

Existing seams checked: `FileViewer.tsx` owns request tokens and anchors; `features/fileTabs` owns
source-qualified tabs and read admission; `fileread.go` supplies bounded safe reads. Existing session
and retained-source GETs admit probe mode without a new remote route: `remote_routes.go` filters the
session GET through tracked-file/no-follow checks. Room GETs use retained source Cwd after session
cleanup (`think_tank_activity_handlers.go`). PUTs must use equivalent retained-source lookup and
explicit remote denial. Existing `mod_time` has insufficient precision for freshness/version
checks; the general config atomic-write helper does not establish arbitrary-file metadata safety.

## How we will know it works

FS-03.A86–A88: deterministic refresh/request-order and draft-store tests; real registered route,
remote denial/filter and adversarial filesystem tests with independent byte/metadata oracles;
rendered live/archive/room/Focus file and read-only phone journeys covering continuous writes,
selection, reduced motion, conflict, storage failure and recovery. Cover all four desktop appearances
at the supported floor and wider, and phone at 360/390/430px. Preserve existing physical-phone gates.
Implementation closure follows TS-06 and workflow §2, including applicable Go/UI/presentation checks
and generated embed; no such product proof is claimed by this design-only change.

## Waiting on

Nothing. Ready for explicit `/work` selection; no implementation started.
