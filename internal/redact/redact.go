// This file is part of PeerFoil.
// internal/redact/redact.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Replaces likely credential values in retained text with a fixed placeholder
// while keeping the surrounding text readable. Retained command output must stay useful
// for diagnosis without ever storing a working secret.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

// Package redact removes likely credential values from text before it is retained or
// shown. It replaces only the secret value, not the line around it, so a person reading
// retained output can still see which setting or header was involved. Redaction is a
// best-effort safety net, not an inventory of every secret format; callers must still
// keep known secrets out of the text they produce.
package redact

import (
	"bytes"
	"regexp"
)

// placeholder replaces every redacted value. Scrubbing already-scrubbed text therefore
// reproduces the same bytes, which keeps the operation idempotent.
const placeholder = "[redacted]"

// rules pair a credential pattern with its replacement. Order matters: structured
// formats (key blocks, Authorization headers, token prefixes) run before the generic
// key/value rule so the most specific match wins. Patterns use word boundaries so
// ordinary words such as "tokenizer" or "passwordless" are left alone.
var rules = []struct {
	pattern *regexp.Regexp
	replace []byte
}{
	// PEM private key blocks: the whole block is secret, including its BEGIN/END lines.
	{regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`), []byte(placeholder)},
	// HTTP Authorization headers: keep the header name, hide the bearer value.
	{regexp.MustCompile(`(?i)(authorization\s*:\s*bearer\s+)\S+`), []byte("${1}" + placeholder)},
	// JSON Web Tokens: three base64url segments; the first two encode JSON objects, so
	// both start with "eyJ" (base64 of `{"`), which keeps false positives rare.
	{regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{6,}\.eyJ[A-Za-z0-9_-]{6,}\.[A-Za-z0-9_-]{6,}`), []byte(placeholder)},
	// AWS access key identifiers.
	{regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`), []byte(placeholder)},
	// GitHub personal access tokens, classic and fine-grained.
	{regexp.MustCompile(`\bghp_[A-Za-z0-9]{16,}`), []byte(placeholder)},
	{regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{22,}`), []byte(placeholder)},
	// Slack bot, app, personal, refresh, and session tokens.
	{regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{8,}`), []byte(placeholder)},
	// Anthropic and OpenAI provider keys, which commands can print standalone with no
	// surrounding assignment for the generic rule to anchor on.
	{regexp.MustCompile(`\bsk-(?:ant-|proj-)?[A-Za-z0-9_-]{20,}\b`), []byte(placeholder)},
	// Generic assignments such as "password: ..." or "api_key=...": keep the key and
	// separator so the reader knows what was set, hide the value. The leading group
	// stands in for a word boundary that also stops at underscores, so "CLIENT_SECRET="
	// is caught while "tokenizer:" and "passwordless" are not. Values shorter than
	// eight characters are too short to be working credentials and stay visible.
	{regexp.MustCompile(`(?i)(^|[^a-z0-9])((?:password|passwd|secret|token|api[_-]?key)"?\s*[:=]\s*)(\S{8,})`), []byte("${1}${2}" + placeholder)},
}

// Scrub replaces the value of every likely credential in data with "[redacted]" and
// reports whether anything changed. Surrounding text is preserved. Scrubbing output
// that was already scrubbed changes nothing, so the reported flag means "this call
// removed something", not "a placeholder is present".
func Scrub(data []byte) ([]byte, bool) {
	if len(data) == 0 {
		return data, false
	}
	out := data
	for _, rule := range rules {
		out = rule.pattern.ReplaceAll(out, rule.replace)
	}
	if bytes.Equal(out, data) {
		return data, false
	}
	return out, true
}

// ScrubString is Scrub for strings.
func ScrubString(s string) (string, bool) {
	out, changed := Scrub([]byte(s))
	if !changed {
		return s, false
	}
	return string(out), true
}
