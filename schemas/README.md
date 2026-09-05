<!--
This file is part of PeerFoil.
schemas/README.md
Author(s): Gabriel Mongefranco; OpenAI Codex.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Explains the JSON schemas that validate PeerFoil's machine-readable project records and pack manifests.
Notes: The human-readable definitions live in plugins/peerfoil/references/records.md.

Copyright © 2026 Gabriel Mongefranco

Permission is granted to copy, distribute and/or modify this document under the terms of
the GNU Free Documentation License, Version 1.3 or any later version published by the Free
Software Foundation; with no Invariant Sections, no Front-Cover Texts, and no Back-Cover
Texts. See <https://www.gnu.org/licenses/fdl-1.3.html>.
-->

# PeerFoil Schemas

## The validated shapes behind PeerFoil's JSON records

[Return to the PeerFoil README](../README.md)

This directory holds the JSON Schema files for the PeerFoil records that need validation.
PeerFoil Skills 0.1 uses them in repository checks. PeerFoil Core will read the same
schemas to validate a project before it acts on it. The readable definitions, identifier
rules, and field meanings are in the
[records reference](../plugins/peerfoil/references/records.md).

## Files

| Schema | Validates |
|---|---|
| `common.schema.json` | Shared definitions: identifiers, timestamps, actors, model seats, and evidence requirements |
| `project.schema.json` | `.peerfoil/project.json` |
| `plan.schema.json` | Version 2 `.peerfoil/plan.json` and retained version 2 snapshots |
| `plan-v1.schema.json` | Existing version 1 plans and their retained snapshots |
| `transition.schema.json` | One line of `.peerfoil/history.jsonl` |
| `pack.schema.json` | A project pack's `pack.json` manifest |

Every schema carries its file notice in the standard `$comment` keyword, because JSON
cannot hold comments.

## Supported JSON Schema subset

The repository check validates instances with a small validator that uses only the Python
standard library. To keep that validator honest, these schemas use only the following
keywords:

`$schema`, `$id`, `$comment`, `$defs`, `$ref` (to `#/$defs/...` in the same file or in a
sibling file), `title`, `type` (a string or a list of strings), `const`, `enum`,
`pattern`, `minLength`, `maxLength`, `minimum`, `maximum`, `properties`, `required`,
`additionalProperties` (boolean), `items`, `minItems`, and `maxItems`.

A change that needs another keyword must also extend the validator in
`tests/static_checks.py` and this list.

## Versioning

Each record carries `schema_version`. Project, transition, and pack records use version `1`; new plans use version `2`. A
change that breaks existing files increases the version, keeps the old schema readable,
and documents an upgrade path.

Plan version 2 adds `changes` entries for revision traceability. The unchanged version 1
shape is retained in `plan-v1.schema.json`, and the repository validator dispatches by
`schema_version`. To upgrade, preserve the accepted version 1 plan in `.peerfoil/plans/`,
create the next candidate with `schema_version: 2` and `changes: []`, and follow normal
change review. Never rewrite historical snapshots. Older consumers must be upgraded
before reading version 2. A recovery can restore the prior accepted snapshot and report
later work as pending; it must not silently accept later work under the older plan.
Cross-record references, disjoint task sets, review eligibility, and file hashes are
semantic host checks; schema validity alone does not establish acceptance.

Transition records may reference phase review records through the optional
`refs.phase_reviews` key, and `common.schema.json` defines the `pr-` identifier. Both
additions are optional, so existing version 1 history lines stay valid and the version
does not change. The review settings in `project.json` also pass an ordering check:
the default pass counts never exceed their maximums, the two phase reviewers use
different tools, and the repair producer never runs at low effort.

## Conclusion

The schemas are small on purpose. They protect the identifiers, states, and settings that
later releases enforce without making the readable project files harder to write by hand.

## Additional Resources

- [Records reference](../plugins/peerfoil/references/records.md)
- [PeerFoil architecture](../docs/architecture.md)
- [JSON Schema specification](https://json-schema.org/specification)

[Return to the PeerFoil README](../README.md)

---

Copyright © 2026 Gabriel Mongefranco
