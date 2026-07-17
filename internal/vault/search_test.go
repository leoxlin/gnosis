package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchSourceLoadsConfiguredRootWithStableURIs(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "Test"
vault_root = "docs"
vault_index = false
vault_log = false
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
	byURI := make(map[string]int)
	for i, document := range documents {
		byURI[document.URI] = i
	}
	for _, uri := range []string{"gnosis://Test/concept.md", "gnosis://Test/related.md"} {
		if _, exists := byURI[uri]; !exists {
			t.Fatalf("missing %s in %+v", uri, documents)
		}
	}
	concept := documents[byURI["gnosis://Test/concept.md"]]
	if concept.Description != "A folded description." {
		t.Fatalf("description = %q", concept.Description)
	}
	if strings.Join(concept.Tags, ",") != "shared,docs" {
		t.Fatalf("tags = %v", concept.Tags)
	}
	if strings.Join(concept.Aliases, ",") != "Primary Idea" {
		t.Fatalf("aliases = %v", concept.Aliases)
	}
	if strings.Join(concept.Links, ",") != "gnosis://Test/related.md" {
		t.Fatalf("links = %v", concept.Links)
	}
	related := documents[byURI["gnosis://Test/related.md"]]
	if related.Description != "Summary fallback." {
		t.Fatalf("summary fallback = %q", related.Description)
	}
	if strings.Join(related.Links, ",") != "gnosis://Test/concept.md" {
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

[[vaults]]
vault_name = "Imported"
vault_root = "imported"
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
	for _, document := range documents {
		if document.URI == "gnosis://Workspace/article.md" && document.Title == "Local" {
			return
		}
	}
	t.Fatalf("documents = %+v", documents)
}

func TestSearchSourceIncludesBundledDocumentsWithVaultPrecedence(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "Workspace"
vault_root = "."
`)
	write(t, root, "concepts/procedure.md", `---
type: ConceptType
title: LocalProcedure
---
`)
	write(t, root, "procedures/using-gnosis.md", `---
type: Procedure
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
	byURI := make(map[string]Document, len(documents))
	for _, document := range documents {
		byURI[document.URI] = document
	}
	if got := byURI["gnosis://Workspace/concepts/procedure.md"].Title; got != "LocalProcedure" {
		t.Fatalf("vault-process title = %q", got)
	}
	if got := byURI["gnosis://Workspace/procedures/using-gnosis.md"].Title; got != "Local using-gnosis" {
		t.Fatalf("using-gnosis title = %q", got)
	}
	if _, exists := byURI["gnosis://core/documentation/basic-usage.md"]; exists {
		t.Fatalf("documents include removed bundled documentation: %+v", documents)
	}
	if _, exists := byURI["gnosis://core/concepts/purpose.md"]; !exists {
		t.Fatalf("documents missing bundled gnosis concepts: %+v", documents)
	}

}

func TestSearchSourceNamesBundledDocumentsCore(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "Workspace"
vault_root = "."
`)

	source, err := NewSearchSource(root)
	if err != nil {
		t.Fatal(err)
	}
	documents, err := source.Documents()
	if err != nil {
		t.Fatal(err)
	}
	for _, document := range documents {
		if document.URI == "gnosis://core/concepts/procedure.md" {
			if document.Origin.Vault != "core" || document.URI != "gnosis://core/concepts/procedure.md" {
				t.Fatalf("bundled document = %+v", document)
			}
			return
		}
	}
	t.Fatal("missing bundled gnosis procedure concept")
}

func TestSearchSourceLetsImportsOverrideBundledDocuments(t *testing.T) {
	workspace := t.TempDir()
	imported := filepath.Join(workspace, "imported")
	if err := os.MkdirAll(imported, 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, workspace, `[[vaults]]
vault_name = "Imported"
vault_root = "imported"
`)
	writeConfig(t, imported, `[vault]
vault_name = "Imported"
vault_root = "."
`)
	write(t, imported, "procedures/query-vault.md", `---
type: Procedure
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
		if document.URI != "gnosis://Imported/procedures/query-vault.md" {
			continue
		}
		if document.Title != "Imported query-vault" {
			t.Fatalf("query-vault = %+v", document)
		}
		return
	}
	t.Fatal("missing query-vault document")
}

func TestSearchSourceAlwaysIncludesConciseBundledDocuments(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "Workspace"
vault_root = "."
`)

	source, err := NewSearchSource(root)
	if err != nil {
		t.Fatal(err)
	}
	documents, err := source.Documents()
	if err != nil {
		t.Fatal(err)
	}
	if len(documents) == 0 {
		t.Fatal("documents = none, want bundled documents")
	}
	byURI := make(map[string]struct{}, len(documents))
	for _, document := range documents {
		byURI[document.URI] = struct{}{}
	}
	for _, uri := range []string{"gnosis://core/concepts/procedure.md", "gnosis://core/concepts/decision.md", "gnosis://core/concepts/directive.md", "gnosis://core/concepts/purpose.md", "gnosis://core/procedures/development/implementing-directive.md", "gnosis://core/procedures/vault/query-vault.md"} {
		if _, exists := byURI[uri]; !exists {
			t.Fatalf("documents missing %s: %+v", uri, documents)
		}
	}
	for _, uri := range []string{"gnosis://core/procedures/executing-plans.md", "gnosis://core/procedures/execution/dispatching-parallel-agents.md", "gnosis://core/procedures/execution/execute-directive.md", "gnosis://core/procedures/execution/finishing-a-development-branch.md", "gnosis://core/procedures/execution/test-driven-development.md", "gnosis://core/procedures/execution/using-git-worktrees.md", "gnosis://core/procedures/execution/verification-before-completion.md", "gnosis://core/procedures/ingest-concept.md", "gnosis://core/procedures/receiving-code-review.md", "gnosis://core/procedures/requesting-code-review.md", "gnosis://core/procedures/review/code-review.md", "gnosis://core/procedures/subagent-driven-development.md", "gnosis://core/procedures/using-gnosis-forge.md", "gnosis://core/procedures/skills/writing-skills.md"} {
		if _, exists := byURI[uri]; exists {
			t.Fatalf("documents include retired process %s: %+v", uri, documents)
		}
	}
	if _, exists := byURI["gnosis://core/documentation/basic-usage.md"]; exists {
		t.Fatalf("documents include removed basic usage page: %+v", documents)
	}
	invocation, err := InvokeProcess(root, "gnosis://core/procedures/development/implementing-directive.md")
	if err != nil {
		t.Fatal(err)
	}
	wantSteps := []string{"selecting-directive", "preparing-workspace", "implementing-tasks", "reviewing-implementation", "verifying-directive", "finishing-directive"}
	if len(invocation.Steps) != len(wantSteps) {
		t.Fatalf("implementing-directive steps = %+v", invocation.Steps)
	}
	for i, want := range wantSteps {
		if invocation.Steps[i].Number != i+1 || invocation.Steps[i].Name != want {
			t.Fatalf("implementing-directive step %d = %+v, want %q", i+1, invocation.Steps[i], want)
		}
	}
	if process := invocation.Steps[0].Sections.Process; !strings.Contains(process, "Bind exactly one directive") || !strings.Contains(process, "Never switch to another directive") && !strings.Contains(process, "never advances automatically") {
		t.Fatalf("implementing-directive selection does not enforce one directive: %s", process)
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
	byURI := make(map[string]Document, len(documents))
	for _, document := range documents {
		byURI[document.URI] = document
	}
	if strings.Join(byURI["gnosis://Test/a.md"].Links, ",") != "gnosis://Test/b.md" {
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
	byURI := make(map[string]Document, len(documents))
	for _, document := range documents {
		byURI[document.URI] = document
	}
	if byURI["gnosis://Test/page.md"].Description != "Before." {
		t.Fatalf("description = %q", byURI["gnosis://Test/page.md"].Description)
	}

	updated := strings.Replace(string(mustReadFile(t, path)), "Before.", "After.", 1)
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	documents, err = source.Documents()
	if err != nil {
		t.Fatal(err)
	}
	byURI = make(map[string]Document, len(documents))
	for _, document := range documents {
		byURI[document.URI] = document
	}
	if byURI["gnosis://Test/page.md"].Description != "After." {
		t.Fatalf("description = %q", byURI["gnosis://Test/page.md"].Description)
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

func TestEffectiveVaultAppliesRecursivePrecedenceAndOrigin(t *testing.T) {
	workspace := t.TempDir()
	first := filepath.Join(workspace, "first")
	nested := filepath.Join(workspace, "nested")
	second := filepath.Join(workspace, "second")
	writeConfig(t, workspace, `[vault]
vault_name = "workspace"
vault_root = "local"

[[vaults]]
vault_name = "first-declaration"
vault_root = "first"

[[vaults]]
vault_name = "second-declaration"
vault_root = "second"
`)
	writeConfig(t, first, `[vault]
vault_name = "first"
vault_root = "."

[[vaults]]
vault_name = "nested-declaration"
vault_root = "../nested"
`)
	writeConfig(t, nested, "[vault]\nvault_name = \"nested\"\nvault_root = \".\"\n")
	writeConfig(t, second, "[vault]\nvault_name = \"second\"\nvault_root = \".\"\n")

	for root, title := range map[string]string{
		filepath.Join(workspace, "local"): "Local",
		first:                             "First",
		nested:                            "Nested",
		second:                            "Second",
	} {
		write(t, root, "collision.md", "---\ntype: Note\ntitle: "+title+"\n---\n")
	}
	write(t, nested, "nested-only.md", "---\ntype: Note\ntitle: Nested winner\n---\n")
	write(t, second, "nested-only.md", "---\ntype: Note\ntitle: Later loser\n---\n")

	source, err := NewSearchSource(workspace)
	if err != nil {
		t.Fatal(err)
	}
	documents, err := source.Documents()
	if err != nil {
		t.Fatal(err)
	}
	byPath := make(map[string]Document, len(documents))
	for _, document := range documents {
		byPath[document.Path] = document
	}
	if got := byPath["collision.md"]; got.Title != "Local" || got.Origin.Kind != OriginLocal || got.Origin.Precedence != 0 {
		t.Fatalf("local collision winner = %+v", got)
	}
	if got := byPath["nested-only.md"]; got.Title != "Nested winner" || got.Origin.Vault != "nested" || got.Origin.Root != nested || got.Origin.Precedence != 2 {
		t.Fatalf("recursive collision winner = %+v", got)
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

func TestSearchSourceExcludesRootDocumentation(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
`)
	write(t, root, "documentation/guide.md", "# Guide\n\nNo frontmatter, no vault links.\n")
	write(t, root, "notes/documentation/thing.md", "---\ntype: Note\ntitle: Thing\n---\n")
	source, err := NewSearchSource(root)
	if err != nil {
		t.Fatal(err)
	}
	documents, err := source.Documents()
	if err != nil {
		t.Fatal(err)
	}
	for _, document := range documents {
		if strings.Contains(document.URI, "documentation/guide.md") {
			t.Fatalf("documentation page loaded: %+v", document)
		}
	}
	found := false
	for _, document := range documents {
		if strings.HasSuffix(document.URI, "notes/documentation/thing.md") {
			found = true
		}
	}
	if !found {
		t.Fatal("nested documentation page was excluded")
	}
}
