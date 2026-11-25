import pathlib
import re
import shutil
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parent.parent
checker = root / "scripts/check-launcher-contract.sh"
errors = []

with tempfile.TemporaryDirectory() as temporary:
    base = pathlib.Path(temporary) / "base"
    for relative in (".agents/skills", ".claude/skills", "docs/features"):
        shutil.copytree(root / relative, base / relative)

    def run(tree):
        return subprocess.run([str(checker), "--root", str(tree)], capture_output=True, text=True)

    result = run(base)
    if result.returncode:
        raise SystemExit("launcher contract test baseline failed:\n" + result.stderr)

    def mutation(name, paths, old, new, diagnostic):
        tree = pathlib.Path(temporary) / name
        shutil.copytree(base, tree)
        for relative in paths:
            path = tree / relative
            body = path.read_text()
            if old not in body:
                raise AssertionError(f"{name}: mutation did not match {relative}")
            path.write_text(body.replace(old, new))
        result = run(tree)
        if result.returncode == 0 or diagnostic not in result.stderr:
            errors.append(f"{name}: expected failure {diagnostic!r}; got {result.stderr!r}")

    def twins(role):
        return [f"{tree}/skills/{role}/SKILL.md" for tree in (".agents", ".claude")]

    workflow = ["docs/features/AGENT-WORKFLOW.md"]
    mutation("missing-start", twins("work"), "AGENT-WORKFLOW.md#1-start-with-the-current-state", "AGENT-WORKFLOW.md", "missing link")
    mutation("missing-role", twins("review"), "AGENT-WORKFLOW.md#7-review-work", "AGENT-WORKFLOW.md", "missing link")
    mutation("invalid-target", twins("fix"), "AGENT-WORKFLOW.md#8-fix-review-findings", "AGENT-WORKFLOW.md#missing", "nonexistent workflow anchor")
    mutation("divergent", [twins("ux")[0]], "#", "##", "launchers differ")
    mutation("identity", twins("work"), "name: work", "name: review", "invalid role frontmatter")
    mutation("explicit-trigger", twins("review"), "Explicit invocation only", "Always invoke", "missing explicit-only trigger")
    mutation("automatic-trigger", twins("design"), "or automatically", "or optionally", "missing automatic companion trigger")
    for target in ("17-write-purposeful-tests", "18-review-architecture-for-a-reliable-development-hub", "finding-and-fix-routing", "independent-fix-check", "lifecycle-design", "receipt-and-release-gate"):
        mutation("link-" + target, workflow, f"](#{target})", "](missing)", "missing link")
    mutation("map-link", workflow, "](MODEL-ROUTING.md)", "](missing)", "missing link MODEL-ROUTING.md")
    tree = pathlib.Path(temporary) / "missing-map"
    shutil.copytree(base, tree)
    (tree / "docs/features/MODEL-ROUTING.md").unlink()
    result = run(tree)
    if result.returncode == 0 or "model routing target is missing" not in result.stderr:
        errors.append("missing model map was not rejected: " + result.stderr)
    mutation("role-heading", workflow, "## 7. Review work", "## 70. Review work", "missing role section 7")
    for heading in ("Finding and fix routing", "Independent fix check", "Lifecycle design", "Receipt and release gate"):
        mutation("heading-" + heading.replace(" ", "-"), workflow, "### " + heading, "### Removed procedure", "missing procedure")
    mutation("misplaced-procedure", workflow, "### Independent fix check", "## 81. Wrong owner\n\n### Independent fix check", "procedure independent-fix-check must belong to §8")

    tree = pathlib.Path(temporary) / "empty-procedure"
    shutil.copytree(base, tree)
    path = tree / workflow[0]
    body, count = re.subn(r"(### Independent fix check\n).*?(?=\n#{2,3} |\Z)", r"\1\n", path.read_text(), flags=re.S)
    assert count == 1, "empty procedure mutation did not match"
    path.write_text(body)
    result = run(tree)
    if result.returncode == 0 or "empty procedure independent-fix-check" not in result.stderr:
        errors.append("empty procedure was not rejected: " + result.stderr)

    # Rephrasing display text leaves link identity and routing intact.
    tree = pathlib.Path(temporary) / "paraphrase"
    shutil.copytree(base, tree)
    for path in [tree / p for p in twins("work")]:
        body = path.read_text()
        path.write_text(re.sub(r"\[[^\]]+\](\(\.\./\.\./\.\./docs/features/AGENT-WORKFLOW\.md#[^)]+\))", r"[Follow the documented procedure]\1", body))
    result = run(tree)
    if result.returncode:
        errors.append("prose paraphrase failed: " + result.stderr)

if errors:
    raise SystemExit("\n".join(errors))
