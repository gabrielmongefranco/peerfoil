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
	"fmt"
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
	// The evidence contract says a nil environment means an empty one; the process
	// runner would substitute its base allowlist for nil, so the nil is made
	// explicit here rather than silently widened.
	if env == nil {
		env = []string{}
	}
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
	// The project records name the pack version whose acceptance contract was agreed;
	// running a different embedded version would silently change the evidence commands
	// and levels, so a mismatch fails closed instead of substituting.
	if loadedPack.Version != state.Project.Pack.Version {
		return nil, errs.New(errs.Configuration,
			fmt.Sprintf("this project records pack %s version %s, but this peerfoil build ships version %s",
				state.Project.Pack.ID, state.Project.Pack.Version, loadedPack.Version),
			"install the peerfoil release that ships the recorded pack version, or update the project's pack with the PeerFoil skills")
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

	// The repair seat may differ from the producer seat in model and effort, so it
	// gets its own adapter; recording one seat while running another would corrupt
	// provenance.
	repairSeatSetting := state.Project.Settings.Roles["repair_producer"]
	repairSeat := provider.Seat{Tool: repairSeatSetting.Tool, Model: repairSeatSetting.Model, Effort: repairSeatSetting.Effort}
	repairProducer, err := findAdapter(ctx, repairSeatSetting.Tool, repairSeat)
	if err != nil {
		repairProducer = nil
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
		RepairProducer: repairProducer,
		RepairSeat:     repairSeat,
		Reviewers:      reviewers,
		ReviewerSeats:  reviewerSeats,
		Evidence:       evidenceRunner,
		WorktreeParent: worktreeParent,
		Out:            env.Stdout,
	}), nil
}
