---
name: status
description: This skill should be used when the user runs /peerfoil:status or asks where a PeerFoil project stands, what is blocking it, or what to do next. It reads the accepted files under .peerfoil and reports the assurance level, state, quality state, blocker, pending decisions, and next action in plain language without calling a model or changing any file.
license: GPL-3.0-or-later
allowed-tools: Read, Glob, Grep, Bash(git rev-parse *), Bash(git status *), Bash(git log *), PowerShell(git rev-parse *), PowerShell(git status *), PowerShell(git log *)
---
<!--
This file is part of PeerFoil.
plugins/peerfoil/skills/status/SKILL.md
Author(s): Gabriel Mongefranco.
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

1. `${CLAUDE_PLUGIN_ROOT}/references/workflow.md`, sections 1, 3, and 7.
2. The repository root from `git rev-parse --show-toplevel`. If it fails, say that the
   current folder is not inside a Git repository and suggest `/peerfoil:setup`. Stop.
3. `.peerfoil/project.json`. If it is missing, say that this repository has no PeerFoil
   project yet and that `/peerfoil:start <idea>` creates one. Stop.
4. `.peerfoil/decisions.md`, the last line of `.peerfoil/history.jsonl`, and
   `.peerfoil/plan.json` when it exists.
5. `git status --porcelain --branch` for the branch name and whether the tree is clean.

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
Quality:   <evidence summary, or "No evidence yet">
Decisions: <answered> answered · <assumed> assumed · <open> open
Blocker:   <blocker, or "none">
Needs you: <what the user must decide or do, or "nothing">
Next:      <one command or action>
Last step: <transition id> at <time> — <summary>
Repository: <branch>, <clean | N changed files>
```

Rules:

- Always print `Guided`. Never print `Enforced` for this release, even if a file says so.
- In the `define` state, `Next` is `/peerfoil:resume` when decisions are open. When every
  decision is answered or assumed, say that the decisions are complete and that
  architecture is not yet available in this build, as listed in the workflow reference,
  section 7.
- For any capability the workflow reference marks "Not yet", say so instead of inventing
  a next step.
- Keep the report under thirty lines. Do not paste file contents.
