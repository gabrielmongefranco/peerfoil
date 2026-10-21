// This file is part of PeerFoil.
// internal/schema/schema.go
// Author(s): Gabriel Mongefranco.
// Created: 2026-10-21
// Last Modified: 2026-10-21
// Summary: Validates JSON instances against the PeerFoil record schemas using the
// keyword subset documented in schemas/README.md, with no third-party dependency.
// Copyright © 2026 Gabriel Mongefranco
// SPDX-License-Identifier: GPL-3.0-or-later

// Package schema implements the same JSON Schema subset as tests/static_checks.py, so
// Core and the repository checks accept and reject exactly the same records (D-0032).
// A schema keyword outside the subset is an error, which keeps the subset honest.
package schema

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"regexp"
	"sort"
	"strings"

	peerfoil "github.com/gabrielmongefranco/peerfoil"
)

// supportedKeywords is the documented subset. Annotations are accepted and ignored.
var supportedKeywords = map[string]bool{
	"$schema": true, "$id": true, "$comment": true, "$defs": true, "$ref": true,
	"title": true, "type": true, "const": true, "enum": true, "pattern": true,
	"minLength": true, "maxLength": true, "minimum": true, "maximum": true,
	"properties": true, "required": true, "additionalProperties": true,
	"items": true, "minItems": true, "maxItems": true,
}

// Set holds the loaded schema files by file name, such as "project.schema.json".
type Set struct {
	schemas map[string]map[string]any
	regexps map[string]*regexp.Regexp
}

// Load reads every *.schema.json under the given directory of the file system.
func Load(fsys fs.FS, dir string) (*Set, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("reading schema directory: %w", err)
	}
	set := &Set{schemas: map[string]map[string]any{}, regexps: map[string]*regexp.Regexp{}}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".schema.json") {
			continue
		}
		raw, err := fs.ReadFile(fsys, dir+"/"+entry.Name())
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", entry.Name(), err)
		}
		set.schemas[entry.Name()] = doc
	}
	if len(set.schemas) == 0 {
		return nil, fmt.Errorf("no *.schema.json files under %s", dir)
	}
	for name, doc := range set.schemas {
		if err := auditKeywords(doc, name, "$"); err != nil {
			return nil, err
		}
	}
	return set, nil
}

// LoadEmbedded loads the schema copy shipped inside the binary.
func LoadEmbedded() (*Set, error) {
	return Load(peerfoil.Assets, "schemas")
}

func auditKeywords(node any, file, where string) error {
	switch typed := node.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			if !supportedKeywords[key] {
				return fmt.Errorf("%s: unsupported schema keyword %q at %s", file, key, where)
			}
			if key == "properties" || key == "$defs" {
				children, ok := typed[key].(map[string]any)
				if !ok {
					return fmt.Errorf("%s: %s at %s must be an object", file, key, where)
				}
				childNames := make([]string, 0, len(children))
				for name := range children {
					childNames = append(childNames, name)
				}
				sort.Strings(childNames)
				for _, name := range childNames {
					if err := auditKeywords(children[name], file, where+"."+key+"."+name); err != nil {
						return err
					}
				}
			} else if key == "items" {
				if err := auditKeywords(typed[key], file, where+".items"); err != nil {
					return err
				}
			}
		}
	case []any:
		for index, item := range typed {
			if err := auditKeywords(item, file, fmt.Sprintf("%s[%d]", where, index)); err != nil {
				return err
			}
		}
	}
	return nil
}

// resolve follows a $ref such as "common.schema.json#/$defs/actor".
func (s *Set) resolve(ref, currentFile string) (map[string]any, string, error) {
	filePart, pointer, _ := strings.Cut(ref, "#")
	fileName := filePart
	if fileName == "" {
		fileName = currentFile
	}
	doc, ok := s.schemas[fileName]
	if !ok {
		return nil, "", fmt.Errorf("unknown schema file in $ref %q", ref)
	}
	var node any = doc
	for _, segment := range strings.Split(pointer, "/") {
		if segment == "" {
			continue
		}
		object, ok := node.(map[string]any)
		if !ok {
			return nil, "", fmt.Errorf("unresolvable $ref %q", ref)
		}
		node, ok = object[segment]
		if !ok {
			return nil, "", fmt.Errorf("unresolvable $ref %q", ref)
		}
	}
	target, ok := node.(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("$ref %q does not point to a schema object", ref)
	}
	return target, fileName, nil
}

// Validate checks a decoded JSON instance against the named schema file and returns
// every problem found. A plan instance with schema_version 1 dispatches to the
// retained version 1 schema, matching the repository checks.
func (s *Set) Validate(instance any, schemaFile string) ([]string, error) {
	if schemaFile == "plan.schema.json" {
		if object, ok := instance.(map[string]any); ok {
			if versionOf(object["schema_version"]) == 1 {
				schemaFile = "plan-v1.schema.json"
			}
		}
	}
	root, ok := s.schemas[schemaFile]
	if !ok {
		return nil, fmt.Errorf("unknown schema file %q", schemaFile)
	}
	var problems []string
	if err := s.validate(instance, root, schemaFile, "$", &problems); err != nil {
		return nil, err
	}
	return problems, nil
}

func versionOf(value any) int {
	number, ok := value.(float64)
	if !ok || number != math.Trunc(number) {
		return 0
	}
	return int(number)
}

func (s *Set) validate(value any, schema map[string]any, file, where string, problems *[]string) error {
	if ref, ok := schema["$ref"].(string); ok {
		target, targetFile, err := s.resolve(ref, file)
		if err != nil {
			return err
		}
		return s.validate(value, target, targetFile, where, problems)
	}

	if typeSpec, ok := schema["type"]; ok {
		allowed := typeNames(typeSpec)
		matched := false
		for _, name := range allowed {
			if typeMatches(name, value) {
				matched = true
				break
			}
		}
		if !matched {
			*problems = append(*problems, fmt.Sprintf("%s: expected type %v", where, allowed))
			return nil
		}
	}

	if constant, ok := schema["const"]; ok && !jsonEqual(value, constant) {
		*problems = append(*problems, fmt.Sprintf("%s: expected constant %v", where, constant))
	}
	if enum, ok := schema["enum"].([]any); ok {
		found := false
		for _, candidate := range enum {
			if jsonEqual(value, candidate) {
				found = true
				break
			}
		}
		if !found {
			*problems = append(*problems, fmt.Sprintf("%s: value %v not in %v", where, value, enum))
		}
	}

	switch typed := value.(type) {
	case string:
		if pattern, ok := schema["pattern"].(string); ok {
			matcher, err := s.compiled(pattern)
			if err != nil {
				return fmt.Errorf("%s: invalid pattern %q: %w", file, pattern, err)
			}
			if !matcher.MatchString(typed) {
				*problems = append(*problems, fmt.Sprintf("%s: %q does not match %s", where, typed, pattern))
			}
		}
		runes := len([]rune(typed))
		if minimum, ok := numberKeyword(schema, "minLength"); ok && runes < int(minimum) {
			*problems = append(*problems, fmt.Sprintf("%s: shorter than %d", where, int(minimum)))
		}
		if maximum, ok := numberKeyword(schema, "maxLength"); ok && runes > int(maximum) {
			*problems = append(*problems, fmt.Sprintf("%s: longer than %d", where, int(maximum)))
		}
	case float64:
		if minimum, ok := numberKeyword(schema, "minimum"); ok && typed < minimum {
			*problems = append(*problems, fmt.Sprintf("%s: below minimum %v", where, minimum))
		}
		if maximum, ok := numberKeyword(schema, "maximum"); ok && typed > maximum {
			*problems = append(*problems, fmt.Sprintf("%s: above maximum %v", where, maximum))
		}
	case map[string]any:
		properties, _ := schema["properties"].(map[string]any)
		if required, ok := schema["required"].([]any); ok {
			for _, nameValue := range required {
				name, _ := nameValue.(string)
				if _, present := typed[name]; !present {
					*problems = append(*problems, fmt.Sprintf("%s: missing required field %q", where, name))
				}
			}
		}
		names := make([]string, 0, len(typed))
		for name := range typed {
			names = append(names, name)
		}
		sort.Strings(names)
		additional, hasAdditional := schema["additionalProperties"].(bool)
		for _, name := range names {
			if childSchema, known := properties[name]; known {
				child, ok := childSchema.(map[string]any)
				if !ok {
					return fmt.Errorf("%s: property %q is not a schema object", file, name)
				}
				if err := s.validate(typed[name], child, file, where+"."+name, problems); err != nil {
					return err
				}
			} else if hasAdditional && !additional {
				*problems = append(*problems, fmt.Sprintf("%s: unexpected field %q", where, name))
			}
		}
	case []any:
		if minimum, ok := numberKeyword(schema, "minItems"); ok && len(typed) < int(minimum) {
			*problems = append(*problems, fmt.Sprintf("%s: fewer than %d items", where, int(minimum)))
		}
		if maximum, ok := numberKeyword(schema, "maxItems"); ok && len(typed) > int(maximum) {
			*problems = append(*problems, fmt.Sprintf("%s: more than %d items", where, int(maximum)))
		}
		if itemsSchema, ok := schema["items"].(map[string]any); ok {
			for index, item := range typed {
				if err := s.validate(item, itemsSchema, file, fmt.Sprintf("%s[%d]", where, index), problems); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *Set) compiled(pattern string) (*regexp.Regexp, error) {
	if matcher, ok := s.regexps[pattern]; ok {
		return matcher, nil
	}
	matcher, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	s.regexps[pattern] = matcher
	return matcher, nil
}

func typeNames(spec any) []string {
	switch typed := spec.(type) {
	case string:
		return []string{typed}
	case []any:
		names := make([]string, 0, len(typed))
		for _, item := range typed {
			if name, ok := item.(string); ok {
				names = append(names, name)
			}
		}
		return names
	}
	return nil
}

// typeMatches mirrors the repository checks: encoding/json decodes every number to
// float64, so "integer" means a float64 with no fractional part.
func typeMatches(name string, value any) bool {
	switch name {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "integer":
		number, ok := value.(float64)
		return ok && number == math.Trunc(number)
	case "number":
		_, ok := value.(float64)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "null":
		return value == nil
	}
	return false
}

func jsonEqual(a, b any) bool {
	left, errA := json.Marshal(a)
	right, errB := json.Marshal(b)
	return errA == nil && errB == nil && string(left) == string(right)
}

func numberKeyword(schema map[string]any, key string) (float64, bool) {
	number, ok := schema[key].(float64)
	return number, ok
}
