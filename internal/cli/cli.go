// This file is part of PeerFoil.
// internal/cli/cli.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Command dispatcher for the peerfoil application: global flags, help text,
// the five commands, and the mapping from errors to messages and exit codes.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

// Package cli parses arguments, runs one command, and reports the result. It never
// prompts interactively: Core Alpha asks through its output and a nonzero exit code,
// so it behaves the same in a terminal, a script, and continuous integration.
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/version"
)

// Env carries what every command needs: where to write and where it runs. Dir is
// always an absolute path by the time a command sees it.
type Env struct {
	Stdout io.Writer
	Stderr io.Writer
	Dir    string
}

type command struct {
	name    string
	summary string
	help    string
	run     func(ctx context.Context, env *Env, args []string) error
}

func commands() []command {
	return []command{
		{name: "init", summary: "Create the .peerfoil project records for this repository.", help: initHelp, run: runInit},
		{name: "doctor", summary: "Check that this computer and repository are ready.", help: doctorHelp, run: runDoctor},
		{name: "start", summary: "Run the next step of the planned software phase.", help: startHelp, run: runStart},
		{name: "status", summary: "Show where the project stands and what comes next.", help: statusHelp, run: runStatus},
		{name: "resume", summary: "Continue safely from the last recorded boundary.", help: resumeHelp, run: runResume},
	}
}

// Run executes one command and returns the process exit code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	global := flag.NewFlagSet("peerfoil", flag.ContinueOnError)
	global.SetOutput(io.Discard)
	dir := global.String("C", "", "")
	showVersion := global.Bool("version", false, "")
	showHelp := global.Bool("help", false, "")

	if err := global.Parse(args); err != nil {
		fmt.Fprintln(stderr, "peerfoil: unknown option before the command name.")
		fmt.Fprintln(stderr, "Next step: run peerfoil --help to see the options and commands.")
		return errs.ExitCode(errs.New(errs.Usage, "unknown global option", ""))
	}
	if *showVersion {
		fmt.Fprintln(stdout, "peerfoil "+version.String())
		return 0
	}
	rest := global.Args()
	if *showHelp || len(rest) == 0 {
		printHelp(stdout)
		return 0
	}

	name := rest[0]
	var selected *command
	for _, c := range commands() {
		if c.name == name {
			selected = &c
			break
		}
	}
	if selected == nil {
		fmt.Fprintf(stderr, "peerfoil: %q is not a peerfoil command.\n", name)
		fmt.Fprintf(stderr, "Next step: run peerfoil --help; the commands are %s.\n", commandNames())
		return errs.ExitCode(errs.New(errs.Usage, "unknown command", ""))
	}

	env := &Env{Stdout: stdout, Stderr: stderr}
	workDir := *dir
	if workDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return report(stderr, errs.Wrap(errs.Configuration, "cannot read the current directory", err))
		}
		workDir = cwd
	}
	absolute, err := absDir(workDir)
	if err != nil {
		return report(stderr, errs.Wrap(errs.Configuration,
			fmt.Sprintf("cannot use %q as the working directory", workDir), err))
	}
	env.Dir = absolute

	// Command-level --help works without running the command.
	if len(rest) > 1 && (rest[1] == "--help" || rest[1] == "-help" || rest[1] == "-h") {
		fmt.Fprintln(stdout, strings.TrimSpace(selected.help))
		return 0
	}

	if err := selected.run(ctx, env, rest[1:]); err != nil {
		return report(stderr, err)
	}
	return 0
}

// report prints what happened and what to do next, then returns the exit code.
// Messages go to standard error so machine-readable output stays clean.
func report(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "peerfoil: %v\n", err)
	if next := errs.NextStep(err); next != "" {
		fmt.Fprintf(stderr, "Next step: %s\n", next)
	}
	return errs.ExitCode(err)
}

func absDir(dir string) (string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", dir)
	}
	return absPath(dir)
}

func commandNames() string {
	names := make([]string, 0, 5)
	for _, c := range commands() {
		names = append(names, c.name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func printHelp(out io.Writer) {
	fmt.Fprintf(out, `PeerFoil Core Alpha, a local controller for the PeerFoil workflow.
It reads the project records under .peerfoil/, runs one software task at a time,
records evidence, and stops when it needs you. Nothing leaves this computer except
the model requests you already signed in for.

Usage:
  peerfoil [-C <directory>] <command> [options]

Commands:
`)
	for _, c := range commands() {
		fmt.Fprintf(out, "  %-8s %s\n", c.name, c.summary)
	}
	fmt.Fprintf(out, `
Options:
  -C <directory>  Run as if peerfoil started in this directory.
  --version       Print the version and exit.
  --help          Print this help and exit.

Run peerfoil <command> --help for one command's options.

Exit codes: 0 success, 2 usage, 3 configuration, 4 authentication, 5 validation,
6 policy, 7 provider, 8 git, 9 evidence, 10 timeout, 11 cancelled, 12 internal.
`)
}
