<!--
Project:  PeerFoil  |  File: README.md
Authors:  Gabriel Mongefranco (@gabrielmongefranco)
Created:  2026-09-04  |  Modified: 2026-09-04
Summary:  Introduces PeerFoil, a coding-first multi-model delivery method and local orchestrator.
SPDX-License-Identifier: GFDL-1.3-or-later
-->

# PeerFoil

***Better vibe, higher-quality coding.***

## Description

PeerFoil is a coding-first, multi-model delivery method and planned local orchestrator for people building software mostly alone. It uses one high-capability AI model to help resolve product decisions and architecture, a fresh planning session to turn that architecture into phases and bounded tasks, and qualified production agents to implement the work.

The producing agent does not grade its own work. PeerFoil records authorship and normally assigns independent approval to a fresh model from another family. At every phase boundary, reviewers inspect the same frozen artifacts and evidence, reconcile findings within a strict pass limit, select a high-effort repair agent when necessary, and independently verify the repair.

Tests and tool output are evidence only when the controller or coordinating host actually runs them. Required failures cannot be voted away. Changes, TODOs, unsupported claims, deviations, skipped checks, and deferrals revise the plan instead of disappearing into chat history.

Software is the flagship use case, with architecture, correctness, reliability, security, privacy, accessibility, maintainability, documentation, licensing, and release checks selected according to risk. The orchestration core is artifact-neutral, allowing planned Documentation, Business Plan, and Research Report packs to use the same peer-review and evidence protocol without turning PeerFoil into a general project-management platform.

PeerFoil reuses Claude Code, Codex, Git, Agent Skills, MCP, project-native tools, and optional local-model runtimes. It is not another coding agent or hosted service. Accepted decisions, plans, reviews, evidence references, and lessons remain in ordinary version-controlled project files.

The product is intended for native Windows, macOS, and Linux use. It will require no paid non-LLM service, hosted control plane, or database server. Hosted models are supported, but qualified local models will be able to fill the same roles.

PeerFoil raises the quality floor for solo builders. It does not certify that software, documents, plans, or research are correct, secure, accessible, viable, or fit for a regulated purpose.

## Quick Start Guide

**Coming soon.** No public PeerFoil plugin, package, or binary has been released yet.

The first usable release is planned as a Claude Code plugin and portable Markdown skills. It will guide the complete decision, architecture, planning, production, validation, phase-review, repair, and resume workflow using the official Codex plugin. Installation instructions will be added only when the package and release locations exist.

The planned normal experience has five primary actions:

1. Start a project or change.
2. Add or revise a requirement.
3. Check status.
4. Resume interrupted work.
5. Preserve a reviewed lesson.

Provider selection, model effort, standards sources, review budgets, skills, MCP access, local-model endpoints, and cost controls will remain under Advanced settings.

## Status

**Pre-implementation design complete; releases coming soon.** PeerFoil is planned in three increments: a guided Skills release by Day 5 of implementation, an enforced Core Alpha by Day 13, and the complete 1.0 scope no later than Week 26. The first release is intentionally useful without a PeerFoil executable; Core then adds deterministic transitions, authoritative command evidence, recovery, and mechanically bounded review.

The documents below define the current project. Where they disagree, `PeerFoil-Method.md` is the product contract, `architecture.md` governs technical boundaries, and `implementation-plan.md` governs sequencing only where consistent with both.

| Document | Purpose |
|---|---|
| [PeerFoil Method](docs/PeerFoil-Method.md) | **Normative product proposal.** Workflow, roles, evidence, review, memory, project packs, constraints, and success criteria |
| [Architecture](docs/architecture.md) | Components, contracts, data model, trust boundaries, persistence, and principal flows |
| [Implementation plan](docs/implementation-plan.md) | High-level delivery phases, exit gates, dependencies, risks, and deferrals |

## Example Applications

**Coming soon.** Planned reference projects include:

- a small cross-platform software project demonstrating architecture, implementation, tests, and dual-family phase review;
- a Markdown documentation project demonstrating technical and editorial review;
- a business-plan fixture separating facts, calculations, assumptions, judgments, and unsupported claims;
- a research-report fixture demonstrating source provenance, synthesis, citations, and limitations.

Reference artifacts will be published with the release that can execute them. Until then, the design documents are the project contract.

## About the Author

PeerFoil is built by [Gabriel Mongefranco](https://gabriel.mongefranco.com), a database and
software architect who has spent two decades building data platforms in healthcare and
research — enterprise data warehouses, BI systems, knowledge bases, and the first architecture for mobile and
wearable research data at a large research university.

Learn more at: https://gabriel.mongefranco.com

## Contact

Questions, bug reports, enhancement ideas and requests are welcome as GitHub issues. Feel free to send pull requests as well!

## Credits

**Coming soon.** The final dependency and attribution list will be generated from the first distributable release.

#### This work is based in part on the following projects and libraries:

The current design plans to reuse the official [Codex plugin for Claude Code](https://github.com/openai/codex-plugin-cc), [Codex CLI](https://github.com/openai/codex), [Git](https://git-scm.com/), the [Agent Skills specification](https://agentskills.io/specification), and the [Model Context Protocol](https://modelcontextprotocol.io/). Inclusion, versions, licenses, copied material, notices, and complete credits will be verified before release.

## License

### Copyright Notice

Copyright © 2026 Gabriel Mongefranco

### Trademark Notice

PeerFoil™ is a trademark of Gabriel Mongefranco.

### Software and Library License Notice

SPDX license: `GPL-3.0-or-later`.

PeerFoil source code, tests, skills, agent definitions, project packs, templates,
schemas, plugin metadata, configuration, and executable or machine-consumed Markdown
are software or operational artifacts under this license unless the file carries a
different SPDX identifier.

This program is free software: you can redistribute it and/or modify it under the terms
of the GNU General Public License as published by the Free Software Foundation, either
version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program.
If not, see <https://www.gnu.org/licenses/gpl-3.0-standalone.html>.

### Documentation License Notice

SPDX license: `GFDL-1.3-or-later`.

Permission is granted to copy, distribute and/or modify the human-facing prose
documentation in this repository—including this README, `PeerFoil-Method.md`,
`architecture.md`, `implementation-plan.md`, and later `docs/` content—under the terms of
the GNU Free Documentation License, Version 1.3 or any later version published by the Free
Software Foundation; with no Invariant Sections, no Front-Cover Texts, and no Back-Cover
Texts, unless a file carries a different SPDX identifier. An explicit file identifier wins
over its path. See
<https://www.gnu.org/licenses/fdl-1.3-standalone.html>

The complete GPL license text is included in [`LICENSE`](LICENSE). The complete GFDL
license text will be added before the first public release.

## Citation

If you find this repository or its specifications useful, please cite it.

> *Mongefranco, Gabriel (2026). PeerFoil™. Software and documentation. <https://github.com/gabrielmongefranco/peerfoil>*

---

Copyright © 2026 Gabriel Mongefranco
