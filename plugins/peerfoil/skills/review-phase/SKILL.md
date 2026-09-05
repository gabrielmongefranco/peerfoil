---
name: review-phase
description: This skill should be used when the user runs /peerfoil:review-phase or asks for the independent Claude and Codex review of a completed PeerFoil phase. In this build the review step is not yet available; the skill explains what the review will do, where its records will live, and stops without changing any file.
license: GPL-3.0-or-later
disable-model-invocation: true
allowed-tools: Read, Glob, Grep, Bash(git rev-parse *), PowerShell(git rev-parse *)
---
<!--
This file is part of PeerFoil.
plugins/peerfoil/skills/review-phase/SKILL.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Boundary notice for phase review, which arrives in Phase 1, Stage 4.
Notes: Assurance is Guided. Nothing in this build may approve work.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# PeerFoil phase review (Guided)

Phase review is not yet available in this build of PeerFoil Skills. Do not improvise a
review, and do not approve or mark anything as reviewed.

## What to do now

1. Read `${CLAUDE_PLUGIN_ROOT}/references/workflow.md`, sections 1 and 7, and
   `${CLAUDE_PLUGIN_ROOT}/references/lineage.md`, section 4.
2. If `.peerfoil/project.json` exists, read `workflow.state` and report it.
3. Tell the user, in plain language:
   - The completed workflow freezes the phase's decisions, architecture, plan, change
     sets, deliverables, evidence, and known risks, then gives that same frozen material
     to one fresh Claude reviewer and one fresh Codex reviewer at medium effort.
   - Each reviewer works independently first. Findings are then compared, duplicates are
     combined without hiding disagreement, and a failed required check can never be
     dismissed by reviewer agreement.
   - No agent approves its own work, and primary approval comes from a different model
     family. The default limit is six passes per reviewer and the maximum is eight, with
     one pass kept for checking a repair. This release states those limits; it does not
     enforce them mechanically.
   - Review records will live under `.peerfoil/reviews/` using the review template.
4. Say that this capability is listed as "Not yet" in the workflow reference, section 7,
   and end with `Assurance: Guided`.

Change no file.
