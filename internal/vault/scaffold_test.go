package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldCreatesBaseVaultWithoutOptionalConcepts(t *testing.T) {
	root := t.TempDir()

	if _, err := Scaffold(root, ScaffoldOptions{}); err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{"index.md", "log.md", "AGENTS.md", "concepts", "references"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s to exist: %v", rel, err)
		}
	}
	if fileExists(filepath.Join(root, "concepts", "repository-purpose.md")) {
		t.Fatal("optional concept files should not be created by default")
	}

	result, err := Validate(root)
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

func TestScaffoldCanIncludeReusableConcepts(t *testing.T) {
	root := t.TempDir()

	if _, err := Scaffold(root, ScaffoldOptions{IncludeConcepts: true}); err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{
		"concepts/index.md",
		"concepts/repository-purpose.md",
		"concepts/repository-decision.md",
		"concepts/repository-directive.md",
		"concepts/repository-delta.md",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s to exist: %v", rel, err)
		}
	}

	body, err := os.ReadFile(filepath.Join(root, "concepts", "repository-purpose.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{
		"type: Concept Type",
		"project, repository, service, or major component",
		"## Minimum record",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("repository purpose template missing %q", want)
		}
	}

	result, err := Validate(root)
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

func TestScaffoldPreservesExistingConceptFilesUnlessForced(t *testing.T) {
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
	write(t, root, rel, existing)

	if _, err := Scaffold(root, ScaffoldOptions{IncludeConcepts: true}); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, filepath.Join(root, rel), existing)

	if _, err := Scaffold(root, ScaffoldOptions{Force: true, IncludeConcepts: true}); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	if string(updated) == existing {
		t.Fatal("expected force scaffold to replace existing concept file")
	}
	if !strings.Contains(string(updated), "title: Repository Purpose") {
		t.Fatal("expected forced concept file to use scaffold template")
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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
