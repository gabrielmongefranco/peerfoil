---
name: example-skill
description: One or two sentences saying what this skill does and when an agent should use it. Replace this with your own description.
---

<!--
This file is part of PeerFoil.
skills/skill-template.md
Author(s): Gabriel Mongefranco.
Created: 2026-10-21
Last Modified: 2026-10-21
Summary: Copyable starting point for a new reusable skill. Copy it to skills/<skill-name>/SKILL.md and replace every example value.
Notes: After copying, replace this header with your project's own name, path, author, dates, and license.
Copyright © 2026 Gabriel Mongefranco
Licensed under the GNU Free Documentation License v1.3 or later.
See <https://www.gnu.org/licenses/fdl-1.3.html>. See README for full license information.
-->

# Example Project

## Example skill

Replace the project title above and this opening paragraph. Say in two or three
sentences what the skill covers and what it cannot do. A skill supplements the
repository's `AGENTS.md`; it cannot weaken its security, privacy, accessibility,
licensing, testing, or authorization rules.

After copying this file into `skills/<skill-name>/SKILL.md`, update the frontmatter
`name` to match the folder name, and point the links below at `../../AGENTS.md` and
`../project-preferences/SKILL.md` so they resolve from the new location.

### When this applies

Name the tasks that should load this skill, and the tasks that should not.

### Workflow

1. State the first step as something the agent can do and verify.
2. Keep each step small. Link supporting files from a `references/` folder inside the
   skill instead of pasting long material here.
3. End with the check that shows the work is done.

### Constraints

List the limits that always apply: files the skill must not touch, commands it must
not run, and decisions that belong to the user.

### Verification

Say how to confirm the result: the command to run, the output to expect, or the
manual check a person performs.
