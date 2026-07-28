package vault

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestResourceMappingPreservesEffectivePageContentAndProvenance(t *testing.T) {
	workspace := t.TempDir()
	imported := filepath.Join(workspace, "imported")
	writeConfig(t, workspace, `[vault]
vault_name = "workspace"
vault_root = "local"

[[vaults]]
vault_name = "imported"
vault_root = "imported"
`)
	writeConfig(t, imported, `[vault]
vault_name = "imported"
vault_root = "."
`)
	write(t, workspace, "local/shared.md", `---
type: Note
title: Local winner
description: Selected by effective precedence.
---

# Local winner
`)
	write(t, imported, "shared.md", "---\ntype: Note\ntitle: Shadowed\n---\n")

	page, err := ListResourcePage(workspace, "")
	if err != nil {
		t.Fatal(err)
	}
	var descriptor ResourceDescriptor
	for _, resource := range page.Resources {
		if resource.URI == "gnosis://workspace/shared.md" {
			descriptor = resource
		}
		if resource.URI == "gnosis://imported/shared.md" {
			t.Fatal("resource discovery exposed a shadowed page")
		}
	}
	if descriptor.Title != "Local winner" || descriptor.Description != "Selected by effective precedence." ||
		descriptor.MediaType != ResourceMediaType || descriptor.Size == 0 ||
		descriptor.Origin.Vault != "workspace" || descriptor.Origin.Kind != OriginLocal ||
		descriptor.Revision == "" {
		t.Fatalf("descriptor = %+v", descriptor)
	}

	content, err := ReadResource(workspace, descriptor.URI)
	if err != nil {
		t.Fatal(err)
	}
	if content.URI != descriptor.URI || content.MediaType != ResourceMediaType ||
		!strings.Contains(content.Markdown, "# Local winner") ||
		int64(len(content.Markdown)) != descriptor.Size ||
		content.Origin != descriptor.Origin || content.Revision != descriptor.Revision {
		t.Fatalf("content = %+v, descriptor = %+v", content, descriptor)
	}
}

func TestPaginateResources(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		page, err := paginateResources(nil, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Resources) != 0 || page.NextCursor != "" {
			t.Fatalf("page = %+v", page)
		}
	})

	t.Run("single page", func(t *testing.T) {
		resources := []ResourceDescriptor{{URI: "gnosis://test/only.md"}}
		page, err := paginateResources(resources, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Resources) != 1 || page.NextCursor != "" {
			t.Fatalf("page = %+v", page)
		}
	})

	t.Run("multiple pages", func(t *testing.T) {
		resources := make([]ResourceDescriptor, resourcePageSize+1)
		for i := range resources {
			resources[i].URI = fmt.Sprintf("gnosis://test/%03d.md", i)
		}
		first, err := paginateResources(resources, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(first.Resources) != resourcePageSize || first.NextCursor == "" {
			t.Fatalf("first page = %+v", first)
		}
		second, err := paginateResources(resources, first.NextCursor)
		if err != nil {
			t.Fatal(err)
		}
		if len(second.Resources) != 1 || second.Resources[0].URI != resources[resourcePageSize].URI ||
			second.NextCursor != "" {
			t.Fatalf("second page = %+v", second)
		}
	})

	for _, cursor := range []string{"not base64!", "Z25vc2lzOi8vdGVzdC9taXNzaW5nLm1k"} {
		if _, err := paginateResources(
			[]ResourceDescriptor{{URI: "gnosis://test/only.md"}},
			cursor,
		); !errors.Is(err, ErrInvalidResourceCursor) {
			t.Fatalf("cursor %q error = %v, want invalid cursor", cursor, err)
		}
	}
}

func TestReadResourceRejectsUnknownAndNoncanonicalURIs(t *testing.T) {
	root := apiTestVault(t)
	for _, uri := range []string{
		"gnosis://agent-test/missing.md",
		"not-a-gnosis-uri",
	} {
		if _, err := ReadResource(root, uri); !errors.Is(err, ErrPageNotFound) {
			t.Fatalf("ReadResource(%q) error = %v, want page not found", uri, err)
		}
	}
}
