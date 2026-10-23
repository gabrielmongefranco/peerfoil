// This file is part of PeerFoil.
// internal/project/discover.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Finds the repository root by walking up the directory tree, with no
// dependency on Git being installed, a home directory, or symlink support.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package project

import (
	"os"
	"path/filepath"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

// FindRoot walks from dir toward the filesystem root and returns the first directory
// holding a .peerfoil directory, or failing that the first holding a .git entry (a
// directory, or the file a linked worktree uses). Status must work offline, so this
// never runs Git.
func FindRoot(dir string) (string, error) {
	current, err := filepath.Abs(dir)
	if err != nil {
		return "", errs.Wrap(errs.Configuration, "cannot resolve the working directory", err)
	}
	current = filepath.Clean(current)
	gitFallback := ""
	for {
		if info, err := os.Stat(filepath.Join(current, ".peerfoil")); err == nil && info.IsDir() {
			return current, nil
		}
		if _, err := os.Lstat(filepath.Join(current, ".git")); err == nil && gitFallback == "" {
			gitFallback = current
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	if gitFallback != "" {
		return gitFallback, nil
	}
	return "", errs.New(errs.Configuration,
		"no PeerFoil project or Git repository was found here or in any parent directory",
		"change to your project's directory, or create a repository with git init")
}
