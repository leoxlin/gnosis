package vault

import (
	"encoding/json"
	"testing"
)

func TestConceptsDiscoversConceptBackedTypeDefinitions(t *testing.T) {
	root := t.TempDir()
	write(t, root, "concepts/note.md", `---
type: Concept
title: Note
description: A short general-purpose record.
---
`)

	catalog, err := Concepts(root, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, summary := range catalog.ConceptTypes {
		if summary.Type == "Note" {
			if summary.Description != "A short general-purpose record." ||
				summary.URI != "gnosis://Test/concepts/note.md" {
				t.Fatalf("Note summary = %+v", summary)
			}
			return
		}
	}
	t.Fatalf("concept types = %+v, want Note", catalog.ConceptTypes)
}

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
