// This file is part of PeerFoil.
// internal/errs/errs_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Tests the error category mapping, exit codes, wrapping, and context errors.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package errs

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestExitCodesAreStableAndDistinct(t *testing.T) {
	want := map[Category]int{
		Usage: 2, Configuration: 3, Authentication: 4, Validation: 5, Policy: 6,
		Provider: 7, Git: 8, Evidence: 9, Timeout: 10, Cancelled: 11, Internal: 12,
	}
	seen := map[int]Category{}
	for category, code := range want {
		got := ExitCode(New(category, "x", ""))
		if got != code {
			t.Errorf("ExitCode(%s) = %d, want %d", category, got, code)
		}
		if other, dup := seen[got]; dup {
			t.Errorf("exit code %d used by both %s and %s", got, other, category)
		}
		seen[got] = category
	}
}

func TestNilIsSuccess(t *testing.T) {
	if got := ExitCode(nil); got != 0 {
		t.Errorf("ExitCode(nil) = %d, want 0", got)
	}
}

func TestUncategorizedIsInternal(t *testing.T) {
	err := errors.New("plain")
	if got := CategoryOf(err); got != Internal {
		t.Errorf("CategoryOf(plain) = %s, want internal", got)
	}
}

func TestContextErrorsMap(t *testing.T) {
	if got := CategoryOf(context.Canceled); got != Cancelled {
		t.Errorf("CategoryOf(Canceled) = %s, want cancelled", got)
	}
	if got := CategoryOf(fmt.Errorf("op: %w", context.DeadlineExceeded)); got != Timeout {
		t.Errorf("wrapped DeadlineExceeded = %s, want timeout", got)
	}
}

func TestWrapKeepsCauseAndNext(t *testing.T) {
	cause := errors.New("disk full")
	err := Wrap(Git, "cannot create the worktree", cause)
	if !errors.Is(err, cause) {
		t.Error("Wrap must keep the cause reachable with errors.Is")
	}
	withNext := New(Configuration, "no project record", "run peerfoil init")
	if got := NextStep(withNext); got != "run peerfoil init" {
		t.Errorf("NextStep = %q", got)
	}
	if got := NextStep(cause); got != "" {
		t.Errorf("NextStep(plain) = %q, want empty", got)
	}
}
