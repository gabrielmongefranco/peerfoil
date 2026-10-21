// This file is part of PeerFoil.
// internal/errs/errs.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Stable error categories and exit codes for the peerfoil command. Every
// failure names what happened and, where possible, the one next step.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

// Package errs defines the error categories the implementation plan requires and maps
// each one to a stable exit code, so scripts and people can tell failure kinds apart.
package errs

import (
	"context"
	"errors"
	"fmt"
)

// Category names the kind of failure. The set is fixed for Core Alpha; a new kind
// needs a decision-log entry because exit codes are a public contract.
type Category string

const (
	Usage          Category = "usage"
	Configuration  Category = "configuration"
	Authentication Category = "authentication"
	Validation     Category = "validation"
	Policy         Category = "policy"
	Provider       Category = "provider"
	Git            Category = "git"
	Evidence       Category = "evidence"
	Timeout        Category = "timeout"
	Cancelled      Category = "cancelled"
	Internal       Category = "internal"
)

// exitCodes is the public mapping. 0 is success and 1 is an uncategorized failure.
var exitCodes = map[Category]int{
	Usage:          2,
	Configuration:  3,
	Authentication: 4,
	Validation:     5,
	Policy:         6,
	Provider:       7,
	Git:            8,
	Evidence:       9,
	Timeout:        10,
	Cancelled:      11,
	Internal:       12,
}

// Error is a categorized failure with an optional next step for the person running
// the command. Message says what happened; Next says what to do about it.
type Error struct {
	Category Category
	Message  string
	Next     string
	Err      error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Category, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Category, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

// New creates a categorized error. next may be empty when there is genuinely nothing
// the person can do, and the CLI then says so.
func New(category Category, message, next string) *Error {
	return &Error{Category: category, Message: message, Next: next}
}

// Wrap attaches a category and message to an underlying error.
func Wrap(category Category, message string, err error) *Error {
	return &Error{Category: category, Message: message, Err: err}
}

// CategoryOf returns the error's category, or Internal for an uncategorized one.
// Context cancellation and deadline errors map to their own categories even when a
// lower layer forgot to wrap them.
func CategoryOf(err error) Category {
	var e *Error
	if errors.As(err, &e) {
		return e.Category
	}
	if errors.Is(err, context.Canceled) {
		return Cancelled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return Timeout
	}
	return Internal
}

// ExitCode maps an error to the process exit code. nil is success.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	if code, ok := exitCodes[CategoryOf(err)]; ok {
		return code
	}
	return 1
}

// NextStep returns the recorded next step, or an empty string.
func NextStep(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Next
	}
	return ""
}
