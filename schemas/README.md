<!--
This file is part of PeerFoil.
schemas/README.md
Author(s): Gabriel Mongefranco.
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
| `plan.schema.json` | `.peerfoil/plan.json` |
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

Each record carries `schema_version`. The current version of every record is `1`. A
change that breaks existing files increases the version, keeps the old schema readable,
and documents an upgrade path.

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
