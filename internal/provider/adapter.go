// This file is part of PeerFoil.
// internal/provider/adapter.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: The provider adapter contract shared by Claude Code, Codex, and test
// fakes, plus the model-lineage mapping that independent review depends on.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package provider

import (
	"context"
	"strings"
	"time"
)

// Sandbox values an invocation may request. Anything else is rejected rather
// than silently mapped, so a typo can never grant write access.
const (
	// SandboxReadOnly lets the model session read the workspace but not change it.
	SandboxReadOnly = "read-only"
	// SandboxWorkspaceWrite lets the model session edit files inside the
	// workspace. It is change isolation, not a security boundary.
	SandboxWorkspaceWrite = "workspace-write"
)

// Canonical lineage roots. Independent approval requires a reviewer whose root
// differs from the author's, and an unknown root means reduced assurance, so an
// unrecognized model identifier must never be guessed into a known family.
const (
	LineageAnthropicClaude = "anthropic-claude"
	LineageOpenAIGPT       = "openai-gpt"
	LineageUnknown         = "unknown"
)

// probeTimeout bounds the quick version and sign-in checks so a hung tool cannot
// stall startup diagnostics.
const probeTimeout = 30 * time.Second

// Seat names the tool, model, and effort one role runs with. Model and Effort may
// be empty or "default", which means the tool's own default and omits the flag.
type Seat struct {
	Tool   string
	Model  string
	Effort string
}

// Invocation is one fresh, non-interactive model session request.
type Invocation struct {
	// Role names the workflow role this session fills, for records only; it is
	// never sent to the tool.
	Role string
	// Packet is the complete instruction text, delivered on standard input so it
	// never appears in a process list or a shell history.
	Packet string
	// Sandbox is SandboxReadOnly or SandboxWorkspaceWrite.
	Sandbox string
	// Dir is the working directory the session runs in.
	Dir string
	// Timeout bounds the session. Zero means the runner's default.
	Timeout time.Duration
}

// Outcome is what one completed invocation produced.
type Outcome struct {
	// Text is the model's final answer.
	Text string
	// SessionID identifies the tool's own session, recorded for lineage and
	// resumability; it contains no conversation content.
	SessionID string
	ExitCode  int
	Duration  time.Duration
	// TimedOut is true when the session passed its soft limit and the text came
	// from a bounded wrap-up rather than a normal finish.
	TimedOut bool
}

// ProbeResult reports whether a tool is installed and signed in.
type ProbeResult struct {
	Found    bool
	Version  string
	LoggedIn bool
	// Detail is a short plain-language note safe to show and record; it carries
	// no paths or identifiers.
	Detail string
}

// RunFunc executes one bounded subprocess. Adapters accept one so tests can
// substitute a deterministic fake for the real runner.
type RunFunc func(context.Context, Request) (Result, error)

// Adapter is the contract every provider implements. Each Invoke starts a fresh
// session; an adapter never reuses one role's conversation for another role.
type Adapter interface {
	// Name identifies the tool in records and messages.
	Name() string
	// LineageRoot maps a model identifier to its canonical lineage root.
	LineageRoot(model string) string
	// Probe checks installation and sign-in without doing any work.
	Probe(ctx context.Context) ProbeResult
	// Invoke runs one fresh non-interactive session to completion.
	Invoke(ctx context.Context, inv Invocation) (Outcome, error)
}

// lineageRoot maps a model identifier to its canonical lineage root: "claude-*"
// is Anthropic; "gpt-*", "codex-*", and the o-series reasoning models ("o1",
// "o3", "o4", with or without a suffix such as "o3-mini") are OpenAI. Everything
// else, including an empty or "default" identifier, is unknown, because review
// independence must never rest on a guessed family.
func lineageRoot(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case m == "" || m == "default":
		return LineageUnknown
	case strings.HasPrefix(m, "claude-"):
		return LineageAnthropicClaude
	case strings.HasPrefix(m, "gpt-"), strings.HasPrefix(m, "codex-"):
		return LineageOpenAIGPT
	case m == "o1" || m == "o3" || m == "o4":
		return LineageOpenAIGPT
	case strings.HasPrefix(m, "o1-"), strings.HasPrefix(m, "o3-"), strings.HasPrefix(m, "o4-"):
		return LineageOpenAIGPT
	default:
		return LineageUnknown
	}
}
