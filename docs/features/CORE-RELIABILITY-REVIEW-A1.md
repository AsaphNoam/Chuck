# CR-A1 — Durable truth and recovery

Prepared 2026-10-20 from two focused GPT-6 Luna research passes and parent integration against
`97c67f39`. This maps **A1 only** from the
[core reliability priorities](CORE-RELIABILITY-AUDIT-PLAN.md). It is one review for a frontier
reviewer, not a set of separately queued component checks. Research inspected documentation,
source entry points and existing tests; it performed no audit, failure probes or product repairs.
[HANDOFF.md](HANDOFF.md#review-queue) owns live status.

## Question and responsibility

Can Chuck preserve and recover accepted development work, find its recorded history, and continue
the same native conversation after Stop, server restart, archive/restore or offline repair?

Follow the complete storage and recovery chain, including callers outside the listed files.
Prioritize silent loss, a successful operation that leaves work unusable, and damage to healthy
conversations while repairing a different one. Correct history rendering alone does not establish
that the provider resumed the intended conversation. A clear refusal for missing native identity
can satisfy the contract; silently opening a fresh conversation cannot.

This review owns the authority split and maintenance consumers of conversation/session data:
normalized history, durable identity and continuation inputs, native history references, archive
visibility, derived search/tracking, relevant migrations and restart reconciliation. Inspect both
chat providers, Codex and Claude. Terminal coverage follows Chuck's currently supported Claude
path; do not invent a Codex terminal contract. Include a previously switched/cloned session only
where its stored shape changes recovery; re-auditing the whole switch/fork implementation is not
required.

## Read first

Read the current handoff's Current position and Active change, then this charter. Use workflow
§§3–7 for review/reporting, with the explicit subsystem scope here replacing a commit-diff scope.
Read the following requirements before following source:

- FS-01.R6/R10/R17–R18/R31/R34: Stop, frozen continuation, chat/terminal crash differences,
  archive distinction and exclusive lifecycle transitions. FS-07.R16–R17/R25: terminal continuity
  and truthful refusal when native identity is unavailable.
- FS-05.R6/R11–R19/R25–R27/R30–R36/R38/R41–R42: resume/readback, search capability and document
  boundaries, archive/restore preservation, tracking and unreadable history. Read supersession
  notes: earlier flat-list and unrestricted transcript-search promises are retired.
- TS-02.R1–R3/R6–R10/R16: writers, authority, migrations, ordered transcript and rebuild/fallback.
  R18/R30/R35/R39/R48/R53–R54 describe stored effort/fast, capabilities/lineage, context, history
  primer, execution contract and terminal identity where their current consumers need them.
  R52 and TS-01.R39 own canonical commit outcomes, journal health and interrupted-writer recovery;
  TS-01's corresponding persistence/lifecycle contracts apply at the other joins.
- Read the invariant trigger index, then matching classes in full. The likely classes are
  §§1/2/7/9/11/15/17; include §5 for lifecycle claims and others only when their surface is involved.
  TS-06.R35 governs proportional evidence. Existing adequate proof can avoid another test.

Product specs govern behavior. This charter, old reviews and source comments are navigation and
hypotheses. Planned backup/cutover or input-receipt promises are not assumed shipped.

## Starting authority map

Verify this map against the selected HEAD; it is not an oracle for a field's correctness.

| Stored fact | Authority and producer | Recovery consumer / question |
|---|---|---|
| Stable agent identity and archive flag | SQLite `agents`; lifecycle/state writers | Archive, Restore and Resume must retain the same `agent_id` and current runtime identity. |
| Frozen continuation inputs | SQLite `sessions`; launch/switch metadata and later owning writers | `ReadSession` and Resume consume cwd, persona, permissions/directories, launch/execution contract and history bridge. Identify which values replay can reconstruct and which require retention. |
| Native conversation ID and live effort/fast | SQLite session fields; authenticated SessionStart/teardown and live settings writers | Current reindex explicitly retains these SQLite-only inputs. Native Resume must consume the retained value, not an obsolete metadata event. |
| Current process and observed status | Ephemeral `running`, plus status projections; runtime/hooks/reconciliation | Removing a dead process must preserve durable continuation inputs. A preview/status repair does not reconstruct a missing session. |
| Normalized conversation | `$CHUCK_HOME/sessions/{agent_id}/transcript.ndjson`; canonical journal/writer | Transcript readers, replay/index and archive readback must preserve accepted content, order and meaningful partial tails. |
| Journal health and interrupted-writer evidence | SQLite `conversation_health`, separate from `sessions`; shared Journal writes bounded health/reason, reserved sequence and open-writer marker | Reopen/startup must validate the tail and recover an explicit gap where required. `not_written`, `uncertain` and `committed` are distinct; `index_degraded` does not undo a committed append. Maintenance must not erase or bypass this recovery authority. |
| Provider conversation | Native provider session/history addressed by the stored ID | Correct Chuck history does not prove that provider-owned history exists or was loaded. Trace the actual load/resume request and its refusal semantics. |
| Search, tracking and rollups | SQLite projections populated by index replay and hook/event capture | Verify each projection's real reconstruction source, including terminal hook activity. Derived search cannot become authority for conversation content or Resume inputs. |
| Project visibility and editable configuration | Project/config JSON; config writer | Archive/Restore crosses JSON and SQLite. Ordinary Resume uses frozen inputs while current project archive state still gates execution. |

Start with the META-1 retention inventory and regression. Extend it only for new writers, changed
semantics or unaccounted consumers; do not redo a large column checklist for its own sake. For a
suspect value, name its writer, durable original, replay/retention rule, overwrite precedence and
first consumer that needs it after interruption. Treat `sessions` as a mixed-purpose table, not
uniformly disposable search data.

## Shipped paths to trace

| Path | Entry points and connected implementation | Contract to follow |
|---|---|---|
| Accepted conversation → disk → restart | `internal/runtime/journal.go`, `internal/transcript/{writer,reader}.go`, `internal/index/indexer.go`, `internal/state/history_health.go`, `internal/server/sessions.go` | Distinguish accepted/synced events from unsent or held input; follow durable health/open-writer recovery, readback and later append sequence. |
| Stop/restart → native continuation | `internal/server/{resume,hook,switch}.go`, `internal/state/{session,running,manager}.go`, runtime native-load path | Separate frozen inputs, current identity, retained native ID and ephemeral process proof. Follow actual consumer payloads beyond `LaunchSpec`. |
| Archive → readback → Restore → explicit Resume | `internal/server/{routes,archive,archive_actions}.go`, `internal/archive/archive.go`, state agent flags and config project writes | Agent/project archival preserves work. Project Restore leaves agents archived; agent Restore returns stopped state. Restore is not transcript recovery or automatic execution. |
| Offline archive/search repair | `internal/cli/reindex.go` → `internal/index/reindex.go` → transcript replay/index/state | Check exclusive-writer admission, retention before reset, per-agent replay/restore, errors, rerun and post-repair continuation. |
| Startup/background reconciliation | `internal/server/reconcile.go` and state correction methods | This scans canonical logs for stale status/preview and prunes stale running rows; establish its limits and avoid overwriting fresh hook/current-generation truth. It is not full reindex. |
| Older stored shape → current consumer | Relevant migrations in `internal/state/schema.go`, session decoding and `session_meta` replay | Defaults must preserve documented meaning or refuse explicitly. Check only shapes actually read by these recovery paths. |

Archive/search reads are `GET /api/archive`, `GET /api/archive/projects/{project}` and
`GET /api/sessions/{id}/transcript`. Agent Archive/Restore/Resume and project Archive/Restore have
separate POST routes in `routes.go`. No general transcript-restoration endpoint is implied.
The CLI refuses reindex while it believes the server is running; source comments alone do not
prove exclusion through process startup, stale/unreadable pidfiles or another maintenance writer.
Investigate only credible overlapping-writer scenarios, not arbitrary unsupported concurrent DB edits.

## Connected searches and useful scenarios

These are coverage facets of **one** review. Integrate their results; do not create a separate
task, test suite or exhaustive matrix for every row.

| Scenario / search | Evidence that discriminates correctness from apparent success |
|---|---|
| Ordinary work → Stop or mid-turn crash → server restart → Resume | Compare known accepted user/assistant/tool events and last durable sequence, frozen inputs plus live settings, actual native load ID, and first subsequent append. Explain partial-tail recovery and avoid claiming durability for never-submitted held input. |
| Agent and project archive → search/read-only history → project Restore → agent Restore → Resume | Compare durable identity/archive flags, project JSON and preserved session/history before/after. Trace interrupted or failed cross-store archive compensation: a success-looking response or dashboard disappearance must not hide lost work or uncontrolled execution. Reuse existing claim-race proofs. |
| Reindex after legitimate live settings/native-ID updates | Follow old and later `session_meta` events versus SQLite-only values into replay, restored snapshot, Archive and actual Resume request. Include an existing switched/bridged stored shape when it changes overwrite precedence; do not reopen closed selection fixes without a new counterexample. |
| Interruption after destructive reset, before replay/retention finishes | Choose one realistic process interruption at the actual commit boundary in an isolated home. Determine which truth survives outside the repair process, whether rerunning repairs it, and whether healthy agents retain usable continuation. Follow journal health/reserved identities through repair and reopen; neither an in-memory saved value nor a cleared status proves durable recovery. |
| One bad/missing transcript among healthy conversations | Distinguish malformed record, truncated tail, unreadable/absent file and missing root only where they take different destructive paths. Check that skipped input does not silently erase unrelated authority; inspect returned errors and remaining usable state. A fixture containing only an empty database cannot establish preservation of existing work. |
| Live indexing versus offline replay versus native/terminal tracking | Compare contractually searchable text, turn/annotation flush boundaries, partial final turns and deduplicated tracking. Exercise FTS5 and metadata fallback only for their different promises; metadata-only search is expected behavior, not lost transcript content. |
| Reconciliation after Stop/switch/restart or delayed native hook | Follow freshness/generation fences and retained IDs through stale-row cleanup. Verify disk/read failures cannot fabricate success or override newer truth. Keep known ASW control/commit defects with their current owners. |
| Migration/defaults of a relevant old session | Use a representative shipped older row/log and actual migration/decoder; compare the recovered session and Resume/refusal. Unknown legacy native identity must not silently produce a new conversation; do not demand restoration of provider data Chuck never stored. |

Start with normal flows, then choose failure boundaries from their actual implementation. Prefer a
single coherent disposable-home journey that combines maintenance, archive readback and native
continuation, plus focused counterexamples where the mechanisms differ. Do not require every
combination across provider, interface, archive state, log damage and migration version.

## Reuse these proofs before adding any

These tests were located during planning; **they were not run or certified by this research**.
Inspect setup and assertions, not just test names. In particular, determine whether the fixture
contains pre-existing work, whether it kills a real repair process or merely returns an error,
and whether it observes the native Resume consumer.

| Evidence seam | Existing starting tests |
|---|---|
| Journal acceptance/health | `internal/runtime/journal_test.go`: `TestJournalFailedAppendIsNotPublished`, `TestJournalIndexFailureStillCommits`, `TestJournalDegradesThenRecoversWithGapNotice`, `TestJournalOpenMarkerRecovery`; reconcile their limits with open ASW1-5 rather than creating a duplicate owner |
| Retention/rebuild | `internal/index/indexer_test.go`: `TestReindexRetainsSQLiteOnlySessionFields`, `TestReindexRebuildsFromRawLogs`, `TestReindexIsolatesBadAgent`, `TestReindexMissingSessionsDirIsNoop`, `TestHistoryPrimerRoundTripAndReindex` |
| Search/partial turns | Same file: `TestResumeAfterRestartPreservesFTSContent`, `TestReindexPreservesFinalPartialTurn`, `TestReindexPreservesAnnotationFlushBoundary`, `TestReindexRollsUpNativeActivityOnce` |
| Torn records/sequence | `internal/transcript/writer_test.go`: `TestReopenContinuesSeq`, `TestReaderSkipsPartialTrailingAndBadMiddleLine`, `TestOpenTruncatesTornTrailingLine`; `reader_test.go`: `TestReaderRecoversMaxSeqPastOversized` |
| Crash and frozen continuation | `internal/server/integration_test.go`: `TestCrashMidTurnPersistsDeliveredTranscript`, `TestResumeHappyPath`, `TestResumeKeepsFrozenEffortAfterCatalogEdit` |
| Terminal identity | `internal/state/manager_test.go`: `TestSessionStartPersistsNativeIDAndTeardownRetainsIt`; `internal/server/switch_test.go`: `TestResumeTerminalAgent`, `TestTerminalSessionStartSurvivesStopForResume` |
| Archive claims/partial failure | `internal/server/archive_actions_test.go`: `TestArchiveProjectConfigFailureRestoresMixedAgentFlags`, `TestArchiveProjectReportsCompensationFailureAndPublishesDurableFallback`; `archive_gate_test.go`: `TestAgentArchiveClaimBlocksConcurrentResume` |
| Freshness/legacy shape | `internal/server/reconcile_test.go`: `TestReconcileSessionsOnceAppliesStaleCorrection`, `TestReconcileSessionsOnceDoesNotOverrideFreshHook`, `TestReconcileIgnoresNonTranscriptSessionArtifacts`; `internal/state/state_test.go`: `TestMigrationFromV47PreservesLegacyAwarenessAndGuardsBoolean` |

These isolated proofs do not by themselves establish process-interrupted repair, preservation when
an entire durable input is absent, or archive/repair followed by successful native continuation.
That combined boundary is the main unresolved question, not a claim that existing tests are useless.
Use known expected content and committed pre-operation state as the oracle. A fake provider can
prove the exact Resume request and refusal handling; only a pinned/native receipt can prove actual
history adoption. Reuse live evidence already owed and report precise gaps when unavailable.

## Joins and finding ownership

Read the relevant evidence in the [original agent-surface charters](AGENT-SURFACE-REVIEW.md),
especially ASR-D and joins J2/J5/J6, and the current
[fix ownership map](AGENT-SURFACE-FIX-UNITS.md). Refresh their live status before review.

- META-1 is closed at `0cafe370`; retention now covers hook-learned native ID and live effort/fast,
  including a failed replay after metadata was read. Challenge remaining current boundaries, not
  the old failing overlay.
- ASW-1 has open event-commit/control/recovery findings; ASW-2 and ASW-3 are closed at the planning
  baseline. Add corroboration to an existing finding when it is the same mechanism. A distinct
  repair consumer or destructive boundary can be a new A1 finding even in the same source file.
  Existing unproved terminal-driver/provider receipts remain unproved.
- A2 will own work admission, waits/wakes, capacity and task/activation liveness. A1 follows these
  callers only to establish conversation storage/continuation; it does not re-audit schedulers.
- A3 will own usable checkout/config realization. A1 owns persistence of the selected snapshot;
  inspect workspace/archive joins enough to establish preservation and correct refusal.
- A4 will own exact information delivery and provider context/knowledge adoption; A1 owns retained
  content/reference and native conversation identity. A5 owns result-driven supervision; A6 owns
  broader action receipts/CLI truth; A7 owns sustained cost. Investigate adjacent code whenever it
  breaks A1's promise, and assign each finding once rather than stopping at an ownership boundary.

Do not add backup, general data migration, new repair APIs, unrelated table audits, UI polish or
an updater rewrite. A high-value redesign discovered by the review is a separately proposed work
item, not implementation performed during the audit.

## Required review result

Record the exact reviewed HEAD, provider/adapter source versions and relevant dirty work. Produce
one concise evidence ledger covering the authority/consumer map and scenarios above, including
what each existing proof actually establishes and where it stops. Record applicable invariant
classes, confirmed findings and explicit evidence gaps separately.

For each real finding, use workflow §7 in the handoff under **CR-A1**, with current source location,
credible trigger, development impact, requirement/invariant and reproducible evidence or a precise
source-backed mechanism. Give the unit one Fix model recommendation based on its hardest open
fix. No severity is assigned to an untested hypothesis merely because data loss sounds serious.

Use isolated temporary Chuck/native homes, preserve user data and dirty work, and keep any server
loopback-only. Add only purposeful regression/probe evidence when needed; change no product code
or normative specifications. Follow the review skill's allowed file/reporting rules when invoked.
Close only this unit when its review is complete, retaining findings and honest live-evidence
limits. Do not start A2–A7, run fixes, or issue a release architectural-review receipt as part of A1.
