package vault

import (
	"bytes"
	"testing"
)

func TestListConceptsWritesTypePreviewsAndTypedConcepts(t *testing.T) {
	root := t.TempDir()
	write(t, root, "concept-type.md", `---
type: Concept Type
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
	if got, want := output.String(), "Type: Concept\nDescription: A reusable knowledge record.\n\nType: Pattern\nDescription: Pattern\n\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}

	output.Reset()
	if err := ListConcepts(root, " Concept ", &output); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "Title: Attention Mechanism\nDescription: Weighted token lookup.\n\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
