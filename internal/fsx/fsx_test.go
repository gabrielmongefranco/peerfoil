// This file is part of PeerFoil.
// internal/fsx/fsx_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Tests atomic replacement: new files, overwrites, no temporary leftovers,
// and failure on a missing directory.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package fsx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteCreatesAndReplaces(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "record.json")
	if err := WriteFileAtomic(path, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(path, []byte("two"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "two" {
		t.Errorf("content %q, want %q", data, "two")
	}
}

func TestNoTemporaryLeftovers(t *testing.T) {
	dir := t.TempDir()
	if err := WriteFileAtomic(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".peerfoil-write-") {
			t.Errorf("temporary file %s was left behind", entry.Name())
		}
	}
	if len(entries) != 1 {
		t.Errorf("expected exactly one file, found %d", len(entries))
	}
}

func TestMissingDirectoryFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "a.txt")
	if err := WriteFileAtomic(path, []byte("x"), 0o644); err == nil {
		t.Error("expected an error for a missing directory")
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	if Exists(filepath.Join(dir, "nope")) {
		t.Error("missing path reported as existing")
	}
	path := filepath.Join(dir, "yes")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !Exists(path) {
		t.Error("existing path reported as missing")
	}
}
