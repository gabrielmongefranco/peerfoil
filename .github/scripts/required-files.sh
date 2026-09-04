#!/usr/bin/env bash
# Project:  PeerFoil  |  File: .github/scripts/required-files.sh
# Authors:  Gabriel Mongefranco (@gabrielmongefranco)
# Created:  2026-09-04  |  Modified: 2026-09-04
# Summary:  Confirms that a PeerFoil checkout includes the files needed to understand the project.
# SPDX-License-Identifier: GPL-3.0-or-later

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

required=(
  README.md
  AGENTS.md
  LICENSE
  NOTICE
  .zenodo.json
  docs/PeerFoil-Method.md
  docs/architecture.md
  docs/implementation-plan.md
)

for path in "${required[@]}"; do
  if [[ ! -s "$path" ]]; then
    echo "required files: missing or empty file: $path" >&2
    exit 1
  fi
done

echo "required files: all PeerFoil project files are present"
