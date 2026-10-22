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
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

var reviewNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,40}[a-z0-9])?$`)

// CreateReviewWorktree checks out the frozen commit in a detached worktree under
// parentDir. It creates no branch, so cleanup removes only the worktree itself.
func (r *Repo) CreateReviewWorktree(ctx context.Context, commit, name, parentDir string) (string, error) {
	if err := r.CheckTree(ctx, commit); err != nil {
		return "", err
	}
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
	// A crash between creation and the deferred cleanup leaves this fixed path
	// registered, and Git refuses to add a registered worktree again. A review
	// worktree is always detached and disposable, so a stale one is removed before
	// the retry — but only one this repository registers. A directory that merely
	// sits at the same path is someone else's data and is never deleted.
	if _, err := os.Stat(path); err == nil {
		if !r.registeredWorktree(ctx, path) {
			return "", errs.New(errs.Git,
				fmt.Sprintf("the review checkout path %s already exists and is not a worktree of this repository", path),
				"move or delete that directory yourself, then try again")
		}
		if _, err := git(ctx, r.Root, "worktree", "remove", "--force", path); err != nil {
			return "", errs.Wrap(errs.Git, "removing the stale review worktree", err)
		}
		_ = os.RemoveAll(path)
	}
	// Pruning also clears a registration whose directory is already gone.
	_, _ = git(ctx, r.Root, "worktree", "prune")
	if _, err := git(ctx, r.Root, "worktree", "add", "--detach", path, commit); err != nil {
		return "", errs.Wrap(errs.Git, "creating the review worktree", err)
	}
	return path, nil
}

// registeredWorktree reports whether the path is a worktree this repository
// registers. Git prints worktree paths with forward slashes, so both sides are
// normalized, and the comparison ignores case because the supported filesystems
// on Windows and macOS do.
func (r *Repo) registeredWorktree(ctx context.Context, path string) bool {
	out, err := git(ctx, r.Root, "worktree", "list", "--porcelain")
	if err != nil {
		return false
	}
	want := filepath.Clean(path)
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.HasPrefix(line, "worktree ") {
			continue
		}
		listed := filepath.Clean(filepath.FromSlash(strings.TrimSpace(strings.TrimPrefix(line, "worktree "))))
		if strings.EqualFold(listed, want) {
			return true
		}
	}
	return false
}

// IsAncestor reports whether one commit is an ancestor of another. Both arguments
// must already be verified commit hashes; with those, a nonzero exit means only that
// the first commit is not an ancestor of the second.
func (r *Repo) IsAncestor(ctx context.Context, ancestor, descendant string) bool {
	_, err := git(ctx, r.Root, "merge-base", "--is-ancestor", ancestor, descendant)
	return err == nil
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

// CheckUnchanged verifies that HEAD, the index, and non-ignored working files still
// match commit. It runs read-only Git checks and returns a safe error on any drift.
func (r *Repo) CheckUnchanged(ctx context.Context, commit string) error {
	invalid := errs.New(errs.Evidence, "an evidence command changed the frozen checkout", "route generated source changes through change intake and rerun validation")
	head, err := r.CommitOf(ctx, "HEAD")
	if err != nil || head != commit {
		return invalid
	}
	clean, _, err := r.IsClean(ctx)
	if err != nil || !clean {
		return invalid
	}
	return nil
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
