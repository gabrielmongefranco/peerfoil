// This file is part of PeerFoil.
// internal/cli/paths.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Small path helpers for the command dispatcher.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import "path/filepath"

// absPath cleans and absolutizes a path without resolving symbolic links, because the
// default product path cannot rely on symlink support.
func absPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absolute), nil
}
