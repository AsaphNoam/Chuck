# FS-13 — Annotate and assign

**Status:** Partial
**Code:** `ui/src/components/chat/`, `ui/src/features/archive/`, `internal/server/`, `internal/runtime/`, `internal/state/` · **Journeys:** J13
**Absorbed:** —

## 1. Purpose

Instead of describing a location in prose, a person points at the thing itself: select lines inside
a rendered diff or a whole transcript event, attach a short instruction, and send the result to the
current agent, another running chat agent, or a newly launched agent. Chuck preserves each
annotation as structured, located context — captured excerpt, anchor, instruction, target — never as
hand-pasted chat text. The chat surface belongs to FS-03, the archived view to FS-05, mail delivery
to FS-06, and launch to FS-01; this spec owns the annotation interaction, its records, and its
delivery behavior. Agent-session and room-source requirements through R27 are shipped;
R28–R31/A19–A21 extend phone capture, batch controls and delivery presentation; they shipped 2026-10-17.

## 2. Behavior

Requirements are user- and API-observable. R-item numbering is continuous through §4.

### 2.1 Selecting and capturing

- **R1.** In a live chat transcript and in the archived read-only transcript view, a user
  can select a contiguous line range inside a rendered diff block, or highlight text inside one
  transcript event (for example a message, tool call, tool result, or error) — or take the whole
  event — and choose **Annotate** on the selection. The transcript carries no standing per-event
  **Annotate** control; the action is reached by right-clicking the event (R19).
- **R2.** Annotating captures a structured record: the source `agent_id`, the anchored
  transcript event `seq`, the diff's file path, side, and 1-based line range when the selection is
  diff lines, the selected excerpt text verbatim (clipped at 2,000 characters with a visible
  truncation marker), and a required instruction of at most 2,000 characters. The user never types a
  file path, line number, or location description; Chuck derives the anchor from the selection.
- **R3.** Annotations accumulate in a pending tray scoped to the source session. The tray
  shows each entry's excerpt and instruction and supports editing an instruction, removing an entry,
  and discarding the tray. It holds at most 20 entries. The tray is per-browser draft state: it
  survives a reload of the same browser, is not visible from other browsers or the API, and creates
  no durable server data until send. A successful send clears it; a failed send preserves it.
- **R4.** The tray is sent as one batch to exactly one target — the current agent, another
  running chat agent, or a new task (R9) — with an optional overall instruction of at most 2,000
  characters.

### 2.2 Delivery

- **R5.** Every successful send appends one durable `annotation` event to the **source**
  session's transcript — whether that session is active or inactive — recording the batch and its
  target. The event is appended **before** the batch is delivered, so no agent ever receives
  annotations the source transcript does not record. When delivery itself fails after that append,
  the batch is undelivered and the tray is preserved (R3); re-sending it delivers exactly once and
  records a second annotation event for the retried batch. Live chat and
  archive replay render it as quoted-excerpt annotation cards with their
  instructions, not as pasted user prose; rendering is sanitized under the same rules as other
  transcript content (FS-03.R20).
- **R6.** Target **current agent** requires the source agent to be a running, idle chat
  agent. The send starts one ordinary prompt turn (FS-03.R8) whose agent-visible content is a
  machine-generated annotation block: for each annotation, the file path and line range when
  present, the quoted excerpt, and its instruction, plus the overall instruction. A busy,
  waiting, stopped, or terminal source returns a typed error and the tray is preserved.
- **R7.** Target **another agent** requires a running chat-interface recipient
  (FS-06.R4, FS-06.R17). The batch is delivered as coordination mail from the reserved user sender
  (FS-06.R21) with the annotation block as its body; normal unread indicators, idle nudging, reading,
  and retention rules apply unchanged. A stopped or invalid recipient returns a typed error and the
  tray is preserved.
- **R8.** Mail delivery may clip excerpts (with a visible marker) to fit the FS-06.R5 body
  limit; anchors and instructions are never clipped. A batch whose anchors and instructions alone
  cannot fit returns `422 validation` naming the constraint, and the tray is preserved.
- **R9.** Target **new task** opens the existing New Agent modal prefilled with the source
  agent's role and project (name auto-suggest per FS-01.R1/R4). When that launch succeeds, the batch
  is delivered to the new agent as reserved-sender mail (R7), whose idle-nudge wakes the fresh agent
  so processing the annotations is its first activity, and the source session's annotation event
  (R5) records the new agent as target. Cancelling the modal preserves the tray. The batch is
  validated for mail delivery before launch; an invalid batch leaves the tray open without creating
  an agent. If delivery fails after launch, the tray names the launched agent and retries delivery
  to that same agent without reopening the launch modal.

### 2.3 Structure and search

- **R10.** Annotation instructions and captured excerpts are transcript content: on an
  FTS5 build they are searchable from the archive (FS-05.R25-R27); the untagged fallback build limits
  search per FS-05.R6.

**R32 — Annotation delivery retries reuse one command.** Desktop and phone allocate
one stable command ID for an attempted batch before its first request and keep it through lost
responses, reload and launched-recipient recovery. TS-03.R74 binds it to the exact canonical source,
target and payload hash; the same ID returns the retained result without duplicate source event,
mail, self prompt or task. Changed payload under the same ID conflicts without effects; a deliberate
new delivery gets a new ID. Temporary derived-index failure is an explicit degraded projection of
the committed command, not permission to redeliver. Room annotations retain their existing command
identity and authority; no common receipt grants broader target/source access.

## 3. States & transitions

- **Tray:** empty → drafting (add/edit/remove entries) → sending → cleared on acknowledged send. A
  failed or rejected send returns to drafting with all content intact.
- **Tray presentation:** a drafting tray is docked or floating purely as a function of the
  current transcript width (R20), so a resize moves it between the two forms without touching the
  drafts. A docked tray is additionally expanded or collapsed (R21); leaving the docked form and
  returning restores the remembered collapsed flag. Clearing the tray ends both.
- **Delivery:** current agent → one prompt turn under the normal FS-03 busy lifecycle; another agent
  → unread mail following FS-06 pending → nudged → read; new task → prefilled modal → launch →
  reserved-sender mail nudges the new agent's first turn.
- **Durability:** the annotation event follows the source transcript's lifetime; a mail copy follows
  FS-06.R8 retention independently.

## 4. Edge cases & errors

- **R11.** An empty tray, an empty or whitespace-only instruction, or an over-limit
  instruction, excerpt count, or overall instruction returns `422 validation` naming the violated
  constraint. An unknown source agent returns `404`.
- **R12.** Annotations are point-in-time captures. The excerpt is authoritative for what
  the user saw; delivery does not re-read the file on disk, and the anchor may reference content
  that has since changed. No revalidation or live link is implied.
- **R13.** Terminal-interface agents are neither an annotation surface nor a valid target
  (FS-03.R22, FS-06.R17). Files on disk, screenshots, and rendered web pages are not selectable
  surfaces; no such viewing surface exists in the product.
- **R14.** There are no comment threads, replies, or resolve/re-open workflow. Once
  recorded, an annotation event is immutable like every other transcript event.
- **R15.** From an archived (inactive) source session, the current-agent target is
  unavailable; another-agent and new-task targets remain, and the tray selects one by default
  (another agent when a running chat agent exists, otherwise new task) rather than leaving the
  disabled current-agent choice selected. The archived view stays composer-free
  (FS-05.R14); annotating never sends a prompt to the archived session itself.
- **R16.** Pending trays are bounded browser-local drafts, not durable data. Deleting an
  agent discards its tray, because no target can accept a batch from a source that no longer exists
  (R11). Independently, stored trays expire 30 days after their last edit and only the 20
  most-recently-edited sources are retained; both are applied when the browser reloads the stored
  trays. This keeps a session's own tray across reloads (R3) — including an archived session's,
  which is not in the live agent list — while bounding what accumulates in browser storage. If a
  retained tray's source can no longer be found, its missing-agent route identifies the pending
  draft count and offers a direct discard action rather than trapping the retained slot. Because
  that discard is destructive, the route treats a source as missing only after the browser's first
  agent hydration has completed; until then a deep link or reload shows a loading state and offers
  no discard.
- **R17.** A selectable diff block labels its line numbers as the selection control and presents
  them with a pointer cursor. Clicking one line starts a selection; clicking another line on the
  same side extends the contiguous range before **Annotate** captures it (R1–R2).
- **R18.** The pending tray keeps its header and delivery controls visible within its bounded
  overlay while the draft list and overall instruction scroll independently. Multiple detailed
  drafts therefore cannot push target selection, errors, or **Send annotations** below the tray's
  visible edge.
- **R19.** Right-clicking an annotatable transcript event opens a context menu holding the
  **Annotate** action instead of the browser's own menu. When the right-click follows a text
  highlight inside that same event, the menu captures the highlighted text as the excerpt; with no
  highlight in that event, it captures the whole event as before. The menu closes on **Escape**, on
  a click outside it, and after the action is taken. Events that cannot be annotated (R13 terminal
  surfaces, and the session, permission-resolution, turn-end, and annotation events) keep the
  browser's own menu.
- **R20.** While a source session holds at least one pending draft and its transcript
  region is at least a defined width, the tray renders as a full-height column along the right edge
  of that region and the transcript reflows into the remaining width instead of being overlapped by
  it. The width is measured on the transcript region itself, not on the browser window, so the
  dashboard's embedded chat pane (FS-02.R55) keeps the bounded floating overlay even inside a wide
  window while the agent page docks. Below the threshold the tray keeps that overlay unchanged.
  R18's guarantee — header, target selection, errors, and **Send annotations** stay visible while
  the draft list and overall instruction scroll — holds in both forms.
- **R21.** A docked tray offers a collapse control that reduces it to a narrow strip
  naming the pending draft count and returns the freed width to the transcript; expanding restores
  the column. The collapsed flag is browser-local per source and is stored, expired, and capped with
  that source's tray (R3, R16), so it survives a reload, is invisible to other browsers and to the
  API, and creates no durable server data. The overlay form offers no collapse control. Sending or
  discarding the tray removes the column and returns the transcript to its full width.
- **R22.** In the docked form each pending draft presents its anchor as its own heading
  element, distinct from the draft's controls; presents its excerpt with room to wrap rather than
  clipped to the overlay's width; and presents an instruction field taller than the overlay's.
- **R24.** With the rename (FS-00.R19) the annotation block's first line is written as
  `[Chuck annotations]`. Recognition accepts both that line and the `[AgentDeck annotations]`
  line every block written before the rename carries, so annotations already recorded in a
  transcript keep rendering as annotation cards and keep being suppressed under R23 instead of
  reappearing as bare prompts. The old spelling is never written again. This dual recognition is
  permanent: those transcripts are append-only (TS-02.R8) and are not rewritten.
- **R23.** In live chat and in archived replay, the transcript does not render the machine
  annotation block a self-targeted send produces (R6): a user prompt event carrying that block and
  immediately following its own `annotation` event is not drawn, because that event's card already
  shows the same excerpts and instructions. The
  suppression is display-only: the event is still appended, still replayed, still returned by the
  transcript API unchanged, and still searchable (R10), and the target agent still receives the
  block verbatim. Because it is evaluated when the transcript renders, sessions recorded before this
  behavior shipped are quieted too. A prompt a person typed, a prompt that does not immediately
  follow an annotation event, and a prompt following an annotation event that was sent to a
  different target all render normally — including the message a person types straight after
  assigning a batch to another agent.
- **R25 (shipped 2026-09-30) — A loaded conversation file is a first-class annotation source.** This
  supersedes R13's file exclusion when it ships; terminal agents, screenshots, and unrelated web
  pages remain excluded. A file annotation is anchored to the normalized path shown by the viewer,
  not to the transcript event that happened to open it, so a link, a rendered-file link, a reload,
  and a hand-edited `?file=` address produce the same record. Source-mode capture adds the
  contiguous 1-based line range containing the exact selection; rendered-Markdown capture keeps
  the path and selected excerpt but no line range. Neither invents a transcript-event sequence or
  a diff side, and neither offers whole-file capture. File drafts join the source session's existing
  bounded browser-local tray and use every existing edit, target, send, failure, retention, and
  archive rule unchanged. The captured excerpt and anchor are point-in-time under R12: reload,
  replacement, file edits, deletion, or losing read access never changes or invalidates a draft.
  Sending makes the selected excerpt and path durable in the source transcript's annotation event,
  makes the excerpt and instruction searchable under R10, and exposes the record to the chosen
  target through the existing prompt or mail delivery; the file itself is never reread and
  unselected content is never persisted.

- **R26** — Think Tank room contributions, displayed diffs and loaded conversation
  files are annotation sources under FS-21.R26. The familiar selection, excerpt/instruction bounds,
  point-in-time capture and bounded browser-local tray behavior apply, with drafts scoped to the
  room source. Captures identify the actual room entry and originating participant/project when
  applicable; a room entry is not represented as an unrelated event in a participant's private
  transcript. Equal relative paths in different participant workspaces remain distinct. Successful
  room-source sends record the annotation in the canonical room history before delivery; failure
  preserves the tray. R27 defines room delivery targets and completed-room behavior; this does
  not change R1–R25's ordinary agent-session contract.
- **R27** — A room-source batch can target Room, a selected agent, or New task under
  FS-21.R30. Room delivery is shared user input between participant turns, preserving correct source
  attribution rather than privately prompting all participants. Selected-agent delivery uses the
  normal recipient availability and annotation delivery contract. New task retains R9's meaning:
  launch a new normal agent through the existing New Agent flow and deliver the batch as its initial
  work, with the chosen normal role/project/runtime configuration. Cancelling launch preserves the
  tray. After discussion ends, selected-agent and New task delivery remain available; Room delivery
  is unavailable and cannot silently resume the room.

- **R28 — Phone selection supports the existing agent-session sources.** On a
  chat-interface agent's phone screen, a touch-accessible **Annotate** action captures highlighted
  text within one eligible transcript event, or the whole event when that event has no highlighted
  text. The same event eligibility and derived event anchor as R1/R19 apply. This supersedes
  R1/R19's right-click-only entry on the phone; desktop entry is unchanged. Native text selection,
  ordinary link taps, and transcript scrolling remain usable. Existing diff-line selection keeps
  its current interaction. A selection in the phone's loaded tracked-file text captures the shown
  file path and contiguous 1-based line range under R25, without whole-file capture or a fabricated
  event sequence. A tracked Markdown preview offers Source and Rendered selection with R25's
  line-range and path-only anchors respectively. Selection text and its anchor are captured before
  moving focus to an instruction field; subsequent transcript updates, file changes, or closing the
  preview do not change them.
- **R29 — Phone sources share one reachable batch.** Transcript, diff, and file drafts
  from the same agent accumulate in its existing browser-local tray, with the same entry and text
  limits, instruction editing, removal, discard, reload survival, expiry and source-deletion rules.
  Both Chat and Files expose the pending count and a way to open that batch without losing the
  person's reading place. The phone also offers R4's optional overall instruction. The draft list
  can scroll while target selection, errors and send remain reachable with the phone keyboard open.
  Sending uses the existing current-agent, another-agent and new-task destinations, availability
  rules, source recording and prompt/mail content; no new agent/API record shape or retention is
  introduced. Acknowledged success clears the submitted batch and names the destination; a draft
  captured while that send was pending stays in the tray. A rejected or failed
  send preserves all drafts and the overall instruction. Disconnection disables delivery without
  discarding drafts or sending automatically on reconnect.
- **R30 — Phone New task has the Mac's reviewable launch step.** A phone batch's
  **New task** destination follows R9: validate the batch and mail size before opening New agent,
  prefill the source's persona and project, and let the person review or change name, persona,
  existing project, group and the phone-supported backend/model/effort/fast choices before launch.
  This supersedes FS-20.R32's immediate default-only launch. It remains a normal chat-agent launch,
  with no terminal or worktree controls. While a launch request is pending, the launch step cannot
  be cancelled or dismissed, so one batch never starts two launches. Cancelling returns to the
  intact tray; launch failure
  preserves both batch and entered launch values. After launch, delivery failure names the created
  agent and offers retry to that same agent without another launch. Only acknowledged delivery
  clears the batch and opens the new agent's conversation.
- **R31 — Phone conversations show recorded annotation cards.** Annotation events in
  a phone conversation render the same structured target, anchor, quoted excerpt, individual
  instructions and overall instruction as Mac live/replay under R5. R23–R24's display-only
  suppression of a self-targeted machine prompt applies unchanged, including legacy blocks.
  Receiving an annotation card does not create a draft or a new selectable annotation source.

- **R33 — Desktop annotation entry.** Instruction and overall instruction fields use the
  regular application text font at the standard input size. A newly mounted draft instruction
  receives keyboard focus in agent and room trays so typing can begin immediately. Editing
  existing fields does not move focus.

- **R34 (planned) — Project-first recipients.** Desktop/phone annotation recipient lists
  default to the source agent's project, or the originating project for room annotations.
  An unchecked **Include other projects** control explicitly reveals other eligible agents,
  labelled by project. Reset the control on source change/reopening; hiding a selected outside
  recipient clears that selection. Existing eligibility, delivery and New task behavior remain.

## 5. Acceptance criteria

Each acceptance item names its delivered verification.

- **A1** (R1–R4) — Selecting diff lines and a whole transcript event captures anchors and
  excerpts without typing a location; the tray edits, survives a reload, and clears on send:
  `ui/src/store/annotationStore.test.ts` plus chat component coverage.
- **A2** (R5–R6) — A self-targeted batch produces one prompt turn and one durable
  annotation event that renders as annotation cards identically live and after reload: focused
  fake-ACP integration check plus transcript-store replay coverage.
- **A3** (R7–R8) — Assigning to a second agent inserts reserved-sender mail that raises
  the unread badge and nudges an idle recipient; an unfittable batch is rejected `422` with the tray
  preserved: `internal/server/annotations_test.go` and existing messaging/state nudge coverage.
- **A4** (R9) — New task opens the prefilled New Agent modal only for a valid batch and, after launch,
  the new agent's first activity is processing the delivered annotations. A failed delivery leaves
  the tray intact and retries against that same agent: `AnnotationTray.test.tsx`,
  `NewAgentModal.test.tsx` launch coverage, and the annotation delivery integration test.
- **A5** (R10) — An annotation instruction is findable through archive search on the FTS5
  build: `internal/index/indexer_test.go::TestAnnotationFlushesSearchableContent`.
- **A6** (R1–R15) — A user annotates a diff and a message, sends to self, to a second
  agent, and to a new task, and sees structured cards and mail arrive: journey **J13** in
  `docs/features/USABILITY-REVIEW.md`.
- **A7** (R5) — A failed source-transcript append delivers no mail, and the preserved tray's
  retry delivers exactly one copy:
  `internal/server/annotations_test.go::TestAnnotationAppendFailureDeliversNoMailAndRetrySendsOnce`.
- **A8** (R16) — Deleting an agent drops its pending tray, stored trays are capped and expired on
  rehydration, and a missing-agent route can discard its retained tray only after hydration
  completes (pre-hydration, present-after-hydration, and absent-after-hydration are covered):
  `ui/src/store/annotationStore.test.ts`, `ui/src/api/sse.test.ts`, and
  `ui/src/components/chat/ChatPanel.test.tsx`.
- **A9** (R1–R2, R17) — A diff block visibly explains line-number selection and applies the pointer
  affordance while the library-shaped id regression captures the selected range:
  `ui/src/components/chat/renderers/DiffBlock.test.tsx`.
- **A10** (R3–R4, R18) — A tray with three drafts renders them in the scrollable body and keeps its
  target and Send action in a separate fixed footer: `ui/src/components/chat/AnnotationTray.test.tsx`.
- **A12** (R20–R21) — With drafts pending, a wide transcript region renders the tray as a
  docked column beside the transcript and a narrow one renders the floating overlay; the collapse
  control reduces the column to its pending-count strip, expands again, and the collapsed flag
  survives a reload and is discarded with its tray, including when the tray ends by having its
  final draft removed:
  `ui/src/components/chat/AnnotationTray.test.tsx` and `ui/src/store/annotationStore.test.ts`.
- **A13** (R22) — A docked draft row renders its anchor as a heading element separate from
  its controls, alongside the excerpt and the instruction field:
  `ui/src/components/chat/AnnotationTray.test.tsx`.
- **A15** (R24) — A transcript containing an annotation block written with the old
  `[AgentDeck annotations]` first line renders as an annotation card and stays suppressed under R23,
  while a newly sent annotation writes `[Chuck annotations]` and behaves identically: a fixture
  transcript holding both spellings renders both, and no code path emits the old line.
  *Verified:* `ui/src/components/chat/TranscriptView.test.tsx` extended with a legacy-prefix fixture
  and an emitter assertion in `internal/runtime`.
- **A14** (R23) — A self-targeted send renders the annotation card and no user message
  carrying the annotation block, identically live and after a replay, while the transcript endpoint
  still returns that prompt event: `ui/src/components/chat/TranscriptView.test.tsx` and
  `internal/server/annotations_test.go`.
- **A11** (R1, R19) — No standing **Annotate** control renders on transcript events; a right-click
  after highlighting captures the highlighted text, a right-click with no highlight captures the
  whole event, and a non-annotatable transcript keeps the browser menu:
  `ui/src/components/chat/TranscriptView.test.tsx`.
- **A16 (shipped 2026-09-30) (R2–R12, R16, R19–R23, R25)** — Source and rendered-Markdown file selections
  create path/line and path-only drafts respectively in the same tray as transcript drafts; the
  mixed batch survives reload, edits and removes normally, preserves excerpts when the file changes
  or disappears, and sends through the existing self, other-agent, and new-task paths. Its durable
  annotation card, agent block, mail, and archive replay name a file anchor rather than `Event 0`,
  while existing transcript and diff records remain byte-compatible and render unchanged. Empty or
  invalid file anchors return the structured validation error with the tray preserved. *Verify by*
  file-viewer/tray/store tests, annotation endpoint and formatter compatibility tests, transcript
  live/replay tests, and journey J13 with a mixed transcript/file batch.

- **A17** (R26) — Annotate a Think Tank message, a displayed diff selection, and a loaded
  file excerpt from participants in different projects, including identical relative paths. Reload
  with drafts pending, then exercise failed and successful delivery. Verify preserved excerpts,
  distinct source attribution, bounded drafts, failure preservation, and a canonical room annotation
  recorded before successful delivery. Ordinary agent-session annotation checks continue to pass.
  *Verified by:* room annotation integration/component tests and the FS-21.A15 rendered journey;
  A18 additionally covers target routing and completed-room cases.
- **A18** (R26–R27) — Send room-source batches to Room, a selected agent, and New task.
  Verify that Room input is shared at a participant-turn boundary, selected-agent delivery reaches
  the chosen normal session, and New task creates a new normal agent with the batch as its initial
  work. Cancel the new-agent flow and preserve the tray. Complete the room and repeat selected-agent
  and New task sends while Room delivery remains unavailable. Ordinary session-source destinations
  are unchanged. *Verified by:* room annotation/launch integration tests and FS-21.A20's rendered
  three-destination journey.

- **A19 (planned)** (R28–R29) — At 360/390/430px, capture selected text and a whole eligible phone
  transcript event, a diff range, and source/rendered-Markdown text in opened tracked files into one
  batch. Verify exact excerpts and event/file/line anchors, exclusion of ineligible events, continued native selection,
  link taps and scrolling, editing/removal and overall instruction, Chat/Files access with reading
  position preserved, reload survival and point-in-time capture after new events or changed files.
  Check that keyboard-open editing leaves delivery and errors reachable. *Verify by:* phone
  component/store tests and a rendered touch-input journey in all three appearances; real paired
  iPhone and Android selection/keyboard checks remain explicit device gates.
- **A20 (planned)** (R28–R31) — Send a mixed phone batch to the current idle agent, another running
  chat agent and a newly launched agent; check the same structured source cards and target
  prompt/mail as desktop, including after replay. Exercise current-agent refusal for busy/stopped
  sources, invalid recipient, validation failure, source-append failure, offline/reconnect and failed delivery after
  launch. Drafts and overall instruction survive failures, reconnect sends nothing automatically,
  and a post-launch retry targets the same launched agent. *Verify by:* phone delivery component
  tests, annotation endpoint/formatter regression tests and a fake-provider phone browser journey.
- **A21 (planned)** (R30–R31) — An invalid batch creates no agent and opens no launch dialog.
  A valid New task opens the source-prefilled form; change persona/project/name/group/runtime,
  cancel and reopen, then exercise launch failure and delivery failure after successful launch.
  Verify preserved drafts, editable launch values after launch refusal, same-agent delivery retry,
  and navigation only after successful delivery. Render transcript, diff and both file-anchor forms
  in phone annotation cards live and after reload, including overall instructions and old/new
  self-targeted blocks, with no duplicate machine prompt. Existing Mac launch and card behavior
  remains covered. *Verify by:* shared delivery and phone launch/card component tests plus the
  rendered fake-provider phone journey.

**A22** (R32) — Discard a successful response, reload/retry the same batch and verify
one canonical annotation plus one target effect for peer mail, self, new-task and launched targets,
desktop and phone. Test payload mismatch, target refusal before effects, index repair and caller
authorization; use real handler/receipt paths rather than a mock that deduplicates only requests.

- **A23** (R33) — Desktop annotation instruction and overall instruction fields use the regular
  input font and size. Adding a draft focuses its instruction immediately, including in a
  room tray; editing an existing draft does not move focus.

- **A24 (planned)** (R34) — With eligible agents in two projects, open agent/room annotation
  recipient lists: only the source project appears until Include other projects is enabled.
  Verify attribution, selection clearing and reset on reopening/source change without losing
  drafts. *Verify:* recipient tests and desktop/phone browser journeys.

## 6. Deviations & open decisions

- R28–R31 shipped 2026-10-17 (`ui/src/remote/AnnotationForm.tsx`, `remote/NewAgentForm.tsx`,
  `components/chat/annotationDelivery.ts`). A19–A21's component tests and the stubbed-API
  `phone-render.mjs --annotation-check` journey (360/390/430px, three appearances) pass; that
  journey uses programmatic selection ranges and a short viewport for the keyboard, so paired
  iPhone/Android selection and keyboard checks keep A19–A21 planned. No phone archive or Think
  Tank surface; remote file reads are not widened.

- R26–R27/A17–A18 shipped 2026-10-06 (`ui/src/features/thinktank/RoomAnnotationTray.tsx`,
  `internal/server/think_tank_annotations.go`); FS-21.R35/R39 and TS-14 govern boundary input and retained room sources.

- The numeric limits in R2, R3, and R4 are initial values and may be tuned only through a
  spec-first update.
- R20's dock threshold and the docked column's width are initial values on the same footing:
  `860px` of transcript region and `min(30%, 460px)` respectively, tunable only spec-first.
- R23 quiets the duplicate prompt but leaves the annotation card's own wording alone. On 2026-09-06
  the operator was offered, and declined, stripping the card's `Event <seq>` anchor and resolving
  its raw target agent id to a name; both remain as shipped.

## 7. Traceability

Delivered anchors: annotation endpoint in `internal/server`;
`annotation` event kind in `internal/runtime/event.go`; `runtime.FormatAnnotationBlock` used by prompt and mail delivery; tray and card components under `ui/src/components/chat/`; reserved-sender mail in
`internal/state/messages.go`; index wiring in `internal/index/indexer.go`; the docked column, its
container query, and the collapsed strip in `ui/src/styles/features/agent.css`; the collapsed flag on
the tray record in `ui/src/store/annotationStore.ts`; prompt suppression in `appendRenderedEvent`
(`ui/src/store/transcriptStore.ts`) against `annotationBlockSentinel` in `ui/src/lib/annotations.ts`.
