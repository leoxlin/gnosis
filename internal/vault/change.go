package vault

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v4"
)

// KnowledgeChangeInput is the complete stateless input for one page change.
type KnowledgeChangeInput struct {
	URI              string `json:"uri"`
	Candidate        string `json:"candidate"`
	ExpectedRevision string `json:"expected_revision,omitempty"`
	ExpectedAbsent   bool   `json:"expected_absent,omitempty"`
}

// KnowledgeChangeFinding describes one validation problem or warning.
type KnowledgeChangeFinding struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
}

// AffectedRelationship describes an outgoing relationship changed by a plan.
type AffectedRelationship struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Relation string `json:"relation"`
	Change   string `json:"change"`
}

// KnowledgeChangePlan is a reviewable, side-effect-free page mutation plan.
type KnowledgeChangePlan struct {
	URI                   string                   `json:"uri"`
	Operation             string                   `json:"operation"`
	Applicable            bool                     `json:"applicable"`
	ExpectedRevision      string                   `json:"expected_revision,omitempty"`
	ExpectedAbsent        bool                     `json:"expected_absent,omitempty"`
	BeforeRevision        string                   `json:"before_revision,omitempty"`
	AfterRevision         string                   `json:"after_revision,omitempty"`
	Diff                  string                   `json:"diff,omitempty"`
	Findings              []KnowledgeChangeFinding `json:"findings"`
	AffectedRelationships []AffectedRelationship   `json:"affected_relationships"`
	Digest                string                   `json:"digest"`
}

// KnowledgeChangeResult reports the actual mutation performed by apply.
type KnowledgeChangeResult struct {
	URI       string `json:"uri"`
	Operation string `json:"operation"`
	Path      string `json:"path,omitempty"`
	Revision  string `json:"revision"`
	Changed   bool   `json:"changed"`
}

// PlanKnowledgeChange validates and describes one page change without writing it.
func PlanKnowledgeChange(root string, input KnowledgeChangeInput) (KnowledgeChangePlan, error) {
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return KnowledgeChangePlan{}, fmt.Errorf("plan knowledge change: %w", err)
	}
	plan, _ := vault.planKnowledgeChange(input)
	return plan, nil
}

// ApplyKnowledgeChange refreshes, revalidates, and conditionally writes one accepted plan.
func ApplyKnowledgeChange(root string, input KnowledgeChangeInput, digest string) (KnowledgeChangeResult, error) {
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return KnowledgeChangeResult{}, fmt.Errorf("apply knowledge change: %w", err)
	}
	plan, prepared := vault.planKnowledgeChange(input)
	if strings.TrimSpace(digest) == "" || digest != plan.Digest {
		return KnowledgeChangeResult{}, fmt.Errorf(
			"apply knowledge change: plan digest changed: expected %q, actual %q; create a new plan",
			digest,
			plan.Digest,
		)
	}
	if !plan.Applicable || prepared == nil {
		return KnowledgeChangeResult{}, fmt.Errorf(
			"apply knowledge change: plan is not applicable: %s",
			findingSummary(plan.Findings),
		)
	}
	result := KnowledgeChangeResult{
		URI:       plan.URI,
		Operation: plan.Operation,
		Revision:  plan.AfterRevision,
		Changed:   plan.Operation != "no-op",
	}
	if !result.Changed {
		if current := currentPageForChange(prepared.pages, input.URI); current != nil {
			result.Path = current.path
		}
		return result, nil
	}
	path, err := vault.writePreparedDocument(*prepared)
	result.Path = path
	if err != nil {
		return result, fmt.Errorf(
			"apply knowledge change: %w; the local write remains at %s",
			err,
			prepared.destination,
		)
	}
	return result, nil
}

func (vault *effectiveVault) planKnowledgeChange(input KnowledgeChangeInput) (KnowledgeChangePlan, *preparedDocumentWrite) {
	plan := KnowledgeChangePlan{
		URI:                   input.URI,
		Operation:             "invalid",
		ExpectedRevision:      input.ExpectedRevision,
		ExpectedAbsent:        input.ExpectedAbsent,
		Findings:              []KnowledgeChangeFinding{},
		AffectedRelationships: []AffectedRelationship{},
		Digest:                knowledgeChangeDigest(input, vault),
	}
	if _, pagePath, ok := canonicalGnosisParts(input.URI); !ok {
		plan.Findings = append(plan.Findings, errorFinding("canonical_uri", "uri must be a canonical gnosis URI"))
		return plan, nil
	} else if isProjectedOpenSpecPath(pagePath) {
		plan.Findings = append(plan.Findings, errorFinding(
			"projected_openspec",
			"projected OpenSpec artifacts are read-only through gnosis",
		))
		return plan, nil
	}
	if strings.TrimSpace(input.Candidate) == "" {
		plan.Findings = append(plan.Findings, errorFinding(
			"physical_deletion_unsupported",
			"physical deletion is unsupported; submit a complete retained archive candidate",
		))
		return plan, nil
	}
	if input.ExpectedAbsent == (strings.TrimSpace(input.ExpectedRevision) != "") {
		plan.Findings = append(plan.Findings, errorFinding(
			"expected_state",
			"provide exactly one of expected_absent or expected_revision",
		))
		return plan, nil
	}

	pages, err := vault.pages()
	if err != nil {
		plan.Findings = append(plan.Findings, errorFinding("vault", err.Error()))
		return plan, nil
	}
	current := currentPageForChange(pages, input.URI)
	content, err := preserveUnknownMetadata(current, []byte(input.Candidate))
	if err != nil {
		plan.Findings = append(plan.Findings, errorFinding("candidate", err.Error()))
		return plan, nil
	}
	prepared, err := vault.prepareDocumentWrite(input.URI, content, current != nil)
	if err != nil {
		plan.Findings = append(plan.Findings, errorFinding("validation", err.Error()))
		return plan, nil
	}

	if current != nil {
		plan.BeforeRevision = current.document.Revision
	}
	plan.AfterRevision = prepared.candidate.document.Revision
	switch {
	case input.ExpectedAbsent && current != nil:
		plan.Findings = append(plan.Findings, errorFinding(
			"target_present",
			fmt.Sprintf("expected target absence, actual revision is %q", current.document.Revision),
		))
	case !input.ExpectedAbsent && current == nil:
		plan.Findings = append(plan.Findings, errorFinding(
			"target_absent",
			fmt.Sprintf("expected revision %q, but the target is absent", input.ExpectedRevision),
		))
	case current != nil && input.ExpectedRevision != current.document.Revision:
		plan.Findings = append(plan.Findings, errorFinding(
			"stale_revision",
			fmt.Sprintf("expected revision %q, actual revision is %q", input.ExpectedRevision, current.document.Revision),
		))
	}

	if err := resolveDocumentEdges(prepared.pages); err != nil {
		plan.Findings = append(plan.Findings, errorFinding("links", err.Error()))
	}
	beforeEdges := relationshipSet(prepared.pages)
	projected := replaceCandidatePage(prepared.pages, prepared.candidate)
	if err := resolveDocumentEdges(projected); err != nil {
		plan.Findings = append(plan.Findings, errorFinding("links", err.Error()))
	} else {
		plan.AffectedRelationships = changedRelationships(beforeEdges, relationshipSet(projected))
	}
	validation := Result{}
	validateFile(
		prepared.candidate.root,
		prepared.candidate.path,
		prepared.config,
		newDocumentResolver(projected),
		prepared.candidate,
		&validation,
	)
	sort.Strings(validation.Errors)
	sort.Strings(validation.Warnings)
	for _, problem := range validation.Errors {
		plan.Findings = append(plan.Findings, errorFinding("validation", problem))
	}
	for _, warning := range validation.Warnings {
		plan.Findings = append(plan.Findings, KnowledgeChangeFinding{
			Severity: "warning",
			Code:     "validation",
			Message:  warning,
		})
	}

	before := []byte(nil)
	if current != nil {
		before = current.data
	}
	plan.Diff = markdownDiff(before, content)
	switch {
	case hasErrorFinding(plan.Findings):
		plan.Operation = "invalid"
	case current == nil:
		plan.Operation = "create"
	case string(current.data) == string(content):
		plan.Operation = "no-op"
	case archivedCandidate(current, prepared.candidate):
		plan.Operation = "archive"
	default:
		plan.Operation = "update"
	}
	plan.Applicable = !hasErrorFinding(plan.Findings)
	return plan, &prepared
}

func currentPageForChange(pages []*effectivePage, uri string) *effectivePage {
	if page, ok := selectPage(pages, uri); ok {
		return page
	}
	_, pagePath, ok := canonicalGnosisParts(uri)
	if !ok {
		return nil
	}
	for _, page := range pages {
		if page.document.Path == pagePath {
			return page
		}
	}
	return nil
}

func preserveUnknownMetadata(current *effectivePage, candidate []byte) ([]byte, error) {
	if current == nil {
		return candidate, nil
	}
	parsed, err := parsePage(candidate)
	if err != nil {
		return nil, err
	}
	changed := false
	for key, value := range current.fields {
		if _, present := parsed.fields[key]; present || knownKnowledgeMetadata[key] {
			continue
		}
		parsed.fields[key] = value
		changed = true
	}
	if !changed {
		return candidate, nil
	}
	frontmatter, err := yaml.Marshal(parsed.fields)
	if err != nil {
		return nil, fmt.Errorf("preserve metadata: %w", err)
	}
	return []byte("---\n" + string(frontmatter) + "---\n" + parsed.body), nil
}

var knownKnowledgeMetadata = map[string]bool{
	"aliases": true, "applies_to": true, "claims": true, "confidence": true,
	"contradictions": true, "description": true, "effects": true, "entities": true,
	"invocation": true, "kind": true, "path": true, "relationships": true,
	"scope": true, "source": true, "status": true, "summary": true,
	"superseded_by": true, "tags": true, "tier": true, "title": true,
	"type": true, "valid_from": true,
}

func isProjectedOpenSpecPath(relativePath string) bool {
	parts := strings.Split(filepath.ToSlash(filepath.Clean(relativePath)), "/")
	if len(parts) == 4 && parts[0] == "openspec" && parts[1] == "specs" &&
		parts[2] != "" && parts[3] == "spec.md" {
		return true
	}
	if len(parts) < 4 || parts[0] != "openspec" || parts[1] != "changes" {
		return false
	}
	changeIndex := 2
	if parts[changeIndex] == "archive" {
		changeIndex++
	}
	if changeIndex >= len(parts) || parts[changeIndex] == "" {
		return false
	}
	artifact := parts[changeIndex+1:]
	if len(artifact) == 1 {
		return artifact[0] == "proposal.md" || artifact[0] == "design.md" || artifact[0] == "tasks.md"
	}
	return len(artifact) == 3 && artifact[0] == "specs" && artifact[1] != "" && artifact[2] == "spec.md"
}

func archivedCandidate(current, candidate *effectivePage) bool {
	before := strings.TrimSpace(fmt.Sprint(current.fields["status"]))
	after := strings.TrimSpace(fmt.Sprint(candidate.fields["status"]))
	return before != after && (after == "archived" || after == "retired")
}

type relationshipKey struct {
	from, to, relation string
}

func relationshipSet(pages []*effectivePage) map[relationshipKey]struct{} {
	result := map[relationshipKey]struct{}{}
	for _, page := range pages {
		for _, edge := range page.document.Edges {
			result[relationshipKey{from: page.document.URI, to: edge.To, relation: edge.Relation}] = struct{}{}
		}
	}
	return result
}

func replaceCandidatePage(pages []*effectivePage, candidate *effectivePage) []*effectivePage {
	result := make([]*effectivePage, 0, len(pages)+1)
	for _, page := range pages {
		if page.document.Path != candidate.document.Path {
			result = append(result, page)
		}
	}
	return append(result, candidate)
}

func changedRelationships(before, after map[relationshipKey]struct{}) []AffectedRelationship {
	result := []AffectedRelationship{}
	for key := range before {
		if _, exists := after[key]; !exists {
			result = append(result, AffectedRelationship{
				From: key.from, To: key.to, Relation: key.relation, Change: "removed",
			})
		}
	}
	for key := range after {
		if _, exists := before[key]; !exists {
			result = append(result, AffectedRelationship{
				From: key.from, To: key.to, Relation: key.relation, Change: "added",
			})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		left := result[i].Change + "\x00" + result[i].From + "\x00" + result[i].Relation + "\x00" + result[i].To
		right := result[j].Change + "\x00" + result[j].From + "\x00" + result[j].Relation + "\x00" + result[j].To
		return left < right
	})
	return result
}

func knowledgeChangeDigest(input KnowledgeChangeInput, vault *effectiveVault) string {
	hash := sha256.New()
	writeDigestPart := func(value string) {
		_ = binary.Write(hash, binary.BigEndian, uint64(len(value)))
		_, _ = hash.Write([]byte(value))
	}
	writeDigestPart("gnosis-knowledge-change-v1")
	writeDigestPart(input.URI)
	writeDigestPart(input.Candidate)
	writeDigestPart(input.ExpectedRevision)
	writeDigestPart(fmt.Sprintf("%t", input.ExpectedAbsent))
	configs := []Config{vault.config}
	for _, source := range vault.sources {
		configs = append(configs, source.config)
	}
	encoded, _ := json.Marshal(configs)
	writeDigestPart(string(encoded))
	return fmt.Sprintf("sha256:%x", hash.Sum(nil))
}

func markdownDiff(before, after []byte) string {
	oldLines := splitDiffLines(string(before))
	newLines := splitDiffLines(string(after))
	common := make([][]int, len(oldLines)+1)
	for i := range common {
		common[i] = make([]int, len(newLines)+1)
	}
	for i := len(oldLines) - 1; i >= 0; i-- {
		for j := len(newLines) - 1; j >= 0; j-- {
			if oldLines[i] == newLines[j] {
				common[i][j] = common[i+1][j+1] + 1
			} else if common[i+1][j] >= common[i][j+1] {
				common[i][j] = common[i+1][j]
			} else {
				common[i][j] = common[i][j+1]
			}
		}
	}
	var diff strings.Builder
	diff.WriteString("--- current\n+++ candidate\n")
	for i, j := 0, 0; i < len(oldLines) || j < len(newLines); {
		switch {
		case i < len(oldLines) && j < len(newLines) && oldLines[i] == newLines[j]:
			diff.WriteString(" " + oldLines[i] + "\n")
			i++
			j++
		case j == len(newLines) || i < len(oldLines) && common[i+1][j] >= common[i][j+1]:
			diff.WriteString("-" + oldLines[i] + "\n")
			i++
		default:
			diff.WriteString("+" + newLines[j] + "\n")
			j++
		}
	}
	return diff.String()
}

func splitDiffLines(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, "\n")
}

func errorFinding(code, message string) KnowledgeChangeFinding {
	return KnowledgeChangeFinding{Severity: "error", Code: code, Message: message}
}

func hasErrorFinding(findings []KnowledgeChangeFinding) bool {
	for _, finding := range findings {
		if finding.Severity == "error" {
			return true
		}
	}
	return false
}

func findingSummary(findings []KnowledgeChangeFinding) string {
	messages := []string{}
	for _, finding := range findings {
		if finding.Severity == "error" {
			messages = append(messages, finding.Message)
		}
	}
	if len(messages) == 0 {
		return "unknown validation failure"
	}
	return strings.Join(messages, "; ")
}
