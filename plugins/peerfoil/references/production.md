<!--
This file is part of PeerFoil.
plugins/peerfoil/references/production.md
Author(s): Gabriel Mongefranco; OpenAI Codex.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Guides one bounded production task, captures provenance, and resumes from retained files.
Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# Production reference (Guided)

Use from `start` or `resume` after planning. Read [records.md](records.md),
[lineage.md](lineage.md), [codex.md](codex.md), and [evidence.md](evidence.md).
This is a written host procedure, not a controller or a crash-recovery guarantee.

## 1. Gate and select one task

1. Read repository and applicable nested `AGENTS.md` files first. Validate project,
   plan, history, decisions, architecture, Quality Contract, and the latest review of
   each accepted draft. Apply the shape and semantic checks in
   [planning.md](planning.md), section 3, with the existing task statuses preserved.
   Require accepted architecture, quality, and plan with matching project revisions,
   recorded user acceptance, and eligible independent reviews or the user's recorded
   Reduced assurance decision for each draft. Check review time against the draft's
   last substantive edit, not subsequent status-only updates. No open consequential
   decision or unresolved blocking finding may remain.
2. A revised plan must also have the acceptance chain in [changes.md](changes.md).
   Never treat an old review as approval of changed requirements. If files disagree,
   stop with the exact mismatch and one recovery action; do not infer acceptance.
3. Read all tasks and the active phase/stage/task identifiers. Continue an existing
   `in_progress`, `produced`, or `validated` task using section 5 before choosing a new
   one. Never launch a second writer. Select the first `planned` or `ready` task in
   the user's authorized stage order whose dependencies are satisfied. A task or stage
   in another phase needs phase approval and authorization to begin that phase.
4. `validated` predecessors may supply inputs within the active phase when their
   current evidence passes; this is readiness for dependent work, not independent
   acceptance. `accepted` predecessors must have eligible review records. Do not set
   tasks `accepted`, stages `approved`, or phases `approved` during production.
   Stale, blocked, deferred, removed, or revision-mismatched work is not ready.
5. Before the call, inventory tracked, staged, unstaged, and untracked files. Require
   a clean deliverable baseline, or an exact retained snapshot of previously validated
   tasks whose uncommitted changes are fully attributed. Known coordinator edits under
   `.peerfoil/` are allowed only after their exact contents are inventoried. Stop for unexplained dirty work;
   never stash, discard, commit, or overwrite it. Require a Git base commit. A user
   may create that baseline; do not commit without authorization. Use a temporary Git
   index to snapshot the launch deliverables as a tree and record its tree hash in
   `Baseline`, alongside prior change-set IDs. Leave the user's index untouched.
6. Resolve the configured producer seat and login. This build supports `codex-cli`
   production only. An unavailable seat or permission blocks the task; do not silently
   substitute a writer, effort, or model. Apply repository effort rules before defaults.

## 2. Persist the launch, then send the packet

Allocate a change-set ID and create `.peerfoil/evidence/cs-NNNN.md` from its template
before invoking the producer. Record `Capture status: pending`, task, base commit,
plan/architecture/quality revisions, intended actor, start time, attempt number, and
`null` for unknown session, patch, snapshot, and captured time. Set task `in_progress`,
workflow `produce`, and active identifiers; regenerate `plan.md` and append history with
`refs.tasks` and `refs.change_sets`. Status changes alone do not revise the plan.

Build one compact transfer packet containing:

- relevant repository rules and the applicable instruction-file paths;
- accepted decisions and architecture excerpts needed by this task;
- project ID, active phase/stage/task, revisions, base commit, and change-set ID;
- exact task scope, inputs, allowed paths, expected output, and acceptance criteria;
- required evidence procedures from the accepted Quality Contract; and
- the handoff shape below and the producer restrictions in this section.

Do not send old chats, review deliberations, credentials, or unrelated private context.
Record model and lineage from configuration and returned tool metadata following the
lineage reference. A self-report is only a cross-check; `default` remains `unknown`.

Call the Codex MCP `codex` tool with `sandbox: workspace-write`, `approval-policy:
never`, and the selected repository as `cwd`, or use the `codex exec` fallback with
`--sandbox workspace-write`. Use only the configured effort and model. One fresh call
gets exactly one task. The producer may edit only approved task paths, must read their
applicable rules, and must not edit project records, Git history, `AGENTS.md`, secrets,
or provider settings. No commit, push, network publication, deployment, deletion of
unrelated work, or permission expansion is granted by the packet. A blocked action is
reported to the host. Producer-run checks are claims until the host reruns them.

Ask for exactly one fenced JSON object:

```json
{
  "task": "tk-001",
  "plan_revision": 1,
  "architecture_revision": 1,
  "base_revision": "<full commit hash>",
  "result": "produced",
  "changed_paths": ["src/example.txt"],
  "summary": "One bounded change.",
  "checks_claimed": [],
  "discovered_work": [],
  "blocker": null
}
```

`result` is `produced`, `partial`, or `blocked`. Require every key and its shown type;
`blocker` is a string or null, and both remaining arrays contain short strings. Reject
extra keys, wrong revisions/ID/base, duplicate or unsafe paths, and oversized output
(more than 32 KiB or a string over 2,000 characters). Check paths against Git, never
against the author's list alone. Missing or malformed output cannot report success.

## 3. Capture before another writer

Once the process is confirmed stopped, compare the entire workspace with the launch
inventory, including staged and untracked files and protected project records. Validate
every actual changed path against the allowed patterns using case-insensitive matching:
`*` and `?` do not cross `/`; `**` matches zero or more directory components. Check both
sides of renames and deletions. Reject absolute, parent-traversal, drive, UNC, alternate
data stream, backslash, credential, `.git`, and `AGENTS.md` paths. Reject symlink or
junction components, submodules, and case-colliding paths. Resolve existing ancestors
for new/deleted paths and require containment inside the selected repository. Check
containment before reading a new path, not just before writing it.

Capture a binary-capable Git patch relative to the recorded launch tree, not the older
HEAD when previous tasks are uncommitted. Include new untracked task files using Git's
no-index diff or a temporary index with
argument arrays. Do not stage into the user's index to capture new files. Retain the
patch under `.peerfoil/evidence/cs-NNNN.patch`, its SHA-256, and the file snapshot defined
in the evidence reference. Git path output must be NUL-delimited and decoded without
shell splitting, so spaces, Unicode, apostrophes, and CRLF remain intact.

Inspect the patch and any retained output for secrets and private material before saving
them. If unsafe content or out-of-scope edits exist, preserve the workspace, record a
redacted blocked capture, and stop for the user. Do not copy unsafe content into evidence
or silently redact an executable patch and claim it still represents the work.

Only after the actual patch, snapshot, changed paths, observed session, lineage, and
capture time are retained, set `Capture status: captured`. A missing session or uncertain
authorship remains an explicit blocker to normal completion. An unknown family remains
Reduced assurance. Capture partial/failed output too; it never authorizes another edit.
An empty or denied attempt with no patch or snapshot stays `blocked`, never `captured`.
Record any later edit in a separate change set with its own author and base snapshot,
and link it under `Later edits by other agents`. Never overwrite the original capture.

For a valid complete handoff, set task `produced`, workflow `validate`, and append the
transition referencing the capture. Then follow [evidence.md](evidence.md). For discovered
work, use [changes.md](changes.md) before continuing; a changed revision invalidates the
old packet. Capturing authorship is not accepting the result.

## 4. Bound failures and retries

Use the ten-minute task limit from the Codex reference. A writing timeout is ambiguous:
it does not prove the writer stopped. For MCP timeout with no thread ID, pause and ask
the user to confirm termination through the provider before inspecting or retrying.
For CLI timeout, stop the original process and descendants through the host, confirm
termination, and only then use a one-minute same-thread answer-only nudge when supported.
No nudge may write more files. If termination is uncertain, pause; never start a retry.

One no-result retry is allowed only after confirmed termination, capture of any partial
work, and a verified unchanged clean baseline. Do not automatically reset partial work.
Malformed output permits one read-only request for a corrected handoff; it does not
permit another production call. Authentication and permission failures are not retried.
Retain attempt counts and the pause reason in the change set and history across chats.
Never reset the retry allowance on resume.

## 5. Resume from records

| Retained state | Next action |
|---|---|
| Accepted plan; no active task | Recheck section 1 and select one task |
| Pending capture or task `in_progress` | Confirm the old writer stopped; inspect the launch record and actual diff; capture attributable work or pause if authorship cannot be recovered |
| Captured change and task `produced` | Verify snapshot and revisions, then run host evidence; do not call the producer again |
| Task `validated` | Verify current evidence, then select the next dependency-ready task within the authorized phase |
| Missing capture, changed hashes, stale task, or disagreeing records | Report the blocker; retain files and route changed work through change intake |
| All active-phase tasks validated | Refresh earlier evidence affected by subsequent tasks against the final snapshot; only then set workflow `review`, phase/stages `review`, report phase review Coming soon, and stop |

Read accepted files, snapshots, and history, never an old conversation or provider
session to reconstruct task state. A provider session may corroborate identity but is
not the project handoff. An interrupted multi-file record update is a visible mismatch
requiring reconciliation, not a successful transition. Do not start the next phase.
