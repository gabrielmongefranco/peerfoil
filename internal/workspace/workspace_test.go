// This file is part of PeerFoil.
// internal/workspace/workspace_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Tests for repository discovery, cleanliness, unsupported-feature
// detection, task worktrees, change capture, integration, and cleanup against real
// Git repositories created in temporary directories.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package workspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

var commitHashPattern = regexp.MustCompile(`^[0-9a-f]{40,64}$`)

// requireGit skips the test when the git executable is not installed, since every
// test in this file drives a real repository.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
}

// isolateGitEnv points git away from the developer's global and system configuration
// so results do not depend on the machine the tests run on.
func isolateGitEnv(t *testing.T) {
	t.Helper()
	missing := filepath.Join(t.TempDir(), "no-config")
	t.Setenv("GIT_CONFIG_GLOBAL", missing)
	t.Setenv("GIT_CONFIG_SYSTEM", missing)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_TERMINAL_PROMPT", "0")
}

// runGit runs a git command for test setup and fails the test on any error.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// writeFile writes content under dir, creating parent directories as needed.
func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newBareInit creates an empty repository with local identity configured but no
// commits yet. The returned path has symbolic links resolved so it compares equal to
// what git itself reports on macOS.
func newBareInit(t *testing.T) string {
	t.Helper()
	requireGit(t)
	isolateGitEnv(t)
	dir := t.TempDir()
	runGit(t, dir, "init", "--initial-branch=main", ".")
	runGit(t, dir, "config", "user.email", "test@example.invalid")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "config", "commit.gpgsign", "false")
	runGit(t, dir, "config", "core.autocrlf", "false")
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

// newTestRepo creates a repository with one initial commit containing README.md.
func newTestRepo(t *testing.T) string {
	t.Helper()
	dir := newBareInit(t)
	writeFile(t, dir, "README.md", "hello\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial commit")
	return dir
}

func TestDiscover(t *testing.T) {
	root := newTestRepo(t)
	sub := filepath.Join(root, "sub", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Discover(context.Background(), sub)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if !samePath(got, root) {
		t.Errorf("Discover returned a different root than the repository created for the test")
	}
}

func TestDiscoverOutsideRepository(t *testing.T) {
	requireGit(t)
	isolateGitEnv(t)
	dir := t.TempDir()
	// Stop git from walking above the temporary directory, in case the test machine
	// keeps its temporary files inside some repository.
	ceiling, err := filepath.EvalSymlinks(filepath.Dir(dir))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CEILING_DIRECTORIES", ceiling)
	_, err = Discover(context.Background(), dir)
	if err == nil {
		t.Fatal("Discover outside a repository succeeded; want an error")
	}
	if errs.CategoryOf(err) != errs.Git {
		t.Errorf("category = %s, want %s", errs.CategoryOf(err), errs.Git)
	}
	if !strings.Contains(errs.NextStep(err), "git init") {
		t.Errorf("next step %q does not mention git init", errs.NextStep(err))
	}
}

func TestHeadCommit(t *testing.T) {
	root := newTestRepo(t)
	r := New(root)
	hash, err := r.HeadCommit(context.Background())
	if err != nil {
		t.Fatalf("HeadCommit: %v", err)
	}
	if !commitHashPattern.MatchString(hash) {
		t.Errorf("HeadCommit returned %q, want a full commit hash", hash)
	}
}

func TestHeadCommitNoCommits(t *testing.T) {
	root := newBareInit(t)
	r := New(root)
	_, err := r.HeadCommit(context.Background())
	if err == nil {
		t.Fatal("HeadCommit on a repository with no commits succeeded; want an error")
	}
	if errs.CategoryOf(err) != errs.Git {
		t.Errorf("category = %s, want %s", errs.CategoryOf(err), errs.Git)
	}
	if !strings.Contains(err.Error(), "no commit") {
		t.Errorf("error %q does not say the repository has no commit", err)
	}
}

func TestIsClean(t *testing.T) {
	root := newTestRepo(t)
	r := New(root)
	ctx := context.Background()

	clean, dirty, err := r.IsClean(ctx)
	if err != nil {
		t.Fatalf("IsClean: %v", err)
	}
	if !clean || len(dirty) != 0 {
		t.Fatalf("fresh repository reported dirty: %v", dirty)
	}

	// Names with a space and with non-ASCII letters must come back intact, which is
	// what the NUL-delimited status form guarantees.
	writeFile(t, root, "spaced name.txt", "a\n")
	writeFile(t, root, "ünïcode-café.txt", "b\n")
	writeFile(t, root, "README.md", "changed\n")

	clean, dirty, err = r.IsClean(ctx)
	if err != nil {
		t.Fatalf("IsClean: %v", err)
	}
	if clean {
		t.Fatal("dirty repository reported clean")
	}
	if len(dirty) != 3 {
		t.Errorf("got %d dirty paths, want 3: %v", len(dirty), dirty)
	}
	joined := strings.Join(dirty, "\n")
	if !strings.Contains(joined, "spaced name.txt") {
		t.Errorf("dirty paths lost the name with a space: %v", dirty)
	}
	// Match on an ASCII substring so macOS Unicode normalization cannot confuse the
	// comparison; the point is that the path survived NUL decoding at all.
	if !strings.Contains(joined, "code-caf") {
		t.Errorf("dirty paths lost the non-ASCII name: %v", dirty)
	}
}

func TestDetectUnsupported(t *testing.T) {
	ctx := context.Background()

	t.Run("plain repository passes", func(t *testing.T) {
		r := New(newTestRepo(t))
		if err := r.DetectUnsupported(ctx); err != nil {
			t.Errorf("DetectUnsupported: %v", err)
		}
	})

	t.Run("submodules rejected", func(t *testing.T) {
		root := newTestRepo(t)
		writeFile(t, root, ".gitmodules", "[submodule \"x\"]\n\tpath = x\n\turl = https://example.invalid/x.git\n")
		err := New(root).DetectUnsupported(ctx)
		if err == nil || errs.CategoryOf(err) != errs.Validation {
			t.Fatalf("err = %v, want a validation error", err)
		}
		if !strings.Contains(err.Error(), "submodule") {
			t.Errorf("error %q does not name submodules", err)
		}
	})

	t.Run("git lfs rejected", func(t *testing.T) {
		root := newTestRepo(t)
		writeFile(t, root, ".gitattributes", "*.bin filter=lfs diff=lfs merge=lfs -text\r\n")
		err := New(root).DetectUnsupported(ctx)
		if err == nil || errs.CategoryOf(err) != errs.Validation {
			t.Fatalf("err = %v, want a validation error", err)
		}
		if !strings.Contains(err.Error(), "LFS") {
			t.Errorf("error %q does not name Git LFS", err)
		}
	})

	t.Run("gitattributes without lfs passes", func(t *testing.T) {
		root := newTestRepo(t)
		writeFile(t, root, ".gitattributes", "# filter=lfs appears only in this comment\n*.txt text\n")
		if err := New(root).DetectUnsupported(ctx); err != nil {
			t.Errorf("DetectUnsupported: %v", err)
		}
	})
}

func TestCreateTaskWorktreeValidation(t *testing.T) {
	root := newTestRepo(t)
	r := New(root)
	ctx := context.Background()
	base, err := r.HeadCommit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()

	badIDs := []string{"", "tk-1", "tk-01", "task-001", "tk-001x", "tk-001; rm -rf", "TK-001", "tk-001/../x"}
	for _, id := range badIDs {
		if _, err := r.CreateTaskWorktree(ctx, base, id, 1, parent); errs.CategoryOf(err) != errs.Validation {
			t.Errorf("task id %q: err = %v, want a validation error", id, err)
		}
	}

	if _, err := r.CreateTaskWorktree(ctx, base, "tk-001", 0, parent); errs.CategoryOf(err) != errs.Validation {
		t.Errorf("plan revision 0: err = %v, want a validation error", err)
	}
	if _, err := r.CreateTaskWorktree(ctx, base, "tk-001", 1, filepath.Join(root, "inside")); errs.CategoryOf(err) != errs.Validation {
		t.Errorf("parent inside repository: err = %v, want a validation error", err)
	}
	if _, err := r.CreateTaskWorktree(ctx, "0000000000000000000000000000000000000000", "tk-001", 1, parent); errs.CategoryOf(err) != errs.Git {
		t.Errorf("unknown base commit: err = %v, want a git error", err)
	}

	if _, err := r.CreateTaskWorktree(ctx, base, "tk-001", 1, parent); err != nil {
		t.Fatalf("first CreateTaskWorktree: %v", err)
	}
	if _, err := r.CreateTaskWorktree(ctx, base, "tk-001", 1, parent); errs.CategoryOf(err) != errs.Git {
		t.Errorf("duplicate branch: err = %v, want a git error", err)
	}
}

func TestWorktreeChangesAndPatch(t *testing.T) {
	root := newTestRepo(t)
	r := New(root)
	ctx := context.Background()

	writeFile(t, root, "modify me.txt", "original content\n")
	writeFile(t, root, "delete.txt", "going away\n")
	writeFile(t, root, "keep.txt", "stable content for rename detection\n")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-m", "add task fixtures")
	base, err := r.HeadCommit(ctx)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := r.CreateTaskWorktree(ctx, base, "tk-100", 1, t.TempDir())
	if err != nil {
		t.Fatalf("CreateTaskWorktree: %v", err)
	}
	if wt.Base != base {
		t.Errorf("worktree base = %q, want %q", wt.Base, base)
	}

	// The producer's edits: an addition in a new directory, a modification, a
	// deletion, a small binary file, and a staged rename.
	writeFile(t, wt.Path, "added dir/new file.txt", "fresh content\n")
	writeFile(t, wt.Path, "modify me.txt", "rewritten content\n")
	if err := os.Remove(filepath.Join(wt.Path, "delete.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt.Path, "blob.bin"), []byte{0x00, 0x01, 0xFF, 0x7F}, 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, wt.Path, "mv", "keep.txt", "renamed.txt")

	changes, err := wt.Changes(ctx)
	if err != nil {
		t.Fatalf("Changes: %v", err)
	}
	byPath := make(map[string]Change)
	for _, c := range changes {
		byPath[c.Path] = c
	}
	want := map[string]string{
		"added dir/new file.txt": "A",
		"modify me.txt":          "M",
		"delete.txt":             "D",
		"blob.bin":               "A",
		"renamed.txt":            "R",
	}
	for p, status := range want {
		got, ok := byPath[p]
		if !ok {
			t.Errorf("Changes is missing %q: %v", p, changes)
			continue
		}
		if got.Status != status {
			t.Errorf("Changes[%q].Status = %q, want %q", p, got.Status, status)
		}
	}
	if byPath["renamed.txt"].OldPath != "keep.txt" {
		t.Errorf("rename old path = %q, want keep.txt", byPath["renamed.txt"].OldPath)
	}

	// Capturing the patch must leave the worktree's real index untouched: the staged
	// rename and the unstaged edits must look exactly the same afterwards.
	statusBefore := runGit(t, wt.Path, "status", "--porcelain")
	patch, err := wt.Patch(ctx)
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	if len(patch) == 0 {
		t.Fatal("Patch returned no content for a worktree with changes")
	}
	if got := runGit(t, wt.Path, "status", "--porcelain"); got != statusBefore {
		t.Errorf("Patch changed the worktree's own status:\nbefore:\n%s\nafter:\n%s", statusBefore, got)
	}

	// The patch must reproduce the content when applied onto a second checkout of
	// the same base commit.
	applyDir := filepath.Join(t.TempDir(), "apply")
	runGit(t, root, "worktree", "add", "--detach", applyDir, base)
	patchFile := filepath.Join(t.TempDir(), "task.patch")
	if err := os.WriteFile(patchFile, patch, 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, applyDir, "apply", "--whitespace=nowarn", patchFile)

	added, err := os.ReadFile(filepath.Join(applyDir, "added dir", "new file.txt"))
	if err != nil || string(added) != "fresh content\n" {
		t.Errorf("applied patch did not reproduce the added file: %q, %v", added, err)
	}
	modified, err := os.ReadFile(filepath.Join(applyDir, "modify me.txt"))
	if err != nil || string(modified) != "rewritten content\n" {
		t.Errorf("applied patch did not reproduce the modified file: %q, %v", modified, err)
	}
	if _, err := os.Stat(filepath.Join(applyDir, "delete.txt")); !os.IsNotExist(err) {
		t.Errorf("applied patch did not delete delete.txt: %v", err)
	}
	blob, err := os.ReadFile(filepath.Join(applyDir, "blob.bin"))
	if err != nil || string(blob) != string([]byte{0x00, 0x01, 0xFF, 0x7F}) {
		t.Errorf("applied patch did not reproduce the binary file: %v, %v", blob, err)
	}

	// The user's own checkout must have stayed clean throughout.
	if got := runGit(t, root, "status", "--porcelain"); got != "" {
		t.Errorf("the main checkout became dirty:\n%s", got)
	}
}

func TestCommitAndIntegrate(t *testing.T) {
	root := newTestRepo(t)
	r := New(root)
	ctx := context.Background()
	base, err := r.HeadCommit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	userBranchBefore := runGit(t, root, "rev-parse", "--abbrev-ref", "HEAD")

	wt, err := r.CreateTaskWorktree(ctx, base, "tk-200", 1, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := wt.Commit(ctx, "   "); errs.CategoryOf(err) != errs.Validation {
		t.Errorf("empty message: err = %v, want a validation error", err)
	}
	if _, err := wt.Commit(ctx, "nothing changed yet"); errs.CategoryOf(err) != errs.Git {
		t.Errorf("commit with no changes: err = %v, want a git error", err)
	}

	writeFile(t, wt.Path, "feature.txt", "feature content\n")
	c1, err := wt.Commit(ctx, "Add the feature file")
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if !commitHashPattern.MatchString(c1) {
		t.Errorf("Commit returned %q, want a full commit hash", c1)
	}
	identity := runGit(t, wt.Path, "log", "-1", "--format=%an <%ae>|%cn <%ce>")
	wantIdentity := "PeerFoil Producer <peerfoil@localhost>|PeerFoil Producer <peerfoil@localhost>"
	if identity != wantIdentity {
		t.Errorf("commit identity = %q, want %q", identity, wantIdentity)
	}

	if name := r.IntegrationBranch(1); name != "peerfoil/integration" {
		t.Errorf("IntegrationBranch = %q, want peerfoil/integration", name)
	}

	got, err := r.Integrate(ctx, wt, c1)
	if err != nil {
		t.Fatalf("first Integrate: %v", err)
	}
	if got != c1 {
		t.Errorf("Integrate returned %q, want %q", got, c1)
	}
	if tip := runGit(t, root, "rev-parse", "refs/heads/peerfoil/integration"); tip != c1 {
		t.Errorf("integration tip = %q, want %q", tip, c1)
	}

	// A second commit on the same task fast-forwards the existing branch.
	writeFile(t, wt.Path, "feature.txt", "feature content, improved\n")
	c2, err := wt.Commit(ctx, "Improve the feature file")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Integrate(ctx, wt, c2); err != nil {
		t.Fatalf("fast-forward Integrate: %v", err)
	}
	if tip := runGit(t, root, "rev-parse", "refs/heads/peerfoil/integration"); tip != c2 {
		t.Errorf("integration tip = %q, want %q", tip, c2)
	}

	// A commit that is not on the task branch must be refused outright.
	otherWt, err := r.CreateTaskWorktree(ctx, base, "tk-201", 1, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, otherWt.Path, "other.txt", "unrelated work\n")
	c3, err := otherWt.Commit(ctx, "Unrelated task work")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Integrate(ctx, wt, c3); errs.CategoryOf(err) != errs.Validation {
		t.Errorf("foreign commit: err = %v, want a validation error", err)
	}

	// Force the integration branch somewhere the task commit cannot fast-forward
	// from; Integrate must refuse and leave the decision to the user.
	runGit(t, root, "branch", "-f", "peerfoil/integration", c3)
	_, err = r.Integrate(ctx, wt, c2)
	if errs.CategoryOf(err) != errs.Git {
		t.Fatalf("diverged branch: err = %v, want a git error", err)
	}
	if !strings.Contains(err.Error(), "diverged") {
		t.Errorf("error %q does not say the branch diverged", err)
	}
	if tip := runGit(t, root, "rev-parse", "refs/heads/peerfoil/integration"); tip != c3 {
		t.Errorf("a refused integration moved the branch to %q", tip)
	}

	// Through all of it the user's branch, head, and working tree stay untouched.
	if got := runGit(t, root, "rev-parse", "--abbrev-ref", "HEAD"); got != userBranchBefore {
		t.Errorf("the user's current branch changed from %q to %q", userBranchBefore, got)
	}
	if got := runGit(t, root, "rev-parse", "HEAD"); got != base {
		t.Errorf("the user's HEAD moved from %q to %q", base, got)
	}
	if got := runGit(t, root, "status", "--porcelain"); got != "" {
		t.Errorf("the user's working tree became dirty:\n%s", got)
	}
}

func TestCleanupWorktree(t *testing.T) {
	root := newTestRepo(t)
	r := New(root)
	ctx := context.Background()
	base, err := r.HeadCommit(ctx)
	if err != nil {
		t.Fatal(err)
	}

	wt, err := r.CreateTaskWorktree(ctx, base, "tk-300", 1, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := r.CleanupWorktree(ctx, wt); err != nil {
		t.Fatalf("CleanupWorktree: %v", err)
	}
	for _, line := range strings.Split(runGit(t, root, "worktree", "list", "--porcelain"), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSuffix(line, "\r"), "worktree "); ok && samePath(rest, wt.Path) {
			t.Error("the task worktree is still registered after cleanup")
		}
	}
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", "refs/heads/"+wt.Branch)
	cmd.Dir = root
	if cmd.Run() == nil {
		t.Errorf("the task branch %s still exists after cleanup", wt.Branch)
	}
}

func TestCleanupWorktreeRefusals(t *testing.T) {
	root := newTestRepo(t)
	r := New(root)
	ctx := context.Background()
	base, err := r.HeadCommit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wt, err := r.CreateTaskWorktree(ctx, base, "tk-301", 1, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	// A record doctored to name the user's branch must be refused and must change
	// nothing, whatever path it carries.
	doctored := &Worktree{Repo: r, Path: wt.Path, Branch: "main"}
	if err := r.CleanupWorktree(ctx, doctored); errs.CategoryOf(err) != errs.Policy {
		t.Errorf("doctored branch: err = %v, want a policy error", err)
	}
	if runGit(t, root, "rev-parse", "--verify", "refs/heads/main") == "" {
		t.Error("the main branch is gone after a refused cleanup")
	}
	if _, err := os.Stat(wt.Path); err != nil {
		t.Errorf("the worktree directory is gone after a refused cleanup: %v", err)
	}

	// A task-looking branch with a path Git never registered is refused too.
	unregistered := &Worktree{Repo: r, Path: filepath.Join(t.TempDir(), "ghost"), Branch: "peerfoil/task/tk-999-r1"}
	if err := r.CleanupWorktree(ctx, unregistered); errs.CategoryOf(err) != errs.Policy {
		t.Errorf("unregistered path: err = %v, want a policy error", err)
	}

	if err := r.CleanupWorktree(ctx, nil); errs.CategoryOf(err) != errs.Policy {
		t.Errorf("nil worktree: err = %v, want a policy error", err)
	}
}
