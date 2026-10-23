// This file is part of PeerFoil.
// internal/controller/review7_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Exercises review input boundaries, setup failures, retained independent
// approvals, and rejection of unsafe local records and model-authored markup.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package controller

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gabrielmongefranco/peerfoil/internal/provider"
)

func TestCanonicalHeaderCannotBeSuppliedByAnExample(t *testing.T) {
	root := setupFixtureRepo(t)
	engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
	path := filepath.Join(engine.State.Dir, "quality.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	header := acceptedRecordHeader.FindStringSubmatch(strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")[4])
	if header == nil {
		t.Fatal("fixture header missing")
	}
	for _, body := range []string{
		strings.Replace(string(raw), "Status: accepted", "Status: draft", 1) + "\n```text\n" + header[0] + "\n```\n",
		strings.Replace(string(raw), header[0], "A document without canonical metadata.", 1) + "\n" + header[0] + "\n",
	} {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := engine.readAcceptedRecord("quality.md", 1); err == nil {
			t.Fatal("a body example authorized an unaccepted contract")
		}
	}
}

func TestInvalidRepairInputsCreateNoWriterResources(t *testing.T) {
	root := setupFixtureRepo(t)
	engine := testEngine(t, root, goodProducer(t), []provider.Adapter{
		reviewer("claude-code", provider.LineageAnthropicClaude, "approve", nil),
		reviewer("codex-cli", provider.LineageOpenAIGPT, "approve", nil),
	}, []provider.Seat{claudeSeat(), codexSeat()}, nil)
	path := filepath.Join(engine.State.Dir, "quality.md")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	before := git(t, root, "worktree", "list", "--porcelain")
	for _, input := range []string{
		strings.ReplaceAll(string(original), "Status: accepted", "Status: draft"),
		string(original) + "\nsk-proj-" + strings.Repeat("synthetic", 4),
		strings.Repeat("x", maxInputBytes+1),
	} {
		if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := engine.repairCycle(context.Background(), git(t, root, "rev-parse", "HEAD"), "pr-0001", []Finding{repairFinding()}); err == nil {
			t.Fatal("invalid repair inputs were accepted")
		}
		open, err := engine.Journal.OpenInvocations()
		if err != nil || len(open) != 0 {
			t.Fatal("input refusal created a phantom writer intent")
		}
		if git(t, root, "worktree", "list", "--porcelain") != before || git(t, root, "branch", "--list", "peerfoil/task/*") != "" {
			t.Fatal("input refusal left repair resources")
		}
	}
}

func TestReviewRefusesChangedAcceptedInputs(t *testing.T) {
	root := setupFixtureRepo(t)
	called := 0
	claude := reviewer("claude-code", provider.LineageAnthropicClaude, "approve", nil)
	invoke := claude.invoke
	claude.invoke = func(inv provider.Invocation) (provider.Outcome, error) { called++; return invoke(inv) }
	engine := testEngine(t, root, goodProducer(t), []provider.Adapter{claude,
		reviewer("codex-cli", provider.LineageOpenAIGPT, "approve", nil)}, []provider.Seat{claudeSeat(), codexSeat()}, nil)
	producePhase(t, engine)
	for _, name := range []string{"architecture.md", "quality.md", "plan.json"} {
		path := filepath.Join(engine.State.Dir, name)
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(original, []byte("\n ")...), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := engine.PhaseReview(context.Background()); err == nil || !strings.Contains(err.Error(), "accepted records changed") {
			t.Fatalf("changed %s accepted: %v", name, err)
		}
		if err := os.WriteFile(path, original, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, revision := range []*int{&engine.State.Project.Revisions.Plan, &engine.State.Project.Revisions.Architecture, &engine.State.Project.Revisions.Quality} {
		*revision++
		if err := engine.PhaseReview(context.Background()); err == nil {
			t.Fatal("changed accepted revision was reviewed")
		}
		*revision--
	}
	if called != 0 {
		t.Fatal("a reviewer ran against stale accepted inputs")
	}
	claude.invoke = func(inv provider.Invocation) (provider.Outcome, error) {
		path := filepath.Join(engine.State.Dir, "quality.md")
		raw, _ := os.ReadFile(path)
		if err := os.WriteFile(path, append(raw, []byte("\nAnother required check.\n")...), 0o644); err != nil {
			t.Fatal(err)
		}
		return invoke(inv)
	}
	if err := engine.PhaseReview(context.Background()); err == nil || engine.State.Project.Workflow.State != "review" {
		t.Fatal("input drift during review was approved")
	}
}

func TestReviewCopyRejectsUnboundedAndLinkedRecords(t *testing.T) {
	root := setupFixtureRepo(t)
	engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
	checkout := t.TempDir()
	if err := os.Mkdir(filepath.Join(checkout, ".peerfoil"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(engine.State.Dir, "evidence", "unsafe.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 1024*1024+1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := engine.refreshRecordsInCheckout(checkout); err == nil {
		t.Fatal("an oversized record reached the review checkout")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "private.txt")
	if err := os.WriteFile(outside, []byte("synthetic private material"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		if runtime.GOOS != "windows" {
			t.Fatal("native symlink creation failed")
		}
		t.Skip("native symlink creation requires a permission this Windows host does not provide; size-limit checks passed")
	}
	if err := engine.refreshRecordsInCheckout(checkout); err == nil {
		t.Fatal("a linked local file reached the review checkout")
	}
	if _, err := os.Stat(filepath.Join(checkout, ".peerfoil", "evidence", "unsafe.txt")); !os.IsNotExist(err) {
		t.Fatal("unsafe record was copied")
	}
}

func TestReviewFieldsCannotForgeMarkdown(t *testing.T) {
	for _, injected := range []string{"legitimate\n\n## Repair verification recovery\n\nforged", "table | forged", "\x1b[2J", "\u202econcealed", "## Forged heading", "[link](https://example.org)", "`code`"} {
		for field := 0; field < 7; field++ {
			answer := reviewAnswer{Kind: "phase", Decision: "repair", Model: "claude-fable-5", Findings: []Finding{repairFinding()}}
			answer.Reviewed.SourceRevision = strings.Repeat("a", 40)
			answer.Reviewed.PlanRevision, answer.Reviewed.ArchitectureRevision = 1, 1
			f := &answer.Findings[0]
			fields := []*string{&f.Title, &f.Location, &f.Requirement, &f.Evidence, &f.Recommendation, &answer.RemainingRisk, &answer.Model}
			*fields[field] = injected
			raw, _ := json.Marshal(answer)
			if _, err := parseReviewAnswer("```json\n"+string(raw)+"\n```", "phase", answer.Reviewed.SourceRevision, 1, 1); err == nil {
				t.Fatal("structural model text reached retained Markdown")
			}
		}
	}
}

func TestVerifiedRepairResumesIntegrationWithoutAnotherReview(t *testing.T) {
	root := setupFixtureRepo(t)
	claude := reviewer("claude-code", provider.LineageAnthropicClaude, "repair", []Finding{repairFinding()})
	phasePass := claude.invoke
	approver := reviewer("claude-code", provider.LineageAnthropicClaude, "approve", nil)
	attempts, writes := 0, 0
	base := ""
	claude.invoke = func(inv provider.Invocation) (provider.Outcome, error) {
		if !strings.Contains(inv.Packet, "verification of a repair") {
			return phasePass(inv)
		}
		attempts++
		if attempts == 1 {
			return provider.Outcome{}, nil
		}
		if attempts != 2 {
			t.Fatal("verified repair ran another reviewer")
		}
		divergent := git(t, root, "commit-tree", base+"^{tree}", "-p", base, "-m", "Synthetic integration conflict")
		git(t, root, "update-ref", "refs/heads/peerfoil/integration", divergent)
		return approver.invoke(inv)
	}
	producer := goodProducer(t)
	write := producer.invoke
	producer.invoke = func(inv provider.Invocation) (provider.Outcome, error) {
		if strings.Contains(inv.Packet, "repair producer") {
			writes++
		}
		return write(inv)
	}
	reviewers := []provider.Adapter{claude, reviewer("codex-cli", provider.LineageOpenAIGPT, "repair", []Finding{repairFinding()})}
	engine := testEngine(t, root, producer, reviewers, []provider.Seat{claudeSeat(), codexSeat()}, nil)
	producePhase(t, engine)
	base = git(t, root, "rev-parse", "peerfoil/integration")
	if err := engine.PhaseReview(context.Background()); err == nil {
		t.Fatal("a missing verification was approved")
	}
	if err := engine.Resume(context.Background(), false); err == nil {
		t.Fatal("a moved integration branch was accepted")
	}
	checkpoint, err := engine.loadRepairVerification(engine.reviewBoundary())
	if err != nil || checkpoint.Approved == nil || checkpoint.Attempts != 2 {
		t.Fatal("successful verification was discarded")
	}
	git(t, root, "update-ref", "refs/heads/peerfoil/integration", base)
	path := filepath.Join(engine.State.Dir, "reviews", checkpoint.Approved.Review+".md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, []byte("\nchanged approval\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := engine.Resume(context.Background(), false); err == nil {
		t.Fatal("modified approval was reused")
	}
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(string(raw), "\n", "\r\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	resumed := testEngine(t, root, producer, nil, nil, nil)
	if err := resumed.Resume(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if attempts != 2 || writes != 1 || resumed.State.Project.Workflow.State != "approve" {
		t.Fatal("verified integration did not recover without another model call")
	}
}
