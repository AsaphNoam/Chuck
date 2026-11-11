# Think Tank templates and participant setup

**State:** Waiting to start
**Why:** Operator request and follow-up decisions on 2026-11-11
**Relevant requirements:** [FS-21.R60–R63/A43–A45](../specs/features/FS-21-think-tanks.md),
[TS-14.R30–R33](../specs/tech/TS-14-think-tank-control-plane.md), TS-05.R2–R3,
[INV §§1–3, 5–11, 13–17](../features/INVARIANTS.md)

## Outcome

Open a new room from any project, load a familiar team and discussion settings, enter a goal,
adjust any participant, and start a fresh discussion.

## Included work

- Save a named setup from the creation modal or room. Capture portable launch choices from any
  existing chats as well as fresh launch drafts. Load all slots as editable fresh launches initially
  assigned to the new room's originating project.
- Edit any participant row before starting. Optional participant instructions travel with the setup
  and are available to the assigned participant on its room turns.
- Scope covers the desktop creation modal and room view. Pipeline definitions, agent-facing template
  management and a separate template-management surface are outside this change.

## How we will know it works

Exercise FS-21.A43–A45: cross-project save/load/edit/start, saving from a retained room, repair of a
stale launch choice, and correct instruction visibility across turn phases and retries. Verify
durable save replay, legacy create replay, serialized boundaries and keyboard use. Render the
primary setup and recovery journeys in the built app per TS-14.R33.
