# Compact Think Tank navigation in agent chat

**State:** Finished — 2026-10-19; available for independent review
**Why:** Direct `/design-feature` request 2026-10-16; operator approved only other-participant
links, subtle bold/raised active-participant treatment, and a room link, reusing Chuck's design.
**Relevant requirements:** FS-21.R57/A40; TS-08.R126; existing FS-03.R71,
FS-21.R45 (tab contents superseded by R57); INV §8, §10, §13, §17.

## Outcome

The agent-chat Think Tank tab becomes compact navigation to other members and the room.
The active room speaker is slightly bolder and raised, including concurrent opening speakers.

## Included work

Replace the expanded tab report with one wrapping link group per room; omit the current agent.
Keep created judge links, retained unavailable members, multiple/ended memberships and query feedback.
Remove tab goals, allowances, state summaries and general explanation; preserve the existing header
cue and private chat behavior. No room controls, new API, persistence, scheduling or phone surface.

## How we will know it works

FS-21.A40 covers routes, active-attempt changes, concurrent speakers, retained identities,
clipped/error feedback, keyboard behavior and long/duplicate names. Reuse shared button CSS and
router links under TS-08.R126; run focused UI checks during work and the applicable TS-06 closure
once at completion. Inspect real rendered chat at desktop floor/wide and narrow pane widths in
all three appearances, comparing adjacent shared controls. Visual acceptance is owed at implementation.

## Design direction

Experienced operators use this as a fast switch between an agent's private chat, peers and shared
discussion. Room-title link first, compact peer links next; no report hierarchy or decorative cards.
Use existing Chuck tokens, focus states and quiet secondary button construction. Subtle static
depth identifies who currently owns a room turn; no motion. Read-only navigation preserves room
lifecycle, histories and private Send/Steer semantics. Current implementation seams verified:
`RoomTurnNotice.tsx`, `ChatPanel.tsx`, `components/ui/Button.tsx`, `think-tank.css`, and TS-14.R27's
existing room membership query. No provider capability or architecture limitation is assumed.

## Waiting on

Nothing. Implementation and closure finished 2026-10-19.
