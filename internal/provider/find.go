// This file is part of PeerFoil.
// internal/provider/find.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Finds the Claude Code and Codex programs on the PATH or inside the IDE
// extension folders that bundle them (D-0020). Paths are re-detected per run and never
// written into project records, because a home-directory path names the user.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package provider

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
)

// FindCodex returns candidate Codex CLI paths, best first: the PATH entry, then the
// newest bundled copies inside the Codex IDE extensions.
func FindCodex() []string {
	return findProgram("codex", codexGlobs())
}

// FindClaude returns candidate Claude Code paths, best first.
func FindClaude() []string {
	return findProgram("claude", claudeGlobs())
}

func findProgram(name string, globs []string) []string {
	var candidates []string
	if onPath, err := exec.LookPath(name); err == nil {
		candidates = append(candidates, onPath)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return candidates
	}
	var found []string
	for _, pattern := range globs {
		matches, err := filepath.Glob(filepath.Join(home, pattern))
		if err != nil {
			continue
		}
		found = append(found, matches...)
	}
	// Extension folders carry the version in the path, so a descending sort puts the
	// newest build first.
	sort.Sort(sort.Reverse(sort.StringSlice(found)))
	return append(candidates, found...)
}

func codexGlobs() []string {
	platform := map[string]string{
		"windows": "windows-x86_64/codex.exe",
		"darwin":  "darwin-*/codex",
		"linux":   "linux-*/codex",
	}[runtime.GOOS]
	if platform == "" {
		return nil
	}
	var globs []string
	for _, ide := range []string{".vscode", ".vscode-insiders", ".cursor"} {
		globs = append(globs, filepath.Join(ide, "extensions", "openai.chatgpt-*", "bin", platform))
	}
	return globs
}

func claudeGlobs() []string {
	binary := "claude"
	if runtime.GOOS == "windows" {
		binary = "claude.exe"
	}
	globs := []string{
		filepath.Join(".local", "bin", binary),
		filepath.Join(".claude", "local", binary),
	}
	if runtime.GOOS == "windows" {
		globs = append(globs, filepath.Join("AppData", "Roaming", "Claude", "claude-code", "*", "claude.exe"))
	}
	for _, ide := range []string{".vscode", ".vscode-insiders", ".cursor"} {
		globs = append(globs, filepath.Join(ide, "extensions", "anthropic.claude-code-*", "resources", "native-binary", binary))
	}
	return globs
}
