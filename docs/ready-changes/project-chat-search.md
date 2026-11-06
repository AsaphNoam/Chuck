# Project chat search and matching-passage navigation

**State:** Waiting to start
**Why:** Operator `/design-feature` request on 2026-11-05; confirmed SQLite direction and requested
technical specification on 2026-11-06.
**Relevant requirements:** FS-05.R46–R53/A28–A32; TS-02.R63–R64; TS-03.R82–R84;
TS-08.R156–R157; INV §1/§2/§7/§8/§9/§11/§14/§16/§17.

## Outcome

A compact search beside a project's heading finds its chats, prioritizes running then stopped
non-archived chats ahead of archived chats, and opens a result at the displayed matching passage.

## Included work

Reuse SQLite FTS5 with prefix terms, existing metadata weighting and an opt-in project archive
view. Add only source-location provenance in the same SQLite database, a read-only reference
resolver and shared live/archive reveal; no second searchable-text index or new search dependency.
Preserve ordinary Archive behavior, document boundaries, existing chat data and lifecycle.
Older documents resolve only through verified transient source reconstruction; unavailable or
over-budget locations remain explicit and retain normal chat opening. Offline reindex rebuilds
provenance through its existing atomic repair path. Unfinished-turn search, typo correction,
semantic retrieval, cross-turn terms and phone search are excluded.

Capability evidence checked 2026-11-06: SQLite's official [prefix-query documentation](https://sqlite.org/fts5.html#fts5_prefix_queries)
supports expansion without rebuilding the FTS table or adding optional prefix indexes; its
[highlight function](https://sqlite.org/fts5.html#the_highlight_function) marks original-text
matches for byte-range attribution. Live code uses `unicode61 remove_diacritics 2`, weighted BM25,
immutable turn documents and `transcript.ProjectEvent`; existing `snippet()` alone exposes no
absolute source location. `appendRenderedEvent` merges assistant deltas and the shared
`TranscriptView` already reveals sequence-based targets. The extension preserves constituent
source positions rather than using a turn's final sequence. Bleve was evaluated and declined:
typo tolerance did not justify another local text index for the approved scope.

## How we will know it works

FS-05.A28–A32 and TS-03.R84 define focused tagged/untagged SQLite, migration, reindex and real-HTTP
checks. Use independently known source positions for Unicode, merged deltas, repeated passages,
annotation/close boundaries and old-data failure cases. The rendered live/stopped/archive journey
must establish exact reveal, collapsed-turn expansion, keyboard use, reload/back, and stable reading
while live content/layout changes; inspect at the desktop floor and a wider viewport across the
supported appearances. These acceptance checks remain owed until implementation.

## Waiting on

No product decision. Specifications are planned; implementation, runtime benchmarks and rendered
acceptance have not started. This change does not alter the current active release work.
