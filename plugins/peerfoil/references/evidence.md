<!--
This file is part of PeerFoil.
plugins/peerfoil/references/evidence.md
Author(s): Gabriel Mongefranco; OpenAI Codex.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Defines host-run checks and exact input snapshots for guided production evidence.
Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# Evidence reference (Guided)

Use after [production.md](production.md) captures authorship. The host runs commands;
the producer's handoff never supplies authoritative pass results.

## Exact input snapshot

Keep `source_revision` null for uncommitted work. A base commit alone does not identify
that work. In `.peerfoil/evidence/cs-NNNN.md`, retain an input snapshot table with each
repository-relative path, file kind, and SHA-256 of its raw bytes, or `absent` for a
deleted path. Include all tracked deliverables, untracked task files, applicable rules,
and task inputs and check configuration. Record hashes of architecture, quality, and
the task's substantive fields separately; exclude mutable workflow status and evidence
records to avoid self-reference. Include any ignored input a check reads, or mark the
check unverified if its inputs cannot be identified. Record ignored generated outputs
and external tool versions separately. Do not retain private input contents.

Sort paths by their UTF-8 bytes and hash the UTF-8 JSON representation of the ordered
`[path, kind, sha256-or-absent]` rows, with no extra whitespace and unescaped Unicode.
Record that SHA-256 as `Input snapshot`. File kinds are `file`, `executable`, or `absent`;
use Git's executable mode for tracked files so Windows does not invent permission bits.
Reject links and case-colliding paths as required by the production reference. Preserve
raw bytes: line-ending changes change the snapshot. Store the full table so another
session can recompute it. The snapshot is inspectable evidence, not tamper resistance.

## Run and record

1. Verify capture is complete and task, architecture, quality, and plan revisions are
   current. Recompute the input snapshot before checks. On mismatch, retain old evidence
   as historical and report stale work. Do not silently relabel it as current.
2. Run each declared procedure with an explicit working directory inside the repository,
   argument arrays, and a finite timeout. Use the declared timeout, or ten minutes if
   none is specified. Check commands against existing authorization before execution.
   Do not concatenate model text into a shell command or install a dependency silently.
3. Retain one `ev-NNNN.md` per check using the evidence template. Include name/kind/level,
   task and change set, exact argument list or inspection procedure, safe relative working
   directory, exit code, elapsed seconds, tool version, UTC time, actor, plan revision,
   source revision, input snapshot, and redacted output. Include stdout and stderr needed
   to explain the result; keep raw logs outside Git. No model claim substitutes for output.
4. A timeout, cancellation, unavailable tool, or permission denial is `blocked` or `not-run`,
   never `pass`. Confirm subprocess cleanup before continuing. A nonzero exit is `fail`.
   Inspection evidence states observations and limitations. Human evidence needs the
   actual person's result and time; an agent cannot supply it.
5. Recompute the snapshot after checks. A changed input makes the result stale, even
   when exit code is zero. Capture any check-generated source changes with their author,
   route them through change intake, and rerun affected evidence. Do not delete outputs
   or undo edits without authorization.
6. Only if every required item and acceptance criterion passes on the current snapshot,
   set task `validated`, regenerate `plan.md`, update project time, and append history
   with `refs.tasks`, `refs.change_sets`, and `refs.evidence`. This is validation only;
   fresh independent phase review remains necessary for acceptance.

Required missing, blocked, stale, or failed evidence blocks completion. Record every
skipped check, deviation, bug, or repair need through [changes.md](changes.md), including
recommended checks intentionally skipped. A deferral or reviewer vote cannot change a
required failure into success. A required-check correction returns to production only
after a recorded plan revision and bounded task; every repair needs fresh different-family
verification. Guided repair coordination is Coming soon in Stage 4, so pause for that
path instead of inventing approval. Evidence remains available for later review.
