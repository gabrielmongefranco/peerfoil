// This file is part of PeerFoil.
// internal/redact/redact_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Tests for credential redaction: every supported pattern is replaced, the
// surrounding text survives, ordinary words are left alone, and scrubbing is idempotent.
// All secrets in this file are synthetic test values.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package redact

import (
	"strings"
	"testing"
)

// TestScrub covers each supported credential pattern and the words that must never be
// touched. Every input uses synthetic values only.
func TestScrub(t *testing.T) {
	t.Parallel()

	// Synthetic tokens shaped like the real formats.
	fakeGitHubClassic := "ghp_" + strings.Repeat("A1b2C3d4", 5)
	fakeGitHubFine := "github_pat_" + strings.Repeat("x9", 15)
	fakeSlack := "xoxb-1234567890-abcdefABCDEF"
	fakeAWS := "AKIAIOSFODNN7EXAMPLE"
	fakeJWT := "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.TJVA95OrM7E2cBab30RMHrHDcEfxjoYZgeFONFh7HgQ"
	fakeKeyBlock := "-----BEGIN RSA PRIVATE KEY-----\nMIIEfakefakefake\nfakefakefake\n-----END RSA PRIVATE KEY-----"

	tests := []struct {
		name        string
		in          string
		want        string
		wantChanged bool
	}{
		{
			name:        "empty input",
			in:          "",
			want:        "",
			wantChanged: false,
		},
		{
			name:        "private key block replaced whole",
			in:          "before\n" + fakeKeyBlock + "\nafter",
			want:        "before\n[redacted]\nafter",
			wantChanged: true,
		},
		{
			name:        "openssh private key block",
			in:          "-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n-----END OPENSSH PRIVATE KEY-----",
			want:        "[redacted]",
			wantChanged: true,
		},
		{
			name:        "aws access key id",
			in:          "using key " + fakeAWS + " for upload",
			want:        "using key [redacted] for upload",
			wantChanged: true,
		},
		{
			name:        "github classic token",
			in:          "remote: " + fakeGitHubClassic + " rejected",
			want:        "remote: [redacted] rejected",
			wantChanged: true,
		},
		{
			name:        "github fine-grained token",
			in:          "auth " + fakeGitHubFine + " expired",
			want:        "auth [redacted] expired",
			wantChanged: true,
		},
		{
			name:        "slack token",
			in:          "posting with " + fakeSlack + " failed",
			want:        "posting with [redacted] failed",
			wantChanged: true,
		},
		{
			name:        "bearer header keeps header name",
			in:          "Authorization: Bearer abc123def456 was sent",
			want:        "Authorization: Bearer [redacted] was sent",
			wantChanged: true,
		},
		{
			name:        "bearer header case-insensitive",
			in:          "authorization: bearer " + fakeJWT,
			want:        "authorization: bearer [redacted]",
			wantChanged: true,
		},
		{
			name:        "bare jwt",
			in:          "session " + fakeJWT + " expired",
			want:        "session [redacted] expired",
			wantChanged: true,
		},
		{
			name:        "generic password colon",
			in:          "password: hunter22222 accepted",
			want:        "password: [redacted] accepted",
			wantChanged: true,
		},
		{
			name:        "generic secret equals keeps separator",
			in:          "export CLIENT_secret=supersecret99",
			want:        "export CLIENT_secret=[redacted]",
			wantChanged: true,
		},
		{
			name:        "generic api_key",
			in:          "api_key = abcd1234efgh",
			want:        "api_key = [redacted]",
			wantChanged: true,
		},
		{
			name:        "generic api-key",
			in:          "api-key: abcd1234efgh",
			want:        "api-key: [redacted]",
			wantChanged: true,
		},
		{
			name:        "generic token in json",
			in:          `{"token": "abcdefgh1234"}`,
			want:        `{"token": [redacted]`,
			wantChanged: true,
		},
		{
			name:        "tokenizer is not a token",
			in:          "the tokenizer: splits words",
			want:        "the tokenizer: splits words",
			wantChanged: false,
		},
		{
			name:        "passwordless is not a password",
			in:          "passwordless = recommended-setup",
			want:        "passwordless = recommended-setup",
			wantChanged: false,
		},
		{
			name:        "short value stays visible",
			in:          "token: abc",
			want:        "token: abc",
			wantChanged: false,
		},
		{
			name:        "akia prefix inside longer word untouched",
			in:          "variable MYAKIA0123456789ABCDEF here",
			want:        "variable MYAKIA0123456789ABCDEF here",
			wantChanged: false,
		},
		{
			name:        "plain build output untouched",
			in:          "ok  \tgithub.com/example/pkg\t0.012s",
			want:        "ok  \tgithub.com/example/pkg\t0.012s",
			wantChanged: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, changed := Scrub([]byte(tt.in))
			if string(got) != tt.want {
				t.Errorf("Scrub() = %q, want %q", got, tt.want)
			}
			if changed != tt.wantChanged {
				t.Errorf("Scrub() changed = %v, want %v", changed, tt.wantChanged)
			}

			// Scrubbing a second time must change nothing more.
			again, changedAgain := Scrub(got)
			if string(again) != string(got) {
				t.Errorf("second Scrub() = %q, want unchanged %q", again, got)
			}
			if changedAgain {
				t.Errorf("second Scrub() reported a change on already-scrubbed text")
			}
		})
	}
}

// TestScrubString checks the string convenience wrapper against the byte version.
func TestScrubString(t *testing.T) {
	t.Parallel()

	got, changed := ScrubString("password: hunter22222")
	if got != "password: [redacted]" || !changed {
		t.Errorf("ScrubString() = %q, %v; want %q, true", got, changed, "password: [redacted]")
	}

	got, changed = ScrubString("nothing sensitive here")
	if got != "nothing sensitive here" || changed {
		t.Errorf("ScrubString() = %q, %v; want input unchanged, false", got, changed)
	}
}
