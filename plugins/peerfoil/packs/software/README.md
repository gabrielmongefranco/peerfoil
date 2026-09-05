<!--
This file is part of PeerFoil.
plugins/peerfoil/packs/software/README.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Describes the Software project pack for people.
Notes: The machine-readable manifest is pack.json in this directory.

Copyright © 2026 Gabriel Mongefranco

Permission is granted to copy, distribute and/or modify this document under the terms of
the GNU Free Documentation License, Version 1.3 or any later version published by the Free
Software Foundation; with no Invariant Sections, no Front-Cover Texts, and no Back-Cover
Texts. See <https://www.gnu.org/licenses/fdl-1.3.html>.
-->

# Software Pack

The Software Pack is PeerFoil's default project pack. It is for applications, services,
libraries, and command-line tools that live in a Git repository and have build and test
commands.

The pack declares, in `pack.json`:

- the artifacts a software project produces: source, tests, packaging, and documentation;
- the typical stages, starting with a first working path that installs, starts, and
  completes one real user action;
- the evidence a Quality Contract can select, such as builds, tests, linting, dependency
  and security checks, a human user-journey check, and license checks;
- the four default review lenses; and
- the tools that setup checks: Git always, plus the toolchain suggested by files such as
  `go.mod` or `package.json`.

The pack contains no executable logic. It cannot run commands itself, change repository
rules, hide a failed check, or allow a model to approve its own work.

This is the pack's first version. Phase 1, Stage 2 completes its practical checks.
