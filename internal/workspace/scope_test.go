// This file is part of PeerFoil.
// internal/workspace/scope_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Table-driven tests for the pure scope checker and the secret scan. Every
// secret in this file is synthetic, including the AWS documentation example key.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package workspace

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

func TestCheckScope(t *testing.T) {
	tests := []struct {
		name     string
		changes  []Change
		patterns []string
		wantOK   bool
	}{
		{
			name:    "no changes pass any allowlist",
			changes: nil, patterns: nil, wantOK: true,
		},
		{
			name:     "double star matches nested paths",
			changes:  []Change{{Status: "A", Path: "src/a/b/c.go"}},
			patterns: []string{"src/**"},
			wantOK:   true,
		},
		{
			name:     "double star matches a direct child",
			changes:  []Change{{Status: "M", Path: "src/main.go"}},
			patterns: []string{"src/**"},
			wantOK:   true,
		},
		{
			name:     "double star matches zero components",
			changes:  []Change{{Status: "A", Path: "docs/readme.md"}},
			patterns: []string{"docs/**/*.md"},
			wantOK:   true,
		},
		{
			name:     "single star matches at the top level",
			changes:  []Change{{Status: "M", Path: "main.go"}},
			patterns: []string{"*.go"},
			wantOK:   true,
		},
		{
			name:     "single star does not cross directories",
			changes:  []Change{{Status: "M", Path: "src/main.go"}},
			patterns: []string{"*.go"},
			wantOK:   false,
		},
		{
			name:     "question mark matches one character inside a component",
			changes:  []Change{{Status: "M", Path: "src/v1.go"}},
			patterns: []string{"src/v?.go"},
			wantOK:   true,
		},
		{
			name:     "question mark does not cross directories",
			changes:  []Change{{Status: "M", Path: "src/a/x"}},
			patterns: []string{"src?a/x"},
			wantOK:   false,
		},
		{
			name:     "matching is case-insensitive",
			changes:  []Change{{Status: "M", Path: "SRC/File.GO"}},
			patterns: []string{"src/*.go"},
			wantOK:   true,
		},
		{
			name:     "parent traversal is rejected",
			changes:  []Change{{Status: "A", Path: "../escape.txt"}},
			patterns: []string{"**"},
			wantOK:   false,
		},
		{
			name:     "embedded traversal is rejected",
			changes:  []Change{{Status: "A", Path: "src/../other.txt"}},
			patterns: []string{"**"},
			wantOK:   false,
		},
		{
			name:     "absolute path is rejected",
			changes:  []Change{{Status: "A", Path: "/etc/hosts"}},
			patterns: []string{"**"},
			wantOK:   false,
		},
		{
			name:     "drive letter is rejected",
			changes:  []Change{{Status: "A", Path: "c:/windows/x.txt"}},
			patterns: []string{"**"},
			wantOK:   false,
		},
		{
			name:     "unc path is rejected",
			changes:  []Change{{Status: "A", Path: "//server/share/x.txt"}},
			patterns: []string{"**"},
			wantOK:   false,
		},
		{
			name:     "backslash path is rejected",
			changes:  []Change{{Status: "A", Path: "src\\main.go"}},
			patterns: []string{"**"},
			wantOK:   false,
		},
		{
			name:     "git directory is rejected in any case",
			changes:  []Change{{Status: "M", Path: ".GIT/config"}},
			patterns: []string{"**"},
			wantOK:   false,
		},
		{
			name:     "agents file is rejected in any case",
			changes:  []Change{{Status: "M", Path: "docs/agents.MD"}},
			patterns: []string{"**"},
			wantOK:   false,
		},
		{
			name:     "alternate data stream is rejected",
			changes:  []Change{{Status: "A", Path: "src/file.txt:stream"}},
			patterns: []string{"**"},
			wantOK:   false,
		},
		{
			name:     "path outside every pattern is rejected",
			changes:  []Change{{Status: "M", Path: "docs/readme.md"}},
			patterns: []string{"src/**", "*.go"},
			wantOK:   false,
		},
		{
			name:     "empty allowlist rejects everything",
			changes:  []Change{{Status: "M", Path: "src/main.go"}},
			patterns: nil,
			wantOK:   false,
		},
		{
			name:     "rename with both sides in scope passes",
			changes:  []Change{{Status: "R", Path: "src/new.go", OldPath: "src/old.go"}},
			patterns: []string{"src/**"},
			wantOK:   true,
		},
		{
			name:     "rename whose original side is out of scope is rejected",
			changes:  []Change{{Status: "R", Path: "src/new.go", OldPath: "docs/old.md"}},
			patterns: []string{"src/**"},
			wantOK:   false,
		},
		{
			name:     "rename whose new side is out of scope is rejected",
			changes:  []Change{{Status: "R", Path: "docs/new.md", OldPath: "src/old.go"}},
			patterns: []string{"src/**"},
			wantOK:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckScope(tt.changes, tt.patterns)
			if tt.wantOK && err != nil {
				t.Errorf("CheckScope = %v, want nil", err)
			}
			if !tt.wantOK {
				if err == nil {
					t.Fatal("CheckScope = nil, want a policy error")
				}
				if errs.CategoryOf(err) != errs.Policy {
					t.Errorf("category = %s, want %s", errs.CategoryOf(err), errs.Policy)
				}
			}
		})
	}
}

func TestCheckScopeListsOffendersAndCapsThem(t *testing.T) {
	err := CheckScope([]Change{{Status: "M", Path: "docs/readme.md"}}, []string{"src/**"})
	if err == nil || !strings.Contains(err.Error(), "docs/readme.md") {
		t.Errorf("error %v does not name the offending path", err)
	}

	var many []Change
	for i := 0; i < 25; i++ {
		many = append(many, Change{Status: "A", Path: fmt.Sprintf("docs/file-%02d.md", i)})
	}
	err = CheckScope(many, []string{"src/**"})
	if err == nil {
		t.Fatal("CheckScope = nil, want a policy error")
	}
	if !strings.Contains(err.Error(), "and 15 more") {
		t.Errorf("error %v does not cap the listed paths at 10", err)
	}
}

func TestScanSecrets(t *testing.T) {
	// Synthetic values only: the AWS key is Amazon's published documentation
	// example, and the rest are obviously fake.
	fakeAWSKey := "AKIAIOSFODNN7EXAMPLE"
	fakePassword := "correct-horse-battery"
	patch := strings.Join([]string{
		"diff --git a/config.py b/config.py",
		"+aws_access_key_id = " + fakeAWSKey,
		"+password = \"" + fakePassword + "\"",
		"+comment = \"nothing secret here\"",
	}, "\n")

	hits := ScanSecrets([]byte(patch))
	if len(hits) < 2 {
		t.Fatalf("ScanSecrets found %d hits, want at least 2: %v", len(hits), hits)
	}
	joined := strings.Join(hits, "\n")
	if !strings.Contains(joined, "AWS access key ID") || !strings.Contains(joined, "line 2") {
		t.Errorf("hits do not name the AWS key pattern and its line: %v", hits)
	}
	if !strings.Contains(joined, "credential assignment") || !strings.Contains(joined, "line 3") {
		t.Errorf("hits do not name the credential assignment and its line: %v", hits)
	}
	// The report must never carry the matched value itself.
	if strings.Contains(joined, fakeAWSKey) || strings.Contains(joined, fakePassword) {
		t.Errorf("a hit contains the planted secret value: %v", hits)
	}
}

func TestScanSecretsMorePatterns(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"private key block", "+-----BEGIN RSA PRIVATE KEY-----", "private key block"},
		{"github classic token", "+tok = ghp_" + strings.Repeat("a", 36), "GitHub token"},
		{"github fine-grained token", "+tok = github_pat_" + strings.Repeat("b", 30), "GitHub fine-grained token"},
		{"slack token", "+slack = xoxb-not-a-real-token", "Slack token"},
		{"jwt", "+jwt = eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjMifQ.sig", "JSON Web Token"},
		{"api key assignment", "+api_key: 'synthetic-value-123'", "credential assignment"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hits := ScanSecrets([]byte(tt.line + "\n"))
			if len(hits) == 0 {
				t.Fatalf("ScanSecrets found nothing in a line with a planted %s", tt.want)
			}
			if !strings.Contains(strings.Join(hits, "\n"), tt.want) {
				t.Errorf("hits %v do not name %q", hits, tt.want)
			}
		})
	}
}

func TestScanSecretsClean(t *testing.T) {
	if hits := ScanSecrets(nil); hits != nil {
		t.Errorf("ScanSecrets(nil) = %v, want nil", hits)
	}
	clean := strings.Join([]string{
		"diff --git a/main.go b/main.go",
		"+func main() {",
		"+\tfmt.Println(\"hello\")",
		"+}",
	}, "\n")
	if hits := ScanSecrets([]byte(clean)); hits != nil {
		t.Errorf("ScanSecrets on a clean patch = %v, want nil", hits)
	}
}

// TestRecordDirectoryIsAlwaysDenied pins the rule that no allowlist, however broad,
// can grant a producer a write under the .peerfoil project records.
func TestRecordDirectoryIsAlwaysDenied(t *testing.T) {
	changes := []Change{{Status: "M", Path: ".peerfoil/evidence/ev-0001.md"}}
	if err := CheckScope(changes, []string{"**"}); err == nil {
		t.Fatal("a change inside .peerfoil must be rejected even under a ** allowlist")
	}
	mixedCase := []Change{{Status: "A", Path: ".PeerFoil/plan.json"}}
	if err := CheckScope(mixedCase, []string{"**"}); err == nil {
		t.Fatal("the .peerfoil denial must be case-insensitive")
	}
}
