// This file is part of PeerFoil.
// embed.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Embeds the JSON schemas and project-pack manifests into the peerfoil binary,
// so Core validates the same record and pack formats the Skills release ships (D-0009).
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

// Package peerfoil carries the repository assets the Core binary ships a copy of.
package peerfoil

import "embed"

// Assets holds the record schemas and the built-in project-pack manifests. The files
// are read-only copies; the repository files remain the source of truth.
//
//go:embed schemas/*.schema.json plugins/peerfoil/packs/*/pack.json
var Assets embed.FS
