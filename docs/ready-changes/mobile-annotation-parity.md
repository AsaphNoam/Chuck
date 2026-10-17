# Mobile annotation parity

**State:** In progress
**Why:** Operator request on 2026-10-17, `/design-feature add annotation functionality to mobile
as well`, confirmed as “essentially it should have the same functionality as annotations do on Mac.”
**Relevant requirements:** FS-13.R28–R31/A19–A21, FS-20.R55/A27, TS-08.R133–R136, TS-13.R31;
existing FS-13.R1–R25, TS-02.R14/R38, TS-03.R14/R49, TS-13.R16/R22/R23;
INV §1/§2/§8/§10/§11/§13–§17.

## Outcome

An operator can collect conversation, diff and tracked-file annotations on the phone, review one
batch and assign it with the Mac's capture, instruction, validation, launch, retry and recorded-card
behavior. Touch controls and a phone tray adapt presentation to the smaller screen.

## Included work

Extend existing event/file capture helpers, the annotation store, delivery hook, PhoneSheet and
phone New agent form. Support whole eligible events, highlighted excerpts, source/rendered tracked
file selections, the optional overall instruction and all three existing destinations. Keep one
source-owned controller across Chat/Files and tray/launch sheets; render ordinary annotation cards
live and after replay. Preflight precedes configurable launch, cancellation preserves drafts and a
post-launch delivery failure retries against the created agent. Keep remote reads restricted to
tracked files; the normalized captured path never replaces the exact tracked read key.

No phone archive/Think Tank screens, terminal/worktree controls, mailbox UI, arbitrary file browsing,
new route, record shape, storage migration or retention policy are included. Existing Mac interaction
and phone navigation remain governed by their current specifications.

## How we will know it works

FS-13.A19–A21 and FS-20.A27 cover exact mixed anchors/excerpts, limits, draft lifetime, keyboard-open
editing, tab/focus/reading-place recovery, all destinations, launch cancellation/refusal,
append/delivery failure, same-agent retry and live/replayed cards with duplicate-prompt suppression.
Use component/store/remote-chain tests, actual source events and recipient prompt/mail assertions,
then a fake-provider phone-entry browser journey at 360/390/430px in Core, Sky & Grove and Studio.
Paired iPhone/Android native selection and keyboard behavior remain explicit device acceptance gates;
emulated browser receipts cannot close them. Run the applicable TS-06 product closure once after
final implementation, including style/presentation checks and generated UI embedding. This design
has no runtime or rendered acceptance receipt for the new behavior.

## Design direction and verified seams

Experienced operators point at exact work and assign a correction without retyping its location.
Keep the conversation/file excerpt primary, event Annotate actions quiet but touch-reachable, and
the pending-count control visible from Chat and Files. A bounded PhoneSheet reads count/close,
anchored drafts/instructions, then destination/error/send. Preserve native selection and the reading
place; use existing semantic tokens, technical path typography and phone button targets. Success
names the recipient; failures preserve work and make retry's destination clear. No new motion.

Code inspection confirms reusable `canAnnotate`/`selectionWithin`/`eventDraft` in `TranscriptView`,
file selection/line mapping in `FileViewer`, `DiffBlock`, `annotationStore`, `annotationBatch`,
`useAnnotationDelivery`, `AnnotationCard` and `remote/ProjectScreen`'s New agent form. The phone's
current `AnnotationForm` launches before preflight, its `EventRow` omits annotation cards, and the
tray mounts only under Chat; these are owned by this parity change. TS-03.R14 and TS-13.R16 already
admit shared batch validation/delivery; R22 remains the read boundary. Incumbent Files composition
was inspected in `/tmp/mobile-effort-20261014-core-files.png`; actual new rendering is still owed.
No provider capability limitation or workaround is assumed.

## Slices

1. (Done) Shared event-capture helpers in `ui/src/lib/annotations.ts`; `useAnnotationDelivery`
   (`components/chat/AnnotationTray.tsx`) takes a source/recipient projection so desktop and phone
   share validate/send/launch/retry. Desktop tests unchanged.
2. (Done) Phone tray: one delivery owner above Chat/Files in `remote/AgentScreen.tsx`, pending-count
   buttons opening a `PhoneSheet`; New task preflights then opens the `ProjectScreen` New agent
   form (extracted, source-prefilled, project chooser, launch callback); same-agent retry;
   navigate only after delivery.
3. Phone capture: event **Annotate** action, `AnnotationCard` rows, tracked-file preview through
   `FileViewerView` with Source/Rendered selection capture.
4. CSS, `phone-render.mjs` journey at 360/390/430 in three appearances, closure + `make embed`.

## Waiting on

Nothing.
