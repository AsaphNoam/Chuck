"""Check links and ownership, not prose spelling, for role launchers."""
import pathlib
import re
import sys

root = pathlib.Path(sys.argv[1])
errors = []
text = (root / "docs/features/AGENT-WORKFLOW.md").read_text()

def slug(title):
    return re.sub(r"[^\w\- ]", "", title.lower()).replace(" ", "-")

sections = {}
procedures = {}
procedure_owners = {}
current = None
procedure = None
anchors = set()
for line in text.splitlines():
    match = re.match(r"^(#{2,6}) (.+)$", line)
    if match:
        anchor = slug(match[2])
        anchors.add(anchor)
        procedure = anchor
        procedures[anchor] = ""
        if len(match[1]) == 2:
            current = int(match[2].split(".")[0])
            sections[current] = ""
        procedure_owners[anchor] = current
    else:
        if procedure:
            procedures[procedure] += line + "\n"
    if current is not None:
        sections[current] += line + "\n"

roles = dict(zip(
    "work review fix usability-review design-feature investigate-bug review-design design ux release write-tests architectural-review".split(),
    [2, 7, 8, 9, 11, 12, 13, 14, 15, 16, 17, 18]))
role_anchors = {}
for number in roles.values():
    heading = re.search(rf"^## {number}\. (.+)$", text, re.M)
    if heading is None:
        errors.append(f"workflow: missing role section {number}")
        role_anchors[number] = f"missing-role-{number}"
        sections[number] = ""
    else:
        role_anchors[number] = slug(heading[0][3:])

def links(body):
    return re.findall(r"\[[^\]]+\]\(([^)]+)\)", body)

def require(body, target, label):
    if target not in links(body):
        errors.append(f"{label}: missing link {target}")

for role, number in roles.items():
    copies = []
    for tree in (".agents", ".claude"):
        path = root / tree / "skills" / role / "SKILL.md"
        label = f"{tree}/{role}"
        if not path.exists():
            errors.append(f"{label}: launcher is missing")
            continue
        body = path.read_text()
        copies.append(body)
        frontmatter = re.match(r"\A---\n(.*?)\n---\n", body, re.S)
        if not frontmatter or not re.search(rf"^name: {re.escape(role)}$", frontmatter[1], re.M):
            errors.append(f"{label}: invalid role frontmatter")
        else:
            description = frontmatter[1]
            if role == "architectural-review":
                if "release" not in description or not re.search(r"explicit|/architectural-review", description, re.I):
                    errors.append(f"{label}: missing release or explicit architecture trigger")
            elif role in ("design", "ux", "write-tests"):
                if not re.search(r"automatic|automatically|accompanies", description, re.I):
                    errors.append(f"{label}: missing automatic companion trigger")
            elif "Explicit invocation only" not in description:
                errors.append(f"{label}: missing explicit-only trigger")
            body = body[frontmatter.end():]
        prefix = "../../../docs/features/AGENT-WORKFLOW.md#"
        require(body, prefix + "1-start-with-the-current-state", label)
        require(body, prefix + role_anchors[number], label)
        for target in links(body):
            if target.startswith(prefix) and target[len(prefix):] not in anchors:
                errors.append(f"{label}: nonexistent workflow anchor {target}")
    if len(copies) == 2 and copies[0] != copies[1]:
        errors.append(f"{role}: Claude and Codex launchers differ")

for number, target_number in ((2, 17), (8, 17), (16, 18)):
    require(sections[number], "#" + role_anchors[target_number], f"workflow §{number}")
require(sections[7], "MODEL-ROUTING.md", "workflow §7")
if not (root / "docs/features/MODEL-ROUTING.md").is_file():
    errors.append("workflow §7: model routing target is missing")
for number in (8, 12, 18):
    require(sections[number], "#finding-and-fix-routing", f"workflow §{number}")
for number, target in ((8, "independent-fix-check"), (11, "lifecycle-design"), (16, "receipt-and-release-gate")):
    require(sections[number], "#" + target, f"workflow §{number}")
for target, owner in (("finding-and-fix-routing", 7), ("independent-fix-check", 8), ("lifecycle-design", 11), ("receipt-and-release-gate", 18)):
    if target not in anchors:
        errors.append(f"workflow: missing procedure {target}")
    elif procedure_owners[target] != owner:
        errors.append(f"workflow: procedure {target} must belong to §{owner}")
    elif not re.search(r"\b[a-zA-Z]{3,}\b", re.sub(r"\[[^\]]+\]\([^)]+\)", "", procedures[target]).strip()):
        errors.append(f"workflow: empty procedure {target}")

for error in errors:
    print("launcher contract: " + error, file=sys.stderr)
sys.exit(bool(errors))
