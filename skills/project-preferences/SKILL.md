---
name: project-preferences
description: PeerFoil's file header format, license notices, and repository-specific conventions. Apply when creating or editing any file that carries a header, and when writing Go in this repository.
---

<!--
This file is part of PeerFoil.
skills/project-preferences/SKILL.md
Author(s): Gabriel Mongefranco
Created: 2026-09-09
Last Modified: 2026-09-09
Summary: Repository-specific header format and conventions for PeerFoil.
Copyright © 2026 Gabriel Mongefranco
Licensed under the GNU Free Documentation License v1.3 or later.
See <https://www.gnu.org/licenses/fdl-1.3.html>. See README for full license information.
-->

# PeerFoil

## Project preferences

Use this skill when planning, implementing, or reviewing changes in this repository. It
supplements [AGENTS.md](../../AGENTS.md), especially section 3, and cannot weaken its
security, privacy, accessibility, licensing, testing, or authorization rules.

### File headers and license notices

Every source, script, workflow, configuration, and Markdown file created or materially
changed for PeerFoil must include a header when its format safely permits comments. Keep
existing license notices. Do not replace a complete notice with only an SPDX line.

Each header includes:

- project name;
- repository-relative path;
- author or authors;
- created date;
- last modified date;
- a one-to-three sentence summary;
- notes when useful;
- copyright; and
- the correct license notice or SPDX identifier.

Use `2026-09-04` as the created date for initial repository files. Update `Last Modified`
when a file changes materially. Use ISO dates in new files.

Example for Go:

```go
// This file is part of PeerFoil.
// internal/controller/controller.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-09-04
// Last Modified: 2026-09-04
// Summary: Advances validated PeerFoil workflow transitions.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later
```

Markdown uses an equivalent hidden HTML comment. JSON may use underscore-prefixed metadata
fields only when the consuming schema allows extra fields. If strict JSON cannot contain
metadata, place the notice in a clearly named sibling file and document the choice.

Do not edit verbatim third-party license text just to add a header.

### Specification order

`docs/PeerFoil-Method.md` is the product contract, `docs/architecture.md` defines
components and boundaries, and `docs/implementation-plan.md` defines delivery order when
it is consistent with both. Read all three before changing PeerFoil behavior. When they
disagree, preserve the product contract, explain the conflict, and update every affected
document in the same change.

### Additional resources

- [Project instructions](../../AGENTS.md)
- [Response style skill](../response-style/SKILL.md)
- [Skills index](../../SKILLS.md)
