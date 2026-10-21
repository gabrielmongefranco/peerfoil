// This file is part of PeerFoil.
// internal/project/load.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Loads and validates the accepted .peerfoil records: schema validation first,
// then the semantic checks a schema cannot express. Invalid state fails closed.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

// Package project reads the accepted project state the Skills release creates and the
// controller enforces. It rejects records it cannot prove consistent rather than
// guessing, because accepted state is the source of truth for every later step.
package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/schema"
)

// State is the loaded, validated project state.
type State struct {
	Root    string // repository root
	Dir     string // the .peerfoil directory
	Project *Project
	Plan    *Plan // nil when no plan is accepted yet
	History []Transition
}

// Load reads .peerfoil under root, validates every record against the schemas and the
// semantic rules, and returns the state or one validation error listing every problem.
func Load(root string, schemas *schema.Set) (*State, error) {
	dir := filepath.Join(root, ".peerfoil")
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return nil, errs.New(errs.Configuration,
			"this repository has no .peerfoil directory",
			"run peerfoil init, or start the project with the PeerFoil skills")
	}

	var problems []string
	state := &State{Root: root, Dir: dir}

	projectRaw, projectAny, err := readJSON(filepath.Join(dir, "project.json"))
	if err != nil {
		return nil, err
	}
	if issues, err := schemas.Validate(projectAny, "project.schema.json"); err != nil {
		return nil, errs.Wrap(errs.Internal, "validating project.json", err)
	} else if len(issues) > 0 {
		problems = append(problems, prefix("project.json", issues)...)
	} else {
		state.Project = &Project{}
		if err := json.Unmarshal(projectRaw, state.Project); err != nil {
			return nil, errs.Wrap(errs.Validation, "decoding project.json", err)
		}
	}
	if len(problems) > 0 {
		return nil, validationError(problems)
	}

	planPath := filepath.Join(dir, "plan.json")
	if state.Project.Revisions.Plan > 0 || fileExists(planPath) {
		planRaw, planAny, err := readJSON(planPath)
		if err != nil {
			return nil, err
		}
		if issues, err := schemas.Validate(planAny, "plan.schema.json"); err != nil {
			return nil, errs.Wrap(errs.Internal, "validating plan.json", err)
		} else if len(issues) > 0 {
			problems = append(problems, prefix("plan.json", issues)...)
		} else {
			state.Plan = &Plan{}
			if err := json.Unmarshal(planRaw, state.Plan); err != nil {
				return nil, errs.Wrap(errs.Validation, "decoding plan.json", err)
			}
		}
	}

	historyPath := filepath.Join(dir, "history.jsonl")
	if fileExists(historyPath) {
		raw, err := os.ReadFile(historyPath)
		if err != nil {
			return nil, errs.Wrap(errs.Configuration, "reading history.jsonl", err)
		}
		for number, line := range strings.Split(string(raw), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			var lineAny any
			if err := json.Unmarshal([]byte(trimmed), &lineAny); err != nil {
				problems = append(problems, fmt.Sprintf("history.jsonl line %d: invalid JSON", number+1))
				continue
			}
			issues, err := schemas.Validate(lineAny, "transition.schema.json")
			if err != nil {
				return nil, errs.Wrap(errs.Internal, "validating history.jsonl", err)
			}
			if len(issues) > 0 {
				problems = append(problems, prefix(fmt.Sprintf("history.jsonl line %d", number+1), issues)...)
				continue
			}
			var transition Transition
			if err := json.Unmarshal([]byte(trimmed), &transition); err != nil {
				problems = append(problems, fmt.Sprintf("history.jsonl line %d: cannot decode", number+1))
				continue
			}
			state.History = append(state.History, transition)
		}
	}

	problems = append(problems, state.semanticProblems()...)
	if len(problems) > 0 {
		return nil, validationError(problems)
	}
	return state, nil
}

// semanticProblems checks what the schemas cannot: identity, revision agreement,
// duplicate identifiers, dependency existence and cycles, stale tasks, change
// relationships, and history consistency.
func (s *State) semanticProblems() []string {
	var problems []string
	add := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf(format, args...))
	}

	if s.Project.Revisions.Plan > 0 && s.Plan == nil {
		add("project.json names plan revision %d but plan.json is missing", s.Project.Revisions.Plan)
	}
	// An inherited rules path is later read and quoted to models, so it must stay a
	// plain repository-relative path; anything else could expose unrelated files.
	if s.Project.RulesSource.Mode == "inherit" {
		path := s.Project.RulesSource.Path
		if path == "" || strings.Contains(path, "\\") || strings.HasPrefix(path, "/") ||
			strings.Contains(path, ":") || hasDotDot(path) {
			add("rules_source.path %q must be a repository-relative path with no traversal", path)
		}
	}
	for _, name := range []string{"architecture.md", "quality.md"} {
		revision := s.Project.Revisions.Architecture
		if name == "quality.md" {
			revision = s.Project.Revisions.Quality
		}
		if revision > 0 && !fileExists(filepath.Join(s.Dir, name)) {
			add("project.json names an accepted %s revision but the file is missing", name)
		}
	}

	if s.Plan != nil {
		plan := s.Plan
		if plan.ProjectID != s.Project.ProjectID {
			add("plan.json project_id %q differs from project.json %q", plan.ProjectID, s.Project.ProjectID)
		}
		if plan.PlanRevision != s.Project.Revisions.Plan {
			add("plan.json revision %d does not match the accepted plan revision %d; the plan is stale",
				plan.PlanRevision, s.Project.Revisions.Plan)
		}
		if plan.ArchitectureRevision != s.Project.Revisions.Architecture {
			add("plan.json architecture revision %d does not match the accepted architecture revision %d",
				plan.ArchitectureRevision, s.Project.Revisions.Architecture)
		}
		if plan.QualityRevision != s.Project.Revisions.Quality {
			add("plan.json quality revision %d does not match the accepted Quality Contract revision %d",
				plan.QualityRevision, s.Project.Revisions.Quality)
		}
		problems = append(problems, planProblems(plan)...)
	}

	for index, transition := range s.History {
		if transition.ProjectID != s.Project.ProjectID {
			add("history.jsonl line %d: project_id %q differs from %q", index+1, transition.ProjectID, s.Project.ProjectID)
		}
		if index > 0 && transition.At < s.History[index-1].At {
			add("history.jsonl line %d: timestamp moves backwards", index+1)
		}
		if index > 0 {
			previous := s.History[index-1].ToState
			if transition.FromState == nil || *transition.FromState != previous {
				add("history.jsonl line %d: from_state does not continue the previous transition's %q", index+1, previous)
			}
		}
	}
	if len(s.History) > 0 {
		last := s.History[len(s.History)-1]
		if last.ToState != s.Project.Workflow.State {
			// The one known benign cause is an interruption between the history
			// append and the project write; name the exact repair so the project
			// is recoverable without guessing.
			if len(s.History) > 1 && s.History[len(s.History)-2].ToState == s.Project.Workflow.State {
				add("project.json workflow state %q is one transition behind history.jsonl, whose last transition %s reached %q; an interrupted run stopped between the two writes. To repair, set workflow.state in project.json to %q",
					s.Project.Workflow.State, last.TransitionID, last.ToState, last.ToState)
			} else {
				add("project.json workflow state %q does not match the last recorded transition %q",
					s.Project.Workflow.State, last.ToState)
			}
		}
	} else if s.Project.Workflow.State != "define" {
		add("project.json workflow state %q has no supporting history", s.Project.Workflow.State)
	}

	return problems
}

func planProblems(plan *Plan) []string {
	var problems []string
	add := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf("plan.json: "+format, args...))
	}

	retained := map[string]bool{}
	for _, change := range plan.Changes {
		for _, taskID := range change.RetainedTasks {
			retained[taskID] = true
		}
	}

	seen := map[string]bool{}
	taskIDs := map[string]bool{}
	stageIDs := map[string]bool{}
	activePhases := 0
	for _, phase := range plan.Phases {
		if seen[phase.ID] {
			add("duplicate identifier %s", phase.ID)
		}
		seen[phase.ID] = true
		if phase.Status == "active" {
			activePhases++
		}
		for _, stage := range phase.Stages {
			if seen[stage.ID] {
				add("duplicate identifier %s", stage.ID)
			}
			seen[stage.ID] = true
			stageIDs[stage.ID] = true
			for _, task := range stage.Tasks {
				if seen[task.ID] {
					add("duplicate identifier %s", task.ID)
				}
				seen[task.ID] = true
				taskIDs[task.ID] = true
				if task.PlanRevision > plan.PlanRevision {
					add("%s names plan revision %d, later than the plan's own revision %d",
						task.ID, task.PlanRevision, plan.PlanRevision)
				}
				// Runnable work from an older revision is valid only when change
				// intake retained it; otherwise its scope was never re-tied to the
				// current plan and it must not be selected.
				if (task.Status == "planned" || task.Status == "ready") &&
					task.PlanRevision < plan.PlanRevision && !retained[task.ID] {
					add("%s is runnable but tied to the older plan revision %d without being retained by a recorded change",
						task.ID, task.PlanRevision)
				}
				if task.ArchitectureRevision > plan.ArchitectureRevision {
					add("%s names architecture revision %d, later than the plan's %d",
						task.ID, task.ArchitectureRevision, plan.ArchitectureRevision)
				}
			}
		}
	}
	requirementIDs := map[string]bool{}
	for _, requirement := range plan.Requirements {
		if requirementIDs[requirement.ID] {
			add("duplicate requirement %s", requirement.ID)
		}
		requirementIDs[requirement.ID] = true
		for _, taskID := range requirement.Tasks {
			if !taskIDs[taskID] {
				add("%s names missing task %s", requirement.ID, taskID)
			}
		}
	}

	if activePhases > 1 {
		add("%d phases are active at once; task selection and review operate on exactly one", activePhases)
	}

	dependencies := map[string][]string{}
	stageDependencies := map[string][]string{}
	for _, phase := range plan.Phases {
		for _, stage := range phase.Stages {
			for _, dependency := range stage.DependsOn {
				if !stageIDs[dependency] {
					add("%s depends on missing stage %s", stage.ID, dependency)
				}
			}
			stageDependencies[stage.ID] = stage.DependsOn
			for _, task := range stage.Tasks {
				for _, dependency := range task.DependsOn {
					if !taskIDs[dependency] {
						add("%s depends on missing task %s", task.ID, dependency)
					}
				}
				dependencies[task.ID] = task.DependsOn
			}
		}
	}
	if cycle := findCycle(dependencies); cycle != "" {
		add("task dependencies form a cycle through %s", cycle)
	}
	if cycle := findCycle(stageDependencies); cycle != "" {
		add("stage dependencies form a cycle through %s", cycle)
	}

	changeIDs := map[string]bool{}
	for _, change := range plan.Changes {
		if changeIDs[change.ID] {
			add("duplicate change %s", change.ID)
		}
		changeIDs[change.ID] = true
		if change.PriorRevision >= plan.PlanRevision {
			add("%s prior revision %d must precede plan revision %d", change.ID, change.PriorRevision, plan.PlanRevision)
		}
		retained := map[string]bool{}
		for _, taskID := range change.RetainedTasks {
			retained[taskID] = true
		}
		for _, taskID := range change.AffectedTasks {
			if retained[taskID] {
				add("%s lists %s as both affected and retained", change.ID, taskID)
			}
		}
	}
	return problems
}

// hasDotDot reports whether any slash-separated component of a path is "..".
func hasDotDot(path string) bool {
	for _, component := range strings.Split(path, "/") {
		if component == ".." {
			return true
		}
	}
	return false
}

// findCycle returns one task on a dependency cycle, or an empty string.
func findCycle(dependencies map[string][]string) string {
	const (
		visiting = 1
		done     = 2
	)
	states := map[string]int{}
	var visit func(node string) bool
	visit = func(node string) bool {
		switch states[node] {
		case visiting:
			return true
		case done:
			return false
		}
		states[node] = visiting
		for _, next := range dependencies[node] {
			if visit(next) {
				return true
			}
		}
		states[node] = done
		return false
	}
	nodes := make([]string, 0, len(dependencies))
	for node := range dependencies {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		if visit(node) {
			return node
		}
	}
	return ""
}

func readJSON(path string) ([]byte, any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, errs.New(errs.Configuration,
			fmt.Sprintf("cannot read %s", filepath.Base(path)),
			"run peerfoil init, or restore the file from Git")
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, nil, errs.Wrap(errs.Validation,
			fmt.Sprintf("%s is not valid JSON", filepath.Base(path)), err)
	}
	return raw, value, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func prefix(label string, issues []string) []string {
	out := make([]string, len(issues))
	for index, issue := range issues {
		out[index] = label + ": " + issue
	}
	return out
}

func validationError(problems []string) error {
	sort.Strings(problems)
	return errs.New(errs.Validation,
		fmt.Sprintf("the project records are not consistent: %s", strings.Join(problems, "; ")),
		"fix the named records, or restore them from Git, then try again")
}

func phasesByOrder(phases []Phase) []*Phase {
	out := make([]*Phase, len(phases))
	for index := range phases {
		out[index] = &phases[index]
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Order < out[b].Order })
	return out
}

func stagesByOrder(stages []Stage) []*Stage {
	out := make([]*Stage, len(stages))
	for index := range stages {
		out[index] = &stages[index]
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Order < out[b].Order })
	return out
}
