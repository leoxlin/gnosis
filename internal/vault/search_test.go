package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchSourceLoadsConfiguredRootWithStableIDs(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "Test"
vault_root = "docs"
vault_index = false
vault_log = false

[vault.imports]
gnosis_vault = false
`)
	write(t, root, "docs/concept.md", `---
type: Concept
title: Shared Concept
description: >
  A folded
  description.
tags: [shared, docs]
aliases:
  - Primary Idea
---

# Shared Concept

[Related](related.md)
`)
	write(t, root, "docs/related.md", `---
type: Concept
title: Related
summary: Summary fallback.
---

# Related

[Concept](/concept.md#details)
`)
	write(t, root, "docs/index.md", "# Index\n")
	write(t, root, "docs/log.md", "# Log\n")
	write(t, root, "docs/.obsidian/hidden.md", `---
type: Hidden
---
`)

	source, err := NewSearchSource(root)
	if err != nil {
		t.Fatal(err)
	}
	documents, err := source.Documents()
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]int)
	for i, document := range documents {
		byID[document.ID] = i
	}
	for _, id := range []string{"concept.md", "related.md"} {
		if _, exists := byID[id]; !exists {
			t.Fatalf("missing %s in %+v", id, documents)
		}
	}
	if len(documents) != 2 {
		t.Fatalf("documents = %+v, want documents from the configured root", documents)
	}
	concept := documents[byID["concept.md"]]
	if concept.Description != "A folded description." {
		t.Fatalf("description = %q", concept.Description)
	}
	if strings.Join(concept.Tags, ",") != "shared,docs" {
		t.Fatalf("tags = %v", concept.Tags)
	}
	if strings.Join(concept.Aliases, ",") != "Primary Idea" {
		t.Fatalf("aliases = %v", concept.Aliases)
	}
	if strings.Join(concept.Links, ",") != "related.md" {
		t.Fatalf("links = %v", concept.Links)
	}
	related := documents[byID["related.md"]]
	if related.Description != "Summary fallback." {
		t.Fatalf("summary fallback = %q", related.Description)
	}
	if strings.Join(related.Links, ",") != "concept.md" {
		t.Fatalf("absolute link = %v", related.Links)
	}
}

func TestSearchSourcePrefersLocalRootOverImportedVaults(t *testing.T) {
	workspace := t.TempDir()
	imported := filepath.Join(workspace, "imported")
	if err := os.MkdirAll(filepath.Join(workspace, "local"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(imported, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, workspace, `[vault]
vault_name = "Workspace"
vault_root = "local"

[vault.imports]
vaults = ["imported"]
gnosis_vault = false
`)
	writeConfig(t, imported, `[vault]
vault_name = "Imported"
vault_root = "."
`)
	write(t, workspace, "local/article.md", "---\ntype: Note\ntitle: Local\n---\n")
	write(t, imported, "article.md", "---\ntype: Note\ntitle: Imported\n---\n")

	source, err := NewSearchSource(workspace)
	if err != nil {
		t.Fatal(err)
	}
	documents, err := source.Documents()
	if err != nil {
		t.Fatal(err)
	}
	if len(documents) != 1 || documents[0].ID != "article.md" || documents[0].Title != "Local" {
		t.Fatalf("documents = %+v", documents)
	}
}

func TestSearchSourceIncludesBundledDocumentsWithVaultPrecedence(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "Workspace"
vault_root = "."

[vault.imports]
gnosis_forge = true
`)
	write(t, root, "concepts/vault-process.md", `---
type: Concept Type
title: Local Vault Process
---
`)
	write(t, root, "repository/processes/using-gnosis.md", `---
type: Repository Process
title: Local using-gnosis
---
`)

	source, err := NewSearchSource(root)
	if err != nil {
		t.Fatal(err)
	}
	documents, err := source.Documents()
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]Document, len(documents))
	for _, document := range documents {
		byID[document.ID] = document
	}
	if got := byID["concepts/vault-process.md"].Title; got != "Local Vault Process" {
		t.Fatalf("vault-process title = %q", got)
	}
	if got := byID["repository/processes/using-gnosis.md"].Title; got != "Local using-gnosis" {
		t.Fatalf("using-gnosis title = %q", got)
	}
	if _, exists := byID["documentation/basic-usage.md"]; !exists {
		t.Fatalf("documents missing bundled vault documentation: %+v", documents)
	}
	if _, exists := byID["concepts/repository-purpose.md"]; !exists {
		t.Fatalf("documents missing bundled forge documentation: %+v", documents)
	}
	data, err := Read(root, "Vault Process", "query-vault")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "`query-vault` answers") {
		t.Fatalf("read = %q", data)
	}
}

func TestSearchSourceLetsImportsOverrideBundledDocuments(t *testing.T) {
	workspace := t.TempDir()
	imported := filepath.Join(workspace, "imported")
	if err := os.MkdirAll(imported, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, workspace, `[vault.imports]
vaults = ["imported"]
`)
	writeConfig(t, imported, `[vault]
vault_name = "Imported"
vault_root = "."
`)
	write(t, imported, "vault/processes/query-vault.md", `---
type: Vault Process
title: Imported query-vault
---
`)

	source, err := NewSearchSource(workspace)
	if err != nil {
		t.Fatal(err)
	}
	documents, err := source.Documents()
	if err != nil {
		t.Fatal(err)
	}
	for _, document := range documents {
		if document.ID != "vault/processes/query-vault.md" {
			continue
		}
		if document.Title != "Imported query-vault" {
			t.Fatalf("query-vault = %+v", document)
		}
		return
	}
	t.Fatal("missing query-vault document")
}

func TestSearchSourceCanDisableBundledVaultDocuments(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "Workspace"
vault_root = "."

[vault.imports]
gnosis_vault = false
`)

	source, err := NewSearchSource(root)
	if err != nil {
		t.Fatal(err)
	}
	documents, err := source.Documents()
	if err != nil {
		t.Fatal(err)
	}
	if len(documents) != 0 {
		t.Fatalf("documents = %+v, want no bundled documents", documents)
	}
}

func TestSearchSourceResolvesExtensionlessLinksAndIgnoresBrokenLinks(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.md", `---
type: Concept
title: A
---

[B](b)
[Missing](missing.md)
[External](https://example.com/page.md)
`)
	write(t, root, "b.md", `---
type: Concept
title: B
---

# B
`)

	source, err := NewSearchSource(root)
	if err != nil {
		t.Fatal(err)
	}
	documents, err := source.Documents()
	if err != nil {
		t.Fatal(err)
	}
	if len(documents) != 2 || strings.Join(documents[0].Links, ",") != "b.md" {
		t.Fatalf("documents = %+v", documents)
	}
}

func TestSearchSourceReadsLiveFilesOnEveryLoad(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "page.md")
	write(t, root, "page.md", `---
type: Concept
title: Page
description: Before.
---

# Page
`)
	source, err := NewSearchSource(root)
	if err != nil {
		t.Fatal(err)
	}
	documents, err := source.Documents()
	if err != nil {
		t.Fatal(err)
	}
	if documents[0].Description != "Before." {
		t.Fatalf("description = %q", documents[0].Description)
	}

	updated := strings.Replace(string(mustReadFile(t, path)), "Before.", "After.", 1)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	documents, err = source.Documents()
	if err != nil {
		t.Fatal(err)
	}
	if documents[0].Description != "After." {
		t.Fatalf("description = %q", documents[0].Description)
	}
}

func TestSearchSourceRejectsInvalidConceptFrontmatter(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		want    string
	}{
		{name: "missing frontmatter", content: "# Page\n", want: "missing YAML frontmatter"},
		{name: "missing type", content: "---\ntitle: Page\n---\n", want: "missing non-empty \"type\""},
		{name: "invalid tags", content: "---\ntype: Page\ntags:\n  nested: value\n---\n", want: "tags"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			write(t, root, "page.md", test.content)
			source, err := NewSearchSource(root)
			if err != nil {
				t.Fatal(err)
			}
			_, err = source.Documents()
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
