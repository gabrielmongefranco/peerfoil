<!--
This file is part of PeerFoil.
plugins/peerfoil/references/lineage.md
Author(s): Gabriel Mongefranco.
Created: 2026-09-05
Last Modified: 2026-09-05
Summary: Defines how PeerFoil records model families and decides whether a reviewer is independent.
Notes: The product contract for independence is docs/PeerFoil-Method.md, section 7.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# Model lineage and independence reference

PeerFoil prefers a reviewer from a different model family than the author. To apply that
rule, every important artifact records who made it and which family the model belongs to.
This file defines the values and the decision rule.

## 1. Lineage root

A **lineage root** names the underlying base model family. Aliases, fine-tunes,
checkpoints, quantized copies, and renamed endpoints of the same base model share one
lineage root. A different endpoint name does not make a model independent.

| `lineage_root` | Use for |
|---|---|
| `anthropic-claude` | Any Claude model reached through Claude Code or another Claude tool |
| `openai-gpt` | Any GPT or Codex model reached through the Codex CLI, its MCP server, or `codex exec` |
| `human` | A person, including the user |
| `unknown` | A model whose base family cannot be established |

Another family may be recorded as a lowercase slug such as `meta-llama` once a later
release qualifies it. Do not guess a family. Record `unknown` when the base model is not
documented.

The family comes from the model identifier, never from the application that ran it,
because Codex CLI and Claude Code can be pointed at other providers. Map identifiers
this way: `claude-*` is `anthropic-claude`; `gpt-*`, `codex-*`, and `o1`, `o3`, or `o4`
identifiers are `openai-gpt`; anything else is `unknown`. The identifier that counts is
the seat's configured model, which setup records from the tool's own configuration; a
seat left at `default` has `unknown` lineage. A reviewer's self-reported identifier is a
claim used only as a cross-check: when it disagrees with the seat model, record
`unknown`.

## 2. Tools

The `tool` field records the application that ran the model:

| `tool` | Meaning |
|---|---|
| `claude-code` | Claude Code, including this plugin's own session |
| `codex-cli` | Codex CLI, including sessions started through its MCP server or `codex exec` |
| `human` | A person acting directly |
| `other` | Any other application; describe it in the record's notes |

## 3. Actor record

Every authored artifact, change set, evidence record, review, and lesson records an
actor:

```json
{
  "role": "evaluator",
  "tool": "claude-code",
  "model": "default",
  "effort": "medium",
  "lineage_root": "anthropic-claude",
  "session": null
}
```

- `role` is one of `evaluator`, `architect`, `planner`, `change_steward`, `producer`,
  `reviewer`, `repair_producer`, `coordinator`, or `user`. The `coordinator` is the Claude
  Code session that runs the PeerFoil skills and writes the project files.
- `model` is the model identifier when known, or `default` when the tool's default model
  was used and its exact name is not available.
- `effort` is `low`, `medium`, `high`, `xhigh`, or `null` when the tool did not expose it.
- `session` is the provider's own session identifier when one exists, or `null`. It is
  never a token or credential.

## 4. Independence rule

For each material artifact, patch, or repair:

1. The exact session that authored it may never approve it.
2. Normal **primary approval** comes from a fresh session whose `lineage_root` differs
   from the author's `lineage_root`.
3. A review from the same `lineage_root` is a **secondary** check. It can add findings.
   It cannot satisfy normal approval by itself.
4. If the author's or reviewer's `lineage_root` is `unknown`, the review's independence
   is `reduced`. Tell the user and let them accept the limitation or wait for an eligible
   reviewer.
5. A phase that mixes authors is judged item by item. Do not label a whole phase
   independent unless every material item has an eligible primary reviewer.

Record the outcome in each review as `independence: independent`, `secondary`, or
`reduced`.

When no eligible different-family reviewer is available, PeerFoil does not guess. It
stops and lets the user either wait for one or accept **Reduced assurance** for that
artifact. An accepted reduced-assurance review by a fresh same-family session is recorded
with `independence: secondary`, the user's acceptance and its time in the review record
and in `history.jsonl`, and the words "Reduced assurance" wherever the artifact's status
is shown. The acceptance covers one artifact draft; a later artifact needs its own.

## 5. Default arrangement

The default hosted arrangement keeps the families separated by role:

| Role | Family |
|---|---|
| Evaluator, architect, planner, change steward | `anthropic-claude` |
| Software producer and repair producer | `openai-gpt` |
| Architecture and plan review | `openai-gpt` |
| Code review | `anthropic-claude` |
| Phase review | One reviewer from each family |

Advanced settings may change the tools and models. They may not remove the rule that
primary approval comes from a different family.
