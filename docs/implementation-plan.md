<!--
Project:  PeerFoil  |  File: docs/implementation-plan.md
Authors:  Gabriel Mongefranco (@gabrielmongefranco)
Created:  2026-09-04  |  Modified: 2026-09-04
Summary:  Provides PeerFoil's high-level delivery plan from guided skills to the 1.0 product.
SPDX-License-Identifier: GFDL-1.3-or-later
-->

# PeerFoil High-Level Implementation Plan

## A usable guided workflow in five days, enforced core in thirteen, and complete 1.0 within six months

- **Version:** 0.1, 4 September 2026
- **Status:** Pre-implementation plan
- **Software license intent:** `GPL-3.0-or-later`
- **Documentation license intent:** `GFDL-1.3-or-later`

## 1. Delivery objective

PeerFoil ships in three useful increments:

| Release | Deadline from project start | User-visible outcome |
|---|---:|---|
| **PeerFoil Skills 0.1** | Day 5 | A guided architecture-to-production-to-review workflow using Markdown skills and the official Codex plugin |
| **PeerFoil Core Alpha 0.2** | Day 13 | A local CLI mechanically enforces one narrow software-first path, evidence capture, recovery, and independent review |
| **PeerFoil 1.0** | No later than Week 26 | A hardened cross-platform product with bounded review, change control, project packs, context, MCP, memory, and qualified local models |

Software development remains the flagship use. The controller and schemas use neutral concepts so documentation, business plans, research reports, and later custom packs can share the same lifecycle:

```text
Define → Architect → Plan → Produce → Validate → Review → Repair → Approve
```

## 2. Fixed constraints

- Native Windows, macOS, and Linux operation.
- Software and operational artifacts—including skills, agents, packs, templates, schemas, and machine-consumed Markdown—licensed `GPL-3.0-or-later`; human-facing prose documentation licensed `GFDL-1.3-or-later`, with explicit SPDX identifiers for ambiguous files.
- No required hosted control plane, database service, paid CI, or other non-LLM subscription.
- No Docker, WSL, Bash, tmux, or Unix-only dependency in the default path.
- No agent approves its own work.
- Normal primary approval comes from another qualified model family.
- Software production uses high effort by default; medium is limited to bounded, reversible, qualified low-risk work.
- Every repair uses high effort and receives fresh cross-family verification.
- Objective failures and missing required evidence cannot be voted away.
- `AGENTS.md` remains authoritative.
- Normal operation requires no hand-edited configuration; advanced controls stay under Advanced settings.
- Sandboxing, parallel writers, hosted dashboards, team administration, and automatic production deployment remain outside 1.0.

The schedule assumes one experienced developer working with focused model assistance and protecting the stated scope. Dates and exit gates are both commitments. Missing either is reported plainly.

## 3. Reuse strategy

PeerFoil builds only the governance layer that existing tools do not provide.

| Need | Reuse |
|---|---|
| Claude-to-Codex cooperation | Official `openai/codex-plugin-cc` |
| Model execution | Separately installed Claude Code, Codex CLI, and later local runtimes |
| Versioning and change isolation | Git and Git worktrees |
| Portable skills | Agent Skills format |
| Connected knowledge and tools | Model Context Protocol |
| Quality checks | Existing project commands and compatible external scanners |
| Local inference | Ollama, OpenAI-compatible/vLLM endpoints, and optional compatible harnesses |

PeerFoil does not build another coding agent, Git implementation, model runtime, scanner, workflow framework, credential store, or hosted service.

## 4. Project-pack delivery

A project pack defines artifacts, roles, phases, validators, evidence, skills, MCP needs, review lenses, and completion criteria within the fixed PeerFoil lifecycle.

| Pack | Priority | Delivery target |
|---|---:|---|
| Software | Flagship/default | Skills 0.1 and every later release |
| Generic | Minimal extension base | Skills 0.1 |
| Documentation | Neutrality fixture, then polished pack | Reference in Skills 0.1; mature by 1.0 |
| Business Plan | Planned built-in | Prototype after Reliable Core; mature by 1.0 |
| Research Report | Planned built-in | Prototype after Reliable Core; mature by 1.0 |
| Custom Pack Kit | Small extension surface | Release Beta |

Packs do not define arbitrary workflows and cannot grant credentials, widen permissions, override `AGENTS.md`, suppress evidence, or relax reviewer independence.

## 5. Phase 1 — PeerFoil Skills 0.1

**Schedule:** Days 1–5 · **Assurance:** Guided

### User-visible outcome

A developer with Git, Claude Code, Codex, and authentication already available can install PeerFoil's skills, resolve consequential decisions, approve architecture and stage order, delegate bounded production work, collect evidence, run independent Claude/Codex review, revise the plan, and resume from checked-in artifacts.

### Fixed scope

- Claude Code marketplace and plugin package.
- Portable Markdown skills and fresh-role agent definitions.
- Documented setup for the official Codex plugin.
- Neutral schemas and templates for decisions, architecture, acceptance contracts, plans, tasks, evidence, reviews, packs, and lessons.
- Guided start, plan, produce-next, change, status, resume, review-phase, remember, and settings actions.
- Personal/work standards profile selection without overwriting repository rules.
- Default hosted role mapping and visible fallbacks.
- Software and basic Generic packs.
- One small Documentation Pack fixture.
- Cross-family provenance and review rules.
- Mandatory different-family architecture and plan review before either artifact governs production.
- Four conditional review lenses:
  - correctness and reliability;
  - security and privacy;
  - accessibility and user experience;
  - maintainability, documentation, licensing, and release integrity.
- Three-OS installation and fresh-session smoke tests.
- Complete software and documentation license texts, SPDX path policy, and initial notices.

### Five-day sequence

| Day | Primary deliverable |
|---:|---|
| 1 | Marketplace/plugin skeleton, pack contract, core artifacts, settings, and decision interview |
| 2 | Fresh evaluator, architect, and planner agents; Quality Contract; Software Pack |
| 3 | Explicit Codex delegation, one-task production, change intake, status/resume, plan amendments |
| 4 | Fresh Claude reviewer, dual-family phase review, repair selection, lessons, specialist lenses |
| 5 | Documentation fixture, two software fixtures, plugin validation, three-OS smoke tests, notices, and user documentation |

### Exit evidence

On two small software fixtures, the release must:

1. reach zero unresolved consequential decisions;
2. create an architecture, acceptance contract, and ordered plan that each receive eligible different-family review before acceptance;
3. delegate at least one bounded Codex implementation task only after both pre-production review gates pass;
4. retain the produced change before any coordinator edit;
5. run and record applicable project checks;
6. complete fresh Claude and Codex phase review;
7. identify the eligible different-family primary reviewer for each material item;
8. revise the plan after a change or deferral;
9. resume from committed artifacts.

The same schema and review path must produce and review one small Markdown deliverable through the Documentation fixture.

### Explicit deferrals

- Mechanical state enforcement.
- Authoritative controller-run evidence.
- Atomic transitions and crash guarantees.
- Direct provider process adapters.
- Automated pass-count enforcement.
- Local-model adapters and MCP orchestration.
- Polished non-code packs.

The release displays **Guided** and never implies those controls are enforced.

## 6. Phase 2 — PeerFoil Core Alpha 0.2

**Schedule:** Days 6–13 · **Assurance:** Enforced within the documented alpha boundary

### User-visible outcome

The `peerfoil` executable consumes the same Skills artifacts and automatically completes one sequential software task with controller-run evidence, task-boundary recovery, and one independent phase-review/repair path.

### Fixed scope

- Small Go CLI and native development binaries.
- Deterministic state loop.
- Stable artifact, plan, and project-pack validation.
- Direct Claude Code and Codex process adapters using provider-native authentication.
- One supported default role mapping.
- Software Pack enforcement and Documentation fixture processing.
- Existing clean Git workspaces with an initial commit.
- One producer at a time and one task per model call.
- Dedicated software-task worktree and integration branch.
- Controller-owned executable evidence bound to exact revisions.
- Durable artifact-, change-set-, invocation-, and review-lineage provenance that survives a clone without provider session data.
- Redacted transition history.
- Task-boundary crash recovery.
- One fixed Claude/Codex phase-review round.
- One guided high-effort repair and fresh cross-family verification.
- Native Windows, macOS, and Linux CI and smoke tests.

Initial commands:

```text
peerfoil init
peerfoil doctor
peerfoil start
peerfoil status
peerfoil resume
```

### Exit evidence

On every supported operating system, Core Alpha must:

1. load Skills-generated architecture, pack, and plan artifacts;
2. invoke a sequential producer task in a dedicated worktree;
3. run declared commands itself and bind results to the exact revision;
4. integrate only in-scope work with passing required evidence and durable author/reviewer lineage records;
5. survive interruption at a task boundary;
6. detect and block an injected out-of-scope or live-checkout mutation;
7. run one dual-family review with provenance-aware approvals;
8. guide one high-effort repair and independent re-verification when seeded;
9. process the Documentation fixture without a controller code path specific to documents.

### Explicit deferrals

- Multi-round review reconciliation and automatic repair consensus.
- Advanced change-impact analysis and selective invalidation.
- Deterministic skills routing and role-scoped MCP.
- Memory promotion and cross-project lessons.
- Arbitrary provider and local-model adapters.
- Scanner installation, parallel workers, submodules, LFS, GUI, and production deployment.
- Mature Documentation, Business Plan, and Research Report experiences.

Core Alpha is a developer alpha, not the finished six-month product.

## 7. Phase 3 — Reliable Core

**Schedule:** Weeks 3–4

### User-visible outcome

Normal subprocess, provider, path, timeout, malformed-output, and restart failures recover safely or stop with an exact next action on all three operating systems.

### Scope

- Structured packet and plan validation.
- Stable versioned pack manifest.
- Timeout and complete process-tree termination.
- Model and effort capability detection with visible fallback.
- Bounded retry escalation.
- Repairable state diagnostics.
- Documentation Pack alpha.
- Business Plan and Research Report pack prototypes.
- Optional detection of compatible Gitleaks and OSV-Scanner installations.

### Exit evidence

Injected command, provider, timeout, malformed-output, pack, encoding, path, and restart failures either recover deterministically or pause honestly. Adding an ordinary pack requires no controller change.

## 8. Phase 4 — Review Beta

**Schedule:** Weeks 5–8

### User-visible outcome

Every phase receives bounded, evidence-backed review from two fresh model families. Accepted repairs are performed once, at high effort, and independently reverified.

### Scope

- Frozen review bundles.
- Artifact- and change-set-level provenance.
- Pack- and risk-selected specialist lenses.
- Normalized finding ledger.
- Different-family primary approval.
- Six review passes per reviewer by default; eight maximum.
- One phase-review pass per reviewer reserved for post-repair verification.
- Three repair-selection passes per reviewer by default; four maximum.
- Exact repair-producer consensus.
- One automatic repair cycle.
- Fresh independent re-verification.
- Non-code lenses for factual support, assumptions, calculations, clarity, feasibility, source quality, and limitations.

### Exit evidence

Seeded software defects and unsupported document claims block acceptance, receive explicit dispositions, and are independently rechecked after repair. Forced disagreement exhausts its configured budget and pauses without invented consensus.

## 9. Phase 5 — Planning and change control

**Schedule:** Weeks 9–12

### User-visible outcome

A new request can be accepted during work. PeerFoil decides whether it belongs now, later in the current phase, in a later phase, in the backlog, or should be declined—then revises the plan and preserves unaffected work.

### Scope

- Impact-aware change placement.
- Selective task and evidence invalidation.
- Explicit plan-revision history.
- Requirement-to-task-to-evidence traceability.
- TODO, unsupported-claim, deviation, skipped-check, and deferral capture.
- Pack-aware acceptance contracts.
- Business Plan and Research Report beta packs.

### Exit evidence

A consequential mid-stage change revises the plan, reopens affected work, preserves unrelated completed work, and prevents stale integration in both a software fixture and a non-code fixture.

## 10. Phase 6 — Context, skills, MCP, and memory

**Schedule:** Weeks 13–16

### User-visible outcome

PeerFoil automatically supplies each role with relevant approved context, pertinent skills, permitted knowledge sources, and durable lessons without mixing personal and work material.

### Scope

- Personal/work standards profiles.
- Deterministic skill eligibility and pinned skill records.
- Reviewable focused skills derived from `AGENTS.md`.
- Ephemeral per-invocation MCP configuration, deny-by-default tool filtering, adapter qualification, and health checks.
- Personal/work data-egress policy.
- Compact shared context packets.
- Manual and discovered lesson candidates.
- Reviewed promotion into tests, decisions, skills, glossary entries, or proposed standards changes.
- Strict personal/work memory separation.

### Exit evidence

A required MCP outage or an adapter unable to enforce role isolation blocks the dependent task; unpermitted servers and tools remain unavailable; private payloads remain outside Git; and a recurring issue becomes the correct reviewed durable artifact.

## 11. Phase 7 — Provider and local-model support

**Schedule:** Weeks 17–20

### User-visible outcome

Advanced settings can replace hosted seats with qualified local models while preserving the same project artifacts, evidence, and review protocol.

### Scope

- Provider-neutral seat contract.
- Canonical hosted-model lineage catalog and user-approved local manifests containing base lineage and model digests.
- Model, fallback, and effort settings.
- Authentication and usage diagnostics.
- Ollama adapter.
- OpenAI-compatible/vLLM adapter.
- Optional OpenCode production harness.
- Transformers.js external-helper path where appropriate.
- Role- and pack-specific qualification fixtures.
- Read-only default for unqualified models.
- Fully local two-lineage configuration.

### Exit evidence

Hosted and qualified local seats can exchange roles without schema changes. No local model performs an unqualified role, endpoint aliases and derivatives of one base remain one lineage, and normal independent approval still requires a distinct qualified lineage root.

## 12. Phase 8 — Release Beta

**Schedule:** Weeks 21–24

### User-visible outcome

Fresh users can complete the default setup in approximately five minutes, use PeerFoil from a terminal or VS Code, and select a mature built-in project pack.

### Scope

- Cross-platform hardening.
- Checksummed and freely signed binaries.
- Schema migration tooling.
- Conditional packaging, SBOM, notices, and license gates.
- Cost ceilings and compact usage reporting.
- VS Code tasks and terminal integration.
- Finished Software, Documentation, Business Plan, and Research Report packs.
- Small Custom Pack Kit with examples and validation.
- Complete reference projects and migration fixtures.

### Exit evidence

The release matrix passes spaces, Unicode, apostrophes, CRLF, case-only changes, cancellation, corrupt state, reconstruction from Git, package installation, offline local-model operation, and every built-in pack fixture on Windows, macOS, and Linux.

## 13. Phase 9 — PeerFoil 1.0

**Schedule:** Weeks 25–26

### User-visible outcome

A supported user can install PeerFoil, complete setup, and finish one reviewed phase without editing configuration files.

### Scope

- Schedule buffer and usability repairs.
- Dependency and GPLv3 compatibility audit.
- Documentation-license verification.
- Release candidate and migration guide.
- Reference projects and tutorials.
- Accessibility, privacy, and security review of PeerFoil itself.
- Final five-minute setup and end-to-end acceptance pass.

### Exit evidence

All committed acceptance criteria pass. Any missed capability remains explicitly unshipped rather than being silently weakened or relabeled.

## 14. Cross-cutting release gates

Every release must satisfy:

| Gate | Requirement |
|---|---|
| Independence | No authoring agent approves its own work; normal primary approval comes from another family |
| Evidence | Required outcomes have executable, inspectable, or explicit human evidence |
| Simplicity | Normal operation needs no configuration-file editing; advanced controls stay hidden |
| Portability | Native Windows, macOS, and Linux acceptance tests pass |
| Licensing | Distributed software is GPLv3-compatible; documentation licensing and notices are correct |
| Cost | No non-LLM subscription, hosted service, or paid account is required |
| Honesty | Missing evidence, exhausted budgets, reduced assurance, and unresolved decisions remain visible |
| Generality | Software remains best-supported; shared controller code contains no pack-level software assumptions |
| Scope | Sandboxing, team administration, hosted dashboards, parallel writers, and automatic deployment stay excluded |

## 15. Critical dependencies

| Capability | Must exist first |
|---|---|
| Skills 0.1 | Stable artifacts, pack contract, fresh-role prompts, Codex plugin integration |
| Core Alpha | Skills schemas, Go build, Git behavior, provider CLI invocation |
| Review Beta | Evidence engine, provenance map, frozen revision bundle |
| Change control | Stable plan graph and accepted-transition history |
| Non-code packs | Neutral artifact and evidence model |
| Context and MCP | Stable roles, capability policy, and private-data boundary |
| Local models | Provider-neutral seat contract and qualification fixtures |
| Release 1.0 | Cross-platform CI, schema migration, license gate, pack reference projects |

## 16. Principal risks

| Risk | Mitigation |
|---|---|
| Five-day release becomes a demo without a full journey | Freeze the exit journey and defer polish before weakening it |
| Day-13 scope becomes the entire product | Enforce the alpha boundary and publish explicit deferrals |
| Generic packs delay software quality | Build Software first; prove neutrality with one small Documentation fixture |
| Packs become a workflow-language project | Keep one fixed lifecycle and declarative pack manifests |
| Core duplicates prompt logic | Keep skills as policy; Core validates and controls transitions |
| Cross-platform process behavior slips | Run three-OS CI from the first commit; test process trees and difficult paths |
| Review loops consume excessive time or tokens or exhaust repair verification | Freeze context, stop early, bound passes, reserve the final pass for post-repair review, allow one repair cycle |
| Same-lineage work is accidentally self-approved through aliases or fine-tunes | Canonicalize hosted families; require pinned local lineage manifests and model digests; treat unknown lineage as Reduced assurance |
| Non-code work lacks executable tests | Support inspectable and human evidence plus pack-specific consistency checks |
| Local models silently reduce quality | Require role qualifications; leave unknown models read-only |
| Provider or plugin behavior changes | Probe capabilities, pin tested versions where possible, expose every fallback |
| License incompatibility appears late | Define path-level licensing with SPDX identifiers; generate dependency reports, notices, and SBOMs; manually review ambiguous licenses |
| Hidden machinery leaks into daily use | Retain five normal actions and user-test the default path |
| Lack of sandboxing is misunderstood | State the trusted-workspace boundary and never call worktrees a security sandbox |
| Six-month scope expands | Cut GUI, parallelism, enterprise features, and optional integrations before gates |

## 17. Definition of done

PeerFoil 1.0 is done only when:

1. Skills, Core, and built-in packs share the same artifacts and lifecycle.
2. A fresh user completes one phase without editing configuration.
3. Software, Documentation, Business Plan, and Research Report reference projects pass their acceptance contracts.
4. Every material artifact receives eligible independent approval or an explicit `Reduced assurance` decision.
5. Required failures cannot be accepted by model consensus.
6. Plans remain current after changes, TODOs, unsupported claims, deviations, and deferrals.
7. Accepted state and author/reviewer lineage eligibility survive clone and reconstruction without raw transcripts, provider sessions, or private MCP payloads.
8. Qualified hosted or local seats can be substituted without changing project schemas.
9. Personal and work standards, context, and memory remain separated.
10. Native Windows, macOS, and Linux releases pass the full matrix.
11. No non-LLM paid service or hosted component is required.
12. Distributed code, documents, dependencies, and notices pass the stated license policy.

## Related documents

- [PeerFoil method](PeerFoil-Method.md)
- [Architecture](architecture.md)
- [Project overview](../README.md)
