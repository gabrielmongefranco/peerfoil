// This file is part of PeerFoil.
// internal/cli/init_status_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Tests peerfoil init (preview, creation, refusal to overwrite) and peerfoil
// status against a fixture and an initialized project.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newRepo creates a bare-bones repository marker so discovery succeeds without Git.
func newRepo(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "My Example-App")
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestInitPreviewCreatesNothing(t *testing.T) {
	dir := newRepo(t)
	code, out, _ := run(t, "-C", dir, "init")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, "project.json") || !strings.Contains(out, "Nothing was created") {
		t.Errorf("preview output %q", out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".peerfoil")); !os.IsNotExist(err) {
		t.Error("preview must not create .peerfoil")
	}
}

func TestInitCreatesALoadableProject(t *testing.T) {
	dir := newRepo(t)
	code, out, errText := run(t, "-C", dir, "init", "--yes", "--goal", "A tiny example application.")
	if code != 0 {
		t.Fatalf("exit %d, stderr %s", code, errText)
	}
	if !strings.Contains(out, "define state") {
		t.Errorf("output %q", out)
	}
	// The created records must load through the same validation the controller uses,
	// and status must present them.
	code, out, errText = run(t, "-C", dir, "status")
	if code != 0 {
		t.Fatalf("status exit %d, stderr %s", code, errText)
	}
	for _, fragment := range []string{"my-example-app", "define", "Guided", "Next:"} {
		if !strings.Contains(out, fragment) {
			t.Errorf("status output %q does not contain %q", out, fragment)
		}
	}
}

func TestInitRefusesToOverwrite(t *testing.T) {
	dir := newRepo(t)
	if code, _, _ := run(t, "-C", dir, "init", "--yes"); code != 0 {
		t.Fatal("first init failed")
	}
	code, _, errText := run(t, "-C", dir, "init", "--yes")
	if code != 3 {
		t.Errorf("exit %d, want 3 (configuration)", code)
	}
	if !strings.Contains(errText, "already has") {
		t.Errorf("stderr %q", errText)
	}
}

func TestInitRejectsBadOptions(t *testing.T) {
	dir := newRepo(t)
	if code, _, _ := run(t, "-C", dir, "init", "--profile", "corporate"); code != 2 {
		t.Error("bad profile must exit 2")
	}
	if code, _, _ := run(t, "-C", dir, "init", "--pack", "research"); code != 2 {
		t.Error("unknown pack must exit 2")
	}
}

func TestStatusOnFixture(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(wd, "..", "..", "fixtures", "software-go")
	code, out, errText := run(t, "-C", fixture, "status")
	if code != 0 {
		t.Fatalf("exit %d, stderr %s", code, errText)
	}
	for _, fragment := range []string{"peerfoil-fixture-go", "produce", "tk-001", "Guided"} {
		if !strings.Contains(out, fragment) {
			t.Errorf("status output %q does not contain %q", out, fragment)
		}
	}
}

func TestStatusOutsideAnyRepository(t *testing.T) {
	code, _, errText := run(t, "-C", t.TempDir(), "status")
	if code != 3 {
		t.Errorf("exit %d, want 3", code)
	}
	if !strings.Contains(errText, "Next step") {
		t.Errorf("stderr %q", errText)
	}
}
