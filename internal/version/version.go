// This file is part of PeerFoil.
// internal/version/version.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Version metadata for the peerfoil command, read locally with no network call.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

// Package version holds the release identity the rest of the program reports.
package version

import "runtime/debug"

// Release names the product release the way project records expect it.
const Release = "peerfoil-core/0.2.0-dev"

// Version is the short version string printed by --version.
const Version = "0.2.0-dev"

// String returns the version with the Git revision when the binary carries build
// information, such as "0.2.0-dev (abc1234)". It never touches the network.
func String() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && len(setting.Value) >= 7 {
				return Version + " (" + setting.Value[:7] + ")"
			}
		}
	}
	return Version
}
