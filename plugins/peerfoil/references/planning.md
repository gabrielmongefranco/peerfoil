<!--
This file is part of PeerFoil.
plugins/peerfoil/references/planning.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Defines the planning step: how a skill turns an accepted architecture into an ordered plan of phases, stages, and small tasks, obtains independent review, and records the user's approval of the stage order.
Notes: Shared by the start and resume skills. The product contract is docs/PeerFoil-Method.md, section 6.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# Planning step reference

This step runs while `workflow.state` is `plan` and the architecture is `accepted`. It
produces `.peerfoil/plan.json` and `.peerfoil/plan.md`, has the plan reviewed by a
different model family, and ends when the user approves the order of stages. In this
build the project then waits in the `plan` state, because production arrives in a later
build.

## 1. Finding the current sub-step in a fresh chat

| Files show | Do |
|---|---|
| No `plan.json` | Section 2 |
| `plan.json` status `draft`, no review at its revision and pass | Section 4 |
| Latest plan review decision `repair` with an open `blocking` or `major` finding | Section 5 |
| Latest plan review decision `repair`, no open `blocking` or `major` finding, and `plan.json` `revised_at` later than that review's `Reviewed at` | Section 4, as the next pass |
| Latest plan review decision `block` | Show the findings and the `paused_for` reason; stop |
| Latest plan review decision `approve` whose frozen plan revision equals the draft's and whose `Reviewed at` is later than `plan.json` `revised_at`, while `plan.json` is still `draft` | Set it to `reviewed`, regenerate `plan.md`, then section 6 |
| `plan.json` status `reviewed` | Section 6 |
| `plan.json` status `accepted` | Section 8 |

## 2. The planner packet

Build a compact packet. Include only:

- the goal, project identifier, name, pack identifier and version, and profile;
- the pack's artifacts and typical stages;
- the full text of `.peerfoil/architecture.md` and `.peerfoil/quality.md`;
- every decision from `decisions.md` with its identifier, question, answer, and status;
- every finding with disposition `deferred` in the architecture reviews, each of which
  the plan must carry as a task or a requirement;
- repository facts: toolchain markers, top-level directories, and whether tests exist;
- the identifier and revision rules from
  `${CLAUDE_PLUGIN_ROOT}/references/records.md`, section 1, and the highest existing
  phase, stage, task, and requirement numbers; and
- when revising, the current `plan.json` and every open finding from the latest review.

Do not include this chat's history, the architect's packet, or the evaluator's packet.

## 3. Writing the draft

1. Launch the `peerfoil:planner` agent with the packet. Record the agent identifier the
   `Agent` tool shows as the planner's `session`, or `null` when none is shown.
2. Validate the returned plan before writing anything. Every rule below must hold:
   1. **Shape.** `phases`, `requirements`, and `notes` are present. Each phase, stage,
      and task has every field that `schemas/plan.schema.json` requires, as listed in
      the records reference.
   2. **Identifiers.** Phase, stage, and task identifiers follow the patterns in the
      records reference, are unique, and continue from the highest existing number.
   3. **Order.** `order` values are unique within their parent and start at 1.
   4. **First phase.** The phase with order 1 produces the pack's required first result.
      For the Software Pack it installs, starts, and completes one real user action.
   5. **Outcomes.** Every phase and stage title and outcome names a result the user can
      recognize. Reject wording that describes model activity, such as "run the
      evaluator", "ask Codex", or "review with Claude".
   6. **Task scope.** Every task is one small assignment for one producer call. Its
      `scope` says what it must produce and what it must not touch, in at most 1,000
      characters; detail that does not fit belongs in `acceptance` or `inputs`. Titles
      stay within 120 characters and outcomes within 500. A task that changes more than
      about five files or crosses several components must be split.
   7. **Allowed paths.** Every task has at least one glob pattern using forward slashes
      and only `*`, `**`, and `?` as wildcards. Reject a pattern that is absolute, starts
      with `/`, `~`, or a drive letter, contains `..` or `\`, is `**` alone or otherwise
      matches the whole repository, or matches `AGENTS.md`, `.git/**`, credential files
      such as `.env` or `*.pem`, or `.peerfoil/**` unless the task's purpose is a project
      record. Matching is case-insensitive so Windows and macOS cannot bypass a
      protected name. When a task is produced, every changed path is rejected if any
      component of it is a symbolic link or junction, and its resolved form must stay
      inside the repository root; that check belongs to production in a later build.
   8. **Evidence.** Every entry in `required_evidence` names an item in the Quality
      Contract with the same `kind`, and its `level` is `required` or `recommended`,
      never `not-applicable`.
   9. **Coverage.** Every `required` item in the Quality Contract appears in the
      `required_evidence` of at least one task in the first phase. Evidence produced at
      a phase boundary, such as a human user journey or a license check, is attached to
      the last task of that phase.
   10. **Dependencies.** Every `depends_on` entry names an existing task that comes
       earlier in the complete delivery sequence, ordered by phase `order`, then stage
       `order`, then position within the stage, or an earlier task in the same stage; the
       graph has no cycle. Stage `depends_on` entries name existing stages earlier in the
       same sequence.
   11. **Acceptance.** Every task has at least one acceptance statement that a person or
       a command can check.
   12. **Revisions.** Every task carries `plan_revision` equal to the draft's revision
       and `architecture_revision` equal to the accepted architecture revision.
       `author_role` is `producer` and `status` is `planned`.
   13. **Requirements.** Every requirement cites a decision identifier or an
       architecture section as `source` and lists at least one task. Every decision in
       the architecture's `Decisions applied` line is the source of at least one
       requirement, or the planner's `notes` say why it creates no work.
   14. **Backlog.** The first plan's backlog is empty.
   Ask the planner once to fix a specific rule; if it still fails, stop and tell the user
   which rule failed.
3. Write `.peerfoil/plan.json` with `record_type` `plan`, `schema_version` 1, the
   project identifier, `plan_revision` equal to `revisions.plan + 1`, the accepted
   `architecture_revision` and `quality_revision`, status `draft`, `revised_at` now,
   `revised_by` the planner actor, `revision_reason` "Initial plan written from
   architecture revision N." or the revision's reason, the validated phases and
   requirements, and an empty backlog.
4. Regenerate `.peerfoil/plan.md` from `plan.json` using
   `${CLAUDE_PLUGIN_ROOT}/templates/plan.md`. `plan.json` is authoritative.
5. Update `updated_at` in `project.json`. Leave `revisions.plan` unchanged until
   approval.
6. Tell the user, in plain language, the phases and stages in order and how many tasks
   the first phase holds.

## 4. Review

Follow [`review.md`](review.md) with kind `plan`. When the decision is `approve`, set
`plan.json` status to `reviewed`, regenerate `plan.md`, and note the review identifier in
the change history table.

## 5. Revising after findings

1. Build the packet from section 2 with the current plan and the open findings.
2. Rerun the planner. It must address every `blocking` and `major` finding and say in its
   `notes` how each finding identifier was handled.
3. Validate as in section 3 and rewrite `plan.json` at the same revision number with
   status `draft`, a new `revised_at`, and a `revision_reason` naming the review.
4. Only after `plan.json` is written, set each addressed finding's disposition to
   `repaired` in the review record. The next pass number is that review's pass plus one.
5. Return to section 4 for the next pass, within the limit in `review.md`, section 10.

## 6. Stage-order approval

Show the user a table of phases and stages in order with their outcomes, followed by
the review outcome and any open `minor` or `note` findings. Then ask with
`AskUserQuestion`:

1. **Approve this order.** Offered only when no `blocking` finding is open. Recommended
   when the review approved it and no `major` finding is open. Approving with an open
   `major` finding records it as `deferred` with the user's reason in the review.
2. **Revise first.** Recommended when the review left a `major` finding open. Return to
   section 5 with the open findings; the revised plan is reviewed again within the pass
   limit.
3. **Reorder, split, remove, or reprioritize a stage.** Ask what. Apply a pure reorder
   yourself by changing `order` values and confirm that every `depends_on` still points
   to earlier work; a reorder needs no new review. Any split, removal, or new stage goes
   back to the planner as a user-raised `major` finding recorded in the latest review,
   and the revised plan is reviewed again.
4. **Stop for now.** Leave the files as they are and end with the status report.

## 7. Recording approval

On approval:

1. Set `plan.json` status to `accepted`, regenerate `plan.md`, and set open `minor` and
   `note` findings in the latest review to `deferred`.
2. Set `revisions.plan` in `project.json` to the accepted plan revision, set
   `workflow.phase` and `workflow.stage` to the identifiers with order 1, leave
   `workflow.task` `null`, keep `workflow.state` `plan`, and update `updated_at`.
3. Append a transition with `from_state` `plan` and `to_state` `plan`, actor
   `coordinator`, `refs.reviews` listing the plan review identifiers, and the summary
   "Plan revision N approved after independent review; production waits for a later
   build." When the review was a reduced-assurance review, say so in the summary. Use
   only the `refs` keys the records reference lists; deferred findings are recorded in
   the review, not in `refs`.

## 8. The production boundary

Production is not available in this build. After approval, tell the user:

- the plan is approved and lives in `.peerfoil/plan.json` and `.peerfoil/plan.md`;
- the completed workflow will delegate the first task to Codex, capture its authorship,
  and run the declared evidence, as listed as "Not yet" in the workflow reference,
  section 7; and
- `/peerfoil:status` shows the approved plan, and `/peerfoil:change` will place new
  requests into it in a later build.

Never begin a production task, and never move `workflow.state` to `produce`. Production
also stays blocked whenever a decision is `open` or a `blocking` finding remains `open`.

## 9. Rules

- The planner proposes; the coordinator validates and writes. Do not rewrite the
  planner's substance beyond the validated structure and the user's own changes.
- The planner's session never reviews its own plan.
- `plan.json` is authoritative; regenerate `plan.md` after every change.
- Everything the user sees carries the label **Guided**.
- Work in few turns: read each reference once per chat, and write `plan.json`,
  `plan.md`, and the review record once per step with all of their changes instead of
  editing them field by field. Budget about ten coordinator turns to write the draft, ten
  per review pass, and five to record approval.
