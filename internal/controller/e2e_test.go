// This file is part of PeerFoil.
// internal/controller/e2e_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Deterministic end-to-end journeys for the enforced vertical slice with fake
// providers: production, capture, evidence, integration, phase review, repair,
// self-approval blocking, and interruption recovery.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/pack"
	"github.com/gabrielmongefranco/peerfoil/internal/project"
	"github.com/gabrielmongefranco/peerfoil/internal/provider"
	"github.com/gabrielmongefranco/peerfoil/internal/schema"
	"github.com/gabrielmongefranco/peerfoil/internal/store"
	"github.com/gabrielmongefranco/peerfoil/internal/workspace"
)

// scripted is a test adapter whose behavior is a function, so a "producer" can really
// edit files in the worktree it was pointed at.
type scripted struct {
	name    string
	lineage string
	invoke  func(inv provider.Invocation) (provider.Outcome, error)
}

func (s *scripted) Name() string { return s.name }
func (s *scripted) LineageRoot(model string) string {
	return s.lineage
}
func (s *scripted) Probe(context.Context) provider.ProbeResult {
	return provider.ProbeResult{Found: true, Version: "test", LoggedIn: true}
}
func (s *scripted) Invoke(_ context.Context, inv provider.Invocation) (provider.Outcome, error) {
	return s.invoke(inv)
}

var (
	packetTask   = regexp.MustCompile(`Task (tk-[0-9]+)`)
	packetBase   = regexp.MustCompile(`base commit ([0-9a-f]{40})`)
	packetPlan   = regexp.MustCompile(`Revisions: plan ([0-9]+), architecture ([0-9]+)`)
	packetFrozen = regexp.MustCompile(`Frozen revision: ([0-9a-f]{40})`)
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// setupFixtureRepo copies the Go fixture into a fresh Git repository.
func setupFixtureRepo(t *testing.T) string {
	t.Helper()
	return setupNamedFixtureRepo(t, "software-go")
}

func setupNamedFixtureRepo(t *testing.T, fixtureName string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(wd, "..", "..", "fixtures", fixtureName)
	root := filepath.Join(t.TempDir(), "repo")
	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(root, relative)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "--initial-branch=main")
	git(t, root, "config", "user.email", "test@example.org")
	git(t, root, "config", "user.name", "Test")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-m", "fixture baseline", "--no-gpg-sign")
	return root
}

func testEngine(t *testing.T, root string, producer provider.Adapter,
	reviewers []provider.Adapter, seats []provider.Seat, runner func(name string) (int, string)) *Engine {
	t.Helper()
	t.Setenv("LOCALAPPDATA", filepath.Join(t.TempDir(), "cache"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(t.TempDir(), "xdg"))
	schemas, err := schema.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	state, err := project.Load(root, schemas)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := store.Open(state.Project.ProjectID, root)
	if err != nil {
		t.Fatal(err)
	}
	loadedPack, err := pack.Load(state.Project.Pack.ID, schemas)
	if err != nil {
		t.Fatal(err)
	}
	evidenceRunner := func(_ context.Context, argv []string, _ string, _ []string,
		_ time.Duration) ([]byte, []byte, int, bool, error) {
		code, output := 0, "ok"
		if runner != nil {
			code, output = runner(strings.Join(argv, " "))
		}
		return []byte(output), nil, code, false, nil
	}
	return NewEngine(Deps{
		State:          state,
		Repo:           workspace.New(root),
		Journal:        journal,
		Pack:           loadedPack,
		Producer:       producer,
		ProducerSeat:   provider.Seat{Tool: "codex-cli", Model: "gpt-6-astra", Effort: "medium"},
		Reviewers:      reviewers,
		ReviewerSeats:  seats,
		Evidence:       evidenceRunner,
		WorktreeParent: t.TempDir(),
		Out:            os.Stderr,
	})
}

// goodProducer edits exactly the allowed files and returns a valid handoff.
func goodProducer(t *testing.T) *scripted {
	return &scripted{name: "codex-cli", lineage: provider.LineageOpenAIGPT,
		invoke: func(inv provider.Invocation) (provider.Outcome, error) {
			if strings.Contains(inv.Packet, "repair producer") {
				path := filepath.Join(inv.Dir, "main.go")
				original, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				// The checks are faked in these journeys, so the repair only needs
				// to be a visible, in-scope edit.
				repaired := string(original) + "\n// repaired: the missing-file error is now handled\n"
				if err := os.WriteFile(path, []byte(repaired), 0o644); err != nil {
					t.Fatal(err)
				}
				return provider.Outcome{Text: "Handled the missing-file error in main.go.", SessionID: "thread-repair"}, nil
			}
			task := packetTask.FindStringSubmatch(inv.Packet)[1]
			base := packetBase.FindStringSubmatch(inv.Packet)[1]
			revisions := packetPlan.FindStringSubmatch(inv.Packet)
			path := filepath.Join(inv.Dir, "main.go")
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			edited := string(original) + "\n// " + task + " marker\n"
			if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
				t.Fatal(err)
			}
			handoff := fmt.Sprintf("Done.\n```json\n{\"task\": %q, \"plan_revision\": %s, \"architecture_revision\": %s, \"base_revision\": %q, \"result\": \"produced\", \"changed_paths\": [\"main.go\"], \"summary\": \"Edited main.go for %s.\", \"checks_claimed\": [], \"discovered_work\": [], \"blocker\": null}\n```\n",
				task, revisions[1], revisions[2], base, task)
			return provider.Outcome{Text: handoff, SessionID: "thread-" + task}, nil
		}}
}

// reviewer returns a scripted reviewer that reads the frozen revision out of the
// packet and answers with the given decision and findings.
func reviewer(name, lineage, decision string, findings []Finding) *scripted {
	return &scripted{name: name, lineage: lineage,
		invoke: func(inv provider.Invocation) (provider.Outcome, error) {
			frozen := packetFrozen.FindStringSubmatch(inv.Packet)[1]
			revisions := packetPlan.FindStringSubmatch(inv.Packet)
			kind := "phase"
			if strings.Contains(inv.Packet, "verification of a repair") {
				kind = "repair"
			}
			answer := map[string]any{
				"kind": kind,
				"reviewed": map[string]any{
					"source_revision":       frozen,
					"plan_revision":         atoi(revisions[1]),
					"architecture_revision": atoi(revisions[2]),
				},
				"model":          "scripted",
				"decision":       decision,
				"findings":       findings,
				"remaining_risk": "None beyond the fixture's known seeds.",
			}
			raw, _ := json.Marshal(answer)
			return provider.Outcome{Text: "```json\n" + string(raw) + "\n```",
				SessionID: name + "-session"}, nil
		}}
}

func atoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func claudeSeat() provider.Seat {
	return provider.Seat{Tool: "claude-code", Model: "claude-fable-5", Effort: "medium"}
}
func codexSeat() provider.Seat {
	return provider.Seat{Tool: "codex-cli", Model: "gpt-6-astra", Effort: "medium"}
}

func TestVerticalSliceToApproval(t *testing.T) {
	root := setupFixtureRepo(t)
	userHead := git(t, root, "rev-parse", "HEAD")
	reviewers := []provider.Adapter{
		reviewer("claude-code", provider.LineageAnthropicClaude, "approve", nil),
		reviewer("codex-cli", provider.LineageOpenAIGPT, "approve", nil),
	}
	seats := []provider.Seat{claudeSeat(), codexSeat()}
	engine := testEngine(t, root, goodProducer(t), reviewers, seats, nil)
	ctx := context.Background()

	// Task 1 and task 2, one bounded production each; the second run ends at the
	// review boundary because no eligible task remains.
	if err := engine.ProduceNext(ctx); err != nil {
		t.Fatalf("task 1: %v", err)
	}
	if err := engine.ProduceNext(ctx); err != nil {
		t.Fatalf("task 2: %v", err)
	}
	if engine.State.Project.Workflow.State != "review" {
		t.Fatalf("state %s, want review", engine.State.Project.Workflow.State)
	}
	// The user's checkout never moved.
	if head := git(t, root, "rev-parse", "HEAD"); head != userHead {
		t.Error("the user's HEAD changed")
	}
	// The integration branch carries both task commits on top of the base.
	integration := git(t, root, "rev-parse", "peerfoil/integration")
	if !strings.Contains(git(t, root, "log", "--oneline", integration), "tk-002") {
		t.Error("the integration branch does not carry task 2")
	}

	// Phase review: both families approve, so the phase closes.
	if err := engine.PhaseReview(ctx); err != nil {
		t.Fatalf("review: %v", err)
	}
	if engine.State.Project.Workflow.State != "approve" {
		t.Fatalf("state %s, want approve", engine.State.Project.Workflow.State)
	}

	// Everything written must still load through full validation in a fresh session.
	schemas, _ := schema.LoadEmbedded()
	reloaded, err := project.Load(root, schemas)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	for _, task := range reloaded.Plan.Tasks() {
		if task.Status != "accepted" {
			t.Errorf("%s status %s, want accepted", task.ID, task.Status)
		}
	}
	for _, name := range []string{"cs-0001.md", "cs-0001.patch", "ev-0001.md",
		filepath.Join("..", "reviews", "rv-0004.md"), filepath.Join("..", "reviews", "pr-0001.md")} {
		path := filepath.Join(root, ".peerfoil", "evidence", name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing record %s", name)
		}
	}
}

func TestOutOfScopeChangeBlocks(t *testing.T) {
	root := setupFixtureRepo(t)
	rogue := &scripted{name: "codex-cli", lineage: provider.LineageOpenAIGPT,
		invoke: func(inv provider.Invocation) (provider.Outcome, error) {
			task := packetTask.FindStringSubmatch(inv.Packet)[1]
			base := packetBase.FindStringSubmatch(inv.Packet)[1]
			revisions := packetPlan.FindStringSubmatch(inv.Packet)
			// Outside every allowed path for the task.
			if err := os.WriteFile(filepath.Join(inv.Dir, "rogue.txt"), []byte("x"), 0o644); err != nil {
				return provider.Outcome{}, err
			}
			handoff := fmt.Sprintf("```json\n{\"task\": %q, \"plan_revision\": %s, \"architecture_revision\": %s, \"base_revision\": %q, \"result\": \"produced\", \"changed_paths\": [\"rogue.txt\"], \"summary\": \"x\", \"checks_claimed\": [], \"discovered_work\": [], \"blocker\": null}\n```",
				task, revisions[1], revisions[2], base)
			return provider.Outcome{Text: handoff}, nil
		}}
	engine := testEngine(t, root, rogue, nil, nil, nil)
	err := engine.ProduceNext(context.Background())
	if err == nil || errs.CategoryOf(err) != errs.Policy {
		t.Fatalf("err = %v, want a policy error", err)
	}
	if engine.State.Project.Workflow.State != "paused" {
		t.Errorf("state %s, want paused", engine.State.Project.Workflow.State)
	}
}

func TestSecretInChangeBlocks(t *testing.T) {
	root := setupFixtureRepo(t)
	leaky := &scripted{name: "codex-cli", lineage: provider.LineageOpenAIGPT,
		invoke: func(inv provider.Invocation) (provider.Outcome, error) {
			task := packetTask.FindStringSubmatch(inv.Packet)[1]
			base := packetBase.FindStringSubmatch(inv.Packet)[1]
			revisions := packetPlan.FindStringSubmatch(inv.Packet)
			content := "package main\n\n// synthetic test credential\nconst key = \"AKIAIOSFODNN7EXAMPLE\"\n"
			if err := os.WriteFile(filepath.Join(inv.Dir, "main.go"), []byte(content), 0o644); err != nil {
				return provider.Outcome{}, err
			}
			handoff := fmt.Sprintf("```json\n{\"task\": %q, \"plan_revision\": %s, \"architecture_revision\": %s, \"base_revision\": %q, \"result\": \"produced\", \"changed_paths\": [\"main.go\"], \"summary\": \"x\", \"checks_claimed\": [], \"discovered_work\": [], \"blocker\": null}\n```",
				task, revisions[1], revisions[2], base)
			return provider.Outcome{Text: handoff}, nil
		}}
	engine := testEngine(t, root, leaky, nil, nil, nil)
	err := engine.ProduceNext(context.Background())
	if err == nil || !strings.Contains(err.Error(), "credential") {
		t.Fatalf("err = %v, want a credential block", err)
	}
}

func TestRequiredCheckFailureReturnsTaskToProduction(t *testing.T) {
	root := setupFixtureRepo(t)
	engine := testEngine(t, root, goodProducer(t), nil, nil, func(command string) (int, string) {
		if strings.Contains(command, "go test") {
			return 1, "FAIL"
		}
		return 0, "ok"
	})
	err := engine.ProduceNext(context.Background())
	if err == nil || errs.CategoryOf(err) != errs.Evidence {
		t.Fatalf("err = %v, want an evidence error", err)
	}
	if engine.State.Project.Workflow.State != "produce" {
		t.Errorf("state %s, want produce (the task went back)", engine.State.Project.Workflow.State)
	}
	// The failing work must not be integrated.
	cmd := exec.Command("git", "rev-parse", "--verify", "peerfoil/integration")
	cmd.Dir = root
	if cmd.Run() == nil {
		t.Error("a failing change reached the integration branch")
	}
}

func TestMalformedHandoffCannotSucceed(t *testing.T) {
	root := setupFixtureRepo(t)
	malformed := &scripted{name: "codex-cli", lineage: provider.LineageOpenAIGPT,
		invoke: func(inv provider.Invocation) (provider.Outcome, error) {
			if err := os.WriteFile(filepath.Join(inv.Dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
				return provider.Outcome{}, err
			}
			return provider.Outcome{Text: "I did things. Everything passed."}, nil
		}}
	engine := testEngine(t, root, malformed, nil, nil, nil)
	err := engine.ProduceNext(context.Background())
	if err == nil || errs.CategoryOf(err) != errs.Provider {
		t.Fatalf("err = %v, want a provider error", err)
	}
}

func TestReviewerDisagreementGoesToTheUser(t *testing.T) {
	root := setupFixtureRepo(t)
	reviewers := []provider.Adapter{
		reviewer("claude-code", provider.LineageAnthropicClaude, "repair", []Finding{{
			Title: "Missing file handling", Location: "main.go", Requirement: "r-0002",
			Severity: "blocking", Evidence: "the error is ignored", Recommendation: "handle it",
		}}),
		reviewer("codex-cli", provider.LineageOpenAIGPT, "approve", nil),
	}
	seats := []provider.Seat{claudeSeat(), codexSeat()}
	engine := testEngine(t, root, goodProducer(t), reviewers, seats, nil)
	ctx := context.Background()
	for range 2 {
		if err := engine.ProduceNext(ctx); err != nil {
			t.Fatal(err)
		}
	}
	err := engine.PhaseReview(ctx)
	if err == nil || errs.CategoryOf(err) != errs.Policy {
		t.Fatalf("err = %v, want a policy stop", err)
	}
	if engine.State.Project.Workflow.State != "paused" {
		t.Errorf("state %s, want paused", engine.State.Project.Workflow.State)
	}
}

func TestSameFamilyApprovalIsBlocked(t *testing.T) {
	root := setupFixtureRepo(t)
	// Both reviewer seats resolve to the producer's family: approval must not happen.
	reviewers := []provider.Adapter{
		reviewer("codex-cli", provider.LineageOpenAIGPT, "approve", nil),
		reviewer("codex-cli", provider.LineageOpenAIGPT, "approve", nil),
	}
	seats := []provider.Seat{codexSeat(), codexSeat()}
	engine := testEngine(t, root, goodProducer(t), reviewers, seats, nil)
	ctx := context.Background()
	for range 2 {
		if err := engine.ProduceNext(ctx); err != nil {
			t.Fatal(err)
		}
	}
	err := engine.PhaseReview(ctx)
	if err == nil || errs.CategoryOf(err) != errs.Configuration {
		t.Fatalf("err = %v, want the two-family refusal", err)
	}
	if engine.State.Project.Workflow.State != "review" {
		t.Errorf("state %s, want review (nothing approved)", engine.State.Project.Workflow.State)
	}
}

func TestRepairCycleWithFreshVerification(t *testing.T) {
	root := setupFixtureRepo(t)
	finding := Finding{
		Title: "Missing-file error handling", Location: "main.go", Requirement: "r-0002",
		Severity: "blocking", Evidence: "os.ReadFile's error is discarded",
		Recommendation: "print a readable error and exit 1",
	}
	// Both phase reviewers ask for the repair; the claude seat later verifies it.
	claude := reviewer("claude-code", provider.LineageAnthropicClaude, "repair", []Finding{finding})
	codex := reviewer("codex-cli", provider.LineageOpenAIGPT, "repair", []Finding{finding})
	// The verification pass must approve: the scripted reviewer approves repair kind.
	claude.invoke = wrapKindAware(claude.invoke, "approve")
	producer := goodProducer(t)
	engine := testEngine(t, root, producer,
		[]provider.Adapter{claude, codex},
		[]provider.Seat{claudeSeat(), codexSeat()}, nil)
	ctx := context.Background()
	for range 2 {
		if err := engine.ProduceNext(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if err := engine.PhaseReview(ctx); err != nil {
		t.Fatalf("review with repair: %v", err)
	}
	if engine.State.Project.Workflow.State != "approve" {
		t.Fatalf("state %s, want approve after the verified repair", engine.State.Project.Workflow.State)
	}
	// The repair is a third change set with the repair producer as its author.
	raw, err := os.ReadFile(filepath.Join(root, ".peerfoil", "evidence", "cs-0003.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "repair_producer") {
		t.Error("the repair change set does not record the repair producer")
	}
}

// wrapKindAware answers repair-verification packets with the given decision while
// keeping the original behavior for the phase review.
func wrapKindAware(original func(provider.Invocation) (provider.Outcome, error), repairDecision string) func(provider.Invocation) (provider.Outcome, error) {
	return func(inv provider.Invocation) (provider.Outcome, error) {
		if strings.Contains(inv.Packet, "verification of a repair") {
			return reviewer("claude-code", provider.LineageAnthropicClaude, repairDecision, nil).invoke(inv)
		}
		return original(inv)
	}
}

func TestInterruptedInvocationNeedsConfirmation(t *testing.T) {
	root := setupFixtureRepo(t)
	engine := testEngine(t, root, goodProducer(t), nil, nil, nil)
	// Simulate a crash: an intent with no result.
	if _, err := engine.Journal.Intent("producer", "cs-0001", map[string]string{"task": "tk-001"}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	err := engine.Resume(ctx, false)
	if err == nil || !strings.Contains(err.Error(), "interrupted") {
		t.Fatalf("err = %v, want an interruption stop", err)
	}
	// Production must also refuse to start something new.
	if err := engine.ProduceNext(ctx); err == nil || !strings.Contains(err.Error(), "no recorded result") {
		t.Fatalf("ProduceNext = %v, want the open-invocation refusal", err)
	}
	// With the user's confirmation the attempt closes as failed and the run pauses.
	err = engine.Resume(ctx, true)
	if err == nil || errs.CategoryOf(err) != errs.Policy {
		t.Fatalf("confirmed resume = %v, want a policy pause", err)
	}
	open, err := engine.Journal.OpenInvocations()
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 0 {
		t.Errorf("open invocations remain: %+v", open)
	}
}

// TestDocumentationFixtureJourney proves the controller is not hard-coded to source
// code: the documentation fixture loads, produces, and captures through the same code,
// and its required inspection and human evidence pauses honestly for a person instead
// of being fabricated.
func TestDocumentationFixtureJourney(t *testing.T) {
	root := setupNamedFixtureRepo(t, "documentation")
	writer := &scripted{name: "codex-cli", lineage: provider.LineageOpenAIGPT,
		invoke: func(inv provider.Invocation) (provider.Outcome, error) {
			task := packetTask.FindStringSubmatch(inv.Packet)[1]
			base := packetBase.FindStringSubmatch(inv.Packet)[1]
			revisions := packetPlan.FindStringSubmatch(inv.Packet)
			path := filepath.Join(inv.Dir, "guide.md")
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			edited := string(original) + "\n## Verify the backup\n\nOpen the copy with sqlite3 and run PRAGMA integrity_check.\n"
			if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
				t.Fatal(err)
			}
			handoff := fmt.Sprintf("```json\n{\"task\": %q, \"plan_revision\": %s, \"architecture_revision\": %s, \"base_revision\": %q, \"result\": \"produced\", \"changed_paths\": [\"guide.md\"], \"summary\": \"Added the verification section.\", \"checks_claimed\": [], \"discovered_work\": [], \"blocker\": null}\n```",
				task, revisions[1], revisions[2], base)
			return provider.Outcome{Text: handoff, SessionID: "thread-doc"}, nil
		}}
	engine := testEngine(t, root, writer, nil, nil, nil)
	err := engine.ProduceNext(context.Background())
	if err == nil || errs.CategoryOf(err) != errs.Evidence {
		t.Fatalf("err = %v, want the honest evidence stop", err)
	}
	if !strings.Contains(err.Error(), "human or inspection") {
		t.Errorf("error %q does not name the human/inspection gap", err.Error())
	}
	// The change was captured with its provenance before the stop.
	raw, readErr := os.ReadFile(filepath.Join(root, ".peerfoil", "evidence", "cs-0001.md"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(raw), "Capture status:** captured") {
		t.Error("the documentation change was not captured")
	}
	// Nothing was integrated without its required evidence.
	cmd := exec.Command("git", "rev-parse", "--verify", "peerfoil/integration")
	cmd.Dir = root
	if cmd.Run() == nil {
		t.Error("work without required evidence reached the integration branch")
	}
}
