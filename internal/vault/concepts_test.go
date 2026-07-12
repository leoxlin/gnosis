package vault

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestConceptRecordsPreserveFrontmatterUnderConceptsKey(t *testing.T) {
	root := t.TempDir()
	write(t, root, "policy.md", `---
type: AccessPolicy
title: Review Policy
enabled: true
reviewers: [alice, bob]
limits:
  retries: 3
---
`)

	catalog, err := ConceptRecords(root, "AccessPolicy")
	if err != nil {
		t.Fatal(err)
	}
	records := catalog["concepts"]
	if len(records) != 1 || records[0]["uri"] != "gnosis://Test/policy.md" {
		t.Fatalf("catalog = %+v", catalog)
	}
	encoded, err := json.Marshal(records[0])
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal(encoded, &record); err != nil {
		t.Fatal(err)
	}
	if record["enabled"] != true || record["title"] != "Review Policy" {
		t.Fatalf("record = %+v", record)
	}
	reviewers, ok := record["reviewers"].([]any)
	if !ok || len(reviewers) != 2 {
		t.Fatalf("reviewers = %#v", record["reviewers"])
	}
	limits, ok := record["limits"].(map[string]any)
	if !ok || limits["retries"] != float64(3) {
		t.Fatalf("limits = %#v", record["limits"])
	}
}

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
