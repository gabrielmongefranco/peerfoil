// This file is part of PeerFoil.
// internal/lifecycle/lifecycle_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Tests every allowed and forbidden workflow transition edge.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package lifecycle

import "testing"

func TestTransitionTable(t *testing.T) {
	allowed := map[[2]State]bool{}
	// The complete forward lifecycle.
	for from, targets := range forward {
		for _, to := range targets {
			allowed[[2]State{from, to}] = true
		}
	}
	for _, state := range States() {
		allowed[[2]State{state, state}] = true  // recorded events keep the state
		allowed[[2]State{state, Paused}] = true // anything can pause
		allowed[[2]State{Paused, state}] = true // resume targets are journal-checked
	}
	for _, from := range States() {
		for _, to := range States() {
			got := CanMove(from, to)
			want := allowed[[2]State{from, to}]
			if got != want {
				t.Errorf("CanMove(%s, %s) = %v, want %v", from, to, got, want)
			}
		}
	}
}

func TestForbiddenEdges(t *testing.T) {
	// A few edges that must never open silently: production before planning, approval
	// without review, and skipping validation after a repair.
	for _, edge := range [][2]State{
		{Define, Produce},
		{PlanState, Review},
		{Produce, Approve},
		{Repair, Approve},
		{Validate, Approve},
		{Approve, Review},
	} {
		if CanMove(edge[0], edge[1]) {
			t.Errorf("CanMove(%s, %s) must be false", edge[0], edge[1])
		}
		if err := Move(edge[0], edge[1]); err == nil {
			t.Errorf("Move(%s, %s) must fail", edge[0], edge[1])
		}
	}
}

func TestUnknownStates(t *testing.T) {
	if CanMove("nonsense", Produce) || CanMove(Produce, "nonsense") {
		t.Error("unknown states must never move")
	}
	if Valid("nonsense") {
		t.Error("nonsense is not a valid state")
	}
}
