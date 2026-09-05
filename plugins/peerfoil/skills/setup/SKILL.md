---
name: setup
description: This skill should be used when the user runs /peerfoil:setup or asks whether their computer and repository are ready for PeerFoil. It checks Git, the repository's AGENTS.md, Claude Code, Node.js, Codex CLI, the official Codex plugin, the personal or work profile, and the tools declared by the selected project pack, then reports each result honestly with one next step.
license: GPL-3.0-or-later
disable-model-invocation: true
allowed-tools: Read, Glob, Grep, Bash(git --version), Bash(git rev-parse *), Bash(git status *), Bash(node --version), Bash(codex --version), Bash(claude --version), PowerShell(git --version), PowerShell(git rev-parse *), PowerShell(git status *), PowerShell(node --version), PowerShell(codex --version), PowerShell(claude --version)
---
<!--
This file is part of PeerFoil.
plugins/peerfoil/skills/setup/SKILL.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Guides the PeerFoil setup check for prerequisites, profile, and pack tools.
Notes: Assurance is Guided. This skill reports; it does not install anything.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# PeerFoil setup (Guided)

Check that this computer and repository are ready for the PeerFoil workflow. Report every
check honestly. Never report a pass for a check that did not run. Install nothing; give
the user one clear next step instead.

## Before the checks

Read `${CLAUDE_PLUGIN_ROOT}/references/workflow.md`, sections 1, 6, and 7, so the report
uses the correct assurance label and build boundary.

Use these statuses only: **Pass**, **Fail**, **Not verified**, **Skipped**.

## Checks

Run the checks in this order. Run each command exactly as written, as its own command,
never assembled into a shell string with other text.

1. **Repository.** Run `git rev-parse --show-toplevel`. If it fails, the status is Fail
   and the next step is: open the project folder that contains the Git repository, or run
   `git init` there, then run `/peerfoil:setup` again. Stop after writing the report.
   If it succeeds, also run `git status --porcelain --branch` and note the branch and
   whether the working tree is clean. A dirty tree is a note, not a failure.
2. **Repository instructions.** Look for `AGENTS.md` at the repository root. If it exists,
   read the whole file now and report Pass with the note "read". If it does not exist,
   report Not verified with the note that the project has no repository instructions.
   PeerFoil never creates, edits, or replaces `AGENTS.md`.
3. **Git.** Run `git --version`. Pass when it prints a version.
4. **Claude Code.** This skill runs inside Claude Code, so the plugin is loaded. Run
   `claude --version`. If the command is not found on the shell path, report Not verified
   with the note that Claude Code is running but its version could not be read from the
   shell. The tested version is 2.1.260.
5. **Node.js.** Run `node --version`. The official Codex plugin requires 18.18 or later.
   Report Fail when the command is missing or older, with the next step: install Node.js
   18.18 or later from https://nodejs.org/ and run `/peerfoil:setup` again.
6. **Codex CLI.** Run `codex --version`. Report Fail when it is missing, with the next
   step: run `npm install -g @openai/codex`, then `codex login`, then `/peerfoil:setup`
   again.
7. **Codex plugin.** Check whether a skill named `codex:setup` is available in this
   session.
   - If it is available, run it, or ask the user to run `/codex:setup` and paste the
     result. Report Pass only when that result says Codex is installed and signed in.
   - If it is not available, report Fail with this next step, exactly:

     ```text
     /plugin marketplace add openai/codex-plugin-cc
     /plugin install codex@openai-codex
     /reload-plugins
     /codex:setup
     ```

     Then run `/peerfoil:setup` again.
8. **Profile.** If `.peerfoil/project.json` exists and has a `profile`, confirm it with
   the user. Otherwise ask whether this project is `personal` or `work`, using
   `AskUserQuestion`. Explain in one sentence that both profiles follow the repository's
   own `AGENTS.md` and that PeerFoil never edits it. Report Pass with the chosen profile.
9. **Pack tools.** If `.peerfoil/project.json` exists, read its `pack.id`, then read
   `${CLAUDE_PLUGIN_ROOT}/packs/<pack id>/pack.json`.
   - Run every entry in `tools[]` using its `command` array exactly. A missing required
     tool is Fail; a missing optional tool is Not verified.
   - For each entry in `project_tool_hints[]`, check whether a file matching `marker`
     exists at the repository root. When it does, run the hint's `command` array. Report
     Pass or Not verified with the hint's `purpose` in the detail.
   - If no project record exists, report Skipped with the note that `/peerfoil:start`
     chooses the pack and setup re-checks pack tools afterwards.

## Record the result

- If `.peerfoil/project.json` exists, update its `environment` field with `checked_at`
  (UTC, ISO 8601, second precision) and one `checks[]` entry per check above. Use the
  check names `repository`, `repository-instructions`, `git`, `claude-code`, `node`,
  `codex-cli`, `codex-plugin`, `profile`, and one entry per pack tool by tool name. Map
  the statuses to `pass`, `fail`, `not-verified`, and `skipped`. Keep each `detail` under
  300 characters and record versions only, never paths, user names, or tokens. Set
  `profile` if the user chose it now. Set `updated_at`.
- If no project record exists, write nothing. Setup never creates files.
- Never write outside the repository root.

## Report

Write the report in this shape:

```text
PeerFoil setup — Assurance: Guided

| Check | Status | Detail | Next step |
|---|---|---|---|
| ... | ... | ... | ... |

Ready for the decision interview: yes | no
Ready for production and review: yes | no
Next action: ...
```

- The decision interview needs the repository, Git, and a profile.
- Production and review also need Node.js, Codex CLI, and the Codex plugin.
- Say "no" whenever any required check is Fail or Not verified. Do not round up.
- Next action is `/peerfoil:start <idea>` when no project record exists and the
  interview prerequisites pass, `/peerfoil:resume` when a project record exists, or the
  first failed check's next step otherwise.
- End with one sentence: this release checks prerequisites but does not enforce them.
