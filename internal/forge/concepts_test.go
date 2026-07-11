package forge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gnosis/internal/vault"
)

func TestConceptsWritesReusableConcepts(t *testing.T) {
	root := t.TempDir()
	if _, err := vault.Scaffold(root, vault.ScaffoldOptions{}); err != nil {
		t.Fatal(err)
	}

	created, err := Concepts(root, ConceptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range created {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.IsDir() {
			t.Fatalf("created paths should contain files only: %s", path)
		}
	}
	unchanged, err := Concepts(root, ConceptOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(unchanged) != 0 {
		t.Fatalf("second run reported unchanged files: %v", unchanged)
	}

	for _, rel := range []string{
		"concepts/repository-purpose.md",
		"concepts/repository-decision.md",
		"concepts/repository-directive.md",
		"concepts/repository-process.md",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s to exist: %v", rel, err)
		}
	}

	processBody, err := os.ReadFile(filepath.Join(root, "concepts", "repository-process.md"))
	if err != nil {
		t.Fatal(err)
	}
	processText := string(processBody)
	for _, want := range []string{
		"type: Concept Type",
		"title: Repository Process",
		"repeatable repository-owned workflow",
		"## Knowledge inputs",
		"## Completion",
	} {
		if !strings.Contains(processText, want) {
			t.Fatalf("repository process template missing %q", want)
		}
	}

	directiveBody, err := os.ReadFile(filepath.Join(root, "concepts", "repository-directive.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"selects `writing-plans`", "# Implementation plan"} {
		if !strings.Contains(string(directiveBody), want) {
			t.Fatalf("repository directive template missing %q", want)
		}
	}

	body, err := os.ReadFile(filepath.Join(root, "concepts", "repository-purpose.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{
		"type: Concept Type",
		"single durable statement of why a repository exists",
		"## Minimum record",
		"# Sub-purposes",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("repository purpose template missing %q", want)
		}
	}

	if _, err := vault.GenerateIndexes(root, vault.IndexOptions{Overwrite: true}); err != nil {
		t.Fatal(err)
	}
	result, err := vault.Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected validation errors: %v", result.Errors)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("unexpected validation warnings: %v", result.Warnings)
	}
}

func TestConceptsPreservesExistingFilesUnlessForced(t *testing.T) {
	root := t.TempDir()
	rel := filepath.Join("concepts", "repository-purpose.md")
	existing := `---
type: Concept Type
title: Custom Purpose
description: Local custom concept.
tags: [custom]
timestamp: 2026-07-09T00:00:00Z
---

# Custom Purpose
`
	writeForgeFile(t, root, rel, existing)

	if _, err := Concepts(root, ConceptOptions{}); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, filepath.Join(root, rel), existing)

	if _, err := Concepts(root, ConceptOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	if string(updated) == existing {
		t.Fatal("expected force to replace existing concept file")
	}
	if !strings.Contains(string(updated), "title: Repository Purpose") {
		t.Fatal("expected forced concept file to use concept template")
	}
}

func writeForgeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s content changed unexpectedly", path)
	}
}
