<!--
This file is part of PeerFoil.
docs/core-alpha.md
Author(s): Gabriel Mongefranco.
Created: 2026-10-21
Last Modified: 2026-10-21
Summary: How to build and use the peerfoil command, what each of its five commands does, how recovery works, and the Alpha boundary.

Copyright © 2026 Gabriel Mongefranco

Permission is granted to copy, distribute and/or modify this document under the terms of
the GNU Free Documentation License, Version 1.3 or any later version published by the Free
Software Foundation; with no Invariant Sections, no Front-Cover Texts, and no Back-Cover
Texts. See <https://www.gnu.org/licenses/fdl-1.3.html>.
-->

# PeerFoil

## Core Alpha: the peerfoil command

[Return to the PeerFoil README](../README.md)

Core Alpha is the first PeerFoil executable. It reads the same `.peerfoil/` records the
Skills release creates, runs one planned software task at a time in a dedicated Git
worktree, runs the declared checks itself, records who produced and reviewed every
change, and stops safely when it needs you. This page covers building it, the five
commands, recovery, and the Alpha boundary. It is a developer preview, not a release.

### Build it

You need Go 1.22 or later. From a checkout of this repository:

```text
go build ./cmd/peerfoil
```

That produces one `peerfoil` binary with no further dependencies. The record schemas
and the built-in project packs are embedded in it. Development builds are not signed or
versioned installers.

### The five commands

```text
peerfoil init      Create the .peerfoil project records for this repository.
peerfoil doctor    Check that this computer and repository are ready.
peerfoil start     Run the next step of the planned software phase.
peerfoil status    Show where the project stands and what comes next.
peerfoil resume    Continue safely from the last recorded boundary.
```

Every command accepts `-C <directory>` to run as if started there, and `--help`. The
exit codes are stable: 0 success, 2 usage, 3 configuration, 4 authentication,
5 validation, 6 policy, 7 provider, 8 git, 9 evidence, 10 timeout, 11 cancelled,
12 internal. `peerfoil` never prompts; it asks through its output and a nonzero exit
code, so it behaves the same in a terminal and a script.

`init` shows the records it would create and writes them only with `--yes`. It never
overwrites an existing `.peerfoil/` directory. A project started with the PeerFoil
skills needs no `init`.

`start` enforces the production gates before anything runs: the plan must be accepted
at the current revision, the architecture and plan must have approved reviews on
record with their independence recorded (a user-accepted Reduced assurance review
counts, exactly as D-0014 allows), the working tree must be clean outside
`.peerfoil/`, and no interrupted operation may be waiting. It then selects the next eligible task
deterministically, sends one bounded request to the producer in a fresh worktree,
captures the patch and authorship before anything else touches the work, checks the
changed paths against the task's allowed paths, scans the patch for credentials, runs
the required checks against the exact produced commit, and integrates passing work onto
the `peerfoil/integration` branch. Your own branch and checkout never change.

`resume` continues from recorded boundaries only. When the phase's tasks are all
validated, it runs the two-family phase review: one fresh Claude reviewer and one fresh
Codex reviewer read the same frozen revision, their findings are merged without hiding
disagreement, and approval requires the reviewer from a different model family than the
work's author. When both ask for a repair, one repair cycle runs with the repair
producer, the affected checks run again, and a fresh reviewer from the other family
verifies the result. Material disagreement stops for you instead of inventing
consensus.

### Recovery after an interruption

Before every operation that changes work, the producer and repair runs, `peerfoil`
writes an intent to a local journal in your user cache directory, and a result
afterward. Reviewer calls and check commands are not journaled in Alpha, because
rerunning them is safe. After a crash, `resume` reads that journal. An operation with no recorded result is never rerun automatically: resume
tells you what was interrupted and asks you to confirm the old process stopped, with
`peerfoil resume --confirm-stopped`. It then closes the attempt as failed, keeps the
retained worktree and captured records for inspection, and pauses. Accepted records are
written atomically, so a crash leaves the old complete file or the new complete file,
never a torn one. The journal is a cache: deleting it loses recovery convenience, never
project truth, which stays in Git under `.peerfoil/`.

One narrow window is a known Alpha limit: a crash exactly between the history append
and the project write leaves `project.json` one transition behind `history.jsonl`.
Every command then refuses with a message that names the exact one-line repair, instead
of guessing which file is right.

### What Core Alpha does not do

- It runs the production part of the journey. Decisions, architecture, planning, and
  change intake stay with the PeerFoil skills in this release.
- One sequential software path, one writer at a time, one review round, one repair
  cycle. There is no SQLite, no local-model routing, and no MCP policy yet.
- Required human and inspection evidence pauses for a person. Core Alpha has no command
  for recording a human result yet, so a documentation project stops honestly at that
  gate.
- Cancelling a run kills the direct child process only; a tool's own child processes
  can keep running until they finish. This is a documented Alpha limit.
- Worktrees isolate changes; they are not a security sandbox, and `peerfoil` provides
  no sandbox of its own.

### Conclusion

You can build the binary, check your environment with `doctor`, and drive one planned
phase from `start` through the two-family review with `resume`, with every step
recorded in your repository. The [known limits page](known-limits.md) states what is
and is not enforced.

### Additional resources

- [Known limits](known-limits.md)
- [Phase 2 implementation plan](plans/phase-2-core-alpha.md)
- [Setup for the Skills release](setup.md)
- [Fixtures to try the journey on](../fixtures/README.md)

[Return to the PeerFoil README](../README.md)
