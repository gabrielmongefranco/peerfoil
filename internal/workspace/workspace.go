// This file is part of PeerFoil.
// internal/workspace/workspace.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Git repository checks, dedicated task worktrees, change capture, and
// integration onto the PeerFoil integration branch. Every Git call runs the git
// executable with argument arrays and never touches the user's checkout or index.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

// Package workspace prepares and captures the Git state PeerFoil tasks run in. A task
// gets its own worktree and branch, its changes are read with NUL-delimited plumbing so
// unusual filenames survive, and accepted work lands only on a dedicated integration
// branch. The user's current branch, working tree, and index are never modified.
package workspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

// taskIDPattern is the only accepted task identifier shape. Branch and directory names
// are built from it, so the strict allowlist keeps shell metacharacters, separators,
// and traversal sequences out of ref names and filesystem paths.
var taskIDPattern = regexp.MustCompile(`^tk-[0-9]{3,}$`)

// taskBranchPrefix marks every branch PeerFoil creates for a task. Cleanup refuses to
// delete anything outside this namespace, so a corrupted record cannot take out a
// branch the user owns.
const taskBranchPrefix = "peerfoil/task/"

// integrationBranchName is the single branch accepted work is integrated onto. It is
// never the user's current branch.
const integrationBranchName = "peerfoil/integration"

// ### Git Plumbing ###

// gitEnv is the one place a git subprocess is started. Arguments are an argv array,
// never a shell string; the working directory is explicit; extraEnv entries are added
// to the inherited environment (git needs the user's normal environment to find its
// own configuration and credential helpers). On failure the error carries the leading
// bytes of stderr so the user sees git's own explanation.
func gitEnv(ctx context.Context, dir string, extraEnv []string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if cerr := ctx.Err(); cerr != nil {
			// Report interruption as such instead of whatever partial failure
			// killing the subprocess produced.
			err = fmt.Errorf("%w (%v)", cerr, err)
		}
		detail := strings.TrimSpace(stderr.String())
		if len(detail) > 400 {
			detail = detail[:400]
		}
		if detail != "" {
			return nil, fmt.Errorf("git %s: %w: %s", args[0], err, detail)
		}
		return nil, fmt.Errorf("git %s: %w", args[0], err)
	}
	return stdout.Bytes(), nil
}

// git runs a git command with the inherited environment. See gitEnv.
func git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	return gitEnv(ctx, dir, nil, args...)
}

// gitExitCode returns the subprocess exit code buried in an error from gitEnv, or -1
// when the error is not an exit status. Some git commands answer yes/no questions
// through the exit code, so callers need it to tell "no" apart from "broken".
func gitExitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

// normPath cleans a path for identity comparison. Windows and macOS filesystems are
// case-insensitive by default, so comparisons there fold case; Linux comparisons stay
// exact to avoid matching a genuinely different path.
func normPath(p string) string {
	p = filepath.Clean(filepath.FromSlash(p))
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		p = strings.ToLower(p)
	}
	return p
}

// samePath reports whether two paths name the same location under normPath's rules.
func samePath(a, b string) bool {
	return normPath(a) == normPath(b)
}

// isWithin reports whether p is root itself or a path inside it, after normPath
// normalization. Paths on different Windows drives are never within each other.
func isWithin(root, p string) bool {
	rel, err := filepath.Rel(normPath(root), normPath(p))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// ### Repository ###

// Discover finds the root of the Git repository containing dir. The returned path is
// cleaned and uses the native separator.
func Discover(ctx context.Context, dir string) (string, error) {
	out, err := git(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		e := errs.Wrap(errs.Git, "no Git repository was found at or above the starting directory", err)
		e.Next = "run peerfoil inside a Git repository, or run git init"
		return "", e
	}
	return filepath.Clean(filepath.FromSlash(strings.TrimSpace(string(out)))), nil
}

// Repo is a Git repository identified by its root directory.
type Repo struct {
	Root string
}

// New returns a Repo rooted at the given directory. The caller obtains the root from
// Discover or an equally trusted source.
func New(root string) *Repo {
	return &Repo{Root: filepath.Clean(root)}
}

// HeadCommit returns the full hash of the commit HEAD points at. A repository that has
// never been committed to has no HEAD commit and is reported as such, because PeerFoil
// needs a base commit to branch from.
func (r *Repo) HeadCommit(ctx context.Context) (string, error) {
	out, err := git(ctx, r.Root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		if ctx.Err() != nil {
			return "", errs.Wrap(errs.CategoryOf(ctx.Err()), "reading the repository head was interrupted", err)
		}
		e := errs.Wrap(errs.Git, "the repository has no commit yet", err)
		e.Next = "make an initial commit, for example: git add . && git commit -m \"Initial commit\""
		return "", e
	}
	return strings.TrimSpace(string(out)), nil
}

// IsClean reports whether the working tree and index have no changes, along with the
// paths that are dirty. Paths are decoded from NUL-delimited records, so names with
// spaces, quotes, or non-ASCII characters come back intact.
func (r *Repo) IsClean(ctx context.Context) (bool, []string, error) {
	out, err := git(ctx, r.Root, "status", "--porcelain", "-z", "--untracked-files=all")
	if err != nil {
		return false, nil, errs.Wrap(errs.Git, "could not read the repository status", err)
	}
	entries, err := parseStatusZ(out)
	if err != nil {
		return false, nil, errs.Wrap(errs.Git, "could not decode the repository status", err)
	}
	var dirty []string
	for _, e := range entries {
		dirty = append(dirty, e.Path)
		if e.OldPath != "" {
			dirty = append(dirty, e.OldPath)
		}
	}
	return len(dirty) == 0, dirty, nil
}

// DetectUnsupported rejects repository features Core Alpha does not handle: submodules
// and Git LFS. Running against them would risk silent data loss during change capture,
// so the check fails closed before any task starts.
func (r *Repo) DetectUnsupported(ctx context.Context) error {
	if _, err := os.Stat(filepath.Join(r.Root, ".gitmodules")); err == nil {
		return errs.New(errs.Validation,
			"the repository uses Git submodules, which this release does not support",
			"use a repository without submodules, or wait for a release that supports them")
	}
	data, err := os.ReadFile(filepath.Join(r.Root, ".gitattributes"))
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
			if strings.HasPrefix(line, "#") {
				continue
			}
			if strings.Contains(line, "filter=lfs") {
				return errs.New(errs.Validation,
					"the repository uses Git LFS, which this release does not support",
					"use a repository without Git LFS, or wait for a release that supports it")
			}
		}
	}
	return nil
}

// ### Status Decoding ###

// statusEntry is one record of git status --porcelain -z: the two-letter status code,
// the current path, and, for renames and copies, the original path.
type statusEntry struct {
	XY      string
	Path    string
	OldPath string
}

// parseStatusZ decodes NUL-delimited porcelain v1 status output. A rename or copy
// entry consumes two NUL records: the new path first, then the original path.
func parseStatusZ(out []byte) ([]statusEntry, error) {
	if len(out) == 0 {
		return nil, nil
	}
	records := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	var entries []statusEntry
	for i := 0; i < len(records); i++ {
		rec := records[i]
		if len(rec) < 4 || rec[2] != ' ' {
			return nil, fmt.Errorf("unexpected status record of %d bytes", len(rec))
		}
		entry := statusEntry{XY: rec[:2], Path: rec[3:]}
		if entry.XY[0] == 'R' || entry.XY[0] == 'C' || entry.XY[1] == 'R' || entry.XY[1] == 'C' {
			i++
			if i >= len(records) {
				return nil, fmt.Errorf("a rename record is missing its original path")
			}
			entry.OldPath = records[i]
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// Change is one changed path in a task worktree. Status is "A" (added, including
// untracked files), "M" (modified, including mode changes), "D" (deleted), or "R"
// (renamed). A rename fills both Path (new name) and OldPath (original name); every
// other change fills Path alone. Paths are repository-relative with forward slashes,
// exactly as Git reports them.
type Change struct {
	Status  string
	Path    string
	OldPath string
}

// changeStatus collapses a porcelain XY code into the single-letter vocabulary Change
// uses. Untracked files count as additions because integrating the task would add
// them; type and mode changes count as modifications.
func changeStatus(xy string) string {
	if xy == "??" {
		return "A"
	}
	if xy[0] == 'R' || xy[1] == 'R' {
		return "R"
	}
	code := xy[0]
	if code == ' ' || code == '.' {
		code = xy[1]
	}
	switch code {
	case 'A', 'C':
		return "A"
	case 'D':
		return "D"
	default:
		return "M"
	}
}

// ### Task Worktrees ###

// Worktree is a dedicated checkout PeerFoil created for one task. Base is the resolved
// commit the task branched from, and Branch always lives under peerfoil/task/.
type Worktree struct {
	Repo   *Repo
	Path   string
	Branch string
	Base   string
	TaskID string
}

// CreateTaskWorktree creates a new branch and worktree for one task, rooted at
// baseCommit. The worktree directory goes under parentDir, which must be outside the
// repository so task files never appear in the user's checkout. The branch name is
// derived only from the validated task identifier and plan revision, so it cannot
// carry injected ref syntax.
func (r *Repo) CreateTaskWorktree(ctx context.Context, baseCommit, taskID string, planRevision int, parentDir string) (*Worktree, error) {
	if !taskIDPattern.MatchString(taskID) {
		return nil, errs.New(errs.Validation,
			fmt.Sprintf("task identifier %q is not valid", taskID),
			"use a task identifier such as tk-001: the letters tk, a hyphen, and at least three digits")
	}
	if planRevision < 1 {
		return nil, errs.New(errs.Validation,
			fmt.Sprintf("plan revision %d is not valid", planRevision),
			"use a plan revision of 1 or higher")
	}
	out, err := git(ctx, r.Root, "rev-parse", "--verify", baseCommit+"^{commit}")
	if err != nil {
		return nil, errs.Wrap(errs.Git, "the base commit for the task could not be resolved", err)
	}
	base := strings.TrimSpace(string(out))

	branch := fmt.Sprintf("%s%s-r%d", taskBranchPrefix, taskID, planRevision)
	if _, err := git(ctx, r.Root, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		e := errs.New(errs.Git,
			fmt.Sprintf("the task branch %s already exists", branch),
			"finish or clean up the earlier run of this task, or use a new task identifier")
		return nil, e
	}

	// Git registers a worktree by its real path, so a symlinked parent (such as
	// /var on macOS, which links to /private/var) is resolved first; otherwise the
	// recorded path never matches what git lists and cleanup refuses its own work.
	if resolved, err := filepath.EvalSymlinks(parentDir); err == nil {
		parentDir = resolved
	}
	wtPath := filepath.Join(parentDir, fmt.Sprintf("%s-r%d", taskID, planRevision))
	if isWithin(r.Root, wtPath) {
		return nil, errs.New(errs.Validation,
			"the task worktree directory must be outside the repository",
			"pass a state or temporary directory that is not inside the repository")
	}
	if err := os.MkdirAll(parentDir, 0o700); err != nil {
		return nil, errs.Wrap(errs.Internal, "could not create the directory that holds task worktrees", err)
	}
	if _, err := git(ctx, r.Root, "worktree", "add", "-b", branch, wtPath, base); err != nil {
		return nil, errs.Wrap(errs.Git, "could not create the task worktree", err)
	}
	return &Worktree{Repo: r, Path: wtPath, Branch: branch, Base: base, TaskID: taskID}, nil
}

// Changes lists what the task changed in its worktree relative to the worktree's own
// HEAD, including untracked files. Paths come from NUL-delimited output, so unusual
// filenames are preserved exactly.
func (w *Worktree) Changes(ctx context.Context) ([]Change, error) {
	// --untracked-files=all lists each new file itself rather than collapsing a new
	// directory into one entry, so scope checking sees every real path.
	out, err := git(ctx, w.Path, "status", "--porcelain", "-z", "--untracked-files=all")
	if err != nil {
		return nil, errs.Wrap(errs.Git, "could not read the task worktree status", err)
	}
	entries, err := parseStatusZ(out)
	if err != nil {
		return nil, errs.Wrap(errs.Git, "could not decode the task worktree status", err)
	}
	var changes []Change
	for _, e := range entries {
		changes = append(changes, Change{Status: changeStatus(e.XY), Path: e.Path, OldPath: e.OldPath})
	}
	return changes, nil
}

// Patch produces a binary-capable patch of the worktree's current content against the
// task's base commit, covering untracked files. It stages into a temporary index file
// that is deleted afterwards, so the worktree's real index, and with it anything the
// producer staged, stays exactly as it was.
func (w *Worktree) Patch(ctx context.Context) ([]byte, error) {
	tmpDir, err := os.MkdirTemp("", "peerfoil-index-")
	if err != nil {
		return nil, errs.Wrap(errs.Internal, "could not create a temporary directory for change capture", err)
	}
	defer os.RemoveAll(tmpDir)
	env := []string{"GIT_INDEX_FILE=" + filepath.Join(tmpDir, "index")}
	if _, err := gitEnv(ctx, w.Path, env, "add", "-A"); err != nil {
		return nil, errs.Wrap(errs.Git, "could not capture the task changes", err)
	}
	out, err := gitEnv(ctx, w.Path, env, "diff", "--binary", "--cached", w.Base)
	if err != nil {
		return nil, errs.Wrap(errs.Git, "could not produce the task patch", err)
	}
	return out, nil
}

// Commit stages everything in the task worktree and commits it with PeerFoil's own
// author and committer identity, returning the commit hash. It refuses to run outside
// a peerfoil/task/ branch so it can never commit to the user's checkout. Signing is
// turned off for this one commit because a machine-made task commit must not depend on
// the user's signing keys or block on a passphrase prompt.
func (w *Worktree) Commit(ctx context.Context, message string) (string, error) {
	if strings.TrimSpace(message) == "" {
		return "", errs.New(errs.Validation, "the commit message is empty", "pass a short description of the task's work")
	}
	if !strings.HasPrefix(w.Branch, taskBranchPrefix) {
		return "", errs.New(errs.Policy,
			"refusing to commit outside a PeerFoil task worktree",
			"commits run only on branches under "+taskBranchPrefix)
	}
	if _, err := git(ctx, w.Path, "add", "-A"); err != nil {
		return "", errs.Wrap(errs.Git, "could not stage the task changes", err)
	}
	_, err := git(ctx, w.Path,
		"-c", "user.name=PeerFoil Producer",
		"-c", "user.email=peerfoil@localhost",
		"-c", "commit.gpgsign=false",
		"commit",
		"--author=PeerFoil Producer <peerfoil@localhost>",
		"-m", message)
	if err != nil {
		return "", errs.Wrap(errs.Git, "could not commit the task changes", err)
	}
	out, err := git(ctx, w.Path, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return "", errs.Wrap(errs.Git, "could not read the new task commit", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ### Integration ###

// IntegrationBranch names the branch accepted work is integrated onto. The name does
// not vary by plan revision: one integration line keeps the user's view simple, and
// the plan revision is already recorded on each task branch.
func (r *Repo) IntegrationBranch(planRevision int) string {
	return integrationBranchName
}

// Integrate moves the integration branch to taskCommit. The first integration creates
// the branch; later ones fast-forward it, and only after proving the current tip is an
// ancestor of the task commit. A diverged branch is refused so the user decides what
// to do; this release does not resolve conflicts. The user's current branch, working
// tree, and index are never touched.
func (r *Repo) Integrate(ctx context.Context, w *Worktree, taskCommit string) (string, error) {
	out, err := git(ctx, r.Root, "rev-parse", "--verify", taskCommit+"^{commit}")
	if err != nil {
		return "", errs.Wrap(errs.Git, "the task commit to integrate could not be resolved", err)
	}
	commit := strings.TrimSpace(string(out))

	// Provenance: only work recorded on this task's branch may be integrated, so a
	// wrong or stale hash cannot smuggle unrelated history onto the integration line.
	if _, err := git(ctx, r.Root, "merge-base", "--is-ancestor", commit, "refs/heads/"+w.Branch); err != nil {
		if gitExitCode(err) == 1 {
			return "", errs.New(errs.Validation,
				"the commit to integrate is not on the task branch "+w.Branch,
				"integrate only commits made in this task's worktree")
		}
		return "", errs.Wrap(errs.Git, "could not verify the task commit belongs to the task branch", err)
	}

	ref := "refs/heads/" + integrationBranchName
	tipOut, err := git(ctx, r.Root, "rev-parse", "--verify", "--quiet", ref)
	if err != nil {
		if _, err := git(ctx, r.Root, "branch", integrationBranchName, commit); err != nil {
			return "", errs.Wrap(errs.Git, "could not create the integration branch", err)
		}
		return commit, nil
	}
	tip := strings.TrimSpace(string(tipOut))
	if tip == commit {
		return commit, nil
	}
	if _, err := git(ctx, r.Root, "merge-base", "--is-ancestor", tip, commit); err != nil {
		if gitExitCode(err) == 1 {
			return "", errs.New(errs.Git,
				"the integration branch has diverged from the task commit",
				"inspect the "+integrationBranchName+" branch and decide how to combine the work; this release does not resolve conflicts")
		}
		return "", errs.Wrap(errs.Git, "could not compare the integration branch with the task commit", err)
	}
	// Fast-forward with a compare-and-swap on the old tip, so a concurrent move of
	// the branch fails loudly instead of being overwritten.
	if _, err := git(ctx, r.Root, "update-ref", ref, commit, tip); err != nil {
		return "", errs.Wrap(errs.Git, "could not fast-forward the integration branch", err)
	}
	return commit, nil
}

// ### Cleanup ###

// CleanupWorktree removes a task worktree and its branch. It deletes only what
// PeerFoil created: the branch must live under peerfoil/task/, and the path must be a
// worktree Git currently has registered. Anything else is refused, so a corrupted or
// doctored record can never delete the user's branches or directories.
func (r *Repo) CleanupWorktree(ctx context.Context, w *Worktree) error {
	if w == nil || !strings.HasPrefix(w.Branch, taskBranchPrefix) {
		return errs.New(errs.Policy,
			"refusing to clean up: the recorded branch is not a PeerFoil task branch",
			"only branches under "+taskBranchPrefix+" are removed automatically; delete anything else yourself")
	}
	if w.Path == "" {
		return errs.New(errs.Policy,
			"refusing to clean up: the recorded worktree path is empty",
			"remove the worktree yourself with git worktree remove")
	}
	out, err := git(ctx, r.Root, "worktree", "list", "--porcelain")
	if err != nil {
		return errs.Wrap(errs.Git, "could not list the repository's worktrees", err)
	}
	registered := false
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if rest, ok := strings.CutPrefix(line, "worktree "); ok && samePath(rest, w.Path) {
			registered = true
			break
		}
	}
	if !registered {
		return errs.New(errs.Policy,
			"refusing to clean up: the recorded worktree path is not registered with Git",
			"check the worktrees with git worktree list and remove the right one yourself")
	}
	if _, err := git(ctx, r.Root, "worktree", "remove", "--force", w.Path); err != nil {
		return errs.Wrap(errs.Git, "could not remove the task worktree", err)
	}
	if _, err := git(ctx, r.Root, "branch", "-D", w.Branch); err != nil {
		return errs.Wrap(errs.Git, "could not delete the task branch "+w.Branch, err)
	}
	return nil
}
