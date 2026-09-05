#!/usr/bin/env bash
# Project:  PeerFoil  |  File: .github/scripts/conformance.sh
# Authors:  Gabriel Mongefranco (@gabrielmongefranco)
# Created:  2026-09-04  |  Modified: 2026-09-05
# Summary:  Checks PeerFoil's required files, project identity, headers, and license notices.
# SPDX-License-Identifier: GPL-3.0-or-later

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

bash .github/scripts/required-files.sh
bash .github/scripts/project-packs.sh

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
  docs/phase-prompt-template.md
  docs/plans/phase-1-skills.md
  docs/plans/phase-2-core-alpha.md
  .github/FUNDING.yml
  .github/copilot-instructions.md
  .github/scripts/conformance.sh
  .github/scripts/project-packs.sh
  .github/scripts/required-files.sh
  .github/workflows/ci.yml
)

for path in "${metadata_files[@]}"; do
  head -n 20 "$path" | grep -Fq "PeerFoil" || {
    echo "conformance: missing PeerFoil header in $path" >&2
    exit 1
  }
  head -n 20 "$path" | grep -Eq '(Created:[[:space:]]+|"_created":[[:space:]]*")20[0-9]{2}-[0-9]{2}-[0-9]{2}' || {
    echo "conformance: missing created date in $path" >&2
    exit 1
  }
  head -n 20 "$path" | grep -Eq '(Last Modified|Modified):[[:space:]]*20[0-9]{2}-[0-9]{2}-[0-9]{2}|"_modified":[[:space:]]*"20[0-9]{2}-[0-9]{2}-[0-9]{2}' || {
    echo "conformance: missing modified date in $path" >&2
    exit 1
  }
done

node -e 'JSON.parse(require("fs").readFileSync(".zenodo.json", "utf8"))'

for path in README.md AGENTS.md NOTICE .zenodo.json docs/*.md docs/plans/*.md .github/* .github/scripts/* .github/workflows/*; do
  [[ -d "$path" ]] && continue
  if ! grep -Fq "Copyright © 2026 Gabriel Mongefranco" "$path" \
    && ! grep -Fq "SPDX-License-Identifier:" "$path" \
    && [[ "$path" != ".github/FUNDING.yml" ]]; then
    echo "conformance: missing copyright or SPDX notice in $path" >&2
    exit 1
  fi
done

echo "conformance: PeerFoil repository checks pass"
