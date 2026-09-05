<!--
This file is part of PeerFoil.
plugins/peerfoil/references/phase-review.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Defines the phase review: freezing the bundle, running two independent reviewers, merging findings without hiding disagreement, deciding, and recording approval or a repair.
Notes: Used by the review-phase, resume, and status skills. The review transfer itself is in review.md; the repair cycle is in repair.md. The product contract is docs/PeerFoil-Method.md, sections 7 and 8.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# Phase review reference (Guided)

A phase review gives one fresh Claude reviewer and one fresh Codex reviewer the same
frozen material and no producer transcript. Each reviews independently first. The
coordinating skill then merges their findings into one shared list, lets each reviewer
compare, and decides from the rules below. The coordinator never approves anything by
its own judgment, and no reviewer approves work its own family authored.

Read [review.md](review.md), [lineage.md](lineage.md), [records.md](records.md), and
[evidence.md](evidence.md) first. This is a written host procedure. Skills 0.1 states
the pass limits and independence rules; it cannot enforce them mechanically.

## 1. Entering the step

Enter only when all of these are recorded:

1. `workflow.state` is `review` and the active phase has status `review`.
2. Every task in the phase is `validated`, or `accepted` from an earlier round, and no
   change set has `Capture status: pending`.
3. Every required evidence record of every task in the phase has result `pass` and its
   `Input snapshot` still matches the current files. Recompute the snapshot as
   [evidence.md](evidence.md) describes before freezing. Stale or missing required
   evidence returns the work to production through [changes.md](changes.md); it is
   never reviewed as current.
4. No decision is `open`, no plan candidate is `draft`, and the accepted architecture,
   quality, and plan revisions match `project.json`.
5. No phase review record for this phase has status `open` or `repair`. When one does,
   continue it from section 9 instead of starting another round.

If any condition fails, report the exact mismatch and one recovery action, and stop.

## 2. Freeze the bundle

1. Run the UTC time command and `git rev-parse HEAD`. Record `source_revision` only
   when `git status --porcelain` shows no change to any bundle item; otherwise record
   `null` and note that the reviewers read uncommitted working-tree files.
2. List the bundle items. Each item gets a stable identifier `item-NN` within the record:
   - the records: `.peerfoil/decisions.md`, `architecture.md`, `quality.md`,
     `plan.json`, and every retained snapshot under `.peerfoil/plans/`;
   - every change set of the phase and its captured patch under `.peerfoil/evidence/`;
   - every deliverable path named in those change sets, at its current content;
   - every evidence record of the phase;
   - every review record of the phase's architecture, plan, and changes; and
   - every lesson under `.peerfoil/lessons/` with status `candidate`.
3. Hash every item's raw bytes with SHA-256 and compute the bundle digest with the
   snapshot recipe in [evidence.md](evidence.md): sort by UTF-8 path bytes and hash the
   compact JSON of `[path, kind, sha256]` rows.
4. Record each item's author from the record itself: the `Author` line of a record, the
   `Author` line of a change set for its patch and deliverables, the `Recorded by` line
   of evidence and lessons, and `coordinator` for `decisions.md` and `plan.json`. Map
   each author to its `lineage_root` with [lineage.md](lineage.md).
5. Assign each item's **primary reviewer**: the seat in `settings.phase_reviewers`
   whose configured model maps to a known lineage root that differs from the item's
   author's known lineage root. A seat whose lineage is `unknown` is never eligible
   as primary, and an item whose author lineage is `unknown` has no eligible primary.
   An item authored by `human` takes the first eligible seat, because every model
   family differs from a person. When no seat qualifies, write `none` and follow
   section 4.
6. Collect the open items the reviewers must see: `changes[]` entries of the plan with
   placement `backlog` or `declined`, findings with disposition `deferred` in every
   review of the phase, tasks with status `deferred`, `blocked`, or `removed`, and any
   `Notes` in evidence records that record a skipped or limited check.
7. Collect the tool versions from the phase's evidence records.
8. Assign the next `pr-NNNN`, write `.peerfoil/reviews/pr-NNNN.md` from
   `${CLAUDE_PLUGIN_ROOT}/templates/phase-review.md` with status `open`, both reviewer
   seats, and every section above filled. The round is 1 for the phase's first record;
   a later record takes the previous record's round plus one and starts each
   reviewer's passes used at the count the previous record reached, because the
   limits apply to the phase, not to the record. Append a same-state
   transition `review` to `review` with `refs.phase_reviews` and the summary "Phase
   ph-NN frozen for review as pr-NNNN." Update `updated_at`.

Do not edit any bundle item while the review runs. Until a repair is recorded, a
changed digest invalidates the round: record it in the phase review record, set its
status `closed-without-approval`, and start a new round from section 1. Once the
record's status is `repair`, the repaired files are expected to differ, and
[repair.md](repair.md) binds the verification to the repair digest instead.

## 3. Reviewer availability

- The Claude seat runs as a fresh `peerfoil:claude-reviewer` agent. It is available
  whenever the skill runs.
- The Codex seat is available under the rules in [review.md](review.md), section 2,
  through the MCP tool or the `codex exec` fallback.
- The lineage of each seat comes from its configured model identifier. A seat left at
  `default` has `unknown` lineage; its review is `reduced` and cannot give primary
  approval. `/peerfoil:setup` records the Codex model so the seat is not `default`.

## 4. When an item has no eligible primary reviewer

When any item's primary reviewer is `none`, whether because the other family's seat
is unavailable, because a seat's lineage is `unknown`, or because the item's author
lineage is `unknown`, stop before launching anyone, list those items, and ask the user
with `AskUserQuestion`, recommending the first option:

1. **Wait for an independent reviewer (recommended).** Set `workflow.state` to `paused`
   with `paused_for` set to "Install the Codex CLI, sign in, and register its MCP server
   with /peerfoil:setup, then run /peerfoil:review-phase". Record the transition. The
   phase review record keeps status `open`.
2. **Accept Reduced assurance for the listed items.** Record the acceptance, the
   items, and the time in the phase review record's Decision section and in the next
   transition summary. The available reviewers review the whole bundle; their
   findings on the listed items are a secondary or reduced check, and the phase
   result shows "Reduced assurance" for them.

The acceptance covers this round only and never turns a reviewer into a primary
reviewer.

## 5. The phase packet

Build the packet from the review packet template in [review.md](review.md), section 4,
with kind `phase` and these changes:

- Name the phase, the phase review identifier, the bundle digest, and the round and
  pass numbers in the "Frozen material" line.
- Replace the "Read these files" list with the bundle manifest: one line per item with
  its identifier, kind, and path, records first, then patches, deliverables, evidence,
  reviews, and lessons. Ask the reviewer to read `AGENTS.md` first, then the records in
  that order, then the rest.
- Use the phase focus list from section 6 and every lens from `quality.md`.
- Add: "For every evidence record, report whether its procedure matches the Quality
  Contract, its result, and its revisions agree with the change set it supports. Do
  not run commands; the host has run them. A record that claims a pass without
  retained output is a finding."
- Add: "For every candidate lesson, say whether the evidence it cites supports its
  rule and whether it conflicts with `AGENTS.md`, a decision, or the architecture."
- Add: "Give each finding the `item` identifier from the manifest and the `lens` it
  falls under."
- Add: "You are the primary reviewer for these items: <list>. You are a secondary
  reviewer for the rest, because their author is your own model family; review them
  fully, but your approval of them is not independent."
- Keep the ten-turn, ten-finding, and one-or-two-sentence limits.

The JSON block a phase reviewer returns:

```json
{
  "kind": "phase",
  "reviewed": { "phase": "ph-01", "phase_review": "pr-0001", "bundle_digest": "<sha256>", "plan_revision": 1, "architecture_revision": 1, "quality_revision": 1, "pass": 1 },
  "decision": "approve | repair | block",
  "findings": [
    { "title": "", "item": "item-03", "location": "", "requirement": "", "lens": "correctness-reliability", "severity": "blocking | major | minor | note", "evidence": "", "recommendation": "" }
  ],
  "evidence_review": [ { "evidence": "ev-0001", "verdict": "consistent | inconsistent | missing", "note": "" } ],
  "lessons": [ { "lesson": "ls-0001", "verdict": "supported | unsupported | conflicts", "note": "" } ],
  "remaining_risk": "",
  "model": "",
  "notes": ""
}
```

Validate it with [review.md](review.md), section 8, plus: every `reviewed` value equals
the frozen material, every `item` exists in the manifest, every `lens` is a lens in
`quality.md`, every `evidence` and `lesson` identifier exists in the bundle, every
evidence record and every candidate lesson in the bundle has exactly one verdict, and
every verdict is one of the listed values. Reject and retry otherwise; a missing
verdict is never read as agreement.

## 6. Phase focus list

- **Correctness and reliability.** Every accepted requirement of the phase traces to a
  validated task; every acceptance criterion has evidence; failure paths named in the
  architecture are handled; no producer claim stands in for a host-run result.
- **Security and privacy.** Inputs are validated at the boundaries the architecture
  names; commands and queries are parameterized; no credential, token, or personal
  data appears in deliverables, patches, evidence, or records; permissions were not
  widened.
- **Accessibility and user experience.** User-facing output meets the accessibility
  section of the architecture; status is not conveyed by color alone; errors name a
  next action; documentation matches shipped behavior.
- **Maintainability, documentation, licensing, and release readiness.** Headers,
  notices, and licenses are present and compatible; comments are current; no unused
  flag or placeholder remains; setup instructions are copyable; unavailable features
  are marked.

## 7. Pass 1: independent review

1. Launch the Claude reviewer with the packet. Record its agent identifier as the
   session, the UTC time before and after, and the duration.
2. Launch the Codex reviewer with the identical packet, as [review.md](review.md),
   section 6, describes. Neither reviewer sees the other's output or existence beyond
   the packet's statement that another reviewer exists.
3. Write one `rv-NNNN.md` per reviewer with kind `phase`, pass 1, one `fd-NNNN` per
   finding with disposition `open`, the `Item` and lens of each finding, the evidence
   and lesson verdicts in the review's Remaining risk section. The review's
   `Independence` line is `independent` and names the items for which this reviewer is
   primary and those for which it is secondary, taken from the manifest; it is
   `reduced` when the reviewer's lineage is `unknown`.
4. Add both review identifiers to the phase review record and increment each
   reviewer's passes used by one.

## 8. Merge into the shared list

1. Normalize each finding: trim the location, compare paths case-insensitively with
   forward slashes, and pair the `item` with the `requirement`.
2. Two findings from different reviewers describe one defect when their `item` and
   normalized location match and their requirement or recommendation says the same
   thing. Merge them into one row keyed by the lower finding identifier. The row keeps
   both identifiers, both severities, and both recommendations. Its working severity
   is the higher one.
3. A row raised by one reviewer only is marked `single`. A merged row whose severities
   differ is marked `disputed`. A merged row whose severities match is `agreed`.
4. Never drop a finding, and never rewrite a reviewer's words. Disagreement stays
   visible in the row.
5. Write the shared list into the phase review record's Shared findings section.

## 9. Pass 2: comparison

Skip this pass only when both reviewers returned no findings.

Launch each comparison as a fresh reviewer run, exactly like pass 1: a new
`peerfoil:claude-reviewer` agent and a new `mcp__codex__codex` thread or `codex exec`
run. Never continue the pass-1 session, message the earlier agent, or call
`codex-reply`; D-0023 forbids it, and a fresh session keeps the comparison honest.
Each reviewer receives the packet again with a "Comparison" section listing every row
of the shared list: the merged identifier, item, location, requirement, the reviewer's
own finding if any, and the other reviewer's title, severity, evidence, and
recommendation, named only as "the other reviewer". The reviewer returns, per row,
`agree`, `disagree` with a reason, or `withdraw` for its own finding, and no new
findings except a `blocking` finding on material changed since pass 1, which cannot
exist in this pass and is therefore recorded as `declined` under D-0024.

Apply the results:

- A `single` row withdrawn by its raiser becomes `declined` with the note "withdrawn
  by the reviewer in comparison".
- A row both reviewers `agree` on keeps its severity and stays `open`.
- A `disputed` or disagreed row keeps both positions. When either position is
  `blocking` and the reviewers do not agree, in either direction, the row is marked
  for the user; only a `blocking` row that both reviewers agree on can enter repair.
  When neither position is `blocking`, the row takes the severity given by the
  primary reviewer of its item.
- A comparison pass's `decision` is `approve` when the reviewer agrees that no
  `blocking` row remains and `repair` otherwise; a `block` that names no decision or
  requirement that must change is an invalid result, rejected and retried under
  [review.md](review.md), section 6, step 5.
- Record each reviewer's pass 2 as a new `rv-NNNN.md` with kind `phase`, pass 2, and
  its agreement table in place of new findings. Increment each reviewer's passes used
  by one.

## 10. Decide

Apply these rules in order and record the outcome in the Decision section:

1. Required evidence that the host found stale, missing, or failed at any point in the
   round blocks approval regardless of reviewer agreement. Return to production
   through [changes.md](changes.md). Reviewer consensus cannot clear it.
2. A row marked for the user, or a `blocking` row that the reviewers agree cannot be
   repaired by one bounded change, pauses the review. Set `workflow.state` to `paused`
   with `paused_for` naming the finding and the choices: repair, decline with a
   recorded reason, or change a decision. Do not invent consensus.
3. Any `open` `blocking` row with an eligible repair uses one repair cycle from
   [repair.md](repair.md) when `settings.review.repair_cycles` allows it and none has
   been used in this record. Otherwise pause as in rule 2.
4. When no `blocking` row is open, show the user every `open` `major` row and ask with
   `AskUserQuestion`: **Repair now (recommended)**, which uses the repair cycle, or
   **Defer**, which records each as `deferred` with the user's reason.
5. `minor` and `note` rows become `deferred`.
6. Every `deferred` row is handed to [changes.md](changes.md) as one change entry with
   placement `backlog` or `later-stage`, so the plan carries it. The phase is not
   approved until that revision is recorded.
7. When no `blocking` or `major` row is open, and the latest pass of every reviewer
   returned `approve` or only deferred and declined findings, approve as in section 11.
   Independence is `independent` when every item had an eligible primary reviewer;
   otherwise it is `reduced`, and the user's acceptance from section 4 is repeated in
   the summary.

After a verified repair, [repair.md](repair.md) returns here at rule 4 with the
repaired rows set `repaired`. Every remaining row is dispositioned and carried into
the plan before the phase is approved; a repair never approves a phase by itself.

## 11. Approve

1. Set every task in the phase to `accepted`, every stage and the phase to `approved`,
   and regenerate `plan.md`.
2. Set the phase review record's status to `approved` with the decision, the time, and
   the independence outcome.
3. Set `workflow.state` to `approve`, `workflow.task` to `null`, and `updated_at`.
4. Append a transition `review` to `approve` with `refs.reviews` listing every review
   of the round, `refs.phase_reviews`, and the summary "Phase ph-NN approved after
   independent review." or "...approved with Reduced assurance accepted by the user."
5. Tell the user the decision, each reviewer's independence per item in one line, the
   deferred findings by identifier, and that the next phase starts only when they run
   `/peerfoil:resume` and authorize it.

## 12. Pass accounting and limits

- Each reviewer's passes are counted in the phase review record and carried across
  rounds of the same phase: pass 1 is the independent review, pass 2 the comparison,
  and one further pass is reserved for verifying a repair. Every launch, including a
  repair verification, increments that reviewer's passes used before its result is
  recorded. `settings.review.default_passes` is six and `max_passes` eight. Before
  any launch, confirm the reviewer has a pass left below the maximum and that one
  pass remains available for verification until the round is decided without a
  repair. When a reviewer reaches the default without a decision, stop and ask the
  user whether to allow passes up to the maximum, decline the remaining findings with
  reasons, or stop. Never exceed the maximum.
- Every reviewer run keeps the ten-turn, ten-minute, and ten-finding limits from
  [review.md](review.md), section 11, and records its duration.
- The coordinator budgets about ten of its own turns to freeze the bundle, ten per
  reviewer pass, ten to merge and decide, and five to record the outcome.
- These limits are stated, not enforced; Core enforces them.

## 13. Resuming a phase review in a fresh chat

Read the newest phase review record for the active phase, not the chat:

When the newest record's status is `repair`, go to [repair.md](repair.md), section 7,
before anything else: the repair changed the bundle on purpose, and that reference
checks the repair change set's snapshot and the second manifest instead of the
original digest. Otherwise check the original bundle digest against the files first; a
mismatch closes the round. Then:

| Record shows | Do |
|---|---|
| No record, or only records with status `approved`, `paused`, or `closed-without-approval`, and state `review` | Section 1, as a new round |
| Status `open`, no reviews listed | Section 3 |
| Status `open`, one pass-1 review | Launch only the missing reviewer's pass 1 with the same packet; then section 8 |
| Status `open`, both pass-1 reviews, no shared list | Section 8 |
| Status `open`, shared list, no pass-2 review | Section 9 |
| Status `open`, shared list, one pass-2 review | Launch only the missing reviewer's comparison; then apply section 9 |
| Status `open`, both pass-2 reviews, dispositions not applied | Section 9, "Apply the results" |
| Status `open`, dispositions applied, no decision | Section 10 |
| Status `open`, decision recorded, deferred rows not yet in the plan | Section 10, rule 6, then section 11 |
| Status `repair` | [repair.md](repair.md), section 7 |
| Status `approved` | Nothing; report the approval |
| Status `paused` or `workflow.state` `paused` | Report `paused_for`; continue only when the need is met |
| Bundle digest no longer matches the files | Record the mismatch, set the status `closed-without-approval`, and start a new round from section 1 |

## 14. Rules

- Reviewers propose; the coordinator validates and records. The coordinator never
  softens, rewrites, or drops a finding.
- The session that authored an item never reviews it, and a same-family review of an
  item is `secondary` however good it is.
- A model's claim that a check passed is not evidence. The host's retained results and
  recomputed snapshots are.
- Everything the user sees carries the label **Guided**.
