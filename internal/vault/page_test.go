package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteDocumentRejectsFrontmatterSearchCannotRead(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false
`)
	write(t, root, "concepts/note.md", `---
type: ConceptType
title: Note
path: notes
---
`)

	content := []byte(`---
type: Note
title: Invalid tags
tags:
  nested: value
---
`)
	_, err := WriteDocument(root, "gnosis://test/notes/invalid-tags.md", content, false)
	if err == nil || !strings.Contains(err.Error(), "tags") {
		t.Fatalf("error = %v, want invalid tags", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "notes", "invalid-tags.md")); !os.IsNotExist(statErr) {
		t.Fatalf("written file error = %v, want not found", statErr)
	}
}

func TestValidateRejectsFrontmatterSearchCannotRead(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false
`)
	write(t, root, "note.md", `---
type: Note
title: Invalid aliases
description: Invalid aliases must not pass validation.
aliases:
  nested: value
---

# Invalid aliases
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(result.Errors, "\n"); !strings.Contains(got, `"aliases" must be a scalar or sequence of scalars`) {
		t.Fatalf("errors = %v, want invalid aliases", result.Errors)
	}
}

func TestValidateRetainsEffectiveLinkIdentitiesBesideMalformedPage(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false
`)
	write(t, root, "target.md", "---\ntype: Note\ntitle: Target\ndescription: Target.\n---\n")
	write(t, root, "source.md", "---\ntype: Note\ntitle: Source\ndescription: Source.\n---\n\n[Target](gnosis://test/target.md)\n")
	write(t, root, "malformed.md", "---\ntype: Note\ntitle: Malformed\naliases:\n  nested: invalid\n---\n")

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	errors := strings.Join(result.Errors, "\n")
	if !strings.Contains(errors, `"aliases" must be a scalar or sequence of scalars`) {
		t.Fatalf("errors = %v, want malformed aliases", result.Errors)
	}
	if strings.Contains(errors, "unresolved internal link") {
		t.Fatalf("errors = %v, want effective target identity retained", result.Errors)
	}
}

func TestWriteAndValidateSharePageMetadataRules(t *testing.T) {
	tests := []struct {
		name   string
		fields string
		want   string
	}{
		{name: "null title", fields: "title: null\ndescription: Valid.\n", want: `frontmatter "title" must be a scalar`},
		{name: "description sequence", fields: "title: Valid\ndescription: [invalid]\n", want: `frontmatter "description" must be a scalar`},
		{name: "summary mapping", fields: "title: Valid\nsummary:\n  nested: invalid\n", want: `frontmatter "summary" must be a scalar`},
		{name: "tags mapping", fields: "title: Valid\ndescription: Valid.\ntags:\n  nested: invalid\n", want: `frontmatter "tags" must be a scalar or sequence of scalars`},
		{name: "aliases mapping", fields: "title: Valid\ndescription: Valid.\naliases:\n  nested: invalid\n", want: `frontmatter "aliases" must be a scalar or sequence of scalars`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false
`)
			write(t, root, "concepts/note.md", "---\ntype: ConceptType\ntitle: Note\npath: notes\n---\n")
			content := []byte("---\ntype: Note\n" + test.fields + "---\n\n# Note\n")

			if _, err := WriteDocument(root, "gnosis://test/notes/input.md", content, false); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("write error = %v, want %q", err, test.want)
			}
			write(t, root, "notes/existing.md", string(content))
			result, err := Validate(root)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(strings.Join(result.Errors, "\n"), test.want) {
				t.Fatalf("validation errors = %v, want %q", result.Errors, test.want)
			}
		})
	}
}

func TestWriteRejectsPageStructuresThatResolvedReadsReject(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false
`)
	write(t, root, "concepts/note.md", "---\ntype: ConceptType\ntitle: Note\npath: notes\n---\n")

	tests := []struct {
		name    string
		path    string
		content string
		want    string
	}{
		{
			name:    "relationships",
			path:    "relationships",
			content: "---\ntype: Note\ntitle: Invalid\nrelationships:\n  type: uses\n  target: target.md\n---\n",
			want:    "relationships",
		},
		{
			name:    "link destination",
			path:    "link-destination",
			content: "---\ntype: Note\ntitle: Invalid\n---\n\n[Invalid](bad%ZZ.md)\n",
			want:    "invalid destination",
		},
		{
			name:    "escaping body link",
			path:    "escaping-body-link",
			content: "---\ntype: Note\ntitle: Invalid\n---\n\n[Outside](../../outside.md)\n",
			want:    "escapes the vault root",
		},
		{
			name: "escaping relationship",
			path: "escaping-relationship",
			content: `---
type: Note
title: Invalid
relationships:
  - type: uses
    target: ../../outside.md
---
`,
			want: "escapes the vault root",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := WriteDocument(root, "gnosis://test/notes/"+test.path+".md", []byte(test.content), false)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestConceptRecordsPreserveCanonicalURIOverAuthoredField(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false
`)
	write(t, root, "policy.md", `---
type: Policy
title: Policy
uri: 42
---
`)

	catalog, err := ConceptRecords(root, "Policy")
	if err != nil {
		t.Fatal(err)
	}
	records := catalog["concepts"]
	if len(records) != 1 || records[0]["uri"] != "gnosis://test/policy.md" {
		t.Fatalf("records = %+v", records)
	}
}
