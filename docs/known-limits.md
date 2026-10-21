<!--
This file is part of PeerFoil.
docs/known-limits.md
Author(s): Gabriel Mongefranco.
Created: 2026-10-21
Last Modified: 2026-10-21
Summary: States plainly what each PeerFoil release can and cannot enforce, so a guided safeguard is never mistaken for a mechanical one.

Copyright © 2026 Gabriel Mongefranco

Permission is granted to copy, distribute and/or modify this document under the terms of
the GNU Free Documentation License, Version 1.3 or any later version published by the Free
Software Foundation; with no Invariant Sections, no Front-Cover Texts, and no Back-Cover
Texts. See <https://www.gnu.org/licenses/fdl-1.3.html>.
-->

# PeerFoil

## Known limits

[Return to the PeerFoil README](../README.md)

PeerFoil labels every safeguard either **Guided** or **Enforced**. Guided means the
written procedure tells the agents what to do and the records make violations visible;
nothing mechanically stops a model that ignores the procedure. Enforced means a local
application validates the step and refuses unsafe transitions. This page says which is
which, and what no release provides.

### PeerFoil Skills 0.1 is guided

Everything in the Skills release is guided. In particular, Skills 0.1 cannot
mechanically enforce:

- workflow states and their order;
- review pass limits, turn limits, or time limits (the host enforces an agent's turn
  cap and a command's timeout; the rest is written procedure);
- command evidence: the coordinating host runs checks and records results, but no
  controller owns the process;
- atomic record updates or crash recovery (an interrupted update is a visible mismatch
  to reconcile, not a rollback);
- a single writer (the capture gates are procedure, not locks); and
- unattended operation.

Skills 0.1 shows **Guided** in setup, status, and every phase result. A status line
claiming Enforced in this release is wrong by definition.

### Core Alpha 0.2 additions and remaining gaps

Core Alpha is the first release with an enforced path: a local `peerfoil` application
validates project records, runs one software task in a dedicated Git worktree, runs the
declared checks itself, ties evidence to exact revisions, blocks self-approval, and
resumes at completed task boundaries. Its limits are also firm:

- one sequential software path, one producer at a time;
- one fixed review round and one guided repair cycle;
- no SQLite, selective invalidation, local-model routing, or MCP policies yet; and
- recovery only at completed task boundaries.

### Limits that apply to every release

- **No operating-system sandbox.** PeerFoil recommends and detects isolation the host
  provides, but provides none itself. The preferred layer is an outside container or
  virtual machine that holds the whole agent toolchain, such as
  [Agent Airlock](https://github.com/gabrielmongefranco/agent-airlock); the agent
  tools' own sandboxes, such as the Claude Code Bash sandbox, come second. The Claude
  Code sandbox does not support native Windows, and its network allowlist on WSL2 is
  best effort.
- **Git worktrees are change isolation, not security.** Nothing in PeerFoil contains
  hostile code. Untrusted projects need an outside disposable environment.
- **Independence needs two model families.** With only one family signed in, every
  review is labeled Reduced assurance. An advisory local model can inspect a draft, and
  the record distinguishes that, but the label does not improve until Phase 7
  qualification exists.
- **A model's claim is not evidence.** Only retained check results, inspection records,
  or explicit human confirmation satisfy required evidence.
- **No certification.** PeerFoil can raise the quality floor. It does not certify that
  work is correct, secure, accessible, or fit for any regulation.

### Conclusion

Use the assurance label as the honest boundary: Guided tells you the agents were
instructed and recorded, Enforced tells you a controller checked. When in doubt,
`/peerfoil:status` prints the label for the current project.

### Additional resources

- [PeerFoil method](PeerFoil-Method.md), sections 11 and 12, on the two releases
- [Implementation plan](implementation-plan.md), on release boundaries
- [Privacy](privacy.md)

[Return to the PeerFoil README](../README.md)
