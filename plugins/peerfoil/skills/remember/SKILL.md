---
name: remember
description: This skill should be used when the user runs /peerfoil:remember or asks PeerFoil to keep a lesson learned. It rewrites the lesson as one clear rule with a trigger, scope, evidence, conflicts, and a proposed destination, records it as a candidate under .peerfoil/lessons, and promotes a verified lesson only with the user's approval and never by editing AGENTS.md.
argument-hint: "[the lesson]"
license: GPL-3.0-or-later
disable-model-invocation: true
allowed-tools: Read, Glob, Grep, Write, Edit, Bash(git rev-parse *), Bash(git status *), Bash(date -u *), PowerShell(git rev-parse *), PowerShell(git status *), PowerShell(Get-Date *)
---
<!--
This file is part of PeerFoil.
plugins/peerfoil/skills/remember/SKILL.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Guides recording a candidate lesson, checking its verification status, and promoting it with the user's approval.
Notes: Assurance is Guided. A lesson never becomes project policy on a model's word alone, and AGENTS.md is never edited.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# PeerFoil remember (Guided)

Lesson from the user: $ARGUMENTS

Keep something worth keeping. Rewrite it as a rule, record it as a candidate, and let a
fresh reviewer verify it before it becomes guidance.

## Read first

1. `${CLAUDE_PLUGIN_ROOT}/references/workflow.md`, sections 1, 6, and 8.
2. `${CLAUDE_PLUGIN_ROOT}/references/records.md`, sections 1 to 3 and the Lesson and
   Transition records in section 4.
3. `${CLAUDE_PLUGIN_ROOT}/references/lessons.md` in full.
4. The repository root from `git rev-parse --show-toplevel`. If it fails, suggest
   `/peerfoil:setup` and stop.
5. `AGENTS.md` at the repository root, before any other project content.
6. `.peerfoil/project.json`. If it is missing, say that lessons belong to a project and
   that `/peerfoil:start <idea>` creates one. Stop.
7. Every file under `.peerfoil/lessons/`, when the directory exists.

## When the request is a new lesson

Follow the lessons reference, section 1: rewrite the rule, trigger, and scope; name the
evidence; list conflicts; propose a destination; confirm with the user; write the
candidate from `${CLAUDE_PLUGIN_ROOT}/templates/lesson.md`; and append the transition.

## When the request names an existing lesson

- If the user asks to promote a `verified` lesson, or a `candidate` to `hint`, follow
  the lessons reference, section 3, after the user's approval.
- If the user asks to reject a lesson, set it `rejected` with the reason and append the
  transition.
- If the lesson is a `candidate` and the user asks whether it is verified, say that
  verification happens at the next phase review, name the phase review record if one
  exists, and change nothing.

## Rules

- Never create, edit, or replace `AGENTS.md`, a skill, a pack, or provider settings.
  Proposed rule text lives in the lesson file until a person applies it.
- Never store the raw chat, a credential, or personal data in a lesson.
- Do not verify the lesson yourself; the session that recorded it never verifies it.
- Update `updated_at` in `project.json` whenever a record changes.

## Report

Show the lesson identifier, status, rule, trigger, scope, evidence, conflicts,
destination, and, for a hint, its expiry; then what happens next: verification at the
next phase review, or the promotion recorded. End with `Assurance: Guided`.
