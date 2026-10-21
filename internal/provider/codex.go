// This file is part of PeerFoil.
// internal/provider/codex.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Adapter that runs the Codex CLI non-interactively, reads its event
// stream, and recovers a bounded answer when a run passes its soft time limit.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

// codexNudgeMessage asks a timed-out session to finish with what it has. It is
// the second stage of the two-stage deadline: the soft limit interrupts, this
// message collects, and the nudge's own hard limit ends the session.
const codexNudgeMessage = "Stop and answer now with what you have, in the format requested."

// codexNudgeTimeout bounds the wrap-up call after a soft-limit interrupt.
const codexNudgeTimeout = time.Minute

// Codex runs the Codex CLI non-interactively through `codex exec`. One value
// serves one seat; every Invoke starts a fresh session.
type Codex struct {
	exe  string
	seat Seat
	run  RunFunc
}

var _ Adapter = (*Codex)(nil)

// NewCodex builds a Codex CLI adapter for one seat. exe is the executable path
// or command name. run may be nil, which uses the real process runner.
func NewCodex(exe string, seat Seat, run RunFunc) *Codex {
	if run == nil {
		run = Run
	}
	return &Codex{exe: exe, seat: seat, run: run}
}

// Name identifies the tool in records and messages.
func (c *Codex) Name() string { return "codex" }

// LineageRoot maps a model identifier to its canonical lineage root.
func (c *Codex) LineageRoot(model string) string { return lineageRoot(model) }

// Probe checks that the Codex CLI answers a version call and then asks the tool
// itself whether it is signed in, so PeerFoil never stores or inspects provider
// credentials.
func (c *Codex) Probe(ctx context.Context) ProbeResult {
	res, err := c.run(ctx, Request{Argv: []string{c.exe, "--version"}, Timeout: probeTimeout})
	if err != nil || res.ExitCode != 0 {
		return ProbeResult{Detail: "Codex did not answer a version check; install the Codex CLI or add it to PATH"}
	}
	probe := ProbeResult{Found: true, Version: strings.TrimSpace(string(res.Stdout))}
	login, loginErr := c.run(ctx, Request{Argv: []string{c.exe, "login", "status"}, Timeout: probeTimeout})
	if loginErr == nil && login.ExitCode == 0 {
		probe.LoggedIn = true
		probe.Detail = "Codex is installed and signed in"
	} else {
		probe.Detail = "Codex is installed but not signed in; run: codex login"
	}
	return probe
}

// Invoke runs one fresh non-interactive Codex session. The packet goes in on
// standard input, the event stream arrives on standard output, and the final
// answer lands in a temporary file in the system temporary directory, never
// inside the workspace, so a produced change set can never contain it. When the
// run passes its soft time limit, one bounded wrap-up call asks the same session
// to answer with what it has.
func (c *Codex) Invoke(ctx context.Context, inv Invocation) (Outcome, error) {
	switch inv.Sandbox {
	case SandboxReadOnly, SandboxWorkspaceWrite:
	default:
		// Deny by default: an unrecognized sandbox must never reach the tool,
		// which could interpret it or fall through to a wider mode.
		return Outcome{}, errs.New(errs.Validation,
			fmt.Sprintf("sandbox mode %q is not supported", inv.Sandbox),
			`use "read-only" or "workspace-write"`)
	}

	answerPath, err := codexAnswerFile()
	if err != nil {
		return Outcome{}, err
	}
	defer os.Remove(answerPath)

	// These are the documented non-interactive options of the Codex CLI version
	// this release was tested against. The trailing "-" reads the prompt from
	// standard input so it never appears in a process list.
	argv := []string{
		c.exe, "exec",
		"--sandbox", inv.Sandbox,
		"--skip-git-repo-check",
		"-C", inv.Dir,
		"--json",
		"--output-last-message", answerPath,
	}
	if effort := c.seat.Effort; effort != "" && effort != "default" {
		argv = append(argv, "-c", "model_reasoning_effort="+effort)
	}
	if model := c.seat.Model; model != "" && model != "default" {
		argv = append(argv, "-m", model)
	}
	argv = append(argv, "-")

	res, runErr := c.run(ctx, Request{
		Argv:    argv,
		Dir:     inv.Dir,
		Stdin:   []byte(inv.Packet),
		Timeout: inv.Timeout,
	})
	// The partial event stream is scanned even after a failed run, because the
	// session identifier inside it is what makes a wrap-up call possible.
	threadID := codexThreadID(res.Stdout)
	if runErr != nil {
		if res.TimedOut {
			return c.nudge(ctx, inv, threadID, res.Duration)
		}
		return Outcome{}, runErr
	}
	if res.ExitCode != 0 {
		return Outcome{}, errs.New(errs.Provider,
			fmt.Sprintf("Codex exited with status %d before returning a result", res.ExitCode),
			"check the sign-in with: codex login status, and that the selected model and effort exist")
	}

	text, readErr := readAnswerFile(answerPath)
	if readErr != nil {
		// The underlying error names a path in the user's temporary directory,
		// which identifies the user, so it is summarized instead of attached.
		return Outcome{}, errs.New(errs.Provider,
			"Codex finished but its final answer file could not be read; the output shape was not recognized",
			"run the task again; if this repeats, check the installed Codex CLI version")
	}
	return Outcome{
		Text:      string(text),
		SessionID: threadID,
		ExitCode:  res.ExitCode,
		Duration:  res.Duration,
	}, nil
}

// nudge runs the one bounded wrap-up call after a soft-limit interrupt. Without
// a session identifier there is nothing to resume, and a failed wrap-up ends the
// invocation; retries would only extend a run that already overran its budget.
func (c *Codex) nudge(ctx context.Context, inv Invocation, threadID string, elapsed time.Duration) (Outcome, error) {
	timeoutErr := errs.New(errs.Timeout,
		"the Codex run passed its time limit and produced no result",
		"allow a longer timeout for this task or make the task smaller")
	if threadID == "" {
		return Outcome{}, timeoutErr
	}

	answerPath, err := codexAnswerFile()
	if err != nil {
		return Outcome{}, err
	}
	defer os.Remove(answerPath)

	// The resume subcommand takes no --sandbox flag, and without an explicit override
	// it can reopen the session under the user's wider default sandbox, so the
	// original mode is re-asserted through the equivalent configuration setting.
	argv := []string{
		c.exe, "exec", "resume", threadID,
		"-c", "sandbox_mode=" + inv.Sandbox,
		"--json",
		"--output-last-message", answerPath,
		codexNudgeMessage,
	}
	res, runErr := c.run(ctx, Request{Argv: argv, Dir: inv.Dir, Timeout: codexNudgeTimeout})
	if runErr != nil || res.ExitCode != 0 {
		return Outcome{}, timeoutErr
	}
	text, readErr := readAnswerFile(answerPath)
	if readErr != nil {
		return Outcome{}, timeoutErr
	}
	return Outcome{
		Text:      string(text),
		SessionID: threadID,
		ExitCode:  res.ExitCode,
		Duration:  elapsed + res.Duration,
		TimedOut:  true,
	}, nil
}

// readAnswerFile reads a final-answer file with the same size bound the runner puts
// on a stream, so a runaway answer cannot be loaded whole into memory.
func readAnswerFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > DefaultMaxOutputBytes {
		return nil, fmt.Errorf("the answer is larger than the %d-byte output bound", DefaultMaxOutputBytes)
	}
	return os.ReadFile(path)
}

// codexAnswerFile creates an empty, closed temporary file in the system
// temporary directory for the tool's final answer. It is created outside any
// workspace and closed at once so the child process can open it, which Windows
// file locking would otherwise prevent.
func codexAnswerFile() (string, error) {
	file, err := os.CreateTemp("", "peerfoil-codex-*.txt")
	if err != nil {
		// The underlying error names the user's temporary directory, which
		// identifies the user, so it is summarized instead of attached.
		return "", errs.New(errs.Provider,
			"could not create a temporary file for the Codex answer",
			"check that the system temporary directory exists and is writable")
	}
	path := file.Name()
	if closeErr := file.Close(); closeErr != nil {
		os.Remove(path)
		return "", errs.New(errs.Provider,
			"could not prepare a temporary file for the Codex answer",
			"check that the system temporary directory is writable")
	}
	return path, nil
}

// codexThreadID scans a --json event stream for the thread.started event and
// returns its thread identifier. Lines that are not complete JSON objects are
// skipped without error, because a stream cut off by a timeout legitimately ends
// mid-line.
func codexThreadID(stream []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(stream))
	// Event lines can exceed the scanner's 64 KiB default; the cap matches the
	// runner's output bound.
	scanner.Buffer(make([]byte, 0, 64*1024), int(DefaultMaxOutputBytes))
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 || line[0] != '{' {
			continue
		}
		var event struct {
			Type     string `json:"type"`
			ThreadID string `json:"thread_id"`
		}
		if json.Unmarshal(line, &event) != nil {
			continue
		}
		if event.Type == "thread.started" && event.ThreadID != "" {
			return event.ThreadID
		}
	}
	return ""
}
