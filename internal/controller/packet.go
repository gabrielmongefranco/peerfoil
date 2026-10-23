// This file is part of PeerFoil.
// internal/controller/packet.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Builds the compact producer transfer packet and validates the structured
// handoff that comes back. Model output is untrusted until it passes these checks.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package controller

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/project"
)

// Handoff is the one JSON object a producer must return.
type Handoff struct {
	Task                 string   `json:"task"`
	PlanRevision         int      `json:"plan_revision"`
	ArchitectureRevision int      `json:"architecture_revision"`
	BaseRevision         string   `json:"base_revision"`
	Result               string   `json:"result"`
	ChangedPaths         []string `json:"changed_paths"`
	Summary              string   `json:"summary"`
	ChecksClaimed        []string `json:"checks_claimed"`
	DiscoveredWork       []string `json:"discovered_work"`
	Blocker              *string  `json:"blocker"`
}

const maxHandoffBytes = 32 * 1024

var fencedJSON = regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```")

// BuildProducerPacket assembles the compact, self-contained request for one task. It
// carries only what the task needs: rules pointers, decisions and architecture
// excerpts, the exact scope, and the handoff contract. It never includes chat history
// or credentials.
func (e *Engine) BuildProducerPacket(selection *project.Selection, changeSetID, baseCommit string) string {
	task := selection.Task
	var b strings.Builder
	b.WriteString("You are the PeerFoil producer for exactly one bounded task. ")
	b.WriteString("Work only inside the current directory, which is a dedicated Git worktree for this task.\n\n")

	// The rules path is loader-validated, and this guard keeps the packet safe even
	// against a record edited after loading: only a plain repository-relative path
	// may be quoted as something to read.
	rulesPath := e.State.Project.RulesSource.Path
	if e.State.Project.RulesSource.Mode == "inherit" && rulesPath != "" &&
		!strings.ContainsAny(rulesPath, "\\:") && !strings.HasPrefix(rulesPath, "/") &&
		!strings.Contains("/"+rulesPath+"/", "/../") {
		fmt.Fprintf(&b, "Repository rules: read %s at the repository root first and follow it.\n", rulesPath)
	}
	fmt.Fprintf(&b, "Project: %s · goal: %s\n", e.State.Project.ProjectID, e.State.Project.Goal)
	fmt.Fprintf(&b, "Identifiers: phase %s, stage %s, task %s, change set %s\n",
		selection.Phase.ID, selection.Stage.ID, task.ID, changeSetID)
	fmt.Fprintf(&b, "Revisions: plan %d, architecture %d, quality %d · base commit %s\n",
		e.State.Project.Revisions.Plan, e.State.Project.Revisions.Architecture,
		e.State.Project.Revisions.Quality, baseCommit)
	b.WriteString("The appended accepted input snapshot supplies the current records; checkout copies may be older.\n\n")

	fmt.Fprintf(&b, "Task %s — %s\nScope: %s\nExpected output: %s\n", task.ID, task.Title, task.Scope, task.Output)
	fmt.Fprintf(&b, "Allowed paths (edit nothing else): %s\n", strings.Join(task.AllowedPaths, ", "))
	if len(task.Inputs) > 0 {
		fmt.Fprintf(&b, "Inputs: %s\n", strings.Join(task.Inputs, ", "))
	}
	if len(task.Acceptance) > 0 {
		b.WriteString("Acceptance criteria:\n")
		for _, criterion := range task.Acceptance {
			fmt.Fprintf(&b, "- %s\n", criterion)
		}
	}
	if len(task.RequiredEvidence) > 0 {
		b.WriteString("Required evidence the controller runs afterwards: ")
		var names []string
		for _, requirement := range task.RequiredEvidence {
			names = append(names, requirement.Name+" ("+requirement.Level+")")
		}
		b.WriteString(strings.Join(names, ", ") + "\n")
	}

	b.WriteString(`
Restrictions: edit only the allowed paths. Do not edit project records under .peerfoil/,
Git history, AGENTS.md, secrets, or provider settings. Do not commit, push, publish,
deploy, delete unrelated work, or widen permissions. Any check you run yourself is a
claim; the controller reruns the required checks. Report a blocked action instead of
working around it.

End your answer with exactly one fenced JSON object in this shape:

` + "```json\n" + `{
  "task": "` + task.ID + `",
  "plan_revision": ` + fmt.Sprint(e.State.Project.Revisions.Plan) + `,
  "architecture_revision": ` + fmt.Sprint(e.State.Project.Revisions.Architecture) + `,
  "base_revision": "` + baseCommit + `",
  "result": "produced",
  "changed_paths": ["relative/path"],
  "summary": "One bounded change.",
  "checks_claimed": [],
  "discovered_work": [],
  "blocker": null
}
` + "```\n" + `
"result" is "produced", "partial", or "blocked". Every key is required.
`)
	return b.String()
}

// ParseHandoff extracts and validates the producer's fenced JSON object. The text is
// untrusted: unknown keys, wrong identifiers, oversized content, and unsafe paths are
// all rejected, and a malformed handoff can never report success.
func ParseHandoff(text string, task *project.Task, planRevision, architectureRevision int, baseCommit string) (*Handoff, error) {
	matches := fencedJSON.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil, errs.New(errs.Provider, "the producer returned no fenced JSON handoff",
			"the run is recorded as blocked; run peerfoil resume to retry once")
	}
	raw := matches[len(matches)-1][1]
	if len(raw) > maxHandoffBytes {
		return nil, errs.New(errs.Provider, "the producer handoff is larger than 32 KiB", "")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var handoff Handoff
	if err := decoder.Decode(&handoff); err != nil {
		return nil, errs.Wrap(errs.Provider, "the producer handoff is not the required JSON shape", err)
	}
	var problems []string
	if handoff.Task != task.ID {
		problems = append(problems, fmt.Sprintf("task %q is not %s", handoff.Task, task.ID))
	}
	if handoff.PlanRevision != planRevision {
		problems = append(problems, fmt.Sprintf("plan revision %d is not %d", handoff.PlanRevision, planRevision))
	}
	if handoff.ArchitectureRevision != architectureRevision {
		problems = append(problems, fmt.Sprintf("architecture revision %d is not %d", handoff.ArchitectureRevision, architectureRevision))
	}
	if handoff.BaseRevision != baseCommit {
		problems = append(problems, "the base revision is not the launch commit")
	}
	switch handoff.Result {
	case "produced", "partial", "blocked":
	default:
		problems = append(problems, fmt.Sprintf("result %q is not produced, partial, or blocked", handoff.Result))
	}
	for _, value := range append(append([]string{handoff.Summary}, handoff.ChecksClaimed...), handoff.DiscoveredWork...) {
		if len(value) > 2000 {
			problems = append(problems, "a text field is longer than 2,000 characters")
			break
		}
	}
	if handoff.Blocker != nil && len(*handoff.Blocker) > 2000 {
		problems = append(problems, "blocker is longer than 2,000 characters")
	}
	seen := map[string]bool{}
	for _, path := range handoff.ChangedPaths {
		if len(path) > 2000 {
			problems = append(problems, "a path is longer than 2,000 characters")
			continue
		}
		if seen[path] {
			problems = append(problems, "duplicate changed path")
		}
		seen[path] = true
	}
	if len(problems) > 0 {
		return nil, errs.New(errs.Provider,
			"the producer handoff failed validation: "+strings.Join(problems, "; "), "")
	}
	return &handoff, nil
}

// nextEvidenceDirID allocates the next cs- or ev- identifier under .peerfoil/evidence.
func (e *Engine) nextEvidenceDirID(prefix string) (string, error) {
	dir := filepath.Join(e.State.Dir, "evidence")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", errs.Wrap(errs.Configuration, "creating the evidence directory", err)
	}
	return nextID(dir, prefix, 4)
}
