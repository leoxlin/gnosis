package vault

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAuditKnowledgeDetectsEveryClassWithoutMutation(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
entry_points = ["gnosis://test/entry.md"]
vault_index = false
vault_log = false
`)
	write(t, root, "entry.md", auditNote("Entry", "tags: [alpha]\n", ""))
	write(t, root, "orphan.md", auditNote("Orphan", "tags: [alpha]\n", ""))
	write(t, root, "duplicate-a.md", auditNote("Same-name", "tags: [foo-bar]\n", ""))
	write(t, root, "duplicate-b.md", auditNote("Other", "aliases: [same name]\ntags: [foo_bar]\n", ""))
	write(t, root, "ambiguous.md", auditNote("Ambiguous", "", "Unresolved. ^[ambiguous]\n"))
	write(t, root, "contradiction-a.md", auditNote("Claim A", `relationships:
  - type: contradicts
    target: contradiction-b.md
`, ""))
	write(t, root, "contradiction-b.md", auditNote("Claim B", "", ""))
	write(t, root, "stale.md", auditNote("Stale", "observed_at: 2020-01-01T00:00:00Z\n", ""))
	write(t, root, "broken.md", auditNote("Broken", "superseded_by: missing.md\n", ""))
	write(t, root, "cycle-a.md", auditNote("Cycle A", "superseded_by: cycle-b.md\n", ""))
	write(t, root, "cycle-b.md", auditNote("Cycle B", "superseded_by: cycle-a.md\n", ""))
	write(t, root, "old.md", auditNote("Old", "superseded_by: retired.md\n", ""))
	write(t, root, "retired.md", auditNote("Retired", "status: archived\n", ""))
	write(t, root, "active.md", auditNote("Active", `relationships:
  - type: uses
    target: retired.md
`, ""))
	write(t, root, "authored.md", auditNote("Authored", `maintenance:
  - kind: stale
    reason: Source changed.
    observed_at: 2026-07-29T09:00:00Z
    author: agent-id
  - kind: incorrect
    reason: Claim is false.
    observed_at: 2026-07-29T09:01:00Z
  - kind: duplicate
    reason: Retired is canonical.
    observed_at: 2026-07-29T09:02:00Z
    target: gnosis://test/retired.md
`, ""))
	write(t, root, "broken-maintenance.md", auditNote("Broken maintenance", `maintenance:
  - kind: duplicate
    reason: Target disappeared.
    observed_at: 2026-07-29T09:03:00Z
    target: gnosis://test/missing.md
`, ""))

	before := auditFixtureFiles(t, root)
	result, err := AuditKnowledge(root, KnowledgeAuditRequest{
		Classes: AllFindingClasses, Types: []string{"Note"}, StaleAfter: "24h",
		PageLimit: 100, FindingLimit: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	after := auditFixtureFiles(t, root)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("audit mutated vault: before=%v after=%v", before, after)
	}
	found := map[FindingClass]bool{}
	for _, finding := range result.Findings {
		found[finding.Class] = true
		if finding.ID == "" || len(finding.URIs) == 0 || len(finding.Evidence) == 0 {
			t.Fatalf("incomplete finding = %+v", finding)
		}
	}
	for _, class := range AllFindingClasses {
		if !found[class] {
			t.Errorf("missing finding class %q: %+v", class, result.Findings)
		}
	}
	for _, finding := range result.Findings {
		if finding.Class == FindingOrphan && finding.URIs[0] == "gnosis://test/entry.md" {
			t.Fatalf("configured entry point reported as orphan: %+v", finding)
		}
	}
}

func TestAuditKnowledgeExemptsConceptBackedTypeDefinitionsFromOrphans(t *testing.T) {
	root := trustTestVault(t)
	write(t, root, "concepts/note.md", `---
type: Concept
title: Note
description: A short general-purpose record.
---
`)
	write(t, root, "orphan.md", auditNote("Orphan", "", ""))

	result, err := AuditKnowledge(root, KnowledgeAuditRequest{
		Classes: []FindingClass{FindingOrphan}, PageLimit: 100, FindingLimit: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if hasAuditFinding(result.Findings, FindingOrphan, "gnosis://test/concepts/note.md") {
		t.Fatalf("Concept-backed type definition reported as orphan: %+v", result.Findings)
	}
	if !hasAuditFinding(result.Findings, FindingOrphan, "gnosis://test/orphan.md") {
		t.Fatalf("ordinary orphan missing from findings: %+v", result.Findings)
	}
}

func TestAuditKnowledgeSeparatesAuthoredMaintenanceFromHeuristics(t *testing.T) {
	root := trustTestVault(t)
	write(t, root, "canonical.md", auditNote("Same", "", ""))
	write(t, root, "duplicate.md", auditNote("Same", `maintenance:
  - kind: duplicate
    reason: Canonical has the same claim.
    observed_at: 2026-07-29T09:00:00Z
    author: agent-id
    target: gnosis://test/canonical.md
`, ""))

	result, err := AuditKnowledge(root, KnowledgeAuditRequest{
		Classes:   []FindingClass{FindingAuthoredDuplicate, FindingDuplicateIdentity},
		PageLimit: 100, FindingLimit: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if hasAuditFinding(result.Findings, FindingDuplicateIdentity, "gnosis://test/duplicate.md") {
		t.Fatalf("authored duplicate also emitted heuristic candidate: %+v", result.Findings)
	}
	for _, finding := range result.Findings {
		if finding.Class != FindingAuthoredDuplicate {
			continue
		}
		if finding.Classification != ClassificationAuthored ||
			finding.Evidence[0].Reason != "Canonical has the same claim." ||
			finding.Evidence[0].Author != "agent-id" ||
			finding.Evidence[0].TargetURI != "gnosis://test/canonical.md" {
			t.Fatalf("authored finding = %+v", finding)
		}
		return
	}
	t.Fatalf("missing authored duplicate: %+v", result.Findings)
}

func TestAuditKnowledgeValidatesBoundsStalenessAndContinuation(t *testing.T) {
	root := trustTestVault(t)
	write(t, root, "a.md", auditNote("A", "", ""))
	write(t, root, "b.md", auditNote("B", "", ""))

	_, err := AuditKnowledge(root, KnowledgeAuditRequest{
		Classes: []FindingClass{FindingStale}, PageLimit: 10, FindingLimit: 10,
	})
	if err == nil || !strings.Contains(err.Error(), "stale_after is required") {
		t.Fatalf("missing stale threshold error = %v", err)
	}
	_, err = AuditKnowledge(root, KnowledgeAuditRequest{
		Classes: []FindingClass{FindingStale}, StaleAfter: "24h",
		PageLimit: 10, FindingLimit: 10,
	})
	if err == nil || !strings.Contains(err.Error(), "type or tier filter") {
		t.Fatalf("missing stale scope error = %v", err)
	}
	_, err = AuditKnowledge(root, KnowledgeAuditRequest{
		Classes: []FindingClass{"unknown"}, PageLimit: 10, FindingLimit: 10,
	})
	if err == nil || !strings.Contains(err.Error(), "unknown class") {
		t.Fatalf("unknown class error = %v", err)
	}
	_, err = AuditKnowledge(root, KnowledgeAuditRequest{
		Classes: []FindingClass{FindingOrphan}, Types: []string{"Note"},
		PageLimit: 1, FindingLimit: 10,
	})
	if err == nil || !strings.Contains(err.Error(), "2 pages match") {
		t.Fatalf("page bound error = %v", err)
	}

	first, err := AuditKnowledge(root, KnowledgeAuditRequest{
		Classes: []FindingClass{FindingOrphan}, Types: []string{"Note"},
		PageLimit: 10, FindingLimit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !first.Bound.Truncated || first.NextCursor == "" || len(first.Findings) != 1 {
		t.Fatalf("first page = %+v", first)
	}
	second, err := AuditKnowledge(root, KnowledgeAuditRequest{
		Classes: []FindingClass{FindingOrphan}, Types: []string{"Note"},
		PageLimit: 10, FindingLimit: 1,
		Cursor: first.NextCursor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Bound.Truncated || second.NextCursor != "" || len(second.Findings) != 1 ||
		second.Findings[0].ID == first.Findings[0].ID {
		t.Fatalf("second page = %+v", second)
	}
	_, err = AuditKnowledge(root, KnowledgeAuditRequest{
		Classes: []FindingClass{FindingOrphan}, Types: []string{"Note"},
		PageLimit: 10, FindingLimit: 2,
		Cursor: first.NextCursor,
	})
	if !errors.Is(err, ErrInvalidAuditCursor) {
		t.Fatalf("changed-request cursor error = %v", err)
	}
}

func TestAuditKnowledgeFiltersFindingsWithoutClippingGraphEvidence(t *testing.T) {
	root := trustTestVault(t)
	write(t, root, "selected.md", auditNote("Selected", "", ""))
	write(t, root, "retired.md", "---\ntype: Other\ntitle: Retired\nstatus: archived\n---\n")
	write(t, root, "outside.md", `---
type: Other
title: Outside
relationships:
  - type: links
    target: selected.md
---
`)
	write(t, root, "active.md", `---
type: Note
title: Active
relationships:
  - type: uses
    target: retired.md
---
`)

	result, err := AuditKnowledge(root, KnowledgeAuditRequest{
		Classes: []FindingClass{FindingOrphan, FindingActiveReferenceRetired},
		Types:   []string{"Note"}, PageLimit: 10, FindingLimit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range result.Findings {
		if finding.Class == FindingOrphan && finding.URIs[0] == "gnosis://test/selected.md" {
			t.Fatalf("selected page ignored inbound edge outside filter: %+v", finding)
		}
	}
	if !hasAuditFinding(result.Findings, FindingActiveReferenceRetired, "gnosis://test/retired.md") {
		t.Fatalf("filtered audit lost retired target evidence: %+v", result.Findings)
	}
}

func hasAuditFinding(findings []KnowledgeFinding, class FindingClass, uri string) bool {
	for _, finding := range findings {
		if finding.Class != class {
			continue
		}
		for _, affected := range finding.URIs {
			if affected == uri {
				return true
			}
		}
	}
	return false
}

func auditNote(title, metadata, body string) string {
	return "---\ntype: Note\ntitle: " + title + "\ntier: core\n" + metadata + "---\n\n" + body
}

func auditFixtureFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		result[relative] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
