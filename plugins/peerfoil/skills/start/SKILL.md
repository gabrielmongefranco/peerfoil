---
name: start
description: This skill should be used when the user runs /peerfoil:start or asks to start a PeerFoil project for something they want to build or change. It records the goal in plain language, chooses the project pack and profile, creates the .peerfoil project records inside the repository, runs the decision interview with the evaluator role, and then continues through the architecture, Quality Contract, and plan with independent review and the user's acceptance.
argument-hint: "[what you want to build or change]"
license: GPL-3.0-or-later
disable-model-invocation: true
allowed-tools: Read, Glob, Grep, Bash(git rev-parse *), Bash(git status *), Bash(git log *), Bash(date -u *), PowerShell(git rev-parse *), PowerShell(git status *), PowerShell(git log *), PowerShell(Get-Date *)
---
<!--
This file is part of PeerFoil.
plugins/peerfoil/skills/start/SKILL.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Guides project start: goal, pack, profile, project records, the decision interview, and the architecture and plan steps.
Notes: Assurance is Guided. Production arrives in a later build; see references/workflow.md section 7.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# PeerFoil start (Guided)

Request from the user: $ARGUMENTS

Turn the request into a recorded project with a plain-language goal, a project pack, a
profile, and a list of important decisions, then into a reviewed architecture and plan
the user accepts. Create only the documented project files, inside the repository, and
label everything **Guided**.

## Read first, in this order

1. `${CLAUDE_PLUGIN_ROOT}/references/workflow.md` in full.
2. `${CLAUDE_PLUGIN_ROOT}/references/records.md`, sections 1 to 3 and section 4 in
   full.
3. `${CLAUDE_PLUGIN_ROOT}/references/lineage.md`, section 3.
4. The repository root from `git rev-parse --show-toplevel`. If the command fails, tell
   the user to run `/peerfoil:setup` and stop.
5. `AGENTS.md` at the repository root, before any other project content. Follow it. If it
   does not exist, continue and note that the project has no repository instructions.
6. `README.md` at the repository root, if it exists, to learn what the project is.
7. `.peerfoil/project.json`, if it exists:
   - When `workflow.state` is `define`, the project already exists. Do not create a
     second one. Tell the user, then continue at step 6 using the new request as added
     information, or suggest `/peerfoil:resume` if the request adds nothing new.
   - When the state is anything else, say that start cannot run twice for one project,
     point to `/peerfoil:status` and `/peerfoil:resume`, and stop.

## Step 1 — Goal

If the request is empty, ask the user for one or two sentences about what they want.
Write the goal in plain language: two to four sentences saying what will be built or
changed, who it is for, and what a finished first phase looks like. Show it and ask the
user to confirm or correct it.

## Step 2 — Project pack

Choose one pack and give a one-line reason:

- `software` for code, applications, services, libraries, and command-line tools. This is
  the default.
- `documentation` for a Markdown document such as a guide, reference, or article.
- `generic` for anything else. Ask the user to name the deliverable.

Confirm the choice. Then read `${CLAUDE_PLUGIN_ROOT}/packs/<pack id>/pack.json` and tell
the user, in one short list, the pack's typical stages and its required evidence.

## Step 3 — Profile and rules source

If `/peerfoil:setup` already recorded a profile in this session, reuse it. Otherwise ask
whether the project is `personal` or `work`. Both profiles follow the repository's own
`AGENTS.md`. Record `rules_source` as `{ "mode": "inherit", "path": "AGENTS.md" }` when
that file exists and `{ "mode": "none" }` when it does not. Never create or edit
`AGENTS.md`.

## Step 4 — Project identifier and name

Propose the identifier from the repository directory name: lowercase it, replace every
run of characters other than `a-z` and `0-9` with one hyphen, trim hyphens from both ends,
and keep at most 64 characters. It must match `^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`.
Propose a short display name too. Confirm both. The identifier never changes afterward.

## Step 5 — Create the project records

Tell the user exactly which files will be created, all inside the repository root:

```text
.peerfoil/project.json
.peerfoil/decisions.md
.peerfoil/history.jsonl
```

Nothing else is created at this step. Then create them from the templates in
`${CLAUDE_PLUGIN_ROOT}/templates/`:

- `project.json`: fill every field. Use `assurance: "guided"`,
  `release: "peerfoil-skills/0.1.0-dev"`, `workflow.state: "define"` with the other
  workflow fields `null`, every revision `0`, and the template's default `settings`. Copy
  the `environment` results from `/peerfoil:setup` if it ran in this session; otherwise
  omit the `environment` field. Set `created_at` and `updated_at` to the current UTC time
  in ISO 8601 form with second precision. Get that time from `date -u +%Y-%m-%dT%H:%M:%SZ`
  in a POSIX shell, or `Get-Date -AsUTC -Format "yyyy-MM-ddTHH:mm:ssZ"` in PowerShell.
- `decisions.md`: use the template's first comment line and title, replace the
  placeholders, remove the example decision section, and leave the summary table with the
  single line "No decisions yet." under it. Decisions are added in step 6.
- `history.jsonl`: one line, transition `tr-0001`, `from_state` `null`, `to_state`
  `define`, actor `coordinator` with `tool` `claude-code`, `lineage_root`
  `anthropic-claude`, and `session` `null`, `plan_revision` `0`, `source_revision` `null`,
  and the summary "Project started; decision interview begins."

Replace every template placeholder. Do not copy PeerFoil's own file header into these
files; the one-line generated-by comment at the top of each Markdown template is the only
header they carry.

## Step 6 — Decision interview

Build a compact packet for the evaluator. Include only:

- the confirmed goal;
- the pack identifier and its typical stages;
- the profile;
- a summary of at most fifteen lines of the `AGENTS.md` rules that constrain decisions,
  or "none" when the file does not exist;
- a summary of at most ten lines of the README, or "none";
- repository facts you can observe: language markers such as `go.mod` or `package.json`,
  whether tests exist, and whether the tree is clean; and
- the decisions already recorded in `decisions.md`, when any exist.

Do not include this chat's history. Launch the `peerfoil:evaluator` agent with the packet
and ask it for the decision list in the format its instructions define.

Validate every returned item before recording it. Each item must have a question, two to
four options with one-line consequences, a recommended option with a reason, an effect
statement, a `needs_answer` value, and a category from the records reference. Ask the
agent once to fix a malformed item. If it is still malformed and its `needs_answer` is
`true` or its category is `privacy`, `data`, `cost`, `ownership`, or `deployment`,
record it anyway as an `open` decision with the question text and the note "raised by
the evaluator in an incomplete form; needs rewording", so it keeps blocking
architecture until the user answers it. Such an item is recorded with `needs_answer`
`yes` whatever the evaluator said, so it is never turned into an assumption. Drop only a
malformed item that is a reversible assumption outside those categories, and tell the
user.

Record the decisions:

1. Assign identifiers in the returned order, continuing from the highest existing
   `d-NNNN` in `decisions.md`.
2. Write every item to `decisions.md` in the template's section shape and add a row to
   the summary table.
3. Items with `needs_answer: no` become `assumed`: the answer is the recommended option,
   the owner is `evaluator`, and `decided_at` is now. Tell the user these assumptions are
   visible and reversible.
4. Ask the user each `needs_answer: yes` item one at a time with `AskUserQuestion`, the
   recommended option first and labeled as recommended. Record each answer as `answered`
   with owner `user` and `decided_at` now. If the user declines to answer, leave it
   `open`.
5. After all answers, run the evaluator once more with the updated packet to check for
   new consequential questions. Repeat at most three rounds, or stop earlier when the
   agent returns no new item or the user asks to stop.
6. Update `updated_at` in `project.json`. The workflow state stays `define`.

## Step 7 — Architecture and plan

Skip this step while any decision is `open`. Otherwise continue in this chat:

1. Read `${CLAUDE_PLUGIN_ROOT}/references/architecture.md`,
   `${CLAUDE_PLUGIN_ROOT}/references/review.md`, and
   `${CLAUDE_PLUGIN_ROOT}/references/planning.md` in full.
2. Follow the architecture reference from its section 1: enter the `architect` state,
   write the draft with the `peerfoil:architect` agent, obtain the independent review,
   and ask the user to accept the architecture.
3. After acceptance, follow the planning reference from its section 2: write the plan
   with the `peerfoil:planner` agent, obtain the independent review, and ask the user to
   approve the stage order.
4. Stop wherever a reference says to stop or pause. The user continues later with
   `/peerfoil:resume`, in this chat or a fresh one.

## Step 8 — Report

Show the user, in plain language:

- the goal, pack, profile, project identifier, and display name;
- how many decisions are answered, assumed, and open;
- the architecture and plan: revision, status, and the review's independence, or
  "Reduced assurance" when the user accepted it;
- the files written; and
- `Assurance: Guided`.

Then state the boundary honestly. When the plan is approved, say that production, the
next step, is not yet available in this build of PeerFoil Skills, as listed in the
workflow reference, section 7. When decisions remain open or a step stopped, say that
`/peerfoil:resume` continues from the files. In every case say that a fresh chat can
resume from the files under `.peerfoil/` without this conversation.
