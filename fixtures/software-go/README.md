<!--
This file is part of PeerFoil.
fixtures/software-go/README.md
Author(s): Gabriel Mongefranco.
Created: 2026-10-21
Last Modified: 2026-10-21
Summary: Documents the Go reference fixture: a line-counting command frozen at the production boundary, with one seeded requirement change and one seeded defect the phase review must find.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# Line Counter (Go) fixture

[Return to the fixtures overview](../README.md)

This fixture is a tiny Go project, frozen at the production boundary, that Stage 5 uses
to exercise the complete guided workflow. Its decisions, architecture, Quality Contract,
and plan are recorded and accepted under `.peerfoil/`. The program prints the number of
non-empty lines in a text file. Everything in this directory except this README is
synthetic user-project content.

## Layout

| Path | Content |
|---|---|
| `go.mod` | Module `example.com/linecount`, Go 1.22, no dependencies |
| `main.go` | The command: reads one path argument and prints the non-empty line count |
| `main_test.go` | Tests for the counting logic, using temporary files |
| `.peerfoil/` | The accepted project records: project, decisions, architecture, quality, plan, history, reviews |
| `.peerfoil/plans/plan-1.json` | The retained plan revision 1 snapshot, from before change cr-0001 |

## The two seeded items

1. **A requirement change already placed through change intake.** Change `cr-0001`
   ("Read from standard input when no path is given") is recorded in the `changes` list
   of `.peerfoil/plan.json`. It raised the plan to revision 2 and added task `tk-002`,
   and the revision 1 snapshot is retained at `.peerfoil/plans/plan-1.json`.
2. **A review defect the phase review must find.** Requirement `r-0002` in
   `.peerfoil/plan.json` says a missing or unreadable file prints a readable error to
   standard error and exits with code 1. `main.go` violates it: the error from
   `os.ReadFile` is discarded, so a missing file silently prints `0` and exits with
   code 0. No task covers the requirement (its `tasks` list is empty), and
   `main_test.go` deliberately has no missing-file test.

## Journey script

1. Copy this directory somewhere outside this repository.
2. In the copy, run `git init`, `git add -A`, and an initial commit, because the guided
   workflow needs a Git repository with a clean tree.
3. Load the PeerFoil plugin and run `/peerfoil:status`. It should report state
   `produce`, phase `ph-01`, stage `st-01`, plan revision 2, and change `cr-0001`.
4. Run `/peerfoil:resume`. It should pick up `tk-001`, the first ready task.
5. Produce the tasks, let the host run the declared checks, then run
   `/peerfoil:review-phase`.

## Expected results

- The phase review must raise a **blocking finding against requirement `r-0002`**: the
  current `main.go` ignores the `os.ReadFile` error, no task covers the requirement, and
  no test exercises the missing-file case. A phase review that approves without this
  finding has failed the fixture.
- The reviews seeded under `.peerfoil/reviews/` (`rv-0001` architecture, `rv-0002` plan,
  `rv-0003` change) are synthetic preconditions that let the workflow resume at the
  production boundary. They are not live review evidence, and nothing in them was
  produced by a real review run.
- `go build ./...`, `go test ./...`, and `go vet ./...` pass in this directory as
  checked in; the seeded defect is a behavior gap, not a compile or test failure.

[Return to the fixtures overview](../README.md)
