// This file is part of PeerFoil.
// internal/fsx/read_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Verifies bounded reads and rejection of traversal and symlink parents.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package fsx

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReadRegularFileBoundsAndLinks(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "safe.txt"), []byte("safe"), 0o644); err != nil {
		t.Fatal(err)
	}
	if raw, err := ReadRegularFile(root, "safe.txt", 4); err != nil || string(raw) != "safe" {
		t.Fatal("regular bounded read failed")
	}
	for _, name := range []string{"safe.txt", "../safe.txt", "."} {
		if _, err := ReadRegularFile(root, name, 3); err == nil {
			t.Fatal("unsafe bounded read succeeded")
		}
	}
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("synthetic private data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		if runtime.GOOS != "windows" {
			t.Fatal("native symlink creation failed")
		}
		t.Skip("native symlink creation requires a permission this Windows host does not provide; bounded-read checks passed")
	}
	if _, err := ReadRegularFile(root, filepath.Join("linked", "secret.txt"), 100); err == nil {
		t.Fatal("a symlinked parent exposed an unrelated file")
	}
}
