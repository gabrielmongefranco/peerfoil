// This file is part of PeerFoil.
// internal/controller/resume.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Recovery after an interruption. Resume continues only from recorded safe
// boundaries; an outside operation with no recorded result is inspected, never rerun.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/store"
)

// Resume continues the workflow from the recorded state. ConfirmStopped says the user
// has confirmed that no interrupted producer process is still running; without it, an
// unknown invocation stops resume so nothing runs twice.
func (e *Engine) Resume(ctx context.Context, confirmStopped bool) error {
	open, err := e.Journal.OpenInvocations()
	if err != nil {
		return err
	}
	if len(open) > 0 {
		return e.recoverOpenInvocations(open, confirmStopped)
	}

	switch State(e.State.Project.Workflow.State) {
	case Produce:
		return e.ProduceNext(ctx)
	case Review:
		return e.PhaseReview(ctx)
	case Validate, Repair:
		// A crash landed between recorded steps inside one task. That is not a
		// completed task boundary, so Core does not guess its way forward.
		return errs.New(errs.Policy,
			fmt.Sprintf("the workflow stopped inside a task (state %s), which is not a safe resume boundary",
				e.State.Project.Workflow.State),
			"inspect the retained change set and worktree named in peerfoil status, then continue with the PeerFoil skills")
	case Approve:
		return errs.New(errs.Policy, "the phase is approved; the next phase needs your explicit go-ahead",
			"merge or review the integration branch, then start the next phase with the PeerFoil skills")
	case Paused:
		reason := "unspecified"
		if e.State.Project.Workflow.PausedFor != nil {
			reason = *e.State.Project.Workflow.PausedFor
		}
		return errs.New(errs.Policy, "the workflow is paused: "+reason,
			"resolve the blocker; production retries happen through peerfoil start once the pause is cleared")
	default:
		return errs.New(errs.Policy,
			fmt.Sprintf("the %s state belongs to the PeerFoil skills", e.State.Project.Workflow.State),
			"finish the decisions, architecture, and plan with the skills, then run peerfoil start")
	}
}

// recoverOpenInvocations handles journal entries whose operation has no result. An
// unknown mutating invocation is never rerun automatically: the user confirms the old
// process stopped, then the attempt is closed as failed and the records say so.
func (e *Engine) recoverOpenInvocations(open []store.OpenInvocation, confirmStopped bool) error {
	var names []string
	for _, invocation := range open {
		names = append(names, fmt.Sprintf("%s (%s, last recorded %s)",
			invocation.InvocationID, invocation.Operation, invocation.LastStatus))
	}
	if !confirmStopped {
		return errs.New(errs.Policy,
			"an earlier run was interrupted and these operations have no recorded result: "+strings.Join(names, "; "),
			"check that no producer process is still running and inspect the retained worktree, then run peerfoil resume --confirm-stopped")
	}
	for _, invocation := range open {
		if _, err := e.Journal.Result(invocation.Operation, invocation.InvocationID,
			store.StatusFailed, map[string]string{"reason": "interrupted; the user confirmed the process stopped"}); err != nil {
			return err
		}
		fmt.Fprintf(e.Out, "Closed the interrupted %s invocation %s as failed.\n",
			invocation.Operation, invocation.InvocationID)
	}
	// The task the interrupted producer was working on cannot be trusted as-is.
	for _, task := range e.State.Plan.Tasks() {
		if task.Status == "in_progress" {
			task.Status = "blocked"
		}
	}
	if err := e.SavePlan(); err != nil {
		return err
	}
	reason := "an interrupted producer run was closed; inspect its retained worktree and change set before the task runs again"
	if State(e.State.Project.Workflow.State) != Paused {
		if err := e.Pause(reason, nil); err != nil {
			return err
		}
	}
	return errs.New(errs.Policy, reason,
		"review the captured records under .peerfoil/evidence, reset the blocked task through the PeerFoil skills, then run peerfoil start")
}
