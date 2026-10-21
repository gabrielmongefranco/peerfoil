// This file is part of PeerFoil.
// internal/provider/find_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Tests the version ordering that puts the newest bundled CLI first when
// several IDE extension folders carry one.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package provider

import (
	"sort"
	"strings"
	"testing"
)

func TestNewerVersionPathOrdersNumerically(t *testing.T) {
	paths := []string{
		`C:\u\.vscode\extensions\openai.chatgpt-0.9\bin\windows-x86_64\codex.exe`,
		`C:\u\.vscode\extensions\openai.chatgpt-0.10\bin\windows-x86_64\codex.exe`,
		`C:\u\.vscode\extensions\openai.chatgpt-0.4.16\bin\windows-x86_64\codex.exe`,
	}
	sort.SliceStable(paths, func(a, b int) bool { return newerVersionPath(paths[a], paths[b]) })
	// Text order would wrongly place 0.9 before 0.10.
	if want := "openai.chatgpt-0.10"; !strings.Contains(paths[0], want) {
		t.Errorf("first candidate %q does not carry %s", paths[0], want)
	}
	if want := "openai.chatgpt-0.4.16"; !strings.Contains(paths[2], want) {
		t.Errorf("last candidate %q does not carry %s", paths[2], want)
	}
}

func TestPathVersionReadsVersionComponents(t *testing.T) {
	version := pathVersion(`AppData\Roaming\Claude\claude-code\2.1.260\claude.exe`)
	if len(version) != 3 || version[0] != 2 || version[1] != 1 || version[2] != 260 {
		t.Errorf("pathVersion = %v, want [2 1 260]", version)
	}
	if pathVersion(`bin/codex`) != nil {
		t.Error("a path without a version must return nil")
	}
}
