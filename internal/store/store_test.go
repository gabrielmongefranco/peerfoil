// This file is part of PeerFoil.
// internal/store/store_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Tests the journal: appends, sequence continuity across reopen, interrupted
// invocations, a truncated final line, and key separation between repository clones.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testStore opens a store rooted in a temporary cache directory.
func testStore(t *testing.T, repoRoot string) *Store {
	t.Helper()
	t.Setenv("LOCALAPPDATA", filepath.Join(t.TempDir(), "cache")) // Windows cache base
	t.Setenv("XDG_CACHE_HOME", filepath.Join(t.TempDir(), "xdg")) // Linux cache base
	t.Setenv("HOME", t.TempDir())                                 // macOS fallback
	st, err := Open("example-project", repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	st.SetClock(func() time.Time { return time.Date(2026, 10, 21, 1, 0, 0, 0, time.UTC) })
	return st
}

func TestAppendAndReopenContinuesSequence(t *testing.T) {
	repo := t.TempDir()
	st := testStore(t, repo)
	if _, err := st.Intent("producer", "inv-0001", map[string]string{"task": "tk-001"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Result("producer", "inv-0001", StatusSucceeded, nil); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open("example-project", repo)
	if err != nil {
		t.Fatal(err)
	}
	entry, err := reopened.Append(Entry{Kind: "intent", Operation: "evidence"})
	if err != nil {
		t.Fatal(err)
	}
	if entry.Seq != 3 {
		t.Errorf("sequence %d after reopen, want 3", entry.Seq)
	}
}

func TestOpenInvocations(t *testing.T) {
	st := testStore(t, t.TempDir())
	st.Intent("producer", "inv-0001", nil)
	st.Started("producer", "inv-0001")
	st.Intent("evidence", "inv-0002", nil)
	st.Result("evidence", "inv-0002", StatusSucceeded, nil)
	open, err := st.OpenInvocations()
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 1 || open[0].InvocationID != "inv-0001" || open[0].LastStatus != StatusStarted {
		t.Errorf("open invocations %+v", open)
	}
}

func TestTruncatedFinalLineIsDropped(t *testing.T) {
	st := testStore(t, t.TempDir())
	st.Intent("producer", "inv-0001", nil)
	st.Result("producer", "inv-0001", StatusSucceeded, nil)
	// A crash mid-write leaves a torn last line.
	file, err := os.OpenFile(filepath.Join(st.Dir, "journal.jsonl"), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(`{"seq":3,"kind":"res`); err != nil {
		t.Fatal(err)
	}
	file.Close()
	entries, err := st.Entries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("%d entries, want the 2 complete ones", len(entries))
	}
}

func TestDifferentClonesGetDifferentDirectories(t *testing.T) {
	first, err := Dir("example-project", `C:\work\clone-one`)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Dir("example-project", `C:\work\clone-two`)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Error("two clones share one state directory")
	}
	if !strings.Contains(filepath.ToSlash(first), "peerfoil/projects/example-project-") {
		t.Errorf("unexpected layout %s", first)
	}
}
