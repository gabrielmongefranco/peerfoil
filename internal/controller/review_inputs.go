// This file is part of PeerFoil.
// internal/controller/review_inputs.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Binds every controller review boundary to the accepted records and
// revisions used for validation, and refuses approval after those inputs change.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package controller

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/fsx"
	"github.com/gabrielmongefranco/peerfoil/internal/project"
)

var boundaryIDPattern = regexp.MustCompile(`^tr-[0-9]{4,}$`)

func (e *Engine) reviewInputs(source string) ([]byte, error) {
	projectBytes, err := e.readAcceptedRecord("project.json", 0)
	if err != nil {
		return nil, err
	}
	var storedProject project.Project
	if err := json.Unmarshal(projectBytes, &storedProject); err != nil {
		return nil, errs.New(errs.Validation, "the current project record cannot be read", "restore the validated project record before review")
	}
	records := map[string]string{}
	for _, item := range []struct {
		name     string
		revision int
	}{
		{"architecture.md", e.State.Project.Revisions.Architecture}, {"quality.md", e.State.Project.Revisions.Quality},
		{"decisions.md", 0}, {"plan.json", 0},
	} {
		raw, err := e.readAcceptedRecord(item.name, item.revision)
		if err != nil {
			return nil, err
		}
		records[item.name] = string(raw)
	}
	data, err := json.Marshal(struct {
		Project         string            `json:"project"`
		Source          string            `json:"source_revision"`
		Revisions       project.Revisions `json:"revisions"`
		StoredRevisions project.Revisions `json:"stored_revisions"`
		AcceptedProject project.Project   `json:"accepted_project"`
		LoadedProject   project.Project   `json:"loaded_project"`
		Plan            *project.Plan     `json:"plan"`
		Records         map[string]string `json:"records"`
	}{e.State.Project.ProjectID, source, e.State.Project.Revisions, storedProject.Revisions,
		stableProject(storedProject), stableProject(*e.State.Project), e.State.Plan, records})
	if err != nil || len(data) > 2*maxInputBytes {
		return nil, errs.New(errs.Validation, "the accepted review input bundle could not be retained within 512 KiB", "reduce the accepted records before validation")
	}
	return data, nil
}

// Workflow position and its save timestamp change during controller transitions.
// Every other accepted project field must remain fixed during review.
func stableProject(p project.Project) project.Project {
	p.Workflow = project.Workflow{}
	p.UpdatedAt = ""
	return p
}

func (e *Engine) saveReviewInputs(boundary project.Transition) error {
	raw, err := e.reviewInputs(*boundary.SourceRevision)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(e.State.Dir, "evidence"), 0o755); err != nil {
		return err
	}
	return fsx.WriteFileAtomic(filepath.Join(e.State.Dir, "evidence", boundary.TransitionID+".inputs.json"), raw, 0o644)
}

func (e *Engine) checkReviewInputs(boundary *project.Transition) error {
	invalid := func() error {
		return errs.New(errs.Validation, "accepted records changed or are missing since the validated review boundary",
			"route the change through change intake and rerun validation before review or approval")
	}
	if boundary == nil || boundary.SourceRevision == nil || !boundaryIDPattern.MatchString(boundary.TransitionID) || boundary.PlanRevision != e.State.Project.Revisions.Plan {
		return invalid()
	}
	retained, err := fsx.ReadRegularFile(e.State.Dir, filepath.Join("evidence", boundary.TransitionID+".inputs.json"), 2*maxInputBytes)
	if err != nil {
		return invalid()
	}
	current, err := e.reviewInputs(*boundary.SourceRevision)
	if err != nil || !bytes.Equal(current, retained) {
		return invalid()
	}
	return nil
}
