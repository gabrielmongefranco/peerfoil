#!/usr/bin/env python3
# This file is part of PeerFoil.
# tests/stage4_checks.py
# Author(s): Gabriel Mongefranco.
# Created: 2026-09-05
# Last Modified: 2026-09-05
# Summary: Checks the phase review, repair, and lesson contracts: identifiers, transition
#          references, review settings, templates, and the build boundary the skills state.
# Notes: These checks read files and schemas; they do not execute the guided workflow.
# Copyright © 2026 Gabriel Mongefranco
# SPDX-License-Identifier: GPL-3.0-or-later
"""Run with python tests/stage4_checks.py; uses only the Python standard library."""

import copy
import json
import re
import unittest

from static_checks import (
    PLUGIN_DIR,
    SCHEMA_DIR,
    TEMPLATE_DIR,
    SchemaValidator,
    review_settings_errors,
)

REFERENCES = PLUGIN_DIR / "references"
SKILLS = PLUGIN_DIR / "skills"


def read(path):
    return path.read_text(encoding="utf-8")


class TransitionReferences(unittest.TestCase):
    """History lines may point at phase reviews and lessons, and at nothing else new."""

    def setUp(self):
        self.validator = SchemaValidator(SCHEMA_DIR)
        line = read(TEMPLATE_DIR / "history.jsonl").splitlines()[0]
        self.transition = json.loads(line)

    def errors(self, transition):
        return self.validator.validate(transition, "transition.schema.json")

    def test_phase_review_and_lesson_references_validate(self):
        self.transition["refs"] = {
            "reviews": ["rv-0003", "rv-0004"],
            "phase_reviews": ["pr-0001"],
            "lessons": ["ls-0001"],
            "tasks": ["tk-002"],
        }
        self.assertEqual([], self.errors(self.transition))

    def test_malformed_phase_review_identifier_is_rejected(self):
        for bad in ("pr-1", "PR-0001", "rv-0001", "pr-0001/../x"):
            with self.subTest(identifier=bad):
                damaged = copy.deepcopy(self.transition)
                damaged["refs"] = {"phase_reviews": [bad]}
                self.assertTrue(any("phase_reviews" in error for error in self.errors(damaged)))

    def test_unknown_reference_key_is_rejected(self):
        damaged = copy.deepcopy(self.transition)
        damaged["refs"] = {"findings": ["fd-0001"]}
        self.assertTrue(any("findings" in error for error in self.errors(damaged)))

    def test_review_and_repair_states_are_valid_transitions(self):
        for from_state, to_state in (
            ("validate", "review"), ("review", "review"), ("review", "repair"),
            ("repair", "validate"), ("validate", "review"), ("review", "approve"),
            ("approve", "produce"), ("review", "paused"),
        ):
            with self.subTest(move=f"{from_state}->{to_state}"):
                moved = copy.deepcopy(self.transition)
                moved["from_state"] = from_state
                moved["to_state"] = to_state
                self.assertEqual([], self.errors(moved))


class ReviewSettings(unittest.TestCase):
    """The pass limits keep their ordering and ceilings."""

    def setUp(self):
        self.project = json.loads(read(TEMPLATE_DIR / "project.json"))

    def test_template_defaults_are_six_eight_three_four_one(self):
        review = self.project["settings"]["review"]
        self.assertEqual(
            (6, 8, 3, 4, 1),
            (
                review["default_passes"], review["max_passes"],
                review["repair_selection_passes"], review["repair_selection_max_passes"],
                review["repair_cycles"],
            ),
        )
        self.assertEqual([], review_settings_errors(self.project))

    def test_default_above_maximum_is_rejected(self):
        self.project["settings"]["review"]["default_passes"] = 8
        self.project["settings"]["review"]["max_passes"] = 6
        self.assertIn("default_passes must not exceed max_passes", review_settings_errors(self.project))
        self.project["settings"]["review"]["max_passes"] = 8
        self.project["settings"]["review"]["repair_selection_passes"] = 4
        self.project["settings"]["review"]["repair_selection_max_passes"] = 3
        self.assertIn(
            "repair_selection_passes must not exceed repair_selection_max_passes",
            review_settings_errors(self.project),
        )

    def test_phase_reviewers_must_use_different_tools(self):
        seats = self.project["settings"]["phase_reviewers"]
        seats[1]["tool"] = seats[0]["tool"]
        self.assertIn("phase_reviewers must use two different tools", review_settings_errors(self.project))

    def test_repair_producer_never_runs_at_low_effort(self):
        self.project["settings"]["roles"]["repair_producer"]["effort"] = "low"
        self.assertIn("repair_producer effort must not be low", review_settings_errors(self.project))


class Templates(unittest.TestCase):
    """The phase review and lesson records carry the fields the references name."""

    def test_phase_review_template_fields(self):
        text = read(TEMPLATE_DIR / "phase-review.md")
        self.assertTrue(text.startswith("<!-- PeerFoil project record: phase review."))
        for field in (
            "**Phase:**", "**Status:**", "**Round:**", "**Frozen at:**", "**Source revision:**",
            "**Bundle digest:**", "**Reviewer 1:**", "**Reviewer 2:**", "**Reviews:**",
            "**Decision:**", "**Independence:**",
        ):
            self.assertIn(field, text)
        for heading in (
            "## Bundle manifest", "## Required evidence", "## Open items", "## Tool versions",
            "## Shared findings", "## Repair", "## Decision",
        ):
            self.assertIn(heading, text)
        self.assertIn("| Merged | Reviewer 1 finding | Reviewer 2 finding |", text)

    def test_lesson_template_fields(self):
        text = read(TEMPLATE_DIR / "lesson.md")
        for field in (
            "**Status:**", "**Trigger:**", "**Scope:**", "**Rule:**", "**Evidence:**",
            "**Conflicts:**", "**Proposed destination:**", "**Verification:**",
            "**Promoted to:**", "**Expires at:**", "**Recorded by:**", "**Recorded at:**",
        ):
            self.assertIn(field, text)
        self.assertIn("candidate | verified | promoted | rejected", text)

    def test_review_template_names_phase_and_repair_kinds_and_item(self):
        text = read(TEMPLATE_DIR / "review.md")
        self.assertIn("architecture | plan | change | phase | repair", text)
        self.assertIn("**Item:**", text)
        self.assertIn("**Lens:**", text)


class BuildBoundary(unittest.TestCase):
    """The skills and references say phase review, repair, and lessons are available."""

    def test_workflow_reference_lists_stage_four_capabilities(self):
        text = read(REFERENCES / "workflow.md")
        self.assertNotIn("| Phase review, guided repair, and lessons | Not yet |", text)
        self.assertRegex(text, r"\| Phase review[^|]*\| Yes \|")
        self.assertRegex(text, r"\| [^|]*repair[^|]*\| Yes \|")
        self.assertRegex(text, r"\| [^|]*[Ll]essons[^|]*\| Yes \|")
        self.assertIn("| `approve` → `produce` |", text)

    def test_review_and_remember_skills_no_longer_stop(self):
        for name in ("review-phase", "remember"):
            with self.subTest(skill=name):
                text = read(SKILLS / name / "SKILL.md")
                self.assertNotIn("not yet available", text)
                self.assertNotIn("Change no file.", text)
                self.assertIn("Guided", text)

    def test_no_plugin_file_defers_phase_review_or_lessons(self):
        pattern = re.compile(r"(phase review|repair|lessons?)[^.\n]*Coming soon", re.IGNORECASE)
        for path in sorted(PLUGIN_DIR.rglob("*.md")):
            with self.subTest(path=path.name):
                self.assertIsNone(pattern.search(read(path)), path)

    def test_repair_coordinator_agent_is_bounded_and_read_only(self):
        text = read(PLUGIN_DIR / "agents" / "repair-coordinator.md")
        self.assertIn("effort: medium", text)
        self.assertIn("maxTurns: 6", text)
        self.assertIn("tools: Read, Glob, Grep", text)
        self.assertIn("Do not write, create, or edit any file.", text)

    def test_phase_review_reference_reserves_a_pass_and_keeps_disagreement(self):
        text = read(REFERENCES / "phase-review.md")
        self.assertIn("reserved for", text)
        self.assertIn("Never drop a finding", text)
        self.assertIn("Reviewer consensus cannot clear it", text)
        self.assertIn("Do not invent consensus", text)

    def test_repair_reference_blocks_self_verification_and_low_effort(self):
        text = read(REFERENCES / "repair.md")
        self.assertIn("never verify the repair", text)
        self.assertIn("never runs at low effort", text)

    def test_lessons_reference_never_edits_agents_md(self):
        text = read(REFERENCES / "lessons.md")
        self.assertIn("Never create, edit, or replace `AGENTS.md`", text)
        self.assertIn("Only a `verified` lesson is promoted, except to `hint`", text)


if __name__ == "__main__":
    unittest.main()
