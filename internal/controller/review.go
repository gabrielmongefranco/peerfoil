// This file is part of PeerFoil.
// internal/controller/review.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: The Core phase review: freeze the reviewed revision, run one fresh Claude
// and one fresh Codex review of the same bundle, block self-approval, and guide one
// repair with fresh different-family verification.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/gabrielmongefranco/peerfoil/internal/errs"
	"github.com/gabrielmongefranco/peerfoil/internal/fsx"
	"github.com/gabrielmongefranco/peerfoil/internal/project"
	"github.com/gabrielmongefranco/peerfoil/internal/provider"
	"github.com/gabrielmongefranco/peerfoil/internal/version"
	"github.com/gabrielmongefranco/peerfoil/internal/workspace"
)

// Finding is one normalized reviewer finding.
type Finding struct {
	Title          string `json:"title"`
	Location       string `json:"location"`
	Requirement    string `json:"requirement"`
	Severity       string `json:"severity"`
	Evidence       string `json:"evidence"`
	Recommendation string `json:"recommendation"`
}

// reviewAnswer is the one JSON object a reviewer must return.
type reviewAnswer struct {
	Kind     string `json:"kind"`
	Reviewed struct {
		SourceRevision       string `json:"source_revision"`
		PlanRevision         int    `json:"plan_revision"`
		ArchitectureRevision int    `json:"architecture_revision"`
	} `json:"reviewed"`
	Model         string    `json:"model"`
	Decision      string    `json:"decision"`
	Findings      []Finding `json:"findings"`
	RemainingRisk string    `json:"remaining_risk"`
}

// reviewRun is one completed reviewer pass with its recorded identity.
type reviewRun struct {
	answer       reviewAnswer
	reviewerName string
	seat         provider.Seat
	lineage      string
	session      string
	independence string
	recordID     string
	duration     float64
}

const maxFindings = 10

var validSeverities = map[string]bool{"blocking": true, "major": true, "minor": true, "note": true}

// changeSetAuthorLine extracts the author lineage from a recorded change set.
var changeSetAuthorLine = regexp.MustCompile(`(?m)^- \*\*Author:\*\* *[a-z_]+ \(([^)]+)\)`)

// PhaseReview runs the one fixed two-family review round and, when both reviewers
// agree on a repair, one guided repair with fresh different-family verification.
func (e *Engine) PhaseReview(ctx context.Context) error {
	if e.State.Project.Workflow.State != string(Review) {
		return errs.New(errs.Policy,
			fmt.Sprintf("the phase review is not available in the %s state", e.State.Project.Workflow.State),
			"run peerfoil status")
	}
	// A blocked or stale task means a required failure is outstanding; reviewers
	// cannot approve around it, so the review does not start.
	if err := e.blockedTaskError(); err != nil {
		return err
	}
	if err := e.requireTwoFamilies(); err != nil {
		return err
	}

	frozen, err := e.Repo.CommitOf(ctx, e.Repo.IntegrationBranch(e.State.Project.Revisions.Plan))
	if err != nil {
		return err
	}
	tree, err := e.Repo.TreeDigest(ctx, frozen)
	if err != nil {
		return err
	}
	digestBytes := sha256.Sum256(tree)
	bundleDigest := hex.EncodeToString(digestBytes[:])

	authors, err := e.materialAuthors()
	if err != nil {
		return err
	}

	reviewPath, err := e.Repo.CreateReviewWorktree(ctx, frozen, "phase", e.WorktreeParent)
	if err != nil {
		return err
	}
	defer func() { _ = e.Repo.RemoveReviewWorktree(ctx, reviewPath) }()

	packet := e.buildReviewPacket("phase", frozen, bundleDigest, nil)
	var runs []reviewRun
	for index, reviewer := range e.Reviewers {
		run, err := e.runReviewer(ctx, reviewer, e.ReviewerSeats[index], "phase", packet, reviewPath, frozen, authors, bundleDigest, "")
		if err != nil {
			return err
		}
		runs = append(runs, *run)
		fmt.Fprintf(e.Out, "Reviewer %s (%s): %s with %d finding(s).\n",
			run.reviewerName, run.lineage, run.answer.Decision, len(run.answer.Findings))
	}

	merged := mergeFindings(runs)
	phaseReviewID, err := nextID(filepath.Join(e.State.Dir, "reviews"), "pr", 4)
	if err != nil {
		return err
	}
	decision := combineDecisions(runs)
	if err := e.writePhaseReviewRecord(phaseReviewID, frozen, bundleDigest, authors, runs, merged, decision); err != nil {
		return err
	}

	switch decision {
	case "approve":
		return e.approvePhase(frozen, phaseReviewID)
	case "repair":
		return e.repairCycle(ctx, frozen, phaseReviewID, blockingFindings(merged), authors)
	default:
		reason := "the phase reviewers did not reach an agreed decision (" + decisionSummary(runs) + "); you decide how to proceed"
		if err := e.Pause(reason, map[string][]string{"phase_reviews": {phaseReviewID}}); err != nil {
			return err
		}
		return errs.New(errs.Policy, reason,
			"read .peerfoil/reviews/"+phaseReviewID+".md, then continue with the PeerFoil skills or adjust the work")
	}
}

// requireTwoFamilies confirms two reviewer seats from different, known families.
func (e *Engine) requireTwoFamilies() error {
	families := map[string]bool{}
	for index, reviewer := range e.Reviewers {
		lineage := reviewer.LineageRoot(e.ReviewerSeats[index].Model)
		if lineage != provider.LineageUnknown {
			families[lineage] = true
		}
	}
	if len(e.Reviewers) < 2 || len(families) < 2 {
		return errs.New(errs.Configuration,
			"the phase review needs two reviewers from different model families with known models",
			"install and sign in to both Claude Code and the Codex CLI, set each reviewer seat's model, then run peerfoil resume")
	}
	return nil
}

// materialAuthors maps each recorded change set to its author lineage.
func (e *Engine) materialAuthors() (map[string]string, error) {
	dir := filepath.Join(e.State.Dir, "evidence")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, errs.New(errs.Validation, "no change sets exist to review", "produce at least one task first")
	}
	if err != nil {
		return nil, errs.Wrap(errs.Configuration, "reading the evidence directory", err)
	}
	authors := map[string]string{}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "cs-") || !strings.HasSuffix(name, ".md") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, errs.Wrap(errs.Configuration, "reading "+name, err)
		}
		lineage := provider.LineageUnknown
		if match := changeSetAuthorLine.FindStringSubmatch(string(raw)); match != nil {
			fields := strings.Split(match[1], ",")
			if len(fields) >= 4 {
				lineage = strings.TrimSpace(fields[3])
			}
		}
		authors[strings.TrimSuffix(name, ".md")] = lineage
	}
	if len(authors) == 0 {
		return nil, errs.New(errs.Validation, "no change sets exist to review", "produce at least one task first")
	}
	return authors, nil
}

// buildReviewPacket writes the self-contained review request. Reviewers read the
// frozen checkout themselves; they never see the producer's conversation.
func (e *Engine) buildReviewPacket(kind, frozen, bundleDigest string, repairFindings []Finding) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are one independent PeerFoil %s reviewer. The current directory is a read-only checkout of the exact revision under review.\n\n", kind)
	fmt.Fprintf(&b, "Project: %s · goal: %s\n", e.State.Project.ProjectID, e.State.Project.Goal)
	fmt.Fprintf(&b, "Frozen revision: %s · bundle digest: %s\n", frozen, bundleDigest)
	fmt.Fprintf(&b, "Revisions: plan %d, architecture %d, quality %d\n\n",
		e.State.Project.Revisions.Plan, e.State.Project.Revisions.Architecture, e.State.Project.Revisions.Quality)
	b.WriteString("Read the accepted records under .peerfoil/ (architecture.md, quality.md, plan.md, decisions.md, evidence/) and the deliverables they describe.\n")
	b.WriteString("Note: the .peerfoil records in this checkout reflect the phase's start; the deliverable files are the reviewed revision, and this packet's revisions and requirement list are current.\n")

	if e.State.Plan != nil {
		b.WriteString("\nRequirements to check:\n")
		for _, requirement := range e.State.Plan.Requirements {
			covered := "covered by " + strings.Join(requirement.Tasks, ", ")
			if len(requirement.Tasks) == 0 {
				covered = "covered by no task"
			}
			fmt.Fprintf(&b, "- %s: %s (%s)\n", requirement.ID, requirement.Text, covered)
		}
	}
	if kind == "repair" && len(repairFindings) > 0 {
		b.WriteString("\nThis is the verification of a repair. Confirm whether each of these findings is repaired at this exact revision:\n")
		for _, finding := range repairFindings {
			fmt.Fprintf(&b, "- [%s] %s at %s: %s\n", finding.Severity, finding.Title, finding.Location, finding.Requirement)
		}
	}
	b.WriteString(`
Review lenses: correctness and reliability; security and privacy; accessibility and
user experience; maintainability, documentation, licensing, and release readiness.

Rules: work alone from the frozen files; a claim in a record is not evidence that a
check ran. Report at most ` + fmt.Sprint(maxFindings) + ` findings. "approve" means no blocking finding
remains; "repair" means a bounded revision can resolve the findings; "block" means a
decision or requirement must change first.

End your answer with exactly one fenced JSON object:

` + "```json\n" + `{
  "kind": "` + kind + `",
  "reviewed": {"source_revision": "` + frozen + `", "plan_revision": ` + fmt.Sprint(e.State.Project.Revisions.Plan) + `, "architecture_revision": ` + fmt.Sprint(e.State.Project.Revisions.Architecture) + `},
  "model": "<the exact model identifier you are running as>",
  "decision": "approve | repair | block",
  "findings": [{"title": "", "location": "", "requirement": "", "severity": "blocking | major | minor | note", "evidence": "", "recommendation": ""}],
  "remaining_risk": ""
}
` + "```\n")
	return b.String()
}

// runReviewer invokes one fresh reviewer session, validates its answer, and records it
// with the digest and, for a repair verification, the change set it is bound to.
func (e *Engine) runReviewer(ctx context.Context, reviewer provider.Adapter, seat provider.Seat,
	kind, packet, dir, frozen string, authors map[string]string, digest, changeSetID string) (*reviewRun, error) {
	outcome, err := reviewer.Invoke(ctx, provider.Invocation{
		Role:    "reviewer",
		Packet:  packet,
		Sandbox: provider.SandboxReadOnly,
		Dir:     dir,
	})
	if err != nil {
		return nil, errs.Wrap(errs.Provider,
			fmt.Sprintf("the %s review returned no usable result", reviewer.Name()), err)
	}
	answer, err := parseReviewAnswer(outcome.Text, kind, frozen,
		e.State.Project.Revisions.Plan, e.State.Project.Revisions.Architecture)
	if err != nil {
		return nil, err
	}
	lineage := reviewer.LineageRoot(seat.Model)
	// The self-reported model is a cross-check, never the source of lineage (D-0023).
	if answer.Model != "" && reviewer.LineageRoot(answer.Model) != lineage {
		lineage = provider.LineageUnknown
	}
	run := &reviewRun{
		answer:       *answer,
		reviewerName: reviewer.Name(),
		seat:         seat,
		lineage:      lineage,
		session:      outcome.SessionID,
		duration:     outcome.Duration.Seconds(),
	}
	run.independence = independenceAgainst(lineage, authors)
	recordID, err := nextID(filepath.Join(e.State.Dir, "reviews"), "rv", 4)
	if err != nil {
		return nil, err
	}
	run.recordID = recordID
	if err := e.writeReviewRecord(run, kind, frozen, authors, digest, changeSetID); err != nil {
		return nil, err
	}
	return run, nil
}

// distinctLineages lists the material authors' lineage roots, sorted and deduplicated,
// so records show real provenance instead of an assumed constant.
func distinctLineages(authors map[string]string) string {
	seen := map[string]bool{}
	var roots []string
	for _, lineage := range authors {
		if lineage == "" {
			lineage = provider.LineageUnknown
		}
		if !seen[lineage] {
			seen[lineage] = true
			roots = append(roots, lineage)
		}
	}
	sort.Strings(roots)
	if len(roots) == 0 {
		return provider.LineageUnknown
	}
	return strings.Join(roots, "+")
}

// independenceAgainst judges one reviewer against every material item's author.
func independenceAgainst(reviewerLineage string, authors map[string]string) string {
	if reviewerLineage == provider.LineageUnknown {
		return "reduced"
	}
	independent := true
	for _, authorLineage := range authors {
		if authorLineage == provider.LineageUnknown {
			return "reduced"
		}
		if authorLineage == reviewerLineage {
			independent = false
		}
	}
	if independent {
		return "independent"
	}
	return "secondary"
}

func parseReviewAnswer(text, kind, frozen string, planRevision, architectureRevision int) (*reviewAnswer, error) {
	matches := fencedJSON.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil, errs.New(errs.Provider, "the reviewer returned no fenced JSON answer", "")
	}
	raw := matches[len(matches)-1][1]
	// The answer is untrusted model output headed for versioned records, so it gets
	// the same kind of size bound as the producer handoff.
	if len(raw) > 2*maxHandoffBytes {
		return nil, errs.New(errs.Provider, "the reviewer answer is larger than 64 KiB", "")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var answer reviewAnswer
	if err := decoder.Decode(&answer); err != nil {
		return nil, errs.Wrap(errs.Provider, "the reviewer answer is not the required JSON shape", err)
	}
	var problems []string
	if answer.Kind != kind {
		problems = append(problems, fmt.Sprintf("kind %q is not %q", answer.Kind, kind))
	}
	if answer.Reviewed.SourceRevision != frozen {
		problems = append(problems, "the reviewed revision is not the frozen revision")
	}
	if answer.Reviewed.PlanRevision != planRevision || answer.Reviewed.ArchitectureRevision != architectureRevision {
		problems = append(problems, "the reviewed plan or architecture revision is wrong")
	}
	switch answer.Decision {
	case "approve", "repair", "block":
	default:
		problems = append(problems, fmt.Sprintf("decision %q is not approve, repair, or block", answer.Decision))
	}
	if len(answer.Findings) > maxFindings {
		problems = append(problems, fmt.Sprintf("more than %d findings", maxFindings))
	}
	for index, finding := range answer.Findings {
		if finding.Title == "" || finding.Location == "" || finding.Requirement == "" ||
			finding.Evidence == "" || finding.Recommendation == "" {
			problems = append(problems, fmt.Sprintf("finding %d is missing a required field", index+1))
		}
		if !validSeverities[finding.Severity] {
			problems = append(problems, fmt.Sprintf("finding %d has severity %q", index+1, finding.Severity))
		}
		for _, value := range []string{finding.Title, finding.Location, finding.Requirement,
			finding.Evidence, finding.Recommendation} {
			if len(value) > 2000 {
				problems = append(problems, fmt.Sprintf("finding %d has a field longer than 2,000 characters", index+1))
				break
			}
		}
	}
	if len(answer.RemainingRisk) > 4000 {
		problems = append(problems, "remaining_risk is longer than 4,000 characters")
	}
	if len(problems) > 0 {
		return nil, errs.New(errs.Provider, "the review answer failed validation: "+strings.Join(problems, "; "), "")
	}
	// A blocking finding with an approve decision is corrected, never trusted.
	if answer.Decision == "approve" {
		for _, finding := range answer.Findings {
			if finding.Severity == "blocking" {
				answer.Decision = "repair"
				break
			}
		}
	}
	return &answer, nil
}

// combineDecisions applies the independence rules: approval needs both reviewers, and
// in particular the primary (different-family) reviewer of every item, to approve.
func combineDecisions(runs []reviewRun) string {
	for _, run := range runs {
		if run.answer.Decision == "block" {
			return "user-decision"
		}
	}
	approvals := 0
	repairs := 0
	primaryApproved := false
	for _, run := range runs {
		switch run.answer.Decision {
		case "approve":
			approvals++
			if run.independence == "independent" {
				primaryApproved = true
			}
		case "repair":
			repairs++
		}
	}
	switch {
	case approvals == len(runs) && primaryApproved:
		return "approve"
	case repairs == len(runs):
		return "repair"
	case approvals == len(runs) && !primaryApproved:
		// Only the author's own family approved; that cannot satisfy approval.
		return "user-decision"
	default:
		return "user-decision"
	}
}

func decisionSummary(runs []reviewRun) string {
	var parts []string
	for _, run := range runs {
		parts = append(parts, fmt.Sprintf("%s said %s", run.reviewerName, run.answer.Decision))
	}
	return strings.Join(parts, ", ")
}

func blockingFindings(findings []Finding) []Finding {
	var blocking []Finding
	for _, finding := range findings {
		if finding.Severity == "blocking" {
			blocking = append(blocking, finding)
		}
	}
	return blocking
}

// mergeFindings combines both reviewers' findings, joining duplicates by title and
// location without losing either reviewer's severity.
func mergeFindings(runs []reviewRun) []Finding {
	var merged []Finding
	index := map[string]int{}
	rank := map[string]int{"blocking": 3, "major": 2, "minor": 1, "note": 0}
	for _, run := range runs {
		for _, finding := range run.answer.Findings {
			key := strings.ToLower(finding.Title + "|" + finding.Location)
			if at, seen := index[key]; seen {
				if rank[finding.Severity] > rank[merged[at].Severity] {
					merged[at].Severity = finding.Severity
				}
				continue
			}
			index[key] = len(merged)
			merged = append(merged, finding)
		}
	}
	return merged
}

// writeReviewRecord renders one reviewer pass in the review template's field shape.
func (e *Engine) writeReviewRecord(run *reviewRun, kind, frozen string, authors map[string]string, digest, changeSetID string) error {
	dir := filepath.Join(e.State.Dir, "reviews")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return errs.Wrap(errs.Configuration, "creating the reviews directory", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- PeerFoil project record: review. Generated by %s (Guided). Keep the field names. -->\n\n", version.Release)
	fmt.Fprintf(&b, "# Review %s — %s\n\n", run.recordID, kind)
	fmt.Fprintf(&b, "Project: `%s` · Assurance: Guided\n\n", e.State.Project.ProjectID)
	fmt.Fprintf(&b, "- **Kind:** %s\n", kind)
	fmt.Fprintf(&b, "- **Frozen material:** source revision %s, plan revision %d, architecture revision %d, quality revision %d, pass 1, digest %s, change set %s\n",
		frozen, e.State.Project.Revisions.Plan, e.State.Project.Revisions.Architecture,
		e.State.Project.Revisions.Quality, orDash(digest), orDash(changeSetID))
	fmt.Fprintf(&b, "- **Reviewer:** reviewer (%s, %s, %s, %s, session %s)\n",
		run.seat.Tool, run.seat.Model, run.seat.Effort, run.lineage, valueOrNull(run.session))
	fmt.Fprintf(&b, "- **Author under review:** producer (%s, %s, session null)\n",
		e.State.Project.Settings.Roles["producer"].Tool, distinctLineages(authors))
	fmt.Fprintf(&b, "- **Independence:** %s — judged against each item's recorded author lineage\n", run.independence)
	fmt.Fprintf(&b, "- **Passes used:** 1 of %d\n", e.State.Project.Settings.Review.DefaultPasses)
	fmt.Fprintf(&b, "- **Reviewed at:** %s\n", Timestamp(e.Now()))
	fmt.Fprintf(&b, "- **Duration (seconds):** %.0f\n", run.duration)
	fmt.Fprintf(&b, "- **Decision:** %s\n\n## Findings\n\n", run.answer.Decision)
	if len(run.answer.Findings) == 0 {
		b.WriteString("None.\n\n")
	}
	for number, finding := range run.answer.Findings {
		fmt.Fprintf(&b, "### fd-%04d — %s\n\n", number+1, finding.Title)
		fmt.Fprintf(&b, "- **Location:** %s\n- **Item:** —\n- **Lens:** —\n", finding.Location)
		fmt.Fprintf(&b, "- **Requirement:** %s\n- **Severity:** %s\n", finding.Requirement, finding.Severity)
		fmt.Fprintf(&b, "- **Evidence:** %s\n- **Recommendation:** %s\n", finding.Evidence, finding.Recommendation)
		fmt.Fprintf(&b, "- **Disposition:** open\n- **Raised by:** reviewer (%s, %s)\n\n", run.seat.Tool, run.lineage)
	}
	fmt.Fprintf(&b, "## Remaining risk\n\n%s\n", orNone(run.answer.RemainingRisk))
	return fsx.WriteFileAtomic(filepath.Join(dir, run.recordID+".md"), []byte(b.String()), 0o644)
}

func (e *Engine) writePhaseReviewRecord(id, frozen, bundleDigest string, authors map[string]string,
	runs []reviewRun, merged []Finding, decision string) error {
	dir := filepath.Join(e.State.Dir, "reviews")
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- PeerFoil project record: phase review. Generated by %s. Keep the field names. -->\n\n", version.Release)
	fmt.Fprintf(&b, "# Phase review %s\n\n", id)
	fmt.Fprintf(&b, "Project: `%s`\n\n", e.State.Project.ProjectID)
	fmt.Fprintf(&b, "- **Frozen revision:** %s\n- **Bundle digest:** %s\n", frozen, bundleDigest)
	fmt.Fprintf(&b, "- **Plan revision:** %d · **Architecture revision:** %d\n",
		e.State.Project.Revisions.Plan, e.State.Project.Revisions.Architecture)
	fmt.Fprintf(&b, "- **Recorded at:** %s\n- **Decision:** %s\n\n", Timestamp(e.Now()), decision)
	b.WriteString("## Items and authorship\n\n| Item | Author lineage | Primary reviewer |\n|---|---|---|\n")
	for item, lineage := range authors {
		primary := "the reviewer from the other family"
		for _, run := range runs {
			if run.lineage != lineage && run.lineage != provider.LineageUnknown {
				primary = run.recordID + " (" + run.lineage + ")"
				break
			}
		}
		fmt.Fprintf(&b, "| %s | %s | %s |\n", item, lineage, primary)
	}
	b.WriteString("\n## Reviewer passes\n\n| Review | Reviewer | Lineage | Independence | Decision | Findings |\n|---|---|---|---|---|---:|\n")
	for _, run := range runs {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %d |\n",
			run.recordID, run.reviewerName, run.lineage, run.independence, run.answer.Decision, len(run.answer.Findings))
	}
	b.WriteString("\n## Merged findings\n\n")
	if len(merged) == 0 {
		b.WriteString("None.\n")
	}
	for _, finding := range merged {
		fmt.Fprintf(&b, "- [%s] %s at %s — %s\n", finding.Severity, finding.Title, finding.Location, finding.Requirement)
	}
	return fsx.WriteFileAtomic(filepath.Join(dir, id+".md"), []byte(b.String()), 0o644)
}

// approvePhase closes the reviewed phase: tasks accepted, statuses approved, and the
// review-to-approve transition recorded.
func (e *Engine) approvePhase(frozen, phaseReviewID string) error {
	for index := range e.State.Plan.Phases {
		phase := &e.State.Plan.Phases[index]
		if phase.Status != "active" {
			continue
		}
		phase.Status = "approved"
		for stageIndex := range phase.Stages {
			stage := &phase.Stages[stageIndex]
			stage.Status = "approved"
			for taskIndex := range stage.Tasks {
				if stage.Tasks[taskIndex].Status == "validated" {
					stage.Tasks[taskIndex].Status = "accepted"
				}
			}
		}
	}
	e.State.Project.Workflow.Task = nil
	if err := e.SavePlan(); err != nil {
		return err
	}
	if err := e.AppendTransition(Approve, map[string][]string{"phase_reviews": {phaseReviewID}},
		"Phase approved by both reviewers; see "+phaseReviewID+".", &frozen); err != nil {
		return err
	}
	fmt.Fprintf(e.Out, "The phase is approved. The reviewed work sits on %s at %s.\n",
		e.Repo.IntegrationBranch(e.State.Project.Revisions.Plan), frozen[:12])
	fmt.Fprintln(e.Out, "Next: merge the integration branch when you are ready; the next phase starts when you ask for it.")
	return nil
}

// repairCycle runs the one allowed repair: a different-family-verified bounded fix.
func (e *Engine) repairCycle(ctx context.Context, frozen, phaseReviewID string,
	blocking []Finding, authors map[string]string) error {
	if err := e.AppendTransition(Repair, map[string][]string{"phase_reviews": {phaseReviewID}},
		"Both reviewers asked for a repair; one repair cycle begins.", &frozen); err != nil {
		return err
	}
	repairSeatSetting := e.State.Project.Settings.Roles["repair_producer"]
	if repairSeatSetting.Tool != e.Producer.Name() {
		return errs.New(errs.Configuration,
			"Core Alpha supports codex-cli repair production only", "adjust the repair_producer seat")
	}
	if repairSeatSetting.Effort == "low" {
		return errs.New(errs.Policy, "a repair never runs at low effort", "raise the repair_producer seat's effort")
	}
	repairerLineage := e.Producer.LineageRoot(repairSeatSetting.Model)
	if repairerLineage == provider.LineageUnknown {
		// Unknown lineage fails closed: different-family verification cannot be
		// proven for a repairer whose family is not established.
		return errs.New(errs.Configuration,
			"the repair producer's model lineage is unknown, so different-family verification cannot be proven",
			"record an explicit model for the repair_producer seat, then run peerfoil resume")
	}

	// The repair works on a fresh worktree at the frozen revision.
	lastTask := e.lastValidatedTaskID()
	worktree, err := e.Repo.CreateTaskWorktree(ctx, frozen, lastTask, e.State.Project.Revisions.Plan+1, e.WorktreeParent)
	if err != nil {
		return err
	}
	changeSetID, err := e.nextEvidenceDirID("cs")
	if err != nil {
		return err
	}
	model := repairSeatSetting.Model
	effort := repairSeatSetting.Effort
	if _, err := e.Journal.Intent("repair-producer", changeSetID, map[string]string{
		"worktree": worktree.Path, "branch": worktree.Branch}); err != nil {
		return err
	}

	var packet strings.Builder
	packet.WriteString("You are the PeerFoil repair producer. The current directory is a dedicated worktree at the reviewed revision. Apply the smallest change that resolves the agreed findings below. Edit only what the findings require; never edit .peerfoil/, Git history, or AGENTS.md. Do not commit.\n\nAgreed findings:\n")
	for _, finding := range blocking {
		fmt.Fprintf(&packet, "- [%s] %s at %s: %s. Recommended: %s\n",
			finding.Severity, finding.Title, finding.Location, finding.Requirement, finding.Recommendation)
	}
	if len(blocking) == 0 {
		packet.WriteString("- Resolve the repair findings recorded in .peerfoil/reviews/" + phaseReviewID + ".md\n")
	}
	packet.WriteString("\nEnd with one short paragraph describing exactly what you changed.\n")

	if _, err := e.Journal.Started("repair-producer", changeSetID); err != nil {
		return err
	}
	fmt.Fprintf(e.Out, "Repairing with %s at %s.\n", e.Producer.Name(), repairSeatSetting.Effort)
	outcome, invokeErr := e.Producer.Invoke(ctx, provider.Invocation{
		Role:    "repair_producer",
		Packet:  packet.String(),
		Sandbox: provider.SandboxWorkspaceWrite,
		Dir:     worktree.Path,
	})
	if invokeErr != nil {
		_, _ = e.Journal.Result("repair-producer", changeSetID, "failed", nil)
		reason := "the repair run ended without a result; the worktree " + worktree.Branch + " is retained"
		if err := e.Pause(reason, map[string][]string{"phase_reviews": {phaseReviewID}}); err != nil {
			return err
		}
		return invokeErr
	}

	changes, err := worktree.Changes(ctx)
	if err != nil {
		return err
	}
	if len(changes) == 0 {
		_, _ = e.Journal.Result("repair-producer", changeSetID, "failed", nil)
		reason := "the repair run changed nothing; you decide how to proceed"
		if err := e.Pause(reason, map[string][]string{"phase_reviews": {phaseReviewID}}); err != nil {
			return err
		}
		return errs.New(errs.Provider, reason, "")
	}
	// The repair may touch any path the user already approved for this phase's
	// tasks, and nothing else. This is the write grant the user accepted when they
	// approved the plan; .peerfoil, .git, and AGENTS.md stay denied regardless.
	allowed := e.unionAllowedPaths()
	if err := workspace.CheckScope(changes, allowed); err != nil {
		_, _ = e.Journal.Result("repair-producer", changeSetID, "failed", nil)
		return err
	}
	if err := rejectSymlinks(worktree.Path, changes); err != nil {
		_, _ = e.Journal.Result("repair-producer", changeSetID, "failed", nil)
		return err
	}
	patch, err := worktree.Patch(ctx)
	if err != nil {
		return err
	}
	if hits := workspace.ScanSecrets(patch); len(hits) > 0 {
		_, _ = e.Journal.Result("repair-producer", changeSetID, "failed", nil)
		return errs.New(errs.Policy, "the repair may contain a credential: "+strings.Join(hits, "; "), "")
	}
	patchPath := filepath.Join(e.State.Dir, "evidence", changeSetID+".patch")
	if err := fsx.WriteFileAtomic(patchPath, patch, 0o644); err != nil {
		return err
	}
	digest := sha256.Sum256(patch)
	session := outcome.SessionID
	repairAuthor := project.Actor{
		Role: "repair_producer", Tool: e.Producer.Name(), Model: &model,
		Effort: &effort, LineageRoot: repairerLineage,
	}
	if session != "" {
		repairAuthor.Session = &session
	}
	changeSet := &ChangeSet{
		ID: changeSetID, TaskID: lastTask, TaskTitle: "Repair of the phase's agreed findings",
		CaptureStatus: "captured", StartedAt: Timestamp(e.Now()), Attempt: 1,
		Baseline: "frozen review revision", BaseRevision: frozen,
		Author: repairAuthor, CapturedAt: Timestamp(e.Now()),
		PatchPath: "evidence/" + changeSetID + ".patch", PatchSHA256: hex.EncodeToString(digest[:]),
		Summary: firstParagraph(outcome.Text),
	}
	for _, change := range changes {
		changeSet.Changed = append(changeSet.Changed, change.Status+"\t"+change.Path)
	}
	if err := e.WriteChangeSet(changeSet); err != nil {
		return err
	}
	if _, err := e.Journal.Result("repair-producer", changeSetID, "succeeded", nil); err != nil {
		return err
	}

	repairCommit, err := worktree.Commit(ctx, "PeerFoil repair: agreed phase findings")
	if err != nil {
		return err
	}
	changeSet.SourceRev = repairCommit
	if err := e.WriteChangeSet(changeSet); err != nil {
		return err
	}
	if err := e.AppendTransition(Validate, map[string][]string{"change_sets": {changeSetID}},
		"Repair captured as "+changeSetID+"; affected checks run again.", &repairCommit); err != nil {
		return err
	}

	// Affected checks run again against the repaired revision.
	var failed []string
	for _, task := range e.State.Plan.Tasks() {
		if task.Status != "accepted" && task.Status != "validated" {
			continue
		}
		_, taskFailed, err := e.runTaskEvidence(ctx, worktree, task, changeSetID, repairCommit)
		if err != nil {
			return err
		}
		failed = append(failed, taskFailed...)
	}
	if len(failed) > 0 {
		reason := "the repair did not clear the required checks: " + strings.Join(failed, ", ")
		if err := e.Pause(reason, map[string][]string{"phase_reviews": {phaseReviewID}}); err != nil {
			return err
		}
		return errs.New(errs.Evidence, reason, "the worktree "+worktree.Branch+" is retained; you decide how to proceed")
	}
	if err := e.AppendTransition(Review, map[string][]string{"change_sets": {changeSetID}},
		"Repair checks pass; fresh different-family verification runs next.", &repairCommit); err != nil {
		return err
	}

	// The verifier is a reviewer whose family is known and differs from the
	// repairer's; an unknown lineage never counts as a different family.
	var verifier provider.Adapter
	var verifierSeat provider.Seat
	for index, reviewer := range e.Reviewers {
		lineage := reviewer.LineageRoot(e.ReviewerSeats[index].Model)
		if lineage != provider.LineageUnknown && lineage != repairerLineage {
			verifier = reviewer
			verifierSeat = e.ReviewerSeats[index]
			break
		}
	}
	if verifier == nil {
		return errs.New(errs.Configuration, "no reviewer from a different family than the repairer is available",
			"install and sign in to the other model family, then run peerfoil resume")
	}
	repairDigest := ""
	if tree, err := e.Repo.TreeDigest(ctx, repairCommit); err == nil {
		sum := sha256.Sum256(tree)
		repairDigest = hex.EncodeToString(sum[:])
	}
	verifyPacket := e.buildReviewPacket("repair", repairCommit, repairDigest, blocking)
	verifyRun, err := e.runReviewer(ctx, verifier, verifierSeat, "repair", verifyPacket, worktree.Path, repairCommit, authors, repairDigest, changeSetID)
	if err != nil {
		return err
	}
	fmt.Fprintf(e.Out, "Repair verification by %s: %s.\n", verifyRun.reviewerName, verifyRun.answer.Decision)
	// Approval needs an independent verification, not just an approve word: the
	// verifier's recorded lineage must differ from every material author's.
	if verifyRun.answer.Decision != "approve" || verifyRun.independence != "independent" {
		reason := "the repair did not receive an independent approval; you decide how to proceed"
		if err := e.Pause(reason, map[string][]string{"phase_reviews": {phaseReviewID}, "change_sets": {changeSetID}}); err != nil {
			return err
		}
		return errs.New(errs.Policy, reason,
			"read .peerfoil/reviews/"+verifyRun.recordID+".md and continue with the PeerFoil skills")
	}

	integrationTip, err := e.Repo.Integrate(ctx, worktree, repairCommit)
	if err != nil {
		return err
	}
	if err := e.Repo.CleanupWorktree(ctx, worktree); err != nil {
		fmt.Fprintf(e.Out, "Note: the repair worktree was not removed: %v\n", err)
	}
	return e.approvePhase(integrationTip, phaseReviewID)
}

func (e *Engine) lastValidatedTaskID() string {
	last := "tk-001"
	for _, task := range e.State.Plan.Tasks() {
		if task.Status == "validated" || task.Status == "accepted" {
			last = task.ID
		}
	}
	return last
}

func (e *Engine) unionAllowedPaths() []string {
	seen := map[string]bool{}
	var union []string
	for _, task := range e.State.Plan.Tasks() {
		for _, pattern := range task.AllowedPaths {
			if !seen[pattern] {
				seen[pattern] = true
				union = append(union, pattern)
			}
		}
	}
	return union
}

func firstParagraph(text string) string {
	trimmed := strings.TrimSpace(text)
	if at := strings.Index(trimmed, "\n\n"); at > 0 {
		trimmed = trimmed[:at]
	}
	if len(trimmed) > 900 {
		trimmed = trimmed[:900]
	}
	if trimmed == "" {
		return "Repair applied; see the captured patch."
	}
	return trimmed
}

func orDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}

func orNone(value string) string {
	if strings.TrimSpace(value) == "" {
		return "None."
	}
	return value
}
