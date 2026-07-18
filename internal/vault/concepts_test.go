package vault

import (
	"encoding/json"
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
