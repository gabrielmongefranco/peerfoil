<!--
This file is part of PeerFoil.
plugins/peerfoil/references/records.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Defines the shared PeerFoil records, their fields, identifiers, revisions, and file locations.
Notes: JSON schemas under schemas/ at the repository root encode the same rules for validated files.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# PeerFoil records reference

All PeerFoil skills, agents, templates, and packs use the same small set of records. This
file defines each record, the identifiers that connect them, and the file that holds
them. Use these names exactly so records do not drift between chats or releases.

## 1. Identifiers

Identifiers are lowercase, contain only letters, digits, and hyphens, and are safe in file
names and Git references. Sequenced identifiers use a type prefix and a zero-padded number
assigned in creation order. A number is never reused, even after an item is removed.

| Record | Prefix | Example | Pattern |
|---|---|---|---|
| Project | none | `weather-cli` | `^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$` |
| Decision | `d-` | `d-0001` | `^d-[0-9]{4,}$` |
| Requirement | `r-` | `r-0001` | `^r-[0-9]{4,}$` |
| Phase | `ph-` | `ph-01` | `^ph-[0-9]{2,}$` |
| Stage | `st-` | `st-01` | `^st-[0-9]{2,}$` |
| Task | `tk-` | `tk-001` | `^tk-[0-9]{3,}$` |
| Change request | `cr-` | `cr-0001` | `^cr-[0-9]{4,}$` |
| Change set | `cs-` | `cs-0001` | `^cs-[0-9]{4,}$` |
| Evidence | `ev-` | `ev-0001` | `^ev-[0-9]{4,}$` |
| Finding | `fd-` | `fd-0001` | `^fd-[0-9]{4,}$` |
| Review | `rv-` | `rv-0001` | `^rv-[0-9]{4,}$` |
| Lesson | `ls-` | `ls-0001` | `^ls-[0-9]{4,}$` |
| Transition | `tr-` | `tr-0001` | `^tr-[0-9]{4,}$` |

Rules:

- The project identifier is proposed from the repository directory name, confirmed by the
  user at start, and never changed afterward.
- Phase, stage, and task numbers are project-wide sequences. They do not encode order.
  Order lives in the `order` field, so the user can reorder stages without renaming them.
- To assign the next number, read the highest existing number for that prefix across the
  project files and add one. Never reuse a number.
- Provider session identifiers are opaque strings recorded as given. They are not PeerFoil
  identifiers and are never tokens.

## 2. Revisions and timestamps

- `architecture_revision`, `quality_revision`, and `plan_revision` are integers. They start
  at `0` before the record exists, become `1` when the record is first accepted, and
  increase by one on every accepted change. The current accepted values live in
  `project.json` under `revisions`.
- A draft carries the next number after the last accepted revision. Every rewrite of that
  draft before acceptance keeps the same number; the review record of each pass carries a
  `pass` counter. `project.json` `revisions` changes only when the user accepts the draft.
- Every task records the `plan_revision` and `architecture_revision` that created it. A
  task whose recorded revisions are older than the current ones is `stale` until the plan
  re-ties it.
- `source_revision` is the full Git commit hash of the checkout that a change set or
  evidence record describes. It is `null` until the work is tied to a commit.
- Timestamps use UTC in ISO 8601 form with second precision: `2026-09-05T14:30:00Z`.

## 3. Actor

Every authored record includes an actor. See [`lineage.md`](lineage.md) for the allowed
values and the independence rule.

```json
{
  "role": "coordinator",
  "tool": "claude-code",
  "model": "default",
  "effort": "high",
  "lineage_root": "anthropic-claude",
  "session": null
}
```

## 4. Records

### Project

File: `.peerfoil/project.json`. Validated by `schemas/project.schema.json`.

| Field | Content |
|---|---|
| `record_type` | Always `project` |
| `schema_version` | Integer, currently `1` |
| `project_id` | Project identifier |
| `name` | Short display name |
| `goal` | Plain-language summary of what the user wants, two to four sentences |
| `pack` | `{ "id": "software", "version": "0.1.0" }` |
| `profile` | `personal` or `work` |
| `rules_source` | `{ "mode": "inherit", "path": "AGENTS.md" }` or `{ "mode": "none" }` |
| `assurance` | `guided` in this release; `enforced` is reserved for Core |
| `release` | The PeerFoil release that created the record, such as `peerfoil-skills/0.1.0-dev` |
| `created_at`, `updated_at` | Timestamps |
| `workflow` | `{ "state", "phase", "stage", "task", "paused_for" }` |
| `revisions` | `{ "architecture", "quality", "plan" }` |
| `environment` | Optional setup results: `checked_at` and a list of checks |
| `settings` | Accepted Advanced settings: role seats and review limits |

Setting values:

- `settings.roles.<role>`: `{ "tool": "claude-code" | "codex-cli", "model": "<id>" | "default", "effort": "low" | "medium" | "high" | "xhigh" }` for `evaluator`, `architect`, `planner`, `change_steward`, `producer`, and `repair_producer`.
- `settings.phase_reviewers`: a list of exactly two seats with the same shape. The two
  seats must use different tools so the two families stay separate.
- `settings.review`: `default_passes` (default 6), `max_passes` (default 8, never above 8),
  `repair_selection_passes` (default 3), `repair_selection_max_passes` (default 4), and
  `repair_cycles` (default 1).

### Decision

File: `.peerfoil/decisions.md`, one section per decision.

| Field | Content |
|---|---|
| `id` | Decision identifier |
| `question` | One plain-language question |
| `category` | `behavior`, `cost`, `privacy`, `ownership`, `portability`, `compatibility`, `deployment`, `data`, or `other` |
| `needs_answer` | `yes` when the user must answer; `no` when a reversible assumption is allowed |
| `options` | Two to four realistic choices, each with one line of consequence |
| `recommended` | The recommended option and its reason |
| `effect` | What changes if the user chooses something else |
| `answer` | The chosen option or the assumption taken |
| `status` | `open`, `answered`, `assumed`, or `superseded` |
| `owner` | `user` for answers; the role that took an assumption |
| `decided_at` | Timestamp when answered or assumed |

### Architecture

File: `.peerfoil/architecture.md`.

| Field | Content |
|---|---|
| `revision` | Architecture revision |
| `status` | `draft`, `reviewed`, `accepted`, or `superseded` |
| `author` | Actor record for the architect, including its session |
| `written_at` | Timestamp |
| `decisions_applied` | Decision identifiers the architecture relies on |
| `quality_contract` | The Quality Contract revision written with this architecture |
| `reviews` | Review identifiers, or none yet |
| Sections | `Goals`, `Users`, `Boundaries`, `Dependencies`, `Data`, `Risks`, `Accessibility`, `Security`, `Privacy`, `Licensing`, and `Open decisions`, in that order |

Every section holds substantive text or an explicit "None" with the reason. The
`Dependencies` table names each dependency's license. The `Open decisions` section lists
assumed and open decision identifiers.

### Quality Contract

File: `.peerfoil/quality.md`.

| Field | Content |
|---|---|
| `revision` | Quality revision |
| `status` | `draft`, `reviewed`, `accepted`, or `superseded` |
| `author` | Actor record for the architect |
| `written_at` | Timestamp |
| `pack` | Pack identifier and version the contract was selected from |
| `architecture_revision` | Architecture revision the contract belongs to |
| `evidence` | Every evidence item declared by the pack, listed exactly once under `Required evidence`, `Recommended evidence`, or `Not applicable` |
| `review_lenses` | The pack's lenses plus at most two project-specific lenses the user approved |
| `completion` | The pack's completion requirements plus any project-specific requirement |

Each evidence item has a `name` (as declared by the pack), a `kind` (`executable`,
`inspection`, or `human`), a `level` (`required`, `recommended`, or `not-applicable`),
and a procedure. An executable procedure is a command written as an argument list with
its working directory. An inspection or human procedure is the list of steps and the
expected result. A pack item whose default level is `required` may become
`not-applicable` only with a stated reason; it is never lowered to `recommended`.

### Plan

Files: `.peerfoil/plan.json` (validated by `schemas/plan.schema.json`) and
`.peerfoil/plan.md` (the readable view). `plan.json` is authoritative.

| Field | Content |
|---|---|
| `plan_revision`, `architecture_revision`, `quality_revision` | Revisions this plan was written against |
| `status` | `draft`, `reviewed`, `accepted`, or `superseded` |
| `revised_at`, `revised_by`, `revision_reason` | Who changed the plan, when, and why |
| `phases[]` | Phase records with `id`, `title`, `outcome`, `order`, `status`, and `stages[]` |
| `stages[]` | Stage records with `id`, `title`, `outcome`, `order`, `status`, `depends_on`, and `tasks[]` |
| `tasks[]` | Task records, defined below |
| `requirements[]` | Optional `{ "id", "text", "source", "tasks" }` links from requirements to tasks |
| `backlog[]` | Change requests placed in the backlog |

Phase and stage status values: `planned`, `active`, `review`, `approved`, `deferred`,
`removed`.

### Task

Inside `plan.json`.

| Field | Content |
|---|---|
| `id` | Task identifier |
| `title` | Short outcome-focused title |
| `scope` | What the task must produce and what it must not touch |
| `allowed_paths` | Glob patterns the producer may change |
| `inputs` | Records or files the producer needs |
| `output` | The expected artifact or change |
| `required_evidence` | List of `{ "name", "kind", "level" }` matching the Quality Contract |
| `depends_on` | Task identifiers that must be accepted first |
| `acceptance` | Completion criteria in plain language |
| `author_role` | Normally `producer` |
| `status` | `planned`, `ready`, `in_progress`, `produced`, `validated`, `accepted`, `stale`, `blocked`, `deferred`, or `removed` |
| `plan_revision`, `architecture_revision` | Revisions that created the task |

### Change set

File: `.peerfoil/evidence/cs-NNNN.md`.

| Field | Content |
|---|---|
| `id` | Change set identifier |
| `task` | Task identifier |
| `base_revision` | Git commit the producer started from |
| `source_revision` | Git commit that contains the produced change, or `null` if uncommitted |
| `changed_paths` | Added, modified, deleted, and renamed paths |
| `author` | Actor record for the producer, including its session |
| `patch` | Location or hash of the captured patch |
| `summary` | What changed, in plain language |
| `captured_at` | Timestamp taken before any other agent edited the work |

### Evidence

File: `.peerfoil/evidence/ev-NNNN.md`.

| Field | Content |
|---|---|
| `id` | Evidence identifier |
| `name`, `kind`, `level` | As declared in the Quality Contract |
| `task`, `change_set` | What the evidence supports |
| `procedure` | Command as an argument list with its working directory, or the inspection or human steps |
| `result` | `pass`, `fail`, `blocked`, or `not-run` |
| `exit_code`, `duration_seconds`, `tool_version` | For executable evidence |
| `source_revision`, `plan_revision` | The exact revision checked |
| `recorded_at`, `recorded_by` | Timestamp and actor |
| `retained_output` | Path to redacted output or a short redacted excerpt |

In this release the coordinating host runs commands and the coordinator records the
result. A model's statement that a check passed is a claim until the output is retained.

### Finding

Inside a review record.

| Field | Content |
|---|---|
| `id` | Finding identifier |
| `location` | File, section, or path |
| `requirement` | The rule, decision, or contract item it concerns |
| `severity` | `blocking`, `major`, `minor`, or `note` |
| `evidence` | What the reviewer observed |
| `recommendation` | The proposed action |
| `disposition` | `open`, `accepted`, `repaired`, `declined`, or `deferred` |
| `raised_by` | Actor record |

### Review

File: `.peerfoil/reviews/rv-NNNN.md`.

| Field | Content |
|---|---|
| `id` | Review identifier |
| `kind` | `architecture`, `plan`, `change`, `phase`, or `repair` |
| `frozen` | `source_revision`, `plan_revision`, `architecture_revision`, and `quality_revision` reviewed |
| `pass` | Review pass number for this artifact draft, starting at 1 |
| `reviewer` | Actor record |
| `author` | Actor record of the author under review |
| `independence` | `independent`, `secondary`, or `reduced` |
| `passes_used` | Number of review passes |
| `findings` | Finding records |
| `decision` | `approve`, `repair`, `block`, or `undecided` |
| `remaining_risk` | Risks the reviewer accepts or flags |
| `reviewed_at` | Timestamp |

`approve` means no `blocking` finding remains. `repair` means the findings are specific
and a revision by the author can resolve them. `block` means a decision or requirement
must change first, so the user decides. The review transfer, the fallback when no
different-family reviewer is available, and the pass limits are defined in
[`review.md`](review.md).

### Lesson

File: `.peerfoil/lessons/ls-NNNN.md`.

| Field | Content |
|---|---|
| `id` | Lesson identifier |
| `trigger` | The situation in which the lesson applies |
| `scope` | Where it applies: this project, this pack, or a proposed repository rule |
| `rule` | The lesson stated as a clear instruction |
| `evidence` | What showed the lesson is true |
| `conflicts` | Existing rules or decisions it may contradict |
| `destination` | `decision`, `test`, `skill`, `agents-md-proposal`, `pack-rule`, or `hint` |
| `status` | `candidate`, `verified`, `promoted`, or `rejected` |
| `expires_at` | Required for `hint`; otherwise `null` |
| `recorded_by` | Actor record |

### Transition

File: `.peerfoil/history.jsonl`, one JSON object per line. Validated by
`schemas/transition.schema.json`.

| Field | Content |
|---|---|
| `record_type` | Always `transition` |
| `schema_version` | Integer, currently `1` |
| `transition_id` | Transition identifier |
| `project_id` | Project identifier |
| `at` | Timestamp |
| `from_state`, `to_state` | Workflow states; `from_state` is `null` at project start. An accepted record that does not change the state, such as plan approval in a build without production, is recorded with the same state on both sides |
| `actor` | Actor record for the coordinator that recorded the move |
| `plan_revision` | Plan revision in effect |
| `source_revision` | Git commit hash or `null` |
| `refs` | Optional identifiers involved, using only the keys `decisions`, `tasks`, `change_sets`, `evidence`, `reviews`, and `lessons`, for example `{ "decisions": ["d-0001"] }` |
| `summary` | One redacted sentence |

History lines never contain prompts, transcripts, tokens, or personal data.

## 5. Writing rules

- Copy the matching template from `${CLAUDE_PLUGIN_ROOT}/templates/` and fill every
  field. Do not remove fields. Use `null` or `not-applicable` where a value does not apply.
- Generated project files belong to the user. Do not copy PeerFoil's own file header into
  them. Each generated Markdown file starts with a one-line comment naming the PeerFoil
  release that generated it.
- Write only inside the repository root that contains `.peerfoil/`.
- Update `project.json` `updated_at` whenever any accepted record changes.
