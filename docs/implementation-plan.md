<!--
This file is part of PeerFoil.
docs/implementation-plan.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-04
Last Modified: 2026-09-04
Summary: Explains what PeerFoil will deliver, when it will arrive, and how each release will be checked.
Notes: See README for an overview and full license information.

Copyright © 2026 Gabriel Mongefranco

Permission is granted to copy, distribute and/or modify this document under the terms of
the GNU Free Documentation License, Version 1.3 or any later version published by the Free
Software Foundation; with no Invariant Sections, no Front-Cover Texts, and no Back-Cover
Texts. See <https://www.gnu.org/licenses/fdl-1.3.html>.
-->

# PeerFoil Implementation Plan

## A useful guided workflow in five days, a working core in thirteen, and version 1.0 within six months

[Return to the PeerFoil README](../README.md)

PeerFoil will be useful before the complete application is finished. The first release is
a set of skills that guides Claude Code and Codex through the whole workflow. The next
release is a small local application that runs and records the most important steps. Later
releases add stronger review, change handling, memory, connected knowledge, local models,
and project types other than software.

This plan gives each release a firm boundary and a clear test. If a feature is not ready,
PeerFoil will say so instead of presenting a guided step as an enforced safeguard.

## 1. Release schedule

The day count starts when implementation begins.

| Release | Target | What the user gets |
|---|---:|---|
| PeerFoil Skills 0.1 | Day 5 | A complete guided workflow inside Claude Code |
| PeerFoil Core Alpha 0.2 | Day 13 | A local command-line application that runs one software phase and records its evidence |
| Reliable Core | Week 4 | Safer recovery, clearer errors, and the first document workflow |
| Review Beta | Week 8 | Bounded two-family review and one independently checked repair cycle |
| Planning Beta | Week 12 | Plan updates that preserve unaffected work when requirements change |
| Context Beta | Week 16 | Focused skills, MCP access, shared context, and reviewed lessons |
| Model Beta | Week 20 | Qualified local models and provider-neutral model settings |
| Release Beta | Week 24 | Finished built-in project packs and easier installation |
| PeerFoil 1.0 | Week 26 | The complete, tested cross-platform product |

## 2. Rules that apply to every release

PeerFoil must:

- work natively on Windows, macOS, and Linux;
- require no paid non-LLM service or hosted control plane;
- keep normal setup simple and place detailed controls under Advanced settings;
- use high effort for software work by default;
- prevent an agent from approving its own work;
- prefer a different model family for independent review;
- check every repair again with an independent model;
- keep required failures visible instead of letting reviewers vote them away;
- revise the plan when requirements, TODOs, deferrals, or discovered work change;
- treat `AGENTS.md` as the repository's highest local instruction source; and
- clearly label safeguards that are guided rather than enforced.

PeerFoil 1.0 will not include operating-system sandboxing, multiple agents writing at the
same time, team administration, a hosted dashboard, or automatic production deployment.
Those features would add complexity without improving the first experience for a solo
developer.

## 3. What PeerFoil will reuse

PeerFoil is a coordination and quality layer. It will use mature tools for the work those
tools already do well.

| Need | Existing tool or standard |
|---|---|
| Claude and Codex cooperation | Official `openai/codex-plugin-cc` plugin |
| Model execution | Separately installed Claude Code, Codex CLI, Ollama, or compatible endpoints |
| Versions and isolated changes | Git branches and worktrees |
| Portable agent instructions | Agent Skills format and `AGENTS.md` |
| Connected tools and knowledge | Model Context Protocol (MCP) |
| Tests and quality checks | Commands and scanners already used by the project |

PeerFoil will not build another coding agent, Git implementation, model runtime, scanner,
credential store, or general workflow framework.

## 4. Project packs

A project pack tells PeerFoil what kind of work is being produced, which checks matter,
and what evidence reviewers need. Every pack uses the same basic workflow:

```text
Define → Architect → Plan → Produce → Validate → Review → Repair → Approve
```

| Pack | Plan |
|---|---|
| Software | The default pack, included from the first release |
| Generic | A small starting point for custom work, included from the first release |
| Documentation | A small example in Skills 0.1; a complete pack by version 1.0 |
| Business Plan | Prototype after the Reliable Core; complete by version 1.0 |
| Research Report | Prototype after the Reliable Core; complete by version 1.0 |
| Custom Pack Kit | Small extension guide and examples in the Release Beta |

A pack may choose artifacts, checks, evidence, skills, and review lenses. It may not grant
itself credentials, weaken `AGENTS.md`, hide failed checks, or allow self-approval.

## 5. Days 1–5: PeerFoil Skills 0.1

### What the user can do

A user who already has Git, Claude Code, Codex, and model authentication can install the
PeerFoil skills and complete the whole guided journey. They can make the important design
decisions, approve an architecture and plan, send small coding tasks to Codex, run project
checks, receive independent Claude and Codex reviews, accept a repair, update the plan,
and resume from files stored in the repository.

### What will be included

- A Claude Code marketplace and plugin package.
- Portable Markdown skills and fresh agent-role instructions.
- Setup instructions for the official Codex plugin.
- Readable templates for decisions, architecture, plans, tasks, evidence, reviews, and
  lessons.
- Guided actions for start, plan, produce next, add a change, status, resume, review a
  phase, remember a lesson, and settings.
- Personal and work standards profiles that do not replace a repository's own rules.
- Software and Generic packs, plus one small Documentation example.
- A different-family review of the architecture and plan before coding begins.
- Review lenses for correctness, reliability, security, privacy, accessibility, user
  experience, maintainability, documentation, licensing, and release readiness.
- Installation and fresh-session checks on Windows, macOS, and Linux.
- Complete license notices and a clear path-based license policy.

### Five-day sequence

| Day | Main result |
|---:|---|
| 1 | Plugin skeleton, project-pack format, settings, and decision interview |
| 2 | Architect, planner, evaluator, quality contract, and Software Pack |
| 3 | Codex delegation, one-task production, change intake, status, resume, and plan updates |
| 4 | Fresh Claude review, two-family phase review, repair choice, lessons, and specialist checks |
| 5 | Documentation example, two software examples, plugin validation, three-OS checks, and user documentation |

### Release check

Two small software examples must complete the following journey:

1. Resolve every important open decision.
2. Create an architecture, quality contract, and ordered plan.
3. Receive a different-family review of the architecture and plan.
4. Send at least one small implementation task to Codex.
5. Preserve who created the change before another agent edits it.
6. Run and record the project checks.
7. Complete fresh Claude and Codex phase reviews.
8. Revise the plan after a change or deferral.
9. Resume the work from checked-in project files.

The Documentation example must use the same artifacts and review path to create and check
one small Markdown document.

### What is not enforced yet

The Skills release depends on agents following the written process. It does not yet
provide mechanical state control, controller-run evidence, crash recovery, direct provider
processes, automatic pass limits, local-model routing, or MCP routing. The interface will
show **Guided** so users understand that boundary.

## 6. Days 6–13: PeerFoil Core Alpha 0.2

### What the user can do

The `peerfoil` command will read the same files created by PeerFoil Skills and run one
small software phase. It will call Claude Code and Codex, keep the change in a Git
worktree, run checks itself, record who produced and reviewed the work, and resume safely
from a completed task boundary.

Initial commands:

```text
peerfoil init
peerfoil doctor
peerfoil start
peerfoil status
peerfoil resume
```

### What will be included

- A small Go command-line application with native development builds.
- One clear, sequential workflow.
- Validation of plans, project packs, and saved artifacts.
- Direct Claude Code and Codex processes using their existing authentication.
- One supported default model arrangement.
- One coding task per model call and one writing agent at a time.
- A dedicated worktree and integration branch for the task.
- Controller-run checks tied to the exact Git revision.
- Authorship and review records that remain understandable after a fresh clone.
- Redacted workflow history and task-boundary recovery.
- One Claude and Codex phase review.
- One guided high-effort repair with fresh different-family verification.
- Native Windows, macOS, and Linux continuous integration.

### Release check

On all three operating systems, Core Alpha must:

1. Read an architecture, project pack, and plan created by Skills 0.1.
2. Run one coding task in its own worktree.
3. Run the declared checks and tie the results to the correct revision.
4. Integrate only work that is in scope and has its required evidence.
5. Resume safely after an interruption between tasks.
6. Block an out-of-scope change or a change made directly in the user's checkout.
7. Run a two-family review without allowing self-approval.
8. Guide one high-effort repair and have another family check it.
9. Process the Documentation example without special document-only controller code.

Core Alpha is a developer preview. It does not claim to contain the complete six-month
product.

## 7. Weeks 3–4: Reliable Core

PeerFoil will handle common command, provider, timeout, path, output, and restart failures
safely. It will either recover or tell the user exactly what to do next.

Main work:

- versioned project-pack and artifact validation;
- timeout and child-process cleanup;
- model and effort detection with visible fallbacks;
- limited retry escalation;
- repairable state diagnostics;
- Documentation Pack alpha;
- Business Plan and Research Report prototypes; and
- optional detection of compatible security scanners already installed by the user.

The release check will inject common failures on all three operating systems. Each failure
must recover predictably or stop with a useful message. Adding a normal project pack must
not require a controller change.

## 8. Weeks 5–8: Review Beta

Every phase will receive limited, evidence-based review from fresh Claude and Codex model
families. Reviewers will work independently first, then compare specific findings. One
accepted repair cycle may run at high effort, followed by a fresh independent check.

Main work:

- freeze the exact material being reviewed;
- retain authorship for each artifact and change;
- select specialist checks from the project pack and risk level;
- combine duplicate findings without hiding disagreement;
- require a different-family primary approval;
- use six review passes per reviewer by default and eight at most;
- reserve the final allowed pass for checking repairs;
- use three passes per reviewer to choose a repair model by default and four at most; and
- stop and ask the user when the allowed review cannot reach a safe conclusion.

Seeded code defects and unsupported document claims must block approval, receive an
explicit decision, and be checked again after repair.

## 9. Weeks 9–12: Planning Beta

PeerFoil will accept a new request while work is in progress. It will decide whether the
request belongs now, later in the current phase, in a later phase, in the backlog, or
should be declined. It will update the plan and preserve work that the change does not
affect.

Main work:

- change-impact review and placement;
- selective reopening of tasks and evidence;
- readable plan history;
- links from requirements to tasks and evidence;
- capture of TODOs, unsupported claims, skipped checks, deviations, and deferrals; and
- beta Business Plan and Research Report packs.

The release check will add an important change to a software example and a non-coding
example. PeerFoil must reopen affected work, keep unrelated work, and prevent stale work
from being approved.

## 10. Weeks 13–16: Context Beta

PeerFoil will give each role only the approved context, skills, knowledge sources, and
lessons it needs. Personal and work information will remain separate.

Main work:

- personal and work standards profiles;
- predictable skill selection and version records;
- small, reviewable skills based on `AGENTS.md`;
- task-specific MCP settings, tool allowlists, and health checks;
- clear rules about which information may leave the computer;
- compact shared context packets;
- lesson candidates that require review before promotion; and
- promotion of useful lessons into tests, decisions, skills, glossary entries, or proposed
  instruction changes.

A required MCP outage must block only the dependent task. Unapproved MCP tools must stay
unavailable, private content must stay out of Git, and a repeated problem must be promoted
to the right durable project file.

## 11. Weeks 17–20: Model Beta

Advanced settings will allow qualified local models to replace hosted model seats without
changing PeerFoil's project files or review rules.

Main work:

- one provider-neutral model adapter contract;
- clear model-family records for hosted and local models;
- configurable model, fallback, and effort settings;
- authentication and usage diagnostics;
- Ollama and OpenAI-compatible/vLLM adapters;
- an optional OpenCode coding adapter;
- small qualification tasks for each role and project pack; and
- read-only access for models that have not qualified for a writing or review role.

Models derived from the same base model will count as the same family for independent
review. A fully local setup will still require two qualified families for normal approval.

## 12. Weeks 21–24: Release Beta

Fresh users should be able to install PeerFoil, complete setup in about five minutes, and
choose a finished built-in project pack.

Main work:

- cross-platform packaging and checksummed binaries;
- safe state upgrades;
- software bills of materials, notices, and license checks;
- cost limits and simple usage information;
- VS Code tasks and terminal integration;
- finished Software, Documentation, Business Plan, and Research Report packs;
- a small Custom Pack Kit; and
- complete examples and upgrade tests.

The release matrix will cover spaces, Unicode, apostrophes, CRLF line endings, case-only
changes, cancellation, damaged local state, reconstruction from Git, package installation,
offline local-model operation, and every built-in project pack on all three operating
systems.

## 13. Weeks 25–26: PeerFoil 1.0

The final two weeks are reserved for fixes, usability, release checks, and documentation.
They are not available for adding major features.

Version 1.0 must include:

- a GPLv3 compatibility and dependency review;
- documentation-license and notice checks;
- an upgrade guide and reference projects;
- accessibility, privacy, and security reviews of PeerFoil itself;
- the five-minute setup check; and
- a complete start-to-approved-phase test.

Any capability that does not meet its acceptance check will remain clearly unshipped.

## 14. Quality gates

Every release must meet these conditions:

| Area | Requirement |
|---|---|
| Independence | No agent approves its own work; another family normally provides primary approval |
| Evidence | Important results have current test output, inspectable support, or explicit human confirmation |
| Simplicity | Normal use requires no configuration-file editing |
| Portability | Windows, macOS, and Linux acceptance checks pass |
| Licensing | Distributed dependencies are GPLv3-compatible and notices are complete |
| Cost | No non-LLM paid account or hosted service is required |
| Honesty | Missing evidence, reduced assurance, and unresolved decisions remain visible |
| Generality | Software remains best supported while other packs use the same controller |
| Scope | Deferred features do not quietly enter the release |

## 15. Main risks

| Risk | Response |
|---|---|
| The five-day release becomes only a demo | Protect the complete guided journey and defer polish first |
| The thirteen-day alpha grows into the whole product | Keep one narrow software path and publish its limits |
| Non-coding packs slow down software quality | Build Software first and use one small Documentation example to test the shared design |
| Packs become a new programming language | Keep one lifecycle and use small declarative pack files |
| Windows, macOS, and Linux behave differently | Run all three in continuous integration from the beginning |
| Reviews use too much time or model credit | Freeze review inputs, combine duplicates, stop early, and enforce pass limits |
| A model reviews a renamed version of its own family | Record the underlying model family and treat unknown lineage as reduced assurance |
| Documents do not have executable tests | Allow inspectable and human evidence with pack-specific consistency checks |
| Local models lower quality without warning | Qualify them for each role and leave unknown models read-only |
| Provider behavior changes | Check capabilities and show every fallback |
| Licensing problems appear late | Check licenses, notices, and dependency reports in every release |
| Advanced machinery makes daily use confusing | Keep five normal actions and test them with new users |
| Six months of ideas become six years of scope | Cut optional integrations before weakening the release checks |

## 16. Definition of done

PeerFoil 1.0 is complete only when:

1. Skills, Core, and built-in packs use the same project files and lifecycle.
2. A new user can finish one phase without editing configuration files.
3. Software, Documentation, Business Plan, and Research Report examples pass their own
   quality contracts.
4. Every important artifact has independent approval or a visible reduced-assurance
   decision.
5. A failed required check cannot be accepted by model agreement.
6. Plans stay current after changes, TODOs, deviations, skipped checks, and deferrals.
7. Accepted work and reviewer independence can be reconstructed from Git without private
   transcripts or provider sessions.
8. Qualified hosted and local models can exchange roles without changing project files.
9. Personal and work context remain separate.
10. Native Windows, macOS, and Linux releases pass the full test matrix.
11. No non-LLM paid service or hosted component is required.
12. Code, documentation, dependencies, and notices pass the license policy.

## Conclusion

PeerFoil will ship the smallest complete experience first and add enforcement in useful
steps. The five-day Skills release proves the workflow. The thirteen-day Core Alpha proves
that the main safeguards can be automated. The remaining schedule improves reliability,
review, planning, context, model choice, project packs, and installation without changing
the simple experience promised in the first release.

## Resources

- [PeerFoil method](PeerFoil-Method.md)
- [PeerFoil architecture](architecture.md)
- [Official Codex plugin for Claude Code](https://github.com/openai/codex-plugin-cc)
- [Model Context Protocol](https://modelcontextprotocol.io/)
- [GNU General Public License, version 3](https://www.gnu.org/licenses/gpl-3.0.html)
- [GNU Free Documentation License, version 1.3](https://www.gnu.org/licenses/fdl-1.3.html)

[Return to the PeerFoil README](../README.md)
