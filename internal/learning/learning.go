// Package learning builds deterministic, evidence-bound proposals from agent runs.
package learning

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"gnosis/internal/trace"
	"gnosis/internal/vault"
)

const (
	MaxRuns        = 20
	MaxEvidence    = 200
	MaxLearningKey = 256
	MaxTitle       = 256
	MaxStatement   = 64 * 1024
)

type Selection struct {
	RunID       string `json:"run_id"`
	LearningKey string `json:"learning_key"`
}

type EvidenceReference struct {
	AgentID     string `json:"agent_id"`
	RunID       string `json:"run_id"`
	Sequence    int64  `json:"sequence"`
	Kind        string `json:"kind"`
	ContentHash string `json:"content_hash"`
}

type Outcome struct {
	Evidence   EvidenceReference `json:"evidence"`
	Content    string            `json:"content"`
	Supporting bool              `json:"supporting"`
}

type Attribution struct {
	URI      string            `json:"uri"`
	Revision string            `json:"revision"`
	Feedback string            `json:"feedback,omitempty"`
	Evidence EvidenceReference `json:"evidence"`
}

type Candidate struct {
	AgentID           string              `json:"agent_id"`
	Selections        []Selection         `json:"selections"`
	LearningKey       string              `json:"learning_key"`
	ProcedureURI      string              `json:"procedure_uri,omitempty"`
	ProcedureRevision string              `json:"procedure_revision,omitempty"`
	Evidence          []EvidenceReference `json:"evidence"`
	Outcomes          []Outcome           `json:"outcomes"`
	Attributions      []Attribution       `json:"attributions"`
}

type ProposalInput struct {
	Candidate        Candidate `json:"candidate"`
	Type             string    `json:"type"`
	URI              string    `json:"uri"`
	Title            string    `json:"title"`
	Statement        string    `json:"statement"`
	OccurredAt       string    `json:"occurred_at,omitempty"`
	ExpectedRevision string    `json:"expected_revision,omitempty"`
	ExpectedAbsent   bool      `json:"expected_absent,omitempty"`
}

type Proposal struct {
	Candidate Candidate                 `json:"candidate"`
	Plan      vault.KnowledgeChangePlan `json:"plan"`
}

func Build(store *trace.Store, selections []Selection) (Candidate, error) {
	if len(selections) == 0 || len(selections) > MaxRuns {
		return Candidate{}, fmt.Errorf("build run learning: select between 1 and %d runs", MaxRuns)
	}
	selections = append([]Selection(nil), selections...)
	for index := range selections {
		selections[index].RunID = strings.TrimSpace(selections[index].RunID)
		selections[index].LearningKey = strings.TrimSpace(selections[index].LearningKey)
		if selections[index].RunID == "" {
			return Candidate{}, errors.New("build run learning: run_id must not be empty")
		}
		if selections[index].LearningKey == "" {
			return Candidate{}, errors.New("build run learning: learning_key must not be empty")
		}
		if len(selections[index].LearningKey) > MaxLearningKey {
			return Candidate{}, fmt.Errorf("build run learning: learning_key exceeds %d bytes", MaxLearningKey)
		}
	}
	sort.Slice(selections, func(i, j int) bool {
		if selections[i].RunID != selections[j].RunID {
			return selections[i].RunID < selections[j].RunID
		}
		return selections[i].LearningKey < selections[j].LearningKey
	})
	for index := 1; index < len(selections); index++ {
		if selections[index].RunID == selections[index-1].RunID {
			return Candidate{}, fmt.Errorf("build run learning: duplicate run_id %q", selections[index].RunID)
		}
		if selections[index].LearningKey != selections[0].LearningKey {
			return Candidate{}, errors.New("build run learning: selected runs have incompatible learning keys")
		}
	}

	candidate := Candidate{
		Selections: selections, LearningKey: selections[0].LearningKey,
		Evidence: []EvidenceReference{}, Outcomes: []Outcome{}, Attributions: []Attribution{},
	}
	for _, selection := range selections {
		run, err := store.Read(selection.RunID, trace.ReadOptions{
			MaxEntries: MaxEvidence + 1, MaxCharacters: trace.MaxReadCharacters,
		})
		if err != nil {
			return Candidate{}, fmt.Errorf("build run learning: run %q: %w", selection.RunID, err)
		}
		if !run.Complete {
			return Candidate{}, fmt.Errorf("build run learning: run %q is incomplete: %s", selection.RunID, diagnosticSummary(run))
		}
		if candidate.AgentID == "" {
			candidate.AgentID = run.AgentID
		}
		for _, record := range run.Entries {
			if len(candidate.Evidence) == MaxEvidence {
				return Candidate{}, fmt.Errorf("build run learning: selected evidence exceeds %d entries", MaxEvidence)
			}
			reference := evidenceReference(record)
			candidate.Evidence = append(candidate.Evidence, reference)
			if err := bindProcedure(&candidate, record); err != nil {
				return Candidate{}, fmt.Errorf("build run learning: run %q sequence %d: %w", selection.RunID, record.Sequence, err)
			}
			if record.Kind == "outcome" {
				success, ok := record.Metadata["success"].(bool)
				if !ok {
					return Candidate{}, fmt.Errorf(
						"build run learning: run %q outcome sequence %d requires boolean metadata.success",
						selection.RunID, record.Sequence,
					)
				}
				candidate.Outcomes = append(candidate.Outcomes, Outcome{
					Evidence: reference, Content: record.Content, Supporting: success,
				})
			}
			if record.Kind == "knowledge_use" || record.Kind == "feedback" {
				candidate.Attributions = append(candidate.Attributions, Attribution{
					URI: record.KnowledgeURI, Revision: record.KnowledgeRevision,
					Feedback: record.Feedback, Evidence: reference,
				})
			}
		}
	}
	if len(selections) > 1 && (candidate.ProcedureURI == "" || candidate.ProcedureRevision == "") {
		return Candidate{}, errors.New("build run learning: cross-run learning requires one Procedure URI and revision")
	}
	return candidate, nil
}

func Propose(vaultRoot string, store *trace.Store, input ProposalInput) (Proposal, error) {
	current, err := Build(store, input.Candidate.Selections)
	if err != nil {
		return Proposal{}, fmt.Errorf(
			"propose run learning: candidate evidence is stale; regenerate it: %w",
			err,
		)
	}
	if !equalCandidate(current, input.Candidate) {
		return Proposal{}, errors.New("propose run learning: candidate evidence is stale; regenerate it")
	}
	candidateMarkdown, err := proposalMarkdown(input)
	if err != nil {
		return Proposal{}, err
	}
	plan, err := vault.PlanKnowledgeChange(vaultRoot, vault.KnowledgeChangeInput{
		URI: input.URI, Candidate: candidateMarkdown,
		ExpectedRevision: input.ExpectedRevision, ExpectedAbsent: input.ExpectedAbsent,
	})
	if err != nil {
		return Proposal{}, err
	}
	return Proposal{Candidate: current, Plan: plan}, nil
}

func bindProcedure(candidate *Candidate, record trace.Record) error {
	uri, uriPresent, err := metadataString(record.Metadata, "procedure_uri")
	if err != nil {
		return err
	}
	revision, revisionPresent, err := metadataString(record.Metadata, "procedure_revision")
	if err != nil {
		return err
	}
	if !uriPresent && !revisionPresent {
		return nil
	}
	if !uriPresent || !revisionPresent || !vault.IsCanonicalURI(uri) || !trace.IsRevision(revision) {
		return errors.New("procedure_uri and procedure_revision must identify one exact Procedure revision")
	}
	if candidate.ProcedureURI == "" {
		candidate.ProcedureURI, candidate.ProcedureRevision = uri, revision
		return nil
	}
	if candidate.ProcedureURI != uri || candidate.ProcedureRevision != revision {
		return errors.New("selected evidence contains incompatible Procedure revisions")
	}
	return nil
}

func metadataString(metadata map[string]any, key string) (string, bool, error) {
	value, present := metadata[key]
	if !present {
		return "", false, nil
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) != text || text == "" {
		return "", true, fmt.Errorf("metadata.%s must be a non-empty string", key)
	}
	return text, true, nil
}

func evidenceReference(record trace.Record) EvidenceReference {
	return EvidenceReference{
		AgentID: record.AgentID, RunID: record.RunID, Sequence: record.Sequence,
		Kind: record.Kind, ContentHash: record.ContentHash,
	}
}

func diagnosticSummary(run trace.Run) string {
	messages := make([]string, 0, len(run.Diagnostics))
	for _, diagnostic := range run.Diagnostics {
		messages = append(messages, diagnostic.Code)
	}
	if run.Truncated {
		messages = append(messages, "truncated")
	}
	return strings.Join(messages, ", ")
}

func equalCandidate(left, right Candidate) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return string(leftJSON) == string(rightJSON)
}

func proposalMarkdown(input ProposalInput) (string, error) {
	input.Type = strings.TrimSpace(input.Type)
	input.URI = strings.TrimSpace(input.URI)
	input.Title = strings.TrimSpace(input.Title)
	input.Statement = strings.TrimSpace(input.Statement)
	input.OccurredAt = strings.TrimSpace(input.OccurredAt)
	if input.Type != "Event" && input.Type != "Reflection" {
		return "", errors.New("propose run learning: type must be Event or Reflection")
	}
	if !vault.IsCanonicalURI(input.URI) {
		return "", errors.New("propose run learning: uri must be a canonical gnosis URI")
	}
	if input.Title == "" || len(input.Title) > MaxTitle {
		return "", fmt.Errorf("propose run learning: title must contain between 1 and %d bytes", MaxTitle)
	}
	if input.Statement == "" || len(input.Statement) > MaxStatement {
		return "", fmt.Errorf("propose run learning: statement must contain between 1 and %d bytes", MaxStatement)
	}

	var builder strings.Builder
	builder.WriteString("---\ntype: " + input.Type + "\ntitle: " + strconv.Quote(input.Title) +
		"\ndescription: " + strconv.Quote(input.Statement) + "\n")
	if input.Type == "Event" {
		if _, err := time.Parse(time.RFC3339, input.OccurredAt); err != nil {
			return "", errors.New("propose run learning: occurred_at must be RFC 3339 for Event")
		}
		builder.WriteString("occurred_at: " + strconv.Quote(input.OccurredAt) +
			"\nactor: " + strconv.Quote(input.Candidate.AgentID) +
			"\nsource: agent-run trace\nstatus: recorded\n---\n\n# Event\n\n")
	} else {
		builder.WriteString("status: draft\n---\n\n# Reflection\n\n")
	}
	builder.WriteString(input.Statement + "\n\n# Evidence\n\n")
	for _, evidence := range input.Candidate.Evidence {
		fmt.Fprintf(
			&builder, "- agent=%q run=%q sequence=%d kind=%q content_hash=%q\n",
			evidence.AgentID, evidence.RunID, evidence.Sequence, evidence.Kind, evidence.ContentHash,
		)
	}
	if input.Type == "Reflection" {
		builder.WriteString("\n# Application\n\nLearning key: " +
			strconv.Quote(input.Candidate.LearningKey) + "\n")
	}
	return builder.String(), nil
}
