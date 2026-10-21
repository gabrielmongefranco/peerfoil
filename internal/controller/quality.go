// This file is part of PeerFoil.
// internal/controller/quality.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Resolves executable procedures from the accepted Quality Contract without
// evaluating shell syntax or substituting pack suggestions for accepted commands.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package controller

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/evidence"
	"github.com/gabrielmongefranco/peerfoil/internal/project"
	"github.com/gabrielmongefranco/peerfoil/internal/redact"
)

var procedurePattern = regexp.MustCompile("^(`\\[[^`]+\\]`(?: then `\\[[^`]+\\]`)*) in (the (?:repository|project) root|`[^`]+`)(?:; .*)?$")

// acceptedCommands resolves exactly one declared evidence row, including every
// command in an ordered procedure. Missing, ambiguous, or unsafe rows fail closed.
func acceptedCommands(contract []byte, requirement project.EvidenceRequirement, root string) ([]evidence.Command, error) {
	invalid := func() ([]evidence.Command, error) {
		return nil, errs.New(errs.Evidence, fmt.Sprintf("the accepted Quality Contract has no unique, valid executable procedure for %q", requirement.Name),
			"declare JSON argument lists and a repository-relative working directory in the Quality Contract, then review the change")
	}
	var procedure, kind, rowLevel string
	matches := 0
	level := ""
	for _, line := range strings.Split(strings.ReplaceAll(string(contract), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "## ") {
			level = ""
			switch line {
			case "## Required evidence":
				level = "required"
			case "## Recommended evidence":
				level = "recommended"
			case "## Not applicable":
				level = "not-applicable"
			}
			continue
		}
		if level == "" || !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(line, "|")
		if len(cells) < 3 || strings.TrimSpace(cells[1]) != requirement.Name {
			continue
		}
		matches++
		if len(cells) != 5 || strings.TrimSpace(cells[4]) != "" {
			return invalid()
		}
		kind, procedure, rowLevel = strings.TrimSpace(cells[2]), strings.TrimSpace(cells[3]), level
	}
	if matches != 1 || kind != "executable" || rowLevel == "not-applicable" || (rowLevel == "required" && requirement.Level != "required") {
		return invalid()
	}
	match := procedurePattern.FindStringSubmatch(procedure)
	if match == nil {
		return invalid()
	}
	dir := ""
	if strings.HasPrefix(match[2], "`") {
		dir = strings.Trim(match[2], "`")
		if dir == "" || strings.ContainsAny(dir, "\\:") || strings.HasPrefix(dir, "/") {
			return invalid()
		}
		for _, component := range strings.Split(dir, "/") {
			if component == ".." || component == "" {
				return invalid()
			}
		}
	}
	var commands []evidence.Command
	for _, encoded := range strings.Split(match[1], " then ") {
		var argv []string
		if err := json.Unmarshal([]byte(strings.Trim(encoded, "`")), &argv); err != nil || len(argv) == 0 || strings.TrimSpace(argv[0]) == "" {
			return invalid()
		}
		command := evidence.Command{Name: requirement.Name, Argv: argv, Dir: dir, TimeoutSeconds: 600}
		if err := evidence.ValidateCommand(command, root); err != nil {
			return invalid()
		}
		if _, scrubbed := redact.ScrubString(strings.Join(argv, " ") + " " + dir); scrubbed {
			return invalid()
		}
		commands = append(commands, command)
	}
	return commands, nil
}
