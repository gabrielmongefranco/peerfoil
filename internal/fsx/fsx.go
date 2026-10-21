// This file is part of PeerFoil.
// internal/fsx/fsx.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Atomic file replacement and small filesystem helpers. A failed write leaves
// either the old complete file or the new complete file, never a partial one (D-0033).
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

// Package fsx writes accepted files atomically where the operating system allows it:
// the content goes to a temporary file in the target's directory, is flushed to disk,
// and then renamed over the target. Go's os.Rename replaces an existing file on
// Windows, macOS, and Linux.
package fsx

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// WriteFileAtomic replaces path with data. The temporary file lives in the same
// directory, so the rename never crosses filesystems. On Windows, a reader holding the
// target open can make the rename fail with a sharing violation, so the rename is
// retried briefly before giving up.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".peerfoil-write-*")
	if err != nil {
		return fmt.Errorf("creating the temporary file: %w", err)
	}
	tempName := temp.Name()
	cleanup := func() {
		temp.Close()
		os.Remove(tempName)
	}
	if _, err := temp.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("writing the temporary file: %w", err)
	}
	if err := temp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("flushing the temporary file: %w", err)
	}
	if err := temp.Close(); err != nil {
		os.Remove(tempName)
		return fmt.Errorf("closing the temporary file: %w", err)
	}
	if err := os.Chmod(tempName, perm); err != nil {
		os.Remove(tempName)
		return fmt.Errorf("setting permissions: %w", err)
	}
	if err := renameWithRetry(tempName, path); err != nil {
		os.Remove(tempName)
		return fmt.Errorf("replacing %s: %w", filepath.Base(path), err)
	}
	syncDir(dir)
	return nil
}

func renameWithRetry(from, to string) error {
	var err error
	attempts := 1
	if runtime.GOOS == "windows" {
		attempts = 5
	}
	for try := 0; try < attempts; try++ {
		if try > 0 {
			time.Sleep(time.Duration(try) * 20 * time.Millisecond)
		}
		if err = os.Rename(from, to); err == nil {
			return nil
		}
	}
	return err
}

// syncDir flushes the directory entry so the rename survives a crash. Windows does
// not support opening a directory for sync, and the rename itself is already durable
// enough there, so the failure is ignored by design.
func syncDir(dir string) {
	handle, err := os.Open(dir)
	if err != nil {
		return
	}
	defer handle.Close()
	_ = handle.Sync()
}

// Exists reports whether the path exists at all; an unreadable path counts as existing
// so callers fail closed instead of overwriting something they cannot see.
func Exists(path string) bool {
	_, err := os.Lstat(path)
	return !errors.Is(err, os.ErrNotExist)
}
