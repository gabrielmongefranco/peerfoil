// This file is part of PeerFoil.
// internal/workspace/scope.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Pure path-scope checking against allowlist patterns and a secret scan over
// captured patches. Both run without Git so they can be tested and audited in
// isolation, and both fail closed.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package workspace

import (
	"bytes"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

// ### Scope Checking ###

// maxReportedViolations caps how many offending paths one error message lists, so a
// runaway change set still produces a readable message.
const maxReportedViolations = 10

// CheckScope verifies that every changed path, including the original side of a
// rename, is safe and matches at least one allowed pattern. Matching is
// case-insensitive; '*' and '?' never cross '/', and '**' matches zero or more path
// components. An empty allowlist rejects every change: scope is deny-by-default.
func CheckScope(changes []Change, allowedPatterns []string) error {
	patterns := make([][]string, 0, len(allowedPatterns))
	for _, p := range allowedPatterns {
		p = strings.ToLower(strings.ReplaceAll(p, "\\", "/"))
		if p == "" {
			continue
		}
		patterns = append(patterns, strings.Split(p, "/"))
	}

	var violations []string
	seen := make(map[string]bool)
	check := func(changed string) {
		if changed == "" || seen[changed] {
			return
		}
		seen[changed] = true
		if reason := unsafePathReason(changed); reason != "" {
			violations = append(violations, fmt.Sprintf("%s (%s)", changed, reason))
			return
		}
		if !matchesAny(changed, patterns) {
			violations = append(violations, fmt.Sprintf("%s (outside the allowed paths)", changed))
		}
	}
	for _, c := range changes {
		check(c.Path)
		check(c.OldPath)
	}
	if len(violations) == 0 {
		return nil
	}
	extra := ""
	if len(violations) > maxReportedViolations {
		extra = fmt.Sprintf("; and %d more", len(violations)-maxReportedViolations)
		violations = violations[:maxReportedViolations]
	}
	return errs.New(errs.Policy,
		fmt.Sprintf("the change touches paths outside the task's allowed scope: %s%s",
			strings.Join(violations, "; "), extra),
		"revise the plan to allow these paths, or remove the out-of-scope changes")
}

// unsafePathReason explains why a changed path can never be accepted, or returns an
// empty string for a safe path. Git reports repository-relative paths with forward
// slashes; anything else, and anything that could escape the repository, name Git's
// own data, alter agent instructions, or address an NTFS alternate data stream, is
// rejected regardless of the allowlist.
func unsafePathReason(p string) string {
	if strings.Contains(p, "\\") {
		return "contains a backslash"
	}
	if strings.HasPrefix(p, "/") {
		// A single leading slash is an absolute path; a double one is a UNC path.
		return "absolute path"
	}
	for i, comp := range strings.Split(p, "/") {
		switch {
		case comp == "":
			return "empty path component"
		case comp == "." || comp == "..":
			return "path traversal component"
		case strings.EqualFold(comp, ".git"):
			return "inside the .git directory"
		case strings.EqualFold(comp, "AGENTS.md"):
			return "agent instruction file"
		case strings.Contains(comp, ":"):
			// Covers drive letters ("c:") in the first component and NTFS
			// alternate data streams ("file.txt:stream") anywhere.
			if i == 0 && len(comp) == 2 && comp[1] == ':' {
				return "drive letter"
			}
			return "colon in path component"
		}
	}
	return ""
}

// matchesAny reports whether the path matches at least one pre-split, lowercased
// pattern.
func matchesAny(p string, patterns [][]string) bool {
	segs := strings.Split(strings.ToLower(p), "/")
	for _, pat := range patterns {
		if matchComponents(pat, segs) {
			return true
		}
	}
	return false
}

// matchComponents matches pattern components against path components. A "**"
// component matches zero or more whole components; every other component matches one
// component with path.Match, whose '*' and '?' cannot cross '/' because a component
// contains none. A malformed pattern matches nothing, which keeps the check closed.
func matchComponents(pat, segs []string) bool {
	if len(pat) == 0 {
		return len(segs) == 0
	}
	if pat[0] == "**" {
		if matchComponents(pat[1:], segs) {
			return true
		}
		return len(segs) > 0 && matchComponents(pat, segs[1:])
	}
	if len(segs) == 0 {
		return false
	}
	ok, err := path.Match(pat[0], segs[0])
	if err != nil || !ok {
		return false
	}
	return matchComponents(pat[1:], segs[1:])
}

// ### Secret Scan ###

// secretPattern pairs a human-readable name with the expression that detects it. The
// name, never the matched text, is what reaches the user, so a real credential cannot
// leak through the report itself.
type secretPattern struct {
	name string
	re   *regexp.Regexp
}

var secretPatterns = []secretPattern{
	{"private key block", regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`)},
	{"AWS access key ID", regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
	{"GitHub token", regexp.MustCompile(`\bghp_[A-Za-z0-9]{36}\b`)},
	{"GitHub fine-grained token", regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{22,}\b`)},
	{"Slack token", regexp.MustCompile(`\bxox[baprs]-`)},
	{"credential assignment", regexp.MustCompile(`(?i)(password|passwd|secret|token|api[_-]?key)\s*[:=]\s*['"][^'"]{8,}`)},
	{"JSON Web Token", regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.`)},
}

// ScanSecrets looks through a captured patch for obvious credentials and returns one
// line per hit naming the pattern and the line number. The matched text itself is
// deliberately absent from the result: the report may be logged or shown, and a
// report must never become a second copy of the secret. A clean patch returns nil.
func ScanSecrets(patch []byte) []string {
	if len(patch) == 0 {
		return nil
	}
	var hits []string
	for i, line := range bytes.Split(patch, []byte("\n")) {
		for _, p := range secretPatterns {
			if p.re.Match(line) {
				hits = append(hits, fmt.Sprintf("possible %s on line %d", p.name, i+1))
			}
		}
	}
	return hits
}
