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
	// Later tasks build on the already-integrated work, so the integration branch tip
	// is the base once it exists; the first task starts from the user's HEAD.
	baseCommit, err := e.Repo.CommitOf(ctx, e.Repo.IntegrationBranch(e.State.Project.Revisions.Plan))
	if err != nil {
		baseCommit, err = e.Repo.HeadCommit(ctx)
		if err != nil {
			return err
		}
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
	captureErr := e.capture(ctx, worktree, changeSet, task, outcome.Text, baseCommit)
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
	evidenceIDs, failed, err := e.runTaskEvidence(ctx, worktree, task, changeSetID, taskCommit)
	if err != nil {
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
	return e.enterReview(integrationTip, task.ID, evidenceIDs)
}

// finishPhaseProduction moves to review when the active phase has no eligible task
// left and everything is validated.
func (e *Engine) finishPhaseProduction(ctx context.Context) error {
	for _, task := range e.State.Plan.Tasks() {
		if task.Status == "blocked" || task.Status == "stale" {
			return errs.New(errs.Policy,
				fmt.Sprintf("task %s is %s and cannot be approved as it stands", task.ID, task.Status),
				"run peerfoil resume, or route the change through the PeerFoil skills change intake")
		}
	}
	head, err := e.Repo.HeadCommit(ctx)
	if err != nil {
		return err
	}
	return e.enterReview(head, "", nil)
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
	task *project.Task, handoffText, baseCommit string) error {
	handoff, err := ParseHandoff(handoffText, task,
		e.State.Project.Revisions.Plan, e.State.Project.Revisions.Architecture, baseCommit)
	if err != nil {
		return err
	}
	if handoff.Result == "blocked" {
		blocker := "the producer reported itself blocked"
		if handoff.Blocker != nil && *handoff.Blocker != "" {
			blocker = *handoff.Blocker
		}
		return errs.New(errs.Provider, "the producer reported no completed work: "+blocker, "")
	}

	changes, err := worktree.Changes(ctx)
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		return errs.New(errs.Provider, "the producer reported work but changed nothing", "")
	}
	if err := workspace.CheckScope(changes, task.AllowedPaths); err != nil {
		return err
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
			return errs.New(errs.Provider,
				fmt.Sprintf("the handoff claims a change to %s that Git does not show", claimed), "")
		}
	}

	patch, err := worktree.Patch(ctx)
	if err != nil {
		return err
	}
	if hits := workspace.ScanSecrets(patch); len(hits) > 0 {
		return errs.New(errs.Policy,
			"the produced change may contain a credential: "+strings.Join(hits, "; "),
			"inspect the retained worktree and remove the secret; a committed secret must be rotated")
	}
	patchName := changeSet.ID + ".patch"
	patchPath := filepath.Join(e.State.Dir, "evidence", patchName)
	if err := fsx.WriteFileAtomic(patchPath, patch, 0o644); err != nil {
		return err
	}
	digest := sha256.Sum256(patch)

	changeSet.CaptureStatus = "captured"
	changeSet.CapturedAt = Timestamp(e.Now())
	changeSet.PatchPath = "evidence/" + patchName
	changeSet.PatchSHA256 = hex.EncodeToString(digest[:])
	changeSet.Summary = handoff.Summary
	for _, change := range changes {
		kind := map[string]string{"A": "added", "M": "modified", "D": "deleted", "R": "renamed"}[change.Status]
		if kind == "" {
			kind = change.Status
		}
		changeSet.Changed = append(changeSet.Changed, kind+"\t"+change.Path)
	}
	return e.WriteChangeSet(changeSet)
}

// runTaskEvidence runs every executable evidence requirement the task declares, using
// the pack's procedure for this repository. It returns the record identifiers and the
// names of required checks that did not pass.
func (e *Engine) runTaskEvidence(ctx context.Context, worktree *workspace.Worktree,
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
		argv := e.Pack.CommandFor(requirement.Name, worktree.Path)
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
			RepoRoot:             worktree.Path,
			Now:                  e.Now,
		}, filepath.Join(e.State.Dir, "evidence"))
		if err != nil {
			return recorded, nil, err
		}
		procedure := fmt.Sprintf("%q in the task worktree", argv)
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
