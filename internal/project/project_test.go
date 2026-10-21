// This file is part of PeerFoil.
// internal/project/project_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Tests project loading against the repository fixtures and seeded invalid
// states: malformed JSON, stale plans, duplicate identifiers, missing dependencies,
// cycles, broken history chains, and task selection order.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabrielmongefranco/peerfoil/internal/schema"
)

func schemas(t *testing.T) *schema.Set {
	t.Helper()
	set, err := schema.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	return set
}

func fixtureRoot(t *testing.T, name string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "..", "..", "fixtures", name)
}

// copyFixture copies a fixture into a temporary directory so a test can damage it.
func copyFixture(t *testing.T, name string) string {
	t.Helper()
	source := fixtureRoot(t, name)
	target := t.TempDir()
	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if info.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func mutateJSON(t *testing.T, path string, mutate func(map[string]any)) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	mutate(doc)
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFixturesLoad(t *testing.T) {
	set := schemas(t)
	for _, name := range []string{"software-go", "software-node", "documentation"} {
		state, err := Load(fixtureRoot(t, name), set)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if state.Project.Workflow.State != "produce" {
			t.Errorf("%s: state %q, want produce", name, state.Project.Workflow.State)
		}
		if state.Plan == nil || state.Plan.PlanRevision != 2 {
			t.Errorf("%s: plan revision not 2", name)
		}
		if len(state.History) != 5 {
			t.Errorf("%s: %d history lines, want 5", name, len(state.History))
		}
	}
}

func TestMissingPeerfoilDirectory(t *testing.T) {
	_, err := Load(t.TempDir(), schemas(t))
	if err == nil || !strings.Contains(err.Error(), ".peerfoil") {
		t.Errorf("err = %v", err)
	}
}

func TestMalformedProjectJSON(t *testing.T) {
	root := copyFixture(t, "software-go")
	path := filepath.Join(root, ".peerfoil", "project.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root, schemas(t)); err == nil || !strings.Contains(err.Error(), "not valid JSON") {
		t.Errorf("err = %v", err)
	}
}

func TestUnsupportedSchemaVersion(t *testing.T) {
	root := copyFixture(t, "software-go")
	mutateJSON(t, filepath.Join(root, ".peerfoil", "project.json"), func(doc map[string]any) {
		doc["schema_version"] = 99
	})
	if _, err := Load(root, schemas(t)); err == nil || !strings.Contains(err.Error(), "expected constant") {
		t.Errorf("err = %v", err)
	}
}

func TestStalePlanRevision(t *testing.T) {
	root := copyFixture(t, "software-go")
	mutateJSON(t, filepath.Join(root, ".peerfoil", "project.json"), func(doc map[string]any) {
		doc["revisions"].(map[string]any)["plan"] = 3.0
	})
	if _, err := Load(root, schemas(t)); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Errorf("err = %v", err)
	}
}

func TestDuplicateAndMissingDependency(t *testing.T) {
	root := copyFixture(t, "software-go")
	mutateJSON(t, filepath.Join(root, ".peerfoil", "plan.json"), func(doc map[string]any) {
		phases := doc["phases"].([]any)
		stages := phases[0].(map[string]any)["stages"].([]any)
		tasks := stages[0].(map[string]any)["tasks"].([]any)
		first := tasks[0].(map[string]any)
		second := tasks[1].(map[string]any)
		second["id"] = first["id"]            // duplicate identifier
		first["depends_on"] = []any{"tk-999"} // missing dependency
	})
	_, err := Load(root, schemas(t))
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, fragment := range []string{"duplicate identifier", "missing task tk-999"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Errorf("error %q does not mention %q", err.Error(), fragment)
		}
	}
}

func TestDependencyCycle(t *testing.T) {
	root := copyFixture(t, "software-go")
	mutateJSON(t, filepath.Join(root, ".peerfoil", "plan.json"), func(doc map[string]any) {
		phases := doc["phases"].([]any)
		stages := phases[0].(map[string]any)["stages"].([]any)
		tasks := stages[0].(map[string]any)["tasks"].([]any)
		tasks[0].(map[string]any)["depends_on"] = []any{"tk-002"}
		tasks[1].(map[string]any)["depends_on"] = []any{"tk-001"}
	})
	if _, err := Load(root, schemas(t)); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Errorf("err = %v", err)
	}
}

func TestBrokenHistoryChain(t *testing.T) {
	root := copyFixture(t, "software-go")
	path := filepath.Join(root, ".peerfoil", "history.jsonl")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	// Dropping a middle transition breaks the from_state chain.
	damaged := append([]string{}, lines[0])
	damaged = append(damaged, lines[2:]...)
	if err := os.WriteFile(path, []byte(strings.Join(damaged, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root, schemas(t)); err == nil || !strings.Contains(err.Error(), "from_state") {
		t.Errorf("err = %v", err)
	}
}

func TestWorkflowStateMustMatchHistory(t *testing.T) {
	root := copyFixture(t, "software-go")
	mutateJSON(t, filepath.Join(root, ".peerfoil", "project.json"), func(doc map[string]any) {
		doc["workflow"].(map[string]any)["state"] = "approve"
	})
	if _, err := Load(root, schemas(t)); err == nil || !strings.Contains(err.Error(), "last recorded transition") {
		t.Errorf("err = %v", err)
	}
}

func TestNextTaskOrderAndDependencies(t *testing.T) {
	set := schemas(t)
	state, err := Load(fixtureRoot(t, "software-go"), set)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := state.NextTask()
	if err != nil {
		t.Fatal(err)
	}
	if selection == nil || selection.Task.ID != "tk-001" {
		t.Fatalf("selected %+v, want tk-001", selection)
	}

	// With tk-001 validated, tk-002 becomes eligible.
	selection.Task.Status = "validated"
	next, err := state.NextTask()
	if err != nil {
		t.Fatal(err)
	}
	if next == nil || next.Task.ID != "tk-002" {
		t.Fatalf("selected %+v, want tk-002", next)
	}

	// An in-flight task is reported for continuation.
	next.Task.Status = "in_progress"
	open := state.InFlight()
	if len(open) != 1 || open[0].ID != "tk-002" {
		t.Fatalf("in-flight %+v, want tk-002", open)
	}
}

func TestStageDependencyCycle(t *testing.T) {
	root := copyFixture(t, "software-go")
	mutateJSON(t, filepath.Join(root, ".peerfoil", "plan.json"), func(doc map[string]any) {
		stages := doc["phases"].([]any)[0].(map[string]any)["stages"].([]any)
		stages[0].(map[string]any)["depends_on"] = []any{"st-01"}
	})
	if _, err := Load(root, schemas(t)); err == nil || !strings.Contains(err.Error(), "stage dependencies form a cycle") {
		t.Errorf("err = %v", err)
	}
}

func TestMultipleActivePhasesRejected(t *testing.T) {
	root := copyFixture(t, "software-go")
	mutateJSON(t, filepath.Join(root, ".peerfoil", "plan.json"), func(doc map[string]any) {
		phases := doc["phases"].([]any)
		first := phases[0].(map[string]any)
		second := map[string]any{
			"id": "ph-02", "title": "Second", "outcome": "Another outcome.",
			"order": 2.0, "status": "active", "stages": []any{},
		}
		doc["phases"] = []any{first, second}
	})
	if _, err := Load(root, schemas(t)); err == nil || !strings.Contains(err.Error(), "phases are active at once") {
		t.Errorf("err = %v", err)
	}
}

func TestRunnableTaskFromOlderRevisionNeedsRetention(t *testing.T) {
	root := copyFixture(t, "software-go")
	// tk-001 drops to revision 1 and the recorded change no longer retains it.
	mutateJSON(t, filepath.Join(root, ".peerfoil", "plan.json"), func(doc map[string]any) {
		tasks := doc["phases"].([]any)[0].(map[string]any)["stages"].([]any)[0].(map[string]any)["tasks"].([]any)
		tasks[0].(map[string]any)["plan_revision"] = 1.0
		changes := doc["changes"].([]any)
		changes[0].(map[string]any)["retained_tasks"] = []any{}
	})
	if _, err := Load(root, schemas(t)); err == nil || !strings.Contains(err.Error(), "without being retained") {
		t.Errorf("err = %v", err)
	}
	// Once the change retains it, the older revision is legitimate carried work.
	mutateJSON(t, filepath.Join(root, ".peerfoil", "plan.json"), func(doc map[string]any) {
		changes := doc["changes"].([]any)
		changes[0].(map[string]any)["retained_tasks"] = []any{"tk-001"}
	})
	if _, err := Load(root, schemas(t)); err != nil {
		t.Errorf("retained task rejected: %v", err)
	}
}

func TestIllegalHistoryEdgeRejected(t *testing.T) {
	root := copyFixture(t, "software-go")
	path := filepath.Join(root, ".peerfoil", "history.jsonl")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	// A chained but illegal edge: the last transition claims define went straight to
	// produce, which the lifecycle never permits.
	last := strings.Replace(lines[len(lines)-1], `"from_state":"produce"`, `"from_state":"define"`, 1)
	if err := os.WriteFile(path, []byte(lines[0]+"\n"+last+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root, schemas(t)); err == nil || !strings.Contains(err.Error(), "permits no transition") {
		t.Errorf("err = %v", err)
	}
}

func TestFirstTransitionMustEnterDefine(t *testing.T) {
	root := copyFixture(t, "software-go")
	path := filepath.Join(root, ".peerfoil", "history.jsonl")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	// Dropping the first line leaves a history that starts mid-lifecycle.
	if err := os.WriteFile(path, []byte(strings.Join(lines[1:], "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root, schemas(t)); err == nil || !strings.Contains(err.Error(), "must enter define") {
		t.Errorf("err = %v", err)
	}
}

func TestRetentionMustChainThroughEveryRevision(t *testing.T) {
	root := copyFixture(t, "software-go")
	planPath := filepath.Join(root, ".peerfoil", "plan.json")
	projectPath := filepath.Join(root, ".peerfoil", "project.json")
	// The plan moves to revision 3, but the only recorded change is the revision-1
	// intake: the revision-2 intake never re-tied the still-planned tasks.
	mutateJSON(t, projectPath, func(doc map[string]any) {
		doc["revisions"].(map[string]any)["plan"] = 3.0
	})
	mutateJSON(t, planPath, func(doc map[string]any) {
		doc["plan_revision"] = 3.0
		tasks := doc["phases"].([]any)[0].(map[string]any)["stages"].([]any)[0].(map[string]any)["tasks"].([]any)
		tasks[0].(map[string]any)["plan_revision"] = 1.0 // retained only by the revision-1 intake
		tasks[1].(map[string]any)["plan_revision"] = 3.0 // current work, no retention needed
	})
	_, err := Load(root, schemas(t))
	if err == nil || !strings.Contains(err.Error(), "without being retained by the change recorded at revision 2") {
		t.Errorf("err = %v", err)
	}
	// A second change covering the revision-2 step completes the chain.
	mutateJSON(t, planPath, func(doc map[string]any) {
		changes := doc["changes"].([]any)
		var second map[string]any
		raw, _ := json.Marshal(changes[0])
		if err := json.Unmarshal(raw, &second); err != nil {
			t.Fatal(err)
		}
		second["id"] = "cr-0002"
		second["prior_revision"] = 2.0
		second["retained_tasks"] = []any{"tk-001"}
		second["affected_tasks"] = []any{}
		doc["changes"] = append(changes, second)
	})
	if _, err := Load(root, schemas(t)); err != nil {
		t.Errorf("a fully chained retention was rejected: %v", err)
	}
}

func TestInheritedRulesPathMustStayInRepository(t *testing.T) {
	root := copyFixture(t, "software-go")
	mutateJSON(t, filepath.Join(root, ".peerfoil", "project.json"), func(doc map[string]any) {
		doc["rules_source"] = map[string]any{"mode": "inherit", "path": "../../secrets/AGENTS.md"}
	})
	if _, err := Load(root, schemas(t)); err == nil || !strings.Contains(err.Error(), "rules_source.path") {
		t.Errorf("err = %v", err)
	}
}
