<!--
This file is part of PeerFoil.
plugins/peerfoil/references/lessons.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Defines how a lesson becomes a candidate record, how it is verified, and how it is promoted without silently changing repository rules.
Notes: Used by the remember, review-phase, resume, and status skills. The product contract is docs/PeerFoil-Method.md, section 14.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# Lessons reference (Guided)

A lesson is something worth keeping after the chat ends. PeerFoil rewrites it as one
clear rule with a trigger and a scope, records the evidence behind it and any conflict
with existing rules, and keeps it as a candidate until a fresh reviewer verifies it.
Raw model memory never becomes project policy on its own, and PeerFoil never edits
`AGENTS.md`.

Read [records.md](records.md), the Lesson record, and [lineage.md](lineage.md),
section 3, first.

## 1. Recording a candidate

1. Rewrite the user's words as a rule: one imperative sentence a producer or reviewer
   can follow. Name the **trigger**, the situation in which it applies, and the
   **scope**: `this project`, `this pack`, or `proposed repository rule`.
2. Name the **evidence**. Prefer a record the repository holds: a finding, evidence,
   change set, review, or test path. When the user offers none, ask once; if nothing
   verifiable exists, record "user observation" and say that the lesson can only
   become a hint until evidence exists.
3. Check for **conflicts**: read `AGENTS.md`, `decisions.md`, the accepted
   architecture, and the lessons already recorded, and list any rule or decision the
   lesson contradicts, or "None".
4. Propose one **destination**:

   | Destination | Use when |
   |---|---|
   | `decision` | The lesson settles a choice the project must respect |
   | `test` | The lesson can be checked by a command or fixture |
   | `skill` | The lesson changes how a role should work |
   | `agents-md-proposal` | The lesson belongs in the repository's rules |
   | `pack-rule` | The lesson applies to every project of this pack |
   | `hint` | The lesson is useful for a while and needs no policy change |

5. Show the rewritten lesson and ask the user to confirm it with `AskUserQuestion`,
   recommended destination first.
6. Assign the next `ls-NNNN`, write `.peerfoil/lessons/ls-NNNN.md` from
   `${CLAUDE_PLUGIN_ROOT}/templates/lesson.md` with status `candidate`, `Verification`
   "none yet", `Promoted to` `null`, `Expires at` `null`, and `Recorded by` the user
   actor: role `user`, tool `human`, lineage `human`. Name the coordinator actor that
   rewrote the words in the Notes.
7. Append a same-state transition with `refs.lessons` and the summary "Lesson ls-NNNN
   recorded as a candidate." Update `updated_at`.

## 2. Verification

A candidate is verified only by a fresh reviewer, never by the session that recorded
it. The next phase review carries every candidate lesson in its bundle, and each
reviewer returns a verdict per lesson, as [phase-review.md](phase-review.md), section 5,
describes.

Each reviewer returns exactly one verdict per candidate; a review with a missing
verdict is rejected and retried, never read as agreement. Apply the verdicts in this
order of precedence:

1. `conflicts` from either reviewer: keep `candidate`, record the conflict, and tell
   the user, who may change a decision, narrow the lesson, or reject it.
2. Otherwise `unsupported` from either reviewer: keep `candidate` and record the
   reason; the user may add evidence, and the lesson is checked again at the next
   review.
3. Otherwise, `supported` from both reviewers: set status `verified` and write the
   review identifiers on the `Verification` line.

Both reviewers' verdicts count equally for a lesson. A person's lesson has no model
family to be protected from, and an objection from either family keeps it a
candidate.

## 3. Promotion

Only a `verified` lesson is promoted, except to `hint`. Ask the user before promoting;
then:

| Destination | What the coordinator writes |
|---|---|
| `decision` | A new answered decision in `decisions.md` with owner `user`; an affected accepted draft returns through [changes.md](changes.md) |
| `test` | A change request through [changes.md](changes.md) whose task adds the test |
| `skill` or `pack-rule` | The proposed text under a "Proposed text" heading in the lesson file; Skills 0.1 cannot edit the installed plugin, so the user applies it |
| `agents-md-proposal` | The proposed rule text under a "Proposed text" heading in the lesson file, for the user to add to `AGENTS.md` themselves |
| `hint` | `Expires at` set to the user's chosen time, thirty days by default |

Set status `promoted`, fill `Promoted to` with the decision, change request, or
destination name, and append a same-state transition with `refs.lessons`. A lesson the
user rejects, or that stays `unsupported` after the user declines to add evidence, is
set `rejected` with the reason.

A `hint` may be promoted from `candidate` when the user accepts that it is unverified.
It expires at `Expires at`; an expired hint is shown as expired and no longer used.

## 4. Using lessons

- Active hints, meaning lessons with status `promoted`, destination `hint`, and an
  `Expires at` in the future, are listed by name and rule in the producer packet of
  [production.md](production.md) and shown by `/peerfoil:status`.
- Promoted decisions and tests act through the records they created.
- A `candidate` lesson is never used as an instruction; it is only reviewed.

## 5. Rules

- Never create, edit, or replace `AGENTS.md`, a skill, or a pack. Proposals stay in
  the lesson file until a person applies them.
- Never store the raw chat, a credential, or personal data in a lesson. The rule,
  trigger, scope, and evidence reference are enough.
- The coordinator that rewrote a lesson does not verify it.
- Everything the user sees carries the label **Guided**.
