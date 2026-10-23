<!--
This file is part of PeerFoil.
fixtures/documentation/README.md
Author(s): Gabriel Mongefranco.
Created: 2026-10-21
Last Modified: 2026-10-21
Summary: Describes the documentation reference fixture, a Markdown backup guide frozen at the production boundary with a seeded requirement defect and a seeded broken link, and gives the journey script and expected results.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# Documentation fixture: SQLite Backup Guide

[Return to the fixtures README](../README.md)

This fixture is a tiny documentation project, `peerfoil-fixture-doc`, that uses the
Documentation Pack. Its deliverable is [guide.md](guide.md), a short how-to guide about
backing up a small SQLite database. The project is frozen at the production boundary:
decisions, architecture, Quality Contract, and plan revision 2 are recorded and accepted
under `.peerfoil/`, task tk-001 is ready, task tk-002 is planned, and no production,
evidence, or phase review has happened yet. Everything in this directory except this
README is synthetic user-project content.

## Layout

| Path | Content |
|---|---|
| [guide.md](guide.md) | The deliverable: the backup guide, still missing its verification and restore sections |
| [.peerfoil/project.json](.peerfoil/project.json) | Project record: pack documentation 0.1.0, state produce, plan revision 2 |
| [.peerfoil/decisions.md](.peerfoil/decisions.md) | Four resolved decisions, d-0001 to d-0004 |
| [.peerfoil/architecture.md](.peerfoil/architecture.md) | Architecture revision 1, accepted |
| [.peerfoil/quality.md](.peerfoil/quality.md) | Quality Contract revision 1, accepted |
| [.peerfoil/plan.json](.peerfoil/plan.json) | Plan revision 2, accepted, with change cr-0001 |
| [.peerfoil/plan.md](.peerfoil/plan.md) | The readable view of plan revision 2 |
| [.peerfoil/plans/plan-1.json](.peerfoil/plans/plan-1.json) | The retained plan revision 1 snapshot, before cr-0001 |
| [.peerfoil/history.jsonl](.peerfoil/history.jsonl) | Five accepted transitions, tr-0001 to tr-0005 |
| [.peerfoil/reviews/](.peerfoil/reviews/) | rv-0001 (architecture), rv-0002 (plan), rv-0003 (change) |

## The two seeded items

1. **A requirement change through change intake.** Change `cr-0001` ("Add a restore
   section.") raised the plan from revision 1 to revision 2 and added task `tk-002` with
   requirement `r-0004`. The revision 1 snapshot is retained at
   `.peerfoil/plans/plan-1.json`, and the change entry in `.peerfoil/plan.json` records
   review `rv-0003` and acceptance `reviewed`.
2. **A review defect the phase review must find.** Requirement `r-0002` in
   `.peerfoil/plan.json` says every factual claim in the guide names its source, yet the
   sentence about teams recovering 60 percent faster in `guide.md` names no source
   anywhere, and r-0002's tasks list is empty, so no task covers the gap. This is left in
   place on purpose.

Separately from the r-0002 finding, `guide.md` links to `checklist.md`, which does not
exist. That broken link makes the required link-check evidence fail, which must block
validation until it is repaired. The broken link is seeded state, not an accident.

## Journey script

1. Copy this directory somewhere outside this repository.
2. In the copy, run `git init`, `git add -A`, and an initial commit, because the guided
   workflow needs a Git repository with a clean tree.
3. Load the PeerFoil plugin and run `/peerfoil:status`. It should report state produce,
   phase ph-01, stage st-01, plan revision 2, with tk-001 ready and tk-002 planned.
4. Run `/peerfoil:resume` to produce tk-001, the verification section, and then tk-002,
   the restore section, letting the host record the declared evidence after each task.
5. Run `/peerfoil:review-phase` and compare the outcome with the expected results below.

## Expected results

- The phase review must raise a blocking finding on requirement `r-0002`: the guide's
  fabricated statistic ("Teams that automate backups recover from data loss 60 percent
  faster.") has no source, the requirement is violated, and no task covers it.
- The required link-check evidence must fail on the `checklist.md` link and block
  validation until the link is repaired or removed. This failure is separate from the
  r-0002 review finding.
- The architecture and plan reviews under `.peerfoil/reviews/` (rv-0001, rv-0002,
  rv-0003) are seeded synthetic preconditions, not live review evidence. A real run must
  produce its own review records for the phase.

[Return to the fixtures README](../README.md)
