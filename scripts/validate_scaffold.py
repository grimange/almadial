#!/usr/bin/env python3
"""Verify Codex starter-kit structure. No network or system modifications."""
from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
REQUIRED = (
    "AGENTS.md",
    "README.md",
    "docs/implementation-plan.md",
    "docs/GOALS.md",
    "docs/PLANS.md",
    "docs/CODEX_SETUP.md",
    "manifests/stack.lock.yaml",
    "prompts/M0-01-codex.md",
    "templates/task.md",
    "templates/acceptance-evidence.md",
    "templates/independent-review.md",
)
EXPECTED_SKILLS = {
    "almadial-compatibility",
    "almadial-install-stage",
    "almadial-webrtc-test",
    "almadial-review",
}


def parse_frontmatter(content: str) -> dict[str, str]:
    lines = content.splitlines()
    if len(lines) < 4 or lines[0].strip() != "---":
        raise ValueError("missing opening YAML frontmatter marker")
    try:
        end = lines.index("---", 1)
    except ValueError as e:
        raise ValueError("missing closing YAML frontmatter marker") from e
    fields: dict[str, str] = {}
    for line in lines[1:end]:
        m = re.fullmatch(r"([a-z_]+):\s*(.+?)\s*", line)
        if not m:
            raise ValueError(f"unexpected frontmatter line: {line!r}")
        key, value = m.groups()
        if key in fields:
            raise ValueError(f"duplicate frontmatter key: {key}")
        fields[key] = value.strip('"\'')
    if not fields.get("name") or not fields.get("description"):
        raise ValueError("name and description are mandatory")
    if not re.fullmatch(r"[a-z0-9]+(?:-[a-z0-9]+)*", fields["name"]):
        raise ValueError("name must be lowercase kebab-case")
    if len(fields["description"]) < 40:
        raise ValueError("description must describe scope and trigger")
    return fields


def validate_tree(root: Path) -> list[str]:
    issues: list[str] = []
    for relpath in REQUIRED:
        if not (root / relpath).is_file():
            issues.append(f"missing file: {relpath}")
    agents = root / "AGENTS.md"
    if agents.is_file() and agents.stat().st_size > 14 * 1024:
        issues.append("AGENTS.md too large: move workflow detail into skills")

    skills_dir = root / ".agents" / "skills"
    found: set[str] = set()
    if not skills_dir.is_dir():
        issues.append("missing .agents/skills")
    else:
        for skill in sorted(p for p in skills_dir.iterdir() if p.is_dir()):
            filename = skill / "SKILL.md"
            if not filename.is_file():
                issues.append(f"missing SKILL.md in {skill.name}")
                continue
            try:
                meta = parse_frontmatter(filename.read_text(encoding="utf-8"))
                if meta["name"] != skill.name:
                    issues.append(f"skill folder mismatch: {skill.name} != {meta['name']}")
                if meta["name"] in found:
                    issues.append(f"duplicate skill: {meta['name']}")
                found.add(meta["name"])
            except (OSError, ValueError) as e:
                issues.append(f"invalid {filename.relative_to(root)}: {e}")
            config = skill / "agents" / "openai.yaml"
            if config.exists():
                text = config.read_text(encoding="utf-8")
                if not all(k in text for k in ("interface:", "display_name:", "short_description:")):
                    issues.append(f"incomplete skill UI metadata: {config.relative_to(root)}")
        for name in sorted(EXPECTED_SKILLS - found):
            issues.append(f"expected skill not valid: {name}")

    manifest = root / "manifests" / "stack.lock.yaml"
    if manifest.exists():
        m = manifest.read_text(encoding="utf-8")
        if not re.search(r"(?m)^status:\s*candidate\s*$", m):
            issues.append("starter manifest must remain candidate until M0 acceptance")
        if "TBD-M0" not in m:
            issues.append("starter manifest unexpectedly claims complete pins")
    return issues


def main() -> int:
    issues = validate_tree(ROOT)
    if issues:
        print("FAILED scaffold validation:")
        for issue in issues:
            print(" -", issue)
        return 1
    print(f"PASS scaffold validation: {len(REQUIRED)} required files, {len(EXPECTED_SKILLS)} skills; no VM/runtime checks performed")
    return 0


if __name__ == "__main__":
    sys.exit(main())
