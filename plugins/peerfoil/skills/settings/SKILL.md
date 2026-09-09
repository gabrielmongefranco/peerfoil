---
name: settings
description: This skill should be used when the user runs /peerfoil:settings or asks to view or change PeerFoil's Advanced settings, such as which tool, model, or effort a role uses, or the review pass limits. It shows the accepted settings from .peerfoil/project.json, validates a requested change against the allowed values, and writes the result back.
argument-hint: "[what to view or change]"
license: GPL-3.0-or-later
disable-model-invocation: true
allowed-tools: Read, Glob, Grep, Bash(git rev-parse *), Bash(date -u *), PowerShell(git rev-parse *), PowerShell(Get-Date *)
---
<!--
This file is part of PeerFoil.
plugins/peerfoil/skills/settings/SKILL.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Guides viewing and changing PeerFoil Advanced settings in project.json.
Notes: Assurance is Guided. Settings cannot remove the different-family review rule.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# PeerFoil Advanced settings (Guided)

Request from the user: $ARGUMENTS

Advanced settings hold the controls that normal use does not need: which tool, model, and
effort each role uses, and the review limits. They live in `.peerfoil/project.json` under
`settings`, so they stay with the project and its history.

## Read first

1. `${CLAUDE_PLUGIN_ROOT}/references/records.md`, the Project record in section 4, for
   the allowed values.
2. `${CLAUDE_PLUGIN_ROOT}/references/lineage.md`, sections 4 and 5.
3. The repository root from `git rev-parse --show-toplevel`. If it fails, suggest
   `/peerfoil:setup` and stop.
4. `.peerfoil/project.json`. If it is missing, show the defaults from
   `${CLAUDE_PLUGIN_ROOT}/templates/project.json` and explain that `/peerfoil:start`
   creates the project record that holds them. Stop.

## Show the settings

Present two tables in plain language:

```text
Role seats
| Role | Tool | Model | Effort |
|---|---|---|---|
| evaluator | claude-code | default | medium |
| ...

Review limits
| Setting | Value | Allowed |
|---|---|---|
| default_passes | 6 | 1 to 8 |
| ...
```

List `phase_reviewers` as two rows named "phase reviewer 1" and "phase reviewer 2".

## Change a setting

When the user asks for a change:

1. Restate the change in one sentence and confirm it.
2. Validate it:
   - `tool` is `claude-code` or `codex-cli`;
   - `model` is a non-empty model identifier or `default`;
   - `effort` is `low`, `medium`, `high`, or `xhigh`;
   - the two phase reviewers use different tools;
   - `default_passes` and `max_passes` are 1 to 8 and `default_passes` is not above
     `max_passes`;
   - `repair_selection_passes` and `repair_selection_max_passes` are 1 to 4 with the same
     ordering rule; and
   - `repair_cycles` is 0 or 1.
3. Warn before accepting these changes, then accept them only if the user confirms:
   - any seat raised to `xhigh`, or a seat other than the architect raised to `high`,
     which makes that role's step noticeably slower;
   - `producer` effort set to `low`, which the method allows only for small, reversible,
     low-risk work; refuse `low` for `repair_producer`; and
   - a change that leaves any authoring role and its normal reviewer on the same tool,
     which reduces assurance for that role.
4. Refuse a change that cannot be expressed in the allowed values, and say why.
5. Write `project.json` with the change and a new `updated_at`. Change nothing else.

Model routing, effort, and review limits are guided in this release. PeerFoil records the
values; it does not enforce them mechanically.
