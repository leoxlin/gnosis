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
	if fileExists(filepath.Join(root, "concepts", "procedure.md")) {
		t.Fatal("optional concept files should not be created by default")
	}
	discovery, err := DiscoverProcesses(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	foundVaultProcedure := false
	for _, procedure := range discovery["procedures"] {
		if procedure["uri"] == "gnosis://core/procedures/vault/query-vault.md" {
			foundVaultProcedure = true
		}
	}
	if !foundVaultProcedure {
		t.Fatalf("scaffolded vault discovery = %+v, want bundled vault procedures", discovery)
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

func TestScaffoldRejectsNoncanonicalVaultNameBeforeWriting(t *testing.T) {
	root := t.TempDir()
	_, err := Scaffold(root, ScaffoldOptions{Name: "bad name"})
	if err == nil || !strings.Contains(err.Error(), "canonical gnosis URI authority") {
		t.Fatalf("scaffold error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "gnosis.toml")); !os.IsNotExist(statErr) {
		t.Fatalf("gnosis.toml error = %v, want not written", statErr)
	}
}

func TestScaffoldKeepsExistingConfigInSpacedDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "My Vault")
	writeConfig(t, root, `[vault]
vault_name = "valid-name"
vault_root = "."
vault_index = false
vault_log = false
`)

	if _, err := Scaffold(root, ScaffoldOptions{}); err != nil {
		t.Fatal(err)
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
	write(t, root, "concepts/topic.md", `---
type: ConceptType
title: Topic
description: Definition of a reusable topic record.
---

# Topic
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
	if strings.Contains(rootText, "topic.md") {
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
	if !strings.Contains(conceptsText, "[Topic](topic.md) - Definition of a reusable topic record.") {
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

func TestGenerateWorkspaceIndexesUsesEffectiveLocalPolicy(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "workspace"
vault_root = "docs"
vault_index = true
vault_log = false
`)
	write(t, root, "docs/note.md", "---\ntype: Note\ntitle: Note\n---\n")

	written, enabled, err := GenerateWorkspaceIndexes(root, IndexOptions{Overwrite: true})
	if err != nil {
		t.Fatal(err)
	}
	if !enabled || len(written) != 1 || written[0] != filepath.Join(root, "docs", "index.md") {
		t.Fatalf("written = %v, enabled = %t", written, enabled)
	}
	if fileExists(filepath.Join(root, "index.md")) {
		t.Fatal("workspace index escaped the configured local vault root")
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
