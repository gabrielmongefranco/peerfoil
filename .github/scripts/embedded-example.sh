#!/usr/bin/env bash
# Project:  PeerFoil  |  File: .github/scripts/embedded-example.sh
# Authors:  Gabriel Mongefranco (@gabrielmongefranco)
# Created:  2026-09-04  |  Modified: 2026-09-04
# Summary:  Guards the pre-release contract's coding-first, artifact-neutral project-pack boundary.
# SPDX-License-Identifier: GPL-3.0-or-later

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

method="docs/PeerFoil-Method.md"
architecture="docs/architecture.md"

for term in "Software Pack" "Documentation" "Business Plan" "Research Report"; do
  if ! grep -Fq "$term" "$method"; then
    echo "artifact-neutral contract: method is missing $term" >&2
    exit 1
  fi
done

if ! grep -Fq "coding-first, artifact-neutral" "$architecture"; then
  echo "artifact-neutral contract: architecture boundary is missing" >&2
  exit 1
fi

echo "artifact-neutral contract: software default and non-code packs retained"
