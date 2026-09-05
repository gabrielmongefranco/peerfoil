<!--
This file is part of PeerFoil.
plugins/peerfoil/README.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Describes the PeerFoil plugin for Claude Code, what this build can do, and how to load it for testing.
Notes: This is a development build of PeerFoil Skills 0.1. See the repository README for the project overview and license.

Copyright © 2026 Gabriel Mongefranco

Permission is granted to copy, distribute and/or modify this document under the terms of
the GNU Free Documentation License, Version 1.3 or any later version published by the Free
Software Foundation; with no Invariant Sections, no Front-Cover Texts, and no Back-Cover
Texts. See <https://www.gnu.org/licenses/fdl-1.3.html>.
-->

# PeerFoil plugin for Claude Code

## The guided PeerFoil workflow as Claude Code skills

[Return to the PeerFoil README](../../README.md)

This plugin is PeerFoil Skills 0.1, a development build. It gives Claude Code a set of
`/peerfoil:*` commands that guide a project from an idea through important decisions,
architecture, planning, production, evidence, independent review, and lessons. The
assurance level of this release is **Guided**: the workflow depends on agents following
written instructions, and nothing here enforces a step mechanically.

## What this build can do

This build is the first of five Phase 1 stages. Today it can:

- check your prerequisites with `/peerfoil:setup`;
- start a project, choose a pack and profile, and create the project records with
  `/peerfoil:start`;
- run the decision interview with a fresh evaluator role;
- show the project's state with `/peerfoil:status`;
- continue the decision interview in a fresh chat with `/peerfoil:resume`; and
- view and change Advanced settings with `/peerfoil:settings`.

Architecture, planning, production with Codex, change placement, phase review, repair,
and lessons are **Coming soon** in later stages. The commands for them exist so that they
can explain the boundary honestly.

## Prerequisites

- Git.
- Claude Code. The tested version is 2.1.260.
- Node.js 18.18 or later, Codex CLI, and the official
  [Codex plugin for Claude Code](https://github.com/openai/codex-plugin-cc). These are
  needed for production and review; the decision interview works without them.
- Your project's own development tools.

## Load the plugin for testing

From a checkout of this repository, start Claude Code in the project you want to work on
and load the plugin from its directory:

```text
claude --plugin-dir <path to checkout>/plugins/peerfoil
```

The commands then appear as `/peerfoil:setup`, `/peerfoil:start`, and so on. Marketplace
installation instructions are **Coming soon** and will be published only after they pass
on Windows, macOS, and Linux.

## Commands

| Command | What it does |
|---|---|
| `/peerfoil:setup` | Checks Git, `AGENTS.md`, Claude Code, Node.js, Codex CLI, the Codex plugin, your profile, and pack tools |
| `/peerfoil:start <idea>` | Records the goal, pack, and profile, creates the project records, and runs the decision interview |
| `/peerfoil:change <request>` | Adds a request to the decision interview; plan placement is Coming soon |
| `/peerfoil:status` | Reports assurance, state, quality state, blocker, pending decisions, and next action |
| `/peerfoil:resume` | Continues the project from repository files in a fresh chat |
| `/peerfoil:review-phase` | Coming soon: independent Claude and Codex review of a completed phase |
| `/peerfoil:remember <lesson>` | Coming soon: candidate lessons that are checked before they become guidance |
| `/peerfoil:settings` | Views and changes Advanced settings: role seats and review limits |

## Files the plugin creates

The plugin writes only inside your repository, under `.peerfoil/`. This build creates
`project.json`, `decisions.md`, and `history.jsonl`. These files belong to you. They hold
decisions, workflow state, and small redacted transition records. They never hold
provider tokens, raw prompts, or full conversations.

## Privacy

The evaluator role receives a compact packet: the goal, pack, profile, a summary of your
repository rules and README, and observable repository facts. It does not receive your
chat history. Setup records tool versions only, never paths or user names.

## Layout

```text
plugins/peerfoil/
  .claude-plugin/plugin.json   plugin manifest
  skills/<command>/SKILL.md    one skill per command
  agents/evaluator.md          fresh evaluator role
  references/                  workflow, records, and lineage rules shared by skills
  packs/<pack>/pack.json       Software, Generic, and Documentation packs
  templates/                   starting points for the .peerfoil/ project files
  LICENSE                      GNU General Public License, version 3
```

## License

The plugin's skills, agents, references, packs, templates, and manifest are licensed
under the GNU General Public License, version 3 or later. Human-facing documentation in
the plugin uses the GNU Free Documentation License, version 1.3 or later. See the
repository [NOTICE](../../NOTICE) for the complete notices.

## Conclusion

This build proves the front of the workflow: prerequisites, a recorded goal, and a
decision interview that a fresh chat can resume. Later stages add the architecture,
plan, production, review, and lesson steps on the same records.

## Additional Resources

- [PeerFoil method](../../docs/PeerFoil-Method.md)
- [PeerFoil architecture](../../docs/architecture.md)
- [Phase 1 implementation plan](../../docs/plans/phase-1-skills.md)
- [Decision log](../../docs/decision-log.md)
- [Claude Code plugin reference](https://code.claude.com/docs/en/plugins-reference)

[Return to the PeerFoil README](../../README.md)

---

Copyright © 2026 Gabriel Mongefranco
