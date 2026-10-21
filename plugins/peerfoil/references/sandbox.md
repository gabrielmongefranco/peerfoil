<!--
This file is part of PeerFoil.
plugins/peerfoil/references/sandbox.md
Author(s): Gabriel Mongefranco.
Created: 2026-10-21
Last Modified: 2026-10-21
Summary: Recommends the Claude Code Bash sandbox for low-supervision PeerFoil runs, gives a starter settings fragment, and defines the preflight check that reports sandbox status without blocking.
Notes: PeerFoil detects and reports the host sandbox. It does not implement, configure, or repair operating-system sandboxing. The decision is D-0029 in docs/decision-log.md.

Copyright © 2026 Gabriel Mongefranco
SPDX-License-Identifier: GPL-3.0-or-later
-->

# Host sandbox reference

PeerFoil runs agents with reduced supervision by design, which is the situation an
operating-system sandbox exists for. PeerFoil does not provide a sandbox and version 1.0
will not add one. The host tool PeerFoil already depends on, Claude Code, ships its own
Bash sandbox: Seatbelt on macOS, and bubblewrap on Linux and WSL2, with filesystem write
restrictions and a hostname-allowlist network proxy. PeerFoil recommends that sandbox,
ships a starter settings fragment, and reports plainly when it is absent. It never blocks
on a missing sandbox, and it never claims the sandbox as a boundary PeerFoil provides.

## 1. Recommended settings fragment

A user who wants the sandbox adds a fragment like this to the project's
`.claude/settings.json` or their user settings. Show it when asked; never write it for
the user, because PeerFoil does not edit the host's configuration on its own.

```json
{
  "sandbox": {
    "enabled": true,
    "autoAllowBashIfSandboxed": true,
    "network": {
      "allowedDomains": ["github.com", "proxy.golang.org", "registry.npmjs.org", "pypi.org"]
    }
  },
  "permissions": {
    "deny": [
      "Read(~/.ssh/**)",
      "Read(~/.aws/**)",
      "Read(./.env)",
      "Read(./.env.*)"
    ]
  }
}
```

The allowed-domain list is a starting point; the user trims or extends it for the
project's own package sources. The deny entries keep common credential locations out of
reach even when the sandbox is off. With the sandbox on, this fragment reduces permission
prompts during guided production rather than adding friction.

## 2. Preflight check

Before guided production begins, and in `/peerfoil:setup`, check the sandbox status and
report it. The check informs; it never blocks.

1. Read the merged Claude Code settings the host exposes, or the project's
   `.claude/settings.json` and the user settings when the merged view is not readable.
   `sandbox.enabled: true` in the effective settings counts as configured. The `/sandbox`
   panel in an interactive session shows the live status when the user wants to confirm
   it.
2. Report one of three results:
   - **Configured.** Say that the host sandbox is enabled in settings and note that
     the network allowlist on WSL2 is best effort (section 3).
   - **Not configured.** Say plainly that production will run without an
     operating-system sandbox, show where section 1's fragment goes, and ask the user to
     confirm before proceeding. Record the confirmation in the history line for the
     transition that starts production.
   - **Not supported here.** On native Windows, say that the Claude Code sandbox needs
     WSL2 and that native Windows runs without it. Ask the same confirmation.
3. Record the result as the `host-sandbox` entry of `environment.checks` in
   `.peerfoil/project.json` when a project record exists, with a detail such as
   "enabled in settings", "not configured; user accepted", or "native Windows; not
   supported". Never record file paths from the user's home directory.

Ask the confirmation once per session, not once per task. PeerFoil Skills checks a
settings file, not the running process, so the result is a report, not proof of
enforcement. Core performs the same check programmatically in `peerfoil doctor` and
records the answer the way other preflight facts are recorded.

## 3. Known limits to state honestly

- The Claude Code sandbox does not support native Windows. On Windows it requires
  running inside WSL2.
- On WSL2 the filesystem isolation is reliable, but the network proxy can fail to start
  or fail to enforce `allowedDomains` without the failure being caught. Describe the
  WSL2 network allowlist as best effort.
- The network proxy filters by hostname and does not inspect TLS traffic.
- The sandbox is a host feature PeerFoil detects and reports, not a boundary PeerFoil
  provides or enforces. Machine-level hardening, such as WSL configuration, firewalls,
  or disposable environments, stays outside PeerFoil.
