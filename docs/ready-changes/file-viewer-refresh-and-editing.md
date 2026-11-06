# File-viewer refresh and editing

**State:** In progress
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
and generated embed.

## Waiting on

Nothing. Selected by `/work` on 2026-11-06.

## Implementation direction

Extend the existing reader: keep the file and read time steady during writes, with a fixed-size
status indicator. Source editing occupies the same reader; Save/Cancel and comparison state stay
explicit. The primary task is reading agent-produced files, with deliberate edits as the second
path. The main failure to prevent is losing either reading context or unsaved text to a late reply.
Use existing controls and semantic tokens; motion is limited to the freshness spinner, static
under reduced motion. Backend, draft editor and refresh integration are separate work slices;
verify their boundaries before the combined actual-product journey.

## Implemented verification

- Real registered session/room routes, retained source after session removal, remote probe
  filtering and explicit PUT denial; exact BOM/CRLF/UTF-8 disk bytes and mode; symlink, hard-link,
  ACL/xattr and unsafe-file refusal; parent/target replacement, concurrent saves and post-replace
  uncertainty injection. Focused server `-race` checks pass.
- Full UI suite: 1170 tests pass. Timer/request-order tests cover quiet coalescing, selection,
  visibility, stale replies, saved-snapshot fencing and one in-flight request per tab. Draft and
  router tests cover capacity, lineage, in-flight typing, conflicts and unsafe Back/navigation.
- Real Chromium IndexedDB transactions preserve exact text/base/source across reload, refuse
  draft 51 without eviction, enforce capacity across pages and coalesce 1000 generations to one
  pending write (`ui/scripts/drafts-browser.mjs`).
- Actual embedded app with isolated `stress-fixture` fake ACP: live/archive/room/Focus saves,
  independently inspected disk bytes and unchanged transcript, conflict/adopt/retry, continuous
  writes, selection-gated refresh, reduced motion and reload recovery. All four appearances at
  1024/1440px; screenshots `/private/tmp/chuck-file-editing-final`.
- Real phone bundle at 360/390/430px: fixture file GETs, no Edit/Save or PUT; screenshots
  `/private/tmp/chuck-file-editing-phone`. Its SSE/tracked-file admission is fixture-controlled;
  this does not close real tailnet pairing, physical-phone or non-Chromium gates.
