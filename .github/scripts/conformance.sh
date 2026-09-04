#!/usr/bin/env bash
# Project:  PeerFoil  |  File: .github/scripts/conformance.sh
# Authors:  Gabriel Mongefranco (@gabrielmongefranco)
# Created:  2026-09-04  |  Modified: 2026-09-04
# Summary:  Runs the repository's pre-implementation documentation and metadata conformance checks.
# SPDX-License-Identifier: GPL-3.0-or-later

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

bash .github/scripts/fresh-clone.sh
bash .github/scripts/embedded-example.sh

if grep -RInE '(^# Crosscut|Project:  (Crosscut|Privatium)|github\.com/gabrielmongefranco/crosscut)' \
  README.md AGENTS.md NOTICE .zenodo.json docs .github; then
  echo "conformance: stale project identity found" >&2
  exit 1
fi

metadata_files=(
  README.md
  AGENTS.md
  NOTICE
  .zenodo.json
  .gitignore
  docs/PeerFoil-Method.md
  docs/architecture.md
  docs/implementation-plan.md
  .github/FUNDING.yml
  .github/copilot-instructions.md
  .github/scripts/conformance.sh
  .github/scripts/embedded-example.sh
  .github/scripts/fresh-clone.sh
  .github/workflows/ci.yml
)

for path in "${metadata_files[@]}"; do
  head -n 20 "$path" | grep -Fq "PeerFoil" || {
    echo "conformance: missing PeerFoil header in $path" >&2
    exit 1
  }
  head -n 20 "$path" | grep -Fq "2026-09-04" || {
    echo "conformance: missing initial date in $path" >&2
    exit 1
  }
done

node -e 'JSON.parse(require("fs").readFileSync(".zenodo.json", "utf8"))'

for path in README.md AGENTS.md NOTICE .zenodo.json docs/*.md .github/* .github/scripts/* .github/workflows/*; do
  [[ -d "$path" ]] && continue
  if ! grep -Fq "Copyright © 2026 Gabriel Mongefranco" "$path" \
    && ! grep -Fq "SPDX-License-Identifier:" "$path" \
    && [[ "$path" != ".github/FUNDING.yml" ]]; then
    echo "conformance: missing copyright or SPDX notice in $path" >&2
    exit 1
  fi
done

echo "conformance: PeerFoil metadata and pre-release contracts pass"
