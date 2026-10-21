// This file is part of PeerFoil.
// internal/cli/status.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: The peerfoil status command: a plain-language view of the project's state,
// quality, blocker, and next action, read from files alone.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/project"
	"github.com/gabrielmongefranco/peerfoil/internal/schema"
)

// stateMeanings explains each workflow state in one sentence a person can act on.
var stateMeanings = map[string]string{
	"define":    "The important decisions are being collected.",
	"architect": "The architecture is being written and reviewed.",
	"plan":      "The plan is being written and reviewed.",
	"produce":   "A producer can work on the next task.",
	"validate":  "A produced change is waiting for its checks.",
	"review":    "The phase is waiting for its independent reviews.",
	"repair":    "An accepted repair is in progress.",
	"approve":   "The phase is approved; the next phase needs your go-ahead.",
	"paused":    "The workflow is paused and needs you.",
}

func statusLoad(env *Env) (*project.State, error) {
	schemas, err := schema.LoadEmbedded()
	if err != nil {
		return nil, errs.Wrap(errs.Internal, "loading the record schemas", err)
	}
	root, err := project.FindRoot(env.Dir)
	if err != nil {
		return nil, err
	}
	return project.Load(root, schemas)
}

func runStatus(ctx context.Context, env *Env, args []string) error {
	if len(args) != 0 {
		return errs.New(errs.Usage, "status takes no arguments", "run peerfoil status")
	}
	state, err := statusLoad(env)
	if err != nil {
		return err
	}
	printStatus(env, state)
	return nil
}

func printStatus(env *Env, state *project.State) {
	p := state.Project
	assurance := "Guided"
	if p.Assurance == "enforced" {
		assurance = "Enforced"
	}
	fmt.Fprintf(env.Stdout, "Project:  %s (%s) · Pack: %s · Assurance: %s\n",
		p.Name, p.ProjectID, p.Pack.ID, assurance)
	meaning := stateMeanings[p.Workflow.State]
	fmt.Fprintf(env.Stdout, "State:    %s — %s\n", p.Workflow.State, meaning)

	position := "none yet"
	if p.Workflow.Phase != nil {
		parts := []string{*p.Workflow.Phase}
		if p.Workflow.Stage != nil {
			parts = append(parts, *p.Workflow.Stage)
		}
		if p.Workflow.Task != nil {
			parts = append(parts, *p.Workflow.Task)
		}
		position = strings.Join(parts, " / ")
	}
	fmt.Fprintf(env.Stdout, "Position: %s\n", position)

	if state.Plan != nil {
		fmt.Fprintf(env.Stdout, "Plan:     revision %d, %s\n", state.Plan.PlanRevision, state.Plan.Status)
		counts := map[string]int{}
		for _, task := range state.Plan.Tasks() {
			counts[task.Status]++
		}
		var quality []string
		for _, status := range []string{"accepted", "validated", "produced", "in_progress", "ready", "planned", "blocked", "stale", "deferred"} {
			if counts[status] > 0 {
				quality = append(quality, fmt.Sprintf("%d %s", counts[status], strings.ReplaceAll(status, "_", " ")))
			}
		}
		fmt.Fprintf(env.Stdout, "Tasks:    %s\n", strings.Join(quality, " · "))
	} else {
		fmt.Fprintln(env.Stdout, "Plan:     none yet")
	}

	blocker := "none"
	if p.Workflow.PausedFor != nil && *p.Workflow.PausedFor != "" {
		blocker = *p.Workflow.PausedFor
	}
	fmt.Fprintf(env.Stdout, "Blocker:  %s\n", blocker)
	fmt.Fprintf(env.Stdout, "Next:     %s\n", nextAction(state))
	if len(state.History) > 0 {
		last := state.History[len(state.History)-1]
		fmt.Fprintf(env.Stdout, "Last step: %s at %s — %s\n", last.TransitionID, last.At, last.Summary)
	}
}

// nextAction names one concrete step. Core Alpha supports the production part of the
// journey; the earlier states belong to the PeerFoil skills.
func nextAction(state *project.State) string {
	switch state.Project.Workflow.State {
	case "define", "architect", "plan":
		return "finish the decisions, architecture, and plan with the PeerFoil skills, then run peerfoil start"
	case "produce":
		if open := state.InFlight(); len(open) > 0 {
			return "run peerfoil resume so the interrupted task " + open[0].ID + " is inspected"
		}
		if selection, err := state.NextTask(); err == nil && selection != nil {
			return "run peerfoil start to produce " + selection.Task.ID
		}
		if state.Plan != nil {
			for _, task := range state.Plan.Tasks() {
				if task.Status == "blocked" || task.Status == "stale" {
					return "task " + task.ID + " is " + task.Status + "; resolve it through the PeerFoil skills change intake"
				}
			}
			for _, task := range state.Plan.ActiveTasks() {
				switch task.Status {
				case "validated", "accepted", "deferred", "removed":
					continue
				}
				return "task " + task.ID + " cannot be selected; resolve its dependencies through the PeerFoil skills change intake"
			}
		}
		return "every eligible task is done; run peerfoil start to move the phase to review"
	case "validate":
		return "the workflow stopped inside a task; inspect the latest change set and continue with the PeerFoil skills"
	case "review":
		return "run peerfoil resume to start the two-family phase review"
	case "repair":
		return "the workflow stopped inside the repair; inspect the latest records and continue with the PeerFoil skills"
	case "approve":
		return "the phase is approved; the next phase starts when you ask for it"
	case "paused":
		return "resolve the blocker above, then continue with the PeerFoil skills; peerfoil start retries production once the records leave the paused state"
	}
	return "run peerfoil doctor"
}
