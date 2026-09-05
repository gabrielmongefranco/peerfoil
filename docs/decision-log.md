<!--
This file is part of PeerFoil.
docs/decision-log.md
Author(s): Gabriel Mongefranco.
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
| D-0005 | Detect and guide the Codex plugin instead of declaring a dependency | Accepted | Phase 1, Stage 1 |
| D-0006 | Ask `personal` or `work` and inherit the repository's `AGENTS.md` | Accepted | Phase 1, Stage 1 |
| D-0007 | Skills 0.1 always shows `Guided` assurance | Accepted | Phase 1, Stage 1 |
| D-0008 | Provisional marketplace identifier `peerfoil` | Provisional | Phase 1, Stage 1 |
| D-0009 | Packs and templates ship inside the plugin; schemas stay at the repository root | Accepted | Phase 1, Stage 1 |
| D-0010 | Template files carry a sibling notice instead of a PeerFoil header | Accepted | Phase 1, Stage 1 |
| D-0011 | Identifier and revision conventions | Accepted | Phase 1, Stage 1 |
| D-0012 | Static checks use Python 3 with no third-party packages | Accepted | Phase 1, Stage 1 |
| D-0013 | Architecture and plan reviews run through the Codex plugin's rescue agent, fresh and read-only | Accepted | Phase 1, Stage 2 |
| D-0014 | Reduced-assurance fallback when no different-family reviewer is available | Accepted | Phase 1, Stage 2 |
| D-0015 | Drafts carry the next revision number; accepted revisions live in `project.json` | Accepted | Phase 1, Stage 2 |
| D-0016 | Packs may declare evidence command hints | Accepted | Phase 1, Stage 2 |
| D-0017 | Architect defaults to high effort and every other role to medium | Accepted | Phase 1, Stage 2 |
| D-0018 | Agents declare turn limits and reviewers cap their findings | Accepted | Phase 1, Stage 2 |

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

- **Status:** Accepted on 2026-09-05.
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

- **Status:** Accepted on 2026-09-05.
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

## Conclusion

These entries settle the layout, formats, names, conventions, review mechanics, effort,
and turn limits that Phase 1 builds on. Later stages add entries for their own required decisions.

## Additional Resources

- [Implementation plan](implementation-plan.md)
- [Phase 1 implementation plan](plans/phase-1-skills.md)
- [PeerFoil architecture](architecture.md)
- [PeerFoil method](PeerFoil-Method.md)

[Return to the PeerFoil README](../README.md)

---

Copyright © 2026 Gabriel Mongefranco
