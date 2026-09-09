---
name: status
description: This skill should be used when the user runs /peerfoil:status or asks where a PeerFoil project stands, what is blocking it, or what to do next. It reads the accepted files under .peerfoil and reports the assurance level, state, architecture and plan status, quality state, blocker, pending decisions, and next action in plain language without calling a model or changing any file.
license: GPL-3.0-or-later
allowed-tools: Read, Glob, Grep, Bash(git diff *), Bash(git ls-files *), Bash(sha256sum *), PowerShell(Get-FileHash *), Bash(git rev-parse *), Bash(git status *), Bash(git log *), PowerShell(git rev-parse *), PowerShell(git status *), PowerShell(git log *)
---
<!--
This file is part of PeerFoil.
plugins/peerfoil/skills/status/SKILL.md
Author(s): Gabriel Mongefranco; OpenAI Codex.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Guides the read-only PeerFoil status report.
Notes: Assurance is Guided. Status never writes a file and never reports Enforced.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# PeerFoil status (Guided)

Report where the project stands using only the files under `.peerfoil/` and the Git
state. Change nothing. Do not reconstruct state from this chat.

## Read

1. `${CLAUDE_PLUGIN_ROOT}/references/workflow.md`, sections 1, 3, 4, and 7.
2. The repository root from `git rev-parse --show-toplevel`. If it fails, say that the
   current folder is not inside a Git repository and suggest `/peerfoil:setup`. Stop.
3. Read root and applicable nested `AGENTS.md` before other project content.
   Then read `.peerfoil/project.json`. If it is missing, say that this repository has no PeerFoil
   project yet and that `/peerfoil:start <idea>` creates one. Stop.
4. `.peerfoil/decisions.md` and the last line of `.peerfoil/history.jsonl`.
5. `.peerfoil/architecture.md`, `.peerfoil/quality.md`, and the full
   `.peerfoil/plan.json` (including task revisions and change entries), and, under `.peerfoil/reviews/`, the latest review of kind
   `architecture` and the latest of kind `plan`, including their findings'
   dispositions, and the newest phase review record `pr-NNNN.md` for the active
   phase with its shared findings, when they exist.
5a. `.peerfoil/lessons/`, when it exists, for the counts of candidate, verified, and
   promoted lessons and the active hints.
6. Active change sets, plan snapshots, and linked evidence. Follow
   `${CLAUDE_PLUGIN_ROOT}/references/evidence.md` to compare current input hashes using
   read-only host commands. If hashing is unavailable, report evidence Not verified.
7. `git status --porcelain --branch` for the branch name and whether the tree is clean.

## Check the project record

Confirm that `project.json` has `record_type` `project`, `schema_version` `1`, and the
fields `project_id`, `pack`, `profile`, `assurance`, `workflow`, and `revisions`. If a
field is missing or `assurance` is not `guided`, report the problem as the blocker, name
the field, and suggest restoring the file from Git history. Continue with what can be
read.

## Report

Use this shape and plain language:

```text
PeerFoil status — Assurance: Guided

Project:   <name> (<project_id>) · Pack: <pack id> · Profile: <profile>
State:     <state> — <one-line meaning from the workflow reference>
Phase / stage / task: <ids and titles, or "none yet">
Architecture: <revision and status, and the latest review's decision and independence, or "none yet">
Plan:      <revision and status, and the latest review's decision and independence, or "none yet">
Quality:   <Quality Contract counts of required, recommended, and not-applicable items, and the evidence summary, or "No Quality Contract yet">
Review:    <phase review record, its status, passes used per reviewer, open blocking and major findings, independence, or "none yet">
Lessons:   <candidate, verified, and promoted counts and active hints, or "none">
Decisions: <answered> answered · <assumed> assumed · <open> open
Blocker:   <blocker, or "none">
Needs you: <what the user must decide or do, or "nothing">
Next:      <one command or action>
Last step: <transition id> at <time> — <summary>
Repository: <branch>, <clean | N changed files>
```

Rules:

- Always print `Guided`. Never print `Enforced` for this release, even if a file says so.
- When a review's independence is `secondary` or `reduced`, print "Reduced assurance"
  next to that artifact.
- In the `define` state, `Next` is `/peerfoil:resume` whether decisions are open or the
  interview is complete; say which.
- In the `architect` and `plan` states, name the current sub-step from the draft's
  status and the latest review: writing the draft, awaiting review, revising after
  findings, awaiting the user's acceptance, or approved. `Next` is `/peerfoil:resume`
  when the plan is accepted and production gates pass. Otherwise name the unfinished
  review, acceptance, or candidate-change step.
- In production and validation, name the active task and report required evidence as
  passed, failed, missing, stale, or Not verified. A producer claim is not a pass. Task
  `validated` means checks passed, not independently approved. A stale task tied to an
  old plan cannot be reported as current; retained work needs the change-entry chain
  and unchanged input hashes. At the phase boundary, `Next` is `/peerfoil:review-phase`.
- In `review` and `repair`, name the current step from the phase review reference,
  section 13, or the repair reference, section 7, and show "Reduced assurance" for
  any item without an eligible primary reviewer. A disputed or unresolved finding
  waiting for the user is the blocker. In `approve`, say the phase is approved, name
  the next phase, and give `/peerfoil:resume` as `Next`.
- A pending capture means the producer may still be running; next is safe reconciliation
  through `/peerfoil:resume`, never another writer. Do not call any model for status.
- `Blocker` names a revision/hash mismatch, missing capture or evidence, an `open`
  decision, an open `blocking` finding, or the `paused_for`
  reason when one exists.
- For anything the workflow reference leaves to PeerFoil Core, say so instead of
  inventing a next step.
- Keep the report under thirty lines. Do not paste file contents.
