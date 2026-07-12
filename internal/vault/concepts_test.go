package vault

import (
	"bytes"
	"strings"
	"testing"
)

func TestListConceptsWritesTypePreviewsAndTypedConcepts(t *testing.T) {
	root := t.TempDir()
	write(t, root, "concept-type.md", `---
type: ConceptType
title: Concept
description: A reusable knowledge record.
---
`)
	write(t, root, "pattern.md", `---
type: Pattern
title: Adapter Pattern
description: Decouples an interface from its implementation.
---
`)
	write(t, root, "concept.md", `---
type: Concept
title: Attention Mechanism
description: Weighted token lookup.
---
`)

	var output bytes.Buffer
	if err := ListConcepts(root, "", &output); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); !strings.Contains(got, "Type: Concept\nDescription: A reusable knowledge record.\n") || !strings.Contains(got, "Type: Pattern\nDescription: Pattern\n") || !strings.Contains(got, "Type: Procedure") {
		t.Fatalf("output = %q", got)
	}
	if got := output.String(); strings.Contains(got, "Type: Vault Process") || strings.Contains(got, "Type: Repository Process") {
		t.Fatalf("output contains legacy process types = %q", got)
	}

	output.Reset()
	if err := ListConcepts(root, " Concept ", &output); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "Title: Attention Mechanism\nDescription: Weighted token lookup.\nLink: gnosis://Test/concept.md\n\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
