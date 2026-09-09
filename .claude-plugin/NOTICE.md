<!--
This file is part of PeerFoil.
.claude-plugin/NOTICE.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Carries the file notice for the sibling marketplace.json, which the Claude Code strict validator does not allow to hold metadata fields.
Notes: See docs/decision-log.md, entry D-0001.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# Notice for `.claude-plugin/marketplace.json`

The file `marketplace.json` in this directory is part of PeerFoil. It declares the
PeerFoil marketplace for Claude Code. Author: Gabriel Mongefranco. Created 2026-09-05.

Copyright © 2026 Gabriel Mongefranco. Licensed under the GNU General Public License,
version 3 or later (`GPL-3.0-or-later`).

The notice lives here because Claude Code's strict manifest validator treats any field it
does not recognize inside `marketplace.json` as an error, so the manifest cannot carry the
usual PeerFoil header fields.
