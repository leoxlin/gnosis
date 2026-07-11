package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldCreatesBaseVaultWithoutOptionalConcepts(t *testing.T) {
	root := t.TempDir()

	created, err := Scaffold(root, ScaffoldOptions{})
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
	unchanged, err := Scaffold(root, ScaffoldOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(unchanged) != 0 {
		t.Fatalf("second scaffold reported unchanged files: %v", unchanged)
	}

	for _, rel := range []string{"index.md", "log.md", "AGENTS.md", "concepts/index.md", "references/index.md"} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Fatalf("expected %s to exist: %v", rel, err)
		}
	}
	if fileExists(filepath.Join(root, "concepts", "gnosis-purpose.md")) {
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

func TestScaffoldCanDisableIndexesAndLog(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_index = false
vault_log = false
`)

	created, err := Scaffold(root, ScaffoldOptions{DisableIndex: true, DisableLog: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range created {
		if filepath.Base(path) == "index.md" || filepath.Base(path) == "log.md" {
			t.Fatalf("disabled navigation file was created: %s", path)
		}
	}
	for _, rel := range []string{"index.md", "log.md", "concepts/index.md", "references/index.md"} {
		if fileExists(filepath.Join(root, rel)) {
			t.Fatalf("disabled navigation file exists: %s", rel)
		}
	}
	if !fileExists(filepath.Join(root, "AGENTS.md")) {
		t.Fatal("expected agent rules to be scaffolded")
	}

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected validation errors: %v", result.Errors)
	}
}

func TestGenerateIndexesWritesFolderIndexes(t *testing.T) {
	root := t.TempDir()
	write(t, root, "log.md", `# Log

## 2026-07-09

* Entry.
`)
	write(t, root, "concepts/gnosis-purpose.md", `---
type: Concept Type
title: Gnosis Purpose
description: Definition of a reusable purpose record.
---

# Gnosis Purpose
`)

	written, err := GenerateIndexes(root, IndexOptions{Overwrite: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 2 {
		t.Fatalf("written = %v, want root and concepts indexes", written)
	}

	rootIndex, err := os.ReadFile(filepath.Join(root, "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	rootText := string(rootIndex)
	if !strings.Contains(rootText, "[Concepts](concepts/index.md)") {
		t.Fatalf("root index missing concepts subindex link:\n%s", rootText)
	}
	if strings.Contains(rootText, "gnosis-purpose.md") {
		t.Fatalf("root index should not list individual pages:\n%s", rootText)
	}

	conceptsIndex, err := os.ReadFile(filepath.Join(root, "concepts", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	conceptsText := string(conceptsIndex)
	if !strings.Contains(conceptsText, "[Parent Index](../index.md)") {
		t.Fatalf("subindex missing parent link:\n%s", conceptsText)
	}
	if !strings.Contains(conceptsText, "[Gnosis Purpose](gnosis-purpose.md) - Definition of a reusable purpose record.") {
		t.Fatalf("subindex missing page metadata:\n%s", conceptsText)
	}
	if strings.HasSuffix(conceptsText, "\n\n") {
		t.Fatalf("subindex has an extra trailing blank line:\n%s", conceptsText)
	}

	unchanged, err := GenerateIndexes(root, IndexOptions{Overwrite: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(unchanged) != 0 {
		t.Fatalf("regenerating identical indexes reported changes: %v", unchanged)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
