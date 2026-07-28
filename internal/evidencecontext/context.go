package evidencecontext

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"gnosis/internal/search"
	"gnosis/internal/vault"
)

const (
	DefaultMaxEvidence = 5
	DefaultMaxChars    = 6_000
	DefaultMaxDepth    = 2
	MaxEvidence        = 20
	MaxChars           = 50_000
	MaxDepth           = 8
	maxCandidates      = 100
)

type Strategy string

const (
	StrategyLexical Strategy = "lexical"
	StrategyVector  Strategy = "vector"
	StrategyHybrid  Strategy = "hybrid"
)

type RelationshipConstraint struct {
	Type   string `json:"type"`
	Target string `json:"target"`
}

type Constraints struct {
	Type          string                  `json:"type,omitempty"`
	Status        string                  `json:"status,omitempty"`
	Tags          []string                `json:"tags,omitempty"`
	Source        string                  `json:"source,omitempty"`
	AsOf          string                  `json:"as_of,omitempty"`
	MinConfidence *float64                `json:"min_confidence,omitempty"`
	Tier          string                  `json:"tier,omitempty"`
	Relationship  *RelationshipConstraint `json:"relationship,omitempty"`
}

type Request struct {
	Question    string      `json:"question"`
	Strategy    Strategy    `json:"strategy,omitempty"`
	Constraints Constraints `json:"constraints,omitempty"`
	MaxEvidence *int        `json:"max_evidence,omitempty"`
	MaxChars    *int        `json:"max_chars,omitempty"`
	MaxDepth    *int        `json:"max_depth,omitempty"`
}

type RetrievalPass struct {
	Backend    string `json:"backend"`
	Candidates int    `json:"candidates"`
}

type Excerpt struct {
	Section   string `json:"section,omitempty"`
	Content   string `json:"content"`
	Truncated bool   `json:"truncated"`
}

type Citation struct {
	URI      string       `json:"uri"`
	Revision string       `json:"revision"`
	Origin   vault.Origin `json:"origin"`
}

type Evidence struct {
	Citation Citation              `json:"citation"`
	Type     string                `json:"type"`
	Title    string                `json:"title"`
	Trust    vault.TrustProjection `json:"trust"`
	Score    float64               `json:"score"`
	Excerpt  Excerpt               `json:"excerpt"`
}

type Gap struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
}

type Omission struct {
	URI    string `json:"uri"`
	Reason string `json:"reason"`
}

type Budget struct {
	MaxEvidence  int  `json:"max_evidence"`
	MaxChars     int  `json:"max_chars"`
	MaxDepth     int  `json:"max_depth"`
	UsedEvidence int  `json:"used_evidence"`
	UsedChars    int  `json:"used_chars"`
	Truncated    bool `json:"truncated"`
}

type Result struct {
	Question     string            `json:"question"`
	Strategy     Strategy          `json:"strategy"`
	Passes       []RetrievalPass   `json:"passes"`
	Evidence     []Evidence        `json:"evidence"`
	Paths        []vault.GraphPath `json:"paths"`
	Gaps         []Gap             `json:"gaps"`
	Omissions    []Omission        `json:"omissions"`
	Budget       Budget            `json:"budget"`
	Continuation []string          `json:"continuation,omitempty"`
}

type SemanticSearch func(
	context.Context,
	string,
	string,
	search.QueryOptions,
) (search.QueryResult, error)

func Defaults(request Request) Request {
	request.Question = strings.TrimSpace(request.Question)
	if request.Strategy == "" {
		request.Strategy = StrategyLexical
	}
	if request.MaxEvidence == nil {
		request.MaxEvidence = intPointer(DefaultMaxEvidence)
	}
	if request.MaxChars == nil {
		request.MaxChars = intPointer(DefaultMaxChars)
	}
	if request.MaxDepth == nil {
		request.MaxDepth = intPointer(DefaultMaxDepth)
	}
	return request
}

func Validate(request Request) error {
	if strings.TrimSpace(request.Question) == "" {
		return errors.New("question must not be empty")
	}
	switch request.Strategy {
	case StrategyLexical, StrategyVector, StrategyHybrid:
	default:
		return fmt.Errorf("strategy must be %q, %q, or %q", StrategyLexical, StrategyVector, StrategyHybrid)
	}
	if request.MaxEvidence == nil || *request.MaxEvidence < 1 || *request.MaxEvidence > MaxEvidence {
		return fmt.Errorf("max evidence must be between 1 and %d", MaxEvidence)
	}
	if request.MaxChars == nil || *request.MaxChars < 1 || *request.MaxChars > MaxChars {
		return fmt.Errorf("max chars must be between 1 and %d", MaxChars)
	}
	if request.MaxDepth == nil || *request.MaxDepth < 1 || *request.MaxDepth > MaxDepth {
		return fmt.Errorf("max depth must be between 1 and %d", MaxDepth)
	}
	for _, tag := range request.Constraints.Tags {
		if strings.TrimSpace(tag) == "" {
			return errors.New("tags must not contain empty values")
		}
	}
	if request.Constraints.MinConfidence != nil {
		value := *request.Constraints.MinConfidence
		if value < 0 || value > 1 {
			return errors.New("minimum confidence must be between 0 and 1")
		}
	}
	if request.Constraints.AsOf != "" {
		if _, err := time.Parse(time.RFC3339, request.Constraints.AsOf); err != nil {
			return errors.New("as-of must be an RFC3339 timestamp")
		}
	}
	if relationship := request.Constraints.Relationship; relationship != nil {
		if strings.TrimSpace(relationship.Type) == "" || strings.TrimSpace(relationship.Target) == "" {
			return errors.New("relationship type and target must not be empty")
		}
	}
	return nil
}

func Resolve(ctx context.Context, root string, request Request) (Result, error) {
	return ResolveWithSemantic(ctx, root, request, querySemantic)
}

func ResolveWithSemantic(
	ctx context.Context,
	root string,
	request Request,
	semantic SemanticSearch,
) (Result, error) {
	request = Defaults(request)
	if err := Validate(request); err != nil {
		return Result{}, fmt.Errorf("context knowledge: %w", err)
	}
	documents, err := vault.LoadDocuments(root)
	if err != nil {
		return Result{}, fmt.Errorf("context knowledge: load documents: %w", err)
	}
	byURI := make(map[string]vault.Document, len(documents))
	for _, document := range documents {
		byURI[document.URI] = document
	}

	options := search.QueryOptions{Top: maxCandidates, MaxRead: maxCandidates, MaxDepth: *request.MaxDepth}
	lexical, err := search.QueryLexical(root, request.Question, options)
	if err != nil {
		return Result{}, fmt.Errorf("context knowledge: lexical pass failed: %w", err)
	}
	passes := []RetrievalPass{{Backend: "lexical", Candidates: len(lexical.Candidates)}}
	ranked := rankedCandidates(request.Strategy, lexical.Candidates, nil)
	if request.Strategy != StrategyLexical {
		vector, err := semantic(ctx, root, request.Question, options)
		if err != nil {
			return Result{}, fmt.Errorf("context knowledge: vector pass failed: %w", err)
		}
		passes = append(passes, RetrievalPass{Backend: "vector", Candidates: len(vector.Candidates)})
		ranked = rankedCandidates(request.Strategy, lexical.Candidates, vector.Candidates)
	}

	result := Result{
		Question:     request.Question,
		Strategy:     request.Strategy,
		Passes:       passes,
		Evidence:     []Evidence{},
		Paths:        []vault.GraphPath{},
		Gaps:         []Gap{},
		Omissions:    []Omission{},
		Continuation: []string{},
		Budget: Budget{
			MaxEvidence: *request.MaxEvidence,
			MaxChars:    *request.MaxChars,
			MaxDepth:    *request.MaxDepth,
		},
	}
	for _, candidate := range ranked {
		document, exists := byURI[candidate.URI]
		if !exists {
			result.Omissions = appendOmission(result.Omissions, candidate.URI, "stale_candidate")
			continue
		}
		if reason := constraintOmission(document, request.Constraints); reason != "" {
			result.Omissions = appendOmission(result.Omissions, candidate.URI, reason)
			continue
		}
		if len(result.Evidence) >= *request.MaxEvidence {
			result.Omissions = appendOmission(result.Omissions, candidate.URI, "evidence_limit")
			continue
		}
		remaining := *request.MaxChars - result.Budget.UsedChars
		if remaining <= 0 {
			result.Omissions = appendOmission(result.Omissions, candidate.URI, "character_limit")
			continue
		}
		excerpt := selectExcerpt(document.Body, request.Question, remaining)
		result.Evidence = append(result.Evidence, Evidence{
			Citation: Citation{URI: document.URI, Revision: document.Revision, Origin: document.Origin},
			Type:     document.Type,
			Title:    document.Title,
			Trust:    document.Trust,
			Score:    candidate.rank,
			Excerpt:  excerpt,
		})
		result.Budget.UsedChars += utf8.RuneCountInString(excerpt.Content)
	}
	result.Budget.UsedEvidence = len(result.Evidence)
	result.Budget.Truncated = len(result.Omissions) > 0
	if result.Budget.Truncated {
		result.Continuation = []string{
			"Read an omitted or cited URI directly, or refine constraints to continue.",
		}
	}

	switch {
	case len(ranked) == 0:
		result.Gaps = append(result.Gaps, Gap{Kind: "no_match", Message: "No retrieval candidate matched the question."})
	case len(result.Evidence) == 0:
		result.Gaps = append(result.Gaps, Gap{Kind: "constraints_excluded", Message: "Retrieval candidates did not satisfy the requested constraints."})
	}
	addPaths(root, request, &result)
	return result, nil
}

func querySemantic(
	ctx context.Context,
	root, question string,
	options search.QueryOptions,
) (search.QueryResult, error) {
	config, err := search.SemanticConfigFromEnv(root)
	if err != nil {
		return search.QueryResult{}, err
	}
	return search.QuerySemantic(ctx, root, question, options, config)
}

type rankedCandidate struct {
	search.Candidate
	rank float64
}

func rankedCandidates(
	strategy Strategy,
	lexical, vector []search.Candidate,
) []rankedCandidate {
	type aggregate struct {
		candidate search.Candidate
		rank      float64
	}
	byURI := make(map[string]aggregate, len(lexical)+len(vector))
	add := func(candidates []search.Candidate, weight float64) {
		for index, candidate := range candidates {
			current := byURI[candidate.URI]
			if current.candidate.URI == "" {
				current.candidate = candidate
			}
			current.rank += weight / float64(60+index+1)
			byURI[candidate.URI] = current
		}
	}
	add(lexical, 1)
	if strategy == StrategyVector {
		add(vector, 100)
	} else if strategy == StrategyHybrid {
		add(vector, 1)
	}
	result := make([]rankedCandidate, 0, len(byURI))
	for _, candidate := range byURI {
		result = append(result, rankedCandidate{Candidate: candidate.candidate, rank: candidate.rank})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].rank != result[j].rank {
			return result[i].rank > result[j].rank
		}
		return result[i].URI < result[j].URI
	})
	return result
}

func constraintOmission(document vault.Document, constraints Constraints) string {
	if constraints.Type != "" && document.Type != constraints.Type {
		return "type"
	}
	if constraints.Status != "" && document.Trust.Status != constraints.Status {
		return "status"
	}
	for _, tag := range constraints.Tags {
		if !contains(document.Tags, strings.TrimSpace(tag)) {
			return "tag"
		}
	}
	if constraints.Source != "" && document.Trust.Source != constraints.Source {
		return "source"
	}
	if constraints.MinConfidence != nil &&
		(document.Trust.Confidence == nil || *document.Trust.Confidence < *constraints.MinConfidence) {
		return "confidence"
	}
	if constraints.Tier != "" && document.Trust.Tier != constraints.Tier {
		return "tier"
	}
	if constraints.AsOf != "" {
		asOf, _ := time.Parse(time.RFC3339, constraints.AsOf)
		applicable, known := applicableAt(document.Trust, asOf)
		if !known {
			return "time_undetermined"
		}
		if !applicable {
			return "time"
		}
	}
	if relationship := constraints.Relationship; relationship != nil {
		found := false
		for _, edge := range document.Edges {
			if edge.Relation == relationship.Type &&
				(edge.To == relationship.Target || edge.Raw == relationship.Target) {
				found = true
				break
			}
		}
		if !found {
			return "relationship"
		}
	}
	return ""
}

func applicableAt(trust vault.TrustProjection, asOf time.Time) (bool, bool) {
	if trust.ValidFrom != "" || trust.ValidUntil != "" {
		if trust.ValidFrom != "" {
			value, err := time.Parse(time.RFC3339, trust.ValidFrom)
			if err != nil {
				return false, false
			}
			if asOf.Before(value) {
				return false, true
			}
		}
		if trust.ValidUntil != "" {
			value, err := time.Parse(time.RFC3339, trust.ValidUntil)
			if err != nil {
				return false, false
			}
			if asOf.After(value) {
				return false, true
			}
		}
		return true, true
	}
	for _, raw := range []string{trust.ObservedAt, trust.OccurredAt} {
		if raw == "" {
			continue
		}
		value, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return false, false
		}
		return !value.After(asOf), true
	}
	return false, false
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func appendOmission(omissions []Omission, uri, reason string) []Omission {
	if len(omissions) >= maxCandidates {
		return omissions
	}
	return append(omissions, Omission{URI: uri, Reason: reason})
}

type markdownExcerpt struct {
	section string
	content string
	score   int
}

func selectExcerpt(body, question string, limit int) Excerpt {
	body = strings.TrimSpace(body)
	selected := markdownExcerpt{content: body}
	terms := questionTerms(question)
	for _, section := range markdownSections(body) {
		score := 0
		lower := strings.ToLower(section.content)
		for _, term := range terms {
			score += strings.Count(lower, term)
		}
		if score > selected.score {
			selected = section
			selected.score = score
		}
	}
	runes := []rune(selected.content)
	truncated := len(runes) > limit
	if truncated {
		runes = runes[:limit]
	}
	return Excerpt{Section: selected.section, Content: string(runes), Truncated: truncated}
}

func markdownSections(body string) []markdownExcerpt {
	lines := strings.Split(body, "\n")
	result := []markdownExcerpt{}
	start := -1
	title := ""
	flush := func(end int) {
		if start < 0 {
			return
		}
		result = append(result, markdownExcerpt{
			section: title,
			content: strings.TrimSpace(strings.Join(lines[start:end], "\n")),
		})
	}
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		heading := strings.TrimLeft(trimmed, "#")
		if heading == trimmed || !strings.HasPrefix(heading, " ") {
			continue
		}
		flush(index)
		start = index
		title = strings.TrimSpace(heading)
	}
	flush(len(lines))
	return result
}

func questionTerms(question string) []string {
	seen := map[string]struct{}{}
	terms := []string{}
	for _, field := range strings.Fields(strings.ToLower(question)) {
		term := strings.TrimFunc(field, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsNumber(r)
		})
		if len([]rune(term)) < 3 {
			continue
		}
		if _, exists := seen[term]; exists {
			continue
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
	}
	return terms
}

func addPaths(root string, request Request, result *Result) {
	if len(result.Evidence) < 2 {
		return
	}
	source := result.Evidence[0].Citation.URI
	relations := []string(nil)
	if request.Constraints.Relationship != nil {
		relations = []string{request.Constraints.Relationship.Type}
	}
	for _, evidence := range result.Evidence[1:] {
		target := evidence.Citation.URI
		path, err := vault.TracePath(
			root, source, target, vault.DirectionBoth, relations, *request.MaxDepth,
		)
		if err != nil {
			result.Gaps = append(result.Gaps, Gap{
				Kind: "path_error", Message: err.Error(), From: source, To: target,
			})
			continue
		}
		if path.Status == vault.PathFound {
			result.Paths = append(result.Paths, path)
			continue
		}
		result.Gaps = append(result.Gaps, Gap{
			Kind: string(path.Status), Message: "No typed path was found within the requested depth.",
			From: source, To: target,
		})
	}
}

func intPointer(value int) *int {
	return &value
}
