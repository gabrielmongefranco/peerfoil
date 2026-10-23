// This file is part of PeerFoil.
// cmd/peerfoil/main.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Entry point for the peerfoil command. Wires interrupt handling to the
// command dispatcher and turns its result into the process exit code.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/gabrielmongefranco/peerfoil/internal/cli"
)

func main() {
	// Ctrl+C cancels the context; every outside operation watches it so the run
	// stops at a recorded boundary instead of mid-write.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	code := cli.Run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}
