#!/usr/bin/env bash
# Project:  PeerFoil  |  File: .github/scripts/project-packs.sh
# Authors:  Gabriel Mongefranco (@gabrielmongefranco)
# Created:  2026-09-04  |  Modified: 2026-09-04
# Summary:  Keeps software as PeerFoil's main use while protecting support for other project types.
# SPDX-License-Identifier: GPL-3.0-or-later

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

method="docs/PeerFoil-Method.md"
architecture="docs/architecture.md"

for term in "Software remains the first" "Documentation Pack" "Business Plan" "Research Report"; do
  if ! grep -Fq "$term" "$method"; then
    echo "project packs: method is missing $term" >&2
    exit 1
  fi
done

if ! grep -Fq "Software is the first and most complete use" "$architecture"; then
  echo "project packs: architecture does not identify software as the main use" >&2
  exit 1
fi

echo "project packs: software and planned non-coding uses are documented"
