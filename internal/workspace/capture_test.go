// This file is part of PeerFoil.
// internal/workspace/capture_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Verifies immutable capture and rejection of tracked links before checkout.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package workspace

import (
	"context"
	"strings"
	"testing"
)

func TestLFSAttributesRejectedInBaseAndCapturedTrees(t *testing.T) {
	for _, attributes := range []string{".gitattributes", "nested/.gitattributes"} {
		t.Run(attributes, func(t *testing.T) {
			root := newTestRepo(t)
			repo := New(root)
			ctx := context.Background()
			base := runGit(t, root, "rev-parse", "HEAD")
			wt, err := repo.CreateTaskWorktree(ctx, base, "tk-001", 1, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, wt.Path, attributes, "*.bin filter=lfs diff=lfs merge=lfs -text\n")
			if _, _, err := wt.Capture(ctx); err == nil || !strings.Contains(err.Error(), "Git LFS") {
				t.Fatalf("capture accepted LFS attributes: %v", err)
			}
			writeFile(t, root, attributes, "[attr]binary filter=lfs\n*.bin binary\n")
			runGit(t, root, "add", "-A")
			runGit(t, root, "commit", "-m", "Synthetic unsupported attributes")
			if err := repo.DetectUnsupported(ctx); err == nil {
				t.Fatal("preflight accepted nested LFS attributes")
			}
			if _, err := repo.CreateReviewWorktree(ctx, runGit(t, root, "rev-parse", "HEAD"), "lfs-test", t.TempDir()); err == nil {
				t.Fatal("a review checkout was created with LFS")
			}
		})
	}
}

func TestLFSPointersFromUntrackedAttributesAreRejected(t *testing.T) {
	for _, config := range []string{"private", "global"} {
		t.Run(config, func(t *testing.T) {
			root := newTestRepo(t)
			repo := New(root)
			if config == "private" {
				writeFile(t, root, ".git/info/attributes", "*.bin filter=lfs\n")
			} else {
				attributes := t.TempDir()
				writeFile(t, attributes, "attributes", "*.bin filter=lfs\n")
				runGit(t, root, "config", "core.attributesFile", attributes+"/attributes")
			}
			wt, err := repo.CreateTaskWorktree(context.Background(), runGit(t, root, "rev-parse", "HEAD"), "tk-001", 1, t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, wt.Path, "image.bin", "version https://git-lfs.github.com/spec/v1\noid sha256:"+strings.Repeat("0", 64)+"\nsize 12345\n")
			if _, _, err := wt.Capture(context.Background()); err == nil || !strings.Contains(err.Error(), "LFS pointer") {
				t.Fatalf("local attribute pointer accepted: %v", err)
			}
		})
	}
}

func TestCommitUsesOnlyCapturedTree(t *testing.T) {
	root := newTestRepo(t)
	repo := New(root)
	ctx := context.Background()
	base := runGit(t, root, "rev-parse", "HEAD")
	wt, err := repo.CreateTaskWorktree(ctx, base, "tk-001", 1, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, wt.Path, "allowed.txt", "captured\n")
	changes, patch, err := wt.Capture(ctx)
	if err != nil || len(changes) != 1 || len(ScanSecrets(patch)) != 0 {
		t.Fatalf("capture: %v; changes %v", err, changes)
	}
	if err := CheckScope(changes, []string{"allowed.txt"}); err != nil {
		t.Fatal(err)
	}
	writeFile(t, wt.Path, "allowed.txt", "later edit\n")
	writeFile(t, wt.Path, "outside-scope.txt", "sk-proj-"+strings.Repeat("synthetic", 4)+"\n")
	runGit(t, wt.Path, "add", "-A")
	commit, err := wt.Commit(ctx, "Commit the captured change")
	if err != nil {
		t.Fatal(err)
	}
	if got := runGit(t, root, "show", commit+":allowed.txt"); got != "captured" {
		t.Fatal("a later edit changed the committed capture")
	}
	if got := runGit(t, root, "ls-tree", "--name-only", commit); strings.Contains(got, "outside-scope.txt") {
		t.Fatal("a later staged file entered the captured commit")
	}
	if got := runGit(t, root, "diff", "--binary", base, commit); got != strings.TrimSpace(string(patch)) {
		t.Fatal("the committed patch differs from the retained patch")
	}
	if _, err := repo.Integrate(ctx, wt, commit); err != nil {
		t.Fatal(err)
	}
}

func TestTrackedSymlinkRejectedWithoutOSLinkSupport(t *testing.T) {
	root := newTestRepo(t)
	repo := New(root)
	ctx := context.Background()
	// A staged Git link exercises the same modes on Windows without symlink privileges.
	writeFile(t, root, "link-target.txt", "../outside.txt\n")
	blob := runGit(t, root, "hash-object", "-w", "link-target.txt")
	runGit(t, root, "update-index", "--add", "--cacheinfo", "120000,"+blob+",escape")
	runGit(t, root, "commit", "-m", "fixture with a symbolic link")
	base := runGit(t, root, "rev-parse", "HEAD")
	if err := repo.DetectUnsupported(ctx); err == nil || !strings.Contains(err.Error(), "symbolic links") {
		t.Fatalf("preflight accepted a tracked link: %v", err)
	}
	if _, err := repo.CreateTaskWorktree(ctx, base, "tk-001", 1, t.TempDir()); err == nil {
		t.Fatal("a writable checkout was created from a tracked link")
	}
	if got := runGit(t, root, "branch", "--list", "peerfoil/task/*"); got != "" {
		t.Fatal("preflight rejection left a task branch")
	}
}
