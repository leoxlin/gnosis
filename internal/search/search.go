// Package search provides source-independent document retrieval and graph
// traversal primitives.
package search

import (
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const (
	defaultTop         = 3
	defaultMaxDepth    = 3
	maxDescriptionRune = 200
	bm25K1             = 1.2
	bm25B              = 0.75
)

// Document is the source-independent unit indexed by a Retriever. ID must be
// stable and unique within the loaded source set. Links contain other document
// IDs, not filesystem paths.
type Document struct {
	ID          string
	Title       string
	Description string
	Type        string
	Aliases     []string
	Tags        []string
	Body        string
	Links       []string
}

// Source loads documents for retrieval. Implementations may read vaults,
// worktrees, or other knowledge stores.
type Source interface {
	Documents() ([]Document, error)
}

// Retriever ranks documents without prescribing how they are stored or
// indexed.
type Retriever interface {
	Search(query string, limit int) []Hit
}

// Hit is a ranked document match.
type Hit struct {
	Document Document
	Score    float64
	Exact    bool
	class    matchClass
}

// AnswerType classifies how a query result should be interpreted.
type AnswerType string

const (
	AnswerDirect AnswerType = "direct"
	AnswerPath   AnswerType = "path"
	AnswerList   AnswerType = "list"
	AnswerGap    AnswerType = "gap"
)

// QueryOptions bounds result and graph traversal sizes.
type QueryOptions struct {
	Top      int
	MaxRead  int
	MaxDepth int
}

// Candidate is the compact, user-facing representation of a search hit.
type Candidate struct {
	Page        string  `json:"page"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
}

// Result is the stable response shared by query and graph-query.
type Result struct {
	AnswerType AnswerType  `json:"answer_type"`
	Candidates []Candidate `json:"candidates"`
	Path       []string    `json:"path"`
	ShouldRead []string    `json:"should_read"`
	IndexOnly  bool        `json:"index_only"`
}

type field int

const (
	fieldTitle field = iota
	fieldAliases
	fieldTags
	fieldDescription
	fieldID
	fieldType
	fieldBody
	fieldCount
)

var fieldWeights = [fieldCount]float64{
	fieldTitle:       5,
	fieldAliases:     5,
	fieldTags:        3,
	fieldDescription: 2,
	fieldID:          2,
	fieldType:        1,
	fieldBody:        1,
}

type tokenField struct {
	frequency map[string]int
	length    int
}

type indexedDocument struct {
	document Document
	fields   [fieldCount]tokenField
}

// Engine is an immutable, in-memory BM25F-style retriever and link graph.
type Engine struct {
	documents         []indexedDocument
	byID              map[string]Document
	documentFrequency map[string]int
	averageLength     [fieldCount]float64
	adjacency         map[string][]string
	retriever         Retriever
}

// New constructs a live in-memory index from documents.
func New(documents []Document) *Engine {
	documents = append([]Document(nil), documents...)
	sort.Slice(documents, func(i, j int) bool {
		return documents[i].ID < documents[j].ID
	})

	engine := &Engine{
		documents:         make([]indexedDocument, 0, len(documents)),
		byID:              make(map[string]Document, len(documents)),
		documentFrequency: make(map[string]int),
		adjacency:         make(map[string][]string, len(documents)),
	}

	var totalLength [fieldCount]int
	for _, document := range documents {
		indexed := indexedDocument{document: document}
		values := [fieldCount]string{
			fieldTitle:       document.Title,
			fieldAliases:     strings.Join(document.Aliases, " "),
			fieldTags:        strings.Join(document.Tags, " "),
			fieldDescription: document.Description,
			fieldID:          document.ID,
			fieldType:        document.Type,
			fieldBody:        document.Body,
		}

		seenTerms := make(map[string]struct{})
		for currentField, value := range values {
			tokens := tokenize(value, true)
			frequencies := make(map[string]int, len(tokens))
			for _, token := range tokens {
				frequencies[token]++
				seenTerms[token] = struct{}{}
			}
			indexed.fields[currentField] = tokenField{
				frequency: frequencies,
				length:    len(tokens),
			}
			totalLength[currentField] += len(tokens)
		}
		for term := range seenTerms {
			engine.documentFrequency[term]++
		}

		engine.documents = append(engine.documents, indexed)
		engine.byID[document.ID] = document
	}

	for currentField := field(0); currentField < fieldCount; currentField++ {
		if len(documents) == 0 || totalLength[currentField] == 0 {
			engine.averageLength[currentField] = 1
			continue
		}
		engine.averageLength[currentField] = float64(totalLength[currentField]) / float64(len(documents))
	}

	engine.buildAdjacency()
	engine.retriever = engine
	return engine
}

// NewWithRetriever constructs the document graph while delegating ranking to
// retriever. A nil retriever uses the built-in BM25F implementation.
func NewWithRetriever(documents []Document, retriever Retriever) *Engine {
	engine := New(documents)
	if retriever != nil {
		engine.retriever = retriever
	}
	return engine
}

// Search returns up to limit relevant documents ordered deterministically.
func (e *Engine) Search(query string, limit int) []Hit {
	if limit <= 0 || len(e.documents) == 0 {
		return []Hit{}
	}
	queryTokens := uniqueTokens(tokenize(query, true))
	if len(queryTokens) == 0 {
		queryTokens = uniqueTokens(tokenize(query, false))
	}
	if len(queryTokens) == 0 {
		return []Hit{}
	}

	hits := make([]Hit, 0, len(e.documents))
	for _, indexed := range e.documents {
		class := classifyMatch(query, indexed.document)
		score := e.score(indexed, queryTokens)
		if class == matchNone && score <= 0 {
			continue
		}
		hits = append(hits, Hit{
			Document: indexed.document,
			Score:    score,
			Exact:    class == matchExact,
			class:    class,
		})
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].class != hits[j].class {
			return hits[i].class > hits[j].class
		}
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		return hits[i].Document.ID < hits[j].Document.ID
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

// Query performs retrieval and optional bounded path traversal.
func (e *Engine) Query(question string, options QueryOptions) Result {
	options = normalizedOptions(options)
	answerType, endpoints := classifyQuestion(question)
	result := Result{
		AnswerType: answerType,
		Candidates: []Candidate{},
		Path:       []string{},
		ShouldRead: []string{},
	}

	var hits []Hit
	if answerType == AnswerPath && len(endpoints) == 2 {
		hits = e.pathCandidates(endpoints, options.Top)
		from := e.retrieve(endpoints[0], 1)
		to := e.retrieve(endpoints[1], 1)
		if len(from) > 0 && len(to) > 0 {
			result.Path = e.FindPath(from[0].Document.ID, to[0].Document.ID, options.MaxDepth)
		}
	} else {
		hits = e.retrieve(question, options.Top)
	}

	for _, hit := range hits {
		result.Candidates = append(result.Candidates, Candidate{
			Page:        hit.Document.ID,
			Title:       hit.Document.Title,
			Description: truncateRunes(hit.Document.Description, maxDescriptionRune),
			Score:       roundScore(hit.Score),
		})
	}

	if len(hits) == 0 {
		result.IndexOnly = true
		return result
	}
	if answerType == AnswerDirect && hits[0].Exact && strings.TrimSpace(hits[0].Document.Description) != "" {
		result.IndexOnly = true
		result.Candidates = result.Candidates[:1]
		return result
	}

	seen := make(map[string]struct{})
	addShouldRead := func(page string) {
		if len(result.ShouldRead) >= options.MaxRead {
			return
		}
		if _, exists := seen[page]; exists {
			return
		}
		seen[page] = struct{}{}
		result.ShouldRead = append(result.ShouldRead, page)
	}
	for _, page := range result.Path {
		addShouldRead(page)
	}
	for _, candidate := range result.Candidates {
		addShouldRead(candidate.Page)
	}
	return result
}

// FindPath returns the deterministic shortest path between two document IDs,
// treating directed document links as traversable in both directions.
func (e *Engine) FindPath(source, target string, maxDepth int) []string {
	if maxDepth < 0 {
		return []string{}
	}
	if _, exists := e.byID[source]; !exists {
		return []string{}
	}
	if _, exists := e.byID[target]; !exists {
		return []string{}
	}
	if source == target {
		return []string{source}
	}

	type queueItem struct {
		node string
		path []string
	}
	queue := []queueItem{{node: source, path: []string{source}}}
	visited := map[string]struct{}{source: {}}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		depth := len(current.path) - 1
		if depth >= maxDepth {
			continue
		}
		for _, neighbor := range e.adjacency[current.node] {
			if _, exists := visited[neighbor]; exists {
				continue
			}
			visited[neighbor] = struct{}{}
			path := append(append([]string(nil), current.path...), neighbor)
			if neighbor == target {
				return path
			}
			queue = append(queue, queueItem{node: neighbor, path: path})
		}
	}
	return []string{}
}

func (e *Engine) score(document indexedDocument, terms []string) float64 {
	documentCount := float64(len(e.documents))
	var score float64
	for _, term := range terms {
		documentFrequency := float64(e.documentFrequency[term])
		if documentFrequency == 0 {
			continue
		}
		idf := math.Log(1 + (documentCount-documentFrequency+0.5)/(documentFrequency+0.5))
		var weightedFrequency float64
		for currentField := field(0); currentField < fieldCount; currentField++ {
			frequency := document.fields[currentField].frequency[term]
			if frequency == 0 {
				continue
			}
			length := float64(document.fields[currentField].length)
			normalization := (1 - bm25B) + bm25B*(length/e.averageLength[currentField])
			weightedFrequency += fieldWeights[currentField] * float64(frequency) / normalization
		}
		if weightedFrequency > 0 {
			score += idf * ((bm25K1 + 1) * weightedFrequency) / (bm25K1 + weightedFrequency)
		}
	}
	return score
}

func (e *Engine) buildAdjacency() {
	sets := make(map[string]map[string]struct{}, len(e.byID))
	for id := range e.byID {
		sets[id] = make(map[string]struct{})
	}
	for _, document := range e.byID {
		for _, target := range document.Links {
			if target == document.ID {
				continue
			}
			if _, exists := e.byID[target]; !exists {
				continue
			}
			sets[document.ID][target] = struct{}{}
			sets[target][document.ID] = struct{}{}
		}
	}
	for id, neighbors := range sets {
		for neighbor := range neighbors {
			e.adjacency[id] = append(e.adjacency[id], neighbor)
		}
		sort.Strings(e.adjacency[id])
	}
}

func (e *Engine) pathCandidates(endpoints []string, limit int) []Hit {
	combined := make([]Hit, 0, limit)
	seen := make(map[string]struct{})
	appendHits := func(hits []Hit) {
		for _, hit := range hits {
			if len(combined) >= limit {
				return
			}
			if _, exists := seen[hit.Document.ID]; exists {
				continue
			}
			seen[hit.Document.ID] = struct{}{}
			combined = append(combined, hit)
		}
	}
	appendHits(e.retrieve(endpoints[0], 1))
	appendHits(e.retrieve(endpoints[1], 1))
	appendHits(e.retrieve(strings.Join(endpoints, " "), limit))
	return combined
}

func (e *Engine) retrieve(query string, limit int) []Hit {
	if e.retriever == nil {
		return []Hit{}
	}
	return e.retriever.Search(query, limit)
}

func normalizedOptions(options QueryOptions) QueryOptions {
	if options.Top <= 0 {
		options.Top = defaultTop
	}
	if options.MaxRead < 0 {
		options.MaxRead = 0
	}
	if options.MaxDepth <= 0 {
		options.MaxDepth = defaultMaxDepth
	}
	return options
}

type matchClass uint8

const (
	matchNone matchClass = iota
	matchTag
	matchPhrase
	matchExact
)

func classifyMatch(query string, document Document) matchClass {
	queryCanonical := canonical(query)
	if queryCanonical == "" {
		return matchNone
	}

	ids := []string{
		document.ID,
		strings.TrimSuffix(document.ID, filepath.Ext(document.ID)),
		strings.TrimSuffix(filepath.Base(document.ID), filepath.Ext(document.ID)),
	}
	for _, value := range append(append([]string{document.Title}, document.Aliases...), ids...) {
		valueCanonical := canonical(value)
		if valueCanonical != "" && valueCanonical == queryCanonical {
			return matchExact
		}
	}
	for _, value := range append(append([]string{document.Title}, document.Aliases...), document.ID) {
		valueCanonical := canonical(value)
		if valueCanonical != "" && strings.Contains(valueCanonical, queryCanonical) {
			return matchPhrase
		}
	}
	for _, tag := range document.Tags {
		if canonical(tag) == queryCanonical {
			return matchTag
		}
	}
	return matchNone
}

var pathPatterns = []struct {
	pattern *regexp.Regexp
	left    int
	right   int
}{
	{regexp.MustCompile(`(?i)^\s*how (is|are) (.+?) (connected|related|linked) to (.+?)\??\s*$`), 2, 4},
	{regexp.MustCompile(`(?i)^\s*how does (.+?) relate to (.+?)\??\s*$`), 1, 2},
	{regexp.MustCompile(`(?i)^\s*trace (the )?(chain|path) from (.+?) to (.+?)\??\s*$`), 3, 4},
	{regexp.MustCompile(`(?i)^\s*what connects (.+?) (to|and) (.+?)\??\s*$`), 1, 3},
}

func classifyQuestion(question string) (AnswerType, []string) {
	for _, candidate := range pathPatterns {
		matches := candidate.pattern.FindStringSubmatch(question)
		if len(matches) > candidate.right {
			return AnswerPath, []string{
				strings.TrimSpace(matches[candidate.left]),
				strings.TrimSpace(matches[candidate.right]),
			}
		}
	}
	lower := strings.ToLower(question)
	for _, marker := range []string{
		"what don't i know", "what do i not know", "what is missing",
		"what's missing", "what gaps", "open questions",
	} {
		if strings.Contains(lower, marker) {
			return AnswerGap, nil
		}
	}
	trimmed := strings.TrimSpace(lower)
	for _, marker := range []string{
		"list all", "show all", "find all", "give me all", "pages about",
	} {
		if strings.HasPrefix(trimmed, marker) {
			return AnswerList, nil
		}
	}
	return AnswerDirect, nil
}

var stopwords = map[string]struct{}{
	"a": {}, "about": {}, "all": {}, "an": {}, "and": {}, "are": {}, "as": {},
	"at": {}, "be": {}, "by": {}, "do": {}, "does": {}, "every": {}, "find": {},
	"for": {}, "from": {}, "give": {}, "how": {}, "i": {}, "in": {}, "is": {},
	"it": {}, "list": {}, "me": {}, "of": {}, "on": {}, "or": {}, "pages": {},
	"show": {}, "that": {}, "the": {}, "this": {}, "to": {}, "was": {}, "what": {},
	"when": {}, "where": {}, "which": {}, "who": {}, "why": {}, "with": {},
}

func tokenize(value string, removeStopwords bool) []string {
	value = strings.ToLower(value)
	raw := strings.FieldsFunc(value, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '-' && r != '_' && r != '/' && r != '.'
	})
	tokens := make([]string, 0, len(raw)*2)
	for _, token := range raw {
		token = strings.Trim(token, "-_/.")
		if token == "" {
			continue
		}
		if !removeStopwords || !isStopword(token) {
			tokens = append(tokens, token)
		}
		parts := strings.FieldsFunc(token, func(r rune) bool {
			return r == '-' || r == '_' || r == '/' || r == '.'
		})
		if len(parts) <= 1 {
			continue
		}
		for _, part := range parts {
			if part == "" || (removeStopwords && isStopword(part)) {
				continue
			}
			tokens = append(tokens, part)
		}
	}
	return tokens
}

func canonical(value string) string {
	value = strings.ToLower(value)
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if !isStopword(part) {
			filtered = append(filtered, part)
		}
	}
	if len(filtered) == 0 {
		filtered = parts
	}
	return strings.Join(filtered, " ")
}

func isStopword(token string) bool {
	_, exists := stopwords[token]
	return exists
}

func uniqueTokens(tokens []string) []string {
	result := make([]string, 0, len(tokens))
	seen := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		if _, exists := seen[token]; exists {
			continue
		}
		seen[token] = struct{}{}
		result = append(result, token)
	}
	return result
}

func truncateRunes(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if limit <= 1 {
		return "…"
	}
	return strings.TrimSpace(string(runes[:limit-1])) + "…"
}

func roundScore(score float64) float64 {
	return math.Round(score*100) / 100
}
