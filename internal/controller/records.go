// This file is part of PeerFoil.
// internal/controller/records.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Writes the accepted project records the controller owns: history
// transitions, plan updates with the regenerated plan.md, change sets, and evidence
// records. Every write is atomic and keeps identifiers sequential.
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
	"strings"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/evidence"
	"github.com/gabrielmongefranco/peerfoil/internal/fsx"
	"github.com/gabrielmongefranco/peerfoil/internal/project"
	"github.com/gabrielmongefranco/peerfoil/internal/version"
)

// Timestamp is the record timestamp format every schema requires.
func Timestamp(now time.Time) string {
	return now.UTC().Format("2006-01-02T15:04:05Z")
}

// controllerActor is the actor the controller itself records for transitions it
// accepts. The controller is a program, not a model, so it has no model lineage.
func controllerActor() project.Actor {
	model := "peerfoil-core"
	return project.Actor{Role: "coordinator", Tool: "other", Model: &model,
		LineageRoot: "unknown"}
}

// nextID returns the next identifier with the given prefix and width by scanning the
// existing records, so identifiers stay sequential across restarts and chats.
func nextID(dir, prefix string, width int) (string, error) {
	pattern := regexp.MustCompile("^" + regexp.QuoteMeta(prefix) + `-([0-9]{` + fmt.Sprint(width) + `,})`)
	highest := 0
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return "", errs.Wrap(errs.Configuration, "reading "+dir, err)
	}
	for _, entry := range entries {
		match := pattern.FindStringSubmatch(entry.Name())
		if match == nil {
			continue
		}
		var number int
		fmt.Sscanf(match[1], "%d", &number)
		if number > highest {
			highest = number
		}
	}
	return fmt.Sprintf("%s-%0*d", prefix, width, highest+1), nil
}

// AppendTransition validates the move, writes the new history line atomically, and
// updates project.json's workflow and updated_at in the same step, so the loaded
// state's consistency check keeps passing.
func (e *Engine) AppendTransition(to State, refs map[string][]string, summary string, sourceRevision *string) error {
	state := e.State
	from := State(state.Project.Workflow.State)
	if err := Move(from, to); err != nil {
		return err
	}
	lastID := "tr-0000"
	if len(state.History) > 0 {
		lastID = state.History[len(state.History)-1].TransitionID
	}
	var lastNumber int
	fmt.Sscanf(strings.TrimPrefix(lastID, "tr-"), "%d", &lastNumber)
	// An empty reference list marshals as null, which the transition schema rejects
	// and would make the next load refuse the whole project, so empty lists are
	// dropped rather than written.
	for key, values := range refs {
		if len(values) == 0 {
			delete(refs, key)
		}
	}
	if len(refs) == 0 {
		refs = nil
	}
	// The schema caps a summary at 300 characters; the full detail lives in the
	// referenced records, so a long reason is cut rather than invalidating history.
	if runes := []rune(summary); len(runes) > 300 {
		summary = string(runes[:297]) + "..."
	}
	fromName := string(from)
	transition := project.Transition{
		RecordType:    "transition",
		SchemaVersion: 1,
		TransitionID:  fmt.Sprintf("tr-%04d", lastNumber+1),
		ProjectID:     state.Project.ProjectID,
		At:            Timestamp(e.Now()),
		FromState:     &fromName,
		ToState:       string(to),
		Actor:         controllerActor(),
		PlanRevision:  state.Project.Revisions.Plan,
		Refs:          refs,
		Summary:       summary,
	}
	transition.SourceRevision = sourceRevision

	line, err := json.Marshal(transition)
	if err != nil {
		return errs.Wrap(errs.Internal, "encoding the transition", err)
	}
	historyPath := filepath.Join(state.Dir, "history.jsonl")
	existing, err := os.ReadFile(historyPath)
	if err != nil && !os.IsNotExist(err) {
		return errs.Wrap(errs.Configuration, "reading history.jsonl", err)
	}
	updated := append(existing, append(line, '\n')...)
	if err := fsx.WriteFileAtomic(historyPath, updated, 0o644); err != nil {
		return err
	}
	state.History = append(state.History, transition)

	state.Project.Workflow.State = string(to)
	state.Project.UpdatedAt = transition.At
	if to != Paused {
		state.Project.Workflow.PausedFor = nil
	}
	return e.saveProject()
}

// Pause records why the workflow stopped and moves to paused.
func (e *Engine) Pause(reason string, refs map[string][]string) error {
	e.State.Project.Workflow.PausedFor = &reason
	if err := e.AppendTransition(Paused, refs, "Paused: "+reason, nil); err != nil {
		return err
	}
	// AppendTransition clears the pause reason only for non-paused targets.
	e.State.Project.Workflow.PausedFor = &reason
	return e.saveProject()
}

func (e *Engine) saveProject() error {
	data, err := json.MarshalIndent(e.State.Project, "", "  ")
	if err != nil {
		return errs.Wrap(errs.Internal, "encoding project.json", err)
	}
	return fsx.WriteFileAtomic(filepath.Join(e.State.Dir, "project.json"), append(data, '\n'), 0o644)
}

// SavePlan writes plan.json and regenerates plan.md from it, because plan.json is
// authoritative and the Markdown twin must never disagree with it.
func (e *Engine) SavePlan() error {
	plan := e.State.Plan
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return errs.Wrap(errs.Internal, "encoding plan.json", err)
	}
	if err := fsx.WriteFileAtomic(filepath.Join(e.State.Dir, "plan.json"), append(data, '\n'), 0o644); err != nil {
		return err
	}
	rendered := renderPlanMarkdown(e.State.Project, plan)
	return fsx.WriteFileAtomic(filepath.Join(e.State.Dir, "plan.md"), []byte(rendered), 0o644)
}

func renderPlanMarkdown(proj *project.Project, plan *project.Plan) string {
	var b strings.Builder
	assurance := "Guided"
	fmt.Fprintf(&b, "<!-- PeerFoil project record: plan. Generated by %s (%s) from plan.json revision %d. plan.json is authoritative; regenerate this file after every plan change. -->\n\n",
		version.Release, assurance, plan.PlanRevision)
	fmt.Fprintf(&b, "# Plan for %s\n\n", proj.Name)
	fmt.Fprintf(&b, "Project: `%s` · Assurance: %s · Plan revision: %d · Status: %s\n\n",
		proj.ProjectID, assurance, plan.PlanRevision, plan.Status)
	reviser := plan.RevisedBy
	model := "unknown"
	if reviser.Model != nil {
		model = *reviser.Model
	}
	_ = model
	fmt.Fprintf(&b, "- **Revised at:** %s by %s (%s, %s)\n", plan.RevisedAt, reviser.Role, reviser.Tool, reviser.LineageRoot)
	fmt.Fprintf(&b, "- **Reason:** %s\n", plan.RevisionReason)
	fmt.Fprintf(&b, "- **Architecture revision:** %d · **Quality revision:** %d\n\n",
		plan.ArchitectureRevision, plan.QualityRevision)

	b.WriteString("## Phases and stages\n\n")
	phases := append([]project.Phase(nil), plan.Phases...)
	sort.SliceStable(phases, func(a, c int) bool { return phases[a].Order < phases[c].Order })
	for _, phase := range phases {
		fmt.Fprintf(&b, "### %s — %s (%s)\n\n%s\n\n", phase.ID, phase.Title, phase.Status, phase.Outcome)
		b.WriteString("| Order | Stage | Outcome | Status | Depends on |\n|---:|---|---|---|---|\n")
		stages := append([]project.Stage(nil), phase.Stages...)
		sort.SliceStable(stages, func(a, c int) bool { return stages[a].Order < stages[c].Order })
		for _, stage := range stages {
			depends := "—"
			if len(stage.DependsOn) > 0 {
				depends = strings.Join(stage.DependsOn, ", ")
			}
			fmt.Fprintf(&b, "| %d | %s %s | %s | %s | %s |\n",
				stage.Order, stage.ID, stage.Title, stage.Outcome, stage.Status, depends)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Tasks\n\n| Task | Stage | Title | Status | Required evidence |\n|---|---|---|---|---|\n")
	for _, phase := range phases {
		for _, stage := range phase.Stages {
			for _, task := range stage.Tasks {
				var names []string
				for _, requirement := range task.RequiredEvidence {
					if requirement.Level == "required" {
						names = append(names, requirement.Name)
					}
				}
				fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
					task.ID, stage.ID, task.Title, task.Status, strings.Join(names, ", "))
			}
		}
	}

	b.WriteString("\n## Requirements\n\n| ID | Requirement | Source | Tasks |\n|---|---|---|---|\n")
	for _, requirement := range plan.Requirements {
		tasks := "—"
		if len(requirement.Tasks) > 0 {
			tasks = strings.Join(requirement.Tasks, ", ")
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", requirement.ID, requirement.Text, requirement.Source, tasks)
	}

	b.WriteString("\n## Backlog\n\n| ID | Request | Placement | Recorded |\n|---|---|---|---|\n")
	if len(plan.Backlog) == 0 {
		b.WriteString("| none | — | — | — |\n")
	}
	for _, item := range plan.Backlog {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", item.ID, item.Summary, item.Placement, item.RecordedAt)
	}

	b.WriteString("\n## Change history\n\n| Change | Prior revision | Placement | Reason | Affected tasks | Retained tasks | Evidence / reviews | Acceptance |\n|---|---:|---|---|---|---|---|---|\n")
	if len(plan.Changes) == 0 {
		b.WriteString("| none | — | — | — | — | — | — | — |\n")
	}
	for _, change := range plan.Changes {
		refs := strings.Join(append(append([]string{}, change.Evidence...), change.Reviews...), ", ")
		if refs == "" {
			refs = "—"
		}
		fmt.Fprintf(&b, "| %s | %d | %s | %s | %s | %s | %s | %s |\n",
			change.ID, change.PriorRevision, change.Placement, change.Reason,
			listOrDash(change.AffectedTasks), listOrDash(change.RetainedTasks), refs, change.Acceptance)
	}
	return b.String()
}

func listOrDash(items []string) string {
	if len(items) == 0 {
		return "—"
	}
	return strings.Join(items, ", ")
}

// ChangeSet is the provenance record saved before any other agent edits the work.
type ChangeSet struct {
	ID            string
	TaskID        string
	TaskTitle     string
	CaptureStatus string
	StartedAt     string
	Attempt       int
	Baseline      string
	BaseRevision  string
	SourceRev     string
	Author        project.Actor
	CapturedAt    string
	PatchPath     string
	PatchSHA256   string
	Changed       []string // "status<TAB>path" pairs rendered into the table
	Summary       string
	Discovered    []string // work the producer found that change intake must absorb
}

// WriteChangeSet renders the change-set record in the template's field shape.
func (e *Engine) WriteChangeSet(cs *ChangeSet) error {
	dir := filepath.Join(e.State.Dir, "evidence")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errs.Wrap(errs.Configuration, "creating the evidence directory", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- PeerFoil project record: change set. Generated by %s (Guided). Keep the field names. -->\n\n", version.Release)
	fmt.Fprintf(&b, "# Change set %s — %s\n\n", cs.ID, cs.TaskTitle)
	fmt.Fprintf(&b, "Project: `%s` · Assurance: Guided\n\n", e.State.Project.ProjectID)
	fmt.Fprintf(&b, "- **Task:** %s\n", cs.TaskID)
	fmt.Fprintf(&b, "- **Capture status:** %s\n", cs.CaptureStatus)
	fmt.Fprintf(&b, "- **Started at:** %s\n", cs.StartedAt)
	fmt.Fprintf(&b, "- **Attempt:** %d\n", cs.Attempt)
	fmt.Fprintf(&b, "- **Baseline:** %s\n", cs.Baseline)
	fmt.Fprintf(&b, "- **Base revision:** %s\n", cs.BaseRevision)
	fmt.Fprintf(&b, "- **Source revision:** %s\n", valueOrNull(cs.SourceRev))
	author := cs.Author
	fmt.Fprintf(&b, "- **Author:** %s (%s, %s, %s, %s, session %s)\n",
		author.Role, author.Tool, deref(author.Model), deref(author.Effort),
		author.LineageRoot, deref(author.Session))
	fmt.Fprintf(&b, "- **Captured at:** %s\n", valueOrNull(cs.CapturedAt))
	fmt.Fprintf(&b, "- **Patch:** %s\n", valueOrNull(cs.PatchPath))
	fmt.Fprintf(&b, "- **Patch SHA-256:** %s\n", valueOrNull(cs.PatchSHA256))
	fmt.Fprintf(&b, "- **Input snapshot:** null\n")
	fmt.Fprintf(&b, "- **Quality revision:** %d\n", e.State.Project.Revisions.Quality)
	fmt.Fprintf(&b, "- **Plan revision:** %d · **Architecture revision:** %d\n\n",
		e.State.Project.Revisions.Plan, e.State.Project.Revisions.Architecture)
	b.WriteString("## Changed paths\n\n| Change | Path |\n|---|---|\n")
	if len(cs.Changed) == 0 {
		b.WriteString("| none | — |\n")
	}
	for _, change := range cs.Changed {
		kind, path, _ := strings.Cut(change, "\t")
		fmt.Fprintf(&b, "| %s | %s |\n", kind, path)
	}
	b.WriteString("\n## Input files\n\n| Path | Kind | SHA-256 or absent |\n|---|---|---|\n")
	// The controller launches producers from a committed base revision, so the base
	// commit itself identifies every input; per-file snapshots are a guided-mode
	// mechanism for uncommitted work (D-0025).
	b.WriteString("| (all inputs are at the base revision) | file | absent |\n")
	fmt.Fprintf(&b, "\n## Summary\n\n%s\n\n## Discovered work\n\n", cs.Summary)
	if len(cs.Discovered) == 0 {
		b.WriteString("None.\n")
	}
	for _, item := range cs.Discovered {
		fmt.Fprintf(&b, "- %s\n", item)
	}
	b.WriteString("\n## Later edits by other agents\n\nNone.\n")
	return fsx.WriteFileAtomic(filepath.Join(dir, cs.ID+".md"), []byte(b.String()), 0o644)
}

// WriteEvidenceRecord renders one evidence record in the template's field shape.
func (e *Engine) WriteEvidenceRecord(record evidence.Record, changeSetID, procedure string) error {
	dir := filepath.Join(e.State.Dir, "evidence")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errs.Wrap(errs.Configuration, "creating the evidence directory", err)
	}
	result := map[string]string{"pass": "pass", "fail": "fail", "timeout": "blocked", "error": "blocked"}[record.Status]
	if result == "" {
		result = "not-run"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- PeerFoil project record: evidence. Generated by %s (Guided). Keep the field names. -->\n\n", version.Release)
	fmt.Fprintf(&b, "# Evidence %s — %s\n\n", record.ID, record.Name)
	fmt.Fprintf(&b, "Project: `%s` · Assurance: Guided\n\n", e.State.Project.ProjectID)
	fmt.Fprintf(&b, "- **Name:** %s\n", record.Name)
	fmt.Fprintf(&b, "- **Kind:** %s\n", record.Kind)
	fmt.Fprintf(&b, "- **Level:** %s\n", record.Level)
	fmt.Fprintf(&b, "- **Task:** %s\n", record.TaskID)
	fmt.Fprintf(&b, "- **Change set:** %s\n", valueOrNull(changeSetID))
	fmt.Fprintf(&b, "- **Procedure:** %s\n", procedure)
	fmt.Fprintf(&b, "- **Result:** %s\n", result)
	fmt.Fprintf(&b, "- **Exit code:** %d\n", record.ExitCode)
	fmt.Fprintf(&b, "- **Duration (seconds):** %.1f\n", record.DurationSeconds)
	fmt.Fprintf(&b, "- **Tool version:** %s\n", valueOrNull(record.ToolVersion))
	fmt.Fprintf(&b, "- **Source revision:** %s\n", valueOrNull(record.SourceRevision))
	fmt.Fprintf(&b, "- **Plan revision:** %d\n", record.PlanRevision)
	fmt.Fprintf(&b, "- **Input snapshot:** %s\n", valueOrNull(record.RetainedSHA256))
	fmt.Fprintf(&b, "- **Recorded at:** %s\n", record.At)
	fmt.Fprintf(&b, "- **Recorded by:** coordinator (other, unknown, session null)\n")
	// The record is versioned and shareable, so the retained-output path is stored
	// relative to .peerfoil; an absolute path under a home directory names the user.
	retained := record.RetainedPath
	if relative, err := filepath.Rel(e.State.Dir, retained); err == nil && !strings.HasPrefix(relative, "..") {
		retained = filepath.ToSlash(relative)
	}
	fmt.Fprintf(&b, "- **Retained output:** %s\n\n## Notes\n\nNone.\n", valueOrNull(retained))
	return fsx.WriteFileAtomic(filepath.Join(dir, record.ID+".md"), []byte(b.String()), 0o644)
}

func valueOrNull(value string) string {
	if strings.TrimSpace(value) == "" {
		return "null"
	}
	return value
}

func deref(value *string) string {
	if value == nil || *value == "" {
		return "null"
	}
	return *value
}
