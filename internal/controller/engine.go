// This file is part of PeerFoil.
// internal/controller/engine.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: The controller engine: the wired dependencies and the gates every enforced
// step checks before it runs.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package controller

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/evidence"
	"github.com/gabrielmongefranco/peerfoil/internal/pack"
	"github.com/gabrielmongefranco/peerfoil/internal/project"
	"github.com/gabrielmongefranco/peerfoil/internal/provider"
	"github.com/gabrielmongefranco/peerfoil/internal/store"
	"github.com/gabrielmongefranco/peerfoil/internal/workspace"
)

// Deps is everything the engine acts through. Tests inject fakes for the adapters and
// the evidence runner; everything else is real.
type Deps struct {
	State          *project.State
	Repo           *workspace.Repo
	Journal        *store.Store
	Pack           *pack.Pack
	Producer       provider.Adapter
	ProducerSeat   provider.Seat
	RepairProducer provider.Adapter
	RepairSeat     provider.Seat
	Reviewers      []provider.Adapter
	ReviewerSeats  []provider.Seat
	Evidence       evidence.Runner
	WorktreeParent string
	Out            io.Writer
	Now            func() time.Time
}

// Engine runs one permitted workflow step at a time.
type Engine struct {
	Deps
}

// NewEngine wires the dependencies and applies defaults.
func NewEngine(deps Deps) *Engine {
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.Out == nil {
		deps.Out = io.Discard
	}
	return &Engine{Deps: deps}
}

// reviewRecord is the part of a recorded review the production gate needs.
type reviewRecord struct {
	ID              string
	Kind            string
	Decision        string
	Independence    string
	IndependenceWhy string
	PlanRevision    int
	ArchRevision    int
	QualityRevision int
}

var (
	reviewKindLine         = regexp.MustCompile(`(?m)^- \*\*Kind:\*\* *([a-z-]+)`)
	reviewDecisionLine     = regexp.MustCompile(`(?m)^- \*\*Decision:\*\* *([a-z-]+)`)
	reviewIndependenceLine = regexp.MustCompile(`(?m)^- \*\*Independence:\*\* *([a-z-]+)(.*)$`)
	reviewPlanRevision     = regexp.MustCompile(`plan revision ([0-9]+)`)
	reviewArchRevision     = regexp.MustCompile(`architecture revision ([0-9]+)`)
	reviewQualityRevision  = regexp.MustCompile(`quality revision ([0-9]+)`)
)

// loadReviews parses the field lines of the recorded reviews. The records are
// Markdown written for people; only the labeled field lines are read, and a record
// that cannot be parsed is skipped rather than guessed at.
func (e *Engine) loadReviews() ([]reviewRecord, error) {
	dir := filepath.Join(e.State.Dir, "reviews")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errs.Wrap(errs.Configuration, "reading the reviews directory", err)
	}
	var reviews []reviewRecord
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "rv-") && strings.HasSuffix(entry.Name(), ".md") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, errs.Wrap(errs.Configuration, "reading "+name, err)
		}
		text := string(raw)
		record := reviewRecord{ID: strings.TrimSuffix(name, ".md")}
		if match := reviewKindLine.FindStringSubmatch(text); match != nil {
			record.Kind = match[1]
		}
		if match := reviewDecisionLine.FindStringSubmatch(text); match != nil {
			record.Decision = match[1]
		}
		if match := reviewIndependenceLine.FindStringSubmatch(text); match != nil {
			record.Independence = match[1]
			record.IndependenceWhy = strings.ToLower(match[2])
		}
		if match := reviewPlanRevision.FindStringSubmatch(text); match != nil {
			fmt.Sscanf(match[1], "%d", &record.PlanRevision)
		}
		if match := reviewArchRevision.FindStringSubmatch(text); match != nil {
			fmt.Sscanf(match[1], "%d", &record.ArchRevision)
		}
		if match := reviewQualityRevision.FindStringSubmatch(text); match != nil {
			fmt.Sscanf(match[1], "%d", &record.QualityRevision)
		}
		reviews = append(reviews, record)
	}
	return reviews, nil
}

// productionGate checks everything production needs: the right state, accepted and
// reviewed architecture and plan at the current revisions, and no interrupted work.
func (e *Engine) productionGate() error {
	p := e.State.Project
	if p.Workflow.State != string(Produce) {
		return errs.New(errs.Policy,
			fmt.Sprintf("production is not available in the %s state", p.Workflow.State),
			"run peerfoil status to see the next permitted step")
	}
	if p.Revisions.Architecture < 1 || p.Revisions.Quality < 1 || p.Revisions.Plan < 1 {
		return errs.New(errs.Policy,
			"production needs an accepted architecture, Quality Contract, and plan",
			"finish those steps with the PeerFoil skills first")
	}
	if e.State.Plan == nil || e.State.Plan.Status != "accepted" {
		return errs.New(errs.Policy, "the plan is not accepted",
			"accept the plan with the PeerFoil skills first")
	}
	reviews, err := e.loadReviews()
	if err != nil {
		return err
	}
	// A usable review names its decision and its independence, and it must have
	// inspected the current Quality Contract revision as well. A non-independent
	// review (D-0014: local-family, secondary, or reduced) counts only when its
	// recorded reason carries the user's acceptance, because that acceptance is the
	// method's condition for proceeding without a different family. Only the
	// method's exact positive phrase counts: a looser match such as any word
	// containing "accept" would also match a recorded refusal like "the user did
	// not accept Reduced assurance".
	usable := func(review reviewRecord) bool {
		switch review.Independence {
		case "independent":
			return true
		case "local-family", "secondary", "reduced":
			return strings.Contains(review.IndependenceWhy, "user accepted reduced assurance")
		}
		return false
	}
	approved := func(kind string, matches func(reviewRecord) bool) bool {
		for _, review := range reviews {
			if review.Kind == kind && review.Decision == "approve" && usable(review) &&
				review.QualityRevision == p.Revisions.Quality && matches(review) {
				return true
			}
		}
		return false
	}
	architectureReviewed := approved("architecture", func(r reviewRecord) bool {
		return r.ArchRevision == p.Revisions.Architecture
	})
	planReviewed := approved("plan", func(r reviewRecord) bool {
		return r.PlanRevision == p.Revisions.Plan
	})
	// A revised plan may carry its acceptance through change intake instead of a
	// fresh full review, so a change review of the current revision also counts.
	if !planReviewed {
		planReviewed = approved("change", func(r reviewRecord) bool {
			return r.PlanRevision == p.Revisions.Plan
		})
	}
	if !architectureReviewed || !planReviewed {
		return errs.New(errs.Policy,
			"production needs an approved review, with its independence recorded, of the current architecture and plan",
			"run the reviews with the PeerFoil skills, or check .peerfoil/reviews for the missing record")
	}
	open, err := e.Journal.OpenInvocations()
	if err != nil {
		return err
	}
	if len(open) > 0 {
		return errs.New(errs.Policy,
			fmt.Sprintf("an earlier operation (%s) has no recorded result", open[0].Operation),
			"run peerfoil resume so the interrupted work is inspected before anything new starts")
	}
	return nil
}

// requireCleanOutsideRecords stops when the working tree holds changes other than the
// controller's own .peerfoil records, which the user commits at their own pace.
func (e *Engine) requireCleanOutsideRecords(dirty []string) error {
	var foreign []string
	for _, path := range dirty {
		normalized := strings.TrimPrefix(filepath.ToSlash(path), "./")
		if !strings.HasPrefix(normalized, ".peerfoil/") {
			foreign = append(foreign, normalized)
		}
	}
	if len(foreign) == 0 {
		return nil
	}
	if len(foreign) > 5 {
		foreign = append(foreign[:5], "…")
	}
	return errs.New(errs.Git,
		"the working tree has uncommitted changes outside .peerfoil: "+strings.Join(foreign, ", "),
		"commit or stash your changes so produced work starts from a known revision")
}
