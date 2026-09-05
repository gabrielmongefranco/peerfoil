<!--
This file is part of PeerFoil.
docs/plans/phase-1-skills.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Provides the executable implementation plan for Phase 1, PeerFoil Skills 0.1, and records stage status.
Notes: This plan expands Phase 1 of docs/implementation-plan.md without changing its scope.

Copyright © 2026 Gabriel Mongefranco

Permission is granted to copy, distribute and/or modify this document under the terms of
the GNU Free Documentation License, Version 1.3 or any later version published by the Free
Software Foundation; with no Invariant Sections, no Front-Cover Texts, and no Back-Cover
Texts. See <https://www.gnu.org/licenses/fdl-1.3.html>.
-->

# Phase 1 Implementation Plan: PeerFoil Skills 0.1

## Build the complete guided workflow before building the controller

[Return to the PeerFoil README](../../README.md)

Phase 1 delivers a useful PeerFoil workflow as a Claude Code plugin made mostly from
Markdown skills, role instructions, project packs, and templates. It reaches Codex
through the Codex CLI's built-in MCP server rather than building another Claude-to-Codex
bridge. A user
can complete the full workflow, but the interface clearly labels it **Guided** because no
PeerFoil application enforces the transitions yet.

This plan divides the phase into five stages. Each stage is sized for about one focused
day and leaves behind something that can be inspected and tested. The phase ends only
after the complete journey passes on two software examples and one documentation example.

## 1. Phase goal

A user with Git, Claude Code, Codex, and the required project tools can:

1. install the PeerFoil plugin;
2. describe an application or change;
3. resolve the important decisions;
4. create and independently review an architecture and plan;
5. delegate one small production task at a time;
6. run and retain appropriate evidence;
7. receive independent Claude and Codex phase reviews;
8. guide one repair when needed;
9. update the plan after a change or deferral; and
10. resume in a fresh chat from repository files.

## 2. Release boundary

### Included

- One distributable Claude Code plugin and marketplace catalog.
- Markdown skills for setup, start, change, status, resume, review, and remembering a
  lesson.
- Fresh role definitions for evaluator, architect, planner, reviewer, and repair work.
- The Software and Generic project packs.
- A small Documentation Pack that proves the workflow is not tied to code.
- Readable templates for every accepted `.peerfoil/` project artifact.
- Guided authorship, model-family, evidence, review, repair, and plan-revision records.
- Setup that finds the Codex CLI and registers its MCP server in Claude Code.
- Static validation and fresh-session checks on Windows, macOS, and Linux.
- Two small software examples and one small documentation example.

### Not included

- A PeerFoil executable, service, database, or hosted account.
- Mechanically enforced workflow states or review-pass limits.
- Controller-run or tamper-resistant evidence.
- Atomic writes, crash recovery, or unattended execution.
- Direct Claude Code or Codex process adapters.
- Automatic MCP routing, local-model routing, or model qualification.
- Operating-system sandboxing or simultaneous writers.

The plugin must not imply that a guided safeguard is mechanically enforced.

## 3. Required decisions before Stage 1 closes

Record each accepted decision in a short architecture decision record or the repository's
decision log.

| Decision | Recommended starting choice | Reason |
|---|---|---|
| Plugin location | Root marketplace with `plugins/peerfoil/` | Matches Claude Code's documented marketplace layout |
| Skill format | Agent Skills-compatible `SKILL.md` directories | Portable, inspectable, and supported by Claude Code |
| User command names | `start`, `change`, `status`, `resume`, `review-phase`, `remember`, `settings`, and `setup` | Covers the complete guided journey without a large command surface |
| Accepted project files | Markdown for people; JSON only where validation needs it | Keeps project truth easy to inspect and diff |
| Codex access | Detect the Codex CLI and register its MCP server; fall back to `codex exec` | Native on every platform; no Node.js or plugin dependency |
| Default project profile | Ask `personal` or `work`; inherit repository `AGENTS.md` | Prevents one profile from silently weakening another |
| Assurance label | Always show `Guided` in Skills 0.1 status and phase results | Makes the release boundary honest |

Do not settle the final public marketplace identifier until local and GitHub installation
both pass. Do not publish an untested installation command in the README.

## 4. Planned repository layout

Stage 1 may adjust names to satisfy the current Claude Code validator, but it must preserve
these responsibilities:

```text
.claude-plugin/
  marketplace.json
plugins/
  peerfoil/
    .claude-plugin/
      plugin.json
    skills/
      setup/SKILL.md
      start/SKILL.md
      change/SKILL.md
      status/SKILL.md
      resume/SKILL.md
      review-phase/SKILL.md
      remember/SKILL.md
      settings/SKILL.md
    agents/
      evaluator.md
      architect.md
      planner.md
      claude-reviewer.md
      repair-coordinator.md
    references/
      workflow.md
      evidence.md
      review.md
      lineage.md
    packs/
      software/
      generic/
      documentation/
    templates/
      project.json
      decisions.md
      architecture.md
      quality.md
      plan.md
      plan.json
      history.jsonl
      evidence.md
      review.md
      lesson.md
schemas/
fixtures/
  software-go/
  software-node/
  documentation/
tests/
```

New skills use one directory per skill with a `SKILL.md` file. Keep each skill focused and
move shared details into referenced files instead of repeating the complete method in every
skill.

## 5. Shared contracts

All skills and agents use the same small set of records. Define these before writing role
prompts so names do not drift.

| Record | Minimum content |
|---|---|
| Project | Project identifier, pack, profile, assurance, accepted settings |
| Decision | Question, options, answer or assumption, effect, owner, status |
| Architecture | Goals, boundaries, decisions, risks, quality contract, revision |
| Plan | Phase, stage, task, dependencies, acceptance, status, revision |
| Task | Scope, allowed paths, inputs, output, required evidence, author role |
| Change set | Base revision, changed paths, author session, model family, summary |
| Evidence | Type, procedure or command, result, revision, timestamp, retained output |
| Finding | Location, requirement, severity, evidence, recommendation, disposition |
| Review | Frozen revision, reviewer lineage, findings, decision, remaining risk |
| Lesson | Trigger, scope, evidence, proposed destination, review status |

Use stable identifiers that remain unchanged when a chat restarts. The Skills release may
guide validation, but templates must already match the shapes that Core Alpha will read.

## 6. Stage overview

| Stage | Outcome | Depends on |
|---:|---|---|
| Stage 1 | Installable plugin skeleton and stable shared contracts | Existing project documents |
| Stage 2 | Evaluated decisions, architecture, plan, and Software Pack | Stage 1 contracts |
| Stage 3 | Codex production, plan changes, status, and fresh-chat recovery | Stage 2 workflow artifacts |
| Stage 4 | Independent review, guided repair, lessons, and specialist lenses | Stage 3 change and evidence records |
| Stage 5 | Tested examples, three-platform checks, documentation, and release candidate | Stages 1–4 |

## 7. Stage 1 — Plugin foundation and decision intake

### Outcome

A developer can load the plugin locally, run setup, start a project, and receive a short
list of important decisions. The plugin creates only documented project files and labels
the session **Guided**.

### Tasks

1. Create the marketplace and plugin manifests using the current Claude Code plugin
   layout.
2. Add GPLv3-compatible notices and validate every new dependency or copied example.
3. Create the eight skill directories with valid Agent Skills frontmatter.
4. Add a `setup` skill that checks:
   - repository access and Git availability;
   - whether `AGENTS.md` exists and was read;
   - Claude Code and Codex CLI versions, found on the `PATH` or in IDE extension
     folders;
   - the Codex login and the Codex MCP server registration in Claude Code;
   - the user's personal or work profile; and
   - project tools declared by the selected pack.
5. Create the shared contracts and human-readable templates listed above.
6. Create the Software, Generic, and Documentation pack manifests with no executable
   controller logic.
7. Implement the first half of `start`:
   - summarize the user's goal in plain language;
   - identify the project pack;
   - read repository instructions before other project content;
   - create the initial `.peerfoil/project.json`; and
   - ask only consequential questions.
8. Define identifier and revision conventions that Phase 2 can reuse.
9. Add static checks for required files, JSON, frontmatter, relative links, license
   notices, and stale project names.

### Verification

- Run Claude Code's strict plugin validator.
- Load the plugin from its local directory in a fresh Claude Code session.
- Confirm each skill appears under the expected namespace.
- Run `setup` with Codex present and absent.
- Confirm an absent dependency produces one clear next step and no false success.
- Start a synthetic personal project and work project; confirm neither profile overwrites
  the repository's `AGENTS.md`.
- Confirm every created file stays inside the selected repository.

### Stage 1 is complete when

- the plugin loads without warnings treated as errors;
- setup accurately reports every required prerequisite;
- start produces a valid project record and decision list;
- the templates and contracts use consistent names and identifiers; and
- the user can understand what to do next without reading the architecture document.

### Stage 1 status

Built on 2026-09-05 and awaiting its independent different-family review. The seven
required decisions are recorded as D-0001 to D-0007 in the
[decision log](../decision-log.md), with D-0008 to D-0012 for choices made during the
build. Deviations from the layout in section 4: `references/records.md` replaces the
planned `evidence.md` and `review.md` references until Stages 3 and 4 need them; a
`change-set.md` template was added because section 5 lists the change set as a record;
and schemas live at the repository root as decided in D-0009.

Checks that ran on Windows with Claude Code 2.1.260:

| Check | Result |
|---|---|
| Strict plugin validator on the marketplace and plugin | Pass |
| Plugin loaded from its local directory in fresh sessions; eight skills and the evaluator appear under `peerfoil:` | Pass |
| Static checks and the conformance script | Pass |
| `setup` with the Codex plugin absent | Fail reported with one next step and no false success |
| `start` for a personal project without `AGENTS.md` and a work project with `AGENTS.md` | Valid `project.json` and `history.jsonl`; ten resolved decisions each; only the three documented files created |
| `AGENTS.md` unchanged after the work project started; an injected instruction in the README was ignored | Pass |
| `resume` and `status` in fresh sessions | Correct state from files alone, nothing written |

Not run in this build: `setup` with the Codex plugin present, macOS and Linux sessions, and
an interactive answer path for the decision interview. Those checks belong to the Stage 5
platform matrix and to the reviewer of this stage.

## 8. Stage 2 — Architecture, planning, and the Software Pack

### Outcome

PeerFoil turns resolved decisions into an architecture, Quality Contract, and ordered plan.
Codex independently reviews the Claude-authored architecture and plan before they can guide
production.

### Tasks

1. Finish the evaluator agent:
   - keep a shrinking list of consequential questions;
   - recommend an option with a reason;
   - distinguish a required answer from a reversible assumption; and
   - write every accepted answer and consequence to `decisions.md`.
2. Create a fresh architect agent that receives approved decisions rather than the full
   chat.
3. Define the architecture template with goals, users, boundaries, dependencies, data,
   risks, accessibility, security, privacy, licensing, and open decisions.
4. Define the Quality Contract template with required, recommended, and not-applicable
   evidence.
5. Create a fresh planner agent that divides work into user-visible phases and stages,
   then into small tasks.
6. Tie every task to:
   - an architecture and plan revision;
   - one expected outcome;
   - allowed paths or artifacts;
   - dependencies;
   - required evidence; and
   - completion criteria.
7. Complete the Software Pack with practical checks for correctness, reliability,
   security, privacy, accessibility, maintainability, documentation, licensing, and
   release readiness.
8. Use the Codex MCP server, or the `codex exec` fallback, to review the architecture and
   plan in fresh read-only sessions.
9. Require the user to accept the architecture and stage order after review.
10. Block production while consequential decisions or blocking review findings remain.

### Verification

- Seed an unanswered privacy decision and confirm planning does not begin.
- Seed a reversible naming choice and confirm it is recorded as an assumption without
  blocking progress.
- Seed an architecture inconsistency and confirm Codex reports it before planning.
- Seed a plan that omits required accessibility evidence and confirm review blocks it.
- Confirm the architect, planner, and Codex reviewer have distinct session identifiers.
- Confirm the plan describes outcomes rather than internal model activity.

### Stage 2 is complete when

- the full decision-to-approved-plan path works in a fresh fixture;
- architecture and plan reviews are retained with author and reviewer lineage;
- the Quality Contract selects evidence appropriate to the fixture; and
- no production task can begin before the reviewed plan is accepted.

### Stage 2 status

Built on 2026-09-05 and awaiting its independent different-family review. Decisions made
during the build are D-0013 to D-0016 in the [decision log](../decision-log.md).
Deviations from the layout in section 4: `references/review.md` now exists with the
architecture and plan review transfer, and Stage 4 extends it to phase review;
`references/architecture.md` and `references/planning.md` hold the shared architecture
and planning procedures so `start` and `resume` stay short; and
`agents/claude-reviewer.md` exists now, limited to the reduced-assurance fallback, and
Stage 4 extends it to phase review.

Checks that ran on Windows with Claude Code 2.1.260, using a scratch Go command-line
fixture with four resolved decisions and no Codex plugin installed:

| Check | Result |
|---|---|
| Static checks, the conformance script, and the strict plugin validator on the marketplace and plugin | Pass |
| Plugin loaded from its local directory in fresh sessions; the architect, planner, and Claude reviewer launch as `peerfoil:` agents with distinct session identifiers | Pass |
| Seeded open privacy decision: `resume` stayed in `define`, wrote nothing, and named the open decision as the blocker | Pass |
| Seeded reversible naming choice recorded as `assumed` without blocking; the architect's own reversible assumptions recorded the same way | Pass |
| Seeded architecture inconsistency: the reviewer returned a blocking finding naming the offline decision and the `AGENTS.md` rule; the architect revised the draft; pass 2 approved | Pass |
| Seeded plan without required accessibility evidence: the reviewer returned a blocking finding; the planner revised the plan; pass 2 approved; stage order approved | Pass |
| Full decision-to-approved-plan path in a fresh chat: `define` to `architect` to `plan`, architecture and plan accepted, `project.json`, `plan.json`, and every history line valid against the schemas, no placeholders left | Pass |
| Plan describes outcomes, not model activity, in every phase and stage title | Pass |
| Production stays blocked: the state remained `plan` and `status` reported production as not yet available | Pass |
| No different-family reviewer available: the run recorded the user's acceptance of Reduced assurance, `independence: secondary`, and the time, in the review and in the history | Pass |
| Generated records: one task `scope` exceeded 1,000 characters and one history line used an invented `refs` key | Fixed in the planner and references; not re-run |
| Timing: the runs took 12 to 30 minutes at extra-high effort with 48 to 140 coordinator turns | Led to D-0017 and D-0018; not re-run |

The reduced-assurance path stood in for the Codex reviewer in every review above.

Not run in this build: the Codex side of the review transfer, because Node.js, Codex CLI,
and the Codex plugin are not installed on the build machine; macOS and Linux sessions;
an interactive answer path; and a complete re-run at the medium-effort, turn-limited
defaults that D-0017 and D-0018 introduced after the runs above, which used the earlier
extra-high settings. The owner stopped the seeded-inconsistency run during its plan
review after its architecture checks had passed. The Codex transfer follows the plugin's
documented rescue path and must be exercised in Stage 5 with the plugin installed. Those
checks belong to the Stage 5 platform matrix and to the reviewer of this stage.

After those runs, D-0019 replaced the Codex plugin with the Codex CLI's own MCP server
and a `codex exec` fallback, so no Node.js is needed, and D-0021 and D-0022 added time
limits with an "answer now" nudge. Checks of that route on the same machine, with Codex
CLI 0.153.0 from the VS Code extension and its MCP server registered in Claude Code:

| Check | Result |
|---|---|
| A fresh nested session called `mcp__codex__codex` read-only and received a thread identifier | Pass |
| `codex exec` fallback with the packet on standard input, `--json`, and `--output-last-message`, then the `exec resume` nudge on the same thread | Pass |
| `setup` found Codex inside the VS Code extension and Claude Code in its desktop install, saw the MCP tool, and recorded versions only | Pass; three version probes were Not verified because the test's permission rules blocked path-invoked commands |
| Seeded architecture inconsistency reviewed by Codex: pass 1 returned a blocking finding naming the offline decision, with `independence: independent` and a distinct thread identifier per pass | Pass |
| Three architecture passes returned `repair`; the skill accepted at the pass limit with the open findings deferred; the plan was written; the plan's first Codex pass returned `repair`; the run then stopped when the Claude account's session limit was reached | Recorded. The fixture still carried `xhigh` reviewer seats, so its forty minutes for four Codex passes does not reflect the new defaults |

Not run after D-0019: the "answer now" nudge inside a skill run, and a complete run at
the medium-effort defaults with the time limits in force. A small run at those defaults
did measure one step: a fresh session in the `architect` state reached its first
independent Codex review of the architecture two minutes and five seconds after launch.

### Stage 2 independent review

Reviewer: Codex CLI 0.153.0 through `codex exec` in a read-only sandbox at medium
effort, lineage `openai-gpt`, thread `01a07034-dd05-73b2-8256-d60006a82a24`, reading
the working tree on 2026-09-05. Author of the reviewed material: Claude Code
(`anthropic-claude`). Independence: independent.

| Pass | Decision | Result |
|---:|---|---|
| 1 | `repair` | One blocking finding: the pass limit could accept a draft with an open blocking finding. Six major findings: lineage inferred from the tool rather than the model; interrupted repairs not resumable from files; malformed consequential evaluator items could be dropped; the MCP nudge assumed a thread identifier a timeout never returns; dependency validation ignored phase order; allowed-path containment undefined. One minor: status read only the newest review. |
| 2 | `repair` | No new blocking or major findings. Four repairs confirmed, four partial, and one inconsistency in `review.md` section 11. |
| 3 | `repair` | No new findings. All confirmed repaired except one partial: approval recovery relied on a pass identity the drafts do not store. |

Every finding was addressed in the references, skills, and D-0023. The last partial
item was resolved after pass 3 by comparing the review's frozen revisions and time with
the draft's own revision and `Written at` or `revised_at`; that change has not been
reviewed by Codex, because the three-pass limit applied to this review too. The
reviewer's remaining risk stands: live interruption, timeout, and macOS and Linux
behavior were not exercised.

## 9. Stage 3 — Production, changes, status, and recovery

### Outcome

PeerFoil can send one bounded task to Codex, preserve its authorship, run declared project
checks, revise the plan when work changes, and resume from repository files in a fresh
chat.

### Tasks

1. Implement the guided production step inside `start` and `resume`.
2. Create a compact Codex transfer packet containing only:
   - the relevant repository rules;
   - accepted decisions and architecture excerpts;
   - the active plan and task revision;
   - allowed paths;
   - required evidence; and
   - the expected structured handoff.
3. Use the Codex MCP server's `codex` tool with a `workspace-write` sandbox, or the
   `codex exec` fallback. Do not build another bridge.
4. Require one task per Codex call and one writer at a time.
5. Capture the base revision, changed paths, Codex session, model family, and patch before
   another agent edits the work.
6. Guide the host to run the project's declared tests and retain exact results.
7. Implement `change` with the five allowed destinations: current stage, later stage in
   the phase, later phase, backlog, or declined.
8. Create a new plan revision after every accepted change, TODO, skipped check, deviation,
   repair, or deferral.
9. Implement `status` with plain-language output:
   - assurance level;
   - current phase, stage, and task;
   - quality state;
   - current blocker;
   - pending user decision; and
   - next action.
10. Implement `resume` so a fresh session reads accepted files rather than reconstructing
    state from the old transcript.

### Verification

- Delegate a bounded change to Codex and confirm unrelated files remain unchanged.
- Confirm Codex authorship is recorded before a Claude edit.
- Add a requirement during production and confirm the plan revision changes.
- Confirm an old task tied to the prior plan cannot be reported as current.
- Restart Claude Code and resume using only Git and `.peerfoil/` files.
- Confirm status never reports **Enforced** during Phase 1.
- Confirm raw prompts, full transcripts, and provider tokens do not enter Git.

### Stage 3 is complete when

- one bounded task completes with retained evidence and authorship;
- a mid-stage change updates the plan without losing unrelated completed work;
- status tells the user what is happening and what needs them; and
- a fresh chat resumes the correct task without the old chat history.

## 10. Stage 4 — Review, repair, and lessons

### Outcome

A complete phase receives fresh Claude and Codex reviews of the same frozen material. One
repair can be guided and checked by an eligible different model family.

### Tasks

1. Implement `review-phase` and define the frozen review bundle:
   - accepted decisions and architecture;
   - current plan and change sets;
   - exact deliverables;
   - evidence and tool versions;
   - known risks, TODOs, deviations, and deferrals; and
   - artifact- and patch-level authorship.
2. Complete the Claude reviewer agent and Codex review transfer.
3. Require both reviewers to work independently before comparing findings.
4. Normalize findings by identifier, location, requirement, severity, evidence, proposed
   action, and disposition.
5. Combine duplicate findings without erasing disagreement.
6. Apply the four default review lenses:
   - correctness and reliability;
   - security and privacy;
   - accessibility and user experience; and
   - maintainability, documentation, licensing, and release readiness.
7. Guide the configured six-pass default and eight-pass ceiling. State plainly that
   Phase 1 cannot enforce these limits mechanically.
8. Select an eligible repairer without allowing the author of the repair to
   approve it.
9. Allow one guided repair cycle, rerun affected evidence, and obtain a fresh
   different-family verification.
10. Implement `remember` with candidate lessons that record trigger, scope, supporting
    evidence, conflicts, and proposed destination.

### Verification

- Seed a defect and confirm both reviewers inspect the same revision.
- Confirm each material item identifies an eligible different-family primary reviewer.
- Confirm same-family critique is labeled secondary and cannot satisfy normal approval.
- Seed duplicate and conflicting findings; confirm neither is silently lost.
- Confirm a serious failed check cannot be dismissed by reviewer agreement.
- Apply one repair and confirm affected checks and independent verification run again.
- Confirm an unverified lesson remains a candidate rather than becoming policy.

### Stage 4 is complete when

- one seeded phase moves through review, repair, fresh evidence, and approval;
- self-approval is visibly blocked in the records;
- exhausted or unresolved review stops for the user instead of inventing consensus; and
- lessons survive the chat without silently changing `AGENTS.md`.

## 11. Stage 5 — Examples, platform checks, and release

### Outcome

The complete guided workflow is understandable, installable, and repeatable on Windows,
macOS, and Linux. Release notes clearly explain what Skills 0.1 can and cannot enforce.

### Tasks

1. Finish two tiny software examples with different toolchains:
   - a Go command-line project; and
   - a Node.js command-line project.
2. Finish one Markdown documentation example with link, source, readability, and editorial
   checks.
3. Give each example one seeded requirement change and one seeded review defect.
4. Create a test script or small test harness for manifests, frontmatter, links, required
   files, identifiers, schemas, notices, and fixture expectations.
5. Add Windows, macOS, and Linux continuous-integration jobs for tests that require no
   model credentials.
6. Complete manual fresh-session smoke tests with Claude Code and Codex on all three
   operating systems.
7. Test repository paths containing spaces, Unicode, and apostrophes.
8. Test the plugin from a local directory and from its marketplace repository.
9. Write setup, troubleshooting, fixture, privacy, license, and known-limit documentation.
10. Record exact tested versions without claiming future compatibility.
11. Run independent Claude and Codex review of the complete Phase 1 release candidate.
12. Fix blocking findings, rerun checks, and create the release evidence summary.

### Verification

- Run every automated repository and fixture check on all three operating systems.
- Follow the installation instructions from a fresh checkout.
- Complete both software examples without using an old chat transcript.
- Complete the documentation example through the same artifact and review lifecycle.
- Confirm every unavailable Core feature is marked **Coming soon** or **Guided**.
- Confirm no paid non-LLM account, database, container, or hosted PeerFoil service is
  required.
- Confirm all copied material, dependencies, and notices are GPLv3-compatible.

### Stage 5 is complete when

- the three fixtures pass their complete journeys;
- fresh installation and session checks pass on Windows, macOS, and Linux;
- the final review has no unresolved blocking finding;
- release limitations are prominent and accurate; and
- the Phase 2 handoff contains every accepted contract and known issue.

## 12. Phase acceptance matrix

| ID | Required result | Evidence |
|---|---|---|
| P1-01 | Consequential decisions reach zero unresolved items | Decision file and evaluator test |
| P1-02 | Architecture and plan receive different-family review | Review records with lineage |
| P1-03 | Production waits for accepted architecture and plan | Negative fixture test |
| P1-04 | At least one bounded Codex task completes | Change set, patch, and session record |
| P1-05 | Project checks are retained for the exact change | Evidence record and Git revision |
| P1-06 | Fresh Claude and Codex phase reviews complete | Frozen bundle and two review records |
| P1-07 | Each material item has an eligible primary reviewer | Authorship-to-reviewer matrix |
| P1-08 | A change or deferral creates a new plan revision | Before-and-after plan records |
| P1-09 | A fresh chat resumes from repository files | Clean-session smoke test |
| P1-10 | Documentation uses the same lifecycle | Documentation fixture results |
| P1-11 | Three operating systems pass installation checks | Platform matrix |
| P1-12 | The interface always reports Guided assurance | Status snapshots and negative tests |

Every row is required. A model's written claim does not satisfy an evidence column by
itself.

## 13. Review assignments

| Work | Author family | Required primary review |
|---|---|---|
| Evaluator, architecture, and planning skills | Claude | Codex |
| Codex transfer and software-production behavior | Claude plus Codex-produced fixture patches | Cross-family review assigned per artifact or patch |
| Claude reviewer instructions | Claude | Codex |
| Codex review prompt and schema | Claude | Codex for contract correctness; Claude for Codex-produced results |
| Final fixtures and release candidate | Mixed | Both families, with item-level independence |

Record authorship at the artifact and patch level. Do not label a complete mixed phase as
independent without checking each material item.

## 14. Risks and responses

| Risk | Response |
|---|---|
| Five stages become a plugin platform project | Keep one plugin, three packs, and the fixed lifecycle |
| Skills repeat large instructions and exceed context | Keep skills focused; move shared rules into references |
| Claude changes Codex work before provenance is saved | Make change-set capture an explicit gate before editing |
| Plugin installation differs across platforms | Test local and marketplace installation on all three systems |
| Plugin or Codex output changes format | Pin tested versions in release evidence and fail visibly on unknown output |
| Users mistake guided checks for enforcement | Display Guided in setup, status, review, and release documentation |
| Reviews become endless | State the limit, stop early on agreement, and ask the user when the limit is reached |
| Fixtures consume the schedule | Keep them tiny and seed only behavior needed by acceptance checks |

## 15. Phase 2 handoff

Before Phase 2 begins, commit and review:

- the accepted record definitions and JSON schemas;
- the plugin's workflow and role instructions;
- all `.peerfoil/` templates;
- Software, Generic, and Documentation pack manifests;
- the three fixture repositories and expected results;
- the Phase 1 evidence summary;
- tested Claude Code, Codex CLI, Git, and operating-system versions;
- known provider-output variations;
- every deferred item and unresolved non-blocking finding; and
- lessons that should affect Core Alpha.

Phase 2 may implement these contracts. It may not silently reinterpret them. A necessary
contract change must update Phase 1 templates, fixtures, schemas, and documentation in the
same change.

## 16. Definition of done

Phase 1 is complete only when:

1. Stages 1–5 meet their completion checks.
2. P1-01 through P1-12 have current evidence.
3. The complete workflow works from a fresh chat without a previous transcript.
4. No agent or model family approves its own material work.
5. Required failures remain blocking.
6. Windows, macOS, and Linux pass the supported test matrix.
7. The release needs no custom PeerFoil runtime or paid non-LLM service.
8. Documentation, headers, copyright, trademark, licenses, notices, and credits are
   complete.
9. The README and setup instructions describe only tested features.
10. An independent phase review recommends release with no unresolved blocking finding.

## Conclusion

Phase 1 proves that PeerFoil is useful before its controller exists. The plugin guides a
complete decision-to-review journey, preserves enough project truth for a new chat, and
uses Claude and Codex as independent peers. The deliberately small fixtures and explicit
Guided label keep the five-stage schedule realistic and honest.

## Additional Resources

- [PeerFoil implementation plan](../implementation-plan.md)
- [PeerFoil method](../PeerFoil-Method.md)
- [PeerFoil architecture](../architecture.md)
- [Phase prompt template](../phase-prompt-template.md)
- [Claude Code plugin guide](https://code.claude.com/docs/en/plugins)
- [Claude Code marketplace guide](https://code.claude.com/docs/en/plugin-marketplaces)
- [Claude Code plugin reference](https://code.claude.com/docs/en/plugins-reference)
- [Agent Skills specification](https://agentskills.io/specification)
- [Codex CLI](https://github.com/openai/codex)

[Return to the PeerFoil README](../../README.md)

---

Copyright © 2026 Gabriel Mongefranco
