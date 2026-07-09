package vault

import (
	"strings"
	"testing"
)

func TestParseFrontmatterSupportsYAMLValues(t *testing.T) {
	markdown := `---
type: Documentation
title: "Quoted: title"
description: >
  A folded description.
tags:
  - docs
  - yaml
---

# Body
`

	fields, body, err := parseFrontmatter(markdown)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := fields.scalar("title"); got != "Quoted: title" {
		t.Fatalf("title = %q", got)
	}
	if got, _ := fields.scalar("description"); got != "A folded description.\n" {
		t.Fatalf("description = %q", got)
	}
	if !fields.nonEmpty("tags") {
		t.Fatal("expected tags sequence to be non-empty")
	}
	if !strings.Contains(body, "# Body") {
		t.Fatalf("body = %q", body)
	}
}

func TestParseFrontmatterSupportsCRLF(t *testing.T) {
	fields, body, err := parseFrontmatter("---\r\ntype: Concept\r\n---\r\n\r\n# Body\r\n")
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := fields.scalar("type"); got != "Concept" {
		t.Fatalf("type = %q", got)
	}
	if !strings.Contains(body, "# Body") {
		t.Fatalf("body = %q", body)
	}
}

func TestParseFrontmatterRejectsInvalidDocuments(t *testing.T) {
	tests := []struct {
		name     string
		markdown string
		message  string
	}{
		{name: "missing", markdown: "# Body\n", message: "missing"},
		{name: "unterminated", markdown: "---\ntype: Concept\n", message: "unterminated"},
		{name: "malformed", markdown: "---\ntype: [\n---\n", message: "invalid YAML"},
		{name: "sequence", markdown: "---\n- Concept\n---\n", message: "must be a mapping"},
		{name: "duplicate", markdown: "---\ntype: A\ntype: B\n---\n", message: "duplicate"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := parseFrontmatter(test.markdown)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("error = %v, want %q", err, test.message)
			}
		})
	}
}

func TestValidateAcceptsSequenceTags(t *testing.T) {
	root := t.TempDir()
	write(t, root, "index.md", "# Index\n\n[Log](log.md)\n")
	write(t, root, "log.md", "# Log\n\n## 2026-07-09\n")
	write(t, root, "note.md", `---
type: Documentation
title: Note
description: A note.
tags: [docs, yaml]
timestamp: 2026-07-09T00:00:00Z
---

# Note
`)

	result, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Errors) != 0 || len(result.Warnings) != 0 {
		t.Fatalf("result = %+v", result)
	}
}
