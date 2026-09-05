#!/usr/bin/env python3
# This file is part of PeerFoil.
# tests/static_checks.py
# Author(s): Gabriel Mongefranco.
# Created: 2026-09-05
# Last Modified: 2026-09-05
# Summary: Runs PeerFoil's static repository checks: required files, JSON and schema validity,
#          skill and agent frontmatter, relative links, file notices, manifests, packs, and stale
#          project names.
# Notes: Uses only the Python 3 standard library so it runs from a clean checkout on Windows,
#        macOS, and Linux. Run it from any directory: python tests/static_checks.py
# Copyright © 2026 Gabriel Mongefranco
# SPDX-License-Identifier: GPL-3.0-or-later
"""Static checks for the PeerFoil repository.

The script prints one line per problem, prefixed with ``static checks:``, and exits with
status 1 when any problem is found. It exits with status 0 and a short summary otherwise.

The JSON Schema validator below supports only the keyword subset documented in
``schemas/README.md``. A schema that uses another keyword fails the schema audit so the
subset stays honest.
"""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path, PurePosixPath

REPO_ROOT = Path(__file__).resolve().parents[1]
PLUGIN_DIR = REPO_ROOT / "plugins" / "peerfoil"
SCHEMA_DIR = REPO_ROOT / "schemas"
TEMPLATE_DIR = PLUGIN_DIR / "templates"

SKIP_DIRS = {
    ".git", "node_modules", "__pycache__", ".venv", ".claude", ".idea", ".vscode",
    "tmp", "temp", "dist", "bin", "coverage", "worktrees",
}
CHECKED_SUFFIXES = {".md", ".json", ".jsonl", ".py", ".sh", ".yml", ".yaml"}
CHECKED_NAMES = {".gitignore", ".gitattributes", "NOTICE"}

# Files that legitimately carry no PeerFoil header of their own.
NOTICE_EXEMPT = {
    "LICENSE",
    "plugins/peerfoil/LICENSE",
    # The strict plugin validator rejects unknown fields; the notice is in a sibling file.
    ".claude-plugin/marketplace.json",
}

EXPECTED_SKILLS = [
    "setup", "start", "change", "status", "resume", "review-phase", "remember", "settings",
]
EXPECTED_PACKS = ["software", "generic", "documentation"]

REQUIRED_FILES = [
    "README.md", "AGENTS.md", "LICENSE", "NOTICE", ".zenodo.json",
    "docs/PeerFoil-Method.md", "docs/architecture.md", "docs/implementation-plan.md",
    "docs/phase-prompt-template.md", "docs/decision-log.md",
    "docs/plans/phase-1-skills.md", "docs/plans/phase-2-core-alpha.md",
    ".claude-plugin/marketplace.json", ".claude-plugin/NOTICE.md",
    "plugins/peerfoil/.claude-plugin/plugin.json", "plugins/peerfoil/README.md",
    "plugins/peerfoil/LICENSE",
    "plugins/peerfoil/agents/evaluator.md",
    "plugins/peerfoil/references/workflow.md", "plugins/peerfoil/references/records.md",
    "plugins/peerfoil/references/lineage.md",
    "plugins/peerfoil/templates/README.md",
    "schemas/README.md", "schemas/common.schema.json", "schemas/project.schema.json",
    "schemas/plan.schema.json", "schemas/transition.schema.json", "schemas/pack.schema.json",
] + [f"plugins/peerfoil/skills/{name}/SKILL.md" for name in EXPECTED_SKILLS] + [
    f"plugins/peerfoil/packs/{name}/{file}"
    for name in EXPECTED_PACKS for file in ("pack.json", "README.md")
] + [
    f"plugins/peerfoil/templates/{name}" for name in (
        "project.json", "decisions.md", "architecture.md", "quality.md", "plan.md",
        "plan.json", "history.jsonl", "evidence.md", "change-set.md", "review.md",
        "lesson.md",
    )
]

# Frontmatter fields accepted by the Agent Skills specification and by Claude Code.
SKILL_FRONTMATTER_FIELDS = {
    "name", "description", "license", "compatibility", "metadata", "allowed-tools",
    "argument-hint", "disable-model-invocation", "user-invocable", "model", "effort",
    "context", "agent", "hooks", "paths", "when_to_use", "arguments", "disallowed-tools",
    "background", "shell",
}
AGENT_FRONTMATTER_FIELDS = {
    "name", "description", "model", "effort", "maxTurns", "tools", "disallowedTools",
    "skills", "memory", "background", "isolation",
}
SKILL_NAME_PATTERN = re.compile(r"^[a-z0-9]+(-[a-z0-9]+)*$")
FRONTMATTER_LINE = re.compile(r"^([A-Za-z_][A-Za-z0-9_-]*):\s*(.*)$")
STALE_IDENTITY = re.compile(
    r"(^# Crosscut|Project:  (Crosscut|Privatium)|github\.com/gabrielmongefranco/crosscut)",
    re.MULTILINE,
)
DATE = r"20[0-9]{2}-[0-9]{2}-[0-9]{2}"
CREATED = re.compile(r'(Created:[ \t]*|"_?created":[ \t]*"|Created: )' + DATE)
MODIFIED = re.compile(r'((Last Modified|Modified):[ \t]*|"_?modified":[ \t]*"|Last Modified: )' + DATE)
MARKDOWN_LINK = re.compile(r"(?<!!)\[[^\]]*\]\(([^)\s]+)(?:\s+\"[^\"]*\")?\)")
FENCED_CODE = re.compile(r"```.*?```", re.DOTALL)
INLINE_CODE = re.compile(r"`[^`\n]*`")

SCHEMA_KEYWORDS = {
    "$schema", "$id", "$comment", "$defs", "$ref", "title", "type", "const", "enum",
    "pattern", "minLength", "maxLength", "minimum", "maximum", "properties", "required",
    "additionalProperties", "items", "minItems", "maxItems",
}

JSON_TYPES = {
    "object": lambda value: isinstance(value, dict),
    "array": lambda value: isinstance(value, list),
    "string": lambda value: isinstance(value, str),
    "integer": lambda value: isinstance(value, int) and not isinstance(value, bool),
    "number": lambda value: isinstance(value, (int, float)) and not isinstance(value, bool),
    "boolean": lambda value: isinstance(value, bool),
    "null": lambda value: value is None,
}


class Problems:
    """Collects problems as (relative path, message) pairs."""

    def __init__(self) -> None:
        self.items: list[tuple[str, str]] = []

    def add(self, path: Path | str, message: str) -> None:
        self.items.append((relative(path), message))

    def __len__(self) -> int:
        return len(self.items)


def relative(path: Path | str) -> str:
    if isinstance(path, str):
        return path
    try:
        return path.resolve().relative_to(REPO_ROOT).as_posix()
    except ValueError:
        return path.as_posix()


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def checked_files() -> list[Path]:
    files: list[Path] = []
    for path in sorted(REPO_ROOT.rglob("*")):
        if any(part in SKIP_DIRS for part in path.relative_to(REPO_ROOT).parts):
            continue
        if not path.is_file():
            continue
        if path.suffix in CHECKED_SUFFIXES or path.name in CHECKED_NAMES:
            files.append(path)
    return files


# --- JSON Schema subset validator -------------------------------------------------------


class SchemaValidator:
    """Validates instances against the JSON Schema subset listed in schemas/README.md."""

    def __init__(self, schema_dir: Path) -> None:
        self.schemas: dict[str, dict] = {}
        for path in sorted(schema_dir.glob("*.schema.json")):
            self.schemas[path.name] = json.loads(read_text(path))

    def audit_keywords(self, problems: Problems) -> None:
        for name, schema in self.schemas.items():
            self._audit(schema, name, "$", problems)

    def _audit(self, node: object, file: str, where: str, problems: Problems) -> None:
        if isinstance(node, dict):
            for key, value in node.items():
                if key not in SCHEMA_KEYWORDS:
                    problems.add(f"schemas/{file}", f"unsupported schema keyword '{key}' at {where}")
                if key in ("properties", "$defs"):
                    for child_name, child in value.items():
                        self._audit(child, file, f"{where}.{key}.{child_name}", problems)
                elif key == "items":
                    self._audit(value, file, f"{where}.items", problems)
        elif isinstance(node, list):
            for index, item in enumerate(node):
                self._audit(item, file, f"{where}[{index}]", problems)

    def resolve(self, ref: str, current_file: str) -> tuple[dict, str]:
        file_part, _, pointer = ref.partition("#")
        file_name = file_part or current_file
        if file_name not in self.schemas:
            raise KeyError(f"unknown schema file in $ref '{ref}'")
        node: object = self.schemas[file_name]
        for segment in pointer.split("/"):
            if segment == "":
                continue
            if not isinstance(node, dict) or segment not in node:
                raise KeyError(f"unresolvable $ref '{ref}'")
            node = node[segment]
        if not isinstance(node, dict):
            raise KeyError(f"$ref '{ref}' does not point to a schema object")
        return node, file_name

    def validate(self, instance: object, schema_file: str) -> list[str]:
        errors: list[str] = []
        self._validate(instance, self.schemas[schema_file], schema_file, "$", errors)
        return errors

    def _validate(self, value: object, schema: dict, file: str, where: str, errors: list[str]) -> None:
        if "$ref" in schema:
            target, target_file = self.resolve(schema["$ref"], file)
            self._validate(value, target, target_file, where, errors)
            return

        if "type" in schema:
            allowed = schema["type"] if isinstance(schema["type"], list) else [schema["type"]]
            if not any(JSON_TYPES[type_name](value) for type_name in allowed):
                errors.append(f"{where}: expected type {allowed}")
                return

        if "const" in schema and value != schema["const"]:
            errors.append(f"{where}: expected constant {schema['const']!r}")
        if "enum" in schema and value not in schema["enum"]:
            errors.append(f"{where}: value {value!r} not in {schema['enum']}")

        if isinstance(value, str):
            if "pattern" in schema and re.search(schema["pattern"], value) is None:
                errors.append(f"{where}: {value!r} does not match {schema['pattern']}")
            if "minLength" in schema and len(value) < schema["minLength"]:
                errors.append(f"{where}: shorter than {schema['minLength']}")
            if "maxLength" in schema and len(value) > schema["maxLength"]:
                errors.append(f"{where}: longer than {schema['maxLength']}")

        if isinstance(value, (int, float)) and not isinstance(value, bool):
            if "minimum" in schema and value < schema["minimum"]:
                errors.append(f"{where}: below minimum {schema['minimum']}")
            if "maximum" in schema and value > schema["maximum"]:
                errors.append(f"{where}: above maximum {schema['maximum']}")

        if isinstance(value, dict):
            properties = schema.get("properties", {})
            for name in schema.get("required", []):
                if name not in value:
                    errors.append(f"{where}: missing required field '{name}'")
            for name, child in value.items():
                if name in properties:
                    self._validate(child, properties[name], file, f"{where}.{name}", errors)
                elif schema.get("additionalProperties", True) is False:
                    errors.append(f"{where}: unexpected field '{name}'")

        if isinstance(value, list):
            if "minItems" in schema and len(value) < schema["minItems"]:
                errors.append(f"{where}: fewer than {schema['minItems']} items")
            if "maxItems" in schema and len(value) > schema["maxItems"]:
                errors.append(f"{where}: more than {schema['maxItems']} items")
            if "items" in schema:
                for index, item in enumerate(value):
                    self._validate(item, schema["items"], file, f"{where}[{index}]", errors)


# --- Individual checks ------------------------------------------------------------------


def check_required_files(problems: Problems) -> None:
    for rel in REQUIRED_FILES:
        path = REPO_ROOT / rel
        if not path.is_file() or path.stat().st_size == 0:
            problems.add(rel, "required file is missing or empty")


def check_json_syntax(files: list[Path], problems: Problems) -> None:
    for path in files:
        if path.suffix == ".json":
            try:
                json.loads(read_text(path))
            except (ValueError, UnicodeDecodeError) as error:
                problems.add(path, f"invalid JSON: {error}")
        elif path.suffix == ".jsonl":
            for number, line in enumerate(read_text(path).splitlines(), start=1):
                if not line.strip():
                    continue
                try:
                    json.loads(line)
                except ValueError as error:
                    problems.add(path, f"line {number}: invalid JSON: {error}")


def load_json(path: Path, problems: Problems) -> object | None:
    try:
        return json.loads(read_text(path))
    except (ValueError, UnicodeDecodeError):
        return None


def check_schemas(problems: Problems) -> None:
    validator = SchemaValidator(SCHEMA_DIR)
    validator.audit_keywords(problems)

    def validate_file(path: Path, schema_file: str) -> None:
        if not path.is_file():
            return
        if path.suffix == ".jsonl":
            for number, line in enumerate(read_text(path).splitlines(), start=1):
                if not line.strip():
                    continue
                try:
                    instance = json.loads(line)
                except ValueError:
                    continue
                for error in validator.validate(instance, schema_file):
                    problems.add(path, f"line {number}: {error}")
            return
        instance = load_json(path, problems)
        if instance is None:
            return
        for error in validator.validate(instance, schema_file):
            problems.add(path, error)

    validate_file(TEMPLATE_DIR / "project.json", "project.schema.json")
    validate_file(TEMPLATE_DIR / "plan.json", "plan.schema.json")
    validate_file(TEMPLATE_DIR / "history.jsonl", "transition.schema.json")
    for pack_json in sorted((PLUGIN_DIR / "packs").glob("*/pack.json")):
        validate_file(pack_json, "pack.schema.json")
    fixtures = REPO_ROOT / "fixtures"
    if fixtures.is_dir():
        for project_json in sorted(fixtures.rglob(".peerfoil/project.json")):
            validate_file(project_json, "project.schema.json")
            validate_file(project_json.parent / "plan.json", "plan.schema.json")
            validate_file(project_json.parent / "history.jsonl", "transition.schema.json")


def parse_frontmatter(path: Path, problems: Problems) -> tuple[dict[str, str], str] | None:
    text = read_text(path)
    lines = text.splitlines()
    if not lines or lines[0].strip() != "---":
        problems.add(path, "frontmatter must start on the first line")
        return None
    fields: dict[str, str] = {}
    for index in range(1, len(lines)):
        line = lines[index]
        if line.strip() == "---":
            body = "\n".join(lines[index + 1:])
            return fields, body
        match = FRONTMATTER_LINE.match(line)
        if match is None:
            problems.add(path, f"frontmatter line {index + 1} is not a simple 'key: value' line")
            continue
        key, value = match.group(1), match.group(2).strip()
        if len(value) >= 2 and value[0] == value[-1] and value[0] in "\"'":
            value = value[1:-1]
        if key in fields:
            problems.add(path, f"duplicate frontmatter field '{key}'")
        fields[key] = value
    problems.add(path, "frontmatter is not closed with '---'")
    return None


def check_skills(problems: Problems) -> None:
    skills_dir = PLUGIN_DIR / "skills"
    found = sorted(p.name for p in skills_dir.iterdir() if p.is_dir()) if skills_dir.is_dir() else []
    if found != sorted(EXPECTED_SKILLS):
        problems.add("plugins/peerfoil/skills", f"expected skills {sorted(EXPECTED_SKILLS)}, found {found}")
    for name in found:
        path = skills_dir / name / "SKILL.md"
        if not path.is_file():
            problems.add(path, "SKILL.md is missing")
            continue
        parsed = parse_frontmatter(path, problems)
        if parsed is None:
            continue
        fields, body = parsed
        for key in fields:
            if key not in SKILL_FRONTMATTER_FIELDS:
                problems.add(path, f"unknown frontmatter field '{key}'")
        skill_name = fields.get("name", "")
        if skill_name != name:
            problems.add(path, f"frontmatter name '{skill_name}' must equal directory name '{name}'")
        if not SKILL_NAME_PATTERN.match(skill_name) or len(skill_name) > 64:
            problems.add(path, "name must be lowercase letters, digits, and single hyphens, at most 64 characters")
        description = fields.get("description", "")
        if not 1 <= len(description) <= 1024:
            problems.add(path, "description must be 1 to 1024 characters")
        if fields.get("license") != "GPL-3.0-or-later":
            problems.add(path, "license must be GPL-3.0-or-later")
        if "Guided" not in body:
            problems.add(path, "skill body must state the Guided assurance level")
        if "This file is part of PeerFoil." not in body:
            problems.add(path, "skill body must carry the PeerFoil header comment after the frontmatter")


def check_agents(problems: Problems) -> None:
    agents_dir = PLUGIN_DIR / "agents"
    if not agents_dir.is_dir():
        problems.add("plugins/peerfoil/agents", "agents directory is missing")
        return
    for path in sorted(agents_dir.glob("*.md")):
        parsed = parse_frontmatter(path, problems)
        if parsed is None:
            continue
        fields, body = parsed
        for key in fields:
            if key not in AGENT_FRONTMATTER_FIELDS:
                problems.add(path, f"unknown frontmatter field '{key}'")
        if fields.get("name") != path.stem:
            problems.add(path, f"frontmatter name must equal file name '{path.stem}'")
        if not fields.get("description"):
            problems.add(path, "description is required")
        if "This file is part of PeerFoil." not in body:
            problems.add(path, "agent body must carry the PeerFoil header comment after the frontmatter")


def check_markdown_links(files: list[Path], problems: Problems) -> None:
    for path in files:
        if path.suffix != ".md":
            continue
        # Code spans and fenced blocks may contain bracket-and-parenthesis text that is
        # not a link, such as regular expressions.
        text = INLINE_CODE.sub("", FENCED_CODE.sub("", read_text(path)))
        for match in MARKDOWN_LINK.finditer(text):
            target = match.group(1)
            if target.startswith(("http://", "https://", "mailto:", "#")):
                continue
            if "{{" in target or "${" in target:
                continue
            target_path, _, _ = target.partition("#")
            if not target_path:
                continue
            resolved = (path.parent / PurePosixPath(target_path).as_posix()).resolve()
            if not resolved.exists():
                problems.add(path, f"relative link target does not exist: {target}")


def check_notices(files: list[Path], problems: Problems) -> None:
    for path in files:
        rel = relative(path)
        if rel in NOTICE_EXEMPT or rel.startswith("plugins/peerfoil/templates/"):
            continue
        text = read_text(path)
        head = text if path.suffix in (".json", ".jsonl") else "\n".join(text.splitlines()[:40])
        if "PeerFoil" not in head:
            problems.add(path, "missing PeerFoil header")
        if rel not in head:
            problems.add(path, "header must name the file's repository-relative path")
        if CREATED.search(head) is None:
            problems.add(path, "missing created date in header")
        if MODIFIED.search(head) is None:
            problems.add(path, "missing modified date in header")
        if "Copyright © 2026 Gabriel Mongefranco" not in text and "SPDX-License-Identifier:" not in text:
            problems.add(path, "missing copyright or SPDX notice")


def check_templates(problems: Problems) -> None:
    readme = TEMPLATE_DIR / "README.md"
    if not readme.is_file():
        return
    readme_text = read_text(readme)
    if "GPL-3.0-or-later" not in readme_text or "Copyright © 2026 Gabriel Mongefranco" not in readme_text:
        problems.add(readme, "template directory notice must state the license and copyright")
    for path in sorted(TEMPLATE_DIR.iterdir()):
        if path.name == "README.md" or not path.is_file():
            continue
        text = read_text(path)
        if "Copyright © 2026 Gabriel Mongefranco" in text or "This file is part of PeerFoil." in text:
            problems.add(path, "a template must not carry a PeerFoil header; copies belong to the user")
        if path.suffix == ".md" and not text.startswith("<!-- PeerFoil project record"):
            problems.add(path, "a Markdown template must start with the generated-by comment")


def check_stale_identity(files: list[Path], problems: Problems) -> None:
    for path in files:
        if STALE_IDENTITY.search(read_text(path)):
            problems.add(path, "stale project identity found")


def check_manifests(problems: Problems) -> None:
    marketplace_path = REPO_ROOT / ".claude-plugin" / "marketplace.json"
    plugin_path = PLUGIN_DIR / ".claude-plugin" / "plugin.json"
    marketplace = load_json(marketplace_path, problems)
    plugin = load_json(plugin_path, problems)
    if not isinstance(marketplace, dict) or not isinstance(plugin, dict):
        return
    entries = [entry for entry in marketplace.get("plugins", []) if entry.get("name") == "peerfoil"]
    if len(entries) != 1:
        problems.add(marketplace_path, "marketplace must list exactly one plugin named 'peerfoil'")
        return
    entry = entries[0]
    source = entry.get("source")
    if source != "./plugins/peerfoil" or not (REPO_ROOT / "plugins" / "peerfoil").is_dir():
        problems.add(marketplace_path, "plugin source must be the relative path ./plugins/peerfoil")
    if plugin.get("name") != "peerfoil":
        problems.add(plugin_path, "plugin name must be 'peerfoil'")
    if entry.get("version") != plugin.get("version"):
        problems.add(marketplace_path, "marketplace entry version must equal plugin.json version")
    if plugin.get("license") != "GPL-3.0-or-later" or entry.get("license") != "GPL-3.0-or-later":
        problems.add(plugin_path, "plugin license must be GPL-3.0-or-later in both manifests")
    metadata = plugin.get("metadata", {})
    if metadata.get("file") != "plugins/peerfoil/.claude-plugin/plugin.json":
        problems.add(plugin_path, "metadata.file must name the manifest's repository path")
    notice = REPO_ROOT / ".claude-plugin" / "NOTICE.md"
    if not notice.is_file() or "marketplace.json" not in read_text(notice):
        problems.add(notice, "sibling notice for marketplace.json is missing or does not name it")


def check_packs(problems: Problems) -> None:
    packs_dir = PLUGIN_DIR / "packs"
    found = sorted(p.name for p in packs_dir.iterdir() if p.is_dir()) if packs_dir.is_dir() else []
    for name in EXPECTED_PACKS:
        if name not in found:
            problems.add(f"plugins/peerfoil/packs/{name}", "required pack is missing")
    for name in found:
        pack_path = packs_dir / name / "pack.json"
        pack = load_json(pack_path, problems) if pack_path.is_file() else None
        if not isinstance(pack, dict):
            problems.add(pack_path, "pack.json is missing or invalid")
            continue
        if pack.get("id") != name:
            problems.add(pack_path, f"pack id must equal directory name '{name}'")
        if pack.get("_file") != relative(pack_path):
            problems.add(pack_path, "_file must name the manifest's repository path")
        for probe in pack.get("tools", []) + pack.get("project_tool_hints", []):
            command = probe.get("command", [])
            if any(any(char in part for char in ";&|`$<>") for part in command):
                problems.add(pack_path, f"tool command must be a plain argument list: {command}")
        if not (packs_dir / name / "README.md").is_file():
            problems.add(packs_dir / name / "README.md", "pack README is missing")


def main() -> int:
    problems = Problems()
    files = checked_files()

    check_required_files(problems)
    check_json_syntax(files, problems)
    check_schemas(problems)
    check_skills(problems)
    check_agents(problems)
    check_markdown_links(files, problems)
    check_notices(files, problems)
    check_templates(problems)
    check_stale_identity(files, problems)
    check_manifests(problems)
    check_packs(problems)

    for path, message in problems.items:
        print(f"static checks: {path}: {message}", file=sys.stderr)
    if problems.items:
        print(f"static checks: {len(problems)} problem(s) found", file=sys.stderr)
        return 1
    print(f"static checks: PeerFoil static checks pass ({len(files)} files checked)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
