// This file is part of PeerFoil.
// internal/store/store.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: The local write-ahead journal outside Git. An intent is recorded before an
// outside operation and its result afterward, so a restart can tell what was running.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

// Package store keeps reconstructible operational state in the operating system's user
// cache directory, keyed by project identity and repository location (D-0034). It holds
// no project truth: everything here can be discarded and rebuilt from Git and the
// accepted .peerfoil files, at the cost of asking the user about interrupted work.
package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

// Entry is one journal line. Intents carry Kind "intent", results Kind "result"; both
// name the operation and the invocation they belong to. Detail must already be
// redacted by the caller; the journal never holds prompts, tokens, or file contents.
type Entry struct {
	Seq          int64             `json:"seq"`
	At           string            `json:"at"`
	Kind         string            `json:"kind"`
	Operation    string            `json:"operation"`
	InvocationID string            `json:"invocation_id,omitempty"`
	Status       string            `json:"status,omitempty"`
	Detail       map[string]string `json:"detail,omitempty"`
}

// Invocation statuses. An invocation that is prepared or started when the journal ends
// is unknown after an interruption, and an unknown mutating invocation is never rerun
// automatically.
const (
	StatusPrepared  = "prepared"
	StatusStarted   = "started"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
	StatusUnknown   = "unknown"
)

// Store is the journal for one project in one repository location.
type Store struct {
	Dir     string
	path    string
	lastSeq int64
	now     func() time.Time
}

// Dir returns the state directory for a project without opening it, so doctor can
// report it. The key mixes the project identifier with a hash of the repository root,
// because the same project can be cloned to two places on one machine.
func Dir(projectID, repoRoot string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", errs.Wrap(errs.Configuration, "no user cache directory is available", err)
	}
	digest := sha256.Sum256([]byte(filepath.Clean(repoRoot)))
	key := fmt.Sprintf("%s-%s", projectID, hex.EncodeToString(digest[:])[:12])
	return filepath.Join(base, "peerfoil", "projects", key), nil
}

// Open creates the state directory when needed and reads the existing journal's last
// sequence number.
func Open(projectID, repoRoot string) (*Store, error) {
	dir, err := Dir(projectID, repoRoot)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, errs.Wrap(errs.Configuration, "cannot create the local state directory", err)
	}
	st := &Store{Dir: dir, path: filepath.Join(dir, "journal.jsonl"), now: time.Now}
	entries, err := st.Entries()
	if err != nil {
		return nil, err
	}
	if len(entries) > 0 {
		st.lastSeq = entries[len(entries)-1].Seq
	}
	return st, nil
}

// SetClock replaces the time source; tests use it for stable timestamps.
func (s *Store) SetClock(now func() time.Time) { s.now = now }

// Append writes one entry durably: the line is written and flushed before Append
// returns, so a crash after Append cannot lose it.
func (s *Store) Append(entry Entry) (Entry, error) {
	s.lastSeq++
	entry.Seq = s.lastSeq
	if entry.At == "" {
		entry.At = s.now().UTC().Format("2006-01-02T15:04:05Z")
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return Entry{}, errs.Wrap(errs.Internal, "encoding a journal entry", err)
	}
	file, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return Entry{}, errs.Wrap(errs.Configuration, "opening the journal", err)
	}
	defer file.Close()
	if _, err := file.Write(append(line, '\n')); err != nil {
		return Entry{}, errs.Wrap(errs.Configuration, "writing the journal", err)
	}
	if err := file.Sync(); err != nil {
		return Entry{}, errs.Wrap(errs.Configuration, "flushing the journal", err)
	}
	return entry, nil
}

// Entries reads the whole journal. A truncated final line, the sign of a crash during
// a write, is dropped with its loss noted in the returned entries' flow rather than
// failing the read, because the journal exists exactly to survive interruption.
func (s *Store) Entries() ([]Entry, error) {
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errs.Wrap(errs.Configuration, "reading the journal", err)
	}
	var entries []Entry
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// A damaged line ends the trustworthy prefix; everything before it
			// stands, and recovery treats the rest as unknown.
			break
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// Intent records that an outside operation is about to run. Record the result with
// Result before trusting anything the operation did.
func (s *Store) Intent(operation, invocationID string, detail map[string]string) (Entry, error) {
	return s.Append(Entry{Kind: "intent", Operation: operation, InvocationID: invocationID,
		Status: StatusPrepared, Detail: detail})
}

// Started marks the operation's process as actually launched.
func (s *Store) Started(operation, invocationID string) (Entry, error) {
	return s.Append(Entry{Kind: "intent", Operation: operation, InvocationID: invocationID,
		Status: StatusStarted})
}

// Result records how the operation ended.
func (s *Store) Result(operation, invocationID, status string, detail map[string]string) (Entry, error) {
	return s.Append(Entry{Kind: "result", Operation: operation, InvocationID: invocationID,
		Status: status, Detail: detail})
}

// OpenInvocation describes an invocation the journal left without a final result.
type OpenInvocation struct {
	Operation    string
	InvocationID string
	LastStatus   string
}

// OpenInvocations returns every invocation whose last journal entry is prepared or
// started. After an interruption these are unknown: the controller inspects the
// workspace and asks the user instead of rerunning a mutating one.
func (s *Store) OpenInvocations() ([]OpenInvocation, error) {
	entries, err := s.Entries()
	if err != nil {
		return nil, err
	}
	last := map[string]Entry{}
	var order []string
	for _, entry := range entries {
		if entry.InvocationID == "" {
			continue
		}
		if _, seen := last[entry.InvocationID]; !seen {
			order = append(order, entry.InvocationID)
		}
		last[entry.InvocationID] = entry
	}
	var open []OpenInvocation
	for _, id := range order {
		entry := last[id]
		if entry.Status == StatusPrepared || entry.Status == StatusStarted {
			open = append(open, OpenInvocation{Operation: entry.Operation,
				InvocationID: id, LastStatus: entry.Status})
		}
	}
	return open, nil
}
