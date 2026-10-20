# CR-A4 — Information and operating knowledge

Prepared 2026-10-20 from two focused GPT-6 Luna research passes and parent integration against
`735ea6e5`. This maps **A4 only** from the
[core reliability priorities](CORE-RELIABILITY-AUDIT-PLAN.md). It is one integrated review for a
frontier reviewer. Research inspected requirements, current callers and existing tests; it performed
no audit, failure probes, test execution or product repairs.
[HANDOFF.md](HANDOFF.md#review-queue) owns live status.

## Question and responsibility

Does the intended agent receive the selected information and the guidance needed to operate Chuck,
with source identity, meaning, authority and documented limits intact? Can it discover and use the
advertised capabilities without missing, stale, contradictory or falsely attributed instructions?

Trace selection/source → accepted record or package → access/composition → bounded read or native
input → delivery receipt. Prioritize silent omission or substitution, wrong-recipient/source context,
undisclosed clipping, stale guidance advertising unavailable behavior, and a misleading receipt that
causes lost or repeated work. A retained file, grant, skill pointer or manifest is not proof that its
contents reached the provider. Supplied input is not proof of comprehension or successful work.

A4 owns the information-to-consumer join: text/files/images, annotations/mail, selected context and
assignment reads, managed operating knowledge, tool/result discoverability and instruction delivery.
Inspect Codex and Claude chat and their native history/instruction seams. Include terminal knowledge
composition only where supported; do not invent terminal messaging/task tools. Follow adjacent
systems to establish this promise without repeating their entire reviews.

## Read first

Read the current handoff's Current position and Active change, this charter and related findings.
Use workflow §§3–7 for review/reporting, with this explicit subsystem scope replacing a commit-diff
scope. Read requirements before following source:

- FS-03.R31/R81–R86/R99 and relevant native input/history contracts: mentions versus uploaded
  contents, accepted/recoverable attachments and history refusal. FS-13.R1–R13/R25–R31 covers
  captured annotations, clipping, targets and room/phone sources. Its R32 command-retry extension
  and FS-03.R100 are planned; existing ASF-2 findings retain their identity.
- FS-06.R30–R38/R39–R44 and TS-01.R31–R33: current inline mail, attribution/bounds/receipts,
  direct peer reads and approved control. Later shipped mail rules supersede older pull-only text.
  A2 owns the opportunity/retry policy; here establish which content a turn receives and confirms.
- FS-15.R1–R18, TS-01.R22–R24, TS-05.R16/R17/R22/R25–R27 and TS-10.R12/R27–R31: immutable
  reference locators, grants versus work membership, paging/tombstones and assignment context.
  Read later task/room extensions rather than treating FS-15's original future-work wording as
  absence of shipped task attachments.
- FS-18.R1/R3–R6/R11/R15–R21 and TS-11.R1–R5/R7–R8/R10/R13/R18–R24: managed publication,
  guidance ownership, native inheritance, frozen awareness and requested-delivery manifests.
  TS-04.R69/R101 govern additive native instructions and operation-purpose differences.
- FS-04.R50–R52/R54–R56 and TS-03.R59 define persona content, preservation, defaults and frozen awareness.
  Read FS-08's source-selection/frozen-delivery requirements to establish instruction provenance
  and preservation; A3 owns actual configuration realization.
- FS-17.R1–R12/R21–R22 and current MCP registration contracts in TS-04: result channels, refusal
  classification and capability discovery. FS-17.R13–R20's replacement action transport remains
  planned/paused; audit the released MCP surface.
- Read matching TS-02 conversation/mail/reference/attachment storage requirements when tracing an
  identity or receipt. Read the invariant trigger index, then matching bodies: likely
  §§1/2/3/4/7/8/10/11/14/15/16/17; include §5 for an atomic claim. TS-06.R35 governs proportional
  evidence; adequate existing proof can justify no new test.

Specs govern behavior. This charter, old reports and package prose are navigation or review subjects,
not new product requirements. Do not assume every bounded transformation must preserve raw bytes,
or that every piece of available context must be automatically injected.

## Preserve the product's information boundaries

- Uploaded files are immutable owned copies. Images use negotiated image input; other files give
  the agent an exact owned path to read with ordinary tools/permissions. Acceptance does not promise
  that the model read the file. `@` paths remain ordinary text, not captured uploads. Historical
  images are not automatically resent on runtime switch; native Clone keeps independent copies.
- An annotation's captured excerpt is its point-in-time truth, even if the file later changes.
  Documented clipping marks excerpts while preserving anchors/instructions. Annotation **New task**
  uses the existing agent-launch dialog and reserved-user mail; it is not the durable `create_task`
  protocol. A browser draft, canonical source annotation and delivered target input have distinct
  lifetimes and success boundaries.
- Context references identify fixed transcript spans or accepted pipeline-attempt reports; labels,
  grants and task attachments do not change source identity. References contain no copied content.
  Grant/list/attach alone neither injects content nor wakes an agent. Workspace/resource files,
  arbitrary URLs/blobs and mutable named values are not reference sources. Deleted sources become
  explicit tombstones, not newer aliases or implicit retained snapshots.
- Direct peer transcript reading is separately authorized across projects and archives. Revoking
  a reference grant restricts that path, not all transcript access. Room-private input must respect
  its public activity/result rules, but do not invent absolute confidentiality from other Chuck
  agents or the trusted local operator beyond the stated policy.
- Waking mail can request a turn; deferred mail cannot. Both may be included within bounds on an
  independently authorized turn. Message selection, wake attempt, successful delivery receipt,
  explicit mailbox read and comprehension are different facts. Uncertain delivery retains content;
  it does not create another activation merely to replay it.
- Awareness opt-out suppresses shared awareness prompt blocks, not skills/native discovery, tool
  registration, directories, permissions or task-specific instructions. Package unavailability
  suppresses its managed path/env/advertisement even if old cache remains. These are different
  controls; neither means the independently registered action surface disappears.

## Starting authority and lifetime map

Verify this map at the selected HEAD; visible context and recorded hashes are not the final oracle.

| Information | Authority / producer | Consumer and lifetime distinction |
|---|---|---|
| Submitted message and attachment bytes | Browser selection until submission; server owned attachment store, ordered descriptors/hash and receipt thereafter | Message parser/runtime reopen the accepted copy, not a changed original. Held/released recovery is explicit; uncertainty cannot silently enqueue another native effect. |
| Captured annotation | Source descriptor/anchor, captured excerpt/instructions and canonical source/room record | Self prompt, reserved-user mail or launched-recipient delivery must preserve source/target meaning. Source append and target delivery are distinct; do not infer end-to-end idempotency from one append test. |
| Mail body and delivery | Durable message identity/attribution; per-turn inline reservation/confirmation | Prompt preparation and mailbox read consume selected IDs under their own bounds. Overflow stays retrievable; a later callback cannot settle a different receiving turn. |
| Context reference and access | Immutable locator in state; direct grants or durable work membership | Service resolves explicit bounded reads, reauthorizes pages and reports unavailable sources. Hidden-list state is not revocation; cursor position is not a capability. |
| Assignment/watched changes | Task owner, confirmed membership/execution handle, attached refs and retained source results | `get_assigned_task` exposes this assignment's context and continuation changes. Runtime registration and durable membership are different lifetimes; terminal task state alone does not erase authorized reference access. |
| Managed operating knowledge | Embedded `internal/agentknowledge/operating-chuck/**` source; verified process-local installation result | Startup publishes provider views; overlay points to current managed files. Cache presence alone is not availability. Exact-digest seed migration cannot rewrite custom user prompts. |
| Persona/native/shared/workflow instructions | Frozen user base/awareness, provider-native guidance and separately owned task/pipeline/runtime layers | Preserve layer role and operation purpose through actual provider requests. Current managed files do not imply a native prompt snapshot was refreshed; native fork must not receive a fresh persona/primer by default. |
| Tools and guidance | Current registered MCP definitions/result helpers plus job-specific managed references | Tool names, schemas, registration, advertised authority and text/structured results must agree. Package prose cannot create a missing tool or grant approval; a registered name alone does not prove native access. |
| Delivery diagnostics | Execution manifest of requested layer hashes/sizes, package/tool digests and observed adapter identity | Useful for correlation only. `delivery=requested`, `adoption=unknown`, truncation and unknown versions remain explicit; the record cannot certify a skill was read or a fact understood. |

## Shipped paths to trace

| Path | Current source entry points and question |
|---|---|
| Selection/Send/hold/Steer → native input | `internal/server/{sessions,attachments,attachment_recovery}.go`, `internal/attachment/store.go`, `internal/state/attachment_receipts.go`, `internal/runtime/{message,images,chat}.go`. Compare source bytes/order/descriptor, retained receipt and consumer-decoded input, including explicit recovery. |
| Annotation → source record → selected target | `internal/server/sessions.go`, `internal/server/think_tank_annotations.go`, `internal/runtime/event.go`, UI annotation submission/launch callers | Follow source/anchor/captured excerpt through validation, formatting, ordinary prompt/mail or room command. Check file/transcript/room target distinctions and selected recipient after launch failure. |
| Share/attach → assignment/discovery → read pages | `internal/contextref/{service,page,render,peer}.go`, `internal/state/context_links.go`, `internal/messaging/{context_tools,peer_tools,task_tools}.go`, task state/read methods | Follow actual canonicalization and authorization to text pages/tool results, not only the grant row. Include task-backed pipeline report sources and the independent peer-read route. |
| Waking/deferred mail → bounded inline prompt → receipt | `internal/state/messages.go`, `internal/runtime/{chat,acpmap}.go`, `internal/server/messaging_loops.go` | Preserve original objective/author and peer attribution, whole-message bounds and confirmation identity. Activation eligibility remains A2's authority. |
| Embedded package → startup publication → discovery/overlay | `internal/agentknowledge/package.go`, `internal/cli/dashboard.go`, `internal/server/{server,launch,resume,switch}.go`, config seed/migration helpers | `prepareAgentKnowledge` supplies verified availability; `applyKnowledgeOverlay` rebuilds runtime-only fields. Compare embedded/provider-view contents and current shipped guidance, including unavailable startup and later retry. |
| Selected guidance/tools → actual provider request | `internal/messaging/messaging.go`, `internal/server/messaging_registration.go`, `internal/runtime/{chat,fork,manifest}.go`, provider instruction parameter helpers | Trace registered inventory and exact requested layers through fresh/resume/wake/switch/native fork. Check only the allowlisted no-work projection for isolated quota checks. |
| Native handoff/history → bounded primer/continuation | `internal/server/{primer,switch,resume}.go` and runtime fork/native-load paths | Establish what facts/omissions were supplied and how long the bridge survives. A1 owns durable history identity/recovery; live adoption after load/handoff/compaction is distinct from a legible Chuck transcript. |

Managed guidance has several owners: persona is role/purpose, tool definitions own exact mechanics,
the main skill owns broad Chuck choices, and references own job detail. Compare that content with
current shipped capability, result and approval semantics, including task reporting, waits, peer
inspection/Stop, deferred mail and room/pipeline work. Do not substitute old direct-executor tests
or the planned action CLI for current task/MCP paths. Do not rewrite product prose during review.

The current manifest's tool digest hashes auto-approved MCP names; it is not an independent inventory
of every capability actually available to a provider. Compare real registration/consumer evidence.

## Connected searches and useful scenarios

These are facets of one review, not separately queued checks or an exhaustive matrix.

| Scenario / search | Evidence that distinguishes faithful delivery from apparent availability |
|---|---|
| A file/image-bearing prompt or annotation starts delegated work with a selected source fact | Compare known source bytes/anchor/ID, accepted copy or captured excerpt, source record, actual recipient and provider-decoded payload/read result. Follow the real annotation-launch versus durable-task route; no assumption that both create a task record. |
| Original file/source changes after selection; delivery is held, retried, withdrawn or recovered | Uploaded/captured content remains the accepted version; ordinary mentions retain their different meaning. Compare payload hashes/ordered descriptors and exact recovery ownership. Lost-response/ambiguous-consumption defects already assigned to ASF-2 keep their owner. |
| A granted/attached reference spans several pages while the source advances or access changes | Reconstruct the fixed selected source under documented projection/omission rules; check per-page authorization, cursor binding, completion and typed tombstone. Later source text cannot enter the original span, and an unavailable record cannot become false completion. |
| Waiting task resumes with source results/refs after multiple watched changes or source deletion | Compare assignment/execution identity and retained result revisions through the actual assignment read and authorized context page. ASRE-1/ASF-3 already owns lost watched continuation context; inspect a new seam only with distinct evidence. |
| Waking/deferred mail meets overflow, user skill-command input or provider failure | Compare stable message IDs/body/attribution, whole-message bounds/remaining counts and actual prompt order. Preserve the original objective/leading skill syntax where required. Only a valid completion settles its inline receipt; deferred overflow or uncertainty cannot create an extra turn. |
| Package installation fails while an old managed tree remains, then a later startup succeeds | Observe bounded warning, process availability, actual env/dirs/prompt claims and independent tool access. Old cache cannot authorize advertisement; later recovery rebuilds one overlay without modifying frozen user fields or generating a provider turn. |
| Aware/unaware/default/custom persona resumes, switches or forks after role/package changes | Compare frozen awareness/base with current managed files and actual native request. No stale duplicate suffix, overwritten custom prompt or new fork persona. Opt-out cannot disable unrelated capabilities; package installation cannot prove provider read/adoption. |
| Guidance advertises a tool/result/approval behavior after capability or package change | Compare registered `tools/list`, process registration, relevant job reference, native-visible tool and text/structured refusal. Unknown/stale registration cannot look successful, and prose/provider auto-approval cannot satisfy a separate host gate. |
| Handoff/clone/compaction retains visible history but changes native context | Compare actual native load/fork/primer input and later bridge behavior, attachment-copy identity and omission disclosure. Do not claim historical images were resent or native adoption occurred from a transcript/manifest alone. |
| Room-private input or annotation crosses agent/room/public result surfaces | Trace source/target identity, authorized rendering/tool response and exact public activity/result publication. Apply documented private/public rules and broad peer-read policy separately, rather than assuming one grant model protects every surface. |

Use a few connected disposable-home journeys, such as selected evidence → annotation or task/mail
→ recipient's bounded read/native input, plus package availability across continuation. Use known
facts and consumer-decoded data as the oracle. A bounded native task that depends on those facts
can support adoption evidence when authorized and available; it is not a demand for a general
model-quality benchmark. Do not multiply every content type, provider and target without distinct risk.

## Reuse these proofs before adding any

These tests were researched, not run or certified. Inspect fixtures and actual assertion boundaries.

| Seam | Existing proof and its limit |
|---|---|
| Attachment native input and clone | `internal/runtime/attachments_test.go`: `TestAttachmentPromptMaximumNativeWireAndFileOrder`; `internal/runtime/fork_test.go`: `TestForkOwnsIndependentAttachmentCopiesBeforeHistoryPublication`. Wire/copy proof is not proof a real provider read every file or adopted an image. |
| Annotation identity/format | `internal/server/annotations_test.go`: `TestAnnotationMailBodyClipsOnlyExcerpts`, `TestAnnotationAgentDeliveryPersistsUserMailAndTranscriptEvent`, `TestAnnotationRoundTripMatchesLiveAndReplayedEventForMixedAnchors`. No lost-response command idempotency follows from those tests. |
| Fixed references and cursor truth | `internal/contextref/service_test.go`: `TestCurrentTurnSpanExcludesTheInFlightShareCall`, `TestReadPagesTraverseTheFixedSource`, `TestArchiveKeepsContextAndDeletionTombstones`, `TestReadRejectsForgedCursorOffsets`. Service-level reads do not prove downstream recipient use. |
| Peer access/source continuity | `internal/contextref/peer_test.go`: `TestPeerCursorRejectsSameBoundaryReplacement`; `internal/messaging/context_tools_test.go`: `TestContextToolsDoNotConsumeMailBudget`. Check grant-scoped reference and independent peer-read authorization separately. |
| Inline mail receipt | `internal/state/messages_test.go`: `TestDurableMailInlineReservationSemantics`; `internal/runtime/acpmap_test.go`: `TestValidInlineReceiptRequiresRecognizedNonCancelledResult`; `internal/runtime/chat_test.go`: `TestStartActivationInjectsInlineMailTurn`. Supplying a batch or getting streaming output alone is not confirmation. |
| Package publication/degradation | `internal/agentknowledge/package_test.go`: `TestInstallPublishesVerifiedOwnerOnlyProviderViews`, `TestEmbeddedSkillUsesBoundedProgressiveReferences`; `internal/cli/knowledge_test.go`: `TestPrepareAgentKnowledgeFailureThenRetry`. Installed bytes and links do not prove native discovery or content is current for every capability. |
| Awareness/lifecycle overlay | `internal/server/knowledge_overlay_test.go`: `TestUnawarePersonaKeepsCapabilitiesWithoutSharedPrompt`, `TestAwarenessIsFrozenAcrossLifecycleComposers`, `TestKnowledgeOverlayIsRuntimeOnlyAndConditional`. Composed fields are not effective native instructions. |
| Tool/result and native diagnostics | `internal/messaging/tool_result_contract_test.go`: `TestRegisteredToolsShareResultContract`; `internal/runtime/params_test.go`: `TestClaudeSessionParamsAppendToNativePreset`; `internal/runtime/codex_native_instructions_test.go`: `TestCodexNativeDeveloperInstructionsPreserved`; `internal/runtime/manifest_test.go`: `TestExecutionManifestIdentifiesRequestedLayers`. Mocked/pinned construction and requested hashes do not certify model knowledge or live adapter behavior. |

The main unresolved join is source/availability through the real current consumer, across a lifecycle
or bounded failure. Existing proofs narrow the search; helper names, package digests and passing
schema inventories cannot close it alone. Respect FS-18.A12/A13 and the existing live attachment,
task reporting, handoff/compaction and provider-policy gates; state precisely which receipts remain
unavailable. Record no sensitive source bodies, credential values or raw personal guidance in reports.

## Joins and finding ownership

Refresh the [fix ownership map](AGENT-SURFACE-FIX-UNITS.md) and live handoff. Reuse ASR-A/B/D/E
and [original joins J1/J3/J5](AGENT-SURFACE-REVIEW.md#mandatory-joins-between-reviews).

- ASF-2 owns ASRD-4's annotation lost-response duplication and ASRH-1/2/4 draft/withdraw/uncertain
  Steer ownership. ASF-3 owns ASRE-1's lost watched continuation context; ASF-4 owns ASRE-2 result
  bounds. Corroboration stays with those findings; a different current consumer/mechanism may be
  a new A4 finding, but do not create duplicate owners.
- ASW-2 is closed, including frozen selection, native-purpose preparation and requested manifests;
  preserve its native inheritance/fork/check and human legacy decisions. ASW-1 has open canonical
  commit/control/recovery findings. Neither an old failing overlay nor an owed live receipt is a
  new demonstrated delivery defect. Closed ASW-3 owns process/driver proof.
- A1 owns durable truth and repair; A2 admission and activation/retry; A3 actual workspace/config
  realization. A4 follows their inputs only far enough to establish faithful information delivery
  and usable guidance. Task/pipeline-specific instructions keep their distinct frozen lifetimes.
- A5 will own accepted result/next-work authority; A6 broad operator receipts and observation;
  A7 sustained overhead/empirical quality. Here a delivery receipt means supplied input, not an
  accepted result, and prompt length alone does not demonstrate degraded agent performance.

Do not add a universal context injector, new messaging/receipt framework, mutable reference sources,
broader privacy model, action-transport migration, prompt rewrite or benchmark project. A valuable
architectural remedy discovered by review is a separately designed work item, not implementation
during this audit.

## Required review result

Record exact reviewed HEAD, relevant dirty work and provider/adapter source versions. Produce one
concise ledger of source/content identities, authorization paths, layer lifetimes, current consumers
and connected scenarios. Separate requested payload, confirmed supplied input and native adoption;
keep hypotheses and unavailable evidence separate from confirmed findings.

Record real findings under **CR-A4** in the handoff using workflow §7: current source location,
credible trigger, development impact, requirement/invariant and reproducible evidence or precise
source-backed mechanism. Give the unit one Fix model recommendation based on its hardest open fix.
Do not classify intentional bounded/pull delivery, awareness opt-out or documented legacy behavior
as a defect merely because eager injection or another prompt policy might be preferable.

Use isolated temporary Chuck/native homes and loopback-only servers, preserve user data/dirty work
and follow the review skill's allowed reporting rules. Change no product code or normative specs.
Close only this unit, retaining findings and honest evidence limits. Do not run other audit units,
fixes or a release architectural-review receipt as part of A4.
