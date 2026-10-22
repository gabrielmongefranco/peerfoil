// This file is part of PeerFoil.
// internal/controller/repair_verification.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Retains and resumes a bounded repair verification without rerunning its writer.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/fsx"
	"github.com/gabrielmongefranco/peerfoil/internal/project"
	"github.com/gabrielmongefranco/peerfoil/internal/provider"
	"github.com/gabrielmongefranco/peerfoil/internal/workspace"
)

// repairVerification is controller-owned recovery data in the phase review record.
// Attempts are retained before each read-only launch, including launches interrupted
// before a result arrived. Two launches implement the method's one no-result retry.
type repairVerification struct {
	PhaseReview string            `json:"phase_review"`
	ChangeSet   string            `json:"change_set"`
	Task        string            `json:"task"`
	Branch      string            `json:"branch,omitempty"`
	Revisions   project.Revisions `json:"revisions"`
	Base        string            `json:"base_revision"`
	Source      string            `json:"source_revision"`
	Author      project.Actor     `json:"author"`
	Findings    []Finding         `json:"findings"`
	Attempts    int               `json:"attempts"`
	Approved    *repairApproval   `json:"approved,omitempty"`
}

type repairApproval struct {
	Review   string        `json:"review"`
	SHA256   string        `json:"sha256"`
	Digest   string        `json:"tree_digest"`
	Reviewer project.Actor `json:"reviewer"`
}

const repairVerificationHeading = "\n## Repair verification recovery\n\n"

var (
	phaseReviewIDPattern = regexp.MustCompile(`^pr-[0-9]{4,}$`)
	changeSetIDPattern   = regexp.MustCompile(`^cs-[0-9]{4,}$`)
	repairTaskIDPattern  = regexp.MustCompile(`^tk-[0-9]{3,}$`)
	commitIDPattern      = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)
	reviewIDPattern      = regexp.MustCompile(`^rv-[0-9]{4,}$`)
	sha256Pattern        = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

func (e *Engine) saveRepairVerification(checkpoint *repairVerification) error {
	if !phaseReviewIDPattern.MatchString(checkpoint.PhaseReview) {
		return errs.New(errs.Validation, "the repair's phase review identifier is invalid", "inspect the retained phase review")
	}
	path := filepath.Join(e.State.Dir, "reviews", checkpoint.PhaseReview+".md")
	raw, err := os.ReadFile(path)
	if err != nil {
		return errs.Wrap(errs.Configuration, "reading the repair's phase review", err)
	}
	prefix, _, _ := strings.Cut(strings.ReplaceAll(string(raw), "\r\n", "\n"), repairVerificationHeading)
	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return errs.Wrap(errs.Internal, "encoding repair verification recovery", err)
	}
	text := prefix + repairVerificationHeading + "```json\n" + string(data) + "\n```\n"
	return fsx.WriteFileAtomic(path, []byte(text), 0o644)
}

func (e *Engine) loadRepairVerification(boundary *project.Transition) (*repairVerification, error) {
	invalid := func() (*repairVerification, error) {
		return nil, errs.New(errs.Validation, "the pending repair verification does not match its recorded boundary",
			"restore the phase review and change set from the validated repair before running peerfoil resume")
	}
	if boundary == nil || boundary.SourceRevision == nil ||
		len(boundary.Refs["phase_reviews"]) != 1 || len(boundary.Refs["change_sets"]) != 1 {
		return invalid()
	}
	phaseID, changeID := boundary.Refs["phase_reviews"][0], boundary.Refs["change_sets"][0]
	if !phaseReviewIDPattern.MatchString(phaseID) || !changeSetIDPattern.MatchString(changeID) {
		return invalid()
	}
	raw, err := os.ReadFile(filepath.Join(e.State.Dir, "reviews", phaseID+".md"))
	if err != nil {
		return nil, errs.Wrap(errs.Configuration, "reading pending repair verification", err)
	}
	_, section, found := strings.Cut(strings.ReplaceAll(string(raw), "\r\n", "\n"), repairVerificationHeading)
	section = strings.TrimSpace(section)
	// Two bounded reviewer answers gain indentation and provenance when retained.
	if !found || len(section) > 8*maxHandoffBytes || !strings.HasPrefix(section, "```json\n") || !strings.HasSuffix(section, "\n```") {
		return invalid()
	}
	decoder := json.NewDecoder(strings.NewReader(strings.TrimSuffix(strings.TrimPrefix(section, "```json\n"), "\n```")))
	decoder.DisallowUnknownFields()
	var checkpoint repairVerification
	if decoder.Decode(&checkpoint) != nil || decoder.Decode(new(any)) != io.EOF {
		return invalid()
	}
	if checkpoint.PhaseReview != phaseID || checkpoint.ChangeSet != changeID ||
		checkpoint.Source != *boundary.SourceRevision || checkpoint.Revisions != e.State.Project.Revisions ||
		boundary.PlanRevision != checkpoint.Revisions.Plan ||
		!repairTaskIDPattern.MatchString(checkpoint.Task) || !commitIDPattern.MatchString(checkpoint.Source) ||
		!commitIDPattern.MatchString(checkpoint.Base) || checkpoint.Attempts < 0 || checkpoint.Attempts > 2 ||
		checkpoint.Author.Role != "repair_producer" || checkpoint.Author.LineageRoot == "" ||
		checkpoint.Author.LineageRoot == provider.LineageUnknown || len(checkpoint.Findings) == 0 || len(checkpoint.Findings) > 2*maxFindings {
		return invalid()
	}
	if checkpoint.Branch != "" && checkpoint.Branch != fmt.Sprintf("peerfoil/task/%s-r%d", checkpoint.Task, checkpoint.Revisions.Plan) {
		return invalid()
	}
	// The checkpoint identifies a captured repair, not an arbitrary branch commit.
	change, err := os.ReadFile(filepath.Join(e.State.Dir, "evidence", changeID+".md"))
	if err != nil {
		return nil, errs.Wrap(errs.Configuration, "reading the pending repair's authorship", err)
	}
	author := checkpoint.Author
	authorLine := fmt.Sprintf("- **Author:** %s (%s, %s, %s, %s, session %s)\n",
		author.Role, author.Tool, deref(author.Model), deref(author.Effort), author.LineageRoot, deref(author.Session))
	changeText := strings.ReplaceAll(string(change), "\r\n", "\n")
	for _, line := range []string{
		"- **Capture status:** captured\n", "- **Task:** " + checkpoint.Task + "\n",
		"- **Base revision:** " + checkpoint.Base + "\n", "- **Source revision:** " + checkpoint.Source + "\n", authorLine,
	} {
		if !strings.Contains(changeText, "\n"+line) {
			return invalid()
		}
	}
	return &checkpoint, nil
}

func (e *Engine) resumeRepairVerification(ctx context.Context, boundary *project.Transition) error {
	checkpoint, err := e.loadRepairVerification(boundary)
	if err != nil {
		return err
	}
	refs := map[string][]string{"phase_reviews": {checkpoint.PhaseReview}, "change_sets": {checkpoint.ChangeSet}}
	if err := e.checkReviewInputs(boundary); err != nil {
		return err
	}
	if checkpoint.Approved != nil {
		return e.integrateVerifiedRepair(ctx, boundary, checkpoint)
	}
	if checkpoint.Attempts >= 2 {
		reason := "the repair verifier returned no usable result within two attempts"
		if err := e.Pause(reason, refs); err != nil {
			return err
		}
		return errs.New(errs.Policy, reason, "inspect the retained repair and continue its review with the PeerFoil skills")
	}
	var verifier provider.Adapter
	var seat provider.Seat
	for index, reviewer := range e.Reviewers {
		lineage := reviewer.LineageRoot(e.ReviewerSeats[index].Model)
		if lineage != provider.LineageUnknown && lineage != checkpoint.Author.LineageRoot {
			verifier, seat = reviewer, e.ReviewerSeats[index]
			break
		}
	}
	if verifier == nil {
		return errs.New(errs.Configuration, "no different-family repair verifier is available", "restore the reviewer seat, then run peerfoil resume")
	}
	branch := repairBranch(checkpoint)
	branchTip, err := e.Repo.CommitOf(ctx, branch)
	if err != nil {
		return err
	}
	tip, err := e.Repo.CommitOf(ctx, e.Repo.IntegrationBranch(checkpoint.Revisions.Plan))
	if err != nil {
		return err
	}
	if branchTip != checkpoint.Source || (tip != checkpoint.Base && tip != checkpoint.Source) {
		return errs.New(errs.Git, "a branch moved after the repair was validated", "restore the recorded repair and integration revisions before resuming")
	}
	checkout, err := e.Repo.CreateReviewWorktree(ctx, checkpoint.Source, "repair-verification", e.WorktreeParent)
	if err != nil {
		return err
	}
	defer func() { _ = e.Repo.RemoveReviewWorktree(ctx, checkout) }()
	if err := e.refreshRecordsInCheckout(checkout); err != nil {
		return err
	}
	tree, err := e.Repo.TreeDigest(ctx, checkpoint.Source)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(tree)
	digest := hex.EncodeToString(sum[:])
	packet := e.buildReviewPacket("repair", checkpoint.Source, digest, checkpoint.Findings)
	checkpoint.Attempts++
	if err := e.saveRepairVerification(checkpoint); err != nil {
		return err
	}
	authors := map[string]string{checkpoint.ChangeSet: checkpoint.Author.LineageRoot}
	run, err := e.runReviewer(ctx, verifier, seat, "repair", packet, checkout, checkpoint.Source, authors, digest, checkpoint.ChangeSet)
	if err != nil {
		if checkpoint.Attempts >= 2 {
			if pauseErr := e.Pause("the repair verifier returned no usable result within two attempts", refs); pauseErr != nil {
				return pauseErr
			}
		}
		return errs.New(errs.Provider, "the repair verifier returned no usable result",
			"run peerfoil status; a pending verification can retry once with peerfoil resume")
	}
	fmt.Fprintf(e.Out, "Repair verification by %s: %s.\n", run.reviewerName, run.answer.Decision)
	sameSession := checkpoint.Author.Session != nil && *checkpoint.Author.Session != "" && *checkpoint.Author.Session == run.session
	if run.answer.Decision != "approve" || run.independence != "independent" || sameSession {
		reason := "the repair did not receive an independent approval; you decide how to proceed"
		refs["reviews"] = []string{run.recordID}
		if err := e.Pause(reason, refs); err != nil {
			return err
		}
		return errs.New(errs.Policy, reason, "read the repair review and continue with the PeerFoil skills")
	}
	raw, err := fsx.ReadRegularFile(e.State.Dir, filepath.Join("reviews", run.recordID+".md"), 1024*1024)
	if err != nil {
		return err
	}
	approvalHash := sha256.Sum256([]byte(strings.ReplaceAll(string(raw), "\r\n", "\n")))
	model, effort, session := run.seat.Model, run.seat.Effort, run.session
	checkpoint.Approved = &repairApproval{Review: run.recordID, SHA256: hex.EncodeToString(approvalHash[:]), Digest: digest,
		Reviewer: project.Actor{Role: "reviewer", Tool: run.seat.Tool, Model: &model, Effort: &effort, Session: &session, LineageRoot: run.lineage}}
	if err := e.saveRepairVerification(checkpoint); err != nil {
		return err
	}
	return e.integrateVerifiedRepair(ctx, boundary, checkpoint)
}

// integrateVerifiedRepair reuses an accepted verification after an integration
// failure. It validates the retained result without consuming another reviewer call.
func (e *Engine) integrateVerifiedRepair(ctx context.Context, boundary *project.Transition, checkpoint *repairVerification) error {
	approval := checkpoint.Approved
	invalid := errs.New(errs.Validation, "the retained repair approval is missing or no longer matches the repair", "restore the recorded approval before resuming integration")
	if approval == nil || !reviewIDPattern.MatchString(approval.Review) || !sha256Pattern.MatchString(approval.SHA256) || !sha256Pattern.MatchString(approval.Digest) ||
		approval.Reviewer.Role != "reviewer" || approval.Reviewer.LineageRoot == "" || approval.Reviewer.LineageRoot == provider.LineageUnknown || approval.Reviewer.LineageRoot == checkpoint.Author.LineageRoot ||
		(checkpoint.Author.Session != nil && *checkpoint.Author.Session != "" && deref(approval.Reviewer.Session) == *checkpoint.Author.Session) {
		return invalid
	}
	raw, err := fsx.ReadRegularFile(e.State.Dir, filepath.Join("reviews", approval.Review+".md"), 1024*1024)
	if err != nil {
		return invalid
	}
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	sum := sha256.Sum256([]byte(text))
	if hex.EncodeToString(sum[:]) != approval.SHA256 {
		return invalid
	}
	actor := approval.Reviewer
	for _, line := range []string{
		"- **Kind:** repair\n", "- **Decision:** approve\n",
		"- **Independence:** independent — judged against each item's recorded author lineage\n",
		fmt.Sprintf("- **Frozen material:** source revision %s, plan revision %d, architecture revision %d, quality revision %d, pass 1, digest %s, change set %s\n",
			checkpoint.Source, checkpoint.Revisions.Plan, checkpoint.Revisions.Architecture, checkpoint.Revisions.Quality, approval.Digest, checkpoint.ChangeSet),
		fmt.Sprintf("- **Reviewer:** reviewer (%s, %s, %s, %s, session %s)\n", actor.Tool, deref(actor.Model), deref(actor.Effort), actor.LineageRoot, valueOrNull(deref(actor.Session))),
	} {
		if !strings.Contains(text, "\n"+line) {
			return invalid
		}
	}
	if err := e.checkReviewInputs(boundary); err != nil {
		return err
	}
	branch := repairBranch(checkpoint)
	branchTip, err := e.Repo.CommitOf(ctx, branch)
	if err != nil {
		return err
	}
	tip, err := e.Repo.CommitOf(ctx, e.Repo.IntegrationBranch(checkpoint.Revisions.Plan))
	if err != nil {
		return err
	}
	if branchTip != checkpoint.Source || (tip != checkpoint.Base && tip != checkpoint.Source) {
		return errs.New(errs.Git, "a branch moved after the repair was verified", "restore the recorded repair and integration revisions, then run peerfoil resume")
	}
	integrationTip, err := e.Repo.Integrate(ctx, &workspace.Worktree{Repo: e.Repo, Branch: branch}, checkpoint.Source)
	if err != nil {
		return err
	}
	refs := map[string][]string{"phase_reviews": {checkpoint.PhaseReview}, "change_sets": {checkpoint.ChangeSet}, "reviews": {approval.Review}}
	return e.approvePhase(integrationTip, refs,
		"Phase approved after a verified repair; see "+checkpoint.PhaseReview+", "+approval.Review+", and "+checkpoint.ChangeSet+".")
}

func repairBranch(checkpoint *repairVerification) string {
	if checkpoint.Branch != "" {
		return checkpoint.Branch
	}
	return fmt.Sprintf("peerfoil/task/%s-r%d", checkpoint.Task, checkpoint.Revisions.Plan+1)
}
