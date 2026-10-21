// This file is part of PeerFoil.
// internal/project/types.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Typed views of the accepted .peerfoil records: project, plan, and history.
// The JSON shapes are defined by the schemas; these structs mirror them for program use.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package project

// Actor identifies who performed a recorded step, with the model lineage the
// independence rules need. Pointer fields are nullable in the records.
type Actor struct {
	Role        string  `json:"role"`
	Tool        string  `json:"tool"`
	Model       *string `json:"model"`
	Effort      *string `json:"effort"`
	LineageRoot string  `json:"lineage_root"`
	Session     *string `json:"session"`
}

// Seat is a configured role assignment: which tool and model fill a role.
type Seat struct {
	Tool   string `json:"tool"`
	Model  string `json:"model"`
	Effort string `json:"effort"`
}

// Workflow is the project's current position in the fixed lifecycle.
type Workflow struct {
	State     string  `json:"state"`
	Phase     *string `json:"phase"`
	Stage     *string `json:"stage"`
	Task      *string `json:"task"`
	PausedFor *string `json:"paused_for"`
}

// Revisions holds the accepted revision numbers; zero means not yet accepted.
type Revisions struct {
	Architecture int `json:"architecture"`
	Quality      int `json:"quality"`
	Plan         int `json:"plan"`
}

// ReviewSettings are the pass limits the review coordinator enforces.
type ReviewSettings struct {
	DefaultPasses            int `json:"default_passes"`
	MaxPasses                int `json:"max_passes"`
	RepairSelectionPasses    int `json:"repair_selection_passes"`
	RepairSelectionMaxPasses int `json:"repair_selection_max_passes"`
	RepairCycles             int `json:"repair_cycles"`
}

// Settings are the accepted model seats and review limits.
type Settings struct {
	Roles          map[string]Seat `json:"roles"`
	PhaseReviewers []Seat          `json:"phase_reviewers"`
	Review         ReviewSettings  `json:"review"`
}

// Pack names the selected project pack.
type Pack struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// RulesSource says where the project's instruction rules come from.
type RulesSource struct {
	Mode   string `json:"mode"`
	Path   string `json:"path,omitempty"`
	Source string `json:"source,omitempty"`
}

// Environment retains the setup checks and their UTC check time across state saves.
type Environment struct {
	CheckedAt string             `json:"checked_at"`
	Checks    []EnvironmentCheck `json:"checks"`
}

// EnvironmentCheck is one recorded prerequisite result; detail is bounded by the schema.
type EnvironmentCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// Project is .peerfoil/project.json.
type Project struct {
	RecordType    string       `json:"record_type"`
	SchemaVersion int          `json:"schema_version"`
	ProjectID     string       `json:"project_id"`
	Name          string       `json:"name"`
	Goal          string       `json:"goal"`
	Pack          Pack         `json:"pack"`
	Profile       string       `json:"profile"`
	RulesSource   RulesSource  `json:"rules_source"`
	Assurance     string       `json:"assurance"`
	Release       string       `json:"release"`
	CreatedAt     string       `json:"created_at"`
	UpdatedAt     string       `json:"updated_at"`
	Workflow      Workflow     `json:"workflow"`
	Revisions     Revisions    `json:"revisions"`
	Settings      Settings     `json:"settings"`
	Environment   *Environment `json:"environment,omitempty"`
}

// EvidenceRequirement names one kind of evidence a task needs.
type EvidenceRequirement struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Level string `json:"level"`
}

// Task is one small assignment for one producer call.
type Task struct {
	ID                   string                `json:"id"`
	Title                string                `json:"title"`
	Scope                string                `json:"scope"`
	AllowedPaths         []string              `json:"allowed_paths"`
	Inputs               []string              `json:"inputs"`
	Output               string                `json:"output"`
	RequiredEvidence     []EvidenceRequirement `json:"required_evidence"`
	DependsOn            []string              `json:"depends_on"`
	Acceptance           []string              `json:"acceptance"`
	AuthorRole           string                `json:"author_role"`
	Status               string                `json:"status"`
	PlanRevision         int                   `json:"plan_revision"`
	ArchitectureRevision int                   `json:"architecture_revision"`
}

// Stage groups tasks into one user-visible outcome.
type Stage struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Outcome   string   `json:"outcome"`
	Order     int      `json:"order"`
	Status    string   `json:"status"`
	DependsOn []string `json:"depends_on"`
	Tasks     []Task   `json:"tasks"`
}

// Phase is one complete, reviewable unit of the plan.
type Phase struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Outcome string  `json:"outcome"`
	Order   int     `json:"order"`
	Status  string  `json:"status"`
	Stages  []Stage `json:"stages"`
}

// Requirement links recorded needs to the tasks that implement them.
type Requirement struct {
	ID     string   `json:"id"`
	Text   string   `json:"text"`
	Source string   `json:"source"`
	Tasks  []string `json:"tasks"`
}

// PlanChange records one accepted change request and its effect on the plan.
type PlanChange struct {
	ID            string   `json:"id"`
	Summary       string   `json:"summary"`
	Placement     string   `json:"placement"`
	RecordedAt    string   `json:"recorded_at"`
	PriorRevision int      `json:"prior_revision"`
	Reason        string   `json:"reason"`
	AffectedTasks []string `json:"affected_tasks"`
	RetainedTasks []string `json:"retained_tasks"`
	Evidence      []string `json:"evidence"`
	Reviews       []string `json:"reviews"`
	Acceptance    string   `json:"acceptance"`
}

// BacklogItem is a recorded request that is not scheduled yet.
type BacklogItem struct {
	ID         string `json:"id"`
	Summary    string `json:"summary"`
	Placement  string `json:"placement"`
	RecordedAt string `json:"recorded_at"`
}

// Plan is .peerfoil/plan.json, schema version 1 or 2. Version 1 has no changes.
type Plan struct {
	RecordType           string        `json:"record_type"`
	SchemaVersion        int           `json:"schema_version"`
	ProjectID            string        `json:"project_id"`
	PlanRevision         int           `json:"plan_revision"`
	ArchitectureRevision int           `json:"architecture_revision"`
	QualityRevision      int           `json:"quality_revision"`
	Status               string        `json:"status"`
	RevisedAt            string        `json:"revised_at"`
	RevisedBy            Actor         `json:"revised_by"`
	RevisionReason       string        `json:"revision_reason"`
	Phases               []Phase       `json:"phases"`
	Requirements         []Requirement `json:"requirements"`
	Changes              []PlanChange  `json:"changes,omitempty"`
	Backlog              []BacklogItem `json:"backlog"`
}

// Transition is one line of .peerfoil/history.jsonl.
type Transition struct {
	RecordType     string              `json:"record_type"`
	SchemaVersion  int                 `json:"schema_version"`
	TransitionID   string              `json:"transition_id"`
	ProjectID      string              `json:"project_id"`
	At             string              `json:"at"`
	FromState      *string             `json:"from_state"`
	ToState        string              `json:"to_state"`
	Actor          Actor               `json:"actor"`
	PlanRevision   int                 `json:"plan_revision"`
	SourceRevision *string             `json:"source_revision"`
	Refs           map[string][]string `json:"refs,omitempty"`
	Summary        string              `json:"summary"`
}

// Tasks returns every task in plan order: phases by order, stages by order, tasks in
// their listed order. The listed order is the planner's sequence within a stage.
func (p *Plan) Tasks() []*Task {
	var tasks []*Task
	for _, phase := range phasesByOrder(p.Phases) {
		for _, stage := range stagesByOrder(phase.Stages) {
			for index := range stage.Tasks {
				tasks = append(tasks, &stage.Tasks[index])
			}
		}
	}
	return tasks
}

// FindTask returns the task with the given identifier along with its phase and stage.
func (p *Plan) FindTask(id string) (*Phase, *Stage, *Task) {
	for phaseIndex := range p.Phases {
		phase := &p.Phases[phaseIndex]
		for stageIndex := range phase.Stages {
			stage := &phase.Stages[stageIndex]
			for taskIndex := range stage.Tasks {
				if stage.Tasks[taskIndex].ID == id {
					return phase, stage, &stage.Tasks[taskIndex]
				}
			}
		}
	}
	return nil, nil, nil
}
