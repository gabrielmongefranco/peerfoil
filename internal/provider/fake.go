// This file is part of PeerFoil.
// internal/provider/fake.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Deterministic scripted adapter so controller and workflow tests can
// run complete journeys without a real model tool or network access.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package provider

import (
	"context"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
)

// Fake is a scripted Adapter. Each Probe and Invoke pops the next scripted
// result in order, and every Invoke records its Invocation in Calls so a test
// can assert exactly what the controller asked for.
type Fake struct {
	// NameValue is returned by Name; empty means "fake".
	NameValue string
	// Probes are returned in order; after the last one, the last repeats so a
	// re-check stays deterministic. Empty means a zero ProbeResult.
	Probes []ProbeResult
	// Outcomes and Errs are popped in parallel by Invoke; a missing entry in one
	// slice means a zero value for that call.
	Outcomes []Outcome
	Errs     []error
	// Calls records every Invocation in the order received.
	Calls []Invocation

	probeIndex  int
	invokeIndex int
}

var _ Adapter = (*Fake)(nil)

// Name identifies the fake in records and messages.
func (f *Fake) Name() string {
	if f.NameValue == "" {
		return "fake"
	}
	return f.NameValue
}

// LineageRoot applies the same mapping as the real adapters, so lineage and
// independence rules behave identically under test.
func (f *Fake) LineageRoot(model string) string { return lineageRoot(model) }

// Probe returns the next scripted ProbeResult.
func (f *Fake) Probe(_ context.Context) ProbeResult {
	if len(f.Probes) == 0 {
		return ProbeResult{}
	}
	index := f.probeIndex
	if index >= len(f.Probes) {
		index = len(f.Probes) - 1
	}
	f.probeIndex++
	return f.Probes[index]
}

// Invoke records the invocation and returns the next scripted outcome and
// error. Running past the script is an Internal error rather than a silent zero
// value, so an unexpected extra call fails the test loudly.
func (f *Fake) Invoke(_ context.Context, inv Invocation) (Outcome, error) {
	f.Calls = append(f.Calls, inv)
	index := f.invokeIndex
	f.invokeIndex++
	if index >= len(f.Outcomes) && index >= len(f.Errs) {
		return Outcome{}, errs.New(errs.Internal,
			"the fake adapter was called more times than its script covers",
			"script one outcome or error for every expected call")
	}
	var outcome Outcome
	if index < len(f.Outcomes) {
		outcome = f.Outcomes[index]
	}
	var err error
	if index < len(f.Errs) {
		err = f.Errs[index]
	}
	return outcome, err
}
