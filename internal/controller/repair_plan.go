// This file is part of PeerFoil.
// internal/controller/repair_plan.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Records bounded agreed repairs as plan changes with their own tasks,
// retained work, accepted scopes, and links to the originating reviews.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package controller

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/fsx"
	"github.com/gabrielmongefranco/peerfoil/internal/project"
	"github.com/gabrielmongefranco/peerfoil/internal/workspace"
)

var findingLineSuffix = regexp.MustCompile(`:[0-9]+(?::[0-9]+)?(?:-[0-9]+)?$`)

// prepareRepairPlan creates a candidate without changing accepted state. A finding
// must name a path within completed work; dismissed work never grants write access.
func (e *Engine) prepareRepairPlan(phaseReviewID string, findings []Finding) (*project.Plan, *project.Task, error) {
	raw, err := json.Marshal(e.State.Plan)
	if err != nil {
		return nil, nil, err
	}
	var plan project.Plan
	if err := json.Unmarshal(raw, &plan); err != nil {
		return nil, nil, err
	}
	affected := map[string]bool{}
	requirements := map[string]bool{}
	paths, reviews, checks := map[string]bool{}, map[string]bool{}, map[string]project.EvidenceRequirement{}
	var stage *project.Stage
	for _, finding := range findings {
		requirements[finding.Requirement] = true
		location := findingLineSuffix.ReplaceAllString(finding.Location, "")
		matched := false
		for pi := range plan.Phases {
			phase := &plan.Phases[pi]
			if phase.Status != "active" {
				continue
			}
			for si := range phase.Stages {
				candidate := &phase.Stages[si]
				if candidate.Status == "deferred" || candidate.Status == "removed" {
					continue
				}
				for _, task := range candidate.Tasks {
					if task.Status != "validated" && task.Status != "accepted" {
						continue
					}
					if workspace.CheckScope([]workspace.Change{{Path: location}}, task.AllowedPaths) != nil {
						continue
					}
					matched, affected[task.ID], stage = true, true, candidate
					for _, path := range task.AllowedPaths {
						paths[path] = true
					}
					for _, check := range task.RequiredEvidence {
						old, exists := checks[check.Name]
						if exists && old.Kind != check.Kind {
							return nil, nil, errs.New(errs.Validation, "affected tasks disagree on an evidence method", "resolve the Quality Contract through change intake")
						}
						rank := map[string]int{"not-applicable": 0, "recommended": 1, "required": 2}
						if !exists || rank[check.Level] > rank[old.Level] {
							checks[check.Name] = check
						}
					}
				}
			}
		}
		if !matched {
			return nil, nil, errs.New(errs.Policy, "a repair finding has no path within validated task scope", "clarify its file location or approve new scope through change intake")
		}
		for _, id := range finding.Reviews {
			if reviewIDPattern.MatchString(id) {
				reviews[id] = true
			}
		}
	}
	if stage == nil {
		return nil, nil, errs.New(errs.Policy, "the repair has no validated task scope", "record specific findings before repairing")
	}
	prior := plan.PlanRevision
	plan.SchemaVersion, plan.PlanRevision = 2, prior+1
	plan.RevisedAt, plan.RevisedBy = Timestamp(e.Now()), controllerActor()
	plan.RevisionReason = "Record the bounded repair agreed in " + phaseReviewID + "."
	taskNumber, changeNumber := 0, 0
	retained := []string{}
	for _, task := range plan.Tasks() {
		var n int
		fmt.Sscanf(task.ID, "tk-%d", &n)
		if n > taskNumber {
			taskNumber = n
		}
		if affected[task.ID] {
			task.PlanRevision = plan.PlanRevision
		} else {
			retained = append(retained, task.ID)
		}
	}
	for _, change := range plan.Changes {
		var n int
		fmt.Sscanf(change.ID, "cr-%d", &n)
		if n > changeNumber {
			changeNumber = n
		}
	}
	task := project.Task{ID: fmt.Sprintf("tk-%03d", taskNumber+1), Title: "Repair the agreed phase findings",
		Scope:        "Resolve the findings in " + phaseReviewID + " within the affected tasks' accepted paths.",
		AllowedPaths: sortedKeys(paths), Inputs: []string{".peerfoil/reviews/" + phaseReviewID + ".md"},
		Output:           "A captured repair with fresh evidence and different-family verification.",
		RequiredEvidence: []project.EvidenceRequirement{}, DependsOn: sortedKeys(affected),
		Acceptance: []string{"Every agreed finding is resolved and independently verified."},
		AuthorRole: "repair_producer", Status: "in_progress", PlanRevision: plan.PlanRevision, ArchitectureRevision: plan.ArchitectureRevision}
	for _, name := range sortedCheckNames(checks) {
		task.RequiredEvidence = append(task.RequiredEvidence, checks[name])
	}
	stage.Tasks = append(stage.Tasks, task)
	affectedIDs := append(sortedKeys(affected), task.ID)
	plan.Changes = append(plan.Changes, project.PlanChange{ID: fmt.Sprintf("cr-%04d", changeNumber+1),
		Summary: "Repair the findings recorded in " + phaseReviewID + ".", Placement: "current-stage", RecordedAt: Timestamp(e.Now()),
		PriorRevision: prior, Reason: plan.RevisionReason, AffectedTasks: affectedIDs, RetainedTasks: retained,
		Evidence: []string{}, Reviews: sortedKeys(reviews), Acceptance: "reviewed"})
	for i := range plan.Requirements {
		if requirements[plan.Requirements[i].ID] {
			plan.Requirements[i].Tasks = append(plan.Requirements[i].Tasks, task.ID)
			continue
		}
		for _, taskID := range plan.Requirements[i].Tasks {
			if affected[taskID] {
				plan.Requirements[i].Tasks = append(plan.Requirements[i].Tasks, task.ID)
				break
			}
		}
	}
	_, _, added := plan.FindTask(task.ID)
	return &plan, added, nil
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedCheckNames(values map[string]project.EvidenceRequirement) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// acceptRepairPlan retains the previous plan before publishing the repair revision.
// An interrupted multi-file save is detected on load and needs manual restoration.
func (e *Engine) acceptRepairPlan(plan *project.Plan, taskID string) error {
	archive := filepath.Join(e.State.Dir, "plans")
	if err := os.MkdirAll(archive, 0o755); err != nil {
		return err
	}
	for _, ext := range []string{"json", "md"} {
		raw, err := e.readAcceptedRecord("plan."+ext, 0)
		if err != nil {
			return err
		}
		name := filepath.Join(archive, fmt.Sprintf("plan-r%04d.%s", e.State.Plan.PlanRevision, ext))
		if _, err := os.Lstat(name); !os.IsNotExist(err) {
			return errs.New(errs.Validation, "the previous plan archive already exists or cannot be inspected", "inspect the retained plans before retrying repair intake")
		}
		if err := fsx.WriteFileAtomic(name, raw, 0o644); err != nil {
			return err
		}
	}
	e.State.Plan = plan
	e.State.Project.Revisions.Plan = plan.PlanRevision
	_, stage, _ := plan.FindTask(taskID)
	e.State.Project.Workflow.Stage, e.State.Project.Workflow.Task = &stage.ID, &taskID
	if err := e.SavePlan(); err != nil {
		return err
	}
	return e.saveProject()
}
