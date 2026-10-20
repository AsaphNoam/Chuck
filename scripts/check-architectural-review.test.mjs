// Real Git boundaries prove the release gate, especially review -> fix -> release (no rerun).
import assert from 'node:assert/strict';
import { execFileSync, spawnSync } from 'node:child_process';
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

const checker = fileURLToPath(new URL('./check-architectural-review.mjs', import.meta.url));
const receiptPath = 'docs/archive/reviews/architecture/v1.0.0-next.json';
const clearHandoff = '# Handoff\n\n## Review findings\n\n## Changelog\n';

function fixture(t) {
  const root = mkdtempSync(join(tmpdir(), 'chuck-architecture-gate-'));
  t.after(() => rmSync(root, { recursive: true, force: true }));
  const env = { ...process.env, GIT_CONFIG_GLOBAL: '/dev/null', GIT_CONFIG_NOSYSTEM: '1',
    GIT_AUTHOR_NAME: 'Fixture', GIT_AUTHOR_EMAIL: 'fixture@example.invalid',
    GIT_COMMITTER_NAME: 'Fixture', GIT_COMMITTER_EMAIL: 'fixture@example.invalid' };
  const git = (...args) => execFileSync('git', args, { cwd: root, env, encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim();
  const write = (path, content) => {
    const full = join(root, path);
    mkdirSync(dirname(full), { recursive: true });
    writeFileSync(full, content);
  };
  const commit = (files, message = 'fixture change') => {
    Object.entries(files).forEach(([path, content]) => write(path, content));
    git('add', '.'); git('commit', '-qm', message);
    return git('rev-parse', 'HEAD');
  };
  git('init', '-q');
  commit({ 'app.txt': 'old behavior', 'docs/features/HANDOFF.md': clearHandoff });
  git('tag', '-a', 'v1.0.0', '-m', 'previous release');
  const reviewed = commit({ 'app.txt': 'candidate behavior' });
  const receipt = { schema: 1, previous_release: 'v1.0.0', reviewed_commit: reviewed,
    review_complete: true, reviewer: 'independent fixture reviewer',
    coverage: ['Saved work: checked actual repair boundary; native adoption remains unproved.'], findings: [], updates: [] };
  return { root, git, commit, reviewed, receipt,
    save: () => commit({ [receiptPath]: JSON.stringify(receipt), 'docs/features/HANDOFF.md': clearHandoff }, 'record review/closure'),
    check: (...args) => spawnSync(process.execPath, [checker, ...args], { cwd: root, env, encoding: 'utf8' }) };
}
function rejected(result, message) {
  assert.equal(result.status, 1, result.stdout + result.stderr);
  assert.match(result.stderr, message);
}
function accepted(result) { assert.equal(result.status, 0, result.stderr); }

test('missing committed review blocks a release', t => {
  const f = fixture(t);
  rejected(f.check(), /Missing committed.*v1\.0\.0-next\.json/);
});
test('completed review and its administrative receipt pass, including release tag CI', t => {
  const f = fixture(t); f.save();
  f.git('tag', '-a', 'v1.1.0', '-m', 'candidate release');
  accepted(f.check('--release-tag', 'v1.1.0'));
});
test('incomplete review and unresolved Must-fix remain blocking', t => {
  const f = fixture(t);
  f.receipt.review_complete = false; f.save();
  rejected(f.check(), /review is incomplete/);
  f.receipt.review_complete = true;
  f.receipt.findings = [{ id: 'ARCH-1', severity: 'must_fix' }]; f.save();
  rejected(f.check(), /Unresolved.*ARCH-1/);
});
test('review -> finding fix -> recorded closure passes without another architectural review', t => {
  const f = fixture(t);
  f.receipt.findings = [{ id: 'ARCH-1', severity: 'must_fix' }]; f.save();
  const repair = f.commit({ 'app.txt': 'repaired behavior' }, 'repair the reported defect');
  f.receipt.updates = [{ kind: 'fix', commits: [repair], finding_ids: ['ARCH-1'], closes: ['ARCH-1'],
    verified: true, evidence: 'Original regression and affected recovery path passed.' }]; f.save();
  assert.equal(f.receipt.reviewed_commit, f.reviewed);
  accepted(f.check());
});
test('a claimed closure cannot hide a live handoff Must-fix', t => {
  const f = fixture(t); f.save();
  f.commit({ 'docs/features/HANDOFF.md': '# Handoff\n## Review findings\n- **Must fix** — OTHER-1 — lost work\n## Changelog\n' });
  rejected(f.check(), /Live handoff still has Must-fix/);
});
test('unrecorded substantive work blocks; focused coverage admits it', t => {
  const f = fixture(t); f.save();
  const change = f.commit({ 'app.txt': 'unrelated new behavior' });
  rejected(f.check(), /Uncovered substantive commit/);
  f.receipt.updates = [{ kind: 'coverage', commits: [change], verified: true,
    evidence: 'Independent focused review of the newly affected owner and consumer.' }]; f.save();
  accepted(f.check());
});
test('unverified or unknown-finding fixes cannot justify closure', t => {
  const f = fixture(t);
  f.receipt.findings = [{ id: 'ARCH-1', severity: 'must_fix' }]; f.save();
  const repair = f.commit({ 'app.txt': 'repair' });
  const update = { kind: 'fix', commits: [repair], finding_ids: ['ARCH-1'], closes: ['ARCH-1'],
    verified: false, evidence: 'not yet checked' };
  f.receipt.updates = [update]; f.save();
  rejected(f.check(), /verified evidence/);
  update.verified = true; update.finding_ids = ['OTHER-1']; update.closes = ['OTHER-1']; f.save();
  rejected(f.check(), /reference original findings/);
});
test('a receipt for the wrong release boundary or reviewed ancestry is rejected', t => {
  const f = fixture(t);
  f.receipt.previous_release = 'v0.9.0'; f.save();
  rejected(f.check(), /wrong previous-release boundary/);
  f.receipt.previous_release = 'v1.0.0'; f.receipt.reviewed_commit = 'a'.repeat(40); f.save();
  rejected(f.check(), /not a full ancestor/);
});
