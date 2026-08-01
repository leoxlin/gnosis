package learning

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gnosis/internal/trace"
	"gnosis/internal/vault"
)

const (
	testProcedureURI      = "gnosis://test/procedures/query.md"
	testProcedureRevision = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

func TestBuildProducesDeterministicSupportingAndContradictingEvidence(t *testing.T) {
	store := learningStore(t)
	recordRun(t, store, "support", true, testProcedureRevision)
	recordRun(t, store, "contradict", false, testProcedureRevision)
	selections := []Selection{
		{RunID: "support", LearningKey: "bounded reads"},
		{RunID: "contradict", LearningKey: "bounded reads"},
	}

	first, err := Build(store, selections)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(store, []Selection{selections[1], selections[0]})
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, _ := json.Marshal(first)
	secondJSON, _ := json.Marshal(second)
	if string(firstJSON) != string(secondJSON) {
		t.Fatalf("candidates differ:\n%s\n%s", firstJSON, secondJSON)
	}
	if first.AgentID != "agent" || first.ProcedureURI != testProcedureURI ||
		first.ProcedureRevision != testProcedureRevision || len(first.Evidence) != 6 ||
		len(first.Outcomes) != 2 || first.Outcomes[0].Supporting ||
		!first.Outcomes[1].Supporting || len(first.Attributions) != 2 {
		t.Fatalf("candidate = %+v", first)
	}
}

func TestBuildRejectsIncompleteIncompatibleAndOversizedSelections(t *testing.T) {
	store := learningStore(t)
	recordRun(t, store, "one", true, testProcedureRevision)
	recordRun(t, store, "two", true, "sha256:"+strings.Repeat("b", 64))
	if _, err := Build(store, []Selection{
		{RunID: "one", LearningKey: "a"},
		{RunID: "two", LearningKey: "b"},
	}); err == nil || !strings.Contains(err.Error(), "learning keys") {
		t.Fatalf("learning key error = %v", err)
	}
	if _, err := Build(store, []Selection{
		{RunID: "one", LearningKey: "a"},
		{RunID: "two", LearningKey: "a"},
	}); err == nil || !strings.Contains(err.Error(), "Procedure revisions") {
		t.Fatalf("Procedure error = %v", err)
	}
	if _, err := store.Record(trace.Input{
		RunID: "incomplete", Sequence: 0, Kind: "run",
		OccurredAt: "2026-07-29T12:00:00Z", Content: "started",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(store, []Selection{{RunID: "incomplete", LearningKey: "a"}}); err == nil ||
		!strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("incomplete error = %v", err)
	}
}

func TestProposePlansWithoutWritingAndRejectsStaleEvidence(t *testing.T) {
	traceRoot := t.TempDir()
	store, err := trace.New(trace.Config{Dir: traceRoot, AgentID: "agent"})
	if err != nil {
		t.Fatal(err)
	}
	recordRun(t, store, "run", true, testProcedureRevision)
	candidate, err := Build(store, []Selection{{RunID: "run", LearningKey: "keep bounds"}})
	if err != nil {
		t.Fatal(err)
	}
	vaultRoot := learningVault(t)
	input := ProposalInput{
		Candidate: candidate, Type: "Reflection",
		URI:   "gnosis://test/reflections/keep-bounds.md",
		Title: "Keep bounds", Statement: "Keep trace reads bounded.",
		ExpectedAbsent: true,
	}
	proposal, err := Propose(vaultRoot, store, input)
	if err != nil {
		t.Fatal(err)
	}
	if !proposal.Plan.Applicable || proposal.Plan.Operation != "create" ||
		!strings.Contains(proposal.Plan.Diff, "content_hash=") {
		t.Fatalf("proposal = %+v", proposal)
	}
	if _, err := os.Stat(filepath.Join(vaultRoot, "reflections", "keep-bounds.md")); !os.IsNotExist(err) {
		t.Fatalf("proposal wrote target: %v", err)
	}

	path := filepath.Join(
		traceRoot, hashPath("agent"), hashPath("run"), "2.json",
	)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record trace.Record
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	record.Content = "tampered"
	data, err = json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Propose(vaultRoot, store, input); err == nil {
		t.Fatal("stale proposal succeeded")
	}
}

func TestProposePreservesKnowledgePlanValidationAndConflictBehavior(t *testing.T) {
	store := learningStore(t)
	recordRun(t, store, "run", true, testProcedureRevision)
	candidate, err := Build(store, []Selection{{RunID: "run", LearningKey: "event"}})
	if err != nil {
		t.Fatal(err)
	}
	vaultRoot := learningVault(t)
	input := ProposalInput{
		Candidate: candidate, Type: "Event",
		URI: "gnosis://test/events/run.md", Title: "Run completed",
		Statement: "The selected run completed.", OccurredAt: "2026-07-29T12:00:00Z",
		ExpectedAbsent: true,
	}
	markdown, err := proposalMarkdown(input)
	if err != nil {
		t.Fatal(err)
	}
	writeLearningFile(t, vaultRoot, "events/run.md", markdown)
	page, err := vault.ReadPage(vaultRoot, input.URI)
	if err != nil {
		t.Fatal(err)
	}
	current := input
	current.ExpectedAbsent = false
	current.ExpectedRevision = page.Document.Revision
	noOp, err := Propose(vaultRoot, store, current)
	if err != nil {
		t.Fatal(err)
	}
	if !noOp.Plan.Applicable || noOp.Plan.Operation != "no-op" {
		t.Fatalf("no-op = %+v", noOp.Plan)
	}
	conflict, err := Propose(vaultRoot, store, input)
	if err != nil {
		t.Fatal(err)
	}
	if conflict.Plan.Applicable || conflict.Plan.Operation != "invalid" {
		t.Fatalf("conflict = %+v", conflict.Plan)
	}
	if _, err := Propose(vaultRoot, store, ProposalInput{
		Candidate: candidate, Type: "Concept", URI: input.URI,
		Title: input.Title, Statement: input.Statement, ExpectedAbsent: true,
	}); err == nil || !strings.Contains(err.Error(), "Event or Reflection") {
		t.Fatalf("type error = %v", err)
	}
}

func learningStore(t *testing.T) *trace.Store {
	t.Helper()
	store, err := trace.New(trace.Config{Dir: t.TempDir(), AgentID: "agent"})
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func recordRun(t *testing.T, store *trace.Store, runID string, success bool, revision string) {
	t.Helper()
	records := []trace.Input{
		{
			RunID: runID, Sequence: 0, Kind: "run",
			OccurredAt: "2026-07-29T12:00:00Z", Content: "started",
			Metadata: map[string]any{
				"procedure_uri": testProcedureURI, "procedure_revision": revision,
			},
		},
		{
			RunID: runID, Sequence: 1, Kind: "feedback",
			OccurredAt: "2026-07-29T12:01:00Z", Content: "assessed",
			KnowledgeURI: testProcedureURI, KnowledgeRevision: revision, Feedback: "helpful",
		},
		{
			RunID: runID, Sequence: 2, Kind: "outcome",
			OccurredAt: "2026-07-29T12:02:00Z", Content: "finished",
			Metadata: map[string]any{"success": success},
		},
	}
	for _, record := range records {
		if _, err := store.Record(record); err != nil {
			t.Fatal(err)
		}
	}
}

func learningVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeLearningFile(t, root, "gnosis.toml", `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false
`)
	return root
}

func writeLearningFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hashPath(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
