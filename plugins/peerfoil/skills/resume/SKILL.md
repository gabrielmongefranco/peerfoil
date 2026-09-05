---
name: resume
description: This skill should be used when the user runs /peerfoil:resume or asks to continue a PeerFoil project in a new chat. It reads the accepted files under .peerfoil instead of any earlier conversation, works out the current state and sub-step, and continues the next allowed step, which is the decision interview, the architecture and its review, or the plan and its review.
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
Notes: Assurance is Guided. The decision interview, architecture, and plan can be resumed in this build; production cannot. See references/workflow.md section 7.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# PeerFoil resume (Guided)

Continue the project from its files. Never rely on an earlier chat; the repository is the
handoff.

## Read first, in this order

1. `${CLAUDE_PLUGIN_ROOT}/references/workflow.md` in full.
2. `${CLAUDE_PLUGIN_ROOT}/references/records.md`, sections 1 to 4.
3. `${CLAUDE_PLUGIN_ROOT}/references/lineage.md`, sections 3 and 4.
4. The repository root from `git rev-parse --show-toplevel`. If it fails, suggest
   `/peerfoil:setup` and stop.
5. `AGENTS.md` at the repository root, before any other project content. Follow it.
6. `.peerfoil/project.json`. If it is missing, say that there is nothing to resume and
   that `/peerfoil:start <idea>` creates a project. Stop.
7. `.peerfoil/decisions.md` and the last line of `.peerfoil/history.jsonl`.
8. `.peerfoil/architecture.md`, `.peerfoil/quality.md`, `.peerfoil/plan.json`, and the
   highest-numbered file under `.peerfoil/reviews/`, when they exist.
9. `git status --porcelain --branch`. If files under `.peerfoil/` have uncommitted
   changes, tell the user so they know the accepted records are not yet in Git history.

## Decide what to continue

Read `workflow.state` from `project.json`.

- **`define`.** Count the decisions in `decisions.md` by status. If any decision is
  `open`, continue the interview: follow `${CLAUDE_PLUGIN_ROOT}/skills/start/SKILL.md`,
  step 6, using the recorded decisions as the starting list and asking only the open
  items. If no decision is open, read
  `${CLAUDE_PLUGIN_ROOT}/references/architecture.md`,
  `${CLAUDE_PLUGIN_ROOT}/references/review.md`, and
  `${CLAUDE_PLUGIN_ROOT}/references/planning.md` in full, then follow the architecture
  reference from its section 1.
- **`architect`.** Read the same three references in full. Find the current sub-step
  with the architecture reference, section 2, and continue from there. After the user
  accepts the architecture, continue with the planning reference.
- **`plan`.** Read the same three references in full. Find the current sub-step with the
  planning reference, section 1, and continue from there. When `plan.json` is already
  `accepted`, report as in the planning reference, section 8: the plan is approved and
  production is not yet available in this build.
- **`paused`.** Read `workflow.paused_for` and tell the user what is needed. The state to
  return to is the `from_state` of the last `history.jsonl` line whose `to_state` is
  `paused`. When the need is met in this chat, for example the user answers the open
  decision or Codex is now available, set `workflow.state` back to that
  state, record the transition, and continue as for that state. Otherwise stop.
- **Any other state.** This build cannot continue that step. Say which state the project
  is in, what the completed workflow would do next, and that the capability is not yet
  available. Change nothing.

## Rules

- Write only the documented files under `.peerfoil/`, inside the repository root.
- Update `updated_at` in `project.json` whenever a record changes.
- Do not change `workflow.state` unless the workflow reference's transition condition is
  met and the next step is available in this build. Never enter `produce`.
- Never let the session that authored a draft review it, and never describe a guided
  step as enforced.

## Report

End with the same report shape as `/peerfoil:status`, then the next action.
