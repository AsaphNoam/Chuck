# Saved file-viewer tabs

**State:** In progress
**Why:** Operator `/design-feature` request on 2026-10-23: following one file to another loses
the previous file; add tabs that open from file links, close and reorder. Confirmed 2026-10-24:
survive navigation/reload in this browser/device, cover desktop conversations and rooms plus
the phone's existing Files screen, keep current phone access/screens, allow 50 tabs per owner
and 500 total without eviction, and let Back hide the reader without deleting tabs.
**Relevant requirements:** FS-03.R101–R106/A80–A82, FS-20.R56/A28, FS-21.R59/A42,
TS-08.R140–R145; existing TS-13.R22 and TS-14.R12; INV §1/§2/§8/§10/§13/§16/§17.

## Outcome

Follow a chain of files, return to any open file at its reading position, close or reorder tabs,
and recover the same set after leaving or reloading. Room files retain their originating source.

## Included work

One shared tab controller and descriptor codec around the existing reader, active-file URL
projection, browser-local persistence, per-tab in-memory reads and display/scroll continuity.
Desktop live/archive share conversation sets; rooms have their own source-qualified sets; phone
uses its current tracked-file entry and authority. Save descriptors until closed/site-data
clearance or owner deletion, never file contents. Keep annotations, split sizing and Focus file's
existing ownership. No editor, directory browser, cross-device/live-window synchronization,
phone room/archive screen, new read authority, endpoint, durable event or server migration.

Direction: the operator follows linked technical evidence while retaining conversation context.
Keep the existing document and conversation dominant, with one quiet compact tab strip and a
clear active file; room-source identity distinguishes real workspaces. Long names/overflow,
loading/refusal/partial/image states, focus and touch drag must remain usable. Reuse the current
visual system and selection/annotation construction; ordinary selection is instant and any drag
feedback serves spatial continuity with reduced-motion support. Rendered iteration belongs to
implementation; no browser or physical-device proof is claimed by this specification-only work.

Verified seams: `ReaderWorkspace` shares `useFileReader`/`FileViewerView` across docked/focused
desktop views; `ThinkTankPage` supplies the immutable retained-source loader; `AgentScreen`
already uses the shared reader behind the phone tracked-path filter. Existing `collapseStore`
and `annotationStore` demonstrate browser state ownership/failure patterns, but their eviction
and expiry policies do not fit the confirmed no-eviction tab lifetime. Extend the file-reader
controller rather than introducing a global file-content cache. No external provider limitation
or package capability workaround motivates this design.

## How we will know it works

FS-03.A80–A82, FS-20.A28 and FS-21.A42 specify navigation, close/reorder, saved mode/position,
deep links/Back, source isolation, storage/read failures and cap evidence. Exercise live/archive,
Focus file and rooms at 1024/1440px, and phone Files at 360/390/430px in all appearances.
Use independently counted requests and loader arguments, not only DOM tab counts. Carry the
actual-device touch/reconnect gate separately. Follow TS-08.R145/TS-06 for implementation closure.

## Waiting on

Nothing. Product scope and persistence/compatibility boundaries are confirmed; no product code
has been written and this change is not active.
