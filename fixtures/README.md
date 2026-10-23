<!--
This file is part of PeerFoil.
fixtures/README.md
Author(s): Gabriel Mongefranco.
Created: 2026-10-21
Last Modified: 2026-10-21
Summary: Explains the three Stage 5 reference fixtures, what they seed, and how to run their guided journeys.
Notes: Everything inside a fixture directory except its README.md is synthetic project content and carries no PeerFoil header, like the plugin's templates. The static checks validate each fixture's .peerfoil JSON records against the schemas.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# PeerFoil fixtures

[Return to the PeerFoil README](../README.md)

These are the three small reference projects that Phase 1, Stage 5 uses to exercise the
complete guided workflow, and that Phase 2 reuses for the enforced path. Each fixture is
a tiny project frozen at the production boundary: its decisions, architecture, Quality
Contract, and plan are recorded and accepted under `.peerfoil/`, the architecture and
plan reviews are seeded synthetic preconditions rather than live review evidence, and no
production, evidence, or phase review has happened yet. A fresh session that resumes a
fixture starts at the first ready task.

| Fixture | Toolchain | Deliverable |
|---|---|---|
| [software-go](software-go/README.md) | Go | A line-counting command |
| [software-node](software-node/README.md) | Node.js | The same command in JavaScript |
| [documentation](documentation/README.md) | Markdown | A short backup-and-restore guide |

Each fixture seeds at least two things, documented in its own README:

1. **A requirement change** already placed through change intake: `cr-0001` raised the
   plan to revision 2 and added one task, with the revision 1 snapshot retained under
   `.peerfoil/plans/`.
2. **A review defect** the phase review must find: one recorded requirement that no task
   covers and that the current content violates.

The documentation fixture additionally seeds a failing required check: a broken relative
link in the guide, which must block validation until it is repaired.

## Running a fixture journey

1. Copy the fixture directory somewhere outside this repository and run `git init`,
   `git add -A`, and an initial commit there, because the guided workflow needs a Git
   repository with a clean tree.
2. Load the PeerFoil plugin and run `/peerfoil:status` to confirm the recorded state,
   then `/peerfoil:resume` to produce the first ready task.
3. Follow the journey script in the fixture's README: produce the tasks, let the host
   run the declared checks, run `/peerfoil:review-phase`, and compare what happens with
   the expected results in that README, including the seeded defect's blocking finding.

The fixtures contain no credentials, no personal data, and no network access. Their
`.peerfoil` records use synthetic timestamps and null session identifiers.

[Return to the PeerFoil README](../README.md)
