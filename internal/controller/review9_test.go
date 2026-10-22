// This file is part of PeerFoil.
// internal/controller/review9_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Checks evidence-procedure isolation and verified repair-plan continuity,
// including retained check references and later-phase authorization failures.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package controller

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/project"
	"github.com/gabrielmongefranco/peerfoil/internal/provider"
)

func TestEvidenceProceduresCannotShareIgnoredInputs(t *testing.T) {
	root := setupFixtureRepo(t)
	ignore := filepath.Join(root, ".gitignore")
	raw, err := os.ReadFile(ignore)
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, []byte("\n.env\ngenerated.go\ntool.conf\n")...)
	if err := os.WriteFile(ignore, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, root, "add", ".gitignore")
	git(t, root, "commit", "-m", "Synthetic ignored inputs")
	engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
	task := *engine.State.Plan.Tasks()[0]
	task.RequiredEvidence = []project.EvidenceRequirement{{Name: "build", Kind: "executable", Level: "required"}, {Name: "unit-tests", Kind: "executable", Level: "required"}}
	calls := 0
	engine.Evidence = func(_ context.Context, _ []string, dir string, _ []string, _ time.Duration) ([]byte, []byte, int, bool, error) {
		calls++
		for _, name := range []string{".env", "generated.go", "tool.conf"} {
			path := filepath.Join(dir, name)
			if calls == 1 {
				if err := os.WriteFile(path, []byte("synthetic input"), 0o644); err != nil {
					t.Fatal(err)
				}
			} else if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("a later procedure consumed ignored output")
			}
		}
		return []byte("pass"), nil, 0, false, nil
	}
	quality, err := engine.readAcceptedRecord("quality.md", 1)
	if err != nil {
		t.Fatal(err)
	}
	ids, failed, err := engine.runTaskEvidence(context.Background(), &task, "cs-0001", git(t, root, "rev-parse", "HEAD"), quality)
	if err != nil || len(failed) != 0 || len(ids) != 2 || calls != 2 {
		t.Fatalf("isolated procedures failed: %v", err)
	}
}

func TestVerifiedRepairAuthorizesRetainedLaterPhase(t *testing.T) {
	root := setupFixtureRepo(t)
	claude := reviewer("claude-code", provider.LineageAnthropicClaude, "repair", []Finding{repairFinding()})
	phasePass := claude.invoke
	approve := reviewer("claude-code", provider.LineageAnthropicClaude, "approve", nil)
	claude.invoke = func(inv provider.Invocation) (provider.Outcome, error) {
		if strings.Contains(inv.Packet, "verification of a repair") {
			return approve.invoke(inv)
		}
		return phasePass(inv)
	}
	engine := testEngine(t, root, goodProducer(t), []provider.Adapter{claude,
		reviewer("codex-cli", provider.LineageOpenAIGPT, "repair", []Finding{repairFinding()})}, []provider.Seat{claudeSeat(), codexSeat()}, nil)
	later := *engine.State.Plan.Tasks()[0]
	later.ID, later.Status, later.DependsOn = "tk-090", "planned", []string{}
	later.PlanRevision = engine.State.Plan.PlanRevision
	engine.State.Plan.Phases = append(engine.State.Plan.Phases, project.Phase{ID: "ph-02", Title: "Later work", Outcome: "A retained later task", Order: 2, Status: "planned",
		Stages: []project.Stage{{ID: "st-090", Title: "Later task", Outcome: "The later task is validated", Order: 1, Status: "planned", DependsOn: []string{}, Tasks: []project.Task{later}}}})
	if err := engine.SavePlan(); err != nil {
		t.Fatal(err)
	}
	producePhase(t, engine)
	if err := engine.PhaseReview(context.Background()); err != nil {
		t.Fatal(err)
	}
	change := engine.State.Plan.Changes[len(engine.State.Plan.Changes)-1]
	if len(change.Evidence) == 0 {
		t.Fatal("the repair change lost its evidence IDs")
	}
	checkpoint, err := engine.loadRepairVerification(engine.reviewBoundary())
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range change.Evidence {
		raw, err := os.ReadFile(filepath.Join(engine.State.Dir, "evidence", id+".md"))
		if err != nil || !strings.Contains(string(raw), "**Source revision:** "+checkpoint.Source) || !strings.Contains(string(raw), "**Change set:** "+checkpoint.ChangeSet) {
			t.Fatal("repair evidence does not identify the captured repair")
		}
	}
	reviews, err := engine.loadReviews()
	if err != nil {
		t.Fatal(err)
	}
	if !engine.repairAgreementAuthorizesPlan(reviews) {
		t.Fatal("verified repair plan has no current authorization")
	}
	history := engine.State.History
	engine.State.History = history[:len(history)-1]
	if engine.repairAgreementAuthorizesPlan(reviews) {
		t.Fatal("verification without phase approval authorized the plan")
	}
	engine.State.History = history
	for i := range reviews {
		if reviews[i].ID == change.Reviews[0] {
			reviews[i].LineageRoot = provider.LineageUnknown
		}
	}
	if engine.repairAgreementAuthorizesPlan(reviews) {
		t.Fatal("missing two-family agreement authorized the plan")
	}
	_, _, retained := engine.State.Plan.FindTask("tk-090")
	originalTitle := retained.Title
	retained.Title += " changed without intake"
	reviews, err = engine.loadReviews()
	if err != nil {
		t.Fatal(err)
	}
	if engine.repairAgreementAuthorizesPlan(reviews) {
		t.Fatal("modified future assignments inherited approval")
	}
	retained.Title = originalTitle
	retained.Status = "deferred"
	if engine.repairAgreementAuthorizesPlan(reviews) {
		t.Fatal("dismissing work silently changed the accepted plan")
	}
	retained.Status = "planned"
	phase, stage, retained := engine.State.Plan.FindTask("tk-090")
	phase.Status, stage.Status, retained.Status = "active", "active", "ready"
	engine.State.Project.Workflow.Phase, engine.State.Project.Workflow.Stage = &phase.ID, &stage.ID
	if err := engine.SavePlan(); err != nil {
		t.Fatal(err)
	}
	if err := engine.AppendTransition(Produce, nil, "The user started the retained later phase.", &checkpoint.Source); err != nil {
		t.Fatal(err)
	}
	resumed := testEngine(t, root, goodProducer(t), nil, nil, nil)
	if err := resumed.ProduceNext(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, _, retained = resumed.State.Plan.FindTask("tk-090")
	if retained.Status != "validated" {
		t.Fatal("the authorized later task did not run")
	}
}
