<!--
This file is part of PeerFoil.
plugins/peerfoil/references/architecture.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Defines the architecture step: how a skill turns resolved decisions into an architecture and Quality Contract, obtains independent review, and records the user's acceptance.
Notes: Shared by the start and resume skills. The product contract is docs/PeerFoil-Method.md, sections 5 and 8.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# Architecture step reference

This step runs while `workflow.state` is `architect`. It produces
`.peerfoil/architecture.md` and `.peerfoil/quality.md`, has them reviewed by a different
model family, and ends when the user accepts them. The architect writes from the approved
decisions, never from the chat.

## 1. Entering the step

Enter from `define` only when `decisions.md` has no decision with status `open`. Then:

1. Set `workflow.state` to `architect` and `updated_at` in `project.json`.
2. Append a transition to `history.jsonl`: `from_state` `define`, `to_state`
   `architect`, actor `coordinator`, `refs.decisions` listing every decision identifier,
   and the summary "Decisions complete; architecture begins."
3. Tell the user that the architecture is being written from the recorded decisions.

## 2. Finding the current sub-step in a fresh chat

Read the files, not the chat:

| Files show | Do |
|---|---|
| No `architecture.md` | Section 3 |
| `architecture.md` status `draft`, no review at its revision and pass | Section 5 |
| Latest architecture review decision `repair` with an open `blocking` or `major` finding | Section 6 |
| Latest architecture review decision `repair`, no open `blocking` or `major` finding, and the draft's `Written at` later than that review's `Reviewed at` | Section 5, as the next pass |
| Latest architecture review decision `block` | Show the findings and the `paused_for` reason; stop |
| Latest architecture review decision `approve` whose frozen architecture and quality revisions equal the draft's and whose `Reviewed at` is later than the draft's `Written at`, while the draft's status is still `draft` | Set both files to `reviewed`, add the review identifier, then section 7 |
| `architecture.md` status `reviewed` | Section 7 |
| `architecture.md` status `accepted` | Section 8 already ran; continue with [`planning.md`](planning.md) |

## 3. The architect packet

Build a compact packet. Include only:

- the goal, project identifier, name, pack identifier and version, and profile;
- the pack's artifacts, typical stages, evidence items with their kinds, default levels,
  and descriptions, evidence hints, review lenses, and completion requirements, copied
  from `${CLAUDE_PLUGIN_ROOT}/packs/<pack id>/pack.json`;
- a summary of at most fifteen lines of the `AGENTS.md` rules that constrain the design,
  or "none";
- a summary of at most ten lines of the README, or "none";
- repository facts you can observe: toolchain markers such as `go.mod` or
  `package.json`, declared build or test scripts, whether tests exist, the top-level
  directory names, and whether the tree is clean;
- every decision from `decisions.md` with its identifier, question, answer, status, and
  effect; and
- when revising, the current draft of both files and every open finding from the latest
  review.

Do not include this chat's history or the evaluator's packet.

## 4. Writing the draft

1. Launch the `peerfoil:architect` agent with the packet. Record the agent identifier the
   `Agent` tool shows as the architect's `session`, or `null` when none is shown.
2. Validate the result before writing anything:
   - The Markdown block contains every section from
     `${CLAUDE_PLUGIN_ROOT}/templates/architecture.md`, in order, and no `{{` remains.
   - `decisions_applied` lists only identifiers that exist in `decisions.md`.
   - `quality.evidence` lists every pack evidence item exactly once and no other name.
     Each has a valid `kind` and `level`; a pack item whose default level is `required`
     is `required` or `not-applicable` with a `reason`; every `executable` item's
     procedure is an argument list plus a working directory; every other item's
     procedure lists steps and an expected result.
   - `quality.additional_lenses` holds at most two lenses, each with an identifier, a
     name, a focus, and a reason.
   - Each item in `new_questions` has the shape the evaluator uses.
   Ask the agent once to fix a specific problem; if it remains, stop and tell the user.
3. If `new_questions` contains an item with `needs_answer: true`, ask the user with
   `AskUserQuestion`, recommended option first, and record the answer in `decisions.md`
   with the next identifier, owner `user`, and status `answered`. Record items with
   `needs_answer: false` as `assumed` with owner `architect`. Then rerun the architect
   with the updated packet. If the user declines to answer, set `workflow.state` to
   `paused` with `paused_for` naming the decision, record the transition, and stop.
4. If `additional_lenses` is not empty, show each lens and ask the user to approve it.
   Keep only approved lenses.
5. Write `.peerfoil/architecture.md` from the template with revision
   `revisions.architecture + 1`, status `draft`, the architect actor, the current UTC
   time, `decisions_applied`, the Quality Contract revision, and "none yet" for reviews.
6. Write `.peerfoil/quality.md` from the template with revision `revisions.quality + 1`,
   status `draft`, the same actor and time, the pack, the architecture revision, the
   evidence split into the three tables, the pack lenses plus approved additional lenses,
   and the completion requirements.
7. Update `updated_at` in `project.json`. Leave `revisions` unchanged until acceptance.
8. Tell the user, in five lines or fewer, what the architecture proposes and which
   evidence is required.

## 5. Review

Follow [`review.md`](review.md) with kind `architecture`. When the decision is
`approve`, set the status of both files to `reviewed` and add the review identifier to
the `Reviews` line.

## 6. Revising after findings

1. Build the packet from section 3 with the current draft and the open findings.
2. Rerun the architect. It must address every `blocking` and `major` finding and say in
   its `notes` how each finding identifier was handled.
3. Validate as in section 4 and rewrite both files at the same revision number and
   status `draft`, with a new `Written at` time.
4. Only after both files are written, set each addressed finding's disposition to
   `repaired` in the review record. The next pass number is that review's pass plus one.
5. Return to section 5 for the next pass, within the limit in `review.md`, section 10.

## 7. Acceptance

Show the user, in plain language and under twenty-five lines:

- the goals and boundaries in two or three sentences;
- the top three risks and their responses;
- the required evidence names and the not-applicable items with reasons;
- the review outcome: decision, independence, and any open `minor` or `note` findings;
- the assumptions that remain reversible, by decision identifier.

Then ask with `AskUserQuestion`:

1. **Accept the architecture.** Offered only when no `blocking` finding is open.
   Recommended when the review approved it and no `major` finding is open. Accepting
   with an open `major` finding records it as `deferred` with the user's reason in the
   review.
2. **Revise first.** Recommended when the review left a `major` finding open. Return to
   section 6 with the open findings; the revised draft is reviewed again within the pass
   limit.
3. **Change something.** Ask what; treat the answer as a finding raised by the user with
   severity `major`, add it to the latest review record with `raised_by` the user actor,
   and return to section 6. A changed draft is reviewed again.
4. **Stop for now.** Leave the files as they are and end with the status report.

## 8. Recording acceptance

On acceptance:

1. Set the status of `architecture.md` and `quality.md` to `accepted`.
2. Set open `minor` and `note` findings in the latest review to `deferred`.
3. Set `revisions.architecture` and `revisions.quality` in `project.json` to the
   accepted revision numbers, set `workflow.state` to `plan`, and update `updated_at`.
4. Append a transition: `from_state` `architect`, `to_state` `plan`, actor
   `coordinator`, `refs.reviews` listing the review identifiers, and the summary
   "Architecture revision N accepted after independent review." When the review was a
   reduced-assurance review, say so in the summary. Use only the `refs` keys the records
   reference lists.
5. Continue with [`planning.md`](planning.md).

## 9. Rules

- The architect proposes; the coordinator validates and writes. The architect never
  writes files, and the coordinator never rewrites the architect's substance beyond the
  validated structure.
- The architect's session never reviews its own draft.
- Nothing in this step may lower a pack-required evidence item to recommended.
- Everything the user sees carries the label **Guided**.
- Work in few turns: read each reference once per chat, gather the facts you need with
  one command per fact, and write each record once per step with all of its changes
  instead of editing it field by field. Budget about ten coordinator turns to write the
  draft, ten per review pass, and five to record acceptance.
