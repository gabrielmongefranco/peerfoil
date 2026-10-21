// This file is part of PeerFoil.
// internal/cli/cli_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Tests the command dispatcher: help, version, unknown commands and options,
// working-directory selection, and stable exit codes.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"strings"
	"testing"
)

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr strings.Builder
	code := Run(context.Background(), args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestHelpListsEveryCommand(t *testing.T) {
	code, out, _ := run(t, "--help")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, name := range []string{"init", "doctor", "start", "status", "resume"} {
		if !strings.Contains(out, name) {
			t.Errorf("help does not mention %q", name)
		}
	}
	if !strings.Contains(out, "Exit codes") {
		t.Error("help does not document the exit codes")
	}
}

func TestNoArgumentsPrintsHelp(t *testing.T) {
	code, out, _ := run(t)
	if code != 0 || !strings.Contains(out, "Usage:") {
		t.Errorf("exit %d, output %q", code, out)
	}
}

func TestVersion(t *testing.T) {
	code, out, _ := run(t, "--version")
	if code != 0 || !strings.HasPrefix(out, "peerfoil ") {
		t.Errorf("exit %d, output %q", code, out)
	}
}

func TestUnknownCommand(t *testing.T) {
	code, _, errText := run(t, "frobnicate")
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
	if !strings.Contains(errText, "frobnicate") || !strings.Contains(errText, "Next step") {
		t.Errorf("stderr %q must name the command and a next step", errText)
	}
}

func TestUnknownGlobalOption(t *testing.T) {
	code, _, errText := run(t, "--frob", "status")
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
	if !strings.Contains(errText, "Next step") {
		t.Errorf("stderr %q has no next step", errText)
	}
}

func TestMissingWorkingDirectory(t *testing.T) {
	code, _, errText := run(t, "-C", "Z:\\does\\not\\exist-peerfoil-test", "status")
	if code != 3 {
		t.Errorf("exit %d, want 3 (configuration)", code)
	}
	if errText == "" {
		t.Error("stderr is empty")
	}
}

func TestCommandHelp(t *testing.T) {
	code, out, _ := run(t, "status", "--help")
	if code != 0 || !strings.Contains(out, "peerfoil status") {
		t.Errorf("exit %d, output %q", code, out)
	}
}
