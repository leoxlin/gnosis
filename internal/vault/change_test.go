package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKnowledgeChangePlanCreateIsDeterministicAndSideEffectFree(t *testing.T) {
	root := knowledgeChangeVault(t)
	input := KnowledgeChangeInput{
		URI:            "gnosis://test/notes/new.md",
		Candidate:      validChangeNote("New", "created"),
		ExpectedAbsent: true,
	}

	first, err := PlanKnowledgeChange(root, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := PlanKnowledgeChange(root, input)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Applicable || first.Operation != "create" {
		t.Fatalf("plan = %+v", first)
	}
	if first.Digest != second.Digest || first.Diff != second.Diff {
		t.Fatal("plan digest or diff is not deterministic")
	}
	if first.BeforeRevision != "" || first.AfterRevision == "" ||
		!strings.Contains(first.Diff, "+title: New") {
		t.Fatalf("plan = %+v", first)
	}
	if _, err := os.Stat(filepath.Join(root, "notes", "new.md")); !os.IsNotExist(err) {
		t.Fatalf("planning wrote target: %v", err)
	}
}

func TestKnowledgeChangeApplyRejectsStaleAndChangedPlans(t *testing.T) {
	root := knowledgeChangeVault(t)
	path := filepath.Join(root, "notes", "existing.md")
	write(t, root, "notes/existing.md", validChangeNote("Existing", "first"))
	page, err := ReadPage(root, "gnosis://test/notes/existing.md")
	if err != nil {
		t.Fatal(err)
	}
	input := KnowledgeChangeInput{
		URI:              "gnosis://test/notes/existing.md",
		Candidate:        validChangeNote("Existing", "planned"),
		ExpectedRevision: page.Document.Revision,
	}
	plan, err := PlanKnowledgeChange(root, input)
	if err != nil || !plan.Applicable {
		t.Fatalf("plan = %+v err = %v", plan, err)
	}

	input.Candidate = validChangeNote("Existing", "changed")
	if _, err := ApplyKnowledgeChange(root, input, plan.Digest); err == nil ||
		!strings.Contains(err.Error(), "plan digest changed") {
		t.Fatalf("changed plan error = %v", err)
	}
	input.Candidate = validChangeNote("Existing", "planned")
	write(t, root, "notes/existing.md", validChangeNote("Existing", "external"))
	if _, err := ApplyKnowledgeChange(root, input, plan.Digest); err == nil ||
		!strings.Contains(err.Error(), "actual revision") {
		t.Fatalf("stale plan error = %v", err)
	}
	if got, err := os.ReadFile(path); err != nil || !strings.Contains(string(got), "external") {
		t.Fatalf("target = %q err = %v", got, err)
	}
}

func TestKnowledgeChangePreservesUnknownMetadataAndClassifiesArchive(t *testing.T) {
	root := knowledgeChangeVault(t)
	write(t, root, "notes/existing.md", `---
type: Note
title: Existing
description: existing
status: active
extension_field: keep-me
---

first
`)
	page, err := ReadPage(root, "gnosis://test/notes/existing.md")
	if err != nil {
		t.Fatal(err)
	}
	input := KnowledgeChangeInput{
		URI: "gnosis://test/notes/existing.md",
		Candidate: `---
type: Note
title: Existing
description: archived
status: archived
---

retained
`,
		ExpectedRevision: page.Document.Revision,
	}
	plan, err := PlanKnowledgeChange(root, input)
	if err != nil || !plan.Applicable || plan.Operation != "archive" {
		t.Fatalf("plan = %+v err = %v", plan, err)
	}
	result, err := ApplyKnowledgeChange(root, input, plan.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Operation != "archive" || result.Revision != plan.AfterRevision {
		t.Fatalf("result = %+v", result)
	}
	data, err := os.ReadFile(filepath.Join(root, "notes", "existing.md"))
	if err != nil || !strings.Contains(string(data), "extension_field: keep-me") {
		t.Fatalf("metadata was not preserved: %q err = %v", data, err)
	}
}

func TestKnowledgeChangeClassifiesNoOpAndUpdateRelationships(t *testing.T) {
	root := knowledgeChangeVault(t)
	write(t, root, "notes/target.md", validChangeNote("Target", "target"))
	existing := `---
type: Note
title: Existing
description: existing
relationships:
  - type: supports
    target: target.md
---

first
`
	write(t, root, "notes/existing.md", existing)
	page, err := ReadPage(root, "gnosis://test/notes/existing.md")
	if err != nil {
		t.Fatal(err)
	}
	noOp, err := PlanKnowledgeChange(root, KnowledgeChangeInput{
		URI:              "gnosis://test/notes/existing.md",
		Candidate:        existing,
		ExpectedRevision: page.Document.Revision,
	})
	if err != nil || !noOp.Applicable || noOp.Operation != "no-op" {
		t.Fatalf("no-op plan = %+v err = %v", noOp, err)
	}
	update, err := PlanKnowledgeChange(root, KnowledgeChangeInput{
		URI:              "gnosis://test/notes/existing.md",
		Candidate:        validChangeNote("Existing", "updated"),
		ExpectedRevision: page.Document.Revision,
	})
	if err != nil || !update.Applicable || update.Operation != "update" {
		t.Fatalf("update plan = %+v err = %v", update, err)
	}
	if len(update.AffectedRelationships) != 1 ||
		update.AffectedRelationships[0].Change != "removed" ||
		update.AffectedRelationships[0].Relation != "supports" {
		t.Fatalf("relationships = %+v", update.AffectedRelationships)
	}
}

func TestKnowledgeChangePlanReportsValidationAndOwnershipFindings(t *testing.T) {
	root := knowledgeChangeVault(t)
	tests := []struct {
		name  string
		input KnowledgeChangeInput
		code  string
	}{
		{
			name: "invalid procedure",
			input: KnowledgeChangeInput{
				URI: "gnosis://test/procedures/bad.md",
				Candidate: `---
type: Procedure
title: Bad
description: invalid
tags: test
---
`,
				ExpectedAbsent: true,
			},
			code: "validation",
		},
		{
			name: "unresolved link",
			input: KnowledgeChangeInput{
				URI:            "gnosis://test/notes/bad-link.md",
				Candidate:      validChangeNote("Bad link", "[missing](missing.md)"),
				ExpectedAbsent: true,
			},
			code: "validation",
		},
		{
			name: "physical deletion",
			input: KnowledgeChangeInput{
				URI:              "gnosis://test/notes/missing.md",
				ExpectedRevision: "sha256:missing",
			},
			code: "physical_deletion_unsupported",
		},
		{
			name: "projected openspec",
			input: KnowledgeChangeInput{
				URI:            "gnosis://test/openspec/changes/example/proposal.md",
				Candidate:      validChangeNote("Proposal", "content"),
				ExpectedAbsent: true,
			},
			code: "projected_openspec",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan, err := PlanKnowledgeChange(root, test.input)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Applicable || plan.Operation != "invalid" || !hasFindingCode(plan.Findings, test.code) {
				t.Fatalf("plan = %+v", plan)
			}
		})
	}
}

func TestKnowledgeChangeApplyCreatesOneRemoteCommitAndRetainsFailedPush(t *testing.T) {
	fixture := newGitRemoteFixture(t, "https://example.test/team/knowledge-change.git")
	configureTestRemoteVault(t, fixture)
	input := KnowledgeChangeInput{
		URI:            "gnosis://remote/notes/accepted.md",
		Candidate:      validChangeNote("Accepted", "remote"),
		ExpectedAbsent: true,
	}
	plan, err := PlanKnowledgeChange(fixture.url, input)
	if err != nil || !plan.Applicable {
		t.Fatalf("plan = %+v err = %v", plan, err)
	}
	before := remoteCommitCount(t, fixture)
	if _, err := ApplyKnowledgeChange(fixture.url, input, plan.Digest); err != nil {
		t.Fatal(err)
	}
	if got := remoteCommitCount(t, fixture); got != before+1 {
		t.Fatalf("remote commits = %d, want %d", got, before+1)
	}

	failedInput := KnowledgeChangeInput{
		URI:            "gnosis://remote/notes/rejected.md",
		Candidate:      validChangeNote("Rejected", "remote"),
		ExpectedAbsent: true,
	}
	failedPlan, err := PlanKnowledgeChange(fixture.url, failedInput)
	if err != nil || !failedPlan.Applicable {
		t.Fatalf("plan = %+v err = %v", failedPlan, err)
	}
	target, err := resolveVaultTarget(fixture.url)
	if err != nil {
		t.Fatal(err)
	}
	beforeLocal := localCommitCount(t, target.root)
	beforeRemote := remoteCommitCount(t, fixture)
	rejectTestPushes(t, fixture.remote)
	result, err := ApplyKnowledgeChange(fixture.url, failedInput, failedPlan.Digest)
	if err == nil || !strings.Contains(err.Error(), "local write remains") {
		t.Fatalf("push error = %v", err)
	}
	if !result.Changed || result.Path == "" {
		t.Fatalf("result = %+v", result)
	}
	if got := localCommitCount(t, target.root); got != beforeLocal+1 {
		t.Fatalf("local commits = %d, want %d", got, beforeLocal+1)
	}
	if got := remoteCommitCount(t, fixture); got != beforeRemote {
		t.Fatalf("remote commits = %d, want %d", got, beforeRemote)
	}
}

func knowledgeChangeVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false
`)
	write(t, root, "types/note.md", `---
type: ConceptType
title: Note
description: A test note.
path: notes
---
`)
	write(t, root, "types/procedure.md", `---
type: ConceptType
title: Procedure
description: An executable procedure.
path: procedures
---
`)
	return root
}

func validChangeNote(title, body string) string {
	return "---\ntype: Note\ntitle: " + title + "\ndescription: test note\n---\n\n" + body + "\n"
}

func hasFindingCode(findings []KnowledgeChangeFinding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}
