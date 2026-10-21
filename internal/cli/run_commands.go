// This file is part of PeerFoil.
// internal/cli/run_commands.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: The start and resume commands, which drive the controller engine.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

func runStartReal(ctx context.Context, env *Env, args []string) error {
	if len(args) != 0 {
		return errs.New(errs.Usage, "start takes no arguments in this build",
			"describe changes with the PeerFoil skills; peerfoil start runs the next planned task")
	}
	engine, err := buildEngine(ctx, env)
	if err != nil {
		return err
	}
	return engine.ProduceNext(ctx)
}

func runResumeReal(ctx context.Context, env *Env, args []string) error {
	confirmStopped := false
	for _, arg := range args {
		switch arg {
		case "--confirm-stopped":
			confirmStopped = true
		default:
			return errs.New(errs.Usage, "resume accepts only --confirm-stopped", "run peerfoil resume --help")
		}
	}
	engine, err := buildEngine(ctx, env)
	if err != nil {
		return err
	}
	return engine.Resume(ctx, confirmStopped)
}
