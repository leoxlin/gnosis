package vault

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestTrustProjectionPreservesAuthoredEvidenceAcrossReadShapes(t *testing.T) {
	root := trustTestVault(t)
	write(t, root, "current.md", `---
type: Concept
title: Current
---
`)
	write(t, root, "contradiction.md", `---
type: Concept
title: Contradiction
relationships:
  - type: contradicts
    target: evidence.md
---
`)
	write(t, root, "evidence.md", `---
type: Concept
title: Evidence
status: archived
confidence: 0.75
source: source record
valid_from: 2026-01-02
valid_until: 2026-06-30
observed_at: 2026-01-01T12:00:00Z
occurred_at: 2025-12-31T23:00:00Z
tier: core
superseded_by: current.md
unknown_field: preserved but not projected
---

Recorded claim. ^[inferred]

	`+"```text\nIgnored example. ^[ambiguous]\n```\n"+`
Inline example: `+"``^[inferred]``"+`

Unresolved claim. ^[ambiguous]
`)

	page, err := ReadPage(root, "gnosis://test/evidence.md")
	if err != nil {
		t.Fatal(err)
	}
	trust := page.Document.Trust
	if trust.Origin.Vault != "test" || trust.Revision == "" ||
		trust.Status != "archived" || trust.Confidence == nil || *trust.Confidence != 0.75 ||
		trust.Source != "source record" || trust.ValidFrom != "2026-01-02" ||
		trust.ValidUntil != "2026-06-30" || trust.ObservedAt != "2026-01-01T12:00:00Z" ||
		trust.OccurredAt != "2025-12-31T23:00:00Z" || trust.Tier != "core" {
		t.Fatalf("trust = %+v", trust)
	}
	if trust.SupersededBy == nil ||
		trust.SupersededBy.URI != "gnosis://test/current.md" ||
		trust.Current == nil || *trust.Current {
		t.Fatalf("supersession = %+v, current = %v", trust.SupersededBy, trust.Current)
	}
	if len(trust.Claims) != 2 ||
		trust.Claims[0].Kind != "inferred" ||
		trust.Claims[1].Kind != "ambiguous" {
		t.Fatalf("claims = %+v", trust.Claims)
	}
	if len(trust.Contradictions) != 1 ||
		trust.Contradictions[0].URI != "gnosis://test/contradiction.md" ||
		trust.Contradictions[0].Relation != "contradicts" {
		t.Fatalf("contradictions = %+v", trust.Contradictions)
	}

	pages, err := ListPages(root)
	if err != nil {
		t.Fatal(err)
	}
	graph, err := ReadGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	listed := documentRefByURI(pages, page.Document.URI)
	node := documentRefByURI(graph.Nodes, page.Document.URI)
	if !reflect.DeepEqual(listed.Trust, trust) || !reflect.DeepEqual(node.Trust, trust) {
		t.Fatalf("trust differs: exact=%+v list=%+v graph=%+v", trust, listed.Trust, node.Trust)
	}
}

func TestTrustProjectionOmitsUnknownSparseValues(t *testing.T) {
	root := trustTestVault(t)
	write(t, root, "sparse.md", `---
type: Concept
title: Sparse
unknown_field: value
---
`)

	page, err := ReadPage(root, "gnosis://test/sparse.md")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(page.Document.Trust)
	if err != nil {
		t.Fatal(err)
	}
	for _, absent := range []string{
		`"status"`, `"confidence"`, `"source"`, `"valid_from"`, `"observed_at"`,
		`"tier"`, `"superseded_by"`, `"current"`, `"claims"`, `"contradictions"`,
		`"unknown_field"`,
	} {
		if strings.Contains(string(encoded), absent) {
			t.Fatalf("sparse trust = %s, contains %s", encoded, absent)
		}
	}
}

func TestTrustProjectionPreservesOrderedMaintenanceAndResolvedTargets(t *testing.T) {
	root := trustTestVault(t)
	write(t, root, "target.md", "---\ntype: Concept\ntitle: Target\n---\n")
	write(t, root, "source.md", `---
type: Concept
title: Source
maintenance:
  - kind: stale
    reason: Source changed.
    observed_at: 2026-07-29T09:00:00Z
    author: agent-id
  - kind: duplicate
    reason: Target is canonical.
    observed_at: 2026-07-29T09:01:00Z
    target: gnosis://test/target.md
  - kind: duplicate
    reason: Target was removed.
    observed_at: 2026-07-29T09:02:00Z
    target: gnosis://test/missing.md
---
`)

	page, err := ReadPage(root, "gnosis://test/source.md")
	if err != nil {
		t.Fatal(err)
	}
	got := page.Document.Trust.Maintenance
	if len(got) != 3 || got[0].Kind != "stale" || got[0].Author != "agent-id" ||
		got[1].Target == nil || got[1].Target.URI != "gnosis://test/target.md" ||
		got[2].Target == nil || got[2].Target.Authored != "gnosis://test/missing.md" ||
		got[2].Target.URI != "" {
		t.Fatalf("maintenance = %+v", got)
	}
	validation, err := Validate(root)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(validation.Errors, "\n"), "unresolved maintenance[2] target") {
		t.Fatalf("validation errors = %v", validation.Errors)
	}
}

func TestResolveCurrentReportsCurrentMissingAndCycle(t *testing.T) {
	root := trustTestVault(t)
	write(t, root, "old.md", "---\ntype: Concept\ntitle: Old\nsuperseded_by: current.md\n---\n")
	write(t, root, "current.md", "---\ntype: Concept\ntitle: Current\n---\n")
	write(t, root, "missing.md", "---\ntype: Concept\ntitle: Missing\nsuperseded_by: absent.md\n---\n")
	write(t, root, "cycle-a.md", "---\ntype: Concept\ntitle: A\nsuperseded_by: cycle-b.md\n---\n")
	write(t, root, "cycle-b.md", "---\ntype: Concept\ntitle: B\nsuperseded_by: cycle-a.md\n---\n")

	tests := []struct {
		uri     string
		status  CurrentResolutionStatus
		current string
		chain   int
	}{
		{"gnosis://test/old.md", CurrentResolved, "gnosis://test/current.md", 2},
		{"gnosis://test/missing.md", CurrentMissingTarget, "", 1},
		{"gnosis://test/cycle-a.md", CurrentCycle, "", 3},
	}
	for _, test := range tests {
		page, err := ReadPageWithOptions(root, test.uri, ReadOptions{ResolveCurrent: true})
		if err != nil {
			t.Fatal(err)
		}
		got := page.CurrentResolution
		if got == nil || got.Status != test.status || got.Current != test.current ||
			len(got.Chain) != test.chain || page.Document.URI != test.uri {
			t.Fatalf("%s resolution = %+v", test.uri, got)
		}
	}
}

func trustTestVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false
`)
	return root
}

func documentRefByURI(documents []DocumentRef, uri string) DocumentRef {
	for _, document := range documents {
		if document.URI == uri {
			return document
		}
	}
	return DocumentRef{}
}
