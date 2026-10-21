// This file is part of PeerFoil.
// internal/pack/pack.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Loads and validates the built-in project packs the binary ships, and maps a
// task's declared evidence to the pack's executable procedures.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

// Package pack reads the declarative project-pack manifests. A pack supplies evidence
// definitions, command hints, and review areas; it never executes controller code.
package pack

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"

	peerfoil "github.com/gabrielmongefranco/peerfoil"
	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/schema"
)

// Tool is one program a pack needs, probed with its command.
type Tool struct {
	Name     string   `json:"name"`
	Command  []string `json:"command"`
	Required bool     `json:"required"`
	Purpose  string   `json:"purpose"`
}

// Hint ties a repository marker file to the command that produces one evidence kind.
type Hint struct {
	Marker   string   `json:"marker"`
	Evidence string   `json:"evidence"`
	Command  []string `json:"command"`
	Purpose  string   `json:"purpose"`
}

// Evidence declares one evidence kind the pack knows about.
type Evidence struct {
	Name         string `json:"name"`
	Kind         string `json:"kind"`
	DefaultLevel string `json:"default_level"`
	Description  string `json:"description"`
}

// Pack is one validated project-pack manifest.
type Pack struct {
	ID               string     `json:"id"`
	Version          string     `json:"version"`
	Evidence         []Evidence `json:"evidence"`
	Tools            []Tool     `json:"tools"`
	ProjectToolHints []Hint     `json:"project_tool_hints"`
	EvidenceHints    []Hint     `json:"evidence_hints"`
}

// Load returns the named built-in pack from the embedded copy, validated against the
// pack schema so a damaged build fails loudly instead of running with partial policy.
func Load(id string, schemas *schema.Set) (*Pack, error) {
	path := "plugins/peerfoil/packs/" + id + "/pack.json"
	raw, err := fs.ReadFile(peerfoil.Assets, path)
	if err != nil {
		return nil, errs.New(errs.Configuration,
			fmt.Sprintf("the project pack %q is not a built-in pack", id),
			"use software, generic, or documentation")
	}
	var instance any
	if err := json.Unmarshal(raw, &instance); err != nil {
		return nil, errs.Wrap(errs.Internal, "parsing the embedded pack manifest", err)
	}
	problems, err := schemas.Validate(instance, "pack.schema.json")
	if err != nil {
		return nil, errs.Wrap(errs.Internal, "validating the pack manifest", err)
	}
	if len(problems) > 0 {
		return nil, errs.New(errs.Internal,
			fmt.Sprintf("the embedded pack %q fails its schema: %v", id, problems), "")
	}
	var loaded Pack
	if err := json.Unmarshal(raw, &loaded); err != nil {
		return nil, errs.Wrap(errs.Internal, "decoding the pack manifest", err)
	}
	return &loaded, nil
}

// CommandFor returns the pack's executable procedure for one evidence name, selected
// by the first hint whose marker file exists at the repository root. A nil return
// means the pack has no matching procedure and the controller must stop for the user
// rather than invent a command.
func (p *Pack) CommandFor(evidenceName, repoRoot string) []string {
	for _, hint := range p.EvidenceHints {
		if hint.Evidence != evidenceName {
			continue
		}
		matches, err := filepath.Glob(filepath.Join(repoRoot, hint.Marker))
		if err == nil && len(matches) > 0 {
			command := make([]string, len(hint.Command))
			copy(command, hint.Command)
			return command
		}
	}
	return nil
}
