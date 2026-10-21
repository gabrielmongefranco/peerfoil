// This file is part of PeerFoil.
// internal/provider/claude_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Tests for the Claude Code adapter and the shared lineage mapping,
// using injected fake run functions instead of real processes.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package provider

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

// captureRun returns a RunFunc that records every Request and answers each call
// with the next scripted result and error.
func captureRun(calls *[]Request, results []Result, errors []error) RunFunc {
	index := 0
	return func(_ context.Context, req Request) (Result, error) {
		*calls = append(*calls, req)
		i := index
		index++
		var res Result
		if i < len(results) {
			res = results[i]
		}
		var err error
		if i < len(errors) {
			err = errors[i]
		}
		return res, err
	}
}

const claudeEnvelope = `{"result":"the answer","session_id":"sess-42"}`

func TestClaudeInvokeArgvConstruction(t *testing.T) {
	cases := []struct {
		name     string
		seat     Seat
		sandbox  string
		wantArgv []string
	}{
		{
			name:    "model and effort set, read-only",
			seat:    Seat{Model: "claude-opus-4", Effort: "high"},
			sandbox: SandboxReadOnly,
			wantArgv: []string{
				"claude", "-p", "--output-format", "json", "--max-turns", "10",
				"--model", "claude-opus-4", "--effort", "high",
				"--allowedTools", "Read,Glob,Grep",
			},
		},
		{
			name:    "empty model omits the model flag",
			seat:    Seat{Model: "", Effort: "medium"},
			sandbox: SandboxWorkspaceWrite,
			wantArgv: []string{
				"claude", "-p", "--output-format", "json", "--max-turns", "10",
				"--effort", "medium",
				"--permission-mode", "acceptEdits",
			},
		},
		{
			name:    "default model omits the model flag",
			seat:    Seat{Model: "default", Effort: "medium"},
			sandbox: SandboxWorkspaceWrite,
			wantArgv: []string{
				"claude", "-p", "--output-format", "json", "--max-turns", "10",
				"--effort", "medium",
				"--permission-mode", "acceptEdits",
			},
		},
		{
			name:    "empty effort omits the effort flag",
			seat:    Seat{Model: "claude-sonnet-4", Effort: ""},
			sandbox: SandboxReadOnly,
			wantArgv: []string{
				"claude", "-p", "--output-format", "json", "--max-turns", "10",
				"--model", "claude-sonnet-4",
				"--allowedTools", "Read,Glob,Grep",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls []Request
			run := captureRun(&calls, []Result{{Stdout: []byte(claudeEnvelope)}}, nil)
			adapter := NewClaude("claude", tc.seat, run)
			_, err := adapter.Invoke(context.Background(), Invocation{
				Packet: "do the task", Sandbox: tc.sandbox, Dir: "work", Timeout: time.Minute,
			})
			if err != nil {
				t.Fatalf("Invoke returned error: %v", err)
			}
			if len(calls) != 1 {
				t.Fatalf("run called %d times, want 1", len(calls))
			}
			if !reflect.DeepEqual(calls[0].Argv, tc.wantArgv) {
				t.Errorf("Argv = %q, want %q", calls[0].Argv, tc.wantArgv)
			}
			if got := string(calls[0].Stdin); got != "do the task" {
				t.Errorf("Stdin = %q, want the packet", got)
			}
			if calls[0].Dir != "work" {
				t.Errorf("Dir = %q, want %q", calls[0].Dir, "work")
			}
			if calls[0].Timeout != time.Minute {
				t.Errorf("Timeout = %v, want %v", calls[0].Timeout, time.Minute)
			}
		})
	}
}

func TestClaudeInvokeSandboxDifferences(t *testing.T) {
	invoke := func(sandbox string) ([]Request, error) {
		var calls []Request
		run := captureRun(&calls, []Result{{Stdout: []byte(claudeEnvelope)}}, nil)
		adapter := NewClaude("claude", Seat{}, run)
		_, err := adapter.Invoke(context.Background(), Invocation{Packet: "p", Sandbox: sandbox})
		return calls, err
	}

	readOnlyCalls, err := invoke(SandboxReadOnly)
	if err != nil {
		t.Fatalf("read-only Invoke returned error: %v", err)
	}
	readOnlyArgs := strings.Join(readOnlyCalls[0].Argv, " ")
	if !strings.Contains(readOnlyArgs, "--allowedTools Read,Glob,Grep") {
		t.Errorf("read-only argv %q lacks the read-only tool allowlist", readOnlyArgs)
	}
	if strings.Contains(readOnlyArgs, "--permission-mode") {
		t.Errorf("read-only argv %q grants edit permissions", readOnlyArgs)
	}

	writeCalls, err := invoke(SandboxWorkspaceWrite)
	if err != nil {
		t.Fatalf("workspace-write Invoke returned error: %v", err)
	}
	writeArgs := strings.Join(writeCalls[0].Argv, " ")
	if !strings.Contains(writeArgs, "--permission-mode acceptEdits") {
		t.Errorf("workspace-write argv %q lacks the edit permission mode", writeArgs)
	}
	if strings.Contains(writeArgs, "--allowedTools") {
		t.Errorf("workspace-write argv %q restricts tools like read-only", writeArgs)
	}
}

func TestClaudeInvokeRejectsUnknownSandbox(t *testing.T) {
	for _, sandbox := range []string{"", "full-access", "Read-Only"} {
		t.Run("sandbox "+sandbox, func(t *testing.T) {
			var calls []Request
			adapter := NewClaude("claude", Seat{}, captureRun(&calls, nil, nil))
			_, err := adapter.Invoke(context.Background(), Invocation{Packet: "p", Sandbox: sandbox})
			if err == nil {
				t.Fatal("Invoke accepted an unsupported sandbox")
			}
			if got := errs.CategoryOf(err); got != errs.Validation {
				t.Errorf("error category = %q, want %q", got, errs.Validation)
			}
			if len(calls) != 0 {
				t.Errorf("run was called %d times for a rejected sandbox, want 0", len(calls))
			}
		})
	}
}

func TestClaudeInvokeParsesEnvelope(t *testing.T) {
	var calls []Request
	run := captureRun(&calls, []Result{{Stdout: []byte(claudeEnvelope), Duration: 2 * time.Second}}, nil)
	adapter := NewClaude("claude", Seat{}, run)
	outcome, err := adapter.Invoke(context.Background(), Invocation{Packet: "p", Sandbox: SandboxReadOnly})
	if err != nil {
		t.Fatalf("Invoke returned error: %v", err)
	}
	if outcome.Text != "the answer" {
		t.Errorf("Text = %q, want %q", outcome.Text, "the answer")
	}
	if outcome.SessionID != "sess-42" {
		t.Errorf("SessionID = %q, want %q", outcome.SessionID, "sess-42")
	}
	if outcome.ExitCode != 0 || outcome.Duration != 2*time.Second {
		t.Errorf("ExitCode = %d, Duration = %v; want 0 and 2s", outcome.ExitCode, outcome.Duration)
	}
}

func TestClaudeInvokeMalformedOutput(t *testing.T) {
	const packet = "SECRET-PACKET-TEXT"
	cases := []struct {
		name   string
		stdout string
	}{
		{"not JSON at all", "claude exploded"},
		{"empty output", ""},
		{"missing result field", `{"session_id":"sess-42"}`},
		{"missing session_id field", `{"result":"the answer"}`},
		{"JSON array instead of object", `[1,2,3]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls []Request
			run := captureRun(&calls, []Result{{Stdout: []byte(tc.stdout)}}, nil)
			adapter := NewClaude("claude", Seat{}, run)
			_, err := adapter.Invoke(context.Background(), Invocation{Packet: packet, Sandbox: SandboxReadOnly})
			if err == nil {
				t.Fatal("Invoke accepted malformed output")
			}
			if got := errs.CategoryOf(err); got != errs.Provider {
				t.Errorf("error category = %q, want %q", got, errs.Provider)
			}
			if !strings.Contains(err.Error(), "Claude Code") {
				t.Errorf("error %q does not name the tool", err.Error())
			}
			if strings.Contains(err.Error(), packet) {
				t.Errorf("error %q contains the packet text", err.Error())
			}
		})
	}
}

func TestClaudeInvokeNonzeroExit(t *testing.T) {
	var calls []Request
	run := captureRun(&calls, []Result{{ExitCode: 7, Stdout: []byte("partial")}}, nil)
	adapter := NewClaude("claude", Seat{}, run)
	_, err := adapter.Invoke(context.Background(), Invocation{Packet: "p", Sandbox: SandboxReadOnly})
	if err == nil {
		t.Fatal("Invoke accepted a nonzero exit")
	}
	if got := errs.CategoryOf(err); got != errs.Provider {
		t.Errorf("error category = %q, want %q", got, errs.Provider)
	}
}

func TestClaudeInvokePropagatesRunErrors(t *testing.T) {
	runErr := errs.New(errs.Timeout, "the run passed its limit", "")
	var calls []Request
	adapter := NewClaude("claude", Seat{}, captureRun(&calls, []Result{{}}, []error{runErr}))
	_, err := adapter.Invoke(context.Background(), Invocation{Packet: "p", Sandbox: SandboxReadOnly})
	if got := errs.CategoryOf(err); got != errs.Timeout {
		t.Errorf("error category = %q, want %q", got, errs.Timeout)
	}
}

func TestClaudeProbe(t *testing.T) {
	cases := []struct {
		name    string
		results []Result
		errors  []error
		want    ProbeResult
	}{
		{
			name:    "installed",
			results: []Result{{Stdout: []byte("2.1.260 (Claude Code)\n")}},
			want:    ProbeResult{Found: true, Version: "2.1.260 (Claude Code)", LoggedIn: true},
		},
		{
			name:    "not installed",
			results: []Result{{}},
			errors:  []error{errs.New(errs.Provider, "could not start claude", "")},
			want:    ProbeResult{Found: false, LoggedIn: false},
		},
		{
			name:    "version check fails",
			results: []Result{{ExitCode: 1}},
			want:    ProbeResult{Found: false, LoggedIn: false},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls []Request
			adapter := NewClaude("claude", Seat{}, captureRun(&calls, tc.results, tc.errors))
			got := adapter.Probe(context.Background())
			if got.Found != tc.want.Found || got.Version != tc.want.Version || got.LoggedIn != tc.want.LoggedIn {
				t.Errorf("Probe = %+v, want Found=%v Version=%q LoggedIn=%v",
					got, tc.want.Found, tc.want.Version, tc.want.LoggedIn)
			}
			if got.Detail == "" {
				t.Error("Probe Detail is empty; it should explain the state")
			}
			if len(calls) != 1 || !reflect.DeepEqual(calls[0].Argv, []string{"claude", "--version"}) {
				t.Errorf("probe argv = %v, want one [claude --version] call", calls)
			}
		})
	}
}

func TestLineageRoot(t *testing.T) {
	adapter := NewClaude("claude", Seat{}, nil)
	cases := []struct {
		model string
		want  string
	}{
		{"claude-opus-4", LineageAnthropicClaude},
		{"claude-sonnet-4-5", LineageAnthropicClaude},
		{"gpt-5", LineageOpenAIGPT},
		{"codex-mini", LineageOpenAIGPT},
		{"o1", LineageOpenAIGPT},
		{"o3", LineageOpenAIGPT},
		{"o3-mini", LineageOpenAIGPT},
		{"o4", LineageOpenAIGPT},
		{"o4-mini", LineageOpenAIGPT},
		{"", LineageUnknown},
		{"default", LineageUnknown},
		{"o2", LineageUnknown},
		{"llama-3", LineageUnknown},
		{"mystery-model", LineageUnknown},
	}
	for _, tc := range cases {
		if got := adapter.LineageRoot(tc.model); got != tc.want {
			t.Errorf("LineageRoot(%q) = %q, want %q", tc.model, got, tc.want)
		}
	}
	codex := NewCodex("codex", Seat{}, nil)
	fake := &Fake{}
	if codex.LineageRoot("claude-opus-4") != LineageAnthropicClaude || fake.LineageRoot("gpt-5") != LineageOpenAIGPT {
		t.Error("adapters disagree on the shared lineage mapping")
	}
}
