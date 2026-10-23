// This file is part of PeerFoil.
// internal/cli/doctor_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Tests the doctor command's sandbox detection: an outside container counts
// as the isolation boundary, and its absence reports without failing.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInsideContainerReadsTheRuntimeMarkers(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, ".containerenv")
	if insideContainer([]string{marker}) {
		t.Error("a missing marker was reported as a container")
	}
	if err := os.WriteFile(marker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if !insideContainer([]string{marker}) {
		t.Error("an existing marker was not reported as a container")
	}
}

func TestSandboxCheckPrefersTheContainerBoundary(t *testing.T) {
	marker := filepath.Join(t.TempDir(), ".containerenv")
	if err := os.WriteFile(marker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	saved := containerMarkers
	containerMarkers = []string{marker}
	defer func() { containerMarkers = saved }()

	check := sandboxCheck(t.TempDir())
	if check.Status != "pass" || !strings.Contains(check.Detail, "inside a container") {
		t.Errorf("check = %+v, want a pass naming the container boundary", check)
	}
}

func TestSandboxCheckNeverFails(t *testing.T) {
	saved := containerMarkers
	containerMarkers = []string{filepath.Join(t.TempDir(), "absent")}
	defer func() { containerMarkers = saved }()

	check := sandboxCheck(t.TempDir())
	if check.Status == "fail" {
		t.Errorf("a missing sandbox must never fail doctor; check = %+v", check)
	}
}
