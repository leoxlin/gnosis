package search

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"gnosis/internal/vault"
)

const (
	defaultTop         = 3
	defaultMaxDepth    = 3
	maxDescriptionRune = 200
	bm25K1             = 1.2
	bm25B              = 0.75
)

type hit struct {
	document vault.Document
	score    float64
	exact    bool
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
	URI         string       `json:"uri"`
	Type        string       `json:"type"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Origin      vault.Origin `json:"origin"`
	Revision    string       `json:"revision"`
	Score       float64      `json:"score"`
}

// QueryResult is the stable response for CLI knowledge queries.
type QueryResult struct {
	AnswerType AnswerType  `json:"answer_type"`
	Candidates []Candidate `json:"candidates"`
	Path       []string    `json:"path"`
	ShouldRead []string    `json:"should_read"`
	IndexOnly  bool        `json:"index_only"`
}

// QueryLexical performs bounded live lexical retrieval.
func QueryLexical(root, question string, options QueryOptions) (QueryResult, error) {
	if strings.TrimSpace(question) == "" {
		return QueryResult{}, fmt.Errorf("question must not be empty")
	}
	documents, err := vault.LoadDocuments(root)
	if err != nil {
		return QueryResult{}, err
	}
	return newEngine(documents).query(question, options), nil
}

type field int

const (
	fieldTitle field = iota
	fieldAliases
	fieldTags
	fieldDescription
	fieldPath
	fieldType
	fieldBody
	fieldCount
)

var fieldWeights = [fieldCount]float64{
	fieldTitle:       5,
	fieldAliases:     5,
	fieldTags:        3,
	fieldDescription: 2,
	fieldPath:        2,
	fieldType:        1,
	fieldBody:        1,
}

type tokenField struct {
	frequency map[string]int
	length    int
}

type indexedDocument struct {
	document vault.Document
	fields   [fieldCount]tokenField
}

// engine is an immutable, in-memory BM25F-style retriever and link graph.
type engine struct {
	documents         []indexedDocument
	byURI             map[string]vault.Document
	documentFrequency map[string]int
	averageLength     [fieldCount]float64
	adjacency         map[string][]string
}

func newEngine(documents []vault.Document) *engine {
	documents = append([]vault.Document(nil), documents...)
	sort.Slice(documents, func(i, j int) bool { return documents[i].URI < documents[j].URI })

	engine := &engine{
		documents:         make([]indexedDocument, 0, len(documents)),
		byURI:             make(map[string]vault.Document, len(documents)),
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
			fieldPath:        document.Path,
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
		engine.byURI[document.URI] = document
	}

	for currentField := field(0); currentField < fieldCount; currentField++ {
		if len(documents) == 0 || totalLength[currentField] == 0 {
			engine.averageLength[currentField] = 1
			continue
		}
		engine.averageLength[currentField] = float64(totalLength[currentField]) / float64(len(documents))
	}

	engine.buildAdjacency()
	return engine
}

func (e *engine) search(query string, limit int) []hit {
	if limit <= 0 || len(e.documents) == 0 {
		return []hit{}
	}
	queryTokens := uniqueTokens(tokenize(query, true))
	if len(queryTokens) == 0 {
		queryTokens = uniqueTokens(tokenize(query, false))
	}
	if len(queryTokens) == 0 {
		return []hit{}
	}

	hits := make([]hit, 0, len(e.documents))
	for _, indexed := range e.documents {
		class := classifyMatch(query, indexed.document)
		score := e.score(indexed, queryTokens)
		if class == matchNone && score <= 0 {
			continue
		}
		hits = append(hits, hit{
			document: indexed.document,
			score:    score,
			exact:    class == matchExact,
			class:    class,
		})
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].class != hits[j].class {
			return hits[i].class > hits[j].class
		}
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].document.URI < hits[j].document.URI
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

func (e *engine) query(question string, options QueryOptions) QueryResult {
	options = normalizedOptions(options)
	answerType, endpoints := classifyQuestion(question)
	result := QueryResult{
		AnswerType: answerType,
		Candidates: []Candidate{},
		Path:       []string{},
		ShouldRead: []string{},
	}

	var hits []hit
	if answerType == AnswerPath && len(endpoints) == 2 {
		hits = e.pathCandidates(endpoints, options.Top)
		from := e.search(endpoints[0], 1)
		to := e.search(endpoints[1], 1)
		if len(from) > 0 && len(to) > 0 {
			result.Path = e.findPath(from[0].document.URI, to[0].document.URI, options.MaxDepth)
		}
	} else {
		hits = e.search(question, options.Top)
	}

	for _, hit := range hits {
		result.Candidates = append(result.Candidates, Candidate{
			URI:         hit.document.URI,
			Type:        hit.document.Type,
			Title:       hit.document.Title,
			Description: truncateRunes(hit.document.Description, maxDescriptionRune),
			Origin:      hit.document.Origin,
			Revision:    hit.document.Revision,
			Score:       roundScore(hit.score),
		})
	}

	if len(hits) == 0 {
		result.IndexOnly = true
		return result
	}
	if answerType == AnswerDirect && hits[0].exact && strings.TrimSpace(hits[0].document.Description) != "" {
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
		addShouldRead(candidate.URI)
	}
	return result
}

// findPath returns the deterministic shortest path between two document URIs,
// treating directed document links as traversable in both directions.
func (e *engine) findPath(source, target string, maxDepth int) []string {
	if maxDepth < 0 {
		return []string{}
	}
	if _, exists := e.byURI[source]; !exists {
		return []string{}
	}
	if _, exists := e.byURI[target]; !exists {
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

func (e *engine) score(document indexedDocument, terms []string) float64 {
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

func (e *engine) buildAdjacency() {
	sets := make(map[string]map[string]struct{}, len(e.byURI))
	for uri := range e.byURI {
		sets[uri] = make(map[string]struct{})
	}
	for _, document := range e.byURI {
		for _, target := range document.Links {
			if target == document.URI {
				continue
			}
			if _, exists := e.byURI[target]; !exists {
				continue
			}
			sets[document.URI][target] = struct{}{}
			sets[target][document.URI] = struct{}{}
		}
	}
	for uri, neighbors := range sets {
		for neighbor := range neighbors {
			e.adjacency[uri] = append(e.adjacency[uri], neighbor)
		}
		sort.Strings(e.adjacency[uri])
	}
}

func (e *engine) pathCandidates(endpoints []string, limit int) []hit {
	combined := make([]hit, 0, limit)
	seen := make(map[string]struct{})
	appendHits := func(hits []hit) {
		for _, hit := range hits {
			if len(combined) >= limit {
				return
			}
			if _, exists := seen[hit.document.URI]; exists {
				continue
			}
			seen[hit.document.URI] = struct{}{}
			combined = append(combined, hit)
		}
	}
	appendHits(e.search(endpoints[0], 1))
	appendHits(e.search(endpoints[1], 1))
	appendHits(e.search(strings.Join(endpoints, " "), limit))
	return combined
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

func classifyMatch(query string, document vault.Document) matchClass {
	queryCanonical := canonical(query)
	if queryCanonical == "" {
		return matchNone
	}

	paths := []string{
		document.Path,
		strings.TrimSuffix(document.Path, filepath.Ext(document.Path)),
		strings.TrimSuffix(filepath.Base(document.Path), filepath.Ext(document.Path)),
	}
	for _, value := range append(append([]string{document.Title}, document.Aliases...), paths...) {
		valueCanonical := canonical(value)
		if valueCanonical != "" && valueCanonical == queryCanonical {
			return matchExact
		}
	}
	for _, value := range append(append([]string{document.Title}, document.Aliases...), document.Path) {
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
