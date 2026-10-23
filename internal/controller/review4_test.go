// This file is part of PeerFoil.
// internal/controller/review4_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Journeys for the repair seat's identity and recovery, the review's frozen
// revision binding, the reduced-assurance acceptance gate, evidence runner failures,
// and the scrubbing of model prose in change-set records.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/provider"
)

// repairFinding is the agreed finding both scripted reviewers raise in these journeys.
func repairFinding() Finding {
	return Finding{
		Title: "Missing-file error handling", Location: "main.go", Requirement: "r-0002",
		Severity: "blocking", Evidence: "os.ReadFile's error is discarded",
		Recommendation: "print a readable error and exit 1",
	}
}

// TestRepairRunsTheConfiguredRepairSeat pins that the repair is written by the wired
// repair adapter, never by the producer seat, so the recorded lineage is the writer's.
func TestRepairRunsTheConfiguredRepairSeat(t *testing.T) {
	root := setupFixtureRepo(t)
	finding := repairFinding()
	claude := reviewer("claude-code", provider.LineageAnthropicClaude, "repair", []Finding{finding})
	codex := reviewer("codex-cli", provider.LineageOpenAIGPT, "repair", []Finding{finding})
	claude.invoke = wrapKindAware(claude.invoke, "approve")
	producer := goodProducer(t)
	produceOnly := producer.invoke
	producer.invoke = func(inv provider.Invocation) (provider.Outcome, error) {
		if strings.Contains(inv.Packet, "repair producer") {
			t.Error("the repair ran through the producer seat instead of the repair seat")
			return provider.Outcome{}, fmt.Errorf("wrong seat")
		}
		return produceOnly(inv)
	}
	repairCalled := false
	repairer := &scripted{name: "codex-cli", lineage: provider.LineageOpenAIGPT,
		invoke: func(inv provider.Invocation) (provider.Outcome, error) {
			repairCalled = true
			path := filepath.Join(inv.Dir, "main.go")
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			repaired := string(original) + "\n// repaired by the repair seat\n"
			if err := os.WriteFile(path, []byte(repaired), 0o644); err != nil {
				t.Fatal(err)
			}
			return provider.Outcome{Text: "Handled the agreed findings in main.go.", SessionID: "thread-repair-seat"}, nil
		}}
	engine := testEngine(t, root, producer,
		[]provider.Adapter{claude, codex},
		[]provider.Seat{claudeSeat(), codexSeat()}, nil)
	engine.RepairProducer = repairer
	ctx := context.Background()
	for range 2 {
		if err := engine.ProduceNext(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := engine.PhaseReview(ctx); err != nil {
		t.Fatalf("review with repair: %v", err)
	}
	if !repairCalled {
		t.Error("the repair seat was never invoked")
	}
	if engine.State.Project.Workflow.State != "approve" {
		t.Fatalf("state %s, want approve", engine.State.Project.Workflow.State)
	}
}

// TestRepairRejectionPausesForTheUser pins that a repair the controller refuses, here
// an out-of-scope edit, records the blocked capture and pauses instead of stranding
// the workflow in the repair state.
func TestRepairRejectionPausesForTheUser(t *testing.T) {
	root := setupFixtureRepo(t)
	finding := repairFinding()
	claude := reviewer("claude-code", provider.LineageAnthropicClaude, "repair", []Finding{finding})
	codex := reviewer("codex-cli", provider.LineageOpenAIGPT, "repair", []Finding{finding})
	rogueRepairer := &scripted{name: "codex-cli", lineage: provider.LineageOpenAIGPT,
		invoke: func(inv provider.Invocation) (provider.Outcome, error) {
			if err := os.WriteFile(filepath.Join(inv.Dir, "rogue.txt"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
			return provider.Outcome{Text: "Edited rogue.txt."}, nil
		}}
	engine := testEngine(t, root, goodProducer(t),
		[]provider.Adapter{claude, codex},
		[]provider.Seat{claudeSeat(), codexSeat()}, nil)
	engine.RepairProducer = rogueRepairer
	ctx := context.Background()
	for range 2 {
		if err := engine.ProduceNext(ctx); err != nil {
			t.Fatal(err)
		}
	}
	err := engine.PhaseReview(ctx)
	if err == nil || errs.CategoryOf(err) != errs.Policy {
		t.Fatalf("err = %v, want a policy refusal", err)
	}
	if engine.State.Project.Workflow.State != string(Paused) {
		t.Errorf("state = %s, want paused", engine.State.Project.Workflow.State)
	}
	raw, readErr := os.ReadFile(filepath.Join(root, ".peerfoil", "evidence", "cs-0003.md"))
	if readErr != nil {
		t.Fatalf("the repair change set was not persisted: %v", readErr)
	}
	if !strings.Contains(string(raw), "Capture status:** blocked") {
		t.Error("the rejected repair is not recorded as blocked")
	}
	if !strings.Contains(string(raw), "repair_producer") {
		t.Error("the rejected repair does not record its authorized author")
	}
}

// TestPhaseReviewRefusesAMovedIntegrationBranch pins that the review binds to the
// revision recorded at the review boundary, not whatever the branch points at now.
func TestPhaseReviewRefusesAMovedIntegrationBranch(t *testing.T) {
	root := setupFixtureRepo(t)
	reviewers := []provider.Adapter{
		reviewer("claude-code", provider.LineageAnthropicClaude, "approve", nil),
		reviewer("codex-cli", provider.LineageOpenAIGPT, "approve", nil),
	}
	engine := testEngine(t, root, goodProducer(t), reviewers,
		[]provider.Seat{claudeSeat(), codexSeat()}, nil)
	ctx := context.Background()
	for range 2 {
		if err := engine.ProduceNext(ctx); err != nil {
			t.Fatal(err)
		}
	}
	// Someone moves the integration branch after validation froze the revision.
	tip := git(t, root, "rev-parse", "peerfoil/integration")
	tree := git(t, root, "rev-parse", tip+"^{tree}")
	extra := git(t, root, "commit-tree", tree, "-p", tip, "-m", "moved after the boundary")
	git(t, root, "update-ref", "refs/heads/peerfoil/integration", extra)

	err := engine.PhaseReview(ctx)
	if err == nil || errs.CategoryOf(err) != errs.Git {
		t.Fatalf("err = %v, want a git refusal", err)
	}
	if !strings.Contains(err.Error(), "recorded when the phase entered review") {
		t.Errorf("error %q does not name the recorded boundary", err.Error())
	}
}

// TestProductionGateNeedsThePositiveAcceptancePhrase checks that a reduced review
// requires positive, dated acceptance in both its reason and its history reference.
func TestProductionGateNeedsThePositiveAcceptancePhrase(t *testing.T) {
	root := setupFixtureRepo(t)
	// The architecture review is the right target: a plan review has a change-review
	// fallback at the same revision, but the architecture gate has exactly one record.
	architectureReview := filepath.Join(root, ".peerfoil", "reviews", "rv-0001.md")
	rewrite := func(reason string) {
		raw, err := os.ReadFile(architectureReview)
		if err != nil {
			t.Fatal(err)
		}
		pattern := regexp.MustCompile(`(?m)^- \*\*Independence:\*\*.*$`)
		updated := pattern.ReplaceAllString(string(raw), "- **Independence:** reduced — "+reason)
		if err := os.WriteFile(architectureReview, []byte(updated), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	rewrite("the user did not accept Reduced assurance")
	engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
	err := engine.ProduceNext(context.Background())
	if err == nil || !strings.Contains(err.Error(), "approved review") {
		t.Fatalf("err = %v, want the production gate to refuse a recorded refusal", err)
	}

	rewrite("No different-family reviewer was available; the user accepted Reduced assurance at 2026-10-20T14:29:00Z")
	engine = testEngine(t, root, goodProducer(t), nil, nil, nil)
	if err := engine.productionGate(); err == nil {
		t.Fatal("acceptance prose without matching history authorized production")
	}
	engine.State.History[2].Summary = "User accepted Reduced assurance for rv-0001 at 2026-10-20T14:29:00Z; plan revision 0, architecture revision 1, quality revision 1."
	if err := engine.productionGate(); err != nil {
		t.Fatalf("a recorded acceptance was refused: %v", err)
	}
	// Git can check Markdown out with CRLF on Windows; acceptance stays identical.
	raw, err := os.ReadFile(architectureReview)
	if err != nil {
		t.Fatal(err)
	}
	crlf := strings.ReplaceAll(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n", "\r\n")
	if err := os.WriteFile(architectureReview, []byte(crlf), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := engine.productionGate(); err != nil {
		t.Fatalf("CRLF changed the recorded acceptance: %v", err)
	}
	engine.State.History[2].Refs["reviews"] = []string{"rv-9999"}
	if err := engine.productionGate(); err == nil {
		t.Fatal("acceptance of another draft authorized production")
	}
}

// TestEvidenceRunnerFailureIsRecordedAndPauses pins that a check command that cannot
// run still leaves a versioned evidence record and a recoverable paused boundary.
func TestEvidenceRunnerFailureIsRecordedAndPauses(t *testing.T) {
	root := setupFixtureRepo(t)
	engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
	engine.Evidence = func(_ context.Context, _ []string, _ string, _ []string,
		_ time.Duration) ([]byte, []byte, int, bool, error) {
		return nil, []byte("spawn failed"), -1, false, fmt.Errorf("spawn failed")
	}
	err := engine.ProduceNext(context.Background())
	if err == nil {
		t.Fatal("a failed evidence runner reported success")
	}
	if engine.State.Project.Workflow.State != string(Paused) {
		t.Errorf("state = %s, want paused", engine.State.Project.Workflow.State)
	}
	raw, readErr := os.ReadFile(filepath.Join(root, ".peerfoil", "evidence", "ev-0001.md"))
	if readErr != nil {
		t.Fatalf("the failed evidence run left no record: %v", readErr)
	}
	if !strings.Contains(string(raw), "- **Result:** blocked") {
		t.Error("the evidence record does not show the blocked result")
	}
}

// TestChangeSetProseIsScrubbed pins that a credential repeated in the producer's own
// prose never reaches the retained change-set record.
func TestChangeSetProseIsScrubbed(t *testing.T) {
	root := setupFixtureRepo(t)
	secret := "sk-abcdefghijklmnopqrstuv"
	leaky := &scripted{name: "codex-cli", lineage: provider.LineageOpenAIGPT,
		invoke: func(inv provider.Invocation) (provider.Outcome, error) {
			task := packetTask.FindStringSubmatch(inv.Packet)[1]
			base := packetBase.FindStringSubmatch(inv.Packet)[1]
			revisions := packetPlan.FindStringSubmatch(inv.Packet)
			path := filepath.Join(inv.Dir, "main.go")
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(string(original)+"\n// edited\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			handoff := fmt.Sprintf("```json\n{\"task\": %q, \"plan_revision\": %s, \"architecture_revision\": %s, \"base_revision\": %q, \"result\": \"produced\", \"changed_paths\": [\"main.go\"], \"summary\": \"Removed the old key %s from the config.\", \"checks_claimed\": [], \"discovered_work\": [], \"blocker\": null}\n```",
				task, revisions[1], revisions[2], base, secret)
			return provider.Outcome{Text: handoff}, nil
		}}
	engine := testEngine(t, root, leaky, nil, nil, nil)
	if err := engine.ProduceNext(context.Background()); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".peerfoil", "evidence", "cs-0001.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), secret) {
		t.Error("the change-set record retains the credential from the producer's prose")
	}
	if !strings.Contains(string(raw), "[redacted]") {
		t.Error("the change-set summary was not scrubbed")
	}
}
