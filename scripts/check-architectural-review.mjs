// TS-06.R36: validate declared review/fix provenance, not architectural reasoning.
import { execFileSync } from 'node:child_process';

function git(...args) {
  return execFileSync('git', args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim();
}
function requireThat(ok, message) {
  if (!ok) throw new Error(message);
}
const nonblank = value => typeof value === 'string' && value.trim().length > 0;
const fullID = value => typeof value === 'string' && /^(?:[a-f0-9]{40}|[a-f0-9]{64})$/.test(value);
const versionTag = /^v\d+\.\d+\.\d+$/;
const receiptRoot = 'docs/archive/reviews/architecture/';

function committed(candidate, path) {
  try { return git('show', `${candidate}:${path}`); }
  catch { throw new Error(`Missing committed ${path}; commit the review/closure state before checking.`); }
}
function ancestor(before, after) {
  try { git('merge-base', '--is-ancestor', before, after); return true; }
  catch { return false; }
}
function administrative(path) {
  return path === 'docs/features/HANDOFF.md' || path.startsWith(receiptRoot) ||
    /^docs\/archive\/state\/(?:HANDOFF-through-\d{4}-\d{2}-\d{2}|RELEASE-v\d+\.\d+\.\d+-notes)\.md$/.test(path);
}

function check(candidateRef, releaseTag) {
  const candidate = git('rev-parse', '--verify', `${candidateRef}^{commit}`);
  const tags = git('tag', '--merged', candidate, '--sort=-version:refname').split('\n');
  const previous = tags.find(tag => versionTag.test(tag) && tag !== releaseTag) ?? null;
  const path = `${receiptRoot}${previous ?? 'initial'}-next.json`;
  let receipt;
  try { receipt = JSON.parse(committed(candidate, path)); }
  catch (error) { throw new Error(`Architecture receipt: ${error.message}`); }
  requireThat(receipt?.schema === 1, 'Unsupported architectural-review receipt schema.');
  requireThat(receipt.previous_release === previous, 'Receipt has the wrong previous-release boundary.');
  requireThat(receipt.review_complete === true, 'Architectural review is incomplete.');
  requireThat(nonblank(receipt.reviewer), 'Receipt must identify the independent reviewer.');
  requireThat(Array.isArray(receipt.coverage) && receipt.coverage.length > 0 && receipt.coverage.every(nonblank),
    'Receipt needs capability/scenario coverage, evidence and limits.');
  const reviewed = receipt.reviewed_commit;
  requireThat(fullID(reviewed) && ancestor(reviewed, candidate), 'Reviewed commit is not a full ancestor of the candidate.');
  requireThat(previous === null || ancestor(previous, reviewed), 'Reviewed commit predates or diverges from the previous release.');
  requireThat(Array.isArray(receipt.findings) && Array.isArray(receipt.updates), 'Receipt needs findings and updates arrays.');

  const findings = new Map();
  for (const finding of receipt.findings) {
    requireThat(nonblank(finding?.id) && !findings.has(finding.id) &&
      ['must_fix', 'worth_fixing'].includes(finding.severity), 'Invalid or duplicate architectural finding.');
    findings.set(finding.id, finding.severity);
  }
  const later = git('rev-list', '--reverse', `${reviewed}..${candidate}`).split('\n').filter(Boolean);
  const laterSet = new Set(later);
  const covered = new Set();
  const closed = new Set();
  for (const update of receipt.updates) {
    requireThat(['fix', 'coverage'].includes(update?.kind) && update.verified === true && nonblank(update.evidence),
      'Each update needs a kind, verified evidence and exact commits.');
    requireThat(Array.isArray(update.commits) && update.commits.length > 0 &&
      update.commits.every(id => fullID(id) && laterSet.has(id)), 'Update commit is outside the post-review candidate range.');
    const ids = update.finding_ids ?? [];
    const closes = update.closes ?? [];
    requireThat(Array.isArray(ids) && Array.isArray(closes), 'Invalid finding references in update.');
    if (update.kind === 'fix') {
      requireThat(ids.length > 0 && ids.every(id => findings.has(id)) && closes.every(id => ids.includes(id)),
        'A fix must reference original findings and close only its verified scope.');
      closes.forEach(id => closed.add(id));
    } else {
      requireThat(ids.length === 0 && closes.length === 0, 'Coverage updates cannot close findings.');
    }
    update.commits.forEach(id => covered.add(id));
  }
  const open = [...findings].filter(([id, severity]) => severity === 'must_fix' && !closed.has(id));
  requireThat(open.length === 0, `Unresolved architectural Must-fix findings: ${open.map(([id]) => id).join(', ')}`);

  for (const id of later) {
    const paths = git('diff-tree', '--root', '--no-commit-id', '--name-only', '-r', '-m', '-z', id)
      .split('\0').filter(Boolean);
    requireThat(paths.every(administrative) || covered.has(id),
      `Uncovered substantive commit ${id}. Record finding-fix or focused affected-coverage evidence; fixes do not require another architectural review.`);
  }
  const handoff = committed(candidate, 'docs/features/HANDOFF.md').split(/\r?\n/);
  const start = handoff.findIndex(line => line.trim() === '## Review findings');
  requireThat(start >= 0, 'Committed handoff has no Review findings section.');
  const end = handoff.findIndex((line, index) => index > start && /^##\s/.test(line));
  requireThat(!handoff.slice(start + 1, end < 0 ? undefined : end)
    .some(line => /^\s*-\s+\*\*Must fix\*\*/i.test(line)), 'Live handoff still has Must-fix findings.');
  console.log(`Architectural release gate: ready (${path}; original review plus verified closure, no rerun required).`);
}

try {
  let candidate = 'HEAD';
  let releaseTag = null;
  const args = process.argv.slice(2);
  while (args.length) {
    const option = args.shift();
    const value = args.shift();
    requireThat(value && !value.startsWith('-'), 'Usage: node scripts/check-architectural-review.mjs [--candidate ref] [--release-tag vX.Y.Z]');
    if (option === '--candidate') candidate = value;
    else if (option === '--release-tag' && versionTag.test(value)) releaseTag = value;
    else throw new Error(`Invalid option ${option}.`);
  }
  check(candidate, releaseTag);
} catch (error) {
  console.error(`Architectural release gate: ${error.message}`);
  process.exitCode = 1;
}
