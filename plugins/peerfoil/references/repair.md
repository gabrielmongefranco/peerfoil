<!--
This file is part of PeerFoil.
plugins/peerfoil/references/repair.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Defines the one guided repair cycle: choosing an eligible repairer, recording the repair task, producing and capturing it, rerunning affected evidence, and obtaining fresh different-family verification.
Notes: Used after a phase review decides repair. Production mechanics are in production.md and evidence.md; the review transfer is in review.md. The product contract is docs/PeerFoil-Method.md, section 7.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# Repair reference (Guided)

A repair is one bounded change that resolves specific findings both reviewers agreed on.
The repairer never verifies its own repair, and verification comes from a fresh session
of a model family different from the repairer's. Skills 0.1 allows one repair cycle per
phase review round, as `settings.review.repair_cycles` records, and never runs a repair
at low effort.

Read [phase-review.md](phase-review.md), [production.md](production.md),
[evidence.md](evidence.md), [changes.md](changes.md), and [review.md](review.md) first.

## 1. What can be repaired

| Finding concerns | Repair path | Repairer | Verifier |
|---|---|---|---|
| A deliverable, patch, or evidence procedure | Section 3 onward | The `repair_producer` seat | The phase reviewer whose lineage differs from the repairer's |
| `architecture.md` or `quality.md` | The revise flow in [architecture.md](architecture.md), section 6, at the next revision | The architect seat | The architecture reviewer, as [review.md](review.md) selects it |
| `plan.json` | The revise flow in [planning.md](planning.md), section 5, at the next revision | The planner seat | The plan reviewer |
| A decision | Not repairable; the user changes the decision and the affected drafts return through their own review | The user | The drafts' reviewers |

The selection is a rule, not a model judgment, so no repair-selection review passes
are spent in this release. `settings.review.repair_selection_passes` and
`repair_selection_max_passes` are recorded for Core, which may select a repairer with
model help within those limits.

Record repairs change an accepted revision, so they end with the user's acceptance and
a plan re-tie through [changes.md](changes.md). The rest of this file covers
deliverable repairs, which are the common case.

## 2. Propose the repair task

1. Build a compact packet for the `peerfoil:repair-coordinator` agent containing:
   - every `open` `blocking` and `major` row from the shared list that the user did
     not defer, with its identifiers, item, location, requirement, severity, evidence,
     and both recommendations;
   - for each affected item, its author actor and the task and change set that
     produced it;
   - the tasks whose `allowed_paths` cover the affected paths, with their
     `required_evidence`;
   - the Quality Contract's evidence names and kinds; and
   - the path rules from [planning.md](planning.md), section 3, rule 7.
2. The agent returns one repair task proposal per group of findings that one bounded
   change can resolve. Validate before writing anything:
   - every finding identifier exists in the shared list and each appears in exactly one
     proposal;
   - `allowed_paths` obey the path rules and stay within the affected items' paths or
     the paths their tasks allowed;
   - every `evidence_to_rerun` entry names Quality Contract evidence, and it includes
     every required evidence of every affected task;
   - `acceptance` restates each finding's recommendation as a checkable statement.
   Ask the agent once to fix a specific problem; if it remains, stop and tell the user.
3. Choose the repairer by section 1. Resolve the seat's effort; when it is `low`, use
   `medium` and record why. Choose the verifier: the phase reviewer seat whose lineage
   differs from the repairer's. When no such seat is available, ask the user as
   [phase-review.md](phase-review.md), section 4, describes, before producing.

## 3. Record the repair in the plan

Follow [changes.md](changes.md) with placement `current-stage`:

- Allocate `cr-NNNN`. Its summary names the findings; its reason says both reviewers
  agreed on the repair in the phase review record.
- Add one new task per proposal with the next `tk-NNN`, `author_role`
  `repair_producer`, status `planned`, the validated scope, `allowed_paths`, `inputs`
  naming the findings, the affected change sets, and the phase review record,
  `required_evidence` from `evidence_to_rerun`, `depends_on` the affected tasks' own
  validated predecessors and never the affected tasks themselves, which the repair
  supersedes, and the validated acceptance.
- List the affected tasks in `affected_tasks` and set them `stale`; their original
  evidence stays as history, and their relationship to the repair is recorded in the
  change entry and the phase review record rather than as a dependency. Every other
  task is retained.
- Set `acceptance` to `reviewed` with `reviews` naming both pass-1 phase reviews. The
  reviewers' recorded agreement on the repair stands in for a separate plan review; no
  further planner or review pass is spent.
- Set the phase review record's status to `repair`, list the change request and task
  identifiers in its Repair section, and mark the addressed rows `accepted`.
- Set `workflow.state` to `repair`, `workflow.task` to the repair task, and append a
  transition `review` to `repair` with `refs.tasks`, `refs.reviews`, and
  `refs.phase_reviews`.

## 4. Produce the repair

Follow [production.md](production.md), sections 2 to 4, with the repair task and the
repairer seat, with one difference: `workflow.state` stays `repair` throughout.
Do not record the production reference's `produce` and `validate` transitions for a
repair. Record the launch as a same-state `repair` transition with `refs.tasks`,
`refs.change_sets`, and `refs.phase_reviews`, set the repair task `in_progress` and
then `produced`, and record the capture as another same-state `repair` transition.
The packet also carries the findings' text and recommendations. One call gets one
repair task. Capture the change set before any other edit, with the repairer as its
author. A blocked, partial, or malformed result pauses the cycle for the user; it
never triggers a second repairer.

## 5. Rerun affected evidence

Follow [evidence.md](evidence.md) for every item in `evidence_to_rerun` on the final
snapshot. Then recompute the input snapshot of every task in the phase against the
repaired files, not only the affected ones: a repair changes tracked deliverables, so
it can make a retained task's evidence stale. Rerun every required check whose
snapshot changed and record fresh evidence for it. Refresh the affected tasks: when
their required evidence passes on the new snapshot, set them `validated` again. Set
the repair task `validated` only when every item passes and every task in the phase
has passing required evidence on the final repaired snapshot. Append a transition
`repair` to `validate`. A required failure pauses the cycle for the user with the
failed evidence named; it cannot be voted away.

## 6. Verify

1. Freeze the verification material: the repair change set and patch, the fresh
   evidence records, every deliverable of the phase at its current content, and the
   findings addressed. Hash them with the snapshot recipe in
   [evidence.md](evidence.md) and add the rows and their digest, the **repair
   digest**, to the phase review record's Repair section as a second manifest.
2. Confirm the verifier has a pass left below `max_passes` and increment its passes
   used in the phase review record. Launch it with the packet from
   [review.md](review.md), section 4, with kind `repair`; a "Frozen material" line
   naming the repair change set, the repair digest, the revisions, and the pass
   number, one more than that reviewer's last pass; the second manifest as the files
   to read; the findings listed by identifier with their recommendations; and the
   instruction to confirm each finding and to report only new `blocking` findings on
   the changed material, as D-0024 requires. Use the same ten-turn and ten-minute
   limits. The verifier returns:

   ```json
   {
     "kind": "repair",
     "reviewed": { "change_set": "cs-0002", "repair_digest": "<sha256>", "plan_revision": 2, "architecture_revision": 1, "quality_revision": 1, "pass": 3 },
     "decision": "approve | repair | block",
     "confirmed": [ { "finding": "fd-0003", "status": "repaired | not-repaired", "evidence": "" } ],
     "findings": [],
     "remaining_risk": "",
     "model": "",
     "notes": ""
   }
   ```

3. Recompute the repair digest from the files immediately before accepting the
   result. Validate it with [review.md](review.md), section 8, plus: `reviewed`
   equals the frozen change set, digest, revisions, and pass; `confirmed` has exactly
   one entry per addressed finding; any new finding is `blocking` and located in the
   changed material, or it is recorded as `declined` under D-0024. Record the result
   as an `rv-NNNN.md` with kind `repair`, the change set and digest on its Frozen
   material line, and independence judged against the repairer's lineage. Append a
   transition `validate` to `review`.
4. When every addressed finding is `repaired` and no new `blocking` finding stands,
   set those rows `repaired` and the repair task `accepted`, then return to
   [phase-review.md](phase-review.md), section 10, at rule 4: every remaining row is
   dispositioned and carried into the plan through change intake before the phase is
   approved as section 11 describes, with the summary naming the repair.
5. Otherwise set `workflow.state` to `paused` with `paused_for` naming the finding that
   did not clear and the choices: allow another cycle under `/peerfoil:settings`,
   decline the finding with a recorded reason, or stop. Record the transition. The
   repair cycle is used; a second one needs the user's change to the setting.

## 7. Resuming a repair in a fresh chat

Read the newest phase review record first. Whenever its status is `repair`, this
table governs, before any ordinary production recovery and before the original bundle
digest check in [phase-review.md](phase-review.md), section 13. The original digest
no longer applies: check the repair change set's input snapshot against the files and,
once it exists, the second manifest's repair digest, and treat a mismatch as stale
work that returns to section 5.

| Record shows | Do |
|---|---|
| Status `repair`, no repair change request in the plan | Section 2 |
| Repair change request recorded, but its repair task is missing from the plan or the plan revision is not yet accepted | Section 3, completing the plan entry |
| Repair change request recorded, repair task `planned`, no change set for it | Section 4 |
| Repair task `in_progress`, or its change set `Capture status: pending` | Confirm the writer stopped and capture as [production.md](production.md), section 5, describes; then section 5 here |
| Repair task `produced`, or phase tasks with stale snapshots | Section 5 |
| Repair task `validated`, no second manifest | Section 6, step 1 |
| Second manifest recorded, no `repair` review | Section 6, step 2 |
| `repair` review `approve` | Section 6, step 4 |
| `repair` review `repair` or `block`, state not yet `paused` | Section 6, step 5 |
| `workflow.state` `paused` | Report `paused_for`; continue only when the need is met |

## 8. Rules

- The repairer's session and family never verify the repair.
- A repair never runs at low effort and never widens `allowed_paths` beyond the
  affected work.
- Fresh evidence and the verification bind to the repaired snapshot, never to the one
  the reviewers first saw.
- One cycle. When it does not clear the findings, the user decides.
- Everything the user sees carries the label **Guided**.
