// This file is part of PeerFoil.
// internal/pack/pack_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Tests loading the built-in packs and mapping evidence names to commands.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package pack

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabrielmongefranco/peerfoil/internal/schema"
)

func schemas(t *testing.T) *schema.Set {
	t.Helper()
	set, err := schema.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func TestBuiltInPacksLoad(t *testing.T) {
	for _, id := range []string{"software", "generic", "documentation"} {
		loaded, err := Load(id, schemas(t))
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if loaded.ID != id {
			t.Errorf("pack id %q, want %q", loaded.ID, id)
		}
		if len(loaded.Evidence) == 0 {
			t.Errorf("%s declares no evidence", id)
		}
	}
}

func TestUnknownPackFails(t *testing.T) {
	if _, err := Load("imaginary", schemas(t)); err == nil {
		t.Error("an unknown pack must fail to load")
	}
}

func TestCommandForUsesMarkers(t *testing.T) {
	software, err := Load("software", schemas(t))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if software.CommandFor("unit-tests", root) != nil {
		t.Error("no marker file exists, so no command should match")
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	argv := software.CommandFor("unit-tests", root)
	if len(argv) == 0 || argv[0] != "go" {
		t.Errorf("unit-tests command %v, want the Go hint", argv)
	}
	// The returned slice is a copy; mutating it must not poison later calls.
	argv[0] = "mutated"
	if again := software.CommandFor("unit-tests", root); again[0] != "go" {
		t.Error("CommandFor returned a shared slice")
	}
	if software.CommandFor("no-such-evidence", root) != nil {
		t.Error("an undeclared evidence name must map to no command")
	}
}
