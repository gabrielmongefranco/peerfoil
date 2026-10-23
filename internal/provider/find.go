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
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
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
	// Extension folders carry the version in the path; a numeric comparison puts the
	// newest build first, which plain text ordering would not (0.9 sorts after 0.10).
	sort.SliceStable(found, func(a, b int) bool { return newerVersionPath(found[a], found[b]) })
	return append(candidates, found...)
}

// pathVersionPattern finds a dotted numeric version after a separator, such as the
// "2.1.260" in "anthropic.claude-code-2.1.260" or a version-named folder.
var pathVersionPattern = regexp.MustCompile(`[-/\\.]([0-9]+(?:\.[0-9]+)+)`)

// pathVersion returns the numeric segments of the last version named in a path, or
// nil when the path names none.
func pathVersion(path string) []int {
	matches := pathVersionPattern.FindAllStringSubmatch(path, -1)
	if len(matches) == 0 {
		return nil
	}
	parts := strings.Split(matches[len(matches)-1][1], ".")
	version := make([]int, len(parts))
	for index, part := range parts {
		version[index], _ = strconv.Atoi(part)
	}
	return version
}

// newerVersionPath reports whether path a names a newer version than path b, comparing
// each numeric segment; ties fall back to descending text order so the sort stays
// deterministic for paths without a version.
func newerVersionPath(a, b string) bool {
	va, vb := pathVersion(a), pathVersion(b)
	for index := 0; index < len(va) && index < len(vb); index++ {
		if va[index] != vb[index] {
			return va[index] > vb[index]
		}
	}
	if len(va) != len(vb) {
		return len(va) > len(vb)
	}
	return a > b
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
