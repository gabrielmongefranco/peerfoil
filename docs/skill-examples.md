<!--
This file is part of PeerFoil.
docs/skill-examples.md
Author(s): Gabriel Mongefranco.
Created: 2026-10-21
Last Modified: 2026-10-21
Summary: Optional worked examples for authoring a reusable repository skill. These are documentation, not installed skills.

Copyright © 2026 Gabriel Mongefranco

Permission is granted to copy, distribute and/or modify this document under the terms of
the GNU Free Documentation License, Version 1.3 or any later version published by the Free
Software Foundation; with no Invariant Sections, no Front-Cover Texts, and no Back-Cover
Texts. See <https://www.gnu.org/licenses/fdl-1.3.html>.
-->

# PeerFoil

## Skill authoring examples

[Return to the PeerFoil README](../README.md)

This page shows two small example skills you can adapt when adding a reusable skill to a
repository. They are starting points for your own commands and rules, not installed
skills, and copying them unchanged into every project adds noise instead of help. Start
from [the skills folder guide](../skills/README.md) and the copyable
[skill template](../skills/skill-template.md), then shape one of these examples around
your project's real tools.

### Example: release checklist skill

A skill like this fits a project that publishes versioned releases. Replace the commands
with the ones your project actually runs.

```markdown
---
name: release-checklist
description: Run the checks a release needs, in order, and stop at the first failure. Apply when the user asks to prepare, tag, or publish a release.
---

# Example Project

## Release checklist

### Workflow

1. Confirm the working tree is clean with `git status --porcelain`.
2. Run the full test suite: `go test ./...`.
3. Run the license and notice check: `python tests/static_checks.py`.
4. Update the version and changelog, and show the diff to the user.
5. Stop. Tagging and publishing are destructive external actions the user runs.

### Constraints

Never tag, push, or publish. Never skip a failed check by rerunning only part of it.

### Verification

Every command above exited with status 0, and the user saw the version diff.
```

### Example: data export skill

A skill like this fits a project that produces data extracts for other people. It shows
how a skill carries domain rules, such as privacy checks, that a generic agent would not
know to apply.

```markdown
---
name: data-export
description: Produce a reviewed CSV extract from the project database. Apply when the user asks for an export, extract, or report file.
---

# Example Project

## Data export

### Workflow

1. State the grain of the extract in one sentence and confirm it with the user.
2. Write the query with explicit column names; never use SELECT *.
3. Run the query against the reporting copy, never production.
4. Scan the output for identifiers that must not leave the project, and report
   anything found instead of deleting it quietly.
5. Save the file under exports/ with the date in the name, and record the row count.

### Constraints

Read-only database access. No real names, emails, or record identifiers in examples
or documentation. The user approves every new export destination.

### Verification

The row count matches the query's expected grain, and the privacy scan reported
nothing.
```

### What these examples show

Both examples keep the pattern the skills folder guide asks for: a frontmatter block the
skill parser reads, a workflow of small verifiable steps, explicit constraints, and a
closing check. Neither example grants permissions or weakens repository rules, because a
skill cannot do either. When your skill needs more material than fits one file, put it in
a `references/` folder inside the skill and link it from the workflow.

### Conclusion

You can now copy the skill template, adapt one of these examples to your project's real
commands, and register the result in the skills index. Keep each skill small enough to
review in one sitting.

### Additional resources

- [Skills folder guide](../skills/README.md)
- [Copyable skill template](../skills/skill-template.md)
- [Skills index](../SKILLS.md)
- [Project instructions](../AGENTS.md)
- [Agent Skills specification](https://agentskills.io/specification)

[Return to the PeerFoil README](../README.md)
