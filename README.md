<!--
This file is part of PeerFoil.
README.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-04
Last Modified: 2026-09-04
Summary: Provides an overview of PeerFoil and its planned releases.
Notes: See the /docs directory for complete project documentation.

Copyright © 2026 Gabriel Mongefranco

Permission is granted to copy, distribute and/or modify this document under the terms of
the GNU Free Documentation License, Version 1.3 or any later version published by the Free
Software Foundation; with no Invariant Sections, no Front-Cover Texts, and no Back-Cover
Texts. See <https://www.gnu.org/licenses/fdl-1.3.html>.
-->

# PeerFoil

***Better vibe, higher-quality coding.***

## Description

PeerFoil is an open-source workflow that lets two different AI coding tools work together.
One helps you think through the application and create a plan. Another writes the code.
They check each other's work so you are not relying on one model to find its own mistakes.

It is designed for people who build software mostly alone, whether they are professional
developers or learning as they go. PeerFoil keeps the complicated coordination in the
background. You approve the important decisions, and it handles the plans, coding tasks,
tests, reviews, fixes, and lessons learned.

Software development is the main use. The same approach can also help write documentation,
business plans, and research reports. Each type of project gets its own instructions and
checks, while using the same independent review process.

PeerFoil will work on Windows, macOS, and Linux. It will not require a paid service other
than the AI models you choose. Local models will also be supported.

## Quick Start Guide

**Coming soon.** PeerFoil has not been released yet.

The first version will be a set of skills for Claude Code. It will use the official
[Codex plugin for Claude Code](https://github.com/openai/codex-plugin-cc) to send coding and
review tasks to Codex. A small local command-line application will follow.

The normal workflow will be simple:

1. Describe what you want to build or change.
2. Answer the important questions.
3. Review the proposed architecture and order of work.
4. Let PeerFoil build and check one phase at a time.
5. Step in only when PeerFoil needs a decision or approval.

Model selection, effort levels, review limits, connected tools, local models, and other
controls will stay under Advanced settings.

## Documentation

- **[PeerFoil Method](docs/PeerFoil-Method.md):** What PeerFoil does and how its workflow
  protects quality.
- **[Architecture](docs/architecture.md):** How the planned application is organized and
  how its parts work together.
- **[Implementation Plan](docs/implementation-plan.md):** What will be built first, what
  will come later, and how each release will be checked.
- **[Phase Prompt Template](docs/phase-prompt-template.md):** A prompt you can paste into
  a new chat before starting a phase or stage.

## Status

**Design complete; implementation coming soon.**

| Release | Target | What you will be able to do |
|---|---:|---|
| PeerFoil Skills 0.1 | Day 5 | Run the complete guided workflow from Claude Code |
| PeerFoil Core Alpha 0.2 | Day 13 | Use a local application that runs and records the workflow for you |
| PeerFoil 1.0 | By Week 26 | Use the complete cross-platform product, including local models and non-coding projects |

The day count starts when implementation begins. Dates may change if a release does not
meet its quality checks.

## Example Applications

**Coming soon.** Planned examples include:

- a small application built and reviewed from start to finish;
- a documentation project with technical and editorial review;
- a business plan with checked facts, calculations, and assumptions; and
- a research report with source tracking and citation checks.

## About the Author

PeerFoil is built by [Gabriel Mongefranco](https://gabriel.mongefranco.com), a database and
software architect who has spent two decades building data platforms in healthcare and
research — enterprise data warehouses, BI systems, knowledge bases, and the first architecture for mobile and
wearable research data at a large research university.

Learn more at: https://gabriel.mongefranco.com

## Contact

Questions, bug reports, enhancement ideas and requests are welcome as GitHub issues. Feel
free to send pull requests as well!

## Credits

**Coming soon.** Complete dependency and attribution details will be added with the first
release.

#### This work is based in part on the following projects and libraries:

- [Codex plugin for Claude Code](https://github.com/openai/codex-plugin-cc) — connects
  Claude Code and Codex without requiring PeerFoil to build another bridge.
- [Codex CLI](https://github.com/openai/codex) — handles coding and review tasks in the
  default setup.
- [Git](https://git-scm.com/) — stores project history and keeps work separated.
- [Agent Skills](https://agentskills.io/specification) — provides the portable format for
  PeerFoil's instructions.
- [Model Context Protocol](https://modelcontextprotocol.io/) — connects approved tools and
  knowledge sources when a project needs them.

Versions, licenses, copied material, and required notices will be checked before release.

## License

### Copyright Notice

Copyright © 2026 Gabriel Mongefranco

### Trademark Notice

PeerFoil™ is a trademark of Gabriel Mongefranco.

### Software and Library License Notice

PeerFoil source code, tests, skills, agent definitions, project packs, templates, schemas,
plugin files, configuration, and machine-consumed Markdown are licensed under the GNU
General Public License, version 3 or later, unless a file says otherwise.

This program is free software: you can redistribute it and/or modify it under the terms of
the GNU General Public License as published by the Free Software Foundation, either version
3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
See the GNU General Public License for more details.

You should have received a copy of the GNU General Public License along with this program.
If not, see <https://www.gnu.org/licenses/gpl-3.0-standalone.html>.

### Documentation License Notice

Permission is granted to copy, distribute and/or modify the human-facing documentation in
this repository under the terms of the GNU Free Documentation License, Version 1.3 or any
later version published by the Free Software Foundation; with no Invariant Sections, no
Front-Cover Texts, and no Back-Cover Texts, unless a file says otherwise.

See <https://www.gnu.org/licenses/fdl-1.3-standalone.html>.

## Citation

If you find this repository or its specifications useful, please cite it.

> *Mongefranco, Gabriel (2026). PeerFoil™. Software and documentation.
> <https://github.com/gabrielmongefranco/peerfoil>*

---

Copyright © 2026 Gabriel Mongefranco
