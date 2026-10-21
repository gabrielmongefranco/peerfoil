// This file is part of PeerFoil.
// internal/fsx/read.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Reads bounded regular files below a trusted directory while refusing
// symbolic links and detecting path replacement before exposing any bytes.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package fsx

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ReadRegularFile reads at most limit bytes from a relative file beneath root.
// It requires read permission, opens no external path intentionally, and rejects
// links, nonregular files, oversized content, and detectable path replacement.
// Errors omit paths and contents. Callers must still validate and scrub returned data.
func ReadRegularFile(root, relative string, limit int64) ([]byte, error) {
	invalid := fmt.Errorf("record must be a bounded regular file beneath its record directory")
	clean := filepath.Clean(relative)
	if limit < 1 || filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, invalid
	}
	paths := []string{root}
	for _, component := range strings.Split(clean, string(filepath.Separator)) {
		paths = append(paths, filepath.Join(paths[len(paths)-1], component))
	}
	infos := make([]os.FileInfo, len(paths))
	for i, path := range paths {
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || (i < len(paths)-1 && !info.IsDir()) || (i == len(paths)-1 && !info.Mode().IsRegular()) {
			return nil, invalid
		}
		infos[i] = info
	}
	file, err := os.Open(paths[len(paths)-1])
	if err != nil {
		return nil, invalid
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(infos[len(infos)-1], opened) || opened.Size() > limit {
		return nil, invalid
	}
	for i, path := range paths {
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !os.SameFile(infos[i], info) {
			return nil, invalid
		}
	}
	raw, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(raw)) > limit {
		return nil, invalid
	}
	return raw, nil
}
