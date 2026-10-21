// This file is part of PeerFoil.
// internal/provider/codex_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Tests for the Codex CLI adapter using injected fake run functions,
// covering argv construction, event-stream parsing, the timeout wrap-up path,
// and temporary-file cleanup.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package provider

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

// answerPathOf finds the file path after --output-last-message in an argv.
func answerPathOf(t *testing.T, argv []string) string {
	t.Helper()
	for i, arg := range argv {
		if arg == "--output-last-message" && i+1 < len(argv) {
			return argv[i+1]
		}
	}
	t.Fatalf("argv %q has no --output-last-message value", argv)
	return ""
}

// writeAnswer simulates the Codex CLI writing its final message to the file
// named in the call's argv.
func writeAnswer(t *testing.T, argv []string, text string) {
	t.Helper()
	if err := os.WriteFile(answerPathOf(t, argv), []byte(text), 0o600); err != nil {
		t.Fatalf("writing the simulated answer file: %v", err)
	}
}

const codexStream = "ignore me, not JSON\n" +
	`{"type":"turn.started"}` + "\n" +
	`{"type":"thread.started","thread_id":"thread-123"}` + "\n" +
	`{"type":"item.completed","item":{"text":"working"}}` + "\n"

func TestCodexInvokeArgvConstruction(t *testing.T) {
	cases := []struct {
		name       string
		seat       Seat
		wantEffort string
		wantModel  string
	}{
		{"model and effort set", Seat{Model: "gpt-5-codex", Effort: "medium"}, "medium", "gpt-5-codex"},
		{"empty model and effort omit their flags", Seat{}, "", ""},
		{"default model and effort omit their flags", Seat{Model: "default", Effort: "default"}, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls []Request
			run := func(_ context.Context, req Request) (Result, error) {
				calls = append(calls, req)
				writeAnswer(t, req.Argv, "done")
				return Result{Stdout: []byte(codexStream)}, nil
			}
			adapter := NewCodex("codex", tc.seat, run)
			_, err := adapter.Invoke(context.Background(), Invocation{
				Packet: "do the task", Sandbox: SandboxWorkspaceWrite, Dir: "work", Timeout: time.Minute,
			})
			if err != nil {
				t.Fatalf("Invoke returned error: %v", err)
			}
			if len(calls) != 1 {
				t.Fatalf("run called %d times, want 1", len(calls))
			}
			argv := calls[0].Argv
			head := []string{"codex", "exec", "--sandbox", "workspace-write", "--skip-git-repo-check", "-C", "work", "--json", "--output-last-message"}
			for i, want := range head {
				if argv[i] != want {
					t.Fatalf("argv[%d] = %q, want %q (full argv %q)", i, argv[i], want, argv)
				}
			}
			rest := strings.Join(argv[len(head)+1:], " ")
			if tc.wantEffort == "" && strings.Contains(rest, "model_reasoning_effort") {
				t.Errorf("argv %q sets an effort that should be omitted", rest)
			}
			if tc.wantEffort != "" && !strings.Contains(rest, "-c model_reasoning_effort="+tc.wantEffort) {
				t.Errorf("argv %q lacks the effort setting", rest)
			}
			if tc.wantModel == "" && strings.Contains(rest, "-m ") {
				t.Errorf("argv %q sets a model that should be omitted", rest)
			}
			if tc.wantModel != "" && !strings.Contains(rest, "-m "+tc.wantModel) {
				t.Errorf("argv %q lacks the model flag", rest)
			}
			if argv[len(argv)-1] != "-" {
				t.Errorf("argv does not end with %q to read the prompt from stdin: %q", "-", argv)
			}
			if got := string(calls[0].Stdin); got != "do the task" {
				t.Errorf("Stdin = %q, want the packet", got)
			}
			if calls[0].Dir != "work" {
				t.Errorf("Dir = %q, want %q", calls[0].Dir, "work")
			}
		})
	}
}

func TestCodexInvokeRejectsUnknownSandbox(t *testing.T) {
	for _, sandbox := range []string{"", "danger-full-access"} {
		var calls []Request
		run := func(_ context.Context, req Request) (Result, error) {
			calls = append(calls, req)
			return Result{}, nil
		}
		adapter := NewCodex("codex", Seat{}, run)
		_, err := adapter.Invoke(context.Background(), Invocation{Packet: "p", Sandbox: sandbox})
		if err == nil {
			t.Fatalf("Invoke accepted sandbox %q", sandbox)
		}
		if got := errs.CategoryOf(err); got != errs.Validation {
			t.Errorf("sandbox %q: error category = %q, want %q", sandbox, got, errs.Validation)
		}
		if len(calls) != 0 {
			t.Errorf("sandbox %q: run was called %d times, want 0", sandbox, len(calls))
		}
	}
}

func TestCodexInvokeExtractsThreadIDAndAnswer(t *testing.T) {
	var answerPath string
	run := func(_ context.Context, req Request) (Result, error) {
		answerPath = answerPathOf(t, req.Argv)
		writeAnswer(t, req.Argv, "the final answer")
		return Result{Stdout: []byte(codexStream), Duration: 3 * time.Second}, nil
	}
	adapter := NewCodex("codex", Seat{}, run)
	outcome, err := adapter.Invoke(context.Background(), Invocation{Packet: "p", Sandbox: SandboxReadOnly, Dir: t.TempDir()})
	if err != nil {
		t.Fatalf("Invoke returned error: %v", err)
	}
	if outcome.Text != "the final answer" {
		t.Errorf("Text = %q, want %q", outcome.Text, "the final answer")
	}
	if outcome.SessionID != "thread-123" {
		t.Errorf("SessionID = %q, want %q", outcome.SessionID, "thread-123")
	}
	if outcome.TimedOut {
		t.Error("TimedOut = true for a normal finish, want false")
	}
	if _, statErr := os.Stat(answerPath); !os.IsNotExist(statErr) {
		t.Errorf("temporary answer file %q was not removed", answerPath)
	}
}

func TestCodexAnswerFileStaysOutsideWorkspace(t *testing.T) {
	workspace := t.TempDir()
	run := func(_ context.Context, req Request) (Result, error) {
		path := answerPathOf(t, req.Argv)
		if strings.HasPrefix(path, workspace) {
			t.Errorf("answer file %q is inside the workspace %q", path, workspace)
		}
		writeAnswer(t, req.Argv, "ok")
		return Result{Stdout: []byte(codexStream)}, nil
	}
	adapter := NewCodex("codex", Seat{}, run)
	if _, err := adapter.Invoke(context.Background(), Invocation{Packet: "p", Sandbox: SandboxReadOnly, Dir: workspace}); err != nil {
		t.Fatalf("Invoke returned error: %v", err)
	}
	entries, err := os.ReadDir(workspace)
	if err != nil {
		t.Fatalf("reading the workspace: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("the invocation left %d files in the workspace, want 0", len(entries))
	}
}

func TestCodexInvokeTimeoutThenNudge(t *testing.T) {
	timeoutErr := errs.New(errs.Timeout, "codex did not finish within 1m0s and was stopped", "")
	var calls []Request
	var nudgePath string
	run := func(_ context.Context, req Request) (Result, error) {
		calls = append(calls, req)
		if len(calls) == 1 {
			// The first call times out after streaming the thread.started event.
			return Result{Stdout: []byte(codexStream), TimedOut: true, Duration: time.Minute}, timeoutErr
		}
		nudgePath = answerPathOf(t, req.Argv)
		writeAnswer(t, req.Argv, "partial but honest answer")
		return Result{Duration: 10 * time.Second}, nil
	}
	adapter := NewCodex("codex", Seat{}, run)
	outcome, err := adapter.Invoke(context.Background(), Invocation{Packet: "p", Sandbox: SandboxReadOnly, Dir: "work", Timeout: time.Minute})
	if err != nil {
		t.Fatalf("Invoke returned error after a successful nudge: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("run called %d times, want 2", len(calls))
	}
	nudgeArgv := calls[1].Argv
	wantHead := []string{"codex", "exec", "resume", "thread-123", "--json", "--output-last-message"}
	for i, want := range wantHead {
		if nudgeArgv[i] != want {
			t.Fatalf("nudge argv[%d] = %q, want %q (full argv %q)", i, nudgeArgv[i], want, nudgeArgv)
		}
	}
	if nudgeArgv[len(nudgeArgv)-1] != codexNudgeMessage {
		t.Errorf("nudge argv does not end with the wrap-up message: %q", nudgeArgv)
	}
	if calls[1].Timeout != codexNudgeTimeout {
		t.Errorf("nudge timeout = %v, want %v", calls[1].Timeout, codexNudgeTimeout)
	}
	if outcome.Text != "partial but honest answer" {
		t.Errorf("Text = %q, want the nudge answer", outcome.Text)
	}
	if !outcome.TimedOut {
		t.Error("TimedOut = false, want true after a soft-limit wrap-up")
	}
	if outcome.SessionID != "thread-123" {
		t.Errorf("SessionID = %q, want %q", outcome.SessionID, "thread-123")
	}
	if outcome.Duration != time.Minute+10*time.Second {
		t.Errorf("Duration = %v, want the sum of both calls", outcome.Duration)
	}
	if _, statErr := os.Stat(nudgePath); !os.IsNotExist(statErr) {
		t.Errorf("nudge answer file %q was not removed", nudgePath)
	}
}

func TestCodexInvokeTimeoutWithoutThreadID(t *testing.T) {
	const packet = "SECRET-PACKET-TEXT"
	var calls []Request
	run := func(_ context.Context, req Request) (Result, error) {
		calls = append(calls, req)
		// The stream never reached thread.started before the deadline.
		return Result{Stdout: []byte(`{"type":"turn.started"}` + "\n"), TimedOut: true},
			errs.New(errs.Timeout, "codex did not finish within 1m0s and was stopped", "")
	}
	adapter := NewCodex("codex", Seat{}, run)
	_, err := adapter.Invoke(context.Background(), Invocation{Packet: packet, Sandbox: SandboxReadOnly})
	if err == nil {
		t.Fatal("Invoke returned no error for a timeout without a session to resume")
	}
	if got := errs.CategoryOf(err); got != errs.Timeout {
		t.Errorf("error category = %q, want %q", got, errs.Timeout)
	}
	if !strings.Contains(err.Error(), "passed its time limit") || !strings.Contains(err.Error(), "no result") {
		t.Errorf("error %q does not say the run passed its limit with no result", err.Error())
	}
	if strings.Contains(err.Error(), packet) {
		t.Errorf("error %q contains the packet text", err.Error())
	}
	if len(calls) != 1 {
		t.Errorf("run called %d times, want 1 (no nudge without a thread id)", len(calls))
	}
}

func TestCodexInvokeNudgeFailure(t *testing.T) {
	cases := []struct {
		name      string
		nudgeRes  Result
		nudgeErr  error
		wantCalls int
	}{
		{"nudge errors", Result{}, errs.New(errs.Timeout, "the nudge was stopped", ""), 2},
		{"nudge exits nonzero", Result{ExitCode: 1}, nil, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls []Request
			run := func(_ context.Context, req Request) (Result, error) {
				calls = append(calls, req)
				if len(calls) == 1 {
					return Result{Stdout: []byte(codexStream), TimedOut: true},
						errs.New(errs.Timeout, "codex did not finish within 1m0s and was stopped", "")
				}
				return tc.nudgeRes, tc.nudgeErr
			}
			adapter := NewCodex("codex", Seat{}, run)
			_, err := adapter.Invoke(context.Background(), Invocation{Packet: "p", Sandbox: SandboxReadOnly})
			if err == nil {
				t.Fatal("Invoke returned no error after a failed nudge")
			}
			if got := errs.CategoryOf(err); got != errs.Timeout {
				t.Errorf("error category = %q, want %q", got, errs.Timeout)
			}
			if len(calls) != tc.wantCalls {
				t.Errorf("run called %d times, want %d (one nudge, never more)", len(calls), tc.wantCalls)
			}
		})
	}
}

func TestCodexInvokeNonzeroExit(t *testing.T) {
	run := func(_ context.Context, req Request) (Result, error) {
		return Result{ExitCode: 2, Stdout: []byte(codexStream)}, nil
	}
	adapter := NewCodex("codex", Seat{}, run)
	_, err := adapter.Invoke(context.Background(), Invocation{Packet: "p", Sandbox: SandboxReadOnly})
	if err == nil {
		t.Fatal("Invoke accepted a nonzero exit")
	}
	if got := errs.CategoryOf(err); got != errs.Provider {
		t.Errorf("error category = %q, want %q", got, errs.Provider)
	}
}

func TestCodexThreadID(t *testing.T) {
	cases := []struct {
		name   string
		stream string
		want   string
	}{
		{"mixed stream", codexStream, "thread-123"},
		{"empty stream", "", ""},
		{"no thread.started event", `{"type":"turn.started"}` + "\n", ""},
		{"thread.started without an id", `{"type":"thread.started"}` + "\n", ""},
		{"truncated final line", `{"type":"thread.started","thread_id":"t-9"}` + "\n" + `{"type":"item.comp`, "t-9"},
		{"windows line endings", "{\"type\":\"thread.started\",\"thread_id\":\"t-crlf\"}\r\n", "t-crlf"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := codexThreadID([]byte(tc.stream)); got != tc.want {
				t.Errorf("codexThreadID = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCodexProbe(t *testing.T) {
	cases := []struct {
		name    string
		results []Result
		errors  []error
		want    ProbeResult
	}{
		{
			name: "installed and signed in",
			results: []Result{
				{Stdout: []byte("codex-cli 0.46.0\n")},
				{},
			},
			want: ProbeResult{Found: true, Version: "codex-cli 0.46.0", LoggedIn: true},
		},
		{
			name: "installed but signed out",
			results: []Result{
				{Stdout: []byte("codex-cli 0.46.0\n")},
				{ExitCode: 1},
			},
			want: ProbeResult{Found: true, Version: "codex-cli 0.46.0", LoggedIn: false},
		},
		{
			name:    "not installed",
			results: []Result{{}},
			errors:  []error{errs.New(errs.Provider, "could not start codex", "")},
			want:    ProbeResult{Found: false},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls []Request
			adapter := NewCodex("codex", Seat{}, captureRun(&calls, tc.results, tc.errors))
			got := adapter.Probe(context.Background())
			if got.Found != tc.want.Found || got.Version != tc.want.Version || got.LoggedIn != tc.want.LoggedIn {
				t.Errorf("Probe = %+v, want Found=%v Version=%q LoggedIn=%v",
					got, tc.want.Found, tc.want.Version, tc.want.LoggedIn)
			}
			if got.Detail == "" {
				t.Error("Probe Detail is empty; it should explain the state")
			}
			if tc.want.Found && len(calls) == 2 {
				wantLogin := []string{"codex", "login", "status"}
				for i, arg := range wantLogin {
					if calls[1].Argv[i] != arg {
						t.Errorf("login check argv = %q, want %q", calls[1].Argv, wantLogin)
						break
					}
				}
			}
		})
	}
}

func TestFakeAdapterScripts(t *testing.T) {
	fake := &Fake{
		NameValue: "scripted",
		Probes:    []ProbeResult{{Found: true, Version: "1"}},
		Outcomes:  []Outcome{{Text: "first"}, {}},
		Errs:      []error{nil, errs.New(errs.Provider, "scripted failure", "")},
	}
	if fake.Name() != "scripted" {
		t.Errorf("Name = %q, want %q", fake.Name(), "scripted")
	}
	if probe := fake.Probe(context.Background()); !probe.Found {
		t.Error("first Probe is not the scripted result")
	}
	if probe := fake.Probe(context.Background()); !probe.Found {
		t.Error("a re-check after the script ends should repeat the last probe")
	}
	first, err := fake.Invoke(context.Background(), Invocation{Role: "producer"})
	if err != nil || first.Text != "first" {
		t.Errorf("first Invoke = (%+v, %v), want the scripted outcome", first, err)
	}
	if _, err := fake.Invoke(context.Background(), Invocation{Role: "reviewer"}); err == nil {
		t.Error("second Invoke did not return the scripted error")
	}
	if _, err := fake.Invoke(context.Background(), Invocation{Role: "extra"}); errs.CategoryOf(err) != errs.Internal {
		t.Errorf("a call past the script should fail loudly, got %v", err)
	}
	if len(fake.Calls) != 3 || fake.Calls[0].Role != "producer" || fake.Calls[1].Role != "reviewer" {
		t.Errorf("Calls = %+v, want all three invocations recorded in order", fake.Calls)
	}
}
