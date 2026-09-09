<!--
This file is part of PeerFoil.
plugins/peerfoil/references/changes.md
Author(s): Gabriel Mongefranco; OpenAI Codex.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Guides change placement, plan revisions, selective staleness, and retained traceability.
Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# Change intake reference (Guided)

Use for a request, discovered bug, TODO, skipped check, deviation, repair, deferral, or
decline after a plan exists. Read [planning.md](planning.md) and [records.md](records.md).
The coordinator acts as change steward with the configured effort; no new workflow
engine or automatic impact analysis exists in Skills 0.1.

## 1. Compare and place

Read the current project, decisions, architecture, quality, authoritative plan, reviews,
captures, and evidence. Confirm any active writer has stopped and capture its work
before changing the plan. A request arriving during production waits at this boundary;
do not edit its inputs underneath it.

Allocate `cr-NNNN`. Summarize the request without private chat content. Compare priority,
dependencies, risk, rework, architecture, required evidence, and the authorized scope.
Recommend one placement with a short reason:

| Placement | Meaning |
|---|---|
| `current-stage` | Needed for the active stage's outcome |
| `later-stage` | Belongs after current work, within this phase |
| `later-phase` | Belongs in a later phase; does not authorize starting it |
| `backlog` | Retained without a scheduled task |
| `declined` | Not accepted, with the reason retained |

Use existing user authorization for routine placement. A decline or backlog placement
that drops or postpones explicitly requested work needs the user's decision unless that
placement is already authorized. Ask when scope, stage order,
behavior, cost, privacy, architecture, permissions, or risk needs a consequential decision.
Never turn a required failed check into an optional or declined requirement. If the
architecture or quality must change, pause production, return through the existing
architecture/plan review and user acceptance steps, and retain the request meanwhile.

## 2. Write the candidate revision

Before replacing an accepted plan, save its exact JSON as
`.peerfoil/plans/plan-N.json`, where N is its revision. Never overwrite a differing
snapshot. These snapshots use the same plan schema and contain no raw model output.
Upgrade a version 1 candidate to schema version 2 and initialize `changes: []`; keep
the version 1 accepted snapshot intact. Keep previous change entries. Add a structured `changes[]` entry as defined in records,
including the reason, prior revision, affected tasks, retained tasks, and evidence links.
Validate unique change IDs, a prior revision that exists and precedes the candidate,
and existing task/evidence/review references. Affected and retained task sets must be
disjoint. Every existing task is either affected or retained; new tasks are affected.
Resolve references inside the repository and reject link/junction paths before reading.
Keep `backlog[]` for unscheduled requests; mirror backlog/declined placements there for
compatibility, using the same ID. Record every destination in `changes[]`.

Increment the candidate plan revision once per intake, including a decline or deferral.
Set status `draft`, timestamp, steward actor, and revision reason. Regenerate `plan.md`,
including its change table. Leave `project.revisions.plan` at the last accepted revision
until the candidate is accepted. A pending candidate blocks new production. If a draft
already exists, retain the new request in its change entries and review that candidate;
do not overwrite it or allocate another accepted revision prematurely.

- Preserve stable IDs and unrelated work. Include direct changes and transitive
  dependents whose assumptions, inputs, scope, or evidence actually change.
- Mark affected existing tasks `stale`; keep their old author and evidence records.
  New tasks are `planned`. A stale task becomes `ready` only after its revised inputs,
  dependencies, scope, and acceptance are validated against the accepted candidate.
- Re-tie unaffected tasks explicitly to the candidate's revisions and retain their
  status. Record why inputs and criteria remain identical. Never rewrite original
  evidence or captures to claim they ran under the new plan.
- Old evidence may support retained work only through the change entry's `retained_tasks`
  and `evidence` links plus a current input comparison. Since the snapshot includes
  other deliverables, any changed snapshot requires fresh evidence; unchanged task
  status alone never proves that old results apply.
- Update requirement-to-task links, dependency order, active pointers, and evidence
  coverage. Validate with planning section 3, except preserve existing statuses/IDs,
  allow existing backlog, and allow affected stale tasks to carry old revisions until
  they are re-tied. Do not reset all work to `planned`.

## 3. Review and accept

Changed task substance, requirements, evidence procedures, acceptance, dependencies,
or stage scope requires a fresh different-family plan review under [review.md](review.md),
then the user's acceptance of consequential changes. A pure reorder uses the existing
planning exception only after dependency order is checked and the user approves it.
Architecture/quality changes also require their own current review and acceptance.

A bookkeeping-only entry (for example an unscheduled backlog item or a decline) that
leaves all scheduled work, rules, criteria, and inputs unchanged may carry forward the
prior review. Record the prior accepted revision and review IDs in `changes[].reviews`
and `acceptance: carried-forward`, explain the comparison in `reason`, and retain the
original review untouched. The candidate `revised_at` records the intake time; for review
freshness, follow the retained prior-plan chain back to the last substantively reviewed
revision and compare that snapshot's `revised_at` with its review. Do not compare an old
review with the bookkeeping candidate's newer timestamp or change the original review. This is review continuity for unchanged material, not an
agent approving its own changed plan. If comparison cannot establish that, obtain review.

On acceptance set the candidate `accepted`, advance `project.revisions.plan`, update
`updated_at`, regenerate `plan.md`, and append a same-state transition with valid `refs`
keys. Reference the change ID in the summary (there is no `refs.changes` key). Set
`changes[].acceptance` to `reviewed` with review IDs for substantive revisions, or
`user-reorder` for the approved pure reorder. Record the accepting decision and time
in the reason/history. Do not alter `revised_at` for status-only approval; it identifies
the substantive content the reviewer inspected.

If paused, retain the recoverable prior state in history. Resume only after the reason
is resolved, the accepted revision chain is coherent, and production gates pass again.
For affected work, collect new capture/evidence before reporting current validation.
Stop at the phase review boundary. A repair enters the plan through [repair.md](repair.md),
which uses this intake with both reviewers' recorded agreement standing in for a
separate plan review.
