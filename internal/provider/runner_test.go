// This file is part of PeerFoil.
// internal/provider/runner_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Tests for the bounded process runner using the exec-self helper
// process pattern, covering exit codes, stream separation, stdin, output caps,
// timeouts, cancellation, and the environment allowlist.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

// TestHelperProcess is not a real test: the runner tests re-execute this test
// binary with GO_WANT_HELPER_PROCESS=1 and a mode after "--", so subprocess
// behavior stays hermetic and identical on Windows, macOS, and Linux.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "helper: no mode")
		os.Exit(90)
	}
	switch args[0] {
	case "echo":
		fmt.Fprint(os.Stdout, "hello stdout")
		fmt.Fprint(os.Stderr, "hello stderr")
		os.Exit(0)
	case "exit":
		code, _ := strconv.Atoi(args[1])
		os.Exit(code)
	case "stdin":
		io.Copy(os.Stdout, os.Stdin)
		os.Exit(0)
	case "spew":
		n, _ := strconv.Atoi(args[1])
		os.Stdout.Write(bytes.Repeat([]byte("a"), n))
		os.Exit(0)
	case "sleep":
		time.Sleep(30 * time.Second)
		os.Exit(0)
	default:
		fmt.Fprintln(os.Stderr, "helper: unknown mode")
		os.Exit(91)
	}
}

// helperRequest builds a Request that re-executes this test binary in helper
// mode with the minimal environment plus the helper trigger variable.
func helperRequest(t *testing.T, mode string, extra ...string) Request {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("locating test binary: %v", err)
	}
	argv := append([]string{exe, "-test.run=TestHelperProcess", "--", mode}, extra...)
	return Request{
		Argv: argv,
		Dir:  t.TempDir(),
		Env:  append(BaseEnv(), "GO_WANT_HELPER_PROCESS=1"),
	}
}

func TestRunSuccessSeparatesStreams(t *testing.T) {
	res, err := Run(context.Background(), helperRequest(t, "echo"))
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", res.ExitCode)
	}
	if got := string(res.Stdout); got != "hello stdout" {
		t.Errorf("Stdout = %q, want %q", got, "hello stdout")
	}
	if got := string(res.Stderr); got != "hello stderr" {
		t.Errorf("Stderr = %q, want %q", got, "hello stderr")
	}
	if res.TimedOut || res.Truncated {
		t.Errorf("TimedOut = %v, Truncated = %v, want false, false", res.TimedOut, res.Truncated)
	}
	if res.Duration <= 0 {
		t.Errorf("Duration = %v, want > 0", res.Duration)
	}
}

func TestRunNonzeroExitIsNotAnError(t *testing.T) {
	res, err := Run(context.Background(), helperRequest(t, "exit", "3"))
	if err != nil {
		t.Fatalf("Run returned error for nonzero exit: %v", err)
	}
	if res.ExitCode != 3 {
		t.Errorf("ExitCode = %d, want 3", res.ExitCode)
	}
}

func TestRunStdinPassthrough(t *testing.T) {
	cases := []struct {
		name  string
		stdin string
	}{
		{"plain text", "hello from stdin"},
		{"empty input", ""},
		{"unicode quotes and newlines", "line one\nline \"two\" with 'quotes' and café ünïcode\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := helperRequest(t, "stdin")
			req.Stdin = []byte(tc.stdin)
			res, err := Run(context.Background(), req)
			if err != nil {
				t.Fatalf("Run returned error: %v", err)
			}
			if got := string(res.Stdout); got != tc.stdin {
				t.Errorf("Stdout = %q, want %q", got, tc.stdin)
			}
		})
	}
}

func TestRunOutputTruncation(t *testing.T) {
	cases := []struct {
		name          string
		writeBytes    int
		cap           int64
		wantLen       int
		wantTruncated bool
	}{
		{"over the cap keeps the prefix", 100_000, 1024, 1024, true},
		{"exactly at the cap is complete", 1024, 1024, 1024, false},
		{"under the cap is complete", 10, 1024, 10, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := helperRequest(t, "spew", strconv.Itoa(tc.writeBytes))
			req.MaxOutputBytes = tc.cap
			res, err := Run(context.Background(), req)
			if err != nil {
				t.Fatalf("Run returned error: %v", err)
			}
			if len(res.Stdout) != tc.wantLen {
				t.Errorf("len(Stdout) = %d, want %d", len(res.Stdout), tc.wantLen)
			}
			if res.Truncated != tc.wantTruncated {
				t.Errorf("Truncated = %v, want %v", res.Truncated, tc.wantTruncated)
			}
			if want := bytes.Repeat([]byte("a"), tc.wantLen); !bytes.Equal(res.Stdout, want) {
				t.Errorf("Stdout is not the prefix of the written output")
			}
		})
	}
}

func TestRunTimeoutKillsAndReports(t *testing.T) {
	req := helperRequest(t, "sleep")
	req.Timeout = 500 * time.Millisecond
	start := time.Now()
	res, err := Run(context.Background(), req)
	if err == nil {
		t.Fatal("Run returned no error on timeout")
	}
	if got := errs.CategoryOf(err); got != errs.Timeout {
		t.Errorf("error category = %q, want %q", got, errs.Timeout)
	}
	if !res.TimedOut {
		t.Error("TimedOut = false, want true")
	}
	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Errorf("run took %v; the child was not killed at the deadline", elapsed)
	}
}

func TestRunContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(300*time.Millisecond, cancel)
	defer timer.Stop()
	defer cancel()
	start := time.Now()
	res, err := Run(ctx, helperRequest(t, "sleep"))
	if err == nil {
		t.Fatal("Run returned no error on cancellation")
	}
	if got := errs.CategoryOf(err); got != errs.Cancelled {
		t.Errorf("error category = %q, want %q", got, errs.Cancelled)
	}
	if res.TimedOut {
		t.Error("TimedOut = true for a cancellation, want false")
	}
	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Errorf("run took %v; the child was not killed on cancellation", elapsed)
	}
}

func TestRunStartFailures(t *testing.T) {
	cases := []struct {
		name string
		req  Request
	}{
		{"empty argv", Request{}},
		{"empty executable name", Request{Argv: []string{""}}},
		{"missing executable", Request{Argv: []string{"peerfoil-no-such-tool-for-test"}, Dir: t.TempDir()}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Run(context.Background(), tc.req)
			if err == nil {
				t.Fatal("Run returned no error")
			}
			if got := errs.CategoryOf(err); got != errs.Provider {
				t.Errorf("error category = %q, want %q", got, errs.Provider)
			}
		})
	}
}

func TestRunErrorNamesOnlyExecutableBaseName(t *testing.T) {
	req := helperRequest(t, "sleep")
	req.Timeout = 500 * time.Millisecond
	req.Stdin = []byte("SECRET-STDIN-CONTENT")
	req.Argv = append(req.Argv, "SECRET-ARGUMENT")
	_, err := Run(context.Background(), req)
	if err == nil {
		t.Fatal("Run returned no error on timeout")
	}
	message := err.Error()
	for _, secret := range []string{"SECRET-STDIN-CONTENT", "SECRET-ARGUMENT"} {
		if strings.Contains(message, secret) {
			t.Errorf("error message contains %q: %q", secret, message)
		}
	}
}

func TestBaseEnvIsAnAllowlist(t *testing.T) {
	t.Setenv("FAKE_TOKEN", "super-secret-value")
	env := BaseEnv()
	var hasPath bool
	for _, entry := range env {
		if strings.HasPrefix(entry, "PATH=") {
			hasPath = true
		}
		if strings.HasPrefix(entry, "FAKE_TOKEN=") || strings.Contains(entry, "super-secret-value") {
			t.Errorf("BaseEnv leaked a non-allowlisted variable: %q", entry)
		}
	}
	if !hasPath {
		t.Error("BaseEnv does not contain PATH")
	}
}
