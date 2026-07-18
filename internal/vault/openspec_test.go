package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenSpecArtifactsAreProjectedFromNestedRepositoryPath(t *testing.T) {
	repository := openSpecTestRepository(t)
	writeOpenSpecTestFile(t, repository, "docs/openspec/specs/vault-management/spec.md", `# vault-management Specification

## Purpose
Manage portable Markdown vaults.

## Requirements

### Requirement: Stable records
gnosis SHALL keep stable records.

#### Scenario: Read a record
- **WHEN** a record exists
- **THEN** gnosis reads it
`)
	writeOpenSpecTestFile(t, repository, "docs/openspec/changes/add-atlas/proposal.md", `## Why

Atlas compatibility preserves a unique quasar marker.

## What Changes

- Add the atlas.
`)
	writeOpenSpecTestFile(t, repository, "docs/openspec/changes/add-atlas/design.md", "## Context\n\nDesign the atlas.\n")
	writeOpenSpecTestFile(t, repository, "docs/openspec/changes/add-atlas/tasks.md", "## 1. Atlas\n\n- [ ] 1.1 Build the atlas\n")
	writeOpenSpecTestFile(t, repository, "docs/openspec/changes/add-atlas/specs/atlas/spec.md", `## ADDED Requirements

### Requirement: Atlas exists
The system SHALL expose an atlas.

#### Scenario: Read the atlas
- **WHEN** the atlas exists
- **THEN** it is readable
`)
	nested := filepath.Join(repository, "docs", "openspec", "changes", "add-atlas")

	source, err := NewSearchSource(nested)
	if err != nil {
		t.Fatal(err)
	}
	documents, err := source.Documents()
	if err != nil {
		t.Fatal(err)
	}
	byURI := make(map[string]Document, len(documents))
	for _, document := range documents {
		byURI[document.URI] = document
	}
	for uri, wantType := range map[string]string{
		"gnosis://local/openspec/specs/vault-management/spec.md":        openSpecSpecType,
		"gnosis://local/openspec/changes/add-atlas/proposal.md":         openSpecProposalType,
		"gnosis://local/openspec/changes/add-atlas/design.md":           openSpecDesignType,
		"gnosis://local/openspec/changes/add-atlas/tasks.md":            openSpecTasksType,
		"gnosis://local/openspec/changes/add-atlas/specs/atlas/spec.md": openSpecSpecType,
	} {
		document, exists := byURI[uri]
		if !exists {
			t.Fatalf("missing %s in %+v", uri, documents)
		}
		if document.Type != wantType || document.Origin.Root != filepath.Join(repository, "docs") || document.Revision == "" {
			t.Fatalf("document %s = %+v", uri, document)
		}
		if strings.Join(document.Tags, ",") == "" {
			t.Fatalf("document %s has no projected tags", uri)
		}
	}

	page, err := ReadPage(nested, "gnosis://local/openspec/changes/add-atlas/proposal.md")
	if err != nil {
		t.Fatal(err)
	}
	if page.Document.Type != openSpecProposalType ||
		page.Document.Title != "Add Atlas Proposal" ||
		!strings.HasPrefix(page.Markdown, "## Why") {
		t.Fatalf("page = %+v", page)
	}

	records, err := ConceptRecords(nested, openSpecProposalType)
	if err != nil {
		t.Fatal(err)
	}
	if len(records["concepts"]) != 1 ||
		records["concepts"][0]["type"] != openSpecProposalType ||
		records["concepts"][0]["uri"] != "gnosis://local/openspec/changes/add-atlas/proposal.md" {
		t.Fatalf("records = %+v", records)
	}

	hits := New(documents).Search("unique quasar marker", 1)
	if len(hits) != 1 || hits[0].Document.URI != "gnosis://local/openspec/changes/add-atlas/proposal.md" {
		t.Fatalf("hits = %+v", hits)
	}

	result, err := Validate(nested)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 || len(result.Warnings) != 0 || result.FilesChecked != 5 {
		t.Fatalf("validation = %+v", result)
	}
}

func TestOpenSpecArtifactMutationIsRejected(t *testing.T) {
	repository := openSpecTestRepository(t)
	nested := filepath.Join(repository, "docs", "openspec")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("## Why\n\nThis is native OpenSpec Markdown without YAML frontmatter.\n")
	target := "gnosis://local/openspec/changes/add-atlas/proposal.md"

	_, err := WriteDocument(nested, target, content, false)
	if err == nil || !strings.Contains(err.Error(), "OpenSpec artifacts are read-only through gnosis") {
		t.Fatalf("WriteDocument error = %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(repository, "docs", "openspec", "changes", "add-atlas", "proposal.md")); !os.IsNotExist(statErr) {
		t.Fatalf("proposal stat error = %v, want not written", statErr)
	}
}

func TestFrontmatterFreeMarkdownOutsideOpenSpecArtifactsRemainsInvalid(t *testing.T) {
	repository := openSpecTestRepository(t)
	writeOpenSpecTestFile(t, repository, "docs/notes/plain.md", "# Plain\n\nNo frontmatter.\n")

	result, err := Validate(filepath.Join(repository, "docs"))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 1 || !strings.Contains(result.Errors[0], "missing YAML frontmatter") {
		t.Fatalf("validation = %+v", result)
	}
}

func openSpecTestRepository(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	repository := t.TempDir()
	writeOpenSpecTestFile(t, repository, ".git/HEAD", "ref: refs/heads/main\n")
	if err := os.MkdirAll(filepath.Join(repository, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	return repository
}

func writeOpenSpecTestFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
