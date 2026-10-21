// This file is part of PeerFoil.
// internal/lifecycle/lifecycle.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: The deterministic workflow state table. The controller permits exactly the
// transitions the architecture defines, and the record loader validates recorded
// history against the same table.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

// Package lifecycle holds the fixed workflow states and the permitted transitions
// between them. It lives apart from the controller so the project-record loader can
// validate history edges against the same table without an import cycle.
package lifecycle

import (
	"fmt"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

// State is one of the fixed workflow states shared with the Skills release.
type State string

const (
	Define    State = "define"
	Architect State = "architect"
	PlanState State = "plan"
	Produce   State = "produce"
	Validate  State = "validate"
	Review    State = "review"
	Repair    State = "repair"
	Approve   State = "approve"
	Paused    State = "paused"
)

// States lists every valid state, in lifecycle order.
func States() []State {
	return []State{Define, Architect, PlanState, Produce, Validate, Review, Repair, Approve, Paused}
}

// forward holds the permitted forward edges of the lifecycle. A state may also repeat
// itself for a recorded event that does not move the workflow, such as change intake,
// and any state may pause. Resuming from paused is validated against the journal's
// recorded prior state by the caller, so the table allows paused to reach any state.
var forward = map[State][]State{
	Define:    {Architect},
	Architect: {PlanState},
	PlanState: {Produce},
	Produce:   {Validate},
	Validate:  {Produce, Review},
	Review:    {Repair, Approve},
	Repair:    {Validate},
	Approve:   {Produce},
}

// Valid reports whether the name is a known state.
func Valid(name string) bool {
	for _, state := range States() {
		if State(name) == state {
			return true
		}
	}
	return false
}

// CanMove reports whether the workflow may move from one state to another.
func CanMove(from, to State) bool {
	if !Valid(string(from)) || !Valid(string(to)) {
		return false
	}
	if from == to {
		return true
	}
	if to == Paused || from == Paused {
		return true
	}
	for _, next := range forward[from] {
		if next == to {
			return true
		}
	}
	return false
}

// Move returns a Policy error when the transition is not permitted, naming both states
// so the refusal is understandable from the message alone.
func Move(from, to State) error {
	if CanMove(from, to) {
		return nil
	}
	return errs.New(errs.Policy,
		fmt.Sprintf("the workflow cannot move from %s to %s", from, to),
		"run peerfoil status to see the current state and the next permitted step")
}
