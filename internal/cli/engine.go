// This file is part of PeerFoil.
// internal/cli/engine.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Builds the controller engine for the start and resume commands: loads and
// validates state, opens the journal, loads the pack, and wires the provider adapters.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/controller"
	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/pack"
	"github.com/gabrielmongefranco/peerfoil/internal/project"
	"github.com/gabrielmongefranco/peerfoil/internal/provider"
	"github.com/gabrielmongefranco/peerfoil/internal/schema"
	"github.com/gabrielmongefranco/peerfoil/internal/store"
	"github.com/gabrielmongefranco/peerfoil/internal/workspace"
)

// evidenceRunner adapts the shared process runner to the evidence package's contract,
// so evidence commands reuse the same bounded, shell-free execution as model calls.
func evidenceRunner(ctx context.Context, argv []string, dir string, env []string,
	timeout time.Duration) ([]byte, []byte, int, bool, error) {
	result, err := provider.Run(ctx, provider.Request{
		Argv: argv, Dir: dir, Env: env, Timeout: timeout,
	})
	if err != nil {
		var categorized *errs.Error
		if errors.As(err, &categorized) && categorized.Category == errs.Timeout {
			return result.Stdout, result.Stderr, result.ExitCode, true, nil
		}
		return result.Stdout, result.Stderr, result.ExitCode, result.TimedOut, err
	}
	return result.Stdout, result.Stderr, result.ExitCode, result.TimedOut, nil
}

// findAdapter probes the candidate paths and returns the first installed one.
func findAdapter(ctx context.Context, tool string, seat provider.Seat) (provider.Adapter, error) {
	var candidates []string
	switch tool {
	case "codex-cli":
		candidates = provider.FindCodex()
	case "claude-code":
		candidates = provider.FindClaude()
	default:
		return nil, errs.New(errs.Configuration,
			"the seat tool "+tool+" is not supported by Core Alpha",
			"use claude-code or codex-cli seats in the project settings")
	}
	for _, candidate := range candidates {
		var adapter provider.Adapter
		if tool == "codex-cli" {
			adapter = provider.NewCodex(candidate, seat, nil)
		} else {
			adapter = provider.NewClaude(candidate, seat, nil)
		}
		if adapter.Probe(ctx).Found {
			return adapter, nil
		}
	}
	name := map[string]string{"codex-cli": "the Codex CLI", "claude-code": "Claude Code"}[tool]
	return nil, errs.New(errs.Configuration, name+" was not found",
		"install it, or run peerfoil doctor to see what was checked")
}

// buildEngine loads everything a controller run needs and fails closed on anything it
// cannot prove: records, journal, pack, and the producer seat.
func buildEngine(ctx context.Context, env *Env) (*controller.Engine, error) {
	schemas, err := schema.LoadEmbedded()
	if err != nil {
		return nil, errs.Wrap(errs.Internal, "loading the record schemas", err)
	}
	root, err := workspace.Discover(ctx, env.Dir)
	if err != nil {
		return nil, err
	}
	state, err := project.Load(root, schemas)
	if err != nil {
		return nil, err
	}
	journal, err := store.Open(state.Project.ProjectID, root)
	if err != nil {
		return nil, err
	}
	loadedPack, err := pack.Load(state.Project.Pack.ID, schemas)
	if err != nil {
		return nil, err
	}

	producerSeat := state.Project.Settings.Roles["producer"]
	if producerSeat.Tool != "codex-cli" {
		return nil, errs.New(errs.Configuration,
			"Core Alpha supports codex-cli production only, and this project's producer seat uses "+producerSeat.Tool,
			"set the producer seat back to codex-cli, or produce with the PeerFoil skills")
	}
	seat := provider.Seat{Tool: producerSeat.Tool, Model: producerSeat.Model, Effort: producerSeat.Effort}
	producer, err := findAdapter(ctx, producerSeat.Tool, seat)
	if err != nil {
		return nil, err
	}

	var reviewers []provider.Adapter
	var reviewerSeats []provider.Seat
	for _, reviewerSeat := range state.Project.Settings.PhaseReviewers {
		seat := provider.Seat{Tool: reviewerSeat.Tool, Model: reviewerSeat.Model, Effort: reviewerSeat.Effort}
		adapter, err := findAdapter(ctx, reviewerSeat.Tool, seat)
		if err != nil {
			// A missing reviewer blocks review, not production; the review step
			// reports it precisely when it needs the seat.
			continue
		}
		reviewers = append(reviewers, adapter)
		reviewerSeats = append(reviewerSeats, seat)
	}

	worktreeParent := filepath.Join(journal.Dir, "worktrees")
	if err := os.MkdirAll(worktreeParent, 0o700); err != nil {
		return nil, errs.Wrap(errs.Configuration, "creating the worktree directory", err)
	}
	return controller.NewEngine(controller.Deps{
		State:          state,
		Repo:           workspace.New(root),
		Journal:        journal,
		Pack:           loadedPack,
		Producer:       producer,
		ProducerSeat:   seat,
		Reviewers:      reviewers,
		ReviewerSeats:  reviewerSeats,
		Evidence:       evidenceRunner,
		WorktreeParent: worktreeParent,
		Out:            env.Stdout,
	}), nil
}
