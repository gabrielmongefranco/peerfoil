// This file is part of PeerFoil.
// internal/workspace/review.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: A read-only review worktree pinned to the exact frozen commit, so both
// reviewers inspect the same revision whatever the user's checkout looks like.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package workspace

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

var reviewNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,40}[a-z0-9])?$`)

// CreateReviewWorktree checks out the frozen commit in a detached worktree under
// parentDir. It creates no branch, so cleanup removes only the worktree itself.
func (r *Repo) CreateReviewWorktree(ctx context.Context, commit, name, parentDir string) (string, error) {
	if !reviewNamePattern.MatchString(name) {
		return "", errs.New(errs.Validation,
			fmt.Sprintf("%q is not a valid review worktree name", name), "")
	}
	// Resolve a symlinked parent so the recorded path matches what git lists,
	// exactly as CreateTaskWorktree does.
	if resolved, err := filepath.EvalSymlinks(parentDir); err == nil {
		parentDir = resolved
	}
	path := filepath.Join(parentDir, "review-"+name)
	if _, err := git(ctx, r.Root, "worktree", "add", "--detach", path, commit); err != nil {
		return "", errs.Wrap(errs.Git, "creating the review worktree", err)
	}
	return path, nil
}

// CommitOf resolves a branch or other reference to its commit hash.
func (r *Repo) CommitOf(ctx context.Context, ref string) (string, error) {
	out, err := git(ctx, r.Root, "rev-parse", "--verify", ref+"^{commit}")
	if err != nil {
		return "", errs.New(errs.Git,
			fmt.Sprintf("the reference %q does not name a commit", ref),
			"run peerfoil status; the integration branch appears after the first validated task")
	}
	return strings.TrimSpace(string(out)), nil
}

// TreeDigest returns a stable digest input for the exact tree at a commit: the full
// recursive listing of blob hashes and paths. Hashing this pins the reviewed content.
func (r *Repo) TreeDigest(ctx context.Context, commit string) ([]byte, error) {
	out, err := git(ctx, r.Root, "ls-tree", "-r", commit)
	if err != nil {
		return nil, errs.Wrap(errs.Git, "listing the frozen tree", err)
	}
	return out, nil
}

// RemoveReviewWorktree removes a worktree CreateReviewWorktree made. It refuses any
// path outside the recorded review naming, so it can never delete user work.
func (r *Repo) RemoveReviewWorktree(ctx context.Context, path string) error {
	if !strings.Contains(filepath.Base(path), "review-") {
		return errs.New(errs.Policy, "refusing to remove a worktree PeerFoil did not create for review", "")
	}
	if _, err := git(ctx, r.Root, "worktree", "remove", "--force", path); err != nil {
		return errs.Wrap(errs.Git, "removing the review worktree", err)
	}
	return nil
}
