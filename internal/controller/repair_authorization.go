// This file is part of PeerFoil.
// internal/controller/repair_authorization.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Carries a verified, agreed repair plan into later phases without
// inventing another model approval or authorizing changed plan content.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package controller

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"slices"

	"github.com/gabrielmongefranco/peerfoil/internal/fsx"
	"github.com/gabrielmongefranco/peerfoil/internal/project"
	"github.com/gabrielmongefranco/peerfoil/internal/provider"
)

// repairAgreementAuthorizesPlan requires the original two-family repair agreement,
// a retained independent verification, and its completed approval transition.
func (e *Engine) repairAgreementAuthorizesPlan(reviews []reviewRecord) bool {
	for i := len(e.State.History) - 1; i >= 0; i-- {
		boundary := &e.State.History[i]
		if boundary.ToState != string(Review) || boundary.PlanRevision != e.State.Project.Revisions.Plan || len(boundary.Refs["change_sets"]) != 1 {
			continue
		}
		checkpoint, err := e.loadRepairVerification(boundary)
		if err != nil || e.checkRepairApproval(checkpoint) != nil {
			continue
		}
		_, _, task := e.State.Plan.FindTask(checkpoint.Task)
		if task == nil || task.AuthorRole != "repair_producer" || task.Status != "accepted" || task.PlanRevision != checkpoint.Revisions.Plan {
			continue
		}
		approved := false
		for _, transition := range e.State.History[i+1:] {
			if transition.ToState == string(Approve) && transition.PlanRevision == checkpoint.Revisions.Plan && transition.SourceRevision != nil && *transition.SourceRevision == checkpoint.Source &&
				slices.Contains(transition.Refs["reviews"], checkpoint.Approved.Review) && slices.Contains(transition.Refs["phase_reviews"], checkpoint.PhaseReview) && slices.Contains(transition.Refs["change_sets"], checkpoint.ChangeSet) {
				approved = true
			}
		}
		if !approved || !e.sameAuthorizedPlan(boundary) {
			continue
		}
		for _, change := range e.State.Plan.Changes {
			if change.PriorRevision != checkpoint.Revisions.Plan-1 || !slices.Contains(change.AffectedTasks, checkpoint.Task) {
				continue
			}
			families := map[string]bool{}
			for _, review := range reviews {
				if !slices.Contains(change.Reviews, review.ID) || review.Kind != "phase" || review.Decision != "repair" || review.PlanRevision != change.PriorRevision ||
					review.ArchRevision != checkpoint.Revisions.Architecture || review.QualityRevision != checkpoint.Revisions.Quality || review.SourceRevision != checkpoint.Base {
					continue
				}
				if review.LineageRoot == provider.LineageOpenAIGPT || review.LineageRoot == provider.LineageAnthropicClaude {
					families[review.LineageRoot] = true
				}
			}
			if len(families) == 2 {
				return true
			}
		}
	}
	return false
}

// Progress statuses may advance without changing the accepted assignments. All
// other plan content must match what the independent repair verifier received.
func (e *Engine) sameAuthorizedPlan(boundary *project.Transition) bool {
	if !boundaryIDPattern.MatchString(boundary.TransitionID) {
		return false
	}
	raw, err := fsx.ReadRegularFile(e.State.Dir, filepath.Join("evidence", boundary.TransitionID+".inputs.json"), 2*maxInputBytes)
	if err != nil {
		return false
	}
	var bundle struct {
		Plan *project.Plan `json:"plan"`
	}
	if json.Unmarshal(raw, &bundle) != nil || bundle.Plan == nil {
		return false
	}
	old, err := planWithoutProgress(bundle.Plan)
	if err != nil {
		return false
	}
	current, err := planWithoutProgress(e.State.Plan)
	return err == nil && bytes.Equal(old, current)
}

func planWithoutProgress(plan *project.Plan) ([]byte, error) {
	raw, err := json.Marshal(plan)
	if err != nil {
		return nil, err
	}
	var copy project.Plan
	if err := json.Unmarshal(raw, &copy); err != nil {
		return nil, err
	}
	for pi := range copy.Phases {
		phase := &copy.Phases[pi]
		phase.Status = authorizationStatus(phase.Status)
		for si := range phase.Stages {
			stage := &phase.Stages[si]
			stage.Status = authorizationStatus(stage.Status)
			for ti := range stage.Tasks {
				stage.Tasks[ti].Status = authorizationStatus(stage.Tasks[ti].Status)
			}
		}
	}
	return json.Marshal(copy)
}

// Dismissal and failure statuses need change intake. Only ordinary progress can
// advance without altering the plan authorization.
func authorizationStatus(status string) string {
	switch status {
	case "deferred", "removed", "blocked", "stale":
		return status
	default:
		return ""
	}
}
