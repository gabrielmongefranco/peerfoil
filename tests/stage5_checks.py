#!/usr/bin/env python3
# This file is part of PeerFoil.
# tests/stage5_checks.py
# Author(s): Gabriel Mongefranco.
# Created: 2026-10-21
# Last Modified: 2026-10-21
# Summary: Checks the three Stage 5 reference fixtures: required files, record contents,
#          seeded requirement changes, seeded review defects, and the no-header rule for
#          synthetic fixture content.
# Notes: Uses only the Python 3 standard library. Schema validation itself runs in
#        tests/static_checks.py; these tests assert the fixture-specific expectations.
# Copyright © 2026 Gabriel Mongefranco
# SPDX-License-Identifier: GPL-3.0-or-later
"""Stage 5 fixture checks for the PeerFoil repository.

Run from any directory: python tests/stage5_checks.py
"""

from __future__ import annotations

import json
import unittest
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[1]
FIXTURES = REPO_ROOT / "fixtures"

EXPECTED = {
    "software-go": {
        "project_id": "peerfoil-fixture-go",
        "pack": "software",
        "sources": ["go.mod", "main.go", "main_test.go"],
        "seeded_text": None,
    },
    "software-node": {
        "project_id": "peerfoil-fixture-node",
        "pack": "software",
        "sources": ["package.json", "linecount.js", "linecount.test.js"],
        "seeded_text": None,
    },
    "documentation": {
        "project_id": "peerfoil-fixture-doc",
        "pack": "documentation",
        "sources": ["guide.md"],
        # The fabricated statistic and the broken link are the documentation fixture's
        # seeded defects; removing either breaks the journey the fixture exists to test.
        "seeded_text": ("guide.md", ["60 percent faster", "](checklist.md)"]),
    },
}

RECORD_FILES = [
    ".peerfoil/project.json",
    ".peerfoil/decisions.md",
    ".peerfoil/architecture.md",
    ".peerfoil/quality.md",
    ".peerfoil/plan.md",
    ".peerfoil/plan.json",
    ".peerfoil/plans/plan-1.json",
    ".peerfoil/history.jsonl",
    ".peerfoil/reviews/rv-0001.md",
    ".peerfoil/reviews/rv-0002.md",
    ".peerfoil/reviews/rv-0003.md",
]


def load(path: Path) -> dict:
    return json.loads(path.read_text(encoding="utf-8"))


class FixtureLayout(unittest.TestCase):
    """Every fixture carries its README, sources, and the full record set."""

    def test_files_exist(self) -> None:
        for name, spec in EXPECTED.items():
            root = FIXTURES / name
            for rel in ["README.md", *spec["sources"], *RECORD_FILES]:
                with self.subTest(fixture=name, file=rel):
                    path = root / rel
                    self.assertTrue(path.is_file() and path.stat().st_size > 0,
                                    f"{name}/{rel} is missing or empty")

    def test_fixture_content_has_no_peerfoil_header(self) -> None:
        """Synthetic project content must not claim to be part of PeerFoil."""
        for name in EXPECTED:
            root = FIXTURES / name
            for path in root.rglob("*"):
                if not path.is_file() or path.name == "README.md":
                    continue
                text = path.read_text(encoding="utf-8")
                with self.subTest(fixture=name, file=path.name):
                    self.assertNotIn("This file is part of PeerFoil", text)
                    self.assertNotIn("Copyright © 2026 Gabriel Mongefranco", text)


class ProjectRecords(unittest.TestCase):
    """The frozen production-boundary state is exactly what the READMEs promise."""

    def test_project_record(self) -> None:
        for name, spec in EXPECTED.items():
            project = load(FIXTURES / name / ".peerfoil" / "project.json")
            with self.subTest(fixture=name):
                self.assertEqual(project["project_id"], spec["project_id"])
                self.assertEqual(project["pack"]["id"], spec["pack"])
                self.assertEqual(project["assurance"], "guided")
                self.assertEqual(project["workflow"]["state"], "produce")
                self.assertEqual(project["revisions"],
                                 {"architecture": 1, "quality": 1, "plan": 2})
                seats = list(project["settings"]["roles"].values()) + \
                    project["settings"]["phase_reviewers"]
                for seat in seats:
                    self.assertNotEqual(seat["model"], "default",
                                        "seat models must give known lineage")

    def test_plan_change_and_snapshot(self) -> None:
        for name in EXPECTED:
            peerfoil = FIXTURES / name / ".peerfoil"
            plan = load(peerfoil / "plan.json")
            snapshot = load(peerfoil / "plans" / "plan-1.json")
            with self.subTest(fixture=name):
                self.assertEqual(plan["plan_revision"], 2)
                self.assertEqual(plan["status"], "accepted")
                self.assertEqual(len(plan["changes"]), 1)
                change = plan["changes"][0]
                self.assertEqual(change["id"], "cr-0001")
                self.assertEqual(change["prior_revision"], 1)
                self.assertEqual(change["placement"], "current-stage")
                self.assertEqual(change["acceptance"], "reviewed")
                self.assertEqual(change["affected_tasks"], ["tk-002"])
                self.assertEqual(change["retained_tasks"], ["tk-001"])
                self.assertEqual(snapshot["plan_revision"], 1)
                self.assertEqual(snapshot["status"], "accepted")
                tasks = {t["id"] for ph in snapshot["phases"]
                         for st in ph["stages"] for t in st["tasks"]}
                self.assertNotIn("tk-002", tasks,
                                 "the revision 1 snapshot predates cr-0001")

    def test_seeded_review_defect(self) -> None:
        """Requirement r-0002 is the seeded defect: recorded, violated, uncovered."""
        for name in EXPECTED:
            plan = load(FIXTURES / name / ".peerfoil" / "plan.json")
            with self.subTest(fixture=name):
                rows = {r["id"]: r for r in plan["requirements"]}
                self.assertIn("r-0002", rows)
                self.assertEqual(rows["r-0002"]["tasks"], [],
                                 "r-0002 must stay uncovered so review can find it")

    def test_history(self) -> None:
        for name, spec in EXPECTED.items():
            lines = (FIXTURES / name / ".peerfoil" / "history.jsonl").read_text(
                encoding="utf-8").splitlines()
            records = [json.loads(line) for line in lines if line.strip()]
            with self.subTest(fixture=name):
                self.assertEqual([r["transition_id"] for r in records],
                                 [f"tr-{n:04d}" for n in range(1, 6)])
                self.assertEqual([r["to_state"] for r in records],
                                 ["define", "architect", "plan", "produce", "produce"])
                self.assertEqual(records[0]["from_state"], None)
                times = [r["at"] for r in records]
                self.assertEqual(times, sorted(times), "timestamps must not go backwards")
                for record in records:
                    self.assertEqual(record["project_id"], spec["project_id"])

    def test_seeded_content_markers(self) -> None:
        for name, spec in EXPECTED.items():
            if not spec["seeded_text"]:
                continue
            rel, markers = spec["seeded_text"]
            text = (FIXTURES / name / rel).read_text(encoding="utf-8")
            for marker in markers:
                with self.subTest(fixture=name, marker=marker):
                    self.assertIn(marker, text,
                                  f"{name}/{rel} lost its seeded defect")


if __name__ == "__main__":
    unittest.main()
