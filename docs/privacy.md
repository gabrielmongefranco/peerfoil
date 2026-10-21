<!--
This file is part of PeerFoil.
docs/privacy.md
Author(s): Gabriel Mongefranco.
Created: 2026-10-21
Last Modified: 2026-10-21
Summary: Explains what PeerFoil keeps on your computer, what reaches a model provider, and what is never stored.

Copyright © 2026 Gabriel Mongefranco

Permission is granted to copy, distribute and/or modify this document under the terms of
the GNU Free Documentation License, Version 1.3 or any later version published by the Free
Software Foundation; with no Invariant Sections, no Front-Cover Texts, and no Back-Cover
Texts. See <https://www.gnu.org/licenses/fdl-1.3.html>.
-->

# PeerFoil

## Privacy

[Return to the PeerFoil README](../README.md)

This page explains where your information goes when you use PeerFoil. The short version:
accepted project records stay in your repository, model requests go only to the model
tools you already signed in to, and PeerFoil never stores your credentials or full
conversations.

### What stays in your repository

PeerFoil writes accepted project truth under `.peerfoil/` in your repository: the
project record, decisions, architecture, Quality Contract, plans, a small redacted
history, evidence records, reviews, and lessons. You can read all of it without
PeerFoil. These files are versioned with your project, so whoever can read your
repository can read them. They contain:

- your goal and decisions, as you approved them;
- plan, task, and evidence records with Git revisions and hashes;
- which model family produced and reviewed each piece of work; and
- captured patches for produced changes, after a scan for secrets.

### What reaches a model provider

When PeerFoil sends work to a model, it builds a compact packet for that one task: the
relevant repository rules, accepted decisions and architecture excerpts, the task scope,
and the expected answer shape. The packet goes to the tool you signed in to, Claude Code
or the Codex CLI, under that provider's own terms. PeerFoil never sends your full chat
history, other tasks' content, or anything from outside the repository.

A local model called through Ollama runs on your own computer, so those packets reach no
outside service at all.

### What is never stored

- **Credentials and tokens.** PeerFoil uses each tool's existing login and never reads,
  copies, or records it.
- **Raw conversations.** Full model transcripts and provider event streams stay out of
  Git. Records keep short redacted summaries and session identifiers only.
- **Paths that name you.** A program path under your home directory names your user
  account, so records keep versions, never paths.
- **Secrets in output.** Captured patches and retained command output are scanned
  before saving. When something looks like a credential, PeerFoil blocks the capture
  and asks you instead of storing it.

### What you control

- The personal or work profile keeps rule sources separate; neither profile edits your
  repository's `AGENTS.md`.
- Every destructive or external action needs your explicit approval.
- Reviews name the model family that approved each piece of work, so you can
  reconstruct who saw what from the repository alone.

PeerFoil Skills is guided, so these rules are written procedure for the agents, not
mechanical enforcement. The Core releases enforce them in a local application. Treat
your repository's own visibility settings as the real boundary for anything sensitive.

### Conclusion

Keep sensitive material out of the repository and out of task packets, and PeerFoil adds
no new place where it could leak: records stay in Git, model traffic uses your existing
logins, and credentials are never written down.

### Additional resources

- [Known limits](known-limits.md)
- [PeerFoil method](PeerFoil-Method.md), section 13, on files and local state
- [Architecture](architecture.md), section 10, on trust boundaries

[Return to the PeerFoil README](../README.md)
