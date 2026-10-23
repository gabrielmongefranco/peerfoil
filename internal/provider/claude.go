// This file is part of PeerFoil.
// internal/provider/claude.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Adapter that runs Claude Code non-interactively and parses its JSON
// result envelope.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

// claudeOptions groups every Claude Code command-line option this adapter uses,
// so a tool upgrade that renames or retires an option is a one-place edit. These
// are the documented non-interactive options of Claude Code 2.1.260, the version
// this release was tested against.
var claudeOptions = struct {
	version        []string
	base           []string
	model          string
	effort         string
	readOnly       []string
	workspaceWrite []string
}{
	version: []string{"--version"},
	// -p runs one prompt without an interactive session; the JSON output format
	// gives a parseable envelope; the turn cap keeps a stuck session bounded.
	base:   []string{"-p", "--output-format", "json", "--max-turns", "10"},
	model:  "--model",
	effort: "--effort",
	// Only read tools are allowed, so a read-only session cannot edit or run
	// anything in the workspace.
	readOnly: []string{"--allowedTools", "Read,Glob,Grep"},
	// Edits inside the workspace are accepted without a prompt; anything wider
	// still requires permission.
	workspaceWrite: []string{"--permission-mode", "acceptEdits"},
}

// Claude runs the Claude Code command-line tool in non-interactive mode. One
// value serves one seat; every Invoke starts a fresh session.
type Claude struct {
	exe  string
	seat Seat
	run  RunFunc
}

var _ Adapter = (*Claude)(nil)

// NewClaude builds a Claude Code adapter for one seat. exe is the executable
// path or command name. run may be nil, which uses the real process runner.
func NewClaude(exe string, seat Seat, run RunFunc) *Claude {
	if run == nil {
		run = Run
	}
	return &Claude{exe: exe, seat: seat, run: run}
}

// Name identifies the tool in records and messages.
func (c *Claude) Name() string { return "claude-code" }

// LineageRoot maps a model identifier to its canonical lineage root.
func (c *Claude) LineageRoot(model string) string { return lineageRoot(model) }

// Probe checks that Claude Code answers a version call. Claude Code manages its
// own sign-in and offers no separate signed-in check, so a found installation is
// reported as signed in and a stale login surfaces on first use instead.
func (c *Claude) Probe(ctx context.Context) ProbeResult {
	argv := append([]string{c.exe}, claudeOptions.version...)
	res, err := c.run(ctx, Request{Argv: argv, Timeout: probeTimeout})
	if err != nil || res.ExitCode != 0 {
		return ProbeResult{Detail: "Claude Code did not answer a version check; install it or add it to PATH"}
	}
	return ProbeResult{
		Found:    true,
		Version:  strings.TrimSpace(string(res.Stdout)),
		LoggedIn: true,
		Detail:   "Claude Code manages its own sign-in; a signed-out installation fails at first use",
	}
}

// Invoke runs one fresh non-interactive Claude Code session. The packet goes to
// the tool on standard input so it never appears in a process list, and the
// result is parsed from the tool's single JSON envelope on standard output.
func (c *Claude) Invoke(ctx context.Context, inv Invocation) (Outcome, error) {
	argv := append([]string{c.exe}, claudeOptions.base...)
	if model := c.seat.Model; model != "" && model != "default" {
		argv = append(argv, claudeOptions.model, model)
	}
	if effort := c.seat.Effort; effort != "" {
		argv = append(argv, claudeOptions.effort, effort)
	}
	switch inv.Sandbox {
	case SandboxReadOnly:
		argv = append(argv, claudeOptions.readOnly...)
	case SandboxWorkspaceWrite:
		argv = append(argv, claudeOptions.workspaceWrite...)
	default:
		// Deny by default: an unrecognized sandbox must never run with the
		// tool's own, wider default permissions.
		return Outcome{}, errs.New(errs.Validation,
			fmt.Sprintf("sandbox mode %q is not supported", inv.Sandbox),
			`use "read-only" or "workspace-write"`)
	}

	res, err := c.run(ctx, Request{
		Argv:    argv,
		Dir:     inv.Dir,
		Stdin:   []byte(inv.Packet),
		Timeout: inv.Timeout,
	})
	if err != nil {
		return Outcome{}, err
	}
	if res.ExitCode != 0 {
		return Outcome{}, errs.New(errs.Provider,
			fmt.Sprintf("Claude Code exited with status %d before returning a result", res.ExitCode),
			"check that Claude Code is signed in and that the selected model and effort exist")
	}

	// Pointer fields distinguish a missing key from an empty value, so a
	// changed envelope fails loudly instead of returning empty work.
	var envelope struct {
		Result    *string `json:"result"`
		SessionID *string `json:"session_id"`
	}
	if parseErr := json.Unmarshal(res.Stdout, &envelope); parseErr != nil || envelope.Result == nil || envelope.SessionID == nil {
		return Outcome{}, errs.New(errs.Provider,
			"Claude Code output shape was not recognized: expected one JSON object with result and session_id",
			"check that the installed Claude Code version matches the versions this release was tested with")
	}
	return Outcome{
		Text:      *envelope.Result,
		SessionID: *envelope.SessionID,
		ExitCode:  res.ExitCode,
		Duration:  res.Duration,
		TimedOut:  res.TimedOut,
	}, nil
}
