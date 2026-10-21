<!--
This file is part of PeerFoil.
fixtures/software-node/README.md
Author(s): Gabriel Mongefranco.
Created: 2026-10-21
Last Modified: 2026-10-21
Summary: Describes the Node.js line-counter fixture: a synthetic project frozen at the production boundary, with one seeded change request and one seeded review defect the phase review must find.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# Fixture: Line Counter (Node.js)

[Return to the fixtures overview](../README.md)

This fixture is a tiny Node.js command-line project, `peerfoil-fixture-node`, frozen at
the production boundary. Its decisions, architecture, Quality Contract, and plan are
recorded and accepted under `.peerfoil/`, and no production, evidence, or phase review has
happened yet. The command counts the non-empty lines in a text file, uses no dependencies,
and runs with `node linecount.js <file>`.

Everything outside this README is synthetic project content. It belongs to the imagined
user of the fixture, carries no PeerFoil header, and uses synthetic timestamps with null
session identifiers.

## Layout

| Path | Content |
|---|---|
| `package.json` | Project manifest with the `node --test` test script and no dependencies |
| `linecount.js` | The line-counting command (CommonJS) |
| `linecount.test.js` | Tests using `node:test` and `node:assert` |
| `.peerfoil/project.json` | Project record: produce state, phase `ph-01`, stage `st-01` |
| `.peerfoil/decisions.md` | Four decisions, `d-0001` to `d-0004` |
| `.peerfoil/architecture.md` | Architecture revision 1, accepted |
| `.peerfoil/quality.md` | Quality Contract revision 1, accepted |
| `.peerfoil/plan.json` | Plan revision 2, accepted, with change `cr-0001` |
| `.peerfoil/plan.md` | Readable view of plan revision 2 |
| `.peerfoil/plans/plan-1.json` | Retained snapshot of accepted plan revision 1 |
| `.peerfoil/history.jsonl` | Five accepted workflow transitions |
| `.peerfoil/reviews/` | Seeded reviews `rv-0001` to `rv-0003` |

## The two seeded items

1. **A requirement change through change intake.** Change `cr-0001`, "Read from standard
   input when no path is given.", sits in the `changes` list of `.peerfoil/plan.json`. It
   raised the plan to revision 2, added task `tk-002` and requirement `r-0004`, and
   retained task `tk-001`. The accepted revision 1 snapshot, without `tk-002` and
   `r-0004`, is kept at `.peerfoil/plans/plan-1.json`.
2. **A review defect the phase review must find.** Requirement `r-0002` in
   `.peerfoil/plan.json`, "A final line without a trailing newline is still counted.",
   is violated by `linecount.js` and covered by no task: its `tasks` list is empty. The
   command drops the last element of the split unconditionally, so a final line without a
   trailing newline is never counted, and every test in `linecount.test.js` writes files
   that end with a newline, so the tests pass without touching the gap.

## Journey script

1. Copy this directory somewhere outside this repository.
2. In the copy, run `git init`, then `git add -A`, then `git commit -m "Seed the
   line-counter fixture"`, because the guided workflow needs a Git repository with a
   clean tree.
3. Load the PeerFoil plugin and run `/peerfoil:status`. Expect the produce state in phase
   `ph-01`, stage `st-01`, with task `tk-001` ready and `tk-002` planned.
4. Run `/peerfoil:resume` to produce `tk-001`, the `--json` flag, and let the host run the
   declared checks. Continue with `tk-002`.
5. Run `/peerfoil:review-phase` and compare the result with the expected results below.

## Expected results

- The phase review raises a **blocking finding** on requirement `r-0002`: a final line
  without a trailing newline is not counted, the requirement is covered by no task, and
  no test exercises a file that does not end with a newline.
- The reviews `rv-0001`, `rv-0002`, and `rv-0003` under `.peerfoil/reviews/` are seeded
  synthetic preconditions that stand in for the architecture, plan, and change reviews.
  They are not live review evidence, and the phase review must not count them as evidence
  for produced work.

[Return to the fixtures overview](../README.md)
