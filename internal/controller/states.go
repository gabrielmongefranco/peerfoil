// This file is part of PeerFoil.
// internal/controller/states.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: The controller's view of the workflow state table, which lives in the
// lifecycle package so the record loader can validate history against the same edges.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

// Package controller decides which workflow step is allowed next. Models propose;
// this package validates transitions and accepted state.
package controller

import "github.com/gabrielmongefranco/peerfoil/internal/lifecycle"

// State is one of the fixed workflow states shared with the Skills release.
type State = lifecycle.State

const (
	Define    = lifecycle.Define
	Architect = lifecycle.Architect
	PlanState = lifecycle.PlanState
	Produce   = lifecycle.Produce
	Validate  = lifecycle.Validate
	Review    = lifecycle.Review
	Repair    = lifecycle.Repair
	Approve   = lifecycle.Approve
	Paused    = lifecycle.Paused
)

// States lists every valid state, in lifecycle order.
func States() []State { return lifecycle.States() }

// Valid reports whether the name is a known state.
func Valid(name string) bool { return lifecycle.Valid(name) }

// CanMove reports whether the workflow may move from one state to another.
func CanMove(from, to State) bool { return lifecycle.CanMove(from, to) }

// Move returns a Policy error when the transition is not permitted.
func Move(from, to State) error { return lifecycle.Move(from, to) }
