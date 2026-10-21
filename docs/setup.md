<!--
This file is part of PeerFoil.
docs/setup.md
Author(s): Gabriel Mongefranco.
Created: 2026-10-21
Last Modified: 2026-10-21
Summary: Prerequisites for PeerFoil Skills and the tested way to load the plugin for testing.

Copyright © 2026 Gabriel Mongefranco

Permission is granted to copy, distribute and/or modify this document under the terms of
the GNU Free Documentation License, Version 1.3 or any later version published by the Free
Software Foundation; with no Invariant Sections, no Front-Cover Texts, and no Back-Cover
Texts. See <https://www.gnu.org/licenses/fdl-1.3.html>.
-->

# PeerFoil

## Setup

[Return to the PeerFoil README](../README.md)

This page explains what you need before using PeerFoil Skills and how to load the plugin.
PeerFoil has not been released yet, so the only supported path today is loading the
plugin from a local checkout for testing. Everything here was tested with the exact
versions named below; newer versions usually work but are not claimed.

### Requirements

1. **Git.** Any recent version. Your project must be a Git repository.
2. **Claude Code.** The plugin was tested with version 2.1.260. PeerFoil runs inside it.
3. **Codex CLI, signed in.** Tested with version 0.153.0. Install it on its own or use
   the one bundled inside the Codex IDE extension; PeerFoil finds either. Sign in with
   `codex login`. Without Codex, PeerFoil still works, but every review is labeled
   Reduced assurance because no second model family is available.
4. **Your project's own tools.** Whatever builds and tests your project, such as Go or
   Node.js. PeerFoil runs the checks your project declares; it installs nothing.
5. **Python 3.9 or later**, only if you want to run this repository's own checks. Using
   the plugin does not need Python.

No Node.js, Docker, WSL, database, or paid non-LLM service is required.

### Load the plugin for testing

1. Clone this repository.
2. Start Claude Code in your own project's directory with the plugin loaded:

   ```text
   claude --plugin-dir <path to this repository>/plugins/peerfoil
   ```

3. Run `/peerfoil:setup`. It checks Git, repository instructions, Claude Code, Codex,
   the Codex MCP server, local models, your profile, and the host sandbox, and it tells
   you the one next step for anything that is missing.

Installing from a published marketplace is planned but not yet tested, so no
installation command is published yet.

### Connect Codex

The smoothest path registers the Codex CLI's own MCP server in Claude Code once:

```text
claude mcp add --scope user codex -- codex mcp-server
```

Use the full path to `codex` when it is not on your `PATH`; `/peerfoil:setup` finds it
and offers to run this command for you. Restart Claude Code afterwards. Without the
registration, PeerFoil falls back to running `codex exec`, which works but is less
smooth. PeerFoil never stores your Codex credentials; it uses the login the CLI already
holds.

Reviews and tasks may take up to ten minutes. If you registered the MCP server, set
`MCP_TOOL_TIMEOUT=600000` in Claude Code's environment so the host does not cut a
long review short.

### Optional extras

- **Local models.** If [Ollama](https://ollama.com/) is installed, PeerFoil can use a
  local model for advisory reading and review. See the known limits page for what
  advisory means.
- **Host sandbox.** PeerFoil recommends isolating the agent tools and checks for the
  strongest layer it can see. Best is a container that holds the whole toolchain, such
  as [Agent Airlock](https://github.com/gabrielmongefranco/agent-airlock). On macOS,
  Linux, and WSL2, Claude Code also ships a Bash sandbox that fits reduced-supervision
  runs; its settings fragment lives in the plugin's sandbox reference.

### Start working

Run `/peerfoil:start` with one sentence about what you want to build or change. PeerFoil
records the goal, asks the important questions, and walks the guided workflow from
there. `/peerfoil:status` shows where you are, and `/peerfoil:resume` continues from the
files in your repository, even in a fresh chat.

### Conclusion

You can now load the plugin, connect Codex, and start a guided project. If a check
fails, the [troubleshooting page](troubleshooting.md) covers the known causes.

### Additional resources

- [Troubleshooting](troubleshooting.md)
- [Known limits](known-limits.md)
- [Privacy](privacy.md)
- [Plugin README](../plugins/peerfoil/README.md)
- [Codex CLI](https://github.com/openai/codex)
- [Ollama](https://ollama.com/)

[Return to the PeerFoil README](../README.md)
