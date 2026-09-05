---
name: remember
description: This skill should be used when the user runs /peerfoil:remember or asks PeerFoil to keep a lesson learned. In this build the lesson step is not yet available; the skill explains how candidate lessons will be recorded and reviewed, where they will live, and stops without changing any file.
argument-hint: "[the lesson]"
license: GPL-3.0-or-later
disable-model-invocation: true
allowed-tools: Read, Glob, Grep, Bash(git rev-parse *), PowerShell(git rev-parse *)
---
<!--
This file is part of PeerFoil.
plugins/peerfoil/skills/remember/SKILL.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Boundary notice for lessons, which arrive in Phase 1, Stage 4.
Notes: Assurance is Guided. A lesson never becomes project policy on a model's word alone.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# PeerFoil remember (Guided)

Lesson from the user: $ARGUMENTS

Recording lessons is not yet available in this build of PeerFoil Skills. Do not write a
lesson file, and do not change `AGENTS.md` or any other rule.

## What to do now

1. Read `${CLAUDE_PLUGIN_ROOT}/references/workflow.md`, section 7.
2. Tell the user, in plain language:
   - The completed workflow rewrites a lesson as a clear rule with a trigger and a scope,
     records the evidence behind it and any conflicts with existing rules, and stores it
     as a candidate under `.peerfoil/lessons/` using the lesson template.
   - A candidate is checked before it becomes durable guidance. Possible destinations are
     a decision, a test, a skill, a proposed `AGENTS.md` change for the user to accept, a
     pack rule, or a temporary hint with an expiry date.
   - Raw model memory never becomes project policy on its own.
3. Suggest that the user keep the lesson in their own notes until the capability
   arrives, and say that it is listed as "Not yet" in the workflow reference, section 7.
4. End with `Assurance: Guided`.

Change no file.
