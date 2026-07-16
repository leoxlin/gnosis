package vault

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderDocumentLinksPreservesMarkdownAroundResolvedDestinations(t *testing.T) {
	root := t.TempDir()
	balancedURI := documentURI("Test", "notes/a_(draft).md")
	referenceURI := documentURI("Test", "notes/reference file.md")
	markdown := "[Balanced](notes/a_\\(draft\\).md?view=full#part%2Fchild \"Draft title\")\n" +
		"[Reference][target]\n\n" +
		"[target]: <notes/reference file.md&quest;download=1&#35;intro%2Fchild> 'Reference title'\n\n" +
		"``[Code span](notes/a_(draft).md)``\n\n" +
		"````markdown\n[Fenced](notes/a_(draft).md)\n~~~\n[Still fenced](notes/a_(draft).md)\n````\n"

	page := &effectivePage{
		root: root,
		path: filepath.Join(root, "source.md"),
		data: []byte(markdown),
		document: Document{
			Path: "source.md",
			URI:  documentURI("Test", "source.md"),
			Body: markdown,
		},
	}
	pages := []*effectivePage{
		page,
		{
			root: root,
			path: filepath.Join(root, "notes", "a_(draft).md"),
			document: Document{
				Path: "notes/a_(draft).md",
				URI:  balancedURI,
			},
		},
		{
			root: root,
			path: filepath.Join(root, "notes", "reference file.md"),
			document: Document{
				Path: "notes/reference file.md",
				URI:  referenceURI,
			},
		},
	}

	got, err := renderDocumentLinks(page, pages)
	if err != nil {
		t.Fatal(err)
	}
	want := "[Balanced](" + balancedURI + "?view=full#part%2Fchild \"Draft title\")\n" +
		"[Reference][target]\n\n" +
		"[target]: <" + referenceURI + "?download=1#intro%2Fchild> 'Reference title'\n\n" +
		"``[Code span](notes/a_(draft).md)``\n\n" +
		"````markdown\n[Fenced](notes/a_(draft).md)\n~~~\n[Still fenced](notes/a_(draft).md)\n````\n"
	if got != want {
		t.Fatalf("render = %q, want %q", got, want)
	}
}

func TestRenderDocumentLinksRejectsMalformedGnosisDestination(t *testing.T) {
	root := t.TempDir()
	for _, markdown := range []string{
		"[Target](GNOSIS://test/target.md)\n",
		"<GNOSIS://test/target.md>\n",
	} {
		page := &effectivePage{
			root:   root,
			path:   filepath.Join(root, "source.md"),
			data:   []byte(markdown),
			fields: frontmatterFields{},
			document: Document{
				Path: "source.md",
				URI:  documentURI("test", "source.md"),
				Body: markdown,
			},
		}

		_, err := renderDocumentLinks(page, []*effectivePage{page})
		if err == nil || !strings.Contains(err.Error(), "canonical gnosis URI") {
			t.Fatalf("render %q error = %v", markdown, err)
		}
	}
}

func TestRenderDocumentLinksResolvesImportedLogicalTarget(t *testing.T) {
	workspace := t.TempDir()
	imported := filepath.Join(workspace, "imported")
	writeConfig(t, workspace, `[vault]
vault_name = "workspace"
vault_root = "local"
vault_index = false
vault_log = false

[[vaults]]
vault_name = "declared"
vault_root = "imported"
`)
	writeConfig(t, imported, `[vault]
vault_name = "shared"
vault_root = "."
vault_index = false
vault_log = false
`)
	write(t, workspace, "local/procedures/start.md", `---
type: Note
title: Start
---

[Imported](../shared/end.md)
`)
	write(t, imported, "shared/end.md", "---\ntype: Note\ntitle: End\n---\n")

	page, err := ReadPage(workspace, "gnosis://workspace/procedures/start.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page.Markdown, "[Imported](gnosis://shared/shared/end.md)") {
		t.Fatalf("rendered page = %q", page.Markdown)
	}
}

func TestAnyVaultLinksResolveToConcreteEffectiveURIs(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false
`)
	write(t, root, "body-target.md", "---\ntype: Note\ntitle: Body target\n---\n")
	write(t, root, "relationship-target.md", "---\ntype: Note\ntitle: Relationship target\n---\n")
	write(t, root, "source.md", `---
type: Note
title: Source
relationships:
  - type: uses
    target: gnosis://_/relationship-target.md
---

[Body](gnosis://_/body-target.md?view=full#part)
`)

	page, err := ReadPage(root, "gnosis://_/source.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page.Markdown, "gnosis://test/body-target.md?view=full#part") {
		t.Fatalf("rendered page = %q", page.Markdown)
	}

	neighbors, err := TraceNeighbors(root, "gnosis://_/source.md", DirectionOut, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(neighbors.Edges) != 2 {
		t.Fatalf("edges = %+v", neighbors.Edges)
	}
	wantURIs := map[string]bool{
		"gnosis://test/body-target.md":         true,
		"gnosis://test/relationship-target.md": true,
	}
	for _, edge := range neighbors.Edges {
		if !wantURIs[edge.To.URI] {
			t.Fatalf("unexpected concrete edge: %+v", edge)
		}
		delete(wantURIs, edge.To.URI)
	}
	if len(wantURIs) != 0 {
		t.Fatalf("missing concrete edge URIs: %v", wantURIs)
	}
}

func TestCanonicalLinksShareGraphRenderAndValidationSemantics(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false
`)
	write(t, root, "target.md", "---\ntype: Note\ntitle: Target\ndescription: Target.\n---\n")
	write(t, root, "source.md", `---
type: Note
title: Source
description: Source.
---

[Target](gnosis://test/target.md?view=full#part)
`)
	write(t, root, "relationship.md", `---
type: Note
title: Relationship
description: Relationship.
relationships:
  - type: uses
    target: "gnosis://test/target.md?view=full#part"
---
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("validation errors = %v", result.Errors)
	}
	neighbors, err := TraceNeighbors(root, "gnosis://test/source.md", DirectionOut, []string{"links_to"})
	if err != nil {
		t.Fatal(err)
	}
	if len(neighbors.Edges) != 1 || neighbors.Edges[0].To.URI != "gnosis://test/target.md" {
		t.Fatalf("canonical body edges = %+v", neighbors.Edges)
	}
	relationships, err := TraceNeighbors(root, "gnosis://test/relationship.md", DirectionOut, []string{"uses"})
	if err != nil {
		t.Fatal(err)
	}
	if len(relationships.Edges) != 1 || relationships.Edges[0].To.URI != "gnosis://test/target.md" {
		t.Fatalf("canonical relationship edges = %+v", relationships.Edges)
	}
	page, err := ReadPage(root, "gnosis://test/source.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page.Markdown, "gnosis://test/target.md?view=full#part") {
		t.Fatalf("rendered canonical link = %q", page.Markdown)
	}
}

func TestValidateRejectsDanglingCanonicalLink(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false
`)
	write(t, root, "source.md", `---
type: Note
title: Source
description: Source.
---

<gnosis://test/missing.md>
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(result.Errors, "\n"); !strings.Contains(got, "unresolved internal link gnosis://test/missing.md") {
		t.Fatalf("validation errors = %v", result.Errors)
	}
}

func TestGraphRejectsMalformedGnosisRelationship(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false
`)
	write(t, root, "source.md", `---
type: Note
title: Source
description: Source.
relationships:
  - type: uses
    target: GNOSIS://test/target.md
---
`)

	_, err := TraceNeighbors(root, "gnosis://test/source.md", DirectionOut, nil)
	if err == nil || !strings.Contains(err.Error(), "canonical gnosis URI") {
		t.Fatalf("graph error = %v", err)
	}
}
