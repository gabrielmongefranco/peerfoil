// This file is part of PeerFoil.
// internal/evidence/evidence_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Tests for controller-run evidence: command validation rejects injection and
// escapes, Run records every outcome and retains scrubbed capped output, and freshness
// checks refuse stale or missing required evidence. All secrets are synthetic.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package evidence

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

// runnerCall captures what Run handed to the injected runner, so tests can assert the
// allowlisted environment and resolved directory were passed through unchanged.
type runnerCall struct {
	argv    []string
	dir     string
	env     []string
	timeout time.Duration
}

// fakeRunner returns a Runner with a fixed result and records each call.
func fakeRunner(stdout, stderr string, exitCode int, timedOut bool, err error, calls *[]runnerCall) Runner {
	return func(_ context.Context, argv []string, dir string, env []string, timeout time.Duration) ([]byte, []byte, int, bool, error) {
		if calls != nil {
			*calls = append(*calls, runnerCall{argv: argv, dir: dir, env: env, timeout: timeout})
		}
		return []byte(stdout), []byte(stderr), exitCode, timedOut, err
	}
}

// fixedClock returns a clock that starts at start and advances by step per call, so
// duration and timestamp fields are deterministic.
func fixedClock(start time.Time, step time.Duration) func() time.Time {
	current := start
	return func() time.Time {
		now := current
		current = current.Add(step)
		return now
	}
}

func testMeta(repoRoot string) Meta {
	return Meta{
		ID:                   "ev-001",
		TaskID:               "T-001",
		PlanRevision:         3,
		ArchitectureRevision: 2,
		SourceRevision:       "rev-abc123",
		ToolVersion:          "go1.22.0",
		RepoRoot:             repoRoot,
		Now:                  fixedClock(time.Date(2026, 10, 21, 14, 3, 3, 0, time.UTC), 2*time.Second),
	}
}

func testCommand() Command {
	return Command{
		Name:           "go-test",
		Argv:           []string{"go", "test", "./..."},
		Env:            []string{"GOCACHE=cache", "GOFLAGS=-count=1"},
		TimeoutSeconds: 60,
	}
}

func testRequirement() Requirement {
	return Requirement{Name: "go-test", Kind: KindExecutable, Level: LevelRequired}
}

func TestRunPass(t *testing.T) {
	t.Parallel()
	repoRoot := t.TempDir()
	retainDir := t.TempDir()
	var calls []runnerCall
	runner := fakeRunner("ok\n", "", 0, false, nil, &calls)

	record, err := Run(context.Background(), runner, testCommand(), testRequirement(), testMeta(repoRoot), retainDir)
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	if record.Status != StatusPass {
		t.Errorf("Status = %q, want %q", record.Status, StatusPass)
	}
	if record.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", record.ExitCode)
	}
	if record.ID != "ev-001" || record.Name != "go-test" || record.TaskID != "T-001" {
		t.Errorf("identity fields = %q %q %q", record.ID, record.Name, record.TaskID)
	}
	if record.Kind != KindExecutable || record.Level != LevelRequired {
		t.Errorf("Kind/Level = %q/%q", record.Kind, record.Level)
	}
	if record.PlanRevision != 3 || record.ArchitectureRevision != 2 || record.SourceRevision != "rev-abc123" {
		t.Errorf("revisions = %d %d %q", record.PlanRevision, record.ArchitectureRevision, record.SourceRevision)
	}
	if record.ToolVersion != "go1.22.0" {
		t.Errorf("ToolVersion = %q", record.ToolVersion)
	}
	if record.DurationSeconds != 2 {
		t.Errorf("DurationSeconds = %v, want 2", record.DurationSeconds)
	}
	if record.At != "2026-10-21T14:03:05Z" {
		t.Errorf("At = %q, want 2026-10-21T14:03:05Z", record.At)
	}
	if record.Redacted {
		t.Errorf("Redacted = true for plain output")
	}

	// The runner received the resolved directory, the exact allowlisted environment,
	// and the declared timeout.
	if len(calls) != 1 {
		t.Fatalf("runner called %d times, want 1", len(calls))
	}
	wantDir, _ := filepath.Abs(repoRoot)
	if calls[0].dir != wantDir {
		t.Errorf("runner dir = %q, want %q", calls[0].dir, wantDir)
	}
	wantEnv := []string{"GOCACHE=cache", "GOFLAGS=-count=1"}
	if len(calls[0].env) != len(wantEnv) || calls[0].env[0] != wantEnv[0] || calls[0].env[1] != wantEnv[1] {
		t.Errorf("runner env = %v, want %v", calls[0].env, wantEnv)
	}
	if calls[0].timeout != 60*time.Second {
		t.Errorf("runner timeout = %v, want 60s", calls[0].timeout)
	}
	if len(calls[0].argv) != 3 || calls[0].argv[0] != "go" {
		t.Errorf("runner argv = %v", calls[0].argv)
	}

	// The retained file holds the labeled output and the hash matches its bytes.
	wantPath := filepath.Join(retainDir, "ev-001.txt")
	if record.RetainedPath != wantPath {
		t.Errorf("RetainedPath = %q, want %q", record.RetainedPath, wantPath)
	}
	data, readErr := os.ReadFile(wantPath)
	if readErr != nil {
		t.Fatalf("reading retained file: %v", readErr)
	}
	if !strings.HasPrefix(string(data), "--- stdout ---\nok\n--- stderr ---\n") {
		t.Errorf("retained content = %q", data)
	}
	sum := sha256.Sum256(data)
	if record.RetainedSHA256 != hex.EncodeToString(sum[:]) {
		t.Errorf("RetainedSHA256 does not match the retained file bytes")
	}
}

func TestRunResolvesSubdirectory(t *testing.T) {
	t.Parallel()
	repoRoot := t.TempDir()
	var calls []runnerCall
	cmd := testCommand()
	cmd.Dir = "sub"

	record, err := Run(context.Background(), fakeRunner("", "", 0, false, nil, &calls),
		cmd, testRequirement(), testMeta(repoRoot), t.TempDir())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	wantDir := filepath.Join(repoRoot, "sub")
	if calls[0].dir != wantDir {
		t.Errorf("runner dir = %q, want %q", calls[0].dir, wantDir)
	}
	// The record keeps the declared relative directory, never an absolute path that
	// would name where the project lives.
	if record.Dir != "sub" {
		t.Errorf("record Dir = %q, want %q", record.Dir, "sub")
	}
}

func TestRunFail(t *testing.T) {
	t.Parallel()
	record, err := Run(context.Background(), fakeRunner("", "assert failed\n", 2, false, nil, nil),
		testCommand(), testRequirement(), testMeta(t.TempDir()), t.TempDir())
	if err != nil {
		t.Fatalf("Run() error = %v, want nil for a clean nonzero exit", err)
	}
	if record.Status != StatusFail || record.ExitCode != 2 {
		t.Errorf("Status/ExitCode = %q/%d, want fail/2", record.Status, record.ExitCode)
	}
	data, readErr := os.ReadFile(record.RetainedPath)
	if readErr != nil {
		t.Fatalf("reading retained file: %v", readErr)
	}
	if !strings.Contains(string(data), "assert failed") {
		t.Errorf("retained output lost stderr: %q", data)
	}
}

func TestRunTimeout(t *testing.T) {
	t.Parallel()
	record, err := Run(context.Background(), fakeRunner("partial", "", -1, true, context.DeadlineExceeded, nil),
		testCommand(), testRequirement(), testMeta(t.TempDir()), t.TempDir())
	if err != nil {
		t.Fatalf("Run() error = %v, want nil for a recorded timeout", err)
	}
	if record.Status != StatusTimeout {
		t.Errorf("Status = %q, want %q", record.Status, StatusTimeout)
	}
	if record.RetainedPath == "" {
		t.Errorf("timeout output was not retained")
	}
}

func TestRunRunnerError(t *testing.T) {
	t.Parallel()
	record, err := Run(context.Background(), fakeRunner("", "", -1, false, errors.New("program not found"), nil),
		testCommand(), testRequirement(), testMeta(t.TempDir()), t.TempDir())
	if err == nil {
		t.Fatalf("Run() error = nil, want an error when the command could not run")
	}
	if errs.CategoryOf(err) != errs.Evidence {
		t.Errorf("category = %q, want %q", errs.CategoryOf(err), errs.Evidence)
	}
	if record.Status != StatusError {
		t.Errorf("Status = %q, want %q", record.Status, StatusError)
	}
	if record.RetainedPath == "" || record.RetainedSHA256 == "" {
		t.Errorf("error record did not retain output where possible")
	}
}

func TestRunCancelled(t *testing.T) {
	t.Parallel()
	_, err := Run(context.Background(), fakeRunner("", "", -1, false, context.Canceled, nil),
		testCommand(), testRequirement(), testMeta(t.TempDir()), t.TempDir())
	if errs.CategoryOf(err) != errs.Cancelled {
		t.Errorf("category = %q, want %q", errs.CategoryOf(err), errs.Cancelled)
	}
}

func TestRunScrubsPlantedToken(t *testing.T) {
	t.Parallel()
	// A synthetic token shaped like a real GitHub credential.
	planted := "ghp_" + strings.Repeat("Zz9", 12)
	runner := fakeRunner("pushing with "+planted+" done\n", "", 0, false, nil, nil)

	record, err := Run(context.Background(), runner, testCommand(), testRequirement(), testMeta(t.TempDir()), t.TempDir())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !record.Redacted {
		t.Errorf("Redacted = false, want true when a credential was removed")
	}
	data, readErr := os.ReadFile(record.RetainedPath)
	if readErr != nil {
		t.Fatalf("reading retained file: %v", readErr)
	}
	if bytes.Contains(data, []byte(planted)) {
		t.Errorf("retained file still contains the planted token")
	}
	if !bytes.Contains(data, []byte("[redacted]")) {
		t.Errorf("retained file has no redaction placeholder: %q", data)
	}
	sum := sha256.Sum256(data)
	if record.RetainedSHA256 != hex.EncodeToString(sum[:]) {
		t.Errorf("RetainedSHA256 does not match the retained file bytes")
	}
}

func TestRunCapsRetainedOutput(t *testing.T) {
	t.Parallel()
	head := strings.Repeat("h", 200*1024)
	tail := strings.Repeat("t", 200*1024)
	runner := fakeRunner(head+tail, "", 0, false, nil, nil)

	record, err := Run(context.Background(), runner, testCommand(), testRequirement(), testMeta(t.TempDir()), t.TempDir())
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	data, readErr := os.ReadFile(record.RetainedPath)
	if readErr != nil {
		t.Fatalf("reading retained file: %v", readErr)
	}
	if len(data) > 256*1024 {
		t.Errorf("retained file is %d bytes, want at most %d", len(data), 256*1024)
	}
	if !bytes.Contains(data, []byte("[... truncated ...]")) {
		t.Errorf("retained file has no truncation marker")
	}
	if !strings.HasPrefix(string(data), "--- stdout ---\nhhhh") {
		t.Errorf("truncation lost the head of the output")
	}
	if !strings.HasSuffix(string(data), "--- stderr ---\n") {
		t.Errorf("truncation lost the tail of the output")
	}
	if !bytes.Contains(data[len(data)-200:], []byte("tttt")) {
		t.Errorf("truncation lost the tail bytes before the stderr label")
	}
}

func TestRunRejectsBadRecordID(t *testing.T) {
	t.Parallel()
	for _, id := range []string{"", "../escape", "a/b", `a\b`, ".hidden", "a b"} {
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			var calls []runnerCall
			meta := testMeta(t.TempDir())
			meta.ID = id
			_, err := Run(context.Background(), fakeRunner("", "", 0, false, nil, &calls),
				testCommand(), testRequirement(), meta, t.TempDir())
			if errs.CategoryOf(err) != errs.Validation {
				t.Errorf("Run(ID=%q) category = %v, want validation error", id, errs.CategoryOf(err))
			}
			if len(calls) != 0 {
				t.Errorf("runner ran despite the invalid record ID %q", id)
			}
		})
	}
}

func TestRunRejectsInvalidCommandBeforeRunning(t *testing.T) {
	t.Parallel()
	var calls []runnerCall
	cmd := testCommand()
	cmd.Argv = []string{"rm;rm"}
	_, err := Run(context.Background(), fakeRunner("", "", 0, false, nil, &calls),
		cmd, testRequirement(), testMeta(t.TempDir()), t.TempDir())
	if errs.CategoryOf(err) != errs.Validation {
		t.Errorf("category = %v, want validation error", errs.CategoryOf(err))
	}
	if len(calls) != 0 {
		t.Errorf("runner ran despite a rejected command")
	}
}

func TestRunRequiresRunnerAndRetainDir(t *testing.T) {
	t.Parallel()
	if _, err := Run(context.Background(), nil, testCommand(), testRequirement(), testMeta(t.TempDir()), t.TempDir()); err == nil {
		t.Errorf("Run() with a nil runner returned no error")
	}
	_, err := Run(context.Background(), fakeRunner("", "", 0, false, nil, nil),
		testCommand(), testRequirement(), testMeta(t.TempDir()), "")
	if errs.CategoryOf(err) != errs.Validation {
		t.Errorf("Run() with no retain directory category = %v, want validation error", errs.CategoryOf(err))
	}
}

func TestValidateCommand(t *testing.T) {
	t.Parallel()
	repoRoot := t.TempDir()
	outside := t.TempDir()

	valid := func() Command { return testCommand() }

	tests := []struct {
		name    string
		mutate  func(*Command)
		root    string
		wantErr bool
	}{
		{name: "valid command", mutate: func(*Command) {}},
		{name: "valid single-character name", mutate: func(c *Command) { c.Name = "a" }},
		{name: "valid subdirectory", mutate: func(c *Command) { c.Dir = "sub" }},
		{name: "valid nested subdirectory", mutate: func(c *Command) { c.Dir = filepath.Join("a", "b") }},
		{name: "valid timeout lower bound", mutate: func(c *Command) { c.TimeoutSeconds = 1 }},
		{name: "valid timeout upper bound", mutate: func(c *Command) { c.TimeoutSeconds = 600 }},
		{name: "empty name", mutate: func(c *Command) { c.Name = "" }, wantErr: true},
		{name: "uppercase name", mutate: func(c *Command) { c.Name = "Go-Test" }, wantErr: true},
		{name: "name leading hyphen", mutate: func(c *Command) { c.Name = "-x" }, wantErr: true},
		{name: "name trailing hyphen", mutate: func(c *Command) { c.Name = "x-" }, wantErr: true},
		{name: "empty argv", mutate: func(c *Command) { c.Argv = nil }, wantErr: true},
		{name: "argv NUL byte", mutate: func(c *Command) { c.Argv = []string{"go", "a\x00b"} }, wantErr: true},
		{name: "argv newline", mutate: func(c *Command) { c.Argv = []string{"go", "a\nb"} }, wantErr: true},
		{name: "argv carriage return", mutate: func(c *Command) { c.Argv = []string{"go", "a\rb"} }, wantErr: true},
		{name: "argv command chain semicolon", mutate: func(c *Command) { c.Argv = []string{"rm;rm"} }, wantErr: true},
		{name: "argv pipe", mutate: func(c *Command) { c.Argv = []string{"a|b"} }, wantErr: true},
		{name: "argv backtick", mutate: func(c *Command) { c.Argv = []string{"`x`"} }, wantErr: true},
		{name: "argv command substitution", mutate: func(c *Command) { c.Argv = []string{"$(x)"} }, wantErr: true},
		{name: "argv background ampersand", mutate: func(c *Command) { c.Argv = []string{"a&b"} }, wantErr: true},
		{name: "argv redirect in", mutate: func(c *Command) { c.Argv = []string{"a<b"} }, wantErr: true},
		{name: "argv redirect out", mutate: func(c *Command) { c.Argv = []string{"a>b"} }, wantErr: true},
		{name: "env missing equals", mutate: func(c *Command) { c.Env = []string{"NOEQUALS"} }, wantErr: true},
		{name: "env bad name", mutate: func(c *Command) { c.Env = []string{"1BAD=x"} }, wantErr: true},
		{name: "env newline in value", mutate: func(c *Command) { c.Env = []string{"A=b\nc"} }, wantErr: true},
		{name: "dir parent escape", mutate: func(c *Command) { c.Dir = ".." }, wantErr: true},
		{name: "dir relative escape", mutate: func(c *Command) { c.Dir = filepath.Join("..", "evil") }, wantErr: true},
		{name: "dir hidden escape through subdirectory", mutate: func(c *Command) { c.Dir = filepath.Join("sub", "..", "..", "evil") }, wantErr: true},
		{name: "dir absolute outside root", mutate: func(c *Command) { c.Dir = outside }, wantErr: true},
		{name: "dir newline", mutate: func(c *Command) { c.Dir = "a\nb" }, wantErr: true},
		{name: "timeout zero", mutate: func(c *Command) { c.TimeoutSeconds = 0 }, wantErr: true},
		{name: "timeout negative", mutate: func(c *Command) { c.TimeoutSeconds = -5 }, wantErr: true},
		{name: "timeout above maximum", mutate: func(c *Command) { c.TimeoutSeconds = 601 }, wantErr: true},
		{name: "empty repo root", mutate: func(*Command) {}, root: "unset", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := valid()
			tt.mutate(&cmd)
			root := repoRoot
			if tt.root == "unset" {
				root = ""
			}
			err := ValidateCommand(cmd, root)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ValidateCommand() = nil, want an error")
				}
				if errs.CategoryOf(err) != errs.Validation {
					t.Errorf("category = %v, want %v", errs.CategoryOf(err), errs.Validation)
				}
				return
			}
			if err != nil {
				t.Errorf("ValidateCommand() = %v, want nil", err)
			}
		})
	}
}

// passRecord builds a passing executable record at the given revisions.
func passRecord(name, sourceRevision string, planRevision int) Record {
	return Record{
		Name:           name,
		Kind:           KindExecutable,
		Status:         StatusPass,
		SourceRevision: sourceRevision,
		PlanRevision:   planRevision,
	}
}

func TestFresh(t *testing.T) {
	t.Parallel()
	matching := passRecord("go-test", "rev-a", 3)

	if err := Fresh(matching, "rev-a", 3); err != nil {
		t.Errorf("Fresh() = %v, want nil for matching revisions", err)
	}

	err := Fresh(matching, "rev-b", 3)
	if err == nil {
		t.Fatalf("Fresh() = nil, want an error for a stale source revision")
	}
	if errs.CategoryOf(err) != errs.Evidence {
		t.Errorf("category = %v, want %v", errs.CategoryOf(err), errs.Evidence)
	}
	for _, want := range []string{"rev-a", "rev-b"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Fresh() error %q does not name revision %q", err, want)
		}
	}

	err = Fresh(matching, "rev-a", 4)
	if err == nil {
		t.Fatalf("Fresh() = nil, want an error for a stale plan revision")
	}
	for _, want := range []string{"plan revision 3", "is 4"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Fresh() error %q does not say %q", err, want)
		}
	}
}

func TestRequiredSatisfied(t *testing.T) {
	t.Parallel()
	const (
		src  = "rev-a"
		plan = 3
	)
	humanPass := Record{Name: "ux-check", Kind: KindHuman, Status: StatusPass, SourceRevision: src, PlanRevision: plan}

	tests := []struct {
		name         string
		requirements []Requirement
		records      []Record
		wantErr      bool
		wantContains []string
	}{
		{
			name: "all required satisfied",
			requirements: []Requirement{
				{Name: "go-test", Kind: KindExecutable, Level: LevelRequired},
				{Name: "ux-check", Kind: KindHuman, Level: LevelRequired},
			},
			records: []Record{passRecord("go-test", src, plan), humanPass},
		},
		{
			name:         "empty requirements",
			requirements: nil,
			records:      nil,
		},
		{
			name:         "missing required executable",
			requirements: []Requirement{{Name: "go-test", Kind: KindExecutable, Level: LevelRequired}},
			wantErr:      true,
			wantContains: []string{"go-test", "no evidence"},
		},
		{
			name:         "failed required executable",
			requirements: []Requirement{{Name: "go-test", Kind: KindExecutable, Level: LevelRequired}},
			records: []Record{{
				Name: "go-test", Kind: KindExecutable, Status: StatusFail,
				SourceRevision: src, PlanRevision: plan,
			}},
			wantErr:      true,
			wantContains: []string{"go-test", "did not pass"},
		},
		{
			name:         "stale required executable",
			requirements: []Requirement{{Name: "go-test", Kind: KindExecutable, Level: LevelRequired}},
			records:      []Record{passRecord("go-test", "rev-old", plan)},
			wantErr:      true,
			wantContains: []string{"rev-old", "rev-a"},
		},
		{
			name:         "stale plan revision rejected",
			requirements: []Requirement{{Name: "go-test", Kind: KindExecutable, Level: LevelRequired}},
			records:      []Record{passRecord("go-test", src, 2)},
			wantErr:      true,
			wantContains: []string{"plan revision 2"},
		},
		{
			name:         "recommended failure accepted",
			requirements: []Requirement{{Name: "go-vet", Kind: KindExecutable, Level: LevelRecommended}},
			records: []Record{{
				Name: "go-vet", Kind: KindExecutable, Status: StatusFail,
				SourceRevision: src, PlanRevision: plan,
			}},
		},
		{
			name:         "not-applicable ignored",
			requirements: []Requirement{{Name: "a11y", Kind: KindHuman, Level: LevelNotApplicable}},
		},
		{
			name:         "human requirement without record",
			requirements: []Requirement{{Name: "ux-check", Kind: KindHuman, Level: LevelRequired}},
			wantErr:      true,
			wantContains: []string{"ux-check", "needs a human result"},
		},
		{
			name:         "human requirement with passing human record",
			requirements: []Requirement{{Name: "ux-check", Kind: KindHuman, Level: LevelRequired}},
			records:      []Record{humanPass},
		},
		{
			name:         "inspection requirement accepts a human record",
			requirements: []Requirement{{Name: "ux-check", Kind: KindInspection, Level: LevelRequired}},
			records:      []Record{humanPass},
		},
		{
			name:         "human record cannot satisfy an executable requirement",
			requirements: []Requirement{{Name: "go-test", Kind: KindExecutable, Level: LevelRequired}},
			records: []Record{{
				Name: "go-test", Kind: KindHuman, Status: StatusPass,
				SourceRevision: src, PlanRevision: plan,
			}},
			wantErr:      true,
			wantContains: []string{"go-test", "no evidence"},
		},
		{
			name:         "stale human record rejected",
			requirements: []Requirement{{Name: "ux-check", Kind: KindHuman, Level: LevelRequired}},
			records: []Record{{
				Name: "ux-check", Kind: KindHuman, Status: StatusPass,
				SourceRevision: "rev-old", PlanRevision: plan,
			}},
			wantErr:      true,
			wantContains: []string{"rev-old"},
		},
		{
			name:         "unknown level fails closed",
			requirements: []Requirement{{Name: "go-test", Kind: KindExecutable, Level: "optional"}},
			wantErr:      true,
			wantContains: []string{"unknown level"},
		},
		{
			name: "every problem listed at once",
			requirements: []Requirement{
				{Name: "go-test", Kind: KindExecutable, Level: LevelRequired},
				{Name: "ux-check", Kind: KindHuman, Level: LevelRequired},
			},
			wantErr:      true,
			wantContains: []string{"go-test", "ux-check"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := RequiredSatisfied(tt.requirements, tt.records, src, plan)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("RequiredSatisfied() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("RequiredSatisfied() = nil, want an error")
			}
			if errs.CategoryOf(err) != errs.Evidence {
				t.Errorf("category = %v, want %v", errs.CategoryOf(err), errs.Evidence)
			}
			for _, want := range tt.wantContains {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
		})
	}
}
