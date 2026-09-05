---
name: resume
description: This skill should be used when the user runs /peerfoil:resume or asks to continue a PeerFoil project in a new chat. It reads the accepted files under .peerfoil instead of any earlier conversation, works out the current state, and continues the next allowed step, which in this build is the decision interview.
license: GPL-3.0-or-later
disable-model-invocation: true
allowed-tools: Read, Glob, Grep, Bash(git rev-parse *), Bash(git status *), Bash(git log *), Bash(date -u *), PowerShell(git rev-parse *), PowerShell(git status *), PowerShell(git log *), PowerShell(Get-Date *)
---
<!--
This file is part of PeerFoil.
plugins/peerfoil/skills/resume/SKILL.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Guides resuming a PeerFoil project from repository files in a fresh chat.
Notes: Assurance is Guided. Only the decision interview can be resumed in this build; see references/workflow.md section 7.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# PeerFoil resume (Guided)

Continue the project from its files. Never rely on an earlier chat; the repository is the
handoff.

## Read first, in this order

1. `${CLAUDE_PLUGIN_ROOT}/references/workflow.md` in full.
2. `${CLAUDE_PLUGIN_ROOT}/references/records.md`, sections 1 to 4.
3. The repository root from `git rev-parse --show-toplevel`. If it fails, suggest
   `/peerfoil:setup` and stop.
4. `AGENTS.md` at the repository root, before any other project content. Follow it.
5. `.peerfoil/project.json`. If it is missing, say that there is nothing to resume and
   that `/peerfoil:start <idea>` creates a project. Stop.
6. `.peerfoil/decisions.md` and the last line of `.peerfoil/history.jsonl`.
7. `git status --porcelain --branch`. If files under `.peerfoil/` have uncommitted
   changes, tell the user so they know the accepted records are not yet in Git history.

## Decide what to continue

Read `workflow.state` from `project.json`.

- **`define`.** Count the decisions in `decisions.md` by status. If any decision is
  `open`, continue the interview: follow `${CLAUDE_PLUGIN_ROOT}/skills/start/SKILL.md`,
  step 6, using the recorded decisions as the starting list and asking only the open
  items. If no decision is open, say that the decisions are complete and that the next
  step, architecture with a different-family review, is not yet available in this build,
  as listed in the workflow reference, section 7. Point to `/peerfoil:change <request>`
  as the way to add new information while the project is in this state.
- **`paused`.** Read `workflow.paused_for` and tell the user what is needed. When it
  concerns an open decision, continue as for `define`.
- **Any other state.** This build cannot continue that step. Say which state the project
  is in, what the completed workflow would do next, and that the capability is not yet
  available. Change nothing.

## Rules

- Write only the documented files under `.peerfoil/`, inside the repository root.
- Update `updated_at` in `project.json` whenever a record changes.
- Do not change `workflow.state` unless the workflow reference's transition condition is
  met and the next step is available in this build.
- Never describe a guided step as enforced.

## Report

End with the same report shape as `/peerfoil:status`, then the next action.
