#!/usr/bin/env python3
# This file is part of PeerFoil.
# tests/stage3_checks.py
# Author(s): Gabriel Mongefranco; OpenAI Codex.
# Created: 2026-09-05
# Last Modified: 2026-09-05
# Summary: Checks plan-change record compatibility and rejection of malformed traceability.
# Notes: These schema checks do not execute or enforce the guided workflow.
# Copyright © 2026 Gabriel Mongefranco
# SPDX-License-Identifier: GPL-3.0-or-later
"""Run with python tests/stage3_checks.py; uses only the Python standard library."""

import copy
import json
import unittest

from static_checks import SCHEMA_DIR, TEMPLATE_DIR, SchemaValidator, plan_change_errors


class PlanChangeContracts(unittest.TestCase):
    """Exercise persisted change records that fresh sessions and Core must read."""

    def setUp(self):
        self.validator = SchemaValidator(SCHEMA_DIR)
        self.plan = json.loads((TEMPLATE_DIR / "plan.json").read_text(encoding="utf-8"))
        self.change = {
            "id": "cr-0001",
            "summary": "Document the command's Unicode output.",
            "placement": "current-stage",
            "recorded_at": "2026-09-05T12:00:00Z",
            "prior_revision": 1,
            "reason": "The output requirement affects the active task.",
            "affected_tasks": ["tk-001"],
            "retained_tasks": [],
            "evidence": ["ev-0001"],
            "reviews": ["rv-0002"],
            "acceptance": "pending",
        }
        self.plan["plan_revision"] = 2
        self.plan["changes"] = [self.change]

    def errors(self):
        return self.validator.validate(self.plan, "plan.schema.json")

    def test_existing_plan_without_extension_remains_readable(self):
        del self.plan["changes"]
        self.plan["schema_version"] = 1
        self.assertEqual([], self.errors())

    def test_new_fields_cannot_masquerade_as_version_one(self):
        self.plan["schema_version"] = 1
        self.assertTrue(any("changes" in error for error in self.errors()))

    def test_ambiguous_impact_and_revision_chain_are_rejected(self):
        self.assertEqual([], plan_change_errors(self.plan))
        self.change["retained_tasks"] = ["tk-001"]
        self.assertIn("cr-0001: affected_tasks and retained_tasks must be disjoint", plan_change_errors(self.plan))
        self.change["retained_tasks"] = []
        self.change["prior_revision"] = 2
        self.assertIn("cr-0001: prior_revision must precede plan_revision", plan_change_errors(self.plan))
        self.plan["changes"].append(copy.deepcopy(self.change))
        self.assertIn("duplicate change id cr-0001", plan_change_errors(self.plan))

    def test_all_five_placements_round_trip_with_crlf_and_unicode(self):
        for placement in ("current-stage", "later-stage", "later-phase", "backlog", "declined"):
            with self.subTest(placement=placement):
                self.change["placement"] = placement
                self.change["summary"] = "Keep café output in the user's guide."
                encoded = json.dumps(self.plan, ensure_ascii=False, indent=2).replace("\n", "\r\n")
                decoded = json.loads(encoded.encode("utf-8").decode("utf-8"))
                self.assertEqual(self.plan, decoded)
                self.assertEqual([], self.validator.validate(decoded, "plan.schema.json"))

    def test_missing_traceability_fields_are_rejected(self):
        for field in tuple(self.change):
            with self.subTest(field=field):
                damaged = copy.deepcopy(self.plan)
                del damaged["changes"][0][field]
                errors = self.validator.validate(damaged, "plan.schema.json")
                self.assertTrue(any(field in error for error in errors), errors)

    def test_unknown_fields_and_invalid_references_are_rejected(self):
        invalid = {
            "placement": "now",
            "acceptance": "self-approved",
            "prior_revision": 0,
            "affected_tasks": ["../tk-001"],
            "retained_tasks": [17],
            "evidence": ["rv-0001"],
            "reviews": ["ev-0001"],
            "reason": "",
            "provider_token": "synthetic-forbidden-field",
        }
        for field, value in invalid.items():
            with self.subTest(field=field):
                damaged = copy.deepcopy(self.plan)
                damaged["changes"][0][field] = value
                errors = self.validator.validate(damaged, "plan.schema.json")
                self.assertTrue(any(field in error for error in errors), errors)



if __name__ == "__main__":
    unittest.main()
