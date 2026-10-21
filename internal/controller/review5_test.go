// This file is part of PeerFoil.
// internal/controller/review5_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Checks frozen review boundaries, repair retry recovery, retained setup
// checks, and redaction of untrusted review and blocker text.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/project"
	"github.com/gabrielmongefranco/peerfoil/internal/provider"
)

func producePhase(t *testing.T, engine *Engine) {
	t.Helper()
	for range 2 {
		if err := engine.ProduceNext(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPhaseReviewRequiresRecordedRevision(t *testing.T) {
	for _, missing := range []string{"null", "empty", "absent history"} {
		t.Run(missing, func(t *testing.T) {
			root := setupFixtureRepo(t)
			engine := testEngine(t, root, goodProducer(t), []provider.Adapter{
				reviewer("claude-code", provider.LineageAnthropicClaude, "approve", nil),
				reviewer("codex-cli", provider.LineageOpenAIGPT, "approve", nil),
			}, []provider.Seat{claudeSeat(), codexSeat()}, nil)
			producePhase(t, engine)
			switch missing {
			case "null":
				engine.reviewBoundary().SourceRevision = nil
			case "empty":
				empty := ""
				engine.reviewBoundary().SourceRevision = &empty
			default:
				engine.State.History = nil
			}
			if err := engine.PhaseReview(context.Background()); err == nil || !strings.Contains(err.Error(), "no recorded source revision") {
				t.Fatalf("missing boundary: %v", err)
			}
		})
	}
}

func TestRepairVerificationResumesWithoutAnotherWriter(t *testing.T) {
	for _, failure := range []string{"timeout", "malformed", "no answer", "retry exhausted", "moved branch"} {
		t.Run(failure, func(t *testing.T) {
			root := setupFixtureRepo(t)
			finding := repairFinding()
			claude := reviewer("claude-code", provider.LineageAnthropicClaude, "repair", []Finding{finding})
			codex := reviewer("codex-cli", provider.LineageOpenAIGPT, "repair", []Finding{finding})
			phaseReview := codex.invoke
			approver := reviewer("codex-cli", provider.LineageOpenAIGPT, "approve", nil)
			attempts, phaseCalls, writes := 0, 0, 0
			codex.invoke = func(inv provider.Invocation) (provider.Outcome, error) {
				if !strings.Contains(inv.Packet, "verification of a repair") {
					phaseCalls++
					return phaseReview(inv)
				}
				attempts++
				if !strings.Contains(inv.Packet, finding.Title) {
					t.Fatal("the verifier lost the accepted finding")
				}
				if attempts == 1 || failure == "retry exhausted" {
					switch failure {
					case "timeout":
						return provider.Outcome{}, errs.New(errs.Timeout, "synthetic timeout", "")
					case "malformed":
						return provider.Outcome{Text: "```json\n{\n```"}, nil
					default:
						return provider.Outcome{}, nil
					}
				}
				if inv.Sandbox != provider.SandboxReadOnly {
					t.Fatal("repair verification has write access")
				}
				return approver.invoke(inv)
			}
			repairer := goodProducer(t)
			repairer.name, repairer.lineage = "claude-code", provider.LineageAnthropicClaude
			write := repairer.invoke
			repairer.invoke = func(inv provider.Invocation) (provider.Outcome, error) {
				writes++
				return write(inv)
			}
			engine := testEngine(t, root, goodProducer(t), []provider.Adapter{claude, codex},
				[]provider.Seat{claudeSeat(), codexSeat()}, nil)
			engine.RepairProducer, engine.RepairSeat = repairer, claudeSeat()
			engine.State.Project.Settings.Roles["repair_producer"] = project.Seat{Tool: "claude-code", Model: "claude-fable-5", Effort: "medium"}
			producePhase(t, engine)
			base := git(t, root, "rev-parse", "peerfoil/integration")
			if err := engine.PhaseReview(context.Background()); err == nil {
				t.Fatal("the failed verifier approved the repair")
			}
			if got := git(t, root, "rev-parse", "peerfoil/integration"); got != base {
				t.Fatal("the unverified repair was integrated")
			}
			if failure == "moved branch" {
				checkpoint, err := engine.loadRepairVerification(engine.reviewBoundary())
				if err != nil {
					t.Fatal(err)
				}
				branch := fmt.Sprintf("peerfoil/task/%s-r%d", checkpoint.Task, checkpoint.Revisions.Plan+1)
				git(t, root, "update-ref", branch, base)
			}
			// A new controller reconstructs only from retained records and Git.
			resumed := testEngine(t, root, goodProducer(t), []provider.Adapter{claude, codex},
				[]provider.Seat{claudeSeat(), codexSeat()}, nil)
			if failure == "moved branch" || failure == "retry exhausted" {
				if err := resumed.Resume(context.Background(), false); err == nil {
					t.Fatal("an invalid repair recovery was approved")
				}
				if writes != 1 || phaseCalls != 1 || git(t, root, "rev-parse", "peerfoil/integration") != base {
					t.Fatal("failed recovery reran a writer or integrated unverified work")
				}
				if failure == "moved branch" && attempts != 1 {
					t.Fatal("the verifier reviewed a moved branch")
				}
				if failure == "retry exhausted" && (attempts != 2 || resumed.State.Project.Workflow.State != "paused") {
					t.Fatal("the exhausted retry did not pause")
				}
				return
			}
			if err := resumed.Resume(context.Background(), false); err != nil {
				t.Fatalf("resume repair verification: %v", err)
			}
			if writes != 1 || phaseCalls != 1 || attempts != 2 || resumed.State.Project.Workflow.State != "approve" {
				t.Fatalf("writes=%d phase calls=%d attempts=%d state=%s", writes, phaseCalls, attempts, resumed.State.Project.Workflow.State)
			}
			last := resumed.State.History[len(resumed.State.History)-1]
			if len(last.Refs["reviews"]) != 1 || len(last.Refs["change_sets"]) != 1 {
				t.Fatal("approval lost the repair verification references")
			}
		})
	}
}

func TestReviewProseIsScrubbedBeforeRetention(t *testing.T) {
	root := setupFixtureRepo(t)
	engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
	secret := "sk-proj-" + strings.Repeat("synthetic", 4)
	finding := Finding{Title: secret, Location: secret, Requirement: secret, Severity: "minor", Evidence: secret, Recommendation: secret}
	seat := claudeSeat()
	r := reviewer(seat.Tool, provider.LineageAnthropicClaude, "approve", []Finding{finding})
	invoke := r.invoke
	r.invoke = func(inv provider.Invocation) (provider.Outcome, error) {
		out, err := invoke(inv)
		out.Text = strings.Replace(out.Text, "None beyond the fixture's known seeds.", secret, 1)
		return out, err
	}
	frozen := git(t, root, "rev-parse", "HEAD")
	authors := map[string]string{"cs-0001": provider.LineageOpenAIGPT}
	run, err := engine.runReviewer(context.Background(), r, seat, "phase", engine.buildReviewPacket("phase", frozen, "digest", nil), root, frozen, authors, "digest", "")
	if err != nil {
		t.Fatal(err)
	}
	runs := []reviewRun{*run}
	if err := engine.writePhaseReviewRecord("pr-0001", frozen, "digest", authors, runs, mergeFindings(runs), "approve"); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{run.recordID, "pr-0001"} {
		raw, err := os.ReadFile(filepath.Join(root, ".peerfoil", "reviews", id+".md"))
		if err != nil || strings.Contains(string(raw), secret) || !strings.Contains(string(raw), "[redacted]") {
			t.Fatalf("unsafe retained review %s: %v", id, err)
		}
	}
}

func TestRepairRecoveryRetainsTwoMaximumReviewAnswers(t *testing.T) {
	root := setupFixtureRepo(t)
	engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
	frozen := git(t, root, "rev-parse", "HEAD")
	model := "claude-fable-5"
	author := project.Actor{Role: "repair_producer", Tool: "claude-code", Model: &model, LineageRoot: provider.LineageAnthropicClaude}
	checkpoint := repairVerification{PhaseReview: "pr-0001", ChangeSet: "cs-0001", Task: "tk-001",
		Revisions: engine.State.Project.Revisions, Base: frozen, Source: frozen, Author: author}
	for round := 0; round < 2; round++ {
		answer := reviewAnswer{Kind: "phase", Decision: "repair", Model: model}
		answer.Reviewed.SourceRevision = frozen
		answer.Reviewed.PlanRevision = checkpoint.Revisions.Plan
		answer.Reviewed.ArchitectureRevision = checkpoint.Revisions.Architecture
		for index := 0; index < maxFindings; index++ {
			field := strings.Repeat("x", 1285)
			answer.Findings = append(answer.Findings, Finding{Title: fmt.Sprintf("%d-%d %s", round, index, field),
				Location: field, Requirement: field, Severity: "major", Evidence: field, Recommendation: field})
		}
		raw, err := json.Marshal(answer)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := parseReviewAnswer("```json\n"+string(raw)+"\n```", "phase", frozen,
			checkpoint.Revisions.Plan, checkpoint.Revisions.Architecture)
		if err != nil {
			t.Fatalf("valid bounded answer: %v", err)
		}
		checkpoint.Findings = append(checkpoint.Findings, parsed.Findings...)
	}
	if err := engine.WriteChangeSet(&ChangeSet{ID: checkpoint.ChangeSet, TaskID: checkpoint.Task,
		CaptureStatus: "captured", BaseRevision: frozen, SourceRev: frozen, Author: author}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(engine.State.Dir, "reviews", checkpoint.PhaseReview+".md")
	if err := os.WriteFile(path, []byte("# Phase review\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := engine.saveRepairVerification(&checkpoint); err != nil {
		t.Fatal(err)
	}
	boundary := &project.Transition{SourceRevision: &frozen, PlanRevision: checkpoint.Revisions.Plan,
		Refs: map[string][]string{"phase_reviews": {checkpoint.PhaseReview}, "change_sets": {checkpoint.ChangeSet}}}
	loaded, err := engine.loadRepairVerification(boundary)
	if err != nil || len(loaded.Findings) != 2*maxFindings {
		t.Fatalf("large valid recovery record: %v", err)
	}
}

func TestBlockedHandoffIsBoundedAndScrubbed(t *testing.T) {
	for _, oversized := range []bool{false, true} {
		t.Run(fmt.Sprint(oversized), func(t *testing.T) {
			root := setupFixtureRepo(t)
			secret := "sk-ant-" + strings.Repeat("synthetic", 4)
			producer := goodProducer(t)
			invoke := producer.invoke
			producer.invoke = func(inv provider.Invocation) (provider.Outcome, error) {
				out, err := invoke(inv)
				blocker := "Cannot use " + secret
				if oversized {
					blocker += strings.Repeat("x", 2100)
				}
				encoded, _ := json.Marshal(blocker)
				out.Text = strings.Replace(out.Text, `"result": "produced"`, `"result": "blocked"`, 1)
				out.Text = strings.Replace(out.Text, `"blocker": null`, `"blocker": `+string(encoded), 1)
				return out, err
			}
			engine := testEngine(t, root, producer, nil, nil, nil)
			var terminal bytes.Buffer
			engine.Out = &terminal
			err := engine.ProduceNext(context.Background())
			if err == nil || strings.Contains(err.Error()+terminal.String(), secret) {
				t.Fatal("the blocked handoff was accepted or exposed a credential")
			}
			for _, name := range []string{"project.json", "history.jsonl", "evidence/cs-0001.md"} {
				raw, readErr := os.ReadFile(filepath.Join(root, ".peerfoil", filepath.FromSlash(name)))
				if readErr != nil || strings.Contains(string(raw), secret) {
					t.Fatalf("unsafe retained blocker in %s: %v", name, readErr)
				}
			}
		})
	}
}

func TestSetupChecksSurviveWorkflowSaves(t *testing.T) {
	root := setupFixtureRepo(t)
	path := filepath.Join(root, ".peerfoil", "project.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc["environment"] = map[string]any{"checked_at": "2026-10-21T00:00:00Z", "checks": []any{
		map[string]any{"name": "host-sandbox", "status": "not-verified", "detail": "No host isolation was detected."},
	}}
	raw, _ = json.Marshal(doc)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
	if err := engine.ProduceNext(context.Background()); err != nil {
		t.Fatal(err)
	}
	state := testEngine(t, root, goodProducer(t), nil, nil, nil).State
	if state.Project.Environment == nil || len(state.Project.Environment.Checks) != 1 || state.Project.Environment.Checks[0].Name != "host-sandbox" {
		t.Fatal("saving the workflow erased the setup checks")
	}
}
