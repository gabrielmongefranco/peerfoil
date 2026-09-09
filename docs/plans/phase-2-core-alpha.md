<!--
This file is part of PeerFoil.
docs/plans/phase-2-core-alpha.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Provides the executable implementation plan for Phase 2, PeerFoil Core Alpha 0.2.
Notes: This plan expands Phase 2 of docs/implementation-plan.md without changing its scope.

Copyright © 2026 Gabriel Mongefranco

Permission is granted to copy, distribute and/or modify this document under the terms of
the GNU Free Documentation License, Version 1.3 or any later version published by the Free
Software Foundation; with no Invariant Sections, no Front-Cover Texts, and no Back-Cover
Texts. See <https://www.gnu.org/licenses/fdl-1.3.html>.
-->

# Phase 2 Implementation Plan: PeerFoil Core Alpha 0.2

## Turn the guided workflow into one small, enforced software journey

[Return to the PeerFoil README](../../README.md)

Phase 2 builds the first PeerFoil executable. It reads the files created by PeerFoil
Skills, runs one software task in a separate Git worktree, calls Claude Code and Codex as
local processes, records current evidence, and stops unsafe or incomplete transitions. It
is a narrow technical alpha, not the complete six-month product.

This plan divides Phase 2 into eight stages. Each stage is sized for about one focused day
and builds on the accepted Phase 1 contracts. Core Alpha ends with one complete software
journey, one independent Claude and Codex review, one guided repair path, and native tests
on Windows, macOS, and Linux.

## 1. Phase goal

A user can run:

```text
peerfoil init
peerfoil doctor
peerfoil start
peerfoil status
peerfoil resume
```

For one supported software project, Core Alpha then:

1. reads and validates the Skills-generated project, architecture, Quality Contract, pack,
   and plan;
2. selects the next eligible task;
3. invokes one qualified producer in a dedicated Git worktree;
4. captures the produced change and its authorship;
5. runs the required project commands itself;
6. ties evidence to the exact Git revision;
7. integrates only in-scope work with passing required evidence;
8. runs one fresh Claude and Codex phase review;
9. guides one repair and independent verification when needed; and
10. resumes safely after an interruption at a completed task boundary.

## 2. Start gate from Phase 1

Do not start Core Alpha until Phase 1 has accepted versions of:

- project, decision, architecture, Quality Contract, plan, task, change-set, evidence,
  finding, review, and lesson records;
- JSON schemas and human-readable templates;
- Software, Generic, and Documentation pack manifests;
- two software fixtures and one documentation fixture;
- role and model-lineage rules;
- tested Claude Code and Codex invocation behavior;
- the complete Phase 1 evidence summary; and
- known limitations and deferred work.

Run the Phase 1 acceptance suite first. If a Core requirement exposes a contract problem,
update the Phase 1 template, schema, fixture, and documentation together before Core uses
the new version.

## 3. Release boundary

### Included

- One small Go command-line application.
- Native development builds for Windows, macOS, and Linux.
- The five initial commands listed above.
- A deterministic controller for one sequential Software Pack path.
- Validation of the Phase 1 project, pack, plan, and result records.
- Direct Claude Code and Codex process adapters using their existing authentication.
- One supported default model and effort arrangement with visible failure.
- One task per model call and one writing agent at a time.
- A dedicated task worktree and integration branch.
- Controller-run executable evidence tied to exact revisions.
- Durable artifact-, patch-, invocation-, and reviewer-lineage records.
- A small redacted transition history.
- Recovery at completed task boundaries.
- One fixed Claude and Codex phase-review round.
- One guided repair and fresh different-family verification.
- Processing of the Phase 1 Documentation fixture through the same controller contracts.

### Not included

- SQLite; it arrives with Reliable Core after the file journal is proven.
- Multi-round review reconciliation or automatic repair-agent consensus.
- Fine-grained change-impact analysis and selective invalidation.
- Automatic skill selection or role-scoped MCP configuration.
- Shared memory promotion or cross-project lessons.
- Ollama, vLLM, OpenCode, or arbitrary provider adapters.
- Automatic scanner installation.
- Parallel writers, submodule support, Git LFS support, a GUI, or deployment.
- An operating-system sandbox or a claim that worktrees provide security isolation.
- Mature non-coding project experiences.

## 4. Technical rules for Core Alpha

- Use Go's standard library unless a small dependency clearly lowers risk.
- Use `flag.FlagSet` or an equally small standard approach for the five commands. Do not
  add a large CLI framework without an accepted reason.
- Pass subprocess arguments as arrays. Do not assemble shell command strings.
- Keep provider-specific parsing behind adapters.
- Keep Software Pack behavior outside the controller.
- Use stable identifiers for projects, plan revisions, phases, stages, tasks, requirements,
  invocations, changes, evidence, findings, reviews, and model sessions.
- Persist an intent before an outside operation and its result afterward.
- Write accepted files atomically where the operating system allows it. A failed write
  must leave either the old complete file or the new complete file, never a partial file.
- Store accepted project truth in Git under `.peerfoil/`.
- Store temporary operational data outside Git in the operating system's user-state or
  cache directory, keyed by repository identity.
- Never store raw prompts, full transcripts, provider tokens, or private MCP payloads.
- Stop rather than guess when state, identity, lineage, evidence, or permission is unclear.

## 5. Planned source layout

```text
cmd/peerfoil/
  main.go
internal/cli/
internal/controller/
internal/project/
internal/pack/
internal/provider/
  claude/
  codex/
internal/workspace/
internal/evidence/
internal/review/
internal/store/
internal/redact/
schemas/
packs/
fixtures/
tests/
```

Keep interfaces close to the package that calls them. Add one only when a production and
test implementation, or two real production implementations, need the boundary.

## 6. Core records and states

Core Alpha uses the Phase 1 record shapes. Its controller adds only the operational fields
needed to enforce and recover the workflow.

### Workflow states

```text
Ready → Producing → Produced → Validating → Validated
      → Reviewing → RepairNeeded → Repairing → Approved
```

Any state may move to `Paused` when the user, authentication, a permission, or an unknown
result blocks safe progress. Core Alpha resumes only from a recorded safe boundary.

### Transition record

Each accepted transition includes:

- transition and project identifiers;
- prior and next state;
- plan, task, and source revisions;
- invocation identifier when an outside process ran;
- input and result hashes;
- author and model lineage when work was produced;
- evidence or review references;
- timestamp in UTC; and
- a redacted reason.

### Invocation record

An invocation begins as `prepared`, then becomes `started`, `succeeded`, `failed`,
`cancelled`, or `unknown`. After an interruption, Core never reruns an `unknown` mutating
invocation automatically. It inspects the workspace and asks the user when it cannot prove
the result.

## 7. Stage overview

| Stage | Outcome | Depends on |
|---:|---|---|
| Stage 1 | Compiling CLI shell and accepted contract decisions | Phase 1 handoff |
| Stage 2 | Validated project state, deterministic transitions, and file journal | Stage 1 |
| Stage 3 | Tested Claude Code and Codex process adapters | Stages 1–2 |
| Stage 4 | Safe Git worktree, change-scope, and provenance handling | Stages 1–2 |
| Stage 5 | Controller-run evidence tied to exact revisions | Stages 2 and 4 |
| Stage 6 | One complete plan-to-integrated-change journey | Stages 2–5 |
| Stage 7 | Independent phase review and guided repair | Stage 6 |
| Stage 8 | Recovery, documentation fixture, three-platform tests, and alpha release | Stages 1–7 |

## 8. Stage 1 — CLI shell and contract decisions

### Outcome

The repository builds one `peerfoil` binary. The five commands have stable help text and
exit-code rules. Architecture decisions required for Core are recorded before deeper code
depends on them.

### Tasks

1. Create the Go module and planned source directories.
2. Implement the command dispatcher and the five command stubs.
3. Define global behavior for:
   - `--help` and `--version`;
   - human-readable and future machine-readable output;
   - standard input and non-interactive use;
   - cancellation;
   - working-directory selection; and
   - exit codes.
4. Create stable error categories for usage, configuration, authentication, validation,
   policy, provider, Git, evidence, timeout, cancellation, and internal errors.
5. Record architecture decisions for:
   - the JSON Schema validation library;
   - atomic file replacement on all three operating systems;
   - the local state-directory convention;
   - process cancellation boundaries in Alpha; and
   - the exact Claude Code and Codex structured-output interfaces tested.
6. Add build, unit-test, formatting, vetting, and license-header checks.
7. Add Windows, macOS, and Linux continuous-integration jobs.
8. Implement version metadata without requiring a network call.

### Verification

- Run formatting, unit tests, vetting, and builds locally.
- Build and run help for every command on Windows, macOS, and Linux.
- Confirm unknown commands and invalid flags return stable nonzero exit codes.
- Confirm help text uses complete, plain-language instructions.
- Confirm the binary starts without Docker, Bash, PowerShell, WSL, or a hosted service.

### Stage 1 is complete when

- all five commands compile and expose stable help;
- the selected dependencies pass GPLv3 compatibility review;
- the architecture decisions above are accepted; and
- the initial three-platform CI matrix passes.

## 9. Stage 2 — Project validation, state transitions, and journal

### Outcome

Core can find a PeerFoil project, validate its accepted files, report its state, and advance
only through permitted transitions. An interruption leaves enough local information to
identify the last safe task boundary.

### Tasks

1. Implement repository discovery without assuming a home directory, drive letter,
   case-sensitive filesystem, or symlink support.
2. Load and validate:
   - `.peerfoil/project.json`;
   - decisions and architecture status;
   - the Quality Contract;
   - `plan.json` and its human-readable plan partner;
   - the selected project pack; and
   - schema and plan revisions.
3. Reject unknown required fields, unsupported schema versions, duplicate identifiers,
   missing dependencies, cycles, stale tasks, and invalid state transitions.
4. Implement the deterministic controller state enum and transition table.
5. Implement `init` to create a new project only after showing the files it will add.
   Never overwrite an existing `.peerfoil/` directory.
6. Implement `status` with the assurance, phase, stage, task, quality state, blocker,
   pending decision, and next action.
7. Create the local write-ahead journal outside Git.
8. Append small redacted accepted-transition records to `.peerfoil/history.jsonl`.
9. Implement atomic accepted-file writes and interruption tests.
10. Implement `resume` for safe completed-task boundaries only.

### Verification

- Load every valid Phase 1 fixture.
- Reject malformed JSON, unsupported schema versions, missing task dependencies, cycles,
  duplicate IDs, and a stale plan revision.
- Interrupt writes before and after replacement; confirm no accepted file is partial.
- Interrupt a prepared, started, and completed invocation; confirm each recovery decision
  is safe and visible.
- Confirm status works offline and does not invoke a model.
- Confirm journal and error output contain no prompt, token, or fixture secret.

### Stage 2 is complete when

- valid Phase 1 state loads without reinterpretation;
- invalid or stale state fails closed with one useful next action;
- transition tests cover every allowed and forbidden edge; and
- resume never repeats an unknown mutating operation automatically.

## 10. Stage 3 — Claude Code and Codex adapters

### Outcome

Core can check the installed tools, start a fresh non-interactive model session, request a
schema-shaped result, stream bounded events, capture metadata, and cancel safely within
the documented Alpha boundary.

### Tasks

1. Define the small provider adapter contract:

   ```text
   probe → prepare request → start fresh process → read events
   → validate final result → record metadata → cancel or finish
   ```

2. Implement a shared process runner with:
   - argument arrays;
   - explicit working directory;
   - a minimal environment allowlist;
   - separate standard output and error capture;
   - bounded output size;
   - context cancellation and timeout, applied as a two-stage deadline: at the soft
     limit the adapter sends "answer now" and waits one minute; at the hard limit it
     kills the process (D-0022);
   - no shell interpolation; and
   - redacted diagnostic output.
3. Implement the Claude Code adapter using the currently documented non-interactive JSON
   and JSON Schema options.
4. Implement the Codex adapter using `codex exec` and its currently documented output
   schema support.
5. Parse provider envelopes separately from PeerFoil result schemas.
6. Record provider, tool version, model identifier, effort setting, session identifier,
   lineage root, timestamps, and exit result.
7. Probe capability and authentication before starting work.
8. Make unavailable models, unsupported effort, missing authentication, malformed output,
   timeouts, and unexpected termination distinct errors.
9. Start a fresh session for every authoring or review role. Do not reuse a producer's
   conversation for review.
10. Add fake-process adapters for deterministic unit and integration tests.

### Verification

- Test successful, malformed, truncated, oversized, timed-out, cancelled, and nonzero-exit
  responses from both adapters.
- Test paths and prompts containing spaces, Unicode, apostrophes, quotes, and newlines.
- Confirm no prompt text enters normal logs.
- Confirm a tool-version change is visible in diagnostics and evidence.
- Confirm an unsupported model or effort never receives a silent fallback.
- Run one real read-only structured-output probe for each installed provider on all three
  operating systems.

### Stage 3 is complete when

- both adapters pass the same contract suite;
- a fresh session and exact lineage are recorded for each call;
- bad provider output cannot change accepted state; and
- cancellation stops the direct provider process and reports any child-process limitation
  honestly.

## 11. Stage 4 — Git worktrees, scope, and provenance

### Outcome

Core prepares one task in a separate worktree, preserves the producer's original patch,
detects out-of-scope changes, and integrates accepted work into a PeerFoil integration
branch without modifying the user's current branch.

### Tasks

1. Require an existing Git repository with at least one commit and a clean starting
   checkout.
2. Detect and reject unsupported Alpha cases, including submodules and Git LFS when they
   affect the planned task.
3. Create stable task and integration branch names from validated identifiers.
4. Place worktrees in a safe temporary or configured directory that works on all three
   operating systems.
5. Record the base commit, active plan revision, task identifier, and allowed paths before
   production.
6. Capture changed, added, deleted, renamed, and mode-changed files after production.
7. Reject changes outside the task's allowed paths unless the user accepts a revised plan.
8. Scan changed content and paths for obvious credentials and private material before
   integration.
9. Save the original patch hash, author session, model family, and affected artifacts
   before any coordinator or conflict-resolution edit.
10. Record conflict-resolution authorship separately.
11. Integrate only to a dedicated PeerFoil integration branch. Do not merge to the user's
    branch without explicit authorization.
12. Clean up only worktrees and branches that Core created and can identify exactly.

### Verification

- Test add, modify, delete, rename, binary, executable-bit, and case-only changes.
- Test paths with spaces, Unicode, apostrophes, long names, and mixed separators.
- Seed an out-of-scope file and confirm integration stops.
- Seed a likely secret and confirm the change is blocked or requires explicit safe review.
- Change the user's current checkout during a task and confirm Core stops.
- Cause a merge conflict and confirm both the producer and resolver remain recorded.
- Confirm cleanup never removes an unknown worktree or branch.

### Stage 4 is complete when

- the producer works only in the dedicated task worktree;
- authorship survives integration and conflict handling;
- scope and secret checks block seeded violations; and
- the user's current branch remains unchanged.

## 12. Stage 5 — Controller-run evidence

### Outcome

Core runs the task's required commands itself and creates evidence records tied to the
exact produced revision. A required failure or missing result blocks integration and
review.

### Tasks

1. Read declared validators and evidence requirements from the Software Pack and active
   task.
2. Validate every executable evidence command, argument, working directory, timeout, and
   environment entry before running it.
3. Reuse the safe process runner from Stage 3 without invoking a shell.
4. Record:
   - project, plan, task, and source revisions;
   - command arguments and working directory;
   - exit code and duration;
   - tool version;
   - configuration and relevant input hashes;
   - retained output hash and location; and
   - redaction status.
5. Support executable, inspectable, and human evidence records without treating a model
   claim as evidence.
6. Mark each item required, recommended, or not applicable.
7. Block advancement when required evidence is failed, missing, stale, or tied to another
   revision.
8. Limit retained output and redact credentials, user content, and private paths where
   practical.
9. Rerun affected evidence after a repair.
10. Make the evidence summary understandable from `peerfoil status`.

### Verification

- Run passing, failing, timed-out, cancelled, and missing commands.
- Change a file after a passing test and confirm the prior evidence becomes stale.
- Supply evidence from another task or revision and confirm rejection.
- Seed a token in command output and confirm it does not enter the retained record.
- Confirm a recommended failure is visible but follows the Quality Contract's policy.
- Confirm a required human check pauses for the user instead of fabricating completion.

### Stage 5 is complete when

- all evidence records identify the exact revision they support;
- required failures block the workflow;
- stale or mismatched evidence cannot be accepted; and
- retained diagnostics are useful without exposing sensitive content.

## 13. Stage 6 — Enforced vertical slice

### Outcome

The controller runs one complete planned coding task from validated input through an
accepted integration-branch change. This is the first end-to-end enforced PeerFoil path.

### Tasks

1. Connect project loading, task selection, provider invocation, worktree handling,
   provenance, evidence, and transition recording.
2. Implement `doctor` with checks for:
   - supported operating system and architecture;
   - repository and Git state;
   - project files and schema versions;
   - Claude Code and Codex availability;
   - authentication and structured-output capabilities;
   - selected model and effort support;
   - pack-required project tools; and
   - writable local state and temporary directories.
3. Implement `start` for one already-architected and planned software phase.
4. Keep task selection deterministic. Do not let a model choose an ineligible task.
5. Use the producer seat's configured effort, medium by default, for the coding producer.
6. Require an accepted, different-family-reviewed architecture and plan before production.
7. Stop after one task if the next transition needs a user decision or falls outside the
   Alpha path.
8. Add a single end-to-end command result that explains the accepted change, evidence,
   branch, remaining work, and next action.
9. Add deterministic fake-provider end-to-end tests and one real-provider smoke test.

### Verification

- Complete the Go software fixture from plan to accepted integration-branch change.
- Confirm production cannot begin with an unreviewed architecture or plan.
- Confirm an ineligible task, stale revision, failed check, provider error, or out-of-scope
  patch stops at the correct transition.
- Confirm repeated `start` or `resume` does not duplicate accepted work.
- Confirm status is correct before, during, after, and between every transition.
- Confirm normal output gives the user only the goal, state, blocker, and next action.

### Stage 6 is complete when

- one supported software task completes without manual file repair;
- every accepted transition has current evidence and provenance;
- every seeded invalid transition stops safely; and
- the integration result can be understood and reviewed after a fresh clone.

## 14. Stage 7 — Independent review and guided repair

### Outcome

Core freezes the completed phase, starts fresh Claude and Codex review sessions, records
their findings, blocks self-approval, and guides one repair with fresh
different-family verification.

### Tasks

1. Build the frozen review bundle from the exact integration-branch revision:
   - accepted decisions, architecture, Quality Contract, and plan;
   - task and change-set records;
   - deliverables and diffs;
   - current evidence;
   - known risks, TODOs, deviations, and deferrals; and
   - artifact- and patch-level authorship.
2. Start one fresh Claude reviewer and one fresh Codex reviewer at medium effort when
   supported.
3. Give both reviewers the same bundle and no producer transcript.
4. Validate findings and normalize their location, requirement, severity, evidence,
   recommendation, and disposition.
5. Assign primary-review eligibility for each material item by model family.
6. Block an author's session and model family from satisfying independent approval of its
   own work.
7. Implement one fixed review round. Do not add Phase 4's multi-pass reconciliation.
8. When both reviews identify an accepted repair, prepare one repair task.
9. Capture repair authorship, rerun affected evidence, and start a fresh eligible reviewer
   from another family.
10. Stop for the user when the reviewers disagree materially or one repair does not clear
    the blocking finding.

### Verification

- Seed a code defect, missing requirement, stale test, and self-approval attempt.
- Confirm both reviewers inspect the same commit and evidence.
- Confirm Claude cannot independently approve Claude-authored artifacts and Codex cannot
  independently approve Codex-authored patches.
- Confirm a same-family review remains a secondary critique.
- Apply one repair and confirm new evidence and review bind to the repaired commit.
- Force unresolved disagreement and confirm Core pauses without invented consensus.

### Stage 7 is complete when

- the fixture completes one two-family review round;
- self-approval is blocked per artifact and patch;
- a seeded repair completes with fresh independent verification; and
- unresolved review produces one clear user decision instead of another automatic loop.

## 15. Stage 8 — Recovery, portability, fixtures, and alpha release

### Outcome

Core Alpha meets its full documented boundary on Windows, macOS, and Linux. The software
journey and Documentation fixture are repeatable from a fresh checkout, and every deferred
feature remains clearly labeled.

### Tasks

1. Add interruption tests at every completed task boundary and every recorded invocation
   state.
2. Add reconstruction tests using a fresh clone with no provider session or local state.
3. Process the Phase 1 Documentation fixture through project, plan, evidence, provenance,
   and review contracts without document-specific controller branches.
4. Complete Windows, macOS, and Linux tests for:
   - builds and unit tests;
   - process invocation and cancellation;
   - atomic file writes;
   - Git worktrees and cleanup;
   - path and encoding edge cases;
   - evidence hashes and redaction; and
   - complete fake-provider end-to-end journeys.
5. Run manual real-provider smoke tests on all three operating systems.
6. Write installation, quick-start, command, configuration, troubleshooting, recovery,
   privacy, security, accessibility, licensing, and Alpha limitation documentation.
7. Add checksums for development artifacts if binaries are shared. Do not call them stable
   installers.
8. Run dependency vulnerability and GPLv3 compatibility checks.
9. Run independent Claude and Codex phase review against the complete release candidate.
10. Fix blocking findings, rerun the full matrix, and create the Phase 2 evidence summary.

### Verification

- Interrupt before production, after production, after evidence, after integration, and
  after review; confirm only documented safe boundaries resume automatically.
- Clone the repository without local state and reconstruct accepted status and provenance.
- Complete the Go and Node.js fixtures on all three operating systems.
- Process the Documentation fixture without a controller condition on its pack name.
- Confirm no Docker, WSL, Bash, PowerShell, database server, or paid non-LLM account is
  required.
- Confirm worktrees are described as change isolation, never as a security sandbox.
- Confirm help and documentation call the release **Core Alpha** and identify every major
  deferral.

### Stage 8 is complete when

- the full platform and failure matrix passes;
- fresh-clone reconstruction retains accepted state and review eligibility;
- the Documentation fixture proves the controller is not hard-coded to source files;
- the final review has no unresolved blocking finding; and
- the Reliable Core handoff identifies every known limit and next dependency.

## 16. Phase acceptance matrix

| ID | Required result | Evidence |
|---|---|---|
| P2-01 | Skills-generated project, architecture, pack, and plan load | Schema and fixture suite |
| P2-02 | One producer task runs in a dedicated worktree | Worktree and invocation records |
| P2-03 | Commands run under controller ownership | Evidence record from exact revision |
| P2-04 | Only in-scope work with passing evidence integrates | Positive and negative integration tests |
| P2-05 | Completed task boundaries survive interruption | Recovery test matrix |
| P2-06 | Out-of-scope and live-checkout mutations block | Seeded mutation tests |
| P2-07 | One Claude and Codex phase review preserves independence | Frozen bundle and lineage matrix |
| P2-08 | One repair receives independent verification | Repair commit, evidence, and review |
| P2-09 | Documentation fixture uses shared controller code | Fixture trace and coverage review |
| P2-10 | Accepted state reconstructs from Git after a fresh clone | Reconstruction test |
| P2-11 | Windows, macOS, and Linux pass the Alpha matrix | CI and manual smoke-test record |
| P2-12 | Deferred capabilities remain absent and documented | Scope and documentation audit |

Every row is required. A passing fake-provider test does not replace the planned real-tool
smoke test, and a model's claim does not replace controller-run evidence.

## 17. Test strategy

### Unit tests

- schemas and record validation;
- controller transitions and forbidden edges;
- identifiers, revisions, and lineage eligibility;
- evidence freshness and hashing;
- redaction;
- path validation; and
- error classification.

### Integration tests

- process runner and fake providers;
- Claude Code and Codex adapter envelopes;
- Git worktrees, branches, diffs, and cleanup;
- atomic writes and journal recovery;
- executable evidence; and
- fresh-clone reconstruction.

### End-to-end tests

- successful software task;
- failed required check;
- malformed provider output;
- stale plan;
- out-of-scope mutation;
- interruption and resume;
- two-family review;
- guided repair and re-verification; and
- Documentation fixture.

Run `go test ./...`, `go vet ./...`, formatting checks, repository conformance, fixture
checks, dependency review, and platform-specific builds. Add race detection where the
supported Go toolchain and runner allow it. Record any platform exception rather than
silently dropping the check.

## 18. Security and accessibility review

Core Alpha's security review covers:

- command injection and shell avoidance;
- path traversal and repository escape;
- unsafe environment inheritance;
- secret and personal-data retention;
- untrusted model and MCP-style content;
- Git target validation and destructive cleanup;
- dependency vulnerabilities and licenses;
- fail-closed state and evidence handling; and
- explicit approval for external or destructive effects.

The CLI accessibility review covers:

- clear headings and status labels in text output;
- no meaning conveyed by color alone;
- useful output with color disabled and redirected to a file;
- keyboard-only operation;
- readable errors with one next action;
- stable plain-text and future machine-readable output; and
- documentation with descriptive links and text explanations for diagrams.

## 19. Risks and responses

| Risk | Response |
|---|---|
| Eight stages expand into the full product | Enforce one software path and the explicit deferral list |
| Phase 1 contracts do not fit code | Version and repair the shared contract instead of creating hidden Core-only shapes |
| Provider output changes | Separate provider envelopes, validate results, and record tested versions |
| Process cancellation behaves differently by OS | Test the same contract on all three systems and document Alpha's child-process limit |
| Atomic replacement differs on Windows | Decide and test the replacement strategy before state depends on it |
| Worktree cleanup removes user work | Tag every owned resource and delete only exact verified targets |
| Evidence output leaks secrets | Bound, redact, hash, and minimize retained output |
| Review logic grows into Phase 4 | Keep one fixed round and one guided repair path |
| Fake adapters hide real failures | Require real-tool smoke tests in addition to deterministic CI |
| Documentation fixture invites a second product | Use it only to prove shared controller contracts |

## 20. Reliable Core handoff

Before Phase 3 begins, commit and review:

- the accepted Go package and command contracts;
- versioned record and schema changes;
- the complete transition table;
- file-journal and recovery behavior;
- Claude Code and Codex adapter contracts and tested versions;
- Git worktree and provenance rules;
- evidence formats and redaction rules;
- review and guided-repair records;
- all test fixtures and the three-platform matrix;
- the Phase 2 evidence summary;
- every unsupported Git, provider, process, and recovery case; and
- the work needed before SQLite becomes a reconstructible cache.

Phase 3 may strengthen recovery and add SQLite. It must keep Git and accepted `.peerfoil/`
files as the source of project truth.

## 21. Definition of done

Phase 2 is complete only when:

1. Stages 1–8 meet their completion checks.
2. P2-01 through P2-12 have current evidence.
3. One real software task completes through the enforced Alpha path.
4. Required failures, stale plans, malformed output, and out-of-scope changes stop safely.
5. Artifact- and patch-level authorship survives integration and reconstruction.
6. No agent or model family independently approves its own material work.
7. One repair completes at the repair producer's effort with fresh different-family verification.
8. The accepted project state survives a fresh clone without private transcripts or local
   provider sessions.
9. Windows, macOS, and Linux pass the supported matrix.
10. Core requires no paid non-LLM service, container, or database server.
11. Documentation and help describe Alpha's real behavior and limitations.
12. An independent phase review recommends release with no unresolved blocking finding.

## Conclusion

Phase 2 proves that PeerFoil's most important rules can be enforced by a small local
controller. It keeps the first automated journey deliberately narrow: one planned software
task, one writer, current evidence, one independent review round, one guided repair, and
safe task-boundary recovery. That boundary gives Phase 3 a dependable base without hiding
unfinished work behind the word “automatic.”

## Additional Resources

- [PeerFoil implementation plan](../implementation-plan.md)
- [Phase 1 implementation plan](phase-1-skills.md)
- [PeerFoil method](../PeerFoil-Method.md)
- [PeerFoil architecture](../architecture.md)
- [Phase prompt template](../phase-prompt-template.md)
- [Claude Code headless mode](https://code.claude.com/docs/en/headless)
- [Claude Code CLI reference](https://code.claude.com/docs/en/cli-reference)
- [Codex non-interactive mode](https://developers.openai.com/codex/non-interactive-mode)
- [Codex developer commands](https://developers.openai.com/codex/developer-commands)
- [Git worktree documentation](https://git-scm.com/docs/git-worktree)

[Return to the PeerFoil README](../../README.md)

---

Copyright © 2026 Gabriel Mongefranco
