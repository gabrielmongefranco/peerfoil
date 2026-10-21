<!--
This file is part of PeerFoil.
docs/troubleshooting.md
Author(s): Gabriel Mongefranco.
Created: 2026-10-21
Last Modified: 2026-10-21
Summary: Known PeerFoil failure modes with their symptom, cause, and fix. Every entry comes from a problem actually hit during the Phase 1 builds.

Copyright © 2026 Gabriel Mongefranco

Permission is granted to copy, distribute and/or modify this document under the terms of
the GNU Free Documentation License, Version 1.3 or any later version published by the Free
Software Foundation; with no Invariant Sections, no Front-Cover Texts, and no Back-Cover
Texts. See <https://www.gnu.org/licenses/fdl-1.3.html>.
-->

# PeerFoil

## Troubleshooting

[Return to the PeerFoil README](../README.md)

This page lists the failure modes seen while building and testing PeerFoil Skills, with
what you see, why it happens, and what to do. If your problem is not here, run
`/peerfoil:setup` first; it reports most missing prerequisites with one next step.

### Codex reviews fail with a usage-limit message

- **Symptom:** A review or production step stops and the record shows a blocked change
  set or a paused run that names the Codex account limit.
- **Cause:** The ChatGPT account behind the Codex CLI reached its usage limit. The
  Codex CLI reports the time when it can retry.
- **Fix:** Wait until the limit resets or add credits, then run `/peerfoil:resume`.
  PeerFoil records the pause honestly and does not substitute another writer, so
  nothing is lost.

### Codex works in the terminal but PeerFoil uses the slower fallback

- **Symptom:** Setup reports the Codex MCP server as Not verified and PeerFoil says it
  is using the `codex exec` fallback.
- **Cause:** The Codex CLI's MCP server is not registered in Claude Code.
- **Fix:** Run `claude mcp add --scope user codex -- <path to codex> mcp-server`, then
  restart Claude Code. `/peerfoil:setup` shows the exact command with the found path
  and offers to run it.

### A long review is cut short

- **Symptom:** A Codex review through the MCP server stops early with no result, well
  before PeerFoil's ten-minute limit.
- **Cause:** Claude Code bounds every MCP call with `MCP_TOOL_TIMEOUT`, and the default
  is shorter than a slow review.
- **Fix:** Set `MCP_TOOL_TIMEOUT=600000` for Claude Code, which matches PeerFoil's
  ten-minute limit.

### `claude` or `codex` is not found

- **Symptom:** Setup reports Claude Code or the Codex CLI as missing although an IDE
  extension runs it fine.
- **Cause:** The IDE extensions bundle their own binaries, which are not on the `PATH`.
- **Fix:** Nothing to install. Setup searches the extension folders and uses what it
  finds. If you want the programs on your `PATH`, follow each tool's own installation
  documentation.

### Non-ASCII output disappears on Windows

- **Symptom:** A script that prints results with non-ASCII characters fails or prints
  nothing at the end, although its work succeeded.
- **Cause:** The default Windows console encoding cannot print some Unicode output.
- **Fix:** Run the script with the environment variable `PYTHONIOENCODING=utf-8`, or
  keep summaries ASCII-safe. The result that was saved to a file is unaffected.

### A project path with spaces or special characters fails

- **Symptom:** A fresh Codex session cannot read files in a repository whose path mixes
  spaces, Unicode, and an apostrophe, while Claude Code reads them fine.
- **Cause:** Unresolved provider-side path handling, seen during Stage 3 verification.
  The exact cause has not been isolated.
- **Fix:** Move the repository to a path without the special characters, or run the
  production step again after the provider updates. PeerFoil stops honestly instead of
  reporting false success. This stays a known open check in the Stage 5 platform matrix.

### A file passes checks locally but is missing from a fresh clone

- **Symptom:** A required file exists in your working tree, every check passes, and a
  fresh clone does not contain it.
- **Cause:** A case-insensitive `.gitignore` pattern matched the file on Windows, so it
  was never committed. Stage 3 hit exactly this.
- **Fix:** Check `git check-ignore -v <file>` and narrow the ignore pattern. When
  validating anything distributable, run the checks from a clean checkout, not only the
  working tree.

### The plugin validator rejects a skill or agent file

- **Symptom:** `claude plugin validate --strict` fails on frontmatter that looks fine.
- **Cause:** An unquoted frontmatter value containing `: ` or starting with a YAML
  special character parses as a mapping, and CRLF checkouts make it worse.
- **Fix:** Quote the value. The repository's static checks catch this before the
  validator does; run `python tests/static_checks.py`.

### Conclusion

Most problems resolve with `/peerfoil:setup`, the MCP registration command, or the
timeout variable above. Anything else is worth a GitHub issue with the exact symptom.

### Additional resources

- [Setup](setup.md)
- [Known limits](known-limits.md)
- [Plugin README](../plugins/peerfoil/README.md)

[Return to the PeerFoil README](../README.md)
