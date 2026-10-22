// This file is part of PeerFoil.
// internal/controller/review8_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Checks project-bound review authorization, complete repair provenance,
// accepted project settings, and evidence checkout integrity.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/project"
	"github.com/gabrielmongefranco/peerfoil/internal/provider"
	"github.com/gabrielmongefranco/peerfoil/internal/schema"
)

func TestReviewApprovalCoversEveryAuthor(t *testing.T) {
	runs := []reviewRun{
		{lineage: provider.LineageOpenAIGPT, answer: reviewAnswer{Decision: "approve"}},
		{lineage: provider.LineageAnthropicClaude, answer: reviewAnswer{Decision: "approve"}},
	}
	authors := map[string]string{"cs-0001": runs[0].lineage, "cs-0002": runs[1].lineage}
	if combineDecisions(runs, authors) != "approve" {
		t.Fatal("mixed authors lack per-item approval")
	}
	authors["cs-0003"] = provider.LineageUnknown
	if combineDecisions(runs, authors) == "approve" {
		t.Fatal("unknown author was approved")
	}
	delete(authors, "cs-0003")
	runs[1].lineage = runs[0].lineage
	if combineDecisions(runs, authors) == "approve" {
		t.Fatal("same-family approval counted")
	}
}

func TestReviewGateRejectsForeignProjectAndOldArchitecture(t *testing.T) {
	for _, mode := range []string{"foreign project", "body header", "old plan", "old change"} {
		t.Run(mode, func(t *testing.T) {
			root := setupFixtureRepo(t)
			engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
			if err := engine.productionGate(); err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(engine.State.Dir, "reviews")
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				name := filepath.Join(dir, entry.Name())
				raw, err := os.ReadFile(name)
				if err != nil {
					t.Fatal(err)
				}
				text := string(raw)
				switch mode {
				case "foreign project", "body header":
					text = strings.ReplaceAll(text, "Project: `"+engine.State.Project.ProjectID+"`", "Project: `synthetic-other-project`")
					if mode == "body header" {
						text += "\nProject: `" + engine.State.Project.ProjectID + "` · Assurance: Guided\n"
					}
				default:
					if !strings.Contains(text, "**Kind:** architecture") {
						text = strings.ReplaceAll(text, "architecture revision 1", "architecture revision 0")
						if mode == "old change" {
							text = strings.ReplaceAll(text, "**Kind:** plan", "**Kind:** change")
						}
					}
				}
				if err := os.WriteFile(name, []byte(text), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := engine.productionGate(); err == nil {
				t.Fatal("unrelated approval authorized production")
			}
		})
	}
}

func TestReviewInputsBindStableProjectFields(t *testing.T) {
	root := setupFixtureRepo(t)
	engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
	source := git(t, root, "rev-parse", "HEAD")
	baseline, err := engine.reviewInputs(source)
	if err != nil {
		t.Fatal(err)
	}
	original := *engine.State.Project
	for _, mutate := range []func(*project.Project){
		func(p *project.Project) { p.Goal += " changed" },
		func(p *project.Project) { p.Settings.Review.RepairCycles = 0 },
		func(p *project.Project) { p.Pack.ID = "generic" },
	} {
		changed := original
		mutate(&changed)
		raw, _ := json.Marshal(changed)
		if err := os.WriteFile(filepath.Join(engine.State.Dir, "project.json"), raw, 0o644); err != nil {
			t.Fatal(err)
		}
		current, err := engine.reviewInputs(source)
		if err != nil || string(current) == string(baseline) {
			t.Fatal("stable project change did not invalidate the bundle")
		}
	}
	original.Workflow.State, original.UpdatedAt = "review", "2026-10-21T01:00:00Z"
	raw, _ := json.Marshal(original)
	if err := os.WriteFile(filepath.Join(engine.State.Dir, "project.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	current, err := engine.reviewInputs(source)
	if err != nil || string(current) != string(baseline) {
		t.Fatal("controller workflow progress invalidated stable inputs")
	}
}

func TestReviewCheckoutRegeneratesCurrentPlan(t *testing.T) {
	root := setupFixtureRepo(t)
	engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
	engine.State.Plan.Tasks()[0].Title = "Current accepted task title"
	if err := engine.SavePlan(); err != nil {
		t.Fatal(err)
	}
	checkout := t.TempDir()
	if err := os.Mkdir(filepath.Join(checkout, ".peerfoil"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := engine.refreshRecordsInCheckout(checkout); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(checkout, ".peerfoil", "plan.md"))
	if err != nil || !strings.Contains(string(raw), "Current accepted task title") {
		t.Fatal("reviewers received stale plan Markdown")
	}
}

func TestMergedFindingsRetainDistinctObligations(t *testing.T) {
	first, second := repairFinding(), repairFinding()
	second.Requirement, second.Evidence, second.Recommendation = "r-0003", "An additional failure", "An additional repair"
	runs := []reviewRun{{recordID: "rv-0001", answer: reviewAnswer{Findings: []Finding{first}}},
		{recordID: "rv-0002", answer: reviewAnswer{Findings: []Finding{second, first}}}}
	merged := mergeFindings(runs)
	if len(merged) != 2 || len(merged[0].Reviews) != 2 || merged[1].Recommendation != second.Recommendation || merged[1].Reviews[0] != "rv-0002" {
		t.Fatal("merge discarded obligations or provenance")
	}
}

func TestEvidenceRejectsCheckoutMutation(t *testing.T) {
	for _, path := range []string{"main.go", "new-input.go"} {
		t.Run(path, func(t *testing.T) {
			root := setupFixtureRepo(t)
			engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
			calls := 0
			engine.Evidence = func(_ context.Context, _ []string, dir string, _ []string, _ time.Duration) ([]byte, []byte, int, bool, error) {
				calls++
				if err := os.WriteFile(filepath.Join(dir, path), []byte("package main\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				return []byte("pass"), nil, 0, false, nil
			}
			quality, err := engine.readAcceptedRecord("quality.md", 1)
			if err != nil {
				t.Fatal(err)
			}
			source := git(t, root, "rev-parse", "HEAD")
			checkout, err := engine.Repo.CreateReviewWorktree(context.Background(), source, "mutation", engine.WorktreeParent)
			if err != nil {
				t.Fatal(err)
			}
			ids, _, err := engine.runTaskEvidence(context.Background(), checkout, engine.State.Plan.Tasks()[0], "cs-0001", git(t, root, "rev-parse", "HEAD"), quality)
			if err == nil || calls != 1 || len(ids) != 1 {
				t.Fatal("modified source supplied passing evidence or later checks ran")
			}
			raw, readErr := os.ReadFile(filepath.Join(engine.State.Dir, "evidence", ids[0]+".md"))
			if readErr != nil || !strings.Contains(string(raw), "**Result:** blocked") {
				t.Fatal("source mutation was not retained as an error")
			}
		})
	}
}

func TestRecommendedStartupFailureRetainsErrorAndContinues(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		root := setupFixtureRepo(t)
		engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
		task := *engine.State.Plan.Tasks()[0]
		task.RequiredEvidence = []project.EvidenceRequirement{{Name: "optional-check", Kind: "executable", Level: "recommended"}, {Name: "unit-tests", Kind: "executable", Level: "required"}}
		quality := []byte("## Recommended evidence\n| optional-check | executable | `[\"missing-tool\"]` in the repository root |\n## Required evidence\n| unit-tests | executable | `[\"go\", \"test\"]` in the repository root |\n")
		calls := 0
		engine.Evidence = func(context.Context, []string, string, []string, time.Duration) ([]byte, []byte, int, bool, error) {
			calls++
			if calls == 1 {
				if cancel {
					return nil, nil, -1, false, context.Canceled
				}
				return nil, nil, -1, false, errors.New("synthetic missing tool")
			}
			return nil, nil, 0, false, nil
		}
		// The retention folder is outside this checkout, as in normal validation.
		checkout, err := engine.Repo.CreateReviewWorktree(context.Background(), git(t, root, "rev-parse", "HEAD"), "optional", engine.WorktreeParent)
		if err != nil {
			t.Fatal(err)
		}
		ids, failed, err := engine.runTaskEvidence(context.Background(), checkout, &task, "cs-0001", git(t, root, "rev-parse", "HEAD"), quality)
		if cancel {
			if err == nil || calls != 1 {
				t.Fatal("cancellation was ignored")
			}
		} else if err != nil || calls != 2 || len(ids) != 2 || len(failed) != 0 {
			t.Fatalf("optional startup failure blocked required checks: %v", err)
		}
	}
}

func TestRepairPlanTracksWorkAndExcludesDismissedScope(t *testing.T) {
	root := setupFixtureRepo(t)
	engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
	for _, task := range engine.State.Plan.ActiveTasks() {
		task.Status = "validated"
	}
	phase := &engine.State.Plan.Phases[0]
	dismissed := phase.Stages[0]
	dismissed.ID, dismissed.Status, dismissed.Order = "st-099", "deferred", 99
	dismissed.Tasks = []project.Task{dismissed.Tasks[0]}
	dismissed.Tasks[0].ID, dismissed.Tasks[0].Status, dismissed.Tasks[0].AllowedPaths = "tk-099", "deferred", []string{"*"}
	phase.Stages = append(phase.Stages, dismissed)
	finding := repairFinding()
	finding.Reviews = []string{"rv-0001", "rv-0002"}
	plan, task, err := engine.prepareRepairPlan("pr-0001", []Finding{finding})
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlanRevision != engine.State.Plan.PlanRevision+1 || task.ID == "tk-002" || task.ID == "tk-099" {
		t.Fatal("repair reused existing work")
	}
	for _, allowed := range task.AllowedPaths {
		if allowed == "*" {
			t.Fatal("dismissed task expanded the repair grant")
		}
	}
	linked := false
	for _, requirement := range plan.Requirements {
		if requirement.ID == finding.Requirement {
			for _, id := range requirement.Tasks {
				if id == task.ID {
					linked = true
				}
			}
		}
	}
	if !linked {
		t.Fatal("the finding's previously uncovered requirement lost its repair task link")
	}
	change := plan.Changes[len(plan.Changes)-1]
	if len(change.AffectedTasks) < 2 || len(change.Reviews) != 2 || !strings.Contains(strings.Join(change.RetainedTasks, ","), "tk-099") {
		t.Fatal("repair intake lost affected, retained, or review links")
	}
	if err := engine.SavePlan(); err != nil {
		t.Fatal(err)
	}
	if err := engine.acceptRepairPlan(plan, task.ID); err != nil {
		t.Fatal(err)
	}
	schemas, err := schema.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := project.Load(root, schemas); err != nil {
		t.Fatal(err)
	}
	finding.Location = "unrelated.txt"
	if _, _, err := engine.prepareRepairPlan("pr-0002", []Finding{finding}); err == nil {
		t.Fatal("dismissed scope authorized a repair")
	}
}

func TestRepairPlanWithNoRetainedTasksLoads(t *testing.T) {
	root := setupFixtureRepo(t)
	engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
	for _, task := range engine.State.Plan.ActiveTasks() {
		task.Status = "validated"
		task.AllowedPaths = []string{"main.go"}
	}
	if err := engine.SavePlan(); err != nil {
		t.Fatal(err)
	}
	plan, task, err := engine.prepareRepairPlan("pr-0001", []Finding{repairFinding()})
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.acceptRepairPlan(plan, task.ID); err != nil {
		t.Fatal(err)
	}
	schemas, err := schema.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := project.Load(root, schemas); err != nil {
		t.Fatal(err)
	}
}

func TestEvidenceAllowsIgnoredBuildOutput(t *testing.T) {
	root := setupFixtureRepo(t)
	engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
	source := git(t, root, "rev-parse", "HEAD")
	checkout, err := engine.Repo.CreateReviewWorktree(context.Background(), source, "ignored-output", engine.WorktreeParent)
	if err != nil {
		t.Fatal(err)
	}
	task := *engine.State.Plan.Tasks()[0]
	task.RequiredEvidence = []project.EvidenceRequirement{{Name: "build", Kind: "executable", Level: "required"}}
	engine.Evidence = func(_ context.Context, _ []string, dir string, _ []string, _ time.Duration) ([]byte, []byte, int, bool, error) {
		for _, name := range []string{"linecount", "linecount.exe"} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("synthetic build output"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return []byte("pass"), nil, 0, false, nil
	}
	quality, err := engine.readAcceptedRecord("quality.md", 1)
	if err != nil {
		t.Fatal(err)
	}
	ids, failed, err := engine.runTaskEvidence(context.Background(), checkout, &task, "cs-0001", source, quality)
	if err != nil || len(failed) != 0 || len(ids) != 1 {
		t.Fatalf("ignored build output blocked validation: %v", err)
	}
}
