// This file is part of PeerFoil.
// internal/project/history_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Rejects ambiguous transition identifiers and missing controller review revisions.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package project

import (
	"strings"
	"testing"
)

func TestHistoryIDsMustIncreaseNumerically(t *testing.T) {
	for _, id := range []string{"tr-0000", "tr-0001", "tr-0004", "tr-00004", "tr-0003"} {
		t.Run(id, func(t *testing.T) {
			state, err := Load(fixtureRoot(t, "software-go"), schemas(t))
			if err != nil {
				t.Fatal(err)
			}
			state.History[len(state.History)-1].TransitionID = id
			if got := strings.Join(state.semanticProblems(), "\n"); !strings.Contains(got, "unique and increasing") {
				t.Fatalf("accepted duplicate or regressing identifier: %s", got)
			}
		})
	}
	state, err := Load(fixtureRoot(t, "software-go"), schemas(t))
	if err != nil {
		t.Fatal(err)
	}
	state.History[len(state.History)-1].TransitionID = "tr-100000000000000000000000000000000000000"
	if got := state.semanticProblems(); len(got) != 0 {
		t.Fatalf("a larger valid identifier was rejected: %v", got)
	}
}

func TestControllerReviewBoundaryRequiresRevision(t *testing.T) {
	state, err := Load(fixtureRoot(t, "software-go"), schemas(t))
	if err != nil {
		t.Fatal(err)
	}
	model, from := "peerfoil-core", "produce"
	transition := state.History[len(state.History)-1]
	transition.Actor.Model = &model
	transition.TransitionID = "tr-0006"
	transition.FromState, transition.ToState = &from, "validate"
	state.History = append(state.History, transition)
	fromReview := "validate"
	transition.TransitionID, transition.FromState, transition.ToState = "tr-0007", &fromReview, "review"
	state.History = append(state.History, transition)
	state.Project.Workflow.State = "review"
	if got := strings.Join(state.semanticProblems(), "\n"); !strings.Contains(got, "requires a source revision") {
		t.Fatalf("a missing review revision was accepted: %s", got)
	}
	// Guided history can use a retained patch instead of a source commit.
	guided := "synthetic-guided-model"
	state.History[len(state.History)-1].Actor.Model = &guided
	if got := state.semanticProblems(); len(got) != 0 {
		t.Fatalf("guided null revision was rejected: %v", got)
	}
}
