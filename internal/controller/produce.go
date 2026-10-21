// This file is part of PeerFoil.
// internal/controller/produce.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: The enforced vertical slice: select one task, run the producer in a
// dedicated worktree, capture provenance, run the required checks, and integrate only
// in-scope work with passing evidence.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/evidence"
	"github.com/gabrielmongefranco/peerfoil/internal/fsx"
	"github.com/gabrielmongefranco/peerfoil/internal/project"
	"github.com/gabrielmongefranco/peerfoil/internal/provider"
	"github.com/gabrielmongefranco/peerfoil/internal/store"
	"github.com/gabrielmongefranco/peerfoil/internal/workspace"
)

// ProduceNext runs one complete task: gate, select, produce, capture, validate, and
// integrate. It stops after one task, or earlier at the first condition that needs
// the user, and every stop leaves records that explain it.
func (e *Engine) ProduceNext(ctx context.Context) error {
	if err := e.productionGate(); err != nil {
		return err
	}
	if open := e.State.InFlight(); len(open) > 0 {
		return errs.New(errs.Policy,
			fmt.Sprintf("task %s is already in flight", open[0].ID),
			"run peerfoil resume to continue it")
	}
	selection, err := e.State.NextTask()
	if err != nil {
		return err
	}
	if selection == nil {
		return e.finishPhaseProduction(ctx)
	}
	task := selection.Task

	clean, dirty, err := e.Repo.IsClean(ctx)
	if err != nil {
		return err
	}
	if !clean {
		if err := e.requireCleanOutsideRecords(dirty); err != nil {
			return err
		}
	}
	if err := e.Repo.DetectUnsupported(ctx); err != nil {
		return err
	}
	baseCommit, err := e.productionBase(ctx)
	if err != nil {
		return err
	}

	changeSetID, err := e.nextEvidenceDirID("cs")
	if err != nil {
		return err
	}
	seatModel := e.ProducerSeat.Model
	seatEffort := e.ProducerSeat.Effort
	author := project.Actor{
		Role:        "producer",
		Tool:        e.Producer.Name(),
		Model:       &seatModel,
		Effort:      &seatEffort,
		LineageRoot: e.Producer.LineageRoot(seatModel),
	}
	changeSet := &ChangeSet{
		ID:            changeSetID,
		TaskID:        task.ID,
		TaskTitle:     task.Title,
		CaptureStatus: "pending",
		StartedAt:     Timestamp(e.Now()),
		Attempt:       1,
		Baseline:      "clean working tree outside .peerfoil at the base revision",
		BaseRevision:  baseCommit,
		Author:        author,
		Summary:       "Launch recorded before the producer ran.",
	}
	// The launch is persisted before the producer runs, so an interruption leaves a
	// pending capture to inspect instead of silent uncertainty (D-0025).
	if err := e.WriteChangeSet(changeSet); err != nil {
		return err
	}
	task.Status = "in_progress"
	e.State.Project.Workflow.Phase = &selection.Phase.ID
	e.State.Project.Workflow.Stage = &selection.Stage.ID
	e.State.Project.Workflow.Task = &task.ID
	if err := e.SavePlan(); err != nil {
		return err
	}
	if err := e.saveProject(); err != nil {
		return err
	}

	worktree, err := e.Repo.CreateTaskWorktree(ctx, baseCommit, task.ID, e.State.Project.Revisions.Plan, e.WorktreeParent)
	if err != nil {
		return err
	}
	if _, err := e.Journal.Intent("producer", changeSetID, map[string]string{
		"task": task.ID, "base": baseCommit, "worktree": worktree.Path, "branch": worktree.Branch}); err != nil {
		return err
	}

	packet := e.BuildProducerPacket(selection, changeSetID, baseCommit)
	if _, err := e.Journal.Started("producer", changeSetID); err != nil {
		return err
	}
	fmt.Fprintf(e.Out, "Producing %s (%s) with %s in a dedicated worktree.\n", task.ID, task.Title, e.Producer.Name())
	outcome, invokeErr := e.Producer.Invoke(ctx, provider.Invocation{
		Role:    "producer",
		Packet:  packet,
		Sandbox: provider.SandboxWorkspaceWrite,
		Dir:     worktree.Path,
	})
	if invokeErr != nil {
		status := store.StatusFailed
		switch errs.CategoryOf(invokeErr) {
		case errs.Cancelled:
			status = store.StatusCancelled
		case errs.Timeout:
			// A writing timeout is ambiguous: the process was killed, but what it
			// wrote is unknown until someone inspects the worktree.
			status = store.StatusUnknown
		}
		_, _ = e.Journal.Result("producer", changeSetID, status, map[string]string{"category": string(errs.CategoryOf(invokeErr))})
		changeSet.CaptureStatus = "blocked"
		changeSet.Summary = "The producer run ended without a usable result; the worktree is retained for inspection."
		_ = e.WriteChangeSet(changeSet)
		task.Status = "blocked"
		_ = e.SavePlan()
		reason := fmt.Sprintf("the producer run for %s ended without a result (%s); the worktree %s is retained",
			task.ID, errs.CategoryOf(invokeErr), worktree.Branch)
		if err := e.Pause(reason, map[string][]string{"tasks": {task.ID}, "change_sets": {changeSetID}}); err != nil {
			return err
		}
		return invokeErr
	}
	if outcome.SessionID != "" {
		session := outcome.SessionID
		changeSet.Author.Session = &session
	}

	// Capture before anything else looks at the work.
	handoff, captureErr := e.capture(ctx, worktree, changeSet, task, outcome.Text, baseCommit)
	if captureErr != nil {
		_, _ = e.Journal.Result("producer", changeSetID, store.StatusFailed, map[string]string{"reason": "capture"})
		changeSet.CaptureStatus = "blocked"
		_ = e.WriteChangeSet(changeSet)
		task.Status = "blocked"
		_ = e.SavePlan()
		reason := fmt.Sprintf("the produced change for %s was not accepted: %v; the worktree %s is retained",
			task.ID, captureErr, worktree.Branch)
		if err := e.Pause(reason, map[string][]string{"tasks": {task.ID}, "change_sets": {changeSetID}}); err != nil {
			return err
		}
		return captureErr
	}
	// A partial result is captured work, never completed work: the patch and record
	// stay for inspection, and the task cannot be validated or integrated until the
	// remaining work is routed through change intake.
	if handoff.Result == "partial" {
		_, _ = e.Journal.Result("producer", changeSetID, store.StatusFailed, map[string]string{"reason": "partial"})
		task.Status = "blocked"
		_ = e.SavePlan()
		reason := fmt.Sprintf("the producer finished only part of %s; the partial change is captured as %s and the worktree %s is retained",
			task.ID, changeSetID, worktree.Branch)
		if err := e.Pause(reason, map[string][]string{"tasks": {task.ID}, "change_sets": {changeSetID}}); err != nil {
			return err
		}
		return errs.New(errs.Provider, reason,
			"inspect the captured patch, then route the remaining work through the PeerFoil skills change intake")
	}
	// Discovered work must reach change intake, not evaporate: the entries are in the
	// change-set record, and the workflow stops before integration so the plan can
	// absorb them first.
	if len(handoff.DiscoveredWork) > 0 {
		_, _ = e.Journal.Result("producer", changeSetID, store.StatusSucceeded, map[string]string{"discovered": fmt.Sprint(len(handoff.DiscoveredWork))})
		task.Status = "blocked"
		_ = e.SavePlan()
		reason := fmt.Sprintf("the producer finished %s but discovered %d new piece(s) of work, recorded in %s; the change is captured and the worktree %s is retained",
			task.ID, len(handoff.DiscoveredWork), changeSetID, worktree.Branch)
		if err := e.Pause(reason, map[string][]string{"tasks": {task.ID}, "change_sets": {changeSetID}}); err != nil {
			return err
		}
		return errs.New(errs.Policy, reason,
			"route the discovered work through the PeerFoil skills change intake, then run peerfoil start")
	}
	if _, err := e.Journal.Result("producer", changeSetID, store.StatusSucceeded, nil); err != nil {
		return err
	}
	task.Status = "produced"
	if err := e.SavePlan(); err != nil {
		return err
	}
	if err := e.AppendTransition(Validate, map[string][]string{
		"tasks": {task.ID}, "change_sets": {changeSetID},
	}, fmt.Sprintf("Change for %s captured as %s; required checks run next.", task.ID, changeSetID), &baseCommit); err != nil {
		return err
	}

	taskCommit, err := worktree.Commit(ctx, fmt.Sprintf("PeerFoil %s: %s", task.ID, task.Title))
	if err != nil {
		return err
	}
	// The captured work now has a commit, so the change set records it (D-0025
	// allows a null source revision only for uncommitted work).
	changeSet.SourceRev = taskCommit
	if err := e.WriteChangeSet(changeSet); err != nil {
		return err
	}
	evidenceIDs, failed, err := e.runTaskEvidence(ctx, worktree.Path, task, changeSetID, taskCommit)
	if err != nil {
		// The workflow is already in validate, which resume refuses as a boundary,
		// so a runner failure pauses with the task blocked instead of stranding it.
		task.Status = "blocked"
		_ = e.SavePlan()
		reason := fmt.Sprintf("an evidence command for %s could not run: %v; the worktree %s is retained",
			task.ID, err, worktree.Branch)
		if pauseErr := e.Pause(reason, map[string][]string{"tasks": {task.ID}, "change_sets": {changeSetID}, "evidence": evidenceIDs}); pauseErr != nil {
			return pauseErr
		}
		return err
	}
	if len(failed) > 0 {
		// A required failure returns the task to production rather than being voted
		// away; the worktree stays for the repair.
		task.Status = "blocked"
		if err := e.SavePlan(); err != nil {
			return err
		}
		if err := e.AppendTransition(Produce, map[string][]string{
			"tasks": {task.ID}, "evidence": evidenceIDs,
		}, fmt.Sprintf("Required checks failed for %s: %s.", task.ID, strings.Join(failed, ", ")), &taskCommit); err != nil {
			return err
		}
		return errs.New(errs.Evidence,
			fmt.Sprintf("required checks failed for %s: %s", task.ID, strings.Join(failed, ", ")),
			"inspect the retained worktree branch "+worktree.Branch+" and the evidence records, then route the fix through the PeerFoil skills change intake")
	}

	integrationTip, err := e.Repo.Integrate(ctx, worktree, taskCommit)
	if err != nil {
		return err
	}
	task.Status = "validated"
	e.State.Project.Workflow.Task = nil
	if err := e.SavePlan(); err != nil {
		return err
	}
	if err := e.Repo.CleanupWorktree(ctx, worktree); err != nil {
		fmt.Fprintf(e.Out, "Note: the task worktree was not removed: %v\n", err)
	}

	next, err := e.State.NextTask()
	if err != nil {
		return err
	}
	if next != nil {
		if err := e.AppendTransition(Produce, map[string][]string{
			"tasks": {task.ID}, "evidence": evidenceIDs,
		}, fmt.Sprintf("%s validated and integrated; %s is ready next.", task.ID, next.Task.ID), &integrationTip); err != nil {
			return err
		}
		fmt.Fprintf(e.Out, "Done. %s is validated and integrated on %s.\nNext: run peerfoil start for %s.\n",
			task.ID, e.Repo.IntegrationBranch(e.State.Project.Revisions.Plan), next.Task.ID)
		return nil
	}
	// The review boundary is only reachable when no task in the phase carries a
	// failed required check, and the required evidence must hold at the exact
	// revision the reviewers will freeze, not only at each task's own commit.
	if err := e.blockedTaskError(); err != nil {
		return err
	}
	if err := e.unreachableTaskError(); err != nil {
		return err
	}
	frozenEvidence, err := e.revalidatePhaseEvidence(ctx, integrationTip)
	if err != nil {
		if pauseErr := e.Pause(fmt.Sprintf("the phase's evidence re-check stopped: %v", err), nil); pauseErr != nil {
			return pauseErr
		}
		return err
	}
	return e.enterReview(integrationTip, task.ID, append(evidenceIDs, frozenEvidence...))
}

// revalidatePhaseEvidence reruns the active phase's validated tasks' executable
// evidence against the frozen revision, so approval can never rest on results from an
// earlier commit that later tasks changed. Earlier phases are already approved and
// closed, so their evidence, including human and inspection results the controller
// cannot rerun, is not re-litigated here.
func (e *Engine) revalidatePhaseEvidence(ctx context.Context, frozen string) ([]string, error) {
	checkout, err := e.Repo.CreateReviewWorktree(ctx, frozen, "evidence", e.WorktreeParent)
	if err != nil {
		return nil, err
	}
	defer func() { _ = e.Repo.RemoveReviewWorktree(ctx, checkout) }()
	var recorded []string
	var failed []string
	for _, task := range e.State.Plan.ActiveTasks() {
		if task.Status != "validated" && task.Status != "accepted" {
			continue
		}
		ids, taskFailed, err := e.runTaskEvidence(ctx, checkout, task, "", frozen)
		if err != nil {
			return recorded, err
		}
		recorded = append(recorded, ids...)
		failed = append(failed, taskFailed...)
	}
	if len(failed) > 0 {
		return recorded, errs.New(errs.Evidence,
			"required checks failed at the frozen revision: "+strings.Join(failed, ", "),
			"the phase cannot enter review; route the fix through the PeerFoil skills change intake")
	}
	return recorded, nil
}

// productionBase picks the commit a task builds on. Mid-phase that is the integration
// tip, so later tasks see earlier integrated work. When the user has merged the
// integration branch and moved HEAD past it, the next work starts from HEAD and the
// integration branch fast-forwards onto it. Divergence between the two is not
// something Alpha resolves, so it stops for the user.
func (e *Engine) productionBase(ctx context.Context) (string, error) {
	head, err := e.Repo.HeadCommit(ctx)
	if err != nil {
		return "", err
	}
	tip, err := e.Repo.CommitOf(ctx, e.Repo.IntegrationBranch(e.State.Project.Revisions.Plan))
	if err != nil {
		return head, nil
	}
	switch {
	case e.Repo.IsAncestor(ctx, head, tip):
		return tip, nil
	case e.Repo.IsAncestor(ctx, tip, head):
		return head, nil
	}
	return "", errs.New(errs.Git,
		"the peerfoil/integration branch and your current HEAD have diverged",
		"merge or rebase one onto the other, or delete the integration branch if its work is already merged, then run peerfoil start")
}

// blockedTaskError refuses to move toward review while any task is blocked or stale,
// because a required failure is never voted away or skipped past.
func (e *Engine) blockedTaskError() error {
	for _, task := range e.State.Plan.Tasks() {
		if task.Status == "blocked" || task.Status == "stale" {
			return errs.New(errs.Policy,
				fmt.Sprintf("task %s is %s and cannot be approved as it stands", task.ID, task.Status),
				"route the fix through the PeerFoil skills change intake, then run peerfoil start")
		}
	}
	return nil
}

// unreachableTaskError refuses the review boundary while the active phase still holds
// a task that is neither finished nor dismissed. Selection can return no task even
// then, when a dependency was deferred or removed, and approving the phase around such
// a task would silently drop planned work.
func (e *Engine) unreachableTaskError() error {
	for _, task := range e.State.Plan.ActiveTasks() {
		switch task.Status {
		case "validated", "accepted", "deferred", "removed":
			continue
		}
		return errs.New(errs.Policy,
			fmt.Sprintf("task %s is still %s and cannot be selected, so the phase cannot enter review; a dependency may be deferred or removed",
				task.ID, task.Status),
			"revise, defer, or remove the task through the PeerFoil skills change intake, then run peerfoil start")
	}
	return nil
}

// finishPhaseProduction moves to review when the active phase has no eligible task
// left and everything is validated. The lifecycle has no direct produce-to-review
// edge, so the completeness re-check is recorded as its own validate step.
func (e *Engine) finishPhaseProduction(ctx context.Context) error {
	if err := e.blockedTaskError(); err != nil {
		return err
	}
	if err := e.unreachableTaskError(); err != nil {
		return err
	}
	head, err := e.Repo.HeadCommit(ctx)
	if err != nil {
		return err
	}
	if tip, err := e.Repo.CommitOf(ctx, e.Repo.IntegrationBranch(e.State.Project.Revisions.Plan)); err == nil {
		head = tip
	}
	if err := e.AppendTransition(Validate, nil,
		"No eligible task remains; the phase's evidence is re-checked before review.", &head); err != nil {
		return err
	}
	frozenEvidence, err := e.revalidatePhaseEvidence(ctx, head)
	if err != nil {
		if pauseErr := e.Pause(fmt.Sprintf("the phase's evidence re-check stopped: %v", err), nil); pauseErr != nil {
			return pauseErr
		}
		return err
	}
	return e.enterReview(head, "", frozenEvidence)
}

func (e *Engine) enterReview(sourceRevision, lastTask string, evidenceIDs []string) error {
	refs := map[string][]string{}
	if lastTask != "" {
		refs["tasks"] = []string{lastTask}
	}
	if len(evidenceIDs) > 0 {
		refs["evidence"] = evidenceIDs
	}
	if err := e.AppendTransition(Review, refs,
		"Every task in the active phase is validated; the phase review is next.", &sourceRevision); err != nil {
		return err
	}
	fmt.Fprintln(e.Out, "Every task in the active phase is validated.")
	fmt.Fprintln(e.Out, "Next: run peerfoil resume to start the two-family phase review.")
	return nil
}

// capture validates the handoff, checks scope, scans for secrets, and retains the
// patch and provenance before any other agent can edit the work.
func (e *Engine) capture(ctx context.Context, worktree *workspace.Worktree, changeSet *ChangeSet,
	task *project.Task, handoffText, baseCommit string) (*Handoff, error) {
	handoff, err := ParseHandoff(handoffText, task,
		e.State.Project.Revisions.Plan, e.State.Project.Revisions.Architecture, baseCommit)
	if err != nil {
		return nil, err
	}
	if handoff.Result == "blocked" {
		blocker := "the producer reported itself blocked"
		if handoff.Blocker != nil && *handoff.Blocker != "" {
			blocker = *handoff.Blocker
		}
		return nil, errs.New(errs.Provider, "the producer reported no completed work: "+blocker, "")
	}

	changes, err := worktree.Changes(ctx)
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return nil, errs.New(errs.Provider, "the producer reported work but changed nothing", "")
	}
	if err := workspace.CheckScope(changes, task.AllowedPaths); err != nil {
		return nil, err
	}
	if err := rejectSymlinks(worktree.Path, changes); err != nil {
		return nil, err
	}
	// The actual diff decides what changed; the handoff's list is only cross-checked.
	actual := map[string]bool{}
	for _, change := range changes {
		actual[change.Path] = true
		if change.OldPath != "" {
			actual[change.OldPath] = true
		}
	}
	for _, claimed := range handoff.ChangedPaths {
		if !actual[filepath.ToSlash(claimed)] {
			return nil, errs.New(errs.Provider,
				fmt.Sprintf("the handoff claims a change to %s that Git does not show", claimed), "")
		}
	}

	patch, err := worktree.Patch(ctx)
	if err != nil {
		return nil, err
	}
	if hits := workspace.ScanSecrets(patch); len(hits) > 0 {
		return nil, errs.New(errs.Policy,
			"the produced change may contain a credential: "+strings.Join(hits, "; "),
			"inspect the retained worktree and remove the secret; a committed secret must be rotated")
	}
	patchName := changeSet.ID + ".patch"
	patchPath := filepath.Join(e.State.Dir, "evidence", patchName)
	if err := fsx.WriteFileAtomic(patchPath, patch, 0o644); err != nil {
		return nil, err
	}
	digest := sha256.Sum256(patch)

	changeSet.CaptureStatus = "captured"
	changeSet.CapturedAt = Timestamp(e.Now())
	changeSet.PatchPath = "evidence/" + patchName
	changeSet.PatchSHA256 = hex.EncodeToString(digest[:])
	changeSet.Summary = handoff.Summary
	changeSet.Discovered = handoff.DiscoveredWork
	for _, change := range changes {
		kind := map[string]string{"A": "added", "M": "modified", "D": "deleted", "R": "renamed"}[change.Status]
		if kind == "" {
			kind = change.Status
		}
		changeSet.Changed = append(changeSet.Changed, kind+"\t"+change.Path)
	}
	if err := e.WriteChangeSet(changeSet); err != nil {
		return nil, err
	}
	return handoff, nil
}

// rejectSymlinks refuses a change set that adds or modifies a symbolic link, because a
// link pointing outside the repository would let any later checkout, build, or check
// read or write beyond it; the default product path does not rely on symlinks at all.
func rejectSymlinks(worktreePath string, changes []workspace.Change) error {
	var links []string
	for _, change := range changes {
		if change.Status == "D" {
			continue
		}
		info, err := os.Lstat(filepath.Join(worktreePath, filepath.FromSlash(change.Path)))
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			links = append(links, change.Path)
		}
	}
	if len(links) > 0 {
		return errs.New(errs.Policy,
			"the produced change adds or edits symbolic links, which are not accepted: "+strings.Join(links, ", "),
			"replace the links with regular files, or route the need through the PeerFoil skills")
	}
	return nil
}

// runTaskEvidence runs every executable evidence requirement the task declares, using
// the pack's procedure for this repository. It returns the record identifiers and the
// names of required checks that did not pass.
func (e *Engine) runTaskEvidence(ctx context.Context, checkoutDir string,
	task *project.Task, changeSetID, taskCommit string) ([]string, []string, error) {
	var recorded []string
	var failedRequired []string
	for _, requirement := range task.RequiredEvidence {
		if requirement.Level == "not-applicable" {
			continue
		}
		if requirement.Kind != "executable" {
			if requirement.Level == "required" {
				failedRequired = append(failedRequired, requirement.Name+" (needs a human or inspection result)")
			}
			continue
		}
		argv := e.Pack.CommandFor(requirement.Name, checkoutDir)
		if argv == nil {
			if requirement.Level == "required" {
				return recorded, nil, errs.New(errs.Evidence,
					fmt.Sprintf("no executable procedure is known for the required evidence %q in this repository", requirement.Name),
					"declare the command in the Quality Contract with the PeerFoil skills, or adjust the task's evidence")
			}
			continue
		}
		evidenceID, err := e.nextEvidenceDirID("ev")
		if err != nil {
			return recorded, nil, err
		}
		fmt.Fprintf(e.Out, "Running %s: %s\n", requirement.Name, strings.Join(argv, " "))
		procedure := fmt.Sprintf("%q in the task worktree", argv)
		record, err := evidence.Run(ctx, e.Evidence, evidence.Command{
			Name:           requirement.Name,
			Argv:           argv,
			TimeoutSeconds: 600,
			Env:            provider.BaseEnv(),
		}, evidence.Requirement{
			Name: requirement.Name, Kind: requirement.Kind, Level: requirement.Level,
		}, evidence.Meta{
			ID:                   evidenceID,
			TaskID:               task.ID,
			PlanRevision:         e.State.Project.Revisions.Plan,
			ArchitectureRevision: e.State.Project.Revisions.Architecture,
			SourceRevision:       taskCommit,
			RepoRoot:             checkoutDir,
			Now:                  e.Now,
		}, filepath.Join(e.State.Dir, "evidence"))
		if err != nil {
			// A command that could not run still produced a record with its retained
			// output; that record belongs in the versioned evidence, not only in the
			// returned error, so the failure stays visible after the run ends.
			if record.Status == evidence.StatusError {
				if writeErr := e.WriteEvidenceRecord(record, changeSetID, procedure); writeErr == nil {
					recorded = append(recorded, evidenceID)
				}
			}
			return recorded, nil, err
		}
		if err := e.WriteEvidenceRecord(record, changeSetID, procedure); err != nil {
			return recorded, nil, err
		}
		recorded = append(recorded, evidenceID)
		if record.Status != evidence.StatusPass && requirement.Level == "required" {
			failedRequired = append(failedRequired, requirement.Name)
		}
		fmt.Fprintf(e.Out, "  %s: %s\n", requirement.Name, record.Status)
	}
	return recorded, failedRequired, nil
}
