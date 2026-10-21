// This file is part of PeerFoil.
// internal/provider/runner.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Bounded local process runner shared by every provider adapter. Runs one
// command with an argument array, an explicit environment, capped output, and a
// deadline, and never builds a shell string.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

// Package provider runs model command-line tools such as Claude Code and Codex as
// local processes and adapts their output to one shared contract.
//
// Every subprocess starts from an argument array with an explicit working
// directory; no shell ever interprets the command. The environment is an explicit
// allowlist, never the full parent environment. Output is captured with a size cap
// and runs end at a deadline.
//
// Alpha limitation: when a run times out or is cancelled, the runner kills only
// the direct child process. Processes that the tool itself started (grandchildren)
// may keep running until they finish on their own. This limit is reported here
// rather than hidden; a stricter process-group stop is deferred work.
package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

// DefaultTimeout bounds a run when the request does not set its own limit.
const DefaultTimeout = 10 * time.Minute

// DefaultMaxOutputBytes caps each of stdout and stderr when the request does not
// set its own limit. Provider envelopes are small; anything larger signals a
// runaway tool, and the retained prefix is enough to diagnose it.
const DefaultMaxOutputBytes int64 = 4 * 1024 * 1024

// waitGrace bounds how long the runner waits for output pipes to drain after the
// child exits or is killed. A grandchild that inherited the pipes could otherwise
// hold them open and block the runner forever, which matters most on Windows.
const waitGrace = 10 * time.Second

// Request describes one bounded subprocess run.
type Request struct {
	// Argv is the complete argument array. Argv[0] is the executable.
	Argv []string
	// Dir is the working directory. Empty means the current directory.
	Dir string
	// Env is the complete environment for the child, already reduced to an
	// allowlist by the caller. When nil, the minimal set from BaseEnv is used.
	// The parent's full environment is never inherited implicitly.
	Env []string
	// Stdin is written to the child's standard input, then the stream closes.
	Stdin []byte
	// Timeout is the hard deadline for the run. Zero means DefaultTimeout.
	Timeout time.Duration
	// MaxOutputBytes caps stdout and stderr separately. Zero means
	// DefaultMaxOutputBytes.
	MaxOutputBytes int64
}

// Result reports what one run produced. It is returned even when Run also
// returns a timeout or cancellation error, so callers can inspect the partial
// output that arrived before the stop.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Duration time.Duration
	// TimedOut is true when the run hit its deadline and the child was killed.
	TimedOut bool
	// Truncated is true when stdout or stderr hit MaxOutputBytes; the captured
	// bytes are the prefix of what the child wrote.
	Truncated bool
}

// BaseEnv builds the minimal child environment from the current process: PATH,
// the home variable (HOME or USERPROFILE), the temporary-directory variables
// (TEMP and TMP), SystemRoot and ComSpec on Windows, and LANG and LC_ALL when
// set. Everything else, including any credential or token variable, is excluded
// so a child tool never sees secrets it was not explicitly given.
func BaseEnv() []string {
	names := []string{"PATH", "HOME", "USERPROFILE", "TEMP", "TMP", "LANG", "LC_ALL"}
	if runtime.GOOS == "windows" {
		names = append(names, "SystemRoot", "ComSpec")
	}
	env := make([]string, 0, len(names))
	for _, name := range names {
		if value, ok := os.LookupEnv(name); ok {
			env = append(env, name+"="+value)
		}
	}
	return env
}

// limitedBuffer keeps at most limit bytes and then silently discards the rest,
// reporting full writes so the child process keeps running instead of dying on a
// broken pipe. The retained bytes are always the prefix of what was written.
type limitedBuffer struct {
	buf       bytes.Buffer
	limit     int64
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	remaining := b.limit - int64(b.buf.Len())
	switch {
	case remaining <= 0:
		b.truncated = true
	case int64(len(p)) > remaining:
		b.buf.Write(p[:remaining])
		b.truncated = true
	default:
		b.buf.Write(p)
	}
	return len(p), nil
}

// Run executes one subprocess within the request's bounds and waits for it.
//
// A nonzero exit is not an error: it is reported in Result.ExitCode so adapters
// can interpret it. Run returns an error only when the process could not start
// (category Provider), the caller's context was cancelled (Cancelled), or the
// deadline expired and the child was killed (Timeout, with Result.TimedOut set).
// Error messages name only the executable's base name, never arguments or input,
// because those can carry prompts, paths, or other content that must stay out of
// logs.
func Run(ctx context.Context, req Request) (Result, error) {
	if len(req.Argv) == 0 || req.Argv[0] == "" {
		return Result{}, errs.New(errs.Provider, "a provider run was requested without a command to execute", "")
	}
	base := filepath.Base(req.Argv[0])

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	maxOutput := req.MaxOutputBytes
	if maxOutput <= 0 {
		maxOutput = DefaultMaxOutputBytes
	}

	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, req.Argv[0], req.Argv[1:]...)
	cmd.Dir = req.Dir
	if req.Env != nil {
		cmd.Env = req.Env
	} else {
		cmd.Env = BaseEnv()
	}
	if len(req.Stdin) > 0 {
		cmd.Stdin = bytes.NewReader(req.Stdin)
	}
	stdout := &limitedBuffer{limit: maxOutput}
	stderr := &limitedBuffer{limit: maxOutput}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.WaitDelay = waitGrace

	started := time.Now()
	if err := cmd.Start(); err != nil {
		return Result{}, errs.Wrap(errs.Provider, fmt.Sprintf("could not start %s", base), err)
	}
	waitErr := cmd.Wait()

	result := Result{
		Stdout:    stdout.buf.Bytes(),
		Stderr:    stderr.buf.Bytes(),
		Duration:  time.Since(started),
		Truncated: stdout.truncated || stderr.truncated,
	}

	// The caller's own context ending takes priority over the run deadline, so a
	// user cancellation is never misreported as a provider timeout.
	if ctxErr := ctx.Err(); ctxErr != nil {
		if errors.Is(ctxErr, context.DeadlineExceeded) {
			result.TimedOut = true
			return result, errs.Wrap(errs.Timeout, fmt.Sprintf("%s was stopped because the surrounding operation passed its time limit", base), ctxErr)
		}
		return result, errs.Wrap(errs.Cancelled, fmt.Sprintf("%s was stopped because the operation was cancelled", base), ctxErr)
	}
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		result.TimedOut = true
		return result, errs.New(errs.Timeout, fmt.Sprintf("%s did not finish within %s and was stopped", base, timeout), "allow a longer timeout for this task or make the task smaller")
	}

	if waitErr != nil {
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return result, errs.Wrap(errs.Provider, fmt.Sprintf("%s failed while running", base), waitErr)
	}
	return result, nil
}
