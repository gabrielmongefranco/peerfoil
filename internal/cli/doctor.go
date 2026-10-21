// This file is part of PeerFoil.
// internal/cli/doctor.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: The peerfoil doctor command: checks the environment, repository, records,
// model tools, pack tools, host sandbox, and local state, and changes nothing.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/pack"
	"github.com/gabrielmongefranco/peerfoil/internal/project"
	"github.com/gabrielmongefranco/peerfoil/internal/provider"
	"github.com/gabrielmongefranco/peerfoil/internal/schema"
	"github.com/gabrielmongefranco/peerfoil/internal/store"
	"github.com/gabrielmongefranco/peerfoil/internal/workspace"
)

type doctorCheck struct {
	Name   string
	Status string // pass, fail, not-verified, skipped
	Detail string
	Next   string
}

func runDoctor(ctx context.Context, env *Env, args []string) error {
	if len(args) != 0 {
		return errs.New(errs.Usage, "doctor takes no arguments", "run peerfoil doctor")
	}
	var checks []doctorCheck
	add := func(check doctorCheck) { checks = append(checks, check) }

	add(doctorCheck{Name: "operating-system", Status: "pass",
		Detail: runtime.GOOS + "/" + runtime.GOARCH})

	// Git and the repository.
	gitVersion := probeCommand(ctx, "git", "--version")
	if gitVersion == "" {
		add(doctorCheck{Name: "git", Status: "fail", Detail: "git did not answer",
			Next: "install Git and make sure it is on the PATH"})
	} else {
		add(doctorCheck{Name: "git", Status: "pass", Detail: gitVersion})
	}

	root, err := workspace.Discover(ctx, env.Dir)
	if err != nil {
		add(doctorCheck{Name: "repository", Status: "fail",
			Detail: "no Git repository found here", Next: errs.NextStep(err)})
		printDoctor(env, checks)
		return errs.New(errs.Configuration, "this directory is not ready", "fix the failed checks above")
	}
	repo := workspace.New(root)
	if _, err := repo.HeadCommit(ctx); err != nil {
		add(doctorCheck{Name: "repository", Status: "fail",
			Detail: "the repository has no commit yet", Next: "make an initial commit"})
	} else if clean, dirty, err := repo.IsClean(ctx); err == nil && !clean {
		add(doctorCheck{Name: "repository", Status: "pass",
			Detail: fmt.Sprintf("found, with %d uncommitted change(s)", len(dirty))})
	} else {
		add(doctorCheck{Name: "repository", Status: "pass", Detail: "found, clean"})
	}

	// Project records and schema versions.
	schemas, err := schema.LoadEmbedded()
	if err != nil {
		return errs.Wrap(errs.Internal, "loading the record schemas", err)
	}
	var state *project.State
	if _, statErr := os.Stat(filepath.Join(root, ".peerfoil")); statErr != nil {
		add(doctorCheck{Name: "project-records", Status: "skipped",
			Detail: "no .peerfoil directory", Next: "run peerfoil init, or start with the PeerFoil skills"})
	} else if state, err = project.Load(root, schemas); err != nil {
		detail := err.Error()
		if len(detail) > 240 {
			detail = detail[:240] + "…"
		}
		add(doctorCheck{Name: "project-records", Status: "fail", Detail: detail,
			Next: "fix the named records or restore them from Git"})
	} else {
		add(doctorCheck{Name: "project-records", Status: "pass",
			Detail: fmt.Sprintf("valid; state %s, plan revision %d",
				state.Project.Workflow.State, state.Project.Revisions.Plan)})
	}

	// Model tools: installation, login, and the seats' models.
	claudeAdapter := firstFound(ctx, "claude-code")
	codexAdapter := firstFound(ctx, "codex-cli")
	for _, probe := range []struct {
		name    string
		adapter provider.Adapter
		install string
	}{
		{"claude-code", claudeAdapter, "install Claude Code"},
		{"codex-cli", codexAdapter, "install the Codex CLI or its IDE extension, then run codex login"},
	} {
		if probe.adapter == nil {
			add(doctorCheck{Name: probe.name, Status: "fail", Detail: "not found", Next: probe.install})
			continue
		}
		result := probe.adapter.Probe(ctx)
		status := "pass"
		detail := result.Version
		next := ""
		if !result.LoggedIn {
			status = "fail"
			detail = result.Version + "; not signed in"
			next = "sign in with the tool's own login command"
		}
		if result.Detail != "" {
			detail = strings.TrimSpace(detail + "; " + result.Detail)
		}
		add(doctorCheck{Name: probe.name, Status: status, Detail: detail, Next: next})
	}
	if state != nil {
		var unknownSeats []string
		for role, seat := range state.Project.Settings.Roles {
			if seat.Model == "" || seat.Model == "default" {
				unknownSeats = append(unknownSeats, role)
			}
		}
		if len(unknownSeats) > 0 {
			add(doctorCheck{Name: "seat-models", Status: "not-verified",
				Detail: "these seats use the tool's default model, so their lineage is unknown: " + strings.Join(unknownSeats, ", "),
				Next:   "record each seat's model so reviewer independence can be proven"})
		} else {
			add(doctorCheck{Name: "seat-models", Status: "pass", Detail: "every seat names its model"})
		}
	}

	// Pack tools.
	if state != nil {
		if loadedPack, err := pack.Load(state.Project.Pack.ID, schemas); err == nil {
			for _, tool := range loadedPack.Tools {
				answer := probeCommand(ctx, tool.Command[0], tool.Command[1:]...)
				if answer != "" {
					add(doctorCheck{Name: "pack-tool-" + tool.Name, Status: "pass", Detail: answer})
				} else if tool.Required {
					add(doctorCheck{Name: "pack-tool-" + tool.Name, Status: "fail",
						Detail: tool.Purpose, Next: "install " + tool.Name})
				} else {
					add(doctorCheck{Name: "pack-tool-" + tool.Name, Status: "not-verified", Detail: tool.Purpose})
				}
			}
			for _, hint := range loadedPack.ProjectToolHints {
				matches, _ := filepath.Glob(filepath.Join(root, hint.Marker))
				if len(matches) == 0 {
					continue
				}
				answer := probeCommand(ctx, hint.Command[0], hint.Command[1:]...)
				if answer != "" {
					add(doctorCheck{Name: "project-tool-" + hint.Command[0], Status: "pass", Detail: answer})
				} else {
					add(doctorCheck{Name: "project-tool-" + hint.Command[0], Status: "fail",
						Detail: hint.Purpose, Next: "install the tool this project's checks need"})
				}
			}
		}
	}

	// Host sandbox (D-0029): detect and report, never block.
	add(sandboxCheck(root))

	// Local state directory.
	projectID := "uninitialized"
	if state != nil {
		projectID = state.Project.ProjectID
	}
	stateDir, err := store.Dir(projectID, root)
	if err != nil {
		add(doctorCheck{Name: "local-state", Status: "fail", Detail: err.Error(),
			Next: "make sure the user cache directory is available"})
	} else if err := os.MkdirAll(stateDir, 0o700); err != nil {
		add(doctorCheck{Name: "local-state", Status: "fail", Detail: "the state directory cannot be created",
			Next: "check permissions on the user cache directory"})
	} else if probe := filepath.Join(stateDir, ".write-check"); os.WriteFile(probe, []byte("ok"), 0o600) != nil {
		add(doctorCheck{Name: "local-state", Status: "fail", Detail: "the state directory is not writable",
			Next: "check permissions on the user cache directory"})
	} else {
		os.Remove(probe)
		add(doctorCheck{Name: "local-state", Status: "pass", Detail: "writable"})
	}

	printDoctor(env, checks)
	for _, check := range checks {
		if check.Status == "fail" {
			return errs.New(errs.Configuration, "this computer or repository is not ready",
				"fix the failed checks above, then run peerfoil doctor again")
		}
	}
	fmt.Fprintln(env.Stdout, "Everything peerfoil needs is in place.")
	return nil
}

// sandboxCheck reads the merged Claude Code settings for sandbox.enabled. It reports;
// a missing sandbox never fails doctor (D-0029).
func sandboxCheck(root string) doctorCheck {
	if runtime.GOOS == "windows" {
		return doctorCheck{Name: "host-sandbox", Status: "not-verified",
			Detail: "the Claude Code sandbox needs WSL2; native Windows runs without it"}
	}
	paths := []string{filepath.Join(root, ".claude", "settings.json")}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".claude", "settings.json"))
	}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var settings struct {
			Sandbox struct {
				Enabled bool `json:"enabled"`
			} `json:"sandbox"`
		}
		if json.Unmarshal(raw, &settings) == nil && settings.Sandbox.Enabled {
			return doctorCheck{Name: "host-sandbox", Status: "pass",
				Detail: "enabled in settings; the WSL2 network allowlist is best effort"}
		}
	}
	return doctorCheck{Name: "host-sandbox", Status: "not-verified",
		Detail: "not configured; production will run without an operating-system sandbox",
		Next:   "see the sandbox settings fragment in the plugin's sandbox reference"}
}

func firstFound(ctx context.Context, tool string) provider.Adapter {
	adapter, err := findAdapter(ctx, tool, provider.Seat{Tool: tool})
	if err != nil {
		return nil
	}
	return adapter
}

// probeCommand runs one version-style probe and returns its first output line.
func probeCommand(ctx context.Context, name string, args ...string) string {
	result, err := provider.Run(ctx, provider.Request{
		Argv:    append([]string{name}, args...),
		Timeout: 30 * time.Second,
	})
	if err != nil || result.ExitCode != 0 {
		return ""
	}
	line, _, _ := strings.Cut(strings.TrimSpace(string(result.Stdout)), "\n")
	return strings.TrimSpace(line)
}

func printDoctor(env *Env, checks []doctorCheck) {
	fmt.Fprintln(env.Stdout, "peerfoil doctor")
	fmt.Fprintln(env.Stdout, "")
	for _, check := range checks {
		fmt.Fprintf(env.Stdout, "%-24s %-13s %s\n", check.Name, check.Status, check.Detail)
		if check.Next != "" {
			fmt.Fprintf(env.Stdout, "%-24s %-13s Next step: %s\n", "", "", check.Next)
		}
	}
	fmt.Fprintln(env.Stdout, "")
}
