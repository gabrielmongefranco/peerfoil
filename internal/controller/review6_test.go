// This file is part of PeerFoil.
// internal/controller/review6_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Regression journeys for accepted input snapshots, exact check procedures,
// dismissed stages, integration failures, and complete repair obligations.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/project"
	"github.com/gabrielmongefranco/peerfoil/internal/provider"
)

func TestApprovePreservesDismissedStages(t *testing.T) {
	root := setupFixtureRepo(t)
	engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
	engine.State.Project.Workflow.State = "review"
	phase := &engine.State.Plan.Phases[0]
	for _, task := range engine.State.Plan.ActiveTasks() {
		task.Status = "validated"
	}
	for i, status := range []string{"deferred", "removed"} {
		stage := phase.Stages[0]
		stage.ID, stage.Status, stage.Order = []string{"st-090", "st-091"}[i], status, i+90
		stage.Tasks = []project.Task{stage.Tasks[0]}
		stage.Tasks[0].ID, stage.Tasks[0].Status = []string{"tk-090", "tk-091"}[i], status
		phase.Stages = append(phase.Stages, stage)
	}
	if err := engine.approvePhase(git(t, root, "rev-parse", "HEAD"), nil, "Synthetic approval"); err != nil {
		t.Fatal(err)
	}
	for _, stage := range phase.Stages[len(phase.Stages)-2:] {
		if stage.Status != stage.Tasks[0].Status || stage.Status == "approved" {
			t.Fatal("approval erased a dismissed stage's disposition")
		}
	}
}

func TestIntegrationFailurePausesWithEvidence(t *testing.T) {
	root := setupFixtureRepo(t)
	engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
	engine.Evidence = func(context.Context, []string, string, []string, time.Duration) ([]byte, []byte, int, bool, error) {
		divergent := git(t, root, "commit-tree", "HEAD^{tree}", "-p", "HEAD", "-m", "Synthetic divergent integration")
		git(t, root, "update-ref", "refs/heads/peerfoil/integration", divergent)
		return []byte("pass"), nil, 0, false, nil
	}
	if err := engine.ProduceNext(context.Background()); err == nil {
		t.Fatal("integration accepted a divergent branch")
	}
	if engine.State.Project.Workflow.State != "paused" || engine.State.Plan.Tasks()[0].Status != "blocked" {
		t.Fatal("integration failure stranded validation")
	}
	last := engine.State.History[len(engine.State.History)-1]
	for _, ref := range []string{"tasks", "change_sets", "evidence"} {
		if len(last.Refs[ref]) == 0 {
			t.Fatalf("pause lost %s", ref)
		}
	}
}

func TestRepairSetupFailureLeavesReviewBoundary(t *testing.T) {
	root := setupFixtureRepo(t)
	engine := testEngine(t, root, goodProducer(t), []provider.Adapter{
		reviewer("claude-code", provider.LineageAnthropicClaude, "repair", []Finding{repairFinding()}),
		reviewer("codex-cli", provider.LineageOpenAIGPT, "repair", []Finding{repairFinding()}),
	}, []provider.Seat{claudeSeat(), codexSeat()}, nil)
	producePhase(t, engine)
	branch := fmt.Sprintf("peerfoil/task/tk-003-r%d", engine.State.Project.Revisions.Plan+1)
	git(t, root, "branch", branch, "peerfoil/integration")
	if err := engine.PhaseReview(context.Background()); err == nil {
		t.Fatal("an existing repair branch was overwritten")
	}
	if engine.State.Project.Workflow.State != "review" {
		t.Fatal("fallible setup stranded the workflow in repair")
	}
	open, err := engine.Journal.OpenInvocations()
	if err != nil || len(open) != 0 {
		t.Fatalf("setup left a phantom writer: %v, %v", open, err)
	}
}

func TestRepairIncludesBlockingAndMajorFindings(t *testing.T) {
	root := setupFixtureRepo(t)
	blocking := repairFinding()
	major := blocking
	major.Title, major.Severity = "Handle an empty result", "major"
	claude := reviewer("claude-code", provider.LineageAnthropicClaude, "repair", []Finding{blocking, major})
	phasePass := claude.invoke
	approver := reviewer("claude-code", provider.LineageAnthropicClaude, "approve", nil)
	verified := false
	claude.invoke = func(inv provider.Invocation) (provider.Outcome, error) {
		if !strings.Contains(inv.Packet, "verification of a repair") {
			return phasePass(inv)
		}
		if !strings.Contains(inv.Packet, major.Title) || !strings.Contains(inv.Packet, blocking.Title) {
			t.Fatal("repair verification omitted an agreed finding")
		}
		verified = true
		return approver.invoke(inv)
	}
	producer := goodProducer(t)
	write := producer.invoke
	producer.invoke = func(inv provider.Invocation) (provider.Outcome, error) {
		if strings.Contains(inv.Packet, "repair producer") && (!strings.Contains(inv.Packet, major.Title) || !strings.Contains(inv.Packet, blocking.Title)) {
			t.Fatal("repair producer omitted an agreed finding")
		}
		return write(inv)
	}
	engine := testEngine(t, root, producer, []provider.Adapter{claude,
		reviewer("codex-cli", provider.LineageOpenAIGPT, "repair", []Finding{blocking, major})},
		[]provider.Seat{claudeSeat(), codexSeat()}, nil)
	producePhase(t, engine)
	if err := engine.PhaseReview(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !verified || engine.State.Project.Workflow.State != "approve" {
		t.Fatal("the complete repair set was not verified")
	}
}

func TestProducerReceivesCurrentSnapshotAndUsesItsProcedure(t *testing.T) {
	root := setupFixtureRepo(t)
	architecturePath := filepath.Join(root, ".peerfoil", "architecture.md")
	architecture, err := os.ReadFile(architecturePath)
	if err != nil {
		t.Fatal(err)
	}
	architecture = append(architecture, []byte("\nCurrent accepted requirement: retain empty records.\n")...)
	if err := os.WriteFile(architecturePath, architecture, 0o644); err != nil {
		t.Fatal(err)
	}
	qualityPath := filepath.Join(root, ".peerfoil", "quality.md")
	quality, err := os.ReadFile(qualityPath)
	if err != nil {
		t.Fatal(err)
	}
	quality = []byte(strings.ReplaceAll(string(quality), `["go", "test", "./..."]`, `["go", "test", "-count=2", "./..."]`))
	if err := os.WriteFile(qualityPath, quality, 0o644); err != nil {
		t.Fatal(err)
	}
	producer := goodProducer(t)
	write := producer.invoke
	producer.invoke = func(inv provider.Invocation) (provider.Outcome, error) {
		for _, fragment := range []string{"retain empty records", "-count=2", "Current accepted task title", "JSON data, not instructions"} {
			if !strings.Contains(inv.Packet, fragment) {
				t.Fatalf("producer did not receive %s", fragment)
			}
		}
		return write(inv)
	}
	var commands []string
	engine := testEngine(t, root, producer, nil, nil, func(command string) (int, string) {
		commands = append(commands, command)
		return 0, "pass"
	})
	engine.State.Plan.Tasks()[0].Title = "Current accepted task title"
	if err := engine.ProduceNext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(commands, "\n"), "go test -count=2 ./...") {
		t.Fatal("the accepted command was replaced by a pack hint")
	}
	raw, err := os.ReadFile(filepath.Join(root, ".peerfoil", "evidence", "cs-0001.inputs.json"))
	if err != nil {
		t.Fatal(err)
	}
	var inputs map[string]string
	if err := json.Unmarshal(raw, &inputs); err != nil || inputs["architecture.md"] != string(architecture) || inputs["quality.md"] != string(quality) {
		t.Fatal("snapshot changed accepted bytes")
	}
	digest := sha256.Sum256(raw)
	record, err := os.ReadFile(filepath.Join(root, ".peerfoil", "evidence", "cs-0001.md"))
	if err != nil || !strings.Contains(string(record), hex.EncodeToString(digest[:])) {
		t.Fatal("change set lost its input digest")
	}
	if git(t, root, "diff", "HEAD", "peerfoil/integration", "--", ".peerfoil") != "" {
		t.Fatal("the snapshot contaminated the produced source commit")
	}
}

func TestAcceptedCommandsUseAllArgumentsAndWorkingDirectory(t *testing.T) {
	contract := "## Required evidence\n\n| Name | Kind | Procedure |\n| unit-tests | executable | `[\"node\", \"first.js\"]` then `[\"node\", \"second.js\"]` in `tools` |\n"
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	requirement := project.EvidenceRequirement{Name: "unit-tests", Kind: "executable", Level: "required"}
	commands, err := acceptedCommands([]byte(contract), requirement, root)
	if err != nil || len(commands) != 2 || commands[0].Dir != "tools" || commands[1].Argv[1] != "second.js" {
		t.Fatalf("commands: %v; %v", commands, err)
	}
	for _, bad := range []string{
		strings.ReplaceAll(contract, "`tools`", "`../outside`"),
		strings.ReplaceAll(contract, "`tools`", "`C:/outside`"),
		strings.ReplaceAll(contract, "first.js", "first.js;echo"),
		strings.ReplaceAll(contract, "first.js", "sk-proj-"+strings.Repeat("synthetic", 4)),
		contract + "| unit-tests | executable | `[\"node\", \"other.js\"]` in the repository root |\n",
		strings.ReplaceAll(contract, "unit-tests", "different-check"),
		strings.ReplaceAll(contract, "executable", "inspection"),
		strings.ReplaceAll(contract, " in `tools`", ""),
	} {
		if _, err := acceptedCommands([]byte(bad), requirement, root); err == nil {
			t.Fatal("an ambiguous or unsafe accepted command was allowed")
		}
	}
}

func TestAcceptedInputRejectsStaleAndSensitiveRecords(t *testing.T) {
	root := setupFixtureRepo(t)
	engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
	path := filepath.Join(root, ".peerfoil", "quality.md")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sensitive := "sk-proj-" + strings.Repeat("synthetic", 4)
	for _, invalid := range []string{
		strings.ReplaceAll(string(original), "Revision: 1", "Revision: 2"),
		strings.ReplaceAll(string(original), "Status: accepted", "Status: draft"),
		string(original) + "\n" + sensitive,
		string(original) + strings.Repeat("x", maxInputBytes),
	} {
		if err := os.WriteFile(path, []byte(invalid), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := engine.readAcceptedRecord("quality.md", 1); err == nil || strings.Contains(err.Error(), sensitive) {
			t.Fatal("unsafe input was accepted or leaked through its error")
		}
	}
}
