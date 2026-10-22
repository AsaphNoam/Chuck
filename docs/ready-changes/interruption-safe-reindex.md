# Preserve conversations during offline reindex

**State:** Waiting to start
**Why:** Explicit `/design-feature` on 2026-10-21 for CR-A1-1/2; the operator approved partial
repair on 2026-10-22. Origin and reproduced failure evidence:
[core-reliability assessment](../features/CORE-RELIABILITY-POST-REVIEW-ASSESSMENT.md#1-reindex-must-be-one-preservation-boundary)
and [CR-A1 findings in the handoff](../features/HANDOFF.md).
**Relevant requirements:** FS-05.R43–R45/A26–A27; TS-02.R56–R58, superseding R9's maintenance
mechanism; existing TS-02.R8/R10/R16/R54. INV §2/§7/§9/§15–§17.

## Outcome

`chuck reindex` repairs healthy conversations while preserving failed conversations' complete prior
archive/search/tracking and Resume data. Missing input never prunes a stored conversation. An
interrupted repair leaves each conversation entirely old or entirely rebuilt, retaining the latest
native identity and live effort/fast settings. Partial repair returns nonzero and identifies failures.

## Included work

Replace global wipe/replay with one transaction per conversation. Route every replay read/write
through that transaction, sharing the live projection helpers; restore SQLite-only authority before
commit. Inventory stored sessions as well as source directories, validate actual readable source and
replacement metadata, isolate per-conversation buffers, roll back bad sources and stop on database
errors. Preserve existing FTS modes, document boundaries, final partial turns, tracking and frozen
configuration normalization. Keep the CLI syntax and offline server guard.

The boundary spans `internal/index/reindex.go` and the transaction seams in `indexer.go`, with
diagnostics in `internal/cli/reindex.go`. `Indexer` currently mixes `db.Begin`, `db.Exec` and
`db.QueryRow`; wrapping only reset does not work. In particular, cwd lookup must join the transaction
and expose unexpected read errors. `state.Open` permits one connection, so escaping to `db` can
also deadlock. Use `state.DetectSessionsFTS` through the same transaction for deletions and writes.

One transaction per conversation implements the accepted partial-repair policy and retains completed
repair progress after interruption. A whole-archive transaction with savepoints adds savepoint
handling and retains all work until the end; durable staging adds recovery/cleanup state. Neither
provides a required benefit here. This is an internal transaction change, with no schema migration,
new authoritative field, retained-field table split or maintenance framework. GPT-6 Luna supplied
source facts; an independent clean-context GPT-6 Astra challenge recommended this boundary, with
the primary agent owning the decision. The transaction APIs are supported directly by
[Go's transaction contract](https://go.dev/doc/database/execute-transactions); SQLite supports
[atomic commit](https://sqlite.org/atomiccommit.html), including WAL under its existing configuration.
This design changes no power-loss durability policy.

CR-A1-3's general reader/writer newline correction remains a separate fix. This work uses the
existing complete-prefix/parser seams so replay agrees on committed records without absorbing all
ordinary-reader changes. No provider turn, online repair, source rewrite/deletion, archive-flag
change, scheduler or new UI/API is included. Implementing this unit covers CR-A1-1/2 together;
do not assign a competing independent fix of the same destructive replay path.

## How we will know it works

FS-05.A26–A27 and TS-02.R58 define the evidence: populated-store process interruption before
publication, reopen/rerun and real Resume consumer with fake native wire; mixed healthy/bad/missing
sources preserving complete failed projections; missing whole root versus empty home; source
restoration/retry; and focused post-replay storage failure rollback. Assert independent old/new
data and exact native-load identity, not merely a successful function return or empty-store no-op.

Reuse `TestReindexRetainsSQLiteOnlySessionFields`, `TestReindexIsolatesBadAgent`, partial-turn and
annotation checks in `internal/index/indexer_test.go`, and `TestArchiveReindexEquivalence` in
`internal/archive/archive_fts_test.go`. Revise `TestReindexRebuildsFromRawLogs`' stale-row pruning
expectation to match the approved preservation policy. Exercise supported FTS5/fallback behavior
and reuse existing unavailable-module evidence. Follow TS-06's applicable closure matrix and
the write-tests skill; no exhaustive kill-phase/provider matrix or live-provider adoption claim.

## Waiting on

None. Specifications are planned; implementation has not started. Normal `/review-design` can
challenge this waiting change; `/work` can select it as one implementation unit. Keep CR-A1-1/2
open until implementation and review establish closure.
