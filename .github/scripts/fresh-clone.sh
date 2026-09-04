#!/usr/bin/env bash
# Project:  PeerFoil  |  File: .github/scripts/fresh-clone.sh
# Authors:  Gabriel Mongefranco (@gabrielmongefranco)
# Created:  2026-09-04  |  Modified: 2026-09-04
# Summary:  Verifies that a fresh PeerFoil checkout contains its required pre-release contract files.
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
    echo "fresh clone: missing or empty required file: $path" >&2
    exit 1
  fi
done

echo "fresh clone: required PeerFoil contract files present"
