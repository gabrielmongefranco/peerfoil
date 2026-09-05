<!--
This file is part of PeerFoil.
docs/decision-log.md
Author(s): Gabriel Mongefranco; OpenAI Codex.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Records the accepted engineering decisions that shape PeerFoil's implementation.
Notes: Each entry is short. A decision changes only through a new entry that supersedes it.

Copyright © 2026 Gabriel Mongefranco

Permission is granted to copy, distribute and/or modify this document under the terms of
the GNU Free Documentation License, Version 1.3 or any later version published by the Free
Software Foundation; with no Invariant Sections, no Front-Cover Texts, and no Back-Cover
Texts. See <https://www.gnu.org/licenses/fdl-1.3.html>.
-->

# PeerFoil Decision Log

## The short record of choices that later work must respect

[Return to the PeerFoil README](../README.md)

This log records the decisions that the implementation plan asked to settle before or
during a stage. Each entry states what was decided, what else was considered, why the
choice won, and what it means for later work. A decision is changed by adding a new entry
that supersedes the old one, never by editing history.

## Index

| ID | Decision | Status | Stage |
|---|---|---|---|
| D-0001 | Plugin lives in a root marketplace with `plugins/peerfoil/` | Accepted | Phase 1, Stage 1 |
| D-0002 | Skills use Agent Skills-compatible `SKILL.md` directories | Accepted | Phase 1, Stage 1 |
| D-0003 | Eight user command names | Accepted | Phase 1, Stage 1 |
| D-0004 | Markdown for people, JSON only where validation needs it | Accepted | Phase 1, Stage 1 |
| D-0005 | Detect and guide the Codex plugin instead of declaring a dependency | Superseded by D-0019 | Phase 1, Stage 1 |
| D-0006 | Ask `personal` or `work` and inherit the repository's `AGENTS.md` | Accepted | Phase 1, Stage 1 |
| D-0007 | Skills 0.1 always shows `Guided` assurance | Accepted | Phase 1, Stage 1 |
| D-0008 | Provisional marketplace identifier `peerfoil` | Provisional | Phase 1, Stage 1 |
| D-0009 | Packs and templates ship inside the plugin; schemas stay at the repository root | Accepted | Phase 1, Stage 1 |
| D-0010 | Template files carry a sibling notice instead of a PeerFoil header | Accepted | Phase 1, Stage 1 |
| D-0011 | Identifier and revision conventions | Accepted | Phase 1, Stage 1 |
| D-0012 | Static checks use Python 3 with no third-party packages | Accepted | Phase 1, Stage 1 |
| D-0013 | Architecture and plan reviews run through the Codex plugin's rescue agent, fresh and read-only | Superseded by D-0019 | Phase 1, Stage 2 |
| D-0014 | Reduced-assurance fallback when no different-family reviewer is available | Accepted | Phase 1, Stage 2 |
| D-0015 | Drafts carry the next revision number; accepted revisions live in `project.json` | Accepted | Phase 1, Stage 2 |
| D-0016 | Packs may declare evidence command hints | Accepted | Phase 1, Stage 2 |
| D-0017 | Architect defaults to high effort and every other role to medium | Accepted | Phase 1, Stage 2 |
| D-0018 | Agents declare turn limits and reviewers cap their findings | Accepted | Phase 1, Stage 2 |
| D-0019 | Codex is reached through the Codex CLI's MCP server, with `codex exec` as fallback | Accepted | Phase 1, Stage 2 |
| D-0020 | Setup finds Codex and Claude Code on the PATH or in IDE extension folders | Accepted | Phase 1, Stage 2 |
| D-0021 | Wall-clock limits: ten minutes for a review, draft, or task; five for small steps | Accepted | Phase 1, Stage 2 |
| D-0022 | Two-stage deadlines: an "answer now" nudge, then a kill one minute later | Accepted; Skills half narrowed by D-0023 | Phase 1, Stage 2 |
| D-0023 | In Skills 0.1 the nudge exists only on the `codex exec` path; lineage comes from the model identifier | Accepted | Phase 1, Stage 2 |
| D-0024 | Later review passes decide on blocking findings only, on changed material only | Accepted | Phase 1, Stage 2 |
| D-0025 | Guided production snapshots and change continuity | Implemented | Phase 1, Stage 3 |
| D-0026 | Phase review record with item-level primary reviewers and a merged finding list | Accepted | Phase 1, Stage 4 |
| D-0027 | Repairer chosen by rule; reviewer agreement stands in for the repair task's plan review | Accepted | Phase 1, Stage 4 |
| D-0028 | Lessons are verified at phase review and promoted only with the user; hints may stay unverified | Accepted | Phase 1, Stage 4 |

## D-0001 — Plugin location

- **Status:** Accepted on 2026-09-05.
- **Decision:** The repository root holds a Claude Code marketplace manifest at
  `.claude-plugin/marketplace.json`. The plugin itself lives at `plugins/peerfoil/` with
  its own `.claude-plugin/plugin.json`.
- **Options considered:** A single-plugin repository with the manifest at the root; a
  separate marketplace repository.
- **Reason:** The layout matches the documented Claude Code marketplace structure, keeps
  the plugin installable from a local directory and from GitHub, and leaves room for the
  fixtures, schemas, and tests that do not belong inside the installed plugin.
- **Consequences:** Everything the installed plugin needs at run time must live under
  `plugins/peerfoil/`. Repository-level tests and schemas are not installed with it. The
  strict validator rejects unknown fields inside `marketplace.json`, so that file's notice
  lives in the sibling `.claude-plugin/NOTICE.md`. The plugin manifest keeps its notice in
  its free-form `metadata` object, which the validator accepts.

## D-0002 — Skill format

- **Status:** Accepted on 2026-09-05.
- **Decision:** Each user action is a directory under `plugins/peerfoil/skills/` with a
  `SKILL.md` file whose frontmatter follows the Agent Skills specification. The plugin may
  also use Claude Code-specific frontmatter fields such as `argument-hint`,
  `disable-model-invocation`, and `allowed-tools`, because the plugin targets Claude Code.
- **Options considered:** Flat `commands/*.md` files; spec-only frontmatter.
- **Reason:** Skill directories can carry supporting files, and the Claude Code fields let
  the plugin stop Claude from starting a side-effect workflow on its own.
- **Consequences:** The skill `name` must match its directory name. Shared detail lives in
  `plugins/peerfoil/references/` so each skill stays short.

## D-0003 — User command names

- **Status:** Accepted on 2026-09-05.
- **Decision:** The plugin exposes exactly eight commands: `setup`, `start`, `change`,
  `status`, `resume`, `review-phase`, `remember`, and `settings`. Each appears as
  `/peerfoil:<name>` in Claude Code.
- **Options considered:** Fewer commands with sub-actions; separate commands for
  architecture, planning, and production.
- **Reason:** Eight names cover the complete guided journey while keeping the normal user
  surface small. Architecture, planning, production, and review run inside `start`,
  `resume`, and `review-phase` rather than as separate commands.
- **Consequences:** New behavior is added inside these commands. Adding a ninth command
  requires a new decision.

## D-0004 — Accepted project files

- **Status:** Accepted on 2026-09-05.
- **Decision:** Files that people read and edit are Markdown. Files that need validation
  are JSON: `project.json`, `plan.json`, and one JSON object per line in `history.jsonl`.
- **Options considered:** JSON for everything; YAML for everything.
- **Reason:** Markdown keeps decisions, plans, evidence, reviews, and lessons readable
  without PeerFoil installed. JSON gives Core Alpha a validated shape for the records it
  must enforce.
- **Consequences:** `plan.md` and `plan.json` describe the same plan. When they disagree,
  `plan.json` is authoritative and `plan.md` must be regenerated.

## D-0005 — Plugin dependency behavior

- **Status:** Accepted on 2026-09-05. Superseded by D-0019 on 2026-09-05.
- **Decision:** The plugin manifest declares no plugin dependency. The `setup` skill
  detects whether the official Codex plugin is installed and guides the user through its
  documented installation when it is not.
- **Options considered:** Declaring `codex` as a manifest dependency across marketplaces.
- **Reason:** Cross-marketplace dependency behavior is not something PeerFoil should
  assume. Detection with clear guidance works on every installation path.
- **Consequences:** A missing Codex plugin produces one clear next step and never a false
  success.

## D-0006 — Default project profile

- **Status:** Accepted on 2026-09-05.
- **Decision:** Setup asks whether a project is `personal` or `work`. Both profiles inherit
  the repository's own `AGENTS.md` when it exists. PeerFoil never creates, edits, or
  replaces `AGENTS.md` on its own.
- **Options considered:** A single default profile; copying a template `AGENTS.md`
  automatically.
- **Reason:** Work policies must not leak into personal projects, and personal defaults
  must not weaken a work repository.
- **Consequences:** The profile and the rules source are recorded in `project.json`.
  Template and reference rule sources are handled in a later release.

## D-0007 — Assurance label

- **Status:** Accepted on 2026-09-05.
- **Decision:** Every status report, phase result, and generated project record in
  PeerFoil Skills 0.1 shows the assurance level `Guided`.
- **Options considered:** Showing no label; showing `Enforced` for steps that follow the
  written process closely.
- **Reason:** Skills depend on agents following instructions. Only Core can claim enforced
  transitions and controller-run evidence.
- **Consequences:** The word `Enforced` may appear in this release only when explaining
  what a later release will add.

## D-0008 — Provisional marketplace identifier

- **Status:** Provisional on 2026-09-05. Final acceptance waits for Phase 1, Stage 5.
- **Decision:** The marketplace name is `peerfoil` and the plugin name is `peerfoil`, so
  the planned installation form is `peerfoil@peerfoil`.
- **Options considered:** A marketplace name that differs from the plugin name.
- **Reason:** One name is simplest to explain, and the official Codex plugin uses the same
  pattern of a short marketplace name with a short plugin name.
- **Consequences:** The README does not publish an installation command until local and
  GitHub installation both pass on Windows, macOS, and Linux.

## D-0009 — Where packs, templates, and schemas live

- **Status:** Accepted on 2026-09-05.
- **Decision:** Project packs and `.peerfoil/` templates live inside the plugin at
  `plugins/peerfoil/packs/` and `plugins/peerfoil/templates/`. JSON schemas live at the
  repository root under `schemas/`.
- **Options considered:** Root-level `packs/` and `templates/` as listed in the long-term
  source layout; shipping schemas inside the plugin.
- **Reason:** Skills can only reach files inside the installed plugin. Schemas are used by
  repository checks and later by Core, which reads them from the repository.
- **Consequences:** Core Alpha reads the same pack and template formats from the plugin
  directory or from a copy it ships. Whether schemas also ship inside the plugin is decided
  in Phase 1, Stage 5.

## D-0010 — Template notices

- **Status:** Accepted on 2026-09-05.
- **Decision:** Files under `plugins/peerfoil/templates/` carry no PeerFoil header. The
  notice for the whole directory lives in `plugins/peerfoil/templates/README.md`.
- **Options considered:** A PeerFoil header inside every template.
- **Reason:** A template becomes the user's own project record when it is copied into
  `.peerfoil/`. A PeerFoil copyright line inside the user's project files would be wrong.
- **Consequences:** Repository checks require the sibling notice and reject a PeerFoil
  copyright line inside a template.

## D-0011 — Identifier and revision conventions

- **Status:** Accepted on 2026-09-05.
- **Decision:** Identifiers are lowercase, use a short type prefix and a zero-padded
  number, and are never reused. Architecture, Quality Contract, and plan revisions are
  positive integers that increase by one on every accepted change. Git commit hashes tie
  work and evidence to exact source revisions. Timestamps use UTC in ISO 8601 form.
- **Options considered:** Random identifiers; hierarchical identifiers that encode plan
  order; timestamps in local time.
- **Reason:** Sequential typed identifiers stay stable when a chat restarts, are safe in
  file names and Git references, and do not change when the user reorders stages.
- **Consequences:** The full rules live in
  [`plugins/peerfoil/references/records.md`](../plugins/peerfoil/references/records.md)
  and the JSON schemas under `schemas/`. Core Alpha implements the same rules.

## D-0012 — Static check tooling

- **Status:** Accepted on 2026-09-05.
- **Decision:** Repository static checks run as one Python 3 script that uses only the
  standard library. The conformance script calls it, and continuous integration runs the
  Claude Code strict plugin validator separately.
- **Options considered:** Node.js scripts; shell-only checks; a third-party JSON Schema
  package.
- **Reason:** Python 3 is present on the Windows, macOS, and Linux runners and on
  developer machines that lack Node.js. Avoiding packages keeps the check runnable from a
  clean checkout.
- **Consequences:** The schema checker supports a documented subset of JSON Schema. Core
  Alpha selects its own Go schema library in Phase 2.

## D-0013 — Architecture and plan review transfer

- **Status:** Accepted on 2026-09-05. Superseded by D-0019 on 2026-09-05.
- **Decision:** A Claude-authored architecture or plan draft is reviewed by Codex through
  the official Codex plugin's `codex:codex-rescue` agent, launched with the `Agent` tool.
  The request is a self-contained, read-only review packet that names the frozen files
  and requires one JSON block of findings. The run uses `--fresh`, the reviewer seat's
  effort, and no `--write`, so it is a new read-only thread whose thread identifier is
  recorded as the reviewer's session. Each draft receives at most three passes before
  PeerFoil asks the user.
- **Options considered:** The plugin's `/codex:review` and `/codex:adversarial-review`
  commands; running Codex CLI directly; the plugin's `/codex:transfer` command.
- **Reason:** The review commands inspect Git diffs, not records, and cannot carry a
  PeerFoil packet. Running Codex CLI directly would duplicate the plugin's bridge.
  Transfer would hand Codex the Claude conversation, which the method forbids for a
  reviewer. The rescue agent is the plugin's documented delegation path and already
  runs read-only unless asked to write.
- **Consequences:** The packet, output contract, validation, and pass limit live in
  `plugins/peerfoil/references/review.md`. Stage 4 extends the same contract to phase
  reviews. The Codex plugin version tested with this transfer is recorded in Stage 5.

## D-0014 — Reduced-assurance fallback

- **Status:** Accepted on 2026-09-05.
- **Decision:** When no different-family reviewer is available for an architecture or
  plan draft, PeerFoil pauses and offers two choices: wait for an independent reviewer,
  which is recommended, or accept **Reduced assurance** for that one draft. On
  acceptance a fresh `peerfoil:claude-reviewer` session reviews the draft, the review
  records `independence: secondary` with the user's acceptance and its time, the history
  record repeats the acceptance, and status shows "Reduced assurance" for the artifact.
- **Options considered:** Always pausing; using a fresh Claude session silently.
- **Reason:** The method lets the user accept the limitation or wait. Pausing only would
  make the workflow untestable on a machine without Codex; silent substitution would hide
  the loss of independence.
- **Consequences:** Acceptance never carries over to a later draft. Phase review in Stage
  4 applies the same rule item by item.

## D-0015 — Draft revision numbering

- **Status:** Accepted on 2026-09-05.
- **Decision:** A draft architecture, Quality Contract, or plan carries the next number
  after the last accepted revision. Rewrites of the draft before acceptance keep that
  number, and each review pass records a `pass` counter. `project.json` `revisions`
  holds only accepted revisions and changes when the user accepts the draft.
- **Options considered:** Numbering every rewrite; numbering drafts as revision 0.
- **Reason:** Reviews and tasks must name the revision they were written against, and
  the accepted revision must be readable from one place without opening every draft.
- **Consequences:** Reviews freeze a revision plus a pass number and, when committed, a
  Git hash. This build cannot make a working-tree draft tamper-proof; Core adds that.

## D-0016 — Pack evidence hints

- **Status:** Accepted on 2026-09-05.
- **Decision:** A pack manifest may declare `evidence_hints`: a marker file, an evidence
  name it declares, a command as an argument list, and a purpose. The architect uses the
  hints that match the repository to fill executable procedures in the Quality Contract,
  and the repository's own declared scripts take precedence. Packs without hints declare
  an empty list.
- **Options considered:** Leaving commands entirely to the architect; hard-coding
  toolchain commands in the architect role.
- **Reason:** Practical checks belong to the pack, not to the controller or the role
  prompt, and the Go and Node.js fixtures in Stage 5 need real commands.
- **Consequences:** `schemas/pack.schema.json` requires the field, and the static checks
  reject a hint that names undeclared evidence.

## D-0017 — Default effort

- **Status:** Accepted on 2026-09-05. Supersedes the extra-high defaults that the method
  gave the evaluator, architect, change steward, and reviewers.
- **Decision:** The architect defaults to `high` effort because the architecture shapes
  every later task. The evaluator, planner, change steward, producer, repair producer,
  and both phase reviewers default to `medium`. `low` is an allowed value only for small,
  reversible, low-risk work and never for a repair. `high` and `xhigh` remain allowed
  values that a user can set for a seat under `/peerfoil:settings`, with a warning that
  the step becomes slower.
- **Options considered:** Keeping extra high for decisions, architecture, and review;
  high for every role.
- **Reason:** Stage 2 measurements showed that extra-high reviewers reading every record
  and returning ten findings per pass made one architecture-and-plan pass take twenty to
  thirty minutes, which is too slow for normal use. The owner chose medium as the normal
  level and reserved high for the architecture, where a weak result costs the most.
- **Consequences:** The method, architecture, implementation plan, Phase 1 and 2 plans,
  `AGENTS.md` rule 7, templates, agents, schemas, and settings skill say so. The
  independence and evidence rules do not change; effort never substitutes for review.

## D-0018 — Turn limits and finding caps

- **Status:** Accepted on 2026-09-05.
- **Decision:** A reviewer run uses at most ten turns; an evaluator, architect, or planner
  run uses at most six. Every PeerFoil agent declares that limit as `maxTurns`, and a
  Codex review request states it in its text. A reviewer returns at most ten findings
  with short evidence and recommendation fields, and on a later pass reports only new
  blocking or major findings after confirming the earlier repairs. The coordinating skill
  budgets about ten of its own turns to write a draft, ten per review pass, and five to
  record acceptance.
- **Options considered:** No limits; limiting only review passes.
- **Reason:** Stage 2 runs used more than one hundred coordinator turns and ten to
  fifteen turns per reviewer, which drove both time and cost. Claude Code enforces an
  agent's `maxTurns`; the coordinator budget and the finding cap are guided.
- **Consequences:** The static checks require `maxTurns` within the limit on every
  agent. A run that stops at its limit without a result is rerun once, then the user is
  asked.

## D-0019 — Codex access through the Codex MCP server

- **Status:** Accepted on 2026-09-05. Supersedes D-0005 and D-0013.
- **Decision:** PeerFoil reaches Codex through the Codex CLI's built-in MCP server,
  registered once in Claude Code with `claude mcp add --scope user codex -- <codex>
  mcp-server`. Every PeerFoil call starts a fresh thread with the `codex` tool, a
  `read-only` sandbox for reviews, `approval-policy` `never`, the repository as `cwd`,
  and the seat's effort passed as `model_reasoning_effort`; the returned thread
  identifier is recorded as the session. When the server is not registered but the CLI
  is present, PeerFoil runs `codex exec` with the packet on standard input, an ephemeral
  read-only session, and the final message written to a temporary file. The official
  Codex plugin for Claude Code is no longer used.
- **Options considered:** Keeping the Codex plugin, which is a set of Node.js scripts;
  calling `codex exec` only.
- **Reason:** The owner does not want Node.js as a prerequisite, and neither Claude Code
  nor Codex needs it. The MCP server is an official Codex feature, gives Claude Code a
  native tool with sandbox and effort control, and returns a thread identifier. The
  `exec` fallback keeps the workflow usable before registration.
- **Consequences:** `AGENTS.md`, the method, the implementation plan, the Phase 1 plan,
  the setup skill, the review and lineage references, and the plugin README say so.
  `plugins/peerfoil/references/codex.md` holds the contract. The Phase 2 controller
  calls the same two paths as processes.

## D-0020 — Finding Codex and Claude Code

- **Status:** Accepted on 2026-09-05.
- **Decision:** Setup looks for `codex` and `claude` on the `PATH` first. When either is
  missing, it looks in the user's IDE extension folders, where the Codex and Claude Code
  extensions bundle native binaries: on Windows
  `.vscode/extensions/openai.chatgpt-*/bin/windows-x86_64/codex.exe` and the Claude Code
  extension's `resources/native-binary/claude.exe`, with the matching `darwin-*` and
  `linux-*` folders on macOS and Linux, plus the VS Code Insiders and Cursor folders.
  The newest version wins, and a candidate counts only when it answers `--version`.
  Project files record versions only; a found path is used to register the MCP server
  in Claude Code's own user configuration and is otherwise re-detected each session.
- **Options considered:** Requiring the programs on the `PATH`; storing the path in
  `project.json`.
- **Reason:** Many Windows users have Codex only through the VS Code extension. A path
  under the home directory names the user, so it does not belong in a shared project
  file.
- **Consequences:** The patterns live in `plugins/peerfoil/references/codex.md` and are
  used by setup and by the review transfer's fallback.

## D-0021 — Time limits

- **Status:** Accepted on 2026-09-05.
- **Decision:** A review pass, an architecture or plan draft, and a production task may
  each take at most ten minutes of wall-clock time. The evaluator, setup probes, and
  status may take at most five. A run that passes its limit without a result counts as
  no result: it is retried once, then the user is asked. Every review records its
  duration in seconds. Where the host can enforce a limit it is used: the `codex exec`
  fallback runs under a ten-minute command timeout, and setup tells the user how to set
  Claude Code's `MCP_TOOL_TIMEOUT` to the same value. Agent runs have no host time
  limit, so their budget is guided alongside `maxTurns`.
- **Options considered:** Turn limits only; a single limit for everything.
- **Reason:** Turn limits alone did not bound duration; a Codex review at extra-high
  effort took about seven minutes, and the owner set an upper bound on what a user
  should wait for one step.
- **Consequences:** The method, review and Codex references, records, review template,
  agents, and setup skill state the limits. Core enforces them mechanically.

## D-0022 — Two-stage deadlines

- **Status:** Accepted on 2026-09-05.
- **Decision:** A time limit from D-0021 is a soft deadline. When it passes, PeerFoil asks
  the model once to stop and answer now with what it has, and gives it one more minute;
  at that hard deadline the run is killed and counts as no result. In Skills 0.1 the
  nudge is implemented for Codex only: `codex-reply` on the same thread for the MCP
  path, and `exec resume <thread id>` for the fallback, which therefore no longer runs
  ephemerally so the session can be resumed. A running Claude Code agent cannot be
  nudged in this release, so its `maxTurns` remains its only hard stop. Core implements
  the two stages for both families by resuming the session with the nudge before
  killing the process.
- **Options considered:** Killing at the limit; no time limit.
- **Reason:** A model that has read everything and is composing its answer should be
  allowed to finish briefly instead of losing the work; the owner asked for exactly this
  behavior.
- **Consequences:** The Codex and review references, the method, and the Phase 2 plan
  say so. The one-minute grace for the MCP nudge is guided in Skills 0.1 because the
  host applies one timeout to every MCP call.

## D-0023 — Corrections from the Stage 2 independent review

- **Status:** Accepted on 2026-09-05.
- **Decision:** A timed-out MCP call returns no thread identifier, so the "answer now"
  nudge of D-0022 applies in Skills 0.1 only to the `codex exec` fallback; a timed-out
  MCP call counts as no result and is retried once. PeerFoil skills never call
  `codex-reply`. A reviewer's `lineage_root` is derived from its model identifier, which
  the review output must state, never from the application that ran it; an unknown
  identifier gives `unknown` lineage and a `reduced` review. A draft with an open
  `blocking` finding is never accepted, and the pass limit offers a further round or a
  decision change instead.
- **Options considered:** Keeping D-0022 as written; inferring lineage from the tool.
- **Reason:** The Stage 2 Codex review found that the MCP nudge assumed an identifier
  the timeout never delivers, that tool-based lineage could label an unverified model
  independent, and that the pass limit could defer a blocking finding.
- **Consequences:** The Codex, review, lineage, architecture, and planning references
  and the review output contract say so. Core implements the MCP nudge itself.

## D-0024 — Convergence of later review passes

- **Status:** Accepted on 2026-09-05.
- **Decision:** Pass 1 of a review may report any severity. Passes 2 and 3 confirm the
  earlier repairs and decide on `blocking` findings only, and only on the material the
  revision changed, which the author lists in its notes. A `major`, `minor`, or `note`
  finding returned on a later pass is recorded as `deferred` for the user and handed to
  the planner as a task or requirement. A later-pass finding on material that was not
  changed and that no earlier pass flagged is recorded as `declined`, because the
  earlier pass cleared it and the new finding is a sign of an unreliable reviewer.
  When only deferred or declined findings remain, the decision is `approve`.
- **Options considered:** Letting every pass raise any severity, which is how the Stage
  2 reviews ran; limiting only the number of passes.
- **Reason:** Both Stage 2 reviews consumed all three passes because each revision drew
  new major findings, some on material the earlier pass had accepted. Under that
  behavior the pass and time limits cannot be met, and the reviews never converged.
- **Consequences:** The review reference, the Claude reviewer, the planner packet, and
  the method say so. Phase review in Stage 4 applies the same rule per pass.

## D-0025 — Guided production snapshots and change continuity

- **Status:** Implemented on 2026-09-05; verification and independent-review status are
  recorded in the detailed Stage 3 plan.
- **Decision:** Preserve a pending launch record before calling a producer and retain its
  patch, actor, session, launch baseline, and raw-byte input hashes before any later
  writer. Leave the source revision null for uncommitted work. Validate dependency-ready
  tasks within the current phase without calling them independently accepted. Capture
  incremental patches relative to the launch tree when earlier tasks are uncommitted.
- **Reason:** A commit hash alone cannot identify uncommitted work, and retrying an
  ambiguous writing timeout can create simultaneous writers. MCP writing timeouts pause
  until termination is confirmed; the read-only retry rule of D-0023 does not authorize
  another producer.
- **Consequences:** The production and evidence references define host checks, safe
  capture, and task-boundary resume. Prior accepted plans are retained under
  `.peerfoil/plans/`; a version 2 plan `changes` array records all five placements,
  affected and retained tasks, evidence, and review continuity. Substantive changes need
  fresh review; unchanged scheduled work may carry forward its recorded prior review.
  The original version 1 schema remains available, and validation dispatches by version.
  Old consumers need the updated schema to read version 2 plans; prior snapshots remain
  unchanged for recovery. Phase approval and guided repair stay in Stage 4.
- **Authorship:** OpenAI Codex (`openai-gpt`) authored the Stage 3 references and patches.
  This entry records implementation provenance, not self-approval. The corresponding
  Claude review records its own session and inspected revision in the detailed plan.

## D-0026 — Phase review record and item-level independence

- **Status:** Accepted on 2026-09-05.
- **Decision:** Each phase review round is recorded in one `pr-NNNN` phase review
  record under `.peerfoil/reviews/`. It holds the frozen bundle manifest with a SHA-256
  per item and a bundle digest, the author and primary reviewer of every item, the
  evidence currency the host recomputed, the open items, one `rv-` review per
  reviewer per pass, the merged shared finding list, the repair, and the decision.
  Each item's primary reviewer is the phase reviewer seat whose configured model
  maps to a lineage root different from the item's author; the other seat is
  secondary for that item. Both reviewers review the whole bundle independently in
  pass 1, then compare through the shared list in pass 2. Duplicate findings are
  merged by keeping both identifiers and both severities; a disputed `blocking`
  finding goes to the user. Transition records may name the phase review through the
  optional `refs.phase_reviews` key.
- **Options considered:** Storing the merged list inside one reviewer's record; a
  single combined review authored by the coordinator; a fixed order in which one
  reviewer sees the other's findings first.
- **Reason:** The method requires independent first passes, a shared list for
  comparison, and item-level independence for a phase that mixes authors. A separate
  record keeps the manifest, disagreement, and decision readable without PeerFoil
  and gives Core the shape it will enforce.
- **Consequences:** `common.schema.json` defines the `pr-` identifier, the transition
  schema accepts `refs.phase_reviews`, the review template gains `Item` and `Lens`
  lines, and `plugins/peerfoil/references/phase-review.md` holds the procedure. The
  manifest is inspectable, not tamper-proof; Core binds it to a commit.

## D-0027 — Repair selection and the repair task's review

- **Status:** Accepted on 2026-09-05.
- **Decision:** In Skills 0.1 the repairer is chosen by rule, not by model passes: a
  deliverable repair goes to the `repair_producer` seat, an architecture or Quality
  Contract repair returns through the architect's revise flow, and a plan repair
  through the planner's. The verifier is the phase reviewer seat whose lineage
  differs from the repairer's, using the pass reserved for verification. A fresh
  `repair-coordinator` agent proposes the bounded repair task from the agreed
  findings; the coordinating skill validates it and records it through change intake
  with `acceptance: reviewed` naming both pass-1 phase reviews, so no separate plan
  review pass is spent. One repair cycle is allowed per round; a repair never runs at
  low effort.
- **Options considered:** Spending the configured repair-selection passes on model
  votes about the repairer; running a full plan review for every repair task;
  letting the coordinator write the repair task without an agent.
- **Reason:** With two seats and the family rule, the eligible repairer and verifier
  are determined, so model passes would add time without changing the answer. The
  reviewers already agreed on the repair, which is the review the plan needs. The
  scope and paths of a repair still need judgment, which a bounded fresh agent gives
  without writing anything.
- **Consequences:** `repair_selection_passes` and `repair_selection_max_passes` stay
  in the settings for Core, which may use model help within them. The procedure is
  `plugins/peerfoil/references/repair.md`; the agent is
  `plugins/peerfoil/agents/repair-coordinator.md`. The static checks refuse a
  `repair_producer` seat at `low` effort.

## D-0028 — Lesson verification and promotion

- **Status:** Accepted on 2026-09-05.
- **Decision:** `/peerfoil:remember` records a lesson as a `candidate` with a rule,
  trigger, scope, evidence, conflicts, and proposed destination, authored by the
  user. A candidate is verified only by the fresh reviewers of the next phase review,
  which carries every candidate in its bundle and records a verdict per lesson. Only
  a `verified` lesson is promoted, and only with the user's approval, to a decision,
  a change request for a test, or proposed text in the lesson file for a skill, pack
  rule, or `AGENTS.md` change that a person applies. The exception is a `hint`,
  which the user may promote unverified with an expiry, thirty days by default.
  Active hints are listed in the producer packet and in status. PeerFoil never edits
  `AGENTS.md`, a skill, or a pack.
- **Options considered:** Letting the coordinator verify a lesson by reading the
  cited record; a separate review kind for lessons; writing proposals directly into
  `AGENTS.md` for the user to revert.
- **Reason:** The session that rewrote the lesson cannot independently verify it,
  a separate review kind would spend passes for a small check that the phase
  reviewers can make while reading the bundle, and repository rules must change only
  by a person's hand.
- **Consequences:** The lesson template gains `Verification` and `Promoted to` lines;
  `plugins/peerfoil/references/lessons.md` holds the procedure; the phase packet
  asks for lesson verdicts. Core keeps the same states.

## Conclusion

These entries settle the layout, formats, names, conventions, review mechanics, effort,
turn and time limits, deadlines, Codex access, phase review, repair, and lessons that
Phase 1 builds on. Later stages add entries for their own required decisions.

## Additional Resources

- [Implementation plan](implementation-plan.md)
- [Phase 1 implementation plan](plans/phase-1-skills.md)
- [PeerFoil architecture](architecture.md)
- [PeerFoil method](PeerFoil-Method.md)

[Return to the PeerFoil README](../README.md)

---

Copyright © 2026 Gabriel Mongefranco
