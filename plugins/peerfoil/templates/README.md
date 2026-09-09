<!--
This file is part of PeerFoil.
plugins/peerfoil/templates/README.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Explains the .peerfoil/ templates and carries the license notice for every file in this directory.
Notes: The template files deliberately contain no PeerFoil header, because a copied template becomes the user's own project record.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# PeerFoil project-file templates

The files in this directory are the starting points for the accepted project records
that PeerFoil keeps under `.peerfoil/` in a user's repository. A skill copies the matching
template, replaces every `{{placeholder}}`, and writes the result into the user's project.

## Notice for this directory

Every file in this directory is part of PeerFoil and is licensed under the GNU General
Public License, version 3 or later (`GPL-3.0-or-later`). Copyright © 2026 Gabriel
Mongefranco. The templates carry no header of their own for one reason: once a template is
copied into a project, the resulting file is the user's project record, and a PeerFoil
copyright line inside it would be wrong. This README is the notice for the whole
directory.

The generated project files belong to the user. PeerFoil claims no rights in them.

## Files

| Template | Becomes | Format |
|---|---|---|
| `project.json` | `.peerfoil/project.json` | JSON, validated by `schemas/project.schema.json` |
| `decisions.md` | `.peerfoil/decisions.md` | Markdown |
| `architecture.md` | `.peerfoil/architecture.md` | Markdown |
| `quality.md` | `.peerfoil/quality.md` | Markdown |
| `plan.md` | `.peerfoil/plan.md` | Markdown, regenerated from `plan.json` |
| `plan.json` | `.peerfoil/plan.json` | JSON, validated by `schemas/plan.schema.json` |
| `history.jsonl` | `.peerfoil/history.jsonl` | One JSON object per line, validated by `schemas/transition.schema.json` |
| `evidence.md` | `.peerfoil/evidence/ev-NNNN.md` | Markdown |
| `change-set.md` | `.peerfoil/evidence/cs-NNNN.md` | Markdown |
| `review.md` | `.peerfoil/reviews/rv-NNNN.md` | Markdown |
| `phase-review.md` | `.peerfoil/reviews/pr-NNNN.md` | Markdown |
| `lesson.md` | `.peerfoil/lessons/ls-NNNN.md` | Markdown |

## Placeholders

- `{{project_id}}`, `{{project_name}}`: from `project.json`.
- `{{release}}`: the PeerFoil release that generated the file, such as
  `peerfoil-skills/0.1.0-dev`.
- `{{timestamp}}`: UTC time in ISO 8601 form, such as `2026-09-05T14:30:00Z`.
- Any other `{{name}}` is described by the field it sits in. Replace every placeholder.
  Use `null`, `—`, or `not-applicable` when a value does not apply; never leave a
  placeholder in a generated file.

The JSON templates hold realistic example values instead of placeholders so that they
validate against their schemas. Replace every example value when generating a project.

## Rules

- Keep field names exactly as written. The
  [records reference](../references/records.md) and the schemas depend on them.
- Do not remove fields. Do not add fields that the records reference does not define.
- Start every generated Markdown file with the one-line comment that names the generating
  release, as the templates show.
