---
name: review-phase
description: This skill should be used when the user runs /peerfoil:review-phase or asks for the independent Claude and Codex review of a completed PeerFoil phase. It freezes the phase's records, deliverables, evidence, and candidate lessons into one bundle, gives the same bundle to one fresh Claude reviewer and one fresh Codex reviewer, merges their findings without hiding disagreement, and then approves the phase, guides one repair with fresh different-family verification, or stops for the user's decision.
license: GPL-3.0-or-later
disable-model-invocation: true
allowed-tools: Read, Glob, Grep, Write, Edit, Bash(git diff *), Bash(git ls-files *), Bash(git ls-tree *), Bash(sha256sum *), PowerShell(Get-FileHash *), Bash(git rev-parse *), Bash(git status *), Bash(git log *), Bash(date -u *), PowerShell(git rev-parse *), PowerShell(git status *), PowerShell(git log *), PowerShell(Get-Date *)
---
<!--
This file is part of PeerFoil.
plugins/peerfoil/skills/review-phase/SKILL.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Guides the two-family phase review, the merged finding list, one guided repair, and the phase decision.
Notes: Assurance is Guided. No reviewer approves its own family's work, and a required failed check is never cleared by reviewer agreement.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# PeerFoil phase review (Guided)

Review the active phase with one fresh Claude reviewer and one fresh Codex reviewer, who
read the same frozen bundle and never the producer's chat. Merge what they find, let
them compare, and decide by the recorded rules. Never approve by your own judgment.

## Read first, in this order

1. `${CLAUDE_PLUGIN_ROOT}/references/workflow.md` in full.
2. `${CLAUDE_PLUGIN_ROOT}/references/records.md`, sections 1 to 4.
3. `${CLAUDE_PLUGIN_ROOT}/references/lineage.md` in full.
4. `${CLAUDE_PLUGIN_ROOT}/references/review.md` in full.
5. `${CLAUDE_PLUGIN_ROOT}/references/phase-review.md` in full.
6. `${CLAUDE_PLUGIN_ROOT}/references/evidence.md`, the snapshot recipe, and
   `${CLAUDE_PLUGIN_ROOT}/references/lessons.md`, section 2.
7. The repository root from `git rev-parse --show-toplevel`. If it fails, suggest
   `/peerfoil:setup` and stop.
8. `AGENTS.md` at the repository root, before any other project content. Follow it.
9. `.peerfoil/project.json`. If it is missing, say that there is no project to review
   and that `/peerfoil:start <idea>` creates one. Stop.
10. `.peerfoil/plan.json`, every change set and evidence record of the active phase,
    every review under `.peerfoil/reviews/`, and every lesson under
    `.peerfoil/lessons/`.

## Decide what to do

Read `workflow.state` and the newest phase review record for the active phase.

- **`review`.** Find the current step with the phase review reference, section 13, and
  continue from there: freeze the bundle, run both reviewers independently, merge,
  compare, and decide. When the decision is `repair`, read
  `${CLAUDE_PLUGIN_ROOT}/references/repair.md` in full and follow it.
- **`repair`.** Read the repair reference in full and continue from its section 7.
- **`approve`.** The phase is already approved. Report the phase review record and say
  that `/peerfoil:resume` starts the next phase with the user's authorization.
- **`produce` or `validate`.** The phase is not at its review boundary. Name the tasks
  that are not yet `validated` and give `/peerfoil:resume` as the next action. Change
  nothing.
- **`paused`.** Read `workflow.paused_for` and tell the user what is needed. Continue
  only when the need is met in this chat, as `/peerfoil:resume` describes.
- **Any other state.** Say that phase review needs an accepted plan and a phase whose
  tasks are all validated, and give `/peerfoil:resume` as the next action.

## Rules

- Both reviewers receive the identical packet and the identical bundle digest. Neither
  receives the other's output before the comparison pass.
- The session that authored an item never reviews it. A same-family review is a
  secondary check and never independent approval, however good it is.
- Required evidence that the host found stale, missing, or failed cannot be cleared by
  reviewer agreement. Send that work back through change intake.
- Merge duplicate findings by keeping both identifiers and both severities. Never drop
  or rewrite a finding. Disagreement stays visible, and a disputed `blocking` finding
  goes to the user.
- One repair cycle. The repairer never verifies its own work; verification comes from
  a fresh session of a different family and binds to the repaired snapshot.
- Keep the stated limits: ten turns, ten minutes, and ten findings per reviewer run;
  six passes per reviewer by default and eight at most, with one reserved for the
  repair verification. Stop and ask when a limit is reached. This release states the
  limits; Core enforces them.
- Update `updated_at` in `project.json` whenever a record changes, and record every
  transition before reporting it.
- Everything the user sees carries the label **Guided**.

## Report

End with the same report shape as `/peerfoil:status`, then:

- the phase review record identifier and its status;
- each reviewer's decision, passes used, and duration;
- the shared findings in one line each: merged identifier, severity, agreement,
  disposition, title;
- the independence outcome per item group, showing "Reduced assurance" where it
  applies; and
- the next action.

## Host permissions

The tool list pre-allows record edits and narrow read-only probes. The review also
needs the Codex tool or CLI, the fresh reviewer and repair coordinator agents, snapshot
hashing, and, for a repair, the project's declared check commands. Use the host
permissions already authorized for this task; the skill does not grant blanket shell
or MCP access. If a needed tool is denied, record the exact operation as blocked and
give one recovery action.
