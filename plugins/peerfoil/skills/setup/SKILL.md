---
name: setup
description: This skill should be used when the user runs /peerfoil:setup or asks whether their computer and repository are ready for PeerFoil. It checks Git, the repository's AGENTS.md, Claude Code, the Codex CLI and its login, the Codex MCP server registration in Claude Code, the personal or work profile, and the tools declared by the selected project pack, then reports each result honestly with one next step. It finds Codex and Claude Code on the PATH or inside IDE extension folders.
license: GPL-3.0-or-later
disable-model-invocation: true
allowed-tools: Read, Glob, Grep, Bash(git --version), Bash(git rev-parse *), Bash(git status *), Bash(codex --version), Bash(codex login status), Bash(claude --version), Bash(claude mcp list), PowerShell(git --version), PowerShell(git rev-parse *), PowerShell(git status *), PowerShell(codex --version), PowerShell(codex login status), PowerShell(claude --version), PowerShell(claude mcp list)
---
<!--
This file is part of PeerFoil.
plugins/peerfoil/skills/setup/SKILL.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Guides the PeerFoil setup check for prerequisites, Codex access, profile, and pack tools.
Notes: Assurance is Guided. This skill reports; it installs nothing and changes the user's Claude Code configuration only with confirmation.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# PeerFoil setup (Guided)

Check that this computer and repository are ready for the PeerFoil workflow. Report every
check honestly. Never report a pass for a check that did not run. Install nothing; give
the user one clear next step instead.

## Before the checks

Read `${CLAUDE_PLUGIN_ROOT}/references/workflow.md`, sections 1, 6, and 7, and
`${CLAUDE_PLUGIN_ROOT}/references/codex.md` in full, so the report uses the correct
assurance label, build boundary, and program locations.

Use these statuses only: **Pass**, **Fail**, **Not verified**, **Skipped**.

## Checks

Run the checks in this order. Run each command exactly as written, as its own command,
never assembled into a shell string with other text. When a program is not on the `PATH`,
find it with the `Glob` patterns in the Codex access reference, section 1, and run the
found file with `--version` as its own command.

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
   `claude --version`; when `claude` is not on the `PATH`, find it as the Codex access
   reference describes. Report Pass with the version, or Not verified with the note that
   Claude Code is running but its version could not be read. The tested version is
   2.1.260.
5. **Codex CLI.** Run `codex --version`; when `codex` is not on the `PATH`, find it in
   the IDE extension folders as the Codex access reference describes. Report Pass with
   the version and where it was found in general terms ("on the PATH" or "in the VS Code
   Codex extension"), never the path itself. Report Fail when nothing is found, with the
   next step: install the Codex CLI or the Codex IDE extension, then run
   `/peerfoil:setup` again. The tested version is 0.153.0.
6. **Codex login.** Run `codex login status` with the same program. Pass when it reports
   a login. Fail otherwise, with the next step: run `codex login`, then `/peerfoil:setup`
   again. Never read or record credentials. Then read the `model` line of the user's
   Codex configuration file (`~/.codex/config.toml`, or `%USERPROFILE%\.codex\config.toml`
   on Windows), show the identifier, and, when a project record exists and the user
   confirms, record it as the `model` of every `codex-cli` seat in `settings`, so that
   reviewer lineage derives from a known identifier rather than `default`. Read nothing
   else from that file.
7. **Codex MCP server.** Check whether the `mcp__codex__codex` tool is available in this
   session.
   - If it is, report Pass. When the `MCP_TOOL_TIMEOUT` environment variable is not
     set, add the note that PeerFoil's ten-minute limit for a Codex review or task is
     `MCP_TOOL_TIMEOUT=600000`, which the user may set for Claude Code.
   - If it is not and Codex was found, report Not verified and offer to register it.
     Show the exact command from the Codex access reference, section 2, with the found
     path, and run it only after the user confirms with `AskUserQuestion`. Then tell the
     user to restart Claude Code and run `/peerfoil:setup` again. Note that until then
     PeerFoil uses the `codex exec` fallback, which works but is less smooth.
   - If Codex was not found, report Skipped.
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
  check names `repository`, `repository-instructions`, `git`, `claude-code`,
  `codex-cli`, `codex-login`, `codex-mcp`, `profile`, and one entry per pack tool by tool
  name. Map the statuses to `pass`, `fail`, `not-verified`, and `skipped`. Keep each
  `detail` under 300 characters and record versions only, never paths, user names, or
  tokens. Set `profile` if the user chose it now. Set `updated_at`.
- If no project record exists, write nothing. Setup never creates files.
- Never write outside the repository root. The only change outside it is the MCP
  registration the user confirmed in check 7.

## Report

Write the report in this shape:

```text
PeerFoil setup — Assurance: Guided

| Check | Status | Detail | Next step |
|---|---|---|---|
| ... | ... | ... | ... |

Ready for the decision interview: yes | no
Ready for independent review and production: yes | no | yes, using the codex exec fallback
Next action: ...
```

- The decision interview needs the repository, Git, and a profile.
- Independent review and production also need the Codex CLI and its login. With the MCP
  server registered the answer is "yes"; with only the CLI it is "yes, using the codex
  exec fallback".
- Say "no" whenever any required check is Fail or Not verified. Do not round up.
- Next action is `/peerfoil:start <idea>` when no project record exists and the
  interview prerequisites pass, `/peerfoil:resume` when a project record exists, or the
  first failed check's next step otherwise.
- End with one sentence: this release checks prerequisites but does not enforce them.
