---
name: change
description: This skill should be used when the user runs /peerfoil:change or asks to add a request, requirement, or change to a PeerFoil project. In this build it can add the request to the decision interview while the project is still in the define state; placing a change into the current stage, a later stage, a later phase, or the backlog arrives in a later build.
argument-hint: "[the request or change]"
license: GPL-3.0-or-later
disable-model-invocation: true
allowed-tools: Read, Glob, Grep, Bash(git rev-parse *), Bash(git status *), Bash(date -u *), PowerShell(git rev-parse *), PowerShell(git status *), PowerShell(Get-Date *)
---
<!--
This file is part of PeerFoil.
plugins/peerfoil/skills/change/SKILL.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Guides change intake; in this build only the define-state path is available.
Notes: Assurance is Guided. Change placement into the plan arrives in Phase 1, Stage 3.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# PeerFoil change (Guided)

Request from the user: $ARGUMENTS

## Read first

1. `${CLAUDE_PLUGIN_ROOT}/references/workflow.md`, sections 3, 6, and 7.
2. The repository root from `git rev-parse --show-toplevel`. If it fails, suggest
   `/peerfoil:setup` and stop.
3. `AGENTS.md` at the repository root, before any other project content.
4. `.peerfoil/project.json`. If it is missing, say that there is no project to change and
   that `/peerfoil:start <idea>` creates one. Stop.

## While the project is in the `define` state

The request is new information for the decision interview.

1. Ask whether the request changes the project goal. If it does, rewrite the goal in two
   to four sentences, confirm it, and update `goal` and `updated_at` in `project.json`.
2. Continue the decision interview as described in
   `${CLAUDE_PLUGIN_ROOT}/skills/start/SKILL.md`, step 6, with the request added to the
   evaluator packet and the existing decisions included. Record new decisions with the
   next identifiers. Mark any earlier decision the request makes obsolete as
   `superseded`, naming the replacement.
3. Report what was added or superseded and the counts of answered, assumed, and open
   decisions. End with `Assurance: Guided`.

## In any other state

This build cannot place a change into a plan yet. Tell the user honestly:

- The completed workflow will decide whether the request belongs in the current stage, a
  later stage of the current phase, a later phase, or the backlog, or should be declined,
  and every placement creates a new plan revision.
- That capability is listed as "Not yet" in the workflow reference, section 7.
- While the project is in `architect` or `plan` and the draft is not yet accepted, the
  request can be raised when `/peerfoil:resume` asks the user to accept the architecture
  or approve the stage order: choose "Change something" and describe it. The author
  revises the draft and it is reviewed again.
- Until then, keep the request in your own notes or as a repository issue. Do not edit
  `.peerfoil/` files by hand to record it.

Change nothing in that case.
