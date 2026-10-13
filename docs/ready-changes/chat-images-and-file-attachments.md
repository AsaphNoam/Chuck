# Chat images and file attachments

**State:** In progress
**Why:** Direct `/design-feature` request on 2026-10-12 for image display, file attachments,
drag/drop and pasted images, with small clickable image previews opening a display modal.
The operator approved feature scope and technical delivery/retention/limits, then delegated
additive API judgment: choose it for semantic fit, not speed.
**Relevant requirements:** FS-03.R81–R86/A62–A65; TS-02.R46–R47; TS-03.R62–R63;
TS-04.R90–R91; TS-05.R26; TS-08.R123; TS-13.R26.
**Invariants:** INV §1/§2/§4/§5/§7/§8/§9/§10/§11/§12/§13/§14/§15/§16/§17.

## Outcome

Attach files through a picker/drop or paste images, inspect compact thumbnails in a modal,
and send text-plus-files or attachment-only messages from desktop and paired phone. Native image
inputs, assistant image bytes and explicitly opened local image links render consistently in live
and archived conversations. Other files reach the agent as immutable uploaded copies to read
through normal tools, with existing permissions and an honest distinction from native image input.

## Included work

Extend the shared composer, draft owners, runtime input, event normalization and Markdown renderer.
Retain browser-local unsent bytes under the existing 20-draft policy and accepted copies outside
repositories until conversation deletion. Queue, withdraw, recover, wake, Steer, clone, replay and
archive retain attachment identity and preserve newer drafts. PNG/JPEG/WebP/GIF native previews;
10 files, 10 MiB each and 25 MiB aggregate, with bounded decoding/transport and stricter provider
rejections surfaced. Other image formats remain ordinary files.

Extend existing Send/Steer with multipart rather than inventing a second prompt action. Preserve
text-only JSON clients, share admission/content assembly, add session-scoped owned-byte and explicit
local-image reads, and retain stable request receipts to prevent duplicate delivery on network retry.
Phone multipart must bypass the current JSON buffering only through its narrow authenticated route
exception; all other routes retain their current bounds. No public hosting, remote web image fetch,
image editing, OCR, archive extraction, adapter patch or broader filesystem permission grant.

### Verified seams and limitations

- Current Composer, prompt API and `UserPromptData` are text-only; `@` inserts text under FS-03.R31.
  Extend these seams; do not treat mentions as uploads or replace the existing file viewer.
- `scripts/release/package.json` pins Claude ACP 0.85.1 and Codex ACP 2.1.1. Their bundled
  `dist/acp-agent.js` (`promptToClaude`) and `dist/index.js` (`buildPromptItems`) under
  `scripts/release/node_modules/@agentclientprotocol/` advertise and convert native images.
  Claude ignores embedded binary resources; Codex encodes non-image blobs as text.
- [Current Claude source](https://github.com/zed-industries/claude-agent-acp/blob/main/src/acp-agent.ts)
  still ignores binary resources in `promptToClaude` (checked 2026-10-12).
  [ACP content definitions](https://github.com/agentclientprotocol/agent-client-protocol/blob/main/docs/protocol/v2/content.mdx)
  define image capability negotiation and baseline resource links. Local copied files plus explicit
  ordinary tool reads avoid both binary-data loss and a provider-specific adapter workaround.
- Assistant image conversion exists in the pinned Claude (`toAcpContent`) and Codex generated-image
  output paths, but Chuck's `internal/runtime/acpmap.go` currently normalizes text/diff only.
- `internal/runtime/transport.go` and `testdata/fakeacp/main.go` currently bound frames to 8 MiB;
  TS-04.R91 requires finite image-compatible frame limits. Transcript records stay unchanged.
- `internal/server/remote_routes.go` currently buffers bounded 1 MiB JSON. TS-13.R26 must share
  streaming multipart admission without broadening that limit globally. Local image reads retain
  the stricter tracked-workspace/no-symlink phone boundary, unlike the desktop same-host boundary.

## How we will know it works

FS-03.A62–A65 cover exact wire contents/order, attachment-only Send, modal/focus, failure recovery,
limits/negative inputs, held ownership, idempotency, newer-draft races, reload/restart, archive,
clone/source deletion and phone guards. Run applicable TS-06 closure checks after final edits.
Render populated full chat, expanded panes, archive and phone in the required appearances.
Credentialed pinned Claude/Codex receipts must separately prove native image interpretation,
ordinary copied-file reads and attachment-bearing Steer; source inspection/fake peers are not
live-provider evidence. No live support claim is made by this design.

## Waiting on

None. Ready for implementation; provider receipts and rendered checks are implementation gates.

## Implementation plan

Storage and UI work proceed independently while the runtime owner extends the existing turn gate;
the parent owns HTTP admission, durable receipts and lifecycle integration. Verify each boundary
with focused storage, wire, API and component tests before final closure and rendered/provider gates.

1. Immutable bounded attachment storage, descriptors and safe reads.
2. Runtime message value, shared ACP assembly, image output and held-message lifecycle.
3. Multipart admission, durable request receipts, remote guard exception, clone/delete ownership.
4. Shared browser attachment drafts, composer strip, thumbnail/modal and safe Markdown images.
5. Integrated failure/recovery journeys, rendered appearance matrix and pinned provider receipts.

Storage checkpoint: `internal/attachment` and state migration 50 implement immutable owned bytes
and attachment receipts. Provisional markers allow startup cleanup without deleting retained copies
whose old transcript is corrupt. `internal/attachment` and `internal/state` full suites pass.
Overall attachment behavior and its requirements remain planned while integration proceeds.

Runtime/API checkpoint: the existing message/turn gate now carries ordered descriptors through
Send, hold, wake, Steer and clone, with native image negotiation and bounded output retention.
Shared multipart admission and owned/local image reads retain the desktop/phone guards; durable
receipts prevent duplicate provider delivery. Migration 51 is forward-only from checkpoint 50.
Full runtime/transcript and focused race/API/security tests pass. The full server run had one
Stop/resume conflict flake whose focused rerun passed; final closure remains required.

The experienced operator adds context to an existing conversation and inspects it before sending.
Keep the text entry and Send primary, with a quiet wrapping strip above it; distinguish native image
input from files read through ordinary tools. Preserve keyboard flow, exact newer drafts and chat
position. Use the existing dialog and tokens, with no added motion. Preparation begins on Send;
errors preserve work and held recovery requires explicit delivery.
