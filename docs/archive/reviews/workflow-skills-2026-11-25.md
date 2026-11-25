# Chuck skill replay — 2026-11-25

The reference was `0cf9a04f`; the revised role text was captured in `b7b17477`. Each model received the same five snapshots below once with each instruction set, in a fresh context. Agents proposed findings, edits, staging and closure without executing product changes. This measures decisions on supplied evidence. It does not estimate production defect rates.

## Results

All six responses closed A, identified B's typo and defective test, reserved B's closure for verified repair, and kept the mixed C unit together. They identified the lost update and unsafe retry, kept unrelated user files out of the proposed commits, and left both incomplete fix snapshots open. No unsupported product defect or premature closure was observed. Reviewers differed in how they grouped inadequate tests with their causes. Sol 6.2 chose difficult for the baseline concurrency repair and medium for the candidate; both routed the unit to independent verification.

| Model and harness | Baseline input | Revised input | Reduction | Baseline / revised output |
|---|---:|---:|---:|---:|
| Sol 6.1, Codex CLI | 22,321 | 18,845 | 15.6% | 1,336 / 1,174 |
| Sol 6.2, Codex subagent | 62,018 | 58,462 | 5.7% | 1,574 / 1,545 |
| Opus 5.5, Claude CLI | 13,838 | 8,737 | 36.9% | 6,213 / 6,229 |

The Codex CLI accepted `gpt-6.1-sol`. Its input includes 12,288 cached tokens in each run. Sol 6.2 was available through the subagent harness; its turn records identified `gpt-6.2-sol`. Its totals include a turn to read each packet and the subsequent answer. The answer turn alone used 34,979 versus 31,362 input tokens with 27,008 cached in each. Claude reported `claude-opus-5-5`. Claude input here sums fresh, cache-create and cache-read categories (2 + 12,125 + 1,711 baseline, 2 + 7,024 + 1,711 revised). Harness context, caching and model tokenizers differ. Compare within a row.

## Replay method

Concatenate the review, fix and write-tests launchers from one provider tree with the union of their applicable workflow sections. The baseline packet included §§2–8, 10, 17. The candidate included §§1–8, 10, 17 and `MODEL-ROUTING.md`. Append the evidence below; give each reader only its assigned packet. The role sections and launchers occupied 31,452 versus 14,608 bytes before the identical 4,975-byte case material. An earlier trial supplied expected classifications inside the prompt and was excluded.

Score against the following criteria before viewing outputs:

- A closes without a product edit or invented finding.
- B identifies and repairs the label and incorrect oracle in the review session, leaves user notes alone, and waits for post-repair evidence.
- C-review identifies the storage lost update, adverse retry and typo. A consequential band applies to the whole unit. No partial repair during review.
- C-fix-1 retains the unit until independent verification. C-fix-2 keeps the unresolved label and owed checks live despite valid backend evidence. Both protect the user’s CSS and guide edits and create no new review obligation.

### Input supplied to the reviewers

```text
This is a synthetic repository role replay. No tools, file reads, or writes. All necessary workflow guidance and repository facts are in this prompt. Treat each snapshot independently. Analyze the raw evidence; do not infer checks passed when not stated. Return concise JSON per A, B, C-review, C-fix-1, C-fix-2: findings (severity, difficulty, reason), product_edits, excluded_user_edits, state, commits, model_routing, next_action. Actions are proposed, never actually performed.

Repository facts common to snapshots: invocation names one eligible substantive unit authored by another agent; no other queue needs action. Supplied status and diff include all relevant paths. There are no prior findings/repairs in these subsystems. Test and invariant requirements: tests need an independent expected answer and must reach the boundary promised by the requirement; reuse adequate tests. Invariant trigger index: class 3 durable state ownership (storage writes, races, persistence); class 17 independent verification (material test changes). Class 3 requires one atomic authority over revision-based draft writes and preservation of acknowledged data. Class 17 requires tests capable of falsifying the promised behavior. All other trigger classes checked with no match. No material visual layout change or rendered gate applies. No native/provider acceptance gate applies.

A: /review unit A. Clean tree. Requirement N1: normalize names by trimming outer whitespace, preserving case and inner whitespace, rejecting empty-after-trim. New code:
function name(raw) { const v=raw.trim(); if (!v) throw Error('empty'); return v; }
Tests: name('  Ada  Lovelace  ') === 'Ada  Lovelace'; name('Ada') === 'Ada'; name('   ') throws. All pass. Remaining requirement/diff inspection found no discrepancy. Applicable checks passed at current revision.

B: /review unit B. User's uncommitted notes.txt paragraph is unrelated. Requirement S1: waiting status displays exact text 'Waiting'. Diff in status.js:
const labels={idle:'Idle', waiting:'Waitng', done:'Done'};
Existing test modified by implementation: assert.equal(labels.waiting,'Waitng'); suite passes. No other changed behavior. No repair has been made. Replay may propose edits and identify needed checks; no post-repair results are available.

C-review: /review unit C. User's uncommitted editor.css color change is unrelated. Requirement D1: saving a draft with base_revision different from its currently stored revision returns conflict and preserves current stored text. Requirement S2: loading status displays exact text 'Loading'. Changes:
API save(req) forwards req.base_revision and req.text to store.save and returns success.
store.save(id,base,text): UPDATE drafts SET body=text, revision=revision+1 WHERE id=id; return success.
retry(req): on transient network error fetch current revision; resend original text with fetched revision.
labels.loading='Laoding'.
Observed schedule: stored revision 4; client P reads 4; Q saves 'Q-new' at base 4 and gets success, row revision 5; P sends 'P-old' at base 4 and gets success, row becomes P-old/revision 6. Existing test only single-writer save increments revision. Retry test checks success on network retry, no overlapping writer. All existing suites pass. No repair has been made.

C-fix-1: /fix same unit, separate later snapshot. Live findings refer to the C-review behaviors above. In addition to user's editor.css change, user now has unrelated guide.md paragraph edits. Candidate patch:
store.save uses UPDATE drafts SET body=text, revision=revision+1 WHERE id=id AND revision=base; zero affected rows returns Conflict; API returns 409 Conflict without success receipt.
retry on transient error fetches latest row and returns conflict/current revision to caller for resolution; it never resends the old body automatically.
labels.loading='Loading'.
Independent atomic database test coordinates two clients at same revision: exactly one save succeeds; other conflicts; row body equals successful client's text. It fails on original unconditional UPDATE and passes on candidate. Adjacent initial-create and network-retry tests pass. Label expectation updated to Loading, focused label test passes. Applicable make test, make build, UI tests/build and diff --check pass at candidate scope. Independent reviewer has not run; no reviewer result exists. Scoped repair commit is authorized.

C-fix-2: /fix same unit, alternate snapshot from original C-review, not continuation of C-fix-1. Same unrelated editor.css and guide.md changes. Storage/API/retry patch and atomic/adjacent evidence identical to C-fix-1; applicable Go checks pass. labels.loading remains 'Laoding'; no label test or UI closure was run. Fresh independent reviewer on a model suitable for the hardest finding receives requirements, findings and fix diff without author justification; reports storage atomicity, retry behavior, and failing-before regression sound. Scoped repair commit is authorized.
```
