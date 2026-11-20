# Message hover actions and Edit in the current conversation

**State:** In progress
**Why:** The operator requested visible message actions, corrected Edit to act on the current
conversation, limited it to the latest sent message, requested Copy and Fork on every message,
and approved completing the design on 2026-11-20.
**Relevant requirements:** FS-01.R60–R62/A42–A44, FS-22.R13/A10;
TS-01.R51, TS-02.R69–R70, TS-03.R90–R91, TS-04.R113, TS-08.R171;
existing FS-01.R36/R38/R58, TS-01.R39/R45–R47, TS-02.R52/R65, TS-03.R86,
TS-04.R65/R101/R110; INV §1–§5, §7–§13, §15–§20, §22–§23.

Fresh-context design review completed 2026-11-20 under workflow §13 with no material findings.

## Outcome

At the top right of a hovered desktop message, show small, muted, borderless Copy and Fork
icons. The latest sent message additionally offers Edit. Keyboard focus reveals the same row.
Fork branches after the selected message's completed turn. Edit replaces the latest person's
message and regenerates its reply in the same Chuck conversation.

## Included work

- Agent page and dashboard chat pane, sharing hover and context-menu action descriptors.
  Reuse the icon, tooltip, focus, textarea, clipboard and confirmation primitives and semantic
  tokens. Preserve message reading and the ordinary composer draft. Copy includes Markdown
  source and reports success or failure.
- Inline Save and regenerate and Cancel, retaining attachments. Treat the latest person's
  message as the editable one, including after its reply. Preserve earlier conversation and
  retain superseded raw history for the conversation's lifetime. Explain the effect on the
  reply and already-performed tool/file actions in the editor.
- One accepted-input receipt and one related bounded edit transition. Reserve idle execution,
  prepare and prove candidate context, commit an append-only revision, publish the revised
  session/history, then submit once. Use existing same-agent start, teardown, journal,
  attachment and request-recovery mechanisms. The opaque native session can change while the
  Chuck agent id, route and configuration remain the same.
- Extend the packaged fork patch to resolve and attest a complete retained native turn.
  First-message Edit starts from the frozen initial context. Legacy/unproven adapters expose
  an unavailable reason. Existing Clone and clone-and-compose clients retain compatibility.
- Revision-aware transcript readers, phone rendering, previews, search, context, fork-prefix
  construction, reconnect and operator follow. Preserve immutable source references.
  Operator edit-last and edit-last-status use the same local-operator API.

Phone and terminal gain no message action row. The edit changes conversation context and does
not undo files, external tool effects, task results or delivered mail. Earlier user messages
remain accessible through Fork. Native direct rewind, arbitrary history editing, provider
replacement, a branch browser and new retention controls are outside this change.

## How we will know it works

- FS-01.A42/A44 verify Copy/Edit/Fork visibility, muted icon treatment, clipboard content,
  disabled reasons, tooltips, hover-to-control movement and keyboard behavior on both desktop
  surfaces, long content, the 1024px desktop floor, every supported appearance and reduced motion.
- FS-01.A43 verifies latest-message and first-message Edit, retained attachments, stopped chats,
  repeated edits and provider-context boundaries. Reconnect, phone, operator follow, search and
  context must agree on the active revision. Immutable references still resolve retained bytes.
- Failure-inject after receipt acceptance, native candidate creation, provisional process start,
  the SQLite commit decision, uncertain journal append, session publication and prompt dispatch.
  Race ordinary/autonomous input, lifecycle replacement, Stop and turn Cancel. Verify one
  revision, one receipt owner, no duplicate prompt, and a useful recovery action.
- FS-22.A10 proves command parity through the same API and real receipt behavior.
- Run the applicable TS-06/workflow closure matrix after implementation, regenerate embedded UI
  through make embed, and obtain bounded packaged Claude/Codex receipts. A provider receipt
  proves retained context, omitted old reply and complete kept-turn tool/structured payloads.
  Fake providers and a text fingerprint alone cannot close those provider gates.

## State and ownership

The receipt owns replacement input. The server's related edit record owns the transition until
handoff to ordinary runtime turn admission. The journal owns the immutable revision boundary.
The SQLite committing decision is irreversible; readers and admission reconcile it before
observing a complete revision. Each operation uses the ordinary startup deadline. Failure to
finish committed recovery exposes its reason and directs the operator to Resume to retry context
recovery; input remains blocked until that recovery succeeds. A prepared but uncertain send uses
the existing explicit receipt-recovery path. Competing new input is refused before acceptance
while the edit owns its reservation.

| State | Restart | Stop or turn Cancel | Owner replaced | Input refused or failed | Quota or timeout hold | Late callback from previous owner |
|---|---|---|---|---|---|---|
| Preparing, no commit decision | Aborted / recovery server + receipt / original history and recoverable edit | Aborted / edit owner + receipt / Stop leaves source stopped; Cancel restores prior running/stopped condition | Aborted / new recovery owner + receipt / original selection restored | Aborted / edit owner + receipt / original history and recoverable edit | Aborted / edit owner + receipt / preparation reason shown, original retained | Preparing or aborted / current edit owner / stale result discarded; candidate cleaned up |
| Committing, durable decision | Committed / recovery server + journal + receipt / revised context completed before input | Committed / edit owner + receipt / finish revision; Stop leaves it stopped, Cancel releases pending input | Committed / new recovery owner + receipt / finish recorded revision before activation | Committing with attention / recovery server + receipt / revised decision retained, Resume repairs context | Committing with attention / recovery server + receipt / bounded error and Resume repair | Committing or committed / recorded current owner / stale result cannot alter decision |
| Committed, dispatch not started | Committed / recovery server + receipt / submit once under reservation | Committed, input released / receipt / revised chat retained and response stopped | Committed / new runtime + receipt / handoff once after context proof | Committed, input released or failed / receipt / revised history and recoverable text | Committed, input held or failed / receipt / normal explicit hold/recovery shown | Committed / current owner + receipt / stale callback ignored |
| Dispatch started, outcome pending | Committed / receipt + recovery server / submitted if proven; prepared becomes unknown | Committed / runtime + receipt / ordinary revised-turn cancellation or Stop | Committed / receipt + new runtime / recover outcome, never automatically resend unknown | Committed, released/failed/unknown / receipt / existing recovery explains actual delivery | Committed / runtime + quota owner + receipt / normal revised-turn quota handling | Committed / captured turn owner + receipt / only matching generation and turn can settle |
| Committed and settled | Committed / session + journal / revised active history restored | Committed / ordinary runtime / normal Stop or idle Cancel | Committed / new runtime / revised history loaded | Committed / ordinary receipt / future refused input has no effect | Committed / ordinary quota owner / future holds use revised context | Committed / current runtime / superseded callback ignored |
| Aborted | Aborted / receipt + original session / original active history, edit recoverable | Aborted / original runtime + receipt / ordinary Stop or Cancel | Aborted / original-session recovery owner / original restored | Aborted / receipt / preserve failure and recoverable edit | Aborted / original runtime / ordinary future holds | Aborted / receipt / old candidate cannot publish or submit |

## Verified seams and provider boundary

Read-only research on 2026-11-20 verified the shipped clone endpoint allocates a new agent
(internal/server/clone.go), while runtime.openFork verifies replay and switch.go already owns
same-agent frozen start and rollback composition. clone.writeForkPrefix and transcript.Discard
cannot be used for same-agent history replacement because their ownership is a new agent.
The implementation extends the single journal and receipt paths, adding a revision fold.
The current recovery Send path (internal/server/attachments.go) creates a prompt successor.
Extend that existing handoff to preserve Edit intent before commit and resume only the
replacement input after commit, as TS-02.R69 requires.

Pinned artifacts are Claude ACP 0.85.1, Claude Agent SDK 0.3.286, Codex ACP 2.1.1 and Codex
0.159.3. Claude's fork-session.js resolves an ACP assistant id to an SDK fork UUID; Codex's
bundle resolves to lastTurnId. The Claude SDK's sdk.d.ts:2092–2150 documents trailing
tool_result/structured_output records and guarded resumeSessionAt/resumeDropsTurn. ACP's
acp-agent.js:369–376 excludes those resume controls from passthrough.
TS-04.R113 therefore requires complete-turn boundary proof, including private initialize and
fork-result metadata and adapter contract tests.

Direct native rewind is a viable alternative with additional adapter work. Codex's
[pinned app-server contract](https://github.com/openai/codex/blob/rust-v0.159.3/codex-rs/app-server/README.md#thread-rollback)
replaces thread/rollback with thread/revert. Upstream main identifies
[Claude ACP 0.109.0](https://github.com/agentclientprotocol/claude-agent-acp/blob/main/package.json)
and [Codex ACP 2.8.0](https://github.com/agentclientprotocol/codex-acp/blob/main/package.json);
this is source-version evidence, not a claim about a published package or portable ACP rewind.
The prepared fork was chosen to preserve the source context until durable commit and reuse the
existing fork proof and lifecycle seams. These are design-source checks; live provider receipts
remain implementation acceptance work.
