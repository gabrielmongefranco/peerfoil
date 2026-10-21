// This file is part of PeerFoil.
// internal/project/select.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Deterministic task selection: the next eligible task in the user's approved
// order, never chosen by a model.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package project

import (
	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

// Selection names the task the controller should act on next and where it sits.
type Selection struct {
	Phase *Phase
	Stage *Stage
	Task  *Task
}

// satisfied reports whether every named dependency is complete enough to build on:
// validated supplies inputs within the phase, accepted is fully approved work.
func satisfied(ids []string, byID map[string]*Task) bool {
	for _, id := range ids {
		task, ok := byID[id]
		if !ok {
			return false
		}
		if task.Status != "validated" && task.Status != "accepted" {
			return false
		}
	}
	return true
}

// ActiveTasks returns the tasks of the active phase, in plan order. Evidence
// revalidation and the review boundary operate on exactly this set: earlier approved
// phases are closed, and their evidence is not re-litigated by a later phase.
func (p *Plan) ActiveTasks() []*Task {
	var tasks []*Task
	for _, phase := range phasesByOrder(p.Phases) {
		if phase.Status != "active" {
			continue
		}
		for _, stage := range stagesByOrder(phase.Stages) {
			for index := range stage.Tasks {
				tasks = append(tasks, &stage.Tasks[index])
			}
		}
	}
	return tasks
}

// InFlight returns a task that is already being worked on, which the controller must
// continue before selecting new work. At most one exists when the records are honest;
// with more than one, the first in plan order is returned and the controller stops.
func (s *State) InFlight() []*Task {
	if s.Plan == nil {
		return nil
	}
	var open []*Task
	for _, task := range s.Plan.Tasks() {
		switch task.Status {
		case "in_progress", "produced":
			open = append(open, task)
		}
	}
	return open
}

// NextTask returns the first planned or ready task, in phase, stage, and listed order,
// inside the active phase, whose stage and task dependencies are satisfied. It returns
// a Validation error when no phase is active, and a nil Selection when the active
// phase simply has no eligible task left.
func (s *State) NextTask() (*Selection, error) {
	if s.Plan == nil {
		return nil, errs.New(errs.Validation, "no accepted plan exists",
			"create and accept a plan with the PeerFoil skills before running production")
	}
	var active *Phase
	for _, phase := range phasesByOrder(s.Plan.Phases) {
		if phase.Status == "active" {
			active = phase
			break
		}
	}
	if active == nil {
		return nil, errs.New(errs.Validation, "no phase is active",
			"a phase becomes active when its plan is accepted and the previous phase is approved")
	}

	byID := map[string]*Task{}
	for _, task := range s.Plan.Tasks() {
		byID[task.ID] = task
	}
	stageDone := func(stage *Stage) bool {
		for index := range stage.Tasks {
			status := stage.Tasks[index].Status
			if status != "validated" && status != "accepted" && status != "removed" && status != "deferred" {
				return false
			}
		}
		return true
	}
	stagesByID := map[string]*Stage{}
	for phaseIndex := range s.Plan.Phases {
		for stageIndex := range s.Plan.Phases[phaseIndex].Stages {
			stage := &s.Plan.Phases[phaseIndex].Stages[stageIndex]
			stagesByID[stage.ID] = stage
		}
	}

	for _, stage := range stagesByOrder(active.Stages) {
		// A deferred or removed stage was dismissed by an accepted change; running
		// its tasks would reverse that decision. The loader rejects records that
		// leave runnable tasks inside such a stage, and selection skips it too.
		if stage.Status == "deferred" || stage.Status == "removed" {
			continue
		}
		ready := true
		for _, dependency := range stage.DependsOn {
			upstream, ok := stagesByID[dependency]
			if !ok || !stageDone(upstream) {
				ready = false
				break
			}
		}
		if !ready {
			continue
		}
		for index := range stage.Tasks {
			task := &stage.Tasks[index]
			if task.Status != "planned" && task.Status != "ready" {
				continue
			}
			if !satisfied(task.DependsOn, byID) {
				continue
			}
			return &Selection{Phase: active, Stage: stage, Task: task}, nil
		}
	}
	return nil, nil
}
