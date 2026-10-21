// This file is part of PeerFoil.
// internal/schema/schema_test.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Tests the embedded schema set against the repository templates and
// fixtures, and the validator's rejection of malformed and off-subset input.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "..", "..")
}

func decodeFile(t *testing.T, path string) any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return value
}

func TestEmbeddedSchemasLoad(t *testing.T) {
	set, err := LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"common.schema.json", "project.schema.json",
		"plan.schema.json", "plan-v1.schema.json", "transition.schema.json", "pack.schema.json"} {
		if _, ok := set.schemas[name]; !ok {
			t.Errorf("embedded set is missing %s", name)
		}
	}
}

func TestTemplatesAndFixturesValidate(t *testing.T) {
	set, err := LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	root := repoRoot(t)
	cases := map[string]string{
		filepath.Join(root, "plugins", "peerfoil", "templates", "project.json"): "project.schema.json",
		filepath.Join(root, "plugins", "peerfoil", "templates", "plan.json"):    "plan.schema.json",
	}
	for _, fixture := range []string{"software-go", "software-node", "documentation"} {
		base := filepath.Join(root, "fixtures", fixture, ".peerfoil")
		cases[filepath.Join(base, "project.json")] = "project.schema.json"
		cases[filepath.Join(base, "plan.json")] = "plan.schema.json"
		cases[filepath.Join(base, "plans", "plan-1.json")] = "plan.schema.json"
	}
	for path, schemaFile := range cases {
		problems, err := set.Validate(decodeFile(t, path), schemaFile)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if len(problems) != 0 {
			t.Errorf("%s: unexpected problems %v", path, problems)
		}
	}
}

func TestHistoryLinesValidate(t *testing.T) {
	set, err := LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repoRoot(t), "fixtures", "software-go", ".peerfoil", "history.jsonl")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for number, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var value any
		if err := json.Unmarshal([]byte(line), &value); err != nil {
			t.Fatalf("line %d: %v", number+1, err)
		}
		problems, err := set.Validate(value, "transition.schema.json")
		if err != nil {
			t.Fatal(err)
		}
		if len(problems) != 0 {
			t.Errorf("line %d: %v", number+1, problems)
		}
	}
}

func TestRejectsBadRecords(t *testing.T) {
	set, err := LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name     string
		instance string
		schema   string
		fragment string
	}{
		{"wrong record type", `{"record_type":"nope"}`, "project.schema.json", "expected constant"},
		{"missing required", `{"record_type":"transition"}`, "transition.schema.json", "missing required field"},
		{"bad identifier", `{"record_type":"transition","schema_version":1,"transition_id":"TR-1","project_id":"p","at":"2026-10-21T00:00:00Z","from_state":null,"to_state":"define","actor":{"role":"coordinator","tool":"claude-code","model":"m","effort":"medium","lineage_root":"anthropic-claude","session":null},"plan_revision":0,"source_revision":null,"summary":"x"}`, "transition.schema.json", "does not match"},
		{"unknown field", `{"record_type":"transition","schema_version":1,"transition_id":"tr-0001","project_id":"p","at":"2026-10-21T00:00:00Z","from_state":null,"to_state":"define","actor":{"role":"coordinator","tool":"claude-code","model":"m","effort":"medium","lineage_root":"anthropic-claude","session":null},"plan_revision":0,"source_revision":null,"summary":"x","invented":true}`, "transition.schema.json", "unexpected field"},
	}
	for _, testCase := range cases {
		var value any
		if err := json.Unmarshal([]byte(testCase.instance), &value); err != nil {
			t.Fatal(err)
		}
		problems, err := set.Validate(value, testCase.schema)
		if err != nil {
			t.Fatal(err)
		}
		joined := strings.Join(problems, "; ")
		if !strings.Contains(joined, testCase.fragment) {
			t.Errorf("%s: problems %q do not mention %q", testCase.name, joined, testCase.fragment)
		}
	}
}

func TestPlanVersionDispatch(t *testing.T) {
	set, err := LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	// A version 1 instance must be judged by the retained version 1 schema, which
	// requires no changes array.
	var v1 any
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "schemas", "plan-v1.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	_ = raw
	if err := json.Unmarshal([]byte(`{"schema_version":1}`), &v1); err != nil {
		t.Fatal(err)
	}
	problems, err := set.Validate(v1, "plan.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, problem := range problems {
		if strings.Contains(problem, "schema_version") && strings.Contains(problem, "expected constant") {
			t.Errorf("version 1 plan was judged by the version 2 schema: %v", problems)
		}
	}
}

func TestOffSubsetKeywordFailsLoading(t *testing.T) {
	fsys := fstest.MapFS{
		"schemas/bad.schema.json": &fstest.MapFile{
			Data: []byte(`{"type":"object","oneOf":[{"type":"string"}]}`),
		},
	}
	if _, err := Load(fsys, "schemas"); err == nil ||
		!strings.Contains(err.Error(), "unsupported schema keyword") {
		t.Errorf("expected an unsupported-keyword error, got %v", err)
	}
}
