// This file is part of PeerFoil.
// internal/cli/init.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: The peerfoil init command: shows the project records it would create under
// .peerfoil/ and creates them on --yes. It never overwrites an existing project.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/fsx"
	"github.com/gabrielmongefranco/peerfoil/internal/project"
	"github.com/gabrielmongefranco/peerfoil/internal/schema"
	"github.com/gabrielmongefranco/peerfoil/internal/version"
)

var projectIDPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

func runInit(ctx context.Context, env *Env, args []string) error {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	yes := flags.Bool("yes", false, "")
	pack := flags.String("pack", "software", "")
	profile := flags.String("profile", "personal", "")
	goal := flags.String("goal", "", "")
	if err := flags.Parse(args); err != nil {
		return errs.New(errs.Usage, "init could not read its options", "run peerfoil init --help")
	}
	if *profile != "personal" && *profile != "work" {
		return errs.New(errs.Usage, "the profile must be personal or work", "pass --profile personal or --profile work")
	}
	if *pack != "software" && *pack != "generic" && *pack != "documentation" {
		return errs.New(errs.Usage, "the pack must be software, generic, or documentation", "pass --pack software")
	}

	root, err := project.FindRoot(env.Dir)
	if err != nil {
		return err
	}
	peerfoilDir := filepath.Join(root, ".peerfoil")
	if fsx.Exists(peerfoilDir) {
		return errs.New(errs.Configuration,
			"this repository already has a .peerfoil directory",
			"run peerfoil status; init never overwrites an existing project")
	}

	projectID := deriveProjectID(filepath.Base(root))
	if projectID == "" {
		return errs.New(errs.Configuration,
			"the repository directory name cannot become a project identifier",
			"rename the directory, or start the project with the PeerFoil skills")
	}
	goalText := strings.TrimSpace(*goal)
	if goalText == "" {
		goalText = "To be described: run the PeerFoil skills decision interview to record the goal."
	}

	files := []string{
		filepath.Join(".peerfoil", "project.json"),
		filepath.Join(".peerfoil", "decisions.md"),
		filepath.Join(".peerfoil", "history.jsonl"),
	}
	fmt.Fprintf(env.Stdout, "peerfoil init would create these files inside %s:\n", root)
	for _, file := range files {
		fmt.Fprintf(env.Stdout, "  %s\n", file)
	}
	if !*yes {
		fmt.Fprintln(env.Stdout, "\nNothing was created. Run peerfoil init --yes to create them.")
		return nil
	}

	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	projectRecord := buildProjectRecord(projectID, filepath.Base(root), goalText, *pack, *profile, root, now)

	// The record must satisfy the same schema Core later validates against.
	schemas, err := schema.LoadEmbedded()
	if err != nil {
		return errs.Wrap(errs.Internal, "loading the record schemas", err)
	}
	var check any
	if err := json.Unmarshal(projectRecord, &check); err != nil {
		return errs.Wrap(errs.Internal, "encoding the project record", err)
	}
	problems, err := schemas.Validate(check, "project.schema.json")
	if err != nil {
		return errs.Wrap(errs.Internal, "validating the project record", err)
	}
	if len(problems) > 0 {
		return errs.New(errs.Internal,
			"the generated project record is invalid: "+strings.Join(problems, "; "), "")
	}

	if err := os.MkdirAll(peerfoilDir, 0o755); err != nil {
		return errs.Wrap(errs.Configuration, "creating the .peerfoil directory", err)
	}
	transition := buildFirstTransition(projectID, now)
	decisions := "<!-- PeerFoil project record: decisions. Edit through the PeerFoil workflow. -->\n" +
		"# Decisions\n\n| ID | Question | Status | Owner |\n|---|---|---|---|\n\nNo decisions yet.\n"
	writes := map[string][]byte{
		filepath.Join(peerfoilDir, "project.json"):  projectRecord,
		filepath.Join(peerfoilDir, "decisions.md"):  []byte(decisions),
		filepath.Join(peerfoilDir, "history.jsonl"): transition,
	}
	for path, data := range writes {
		if err := fsx.WriteFileAtomic(path, data, 0o644); err != nil {
			return errs.Wrap(errs.Configuration, "writing "+filepath.Base(path), err)
		}
	}
	fmt.Fprintf(env.Stdout, "\nCreated the project %s in the define state.\n", projectID)
	fmt.Fprintln(env.Stdout, "Next: resolve the decisions, architecture, and plan with the PeerFoil skills, then run peerfoil start.")
	return nil
}

// deriveProjectID lowercases the directory name and replaces every run of other
// characters with one hyphen, matching the identifier convention (D-0011).
func deriveProjectID(name string) string {
	lowered := strings.ToLower(name)
	var builder strings.Builder
	lastHyphen := true
	for _, r := range lowered {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastHyphen = false
		} else if !lastHyphen {
			builder.WriteByte('-')
			lastHyphen = true
		}
	}
	id := strings.Trim(builder.String(), "-")
	if len(id) > 64 {
		id = strings.Trim(id[:64], "-")
	}
	if !projectIDPattern.MatchString(id) {
		return ""
	}
	return id
}

func buildProjectRecord(id, name, goal, pack, profile, root, now string) []byte {
	rules := map[string]any{"mode": "none"}
	if fsx.Exists(filepath.Join(root, "AGENTS.md")) {
		rules = map[string]any{"mode": "inherit", "path": "AGENTS.md"}
	}
	seat := func(tool, effort string) map[string]any {
		return map[string]any{"tool": tool, "model": "default", "effort": effort}
	}
	record := map[string]any{
		"record_type":    "project",
		"schema_version": 1,
		"project_id":     id,
		"name":           name,
		"goal":           goal,
		"pack":           map[string]any{"id": pack, "version": "0.1.0"},
		"profile":        profile,
		"rules_source":   rules,
		"assurance":      "guided",
		"release":        version.Release,
		"created_at":     now,
		"updated_at":     now,
		"workflow": map[string]any{
			"state": "define", "phase": nil, "stage": nil, "task": nil, "paused_for": nil,
		},
		"revisions": map[string]any{"architecture": 0, "quality": 0, "plan": 0},
		"settings": map[string]any{
			"roles": map[string]any{
				"evaluator":       seat("claude-code", "medium"),
				"architect":       seat("claude-code", "high"),
				"planner":         seat("claude-code", "medium"),
				"change_steward":  seat("claude-code", "medium"),
				"producer":        seat("codex-cli", "medium"),
				"repair_producer": seat("codex-cli", "medium"),
			},
			"phase_reviewers": []any{seat("claude-code", "medium"), seat("codex-cli", "medium")},
			"review": map[string]any{
				"default_passes": 6, "max_passes": 8,
				"repair_selection_passes": 3, "repair_selection_max_passes": 4,
				"repair_cycles": 1,
			},
		},
	}
	data, _ := json.MarshalIndent(record, "", "  ")
	return append(data, '\n')
}

func buildFirstTransition(projectID, now string) []byte {
	record := map[string]any{
		"record_type":    "transition",
		"schema_version": 1,
		"transition_id":  "tr-0001",
		"project_id":     projectID,
		"at":             now,
		"from_state":     nil,
		"to_state":       "define",
		"actor": map[string]any{
			"role": "coordinator", "tool": "other", "model": "peerfoil-core",
			"effort": nil, "lineage_root": "unknown", "session": nil,
		},
		"plan_revision":   0,
		"source_revision": nil,
		"summary":         "Project started by peerfoil init; decision interview pending.",
	}
	data, _ := json.Marshal(record)
	return append(data, '\n')
}
