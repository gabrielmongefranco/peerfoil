// This file is part of PeerFoil.
// internal/evidence/evidence.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Controller-run evidence: validates evidence commands, runs them through an
// injected process runner, retains scrubbed and size-capped output, and decides whether
// the required evidence for a revision is complete and fresh.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

// Package evidence produces and checks the records that prove a check actually ran
// against the exact revision under review. A model's claim is never evidence: records
// come from commands the controller ran itself, or from an explicit human result. The
// package records outcomes without deciding policy; whether a failure blocks the
// workflow is the controller's decision.
package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/fsx"
	"github.com/gabrielmongefranco/peerfoil/internal/redact"
)

// Requirement kinds. Executable checks are commands the controller runs; inspection
// and human checks are completed by a person and recorded by the caller.
const (
	KindExecutable = "executable"
	KindInspection = "inspection"
	KindHuman      = "human"
)

// Requirement levels. Only required evidence can block the workflow; recommended
// failures stay visible but do not block, and not-applicable items are skipped.
const (
	LevelRequired      = "required"
	LevelRecommended   = "recommended"
	LevelNotApplicable = "not-applicable"
)

// Record statuses. Pass means exit code zero; fail means the command finished with a
// nonzero exit code; timeout means it was stopped at its time limit; error means it
// could not run or finish at all.
const (
	StatusPass    = "pass"
	StatusFail    = "fail"
	StatusTimeout = "timeout"
	StatusError   = "error"
)

// Runner executes one command without a shell and returns its captured output.
//
// The contract: argv is the complete argument vector (argv[0] is the program), dir is
// the absolute working directory, and env is the complete environment, already reduced
// to an allowlist by the caller; an implementation must never add the parent process
// environment. timedOut is true when the command was stopped because it exceeded
// timeout. err is non-nil only when the command could not be started or was
// interrupted, never for an ordinary nonzero exit. The controller adapts its shared
// process runner to this type; tests inject fakes.
type Runner func(ctx context.Context, argv []string, dir string, env []string, timeout time.Duration) (stdout, stderr []byte, exitCode int, timedOut bool, err error)

// Requirement is one evidence item an acceptance contract asks for.
type Requirement struct {
	// Name identifies the check; a Record satisfies the Requirement with the same Name.
	Name string
	// Kind is one of the Kind constants: executable, inspection, or human.
	Kind string
	// Level is one of the Level constants: required, recommended, or not-applicable.
	Level string
}

// Command describes one executable evidence command. It is declared data from a pack
// or task, so ValidateCommand treats every field as untrusted.
type Command struct {
	// Name identifies the check, in lowercase letters, digits, and inner hyphens.
	Name string
	// Argv is the complete argument vector; Argv[0] is the program to run.
	Argv []string
	// Dir is the working directory relative to the project root; empty means the root.
	Dir string
	// Env is the complete environment as KEY=value entries. The caller builds it as an
	// allowlist; the parent process environment is never inherited implicitly, so a nil
	// Env runs the command with an empty environment.
	Env []string
	// TimeoutSeconds is the time limit, between 1 and 600 seconds.
	TimeoutSeconds int
}

// Meta carries the identity a Record is tied to: which project state the evidence
// supports and how the record is named and timed.
type Meta struct {
	// ID names the record and its retained-output file; letters, digits, dot, dash,
	// and underscore only, so it can never leave the retention directory.
	ID string
	// TaskID is the task the evidence supports.
	TaskID string
	// PlanRevision and ArchitectureRevision are the accepted document revisions the
	// evidence was collected under.
	PlanRevision         int
	ArchitectureRevision int
	// SourceRevision is the exact produced revision the command ran against.
	SourceRevision string
	// ToolVersion records the version of the tool that produced the output.
	ToolVersion string
	// RepoRoot is the project root that Command.Dir is resolved against.
	RepoRoot string
	// Now supplies timestamps; nil means time.Now. Tests inject a fixed clock.
	Now func() time.Time
}

// Record is one piece of evidence tied to an exact revision. Argv and Dir are stored
// as declared (Dir relative to the project root) so records stay portable and never
// name a user's home directory.
type Record struct {
	ID                   string
	Name                 string
	Kind                 string
	Level                string
	TaskID               string
	PlanRevision         int
	ArchitectureRevision int
	SourceRevision       string
	Argv                 []string
	Dir                  string
	ExitCode             int
	DurationSeconds      float64
	ToolVersion          string
	// Status is one of the Status constants. For human and inspection records the
	// caller sets it from the person's explicit result.
	Status string
	// RetainedPath is where the scrubbed, size-capped output was written, and
	// RetainedSHA256 is the SHA-256 of exactly those bytes.
	RetainedPath   string
	RetainedSHA256 string
	// Redacted reports whether scrubbing removed anything from the retained output.
	Redacted bool
	// At is the completion time in UTC ISO 8601 with seconds, such as
	// 2026-10-21T14:03:05Z.
	At string
}

// ### Validate Inputs ###

// namePattern allows lowercase letters and digits with inner hyphens, so names are
// safe in filenames, JSON, and status output.
var namePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// idPattern keeps record IDs free of path separators and leading dots, so the
// retained-output file can never land outside the retention directory.
var idPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// envPattern requires POSIX-style variable names followed by an equals sign.
var envPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// shellMetacharacters are rejected in every argument. The runner never uses a shell,
// but rejecting them keeps a declared command from smuggling shell syntax toward any
// later consumer, such as a log viewer or a copy-pasted reproduction.
const shellMetacharacters = ";&|`$<>"

// ValidateCommand checks a declared evidence command before anything runs it. It
// returns an errs.Validation error naming the first problem: a name outside the
// allowed pattern, an empty argument vector, control characters or shell
// metacharacters in an argument, a malformed environment entry, a working directory
// outside repoRoot, or a timeout outside 1 to 600 seconds. Argument and environment
// values are never echoed into the error, because they may hold secrets.
func ValidateCommand(cmd Command, repoRoot string) error {
	if !namePattern.MatchString(cmd.Name) {
		return errs.New(errs.Validation,
			fmt.Sprintf("evidence command name %q is not allowed", cmd.Name),
			"use lowercase letters, digits, and inner hyphens, such as go-test")
	}
	if len(cmd.Argv) == 0 {
		return errs.New(errs.Validation,
			fmt.Sprintf("evidence command %q has no arguments", cmd.Name),
			"declare the program and its arguments as a list")
	}
	for i, arg := range cmd.Argv {
		if strings.ContainsAny(arg, "\x00\r\n") {
			return errs.New(errs.Validation,
				fmt.Sprintf("argument %d of evidence command %q contains a control character", i+1, cmd.Name),
				"remove newlines and NUL bytes from the declared arguments")
		}
		if strings.ContainsAny(arg, shellMetacharacters) {
			return errs.New(errs.Validation,
				fmt.Sprintf("argument %d of evidence command %q contains a shell metacharacter", i+1, cmd.Name),
				"evidence commands run without a shell; declare plain arguments without ;&|`$<>")
		}
	}
	for i, entry := range cmd.Env {
		if strings.ContainsAny(entry, "\x00\r\n") || !envPattern.MatchString(entry) {
			return errs.New(errs.Validation,
				fmt.Sprintf("environment entry %d of evidence command %q is not a plain KEY=value pair", i+1, cmd.Name),
				"declare each entry as NAME=value without newlines")
		}
	}
	if repoRoot == "" {
		return errs.New(errs.Validation,
			fmt.Sprintf("evidence command %q has no project root to run under", cmd.Name),
			"")
	}
	if strings.ContainsAny(cmd.Dir, "\x00\r\n") {
		return errs.New(errs.Validation,
			fmt.Sprintf("working directory of evidence command %q contains a control character", cmd.Name),
			"declare a plain relative directory")
	}
	if cmd.Dir != "" {
		if _, err := resolveDir(cmd.Dir, repoRoot); err != nil {
			return err
		}
	}
	if cmd.TimeoutSeconds < 1 || cmd.TimeoutSeconds > 600 {
		return errs.New(errs.Validation,
			fmt.Sprintf("timeout of evidence command %q is %d seconds; it must be between 1 and 600", cmd.Name, cmd.TimeoutSeconds),
			"set TimeoutSeconds between 1 and 600")
	}
	return nil
}

// resolveDir turns a declared working directory into an absolute path under root, or
// returns an errs.Validation error when the directory would escape the project. The
// declared value, not the resolved absolute path, goes into the error so messages
// never reveal where the project lives on disk.
func resolveDir(dir, root string) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", errs.Wrap(errs.Validation, "the project root could not be resolved", err)
	}
	if dir == "" {
		return absRoot, nil
	}
	abs := dir
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(absRoot, abs)
	}
	abs = filepath.Clean(abs)
	rel, err := filepath.Rel(absRoot, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errs.New(errs.Validation,
			fmt.Sprintf("working directory %q is outside the project", dir),
			"declare the directory relative to the project root")
	}
	return abs, nil
}

// ### Retrieve Source Data ###

// retainedByteCap limits every retained output file. Diagnostic value lives at the
// start (what ran) and the end (how it failed), so truncation keeps both halves.
const retainedByteCap = 256 * 1024

var truncationMarker = []byte("\n[... truncated ...]\n")

// Run validates cmd, executes it through runner, and returns a Record tied to the
// revisions in meta. The combined output is scrubbed of likely credentials, capped at
// 256 KiB keeping its head and tail, written atomically to retainDir/<meta.ID>.txt,
// and hashed.
//
// A finished command always yields a nil error, whatever its exit code: Status says
// pass, fail, or timeout, and blocking on a failure is the controller's decision. Run
// returns an error only when validation fails or the command could not run at all; in
// the latter case the Record is still returned with Status "error" and whatever output
// was captured.
func Run(ctx context.Context, runner Runner, cmd Command, req Requirement, meta Meta, retainDir string) (Record, error) {
	record := Record{
		ID:                   meta.ID,
		Name:                 cmd.Name,
		Kind:                 req.Kind,
		Level:                req.Level,
		TaskID:               meta.TaskID,
		PlanRevision:         meta.PlanRevision,
		ArchitectureRevision: meta.ArchitectureRevision,
		SourceRevision:       meta.SourceRevision,
		Argv:                 append([]string(nil), cmd.Argv...),
		Dir:                  cmd.Dir,
		ToolVersion:          meta.ToolVersion,
	}
	if runner == nil {
		return record, errs.New(errs.Internal, "no process runner was supplied for evidence collection", "")
	}
	if !idPattern.MatchString(meta.ID) {
		return record, errs.New(errs.Validation,
			"the evidence record ID is not allowed",
			"use letters, digits, dots, dashes, and underscores, starting with a letter or digit")
	}
	if retainDir == "" {
		return record, errs.New(errs.Validation, "no directory was supplied to retain evidence output", "")
	}
	if err := ValidateCommand(cmd, meta.RepoRoot); err != nil {
		return record, err
	}
	dir, err := resolveDir(cmd.Dir, meta.RepoRoot)
	if err != nil {
		return record, err
	}

	now := meta.Now
	if now == nil {
		now = time.Now
	}

	// ### Transform Records ###

	started := now()
	stdout, stderr, exitCode, timedOut, runErr := runner(ctx,
		append([]string(nil), cmd.Argv...), dir, cmd.Env,
		time.Duration(cmd.TimeoutSeconds)*time.Second)
	finished := now()

	record.ExitCode = exitCode
	record.DurationSeconds = finished.Sub(started).Seconds()
	record.At = finished.UTC().Format(time.RFC3339)
	switch {
	case timedOut:
		record.Status = StatusTimeout
	case runErr != nil:
		record.Status = StatusError
	case exitCode == 0:
		record.Status = StatusPass
	default:
		record.Status = StatusFail
	}

	// ### Save Results ###

	// Scrub before capping: redaction sees every complete value, and cutting
	// already-scrubbed text cannot reintroduce one.
	scrubbed, redacted := redact.Scrub(combineOutput(stdout, stderr))
	record.Redacted = redacted
	retained := capOutput(scrubbed)

	// The retention directory is private to the user running the controller.
	if err := os.MkdirAll(retainDir, 0o700); err != nil {
		return record, errs.Wrap(errs.Internal,
			fmt.Sprintf("the retention directory for evidence command %q could not be created", cmd.Name), err)
	}
	path := filepath.Join(retainDir, meta.ID+".txt")
	if err := fsx.WriteFileAtomic(path, retained, 0o600); err != nil {
		return record, errs.Wrap(errs.Internal,
			fmt.Sprintf("the output of evidence command %q could not be retained", cmd.Name), err)
	}
	sum := sha256.Sum256(retained)
	record.RetainedPath = path
	record.RetainedSHA256 = hex.EncodeToString(sum[:])

	if runErr != nil && !timedOut {
		category := errs.Evidence
		if errors.Is(runErr, context.Canceled) {
			category = errs.Cancelled
		}
		return record, errs.Wrap(category,
			fmt.Sprintf("evidence command %q could not run", cmd.Name), runErr)
	}
	return record, nil
}

// combineOutput joins stdout and stderr into one labeled document so one retained
// file preserves both streams and their origin.
func combineOutput(stdout, stderr []byte) []byte {
	out := make([]byte, 0, len(stdout)+len(stderr)+32)
	out = append(out, "--- stdout ---\n"...)
	out = append(out, stdout...)
	if len(stdout) > 0 && stdout[len(stdout)-1] != '\n' {
		out = append(out, '\n')
	}
	out = append(out, "--- stderr ---\n"...)
	out = append(out, stderr...)
	if len(stderr) > 0 && stderr[len(stderr)-1] != '\n' {
		out = append(out, '\n')
	}
	return out
}

// capOutput enforces retainedByteCap by keeping the first and last halves with a
// marker between them.
func capOutput(data []byte) []byte {
	if len(data) <= retainedByteCap {
		return data
	}
	half := (retainedByteCap - len(truncationMarker)) / 2
	out := make([]byte, 0, retainedByteCap)
	out = append(out, data[:half]...)
	out = append(out, truncationMarker...)
	out = append(out, data[len(data)-half:]...)
	return out
}

// ### Validate Freshness ###

// Fresh reports whether a record still describes the work under review. It returns an
// errs.Evidence error naming both revisions when the record's SourceRevision or
// PlanRevision differs from the ones given.
func Fresh(record Record, sourceRevision string, planRevision int) error {
	var problems []string
	if record.SourceRevision != sourceRevision {
		problems = append(problems, fmt.Sprintf(
			"evidence %q was collected at source revision %q but the work under review is %q",
			record.Name, record.SourceRevision, sourceRevision))
	}
	if record.PlanRevision != planRevision {
		problems = append(problems, fmt.Sprintf(
			"evidence %q was collected at plan revision %d but the work under review is %d",
			record.Name, record.PlanRevision, planRevision))
	}
	if len(problems) == 0 {
		return nil
	}
	return errs.New(errs.Evidence, strings.Join(problems, "; "),
		"rerun the check against the current revision")
}

// RequiredSatisfied reports whether every required requirement has fresh, passing
// evidence for the given revisions. Recommended and not-applicable requirements never
// block. Every missing, failed, or stale required item is listed in one errs.Evidence
// error, so the person sees the whole remaining list at once. A requirement of kind
// human or inspection without a record needs a human result; a human record cannot
// satisfy an executable requirement, because a claim is not evidence that a command
// ran.
func RequiredSatisfied(requirements []Requirement, records []Record, sourceRevision string, planRevision int) error {
	var problems []string
	for _, req := range requirements {
		switch req.Level {
		case LevelNotApplicable, LevelRecommended:
			continue
		case LevelRequired:
			// Checked below.
		default:
			problems = append(problems, fmt.Sprintf(
				"requirement %q has the unknown level %q; use required, recommended, or not-applicable",
				req.Name, req.Level))
			continue
		}
		if problem := requirementProblem(req, records, sourceRevision, planRevision); problem != "" {
			problems = append(problems, problem)
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return errs.New(errs.Evidence, strings.Join(problems, "; "),
		"rerun or complete the listed checks against the current revision")
}

// requirementProblem returns an empty string when req is satisfied, or one sentence
// describing why it is not. A stale passing record is reported as stale rather than
// missing, so the person knows a rerun is enough.
func requirementProblem(req Requirement, records []Record, sourceRevision string, planRevision int) string {
	var matches []Record
	for _, rec := range records {
		if rec.Name == req.Name && kindSatisfies(req.Kind, rec.Kind) {
			matches = append(matches, rec)
		}
	}
	if len(matches) == 0 {
		if req.Kind == KindExecutable {
			return fmt.Sprintf("required check %q has no evidence", req.Name)
		}
		return fmt.Sprintf("required %s check %q needs a human result", req.Kind, req.Name)
	}
	var stale, failed string
	for _, rec := range matches {
		if rec.Status != StatusPass {
			failed = fmt.Sprintf("required check %q did not pass (last status %q)", req.Name, rec.Status)
			continue
		}
		if err := Fresh(rec, sourceRevision, planRevision); err != nil {
			stale = staleMessage(err)
			continue
		}
		return ""
	}
	if stale != "" {
		return stale
	}
	return failed
}

// kindSatisfies decides which record kinds can satisfy a requirement kind. Executable
// requirements accept only controller-run command records. Inspection requirements
// accept inspection or human records, since both are explicit human results.
func kindSatisfies(requirementKind, recordKind string) bool {
	switch requirementKind {
	case KindExecutable:
		return recordKind == KindExecutable
	case KindInspection:
		return recordKind == KindInspection || recordKind == KindHuman
	case KindHuman:
		return recordKind == KindHuman
	default:
		return false
	}
}

// staleMessage extracts the plain message from a Fresh error so it can be merged into
// a combined list without repeating the category prefix.
func staleMessage(err error) string {
	var e *errs.Error
	if errors.As(err, &e) {
		return e.Message
	}
	return err.Error()
}
