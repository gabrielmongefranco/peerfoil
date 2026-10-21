// This file is part of PeerFoil.
// internal/cli/commands.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: The five peerfoil commands: their help text and their connection to the
// controller, project, and doctor packages.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
)

const initHelp = `Usage: peerfoil init

Shows the project records it would create under .peerfoil/ in this repository,
then creates them when you pass --yes. It never overwrites an existing
.peerfoil/ directory.

Options:
  --yes   Create the files without showing the preview again.`

const doctorHelp = `Usage: peerfoil doctor

Checks that this computer and repository are ready: the operating system, Git
and the repository state, the project records and their schema versions, Claude
Code and the Codex CLI with their logins, the selected models, the project
pack's tools, the host sandbox, and the local state directory. Reports each
check with one next step and changes nothing.`

const startHelp = `Usage: peerfoil start

Runs the next step of the planned software phase: selects the next eligible
task, runs the producer in a dedicated Git worktree, runs the declared checks,
and integrates accepted work into the PeerFoil integration branch. Stops when a
decision needs you.`

const statusHelp = `Usage: peerfoil status

Shows the assurance level, the current phase, stage, and task, the quality
state, any blocker, the decision waiting for you, and the next action. Reads
files only; it never calls a model and works offline.`

const resumeHelp = `Usage: peerfoil resume

Continues safely from the last recorded boundary after an interruption. When
the record shows an outside operation whose result cannot be proven, resume
stops and asks instead of repeating it.`

func runStart(ctx context.Context, env *Env, args []string) error {
	return runStartReal(ctx, env, args)
}

func runResume(ctx context.Context, env *Env, args []string) error {
	return runResumeReal(ctx, env, args)
}
