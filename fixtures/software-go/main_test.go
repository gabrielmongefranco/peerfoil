package main

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTemp writes content to a file inside the test's temporary directory
// and returns the file's path.
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.txt")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

// TestCountNonEmpty checks the counting logic against files that hold blank
// lines, whitespace-only lines, indented lines, and a trailing newline.
func TestCountNonEmpty(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    int
	}{
		{"blank lines between text", "one\n\ntwo\n\n\nthree\n", 3},
		{"spaces and tabs only do not count", " \t\n\t \nword\n", 1},
		{"leading spaces still count", "  indented\n\talso indented\n", 2},
		{"trailing newline adds no line", "last\n", 1},
		{"empty file counts zero", "", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeTemp(t, tc.content)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read temp file: %v", err)
			}
			if got := countNonEmpty(string(data)); got != tc.want {
				t.Errorf("countNonEmpty = %d, want %d", got, tc.want)
			}
		})
	}
}
