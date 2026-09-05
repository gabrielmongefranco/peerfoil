---
name: change
description: This skill should be used when the user runs /peerfoil:change or asks to add a request, requirement, or change to a PeerFoil project. It places requests and discovered work into the current stage, later stage, later phase, backlog, or declined, retaining plan revisions and affected evidence.
argument-hint: "[the request or change]"
license: GPL-3.0-or-later
disable-model-invocation: true
allowed-tools: Read, Glob, Grep, Write, Edit, Bash(git diff *), Bash(git ls-files *), Bash(git ls-tree *), Bash(sha256sum *), PowerShell(Get-FileHash *), Bash(git rev-parse *), Bash(git status *), Bash(git log *), Bash(date -u *), PowerShell(git rev-parse *), PowerShell(git status *), PowerShell(git log *), PowerShell(Get-Date *)
---
<!--
This file is part of PeerFoil.
plugins/peerfoil/skills/change/SKILL.md
Author(s): Gabriel Mongefranco; OpenAI Codex.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Guides decision intake and traceable plan revisions for changes and discovered work.
Notes: Assurance is Guided. Changes never authorize a second writer or a later phase.

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

## After decisions or planning

If no plan exists yet, record the request as a new decision, superseding any affected
answer. Return through the architecture review path in `/peerfoil:resume`; do not lose
the request in chat. An open consequential decision keeps production blocked.

When a plan exists, read `${CLAUDE_PLUGIN_ROOT}/references/changes.md` in full and follow
its compare, candidate, review, and acceptance steps. Keep one writer at a time. Report
the placement and reason, new or pending plan revision, affected and retained tasks,
current blocker, and next action. Every report says `Assurance: Guided`.

## Host permissions

The tool list pre-allows record edits and narrow read-only probes. Production also
needs the Codex tool or CLI, temporary-index Git operations, snapshot hashing, and the
project's declared check commands. Use the host permissions already authorized for this
task; the skill does not grant blanket shell or MCP access. If a needed tool is denied,
record the exact operation as blocked and give one recovery action. A non-interactive
host must supply task-scoped permissions before production can continue.
