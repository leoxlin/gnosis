package vault

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/adrg/frontmatter"
	"go.yaml.in/yaml/v4"
)

const ProcedureType = "Procedure"

// OriginKind identifies where an effective document came from.
type OriginKind string

const (
	OriginLocal  OriginKind = "local"
	OriginImport OriginKind = "import"
	OriginBundle OriginKind = "bundle"
)

// Origin preserves the selected source behind an effective vault document.
type Origin struct {
	Vault      string     `json:"vault"`
	Kind       OriginKind `json:"kind"`
	Root       string     `json:"root,omitempty"`
	Path       string     `json:"path,omitempty"`
	Precedence int        `json:"precedence"`
}

// Edge is one directed relationship from a document to another effective URI.
type Edge struct {
	To       string `json:"to"`
	Relation string `json:"relation"`
	Raw      string `json:"raw,omitempty"`
	Source   string `json:"source,omitempty"`
}

// DocumentRef is the compact agent-facing representation of a document.
type DocumentRef struct {
	URI         string `json:"uri"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Origin      Origin `json:"origin"`
	Revision    string `json:"revision"`
}

// Ref returns the agent-facing identity for a document.
func (d Document) Ref() DocumentRef {
	return DocumentRef{
		URI:         d.URI,
		Type:        d.Type,
		Title:       d.Title,
		Description: d.Description,
		Origin:      d.Origin,
		Revision:    d.Revision,
	}
}

// Page is an exact vault page and its complete Markdown source.
type Page struct {
	Document DocumentRef `json:"document"`
	Markdown string      `json:"markdown"`
}

// ProcessSections are the canonical executable sections of a process record.
type ProcessSections struct {
	KnowledgeInputs string `json:"knowledge_inputs"`
	Process         string `json:"process"`
	Completion      string `json:"completion"`
}

// ProcessStep is one ordered stage of a multi-step procedure.
type ProcessStep struct {
	Number   int             `json:"number"`
	Name     string          `json:"name"`
	Sections ProcessSections `json:"sections"`
}

// ProcessSummary is a procedure descriptor for invocation.
type ProcessSummary struct {
	DocumentRef
	Invocation string   `json:"invocation"`
	Tags       []string `json:"tags"`
}

// GraphEdge is a resolved, directed edge with exact endpoint identities.
type GraphEdge struct {
	From     DocumentRef `json:"from"`
	To       DocumentRef `json:"to"`
	Relation string      `json:"relation"`
	Raw      string      `json:"raw,omitempty"`
	Source   string      `json:"source,omitempty"`
}

// ProcessInvocation binds one exact procedure revision for an agent to execute.
type ProcessInvocation struct {
	Process  ProcessSummary  `json:"process"`
	Sections ProcessSections `json:"sections,omitzero"`
	Steps    []ProcessStep   `json:"steps,omitempty"`
}

// Direction controls graph traversal without discarding edge direction.
type Direction string

const (
	DirectionOut  Direction = "out"
	DirectionIn   Direction = "in"
	DirectionBoth Direction = "both"
)

// GraphNeighbors is a bounded one-hop graph view.
type GraphNeighbors struct {
	Node      DocumentRef `json:"node"`
	Direction Direction   `json:"direction"`
	Edges     []GraphEdge `json:"edges"`
}

// PathStatus explains why a path is or is not present.
type PathStatus string

const (
	PathFound         PathStatus = "found"
	PathUnknownSource PathStatus = "unknown_source"
	PathUnknownTarget PathStatus = "unknown_target"
	PathDisconnected  PathStatus = "disconnected"
	PathDepthExceeded PathStatus = "depth_exceeded"
)

// GraphPath is an exact, typed traversal result.
type GraphPath struct {
	Status    PathStatus    `json:"status"`
	From      *DocumentRef  `json:"from,omitempty"`
	To        *DocumentRef  `json:"to,omitempty"`
	Direction Direction     `json:"direction"`
	MaxDepth  int           `json:"max_depth"`
	Nodes     []DocumentRef `json:"nodes"`
	Edges     []GraphEdge   `json:"edges"`
}

type relationshipSpec struct {
	Type   string `yaml:"type"`
	Target string `yaml:"target"`
}

// QueryKnowledge performs bounded live retrieval for the CLI.
func QueryKnowledge(root, question string, options QueryOptions) (QueryResult, error) {
	if strings.TrimSpace(question) == "" {
		return QueryResult{}, fmt.Errorf("question must not be empty")
	}
	source, err := NewSearchSource(root)
	if err != nil {
		return QueryResult{}, err
	}
	documents, err := source.Documents()
	if err != nil {
		return QueryResult{}, err
	}
	return New(documents).Query(question, options), nil
}

// ListPages returns every effective page in deterministic URI order.
func ListPages(root string) ([]DocumentRef, error) {
	source, err := NewSearchSource(root)
	if err != nil {
		return nil, err
	}
	pages, err := source.resolvedPages()
	if err != nil {
		return nil, err
	}
	result := make([]DocumentRef, 0, len(pages))
	for _, page := range pages {
		result = append(result, page.document.Ref())
	}
	return result, nil
}

// ReadPage reads one exact effective page by gnosis URI.
func ReadPage(root, selector string) (Page, error) {
	source, err := NewSearchSource(root)
	if err != nil {
		return Page{}, err
	}
	pages, err := source.resolvedPages()
	if err != nil {
		return Page{}, err
	}
	page, ok := selectPage(pages, selector)
	if !ok {
		return Page{}, fmt.Errorf("no document found with URI %q", selector)
	}
	markdown, err := renderDocumentLinks(page, pages)
	if err != nil {
		return Page{}, err
	}
	return Page{Document: page.document.Ref(), Markdown: markdown}, nil
}

// DiscoverProcesses returns procedure records under the discovery API's
// procedure-specific collection name.
func DiscoverProcesses(root string) (ConceptRecordCatalog, error) {
	catalog, err := ConceptRecords(root, ProcedureType)
	if err != nil {
		return nil, err
	}
	return ConceptRecordCatalog{"procedures": catalog["concepts"]}, nil
}

// InvokeProcess loads one exact process as an execution contract. It is read-only.
func InvokeProcess(root, selector string) (ProcessInvocation, error) {
	source, err := NewSearchSource(root)
	if err != nil {
		return ProcessInvocation{}, err
	}
	pages, err := source.resolvedPages()
	if err != nil {
		return ProcessInvocation{}, err
	}
	page, ok := selectPage(pages, selector)
	if !ok {
		return ProcessInvocation{}, fmt.Errorf("no process found with URI %q", selector)
	}
	if !isProcessType(page.document.Type) {
		return ProcessInvocation{}, fmt.Errorf("document %q has non-executable type %q", page.document.URI, page.document.Type)
	}
	summary, err := processSummary(page)
	if err != nil {
		return ProcessInvocation{}, err
	}
	sections, steps, problems := parseProcess(page.document.Body)
	if len(problems) > 0 {
		return ProcessInvocation{}, fmt.Errorf("process %q has invalid sections", page.document.URI)
	}
	return ProcessInvocation{
		Process:  summary,
		Sections: sections,
		Steps:    steps,
	}, nil
}

// TraceNeighbors returns exact typed edges adjacent to a selected document.
func TraceNeighbors(root, selector string, direction Direction, relations []string) (GraphNeighbors, error) {
	direction, err := normalizeDirection(direction)
	if err != nil {
		return GraphNeighbors{}, err
	}
	graph, pages, err := loadAgentGraph(root)
	if err != nil {
		return GraphNeighbors{}, err
	}
	page, ok := selectPage(pages, selector)
	if !ok {
		return GraphNeighbors{}, fmt.Errorf("no document found with URI %q", selector)
	}
	edges := graph.neighborEdges(page.document.URI, direction, relationSet(relations))
	return GraphNeighbors{
		Node:      page.document.Ref(),
		Direction: direction,
		Edges:     edges,
	}, nil
}

// TracePath finds a deterministic directed or bidirectional typed path.
func TracePath(root, fromSelector, toSelector string, direction Direction, relations []string, maxDepth int) (GraphPath, error) {
	direction, err := normalizeDirection(direction)
	if err != nil {
		return GraphPath{}, err
	}
	if maxDepth < 0 {
		return GraphPath{}, fmt.Errorf("max depth must be zero or greater")
	}
	graph, pages, err := loadAgentGraph(root)
	if err != nil {
		return GraphPath{}, err
	}
	result := GraphPath{
		Status:    PathDisconnected,
		Direction: direction,
		MaxDepth:  maxDepth,
		Nodes:     []DocumentRef{},
		Edges:     []GraphEdge{},
	}
	fromPage, ok := selectPage(pages, fromSelector)
	if !ok {
		result.Status = PathUnknownSource
		return result, nil
	}
	fromRef := fromPage.document.Ref()
	result.From = &fromRef
	toPage, ok := selectPage(pages, toSelector)
	if !ok {
		result.Status = PathUnknownTarget
		return result, nil
	}
	toRef := toPage.document.Ref()
	result.To = &toRef

	filter := relationSet(relations)
	nodes, edges, found := graph.findPath(fromPage.document.URI, toPage.document.URI, direction, filter, maxDepth)
	if found {
		result.Status = PathFound
		result.Nodes = graph.refs(nodes)
		result.Edges = edges
		return result, nil
	}
	if _, _, reachable := graph.findPath(fromPage.document.URI, toPage.document.URI, direction, filter, -1); reachable {
		result.Status = PathDepthExceeded
	}
	return result, nil
}

func isProcessType(value string) bool {
	return value == ProcedureType
}

func processSummary(page *searchPage) (ProcessSummary, error) {
	fields := frontmatterFields{}
	_, err := frontmatter.MustParse(strings.NewReader(string(page.data)), &fields, yamlFrontmatter)
	if err != nil {
		return ProcessSummary{}, frontmatterError(err)
	}
	_, _, problems := parseProcess(page.document.Body)
	if len(problems) > 0 {
		return ProcessSummary{}, fmt.Errorf("process %q has invalid structure: %s", page.document.URI, strings.Join(problems, "; "))
	}
	if strings.TrimSpace(page.document.Description) == "" {
		return ProcessSummary{}, fmt.Errorf("process %q missing non-empty description frontmatter", page.document.URI)
	}
	invocation, _ := frontmatterScalar(fields, "invocation")
	invocation = strings.TrimSpace(invocation)
	if invocation == "" {
		invocation = "model"
	}
	if !validProcessInvocation(invocation) {
		return ProcessSummary{}, fmt.Errorf("process %q has unsupported invocation %q", page.document.URI, invocation)
	}
	return ProcessSummary{
		DocumentRef: page.document.Ref(),
		Invocation:  invocation,
		Tags:        append([]string(nil), page.document.Tags...),
	}, nil
}

func parseProcess(body string) (ProcessSections, []ProcessStep, []string) {
	blocks := markdownSectionBlocks(body, 2)
	multiStep := false
	for _, block := range blocks {
		if _, _, ok := parseProcessStepHeading(block.Title); ok {
			multiStep = true
			break
		}
	}
	if !multiStep {
		sections, missing, duplicates := parseProcessSections(body)
		return sections, nil, processSectionProblems("", missing, duplicates)
	}

	problems := []string{}
	if len(blocks) < 2 {
		problems = append(problems, "multi-step procedure requires at least two steps")
	}
	steps := make([]ProcessStep, 0, len(blocks))
	names := make(map[string]struct{}, len(blocks))
	for index, block := range blocks {
		number, name, ok := parseProcessStepHeading(block.Title)
		if !ok {
			problems = append(problems, fmt.Sprintf("invalid multi-step heading %q; want %q", block.Title, "STEP <number> - <name>"))
			continue
		}
		expected := index + 1
		if number != expected {
			problems = append(problems, fmt.Sprintf("%s is out of order; expected STEP %d", block.Title, expected))
		}
		if _, exists := names[name]; exists {
			problems = append(problems, fmt.Sprintf("duplicate step name %q", name))
		}
		names[name] = struct{}{}
		sections, missing, duplicates := parseProcessSectionsAtLevel(block.Body, 3)
		problems = append(problems, processSectionProblems(block.Title, missing, duplicates)...)
		steps = append(steps, ProcessStep{Number: number, Name: name, Sections: sections})
	}
	return ProcessSections{}, steps, problems
}

func parseProcessStepHeading(title string) (int, string, bool) {
	rest, ok := strings.CutPrefix(title, "STEP ")
	if !ok {
		return 0, "", false
	}
	numberText, name, ok := strings.Cut(rest, " - ")
	number, err := strconv.Atoi(numberText)
	name = strings.TrimSpace(name)
	if !ok || err != nil || number < 1 || strconv.Itoa(number) != numberText || name == "" {
		return 0, "", false
	}
	return number, name, true
}

func parseProcessSections(body string) (ProcessSections, []string, []string) {
	return parseProcessSectionsAtLevel(body, 2)
}

func parseProcessSectionsAtLevel(body string, level int) (ProcessSections, []string, []string) {
	sections, duplicates := markdownSectionsAtLevel(body, level)
	required := []string{"Inputs", "Process", "Completion"}
	missing := []string{}
	for _, name := range required {
		if strings.TrimSpace(sections[name]) == "" {
			missing = append(missing, name)
		}
	}
	return ProcessSections{
		KnowledgeInputs: sections["Inputs"],
		Process:         sections["Process"],
		Completion:      sections["Completion"],
	}, missing, duplicates
}

func processSectionProblems(context string, missing, duplicates []string) []string {
	prefix := ""
	if context != "" {
		prefix = context + " "
	}
	problems := make([]string, 0, len(missing)+len(duplicates))
	for _, section := range missing {
		problems = append(problems, fmt.Sprintf("%smissing required section %q", prefix, section))
	}
	for _, section := range duplicates {
		problems = append(problems, fmt.Sprintf("%sduplicate process section %q", prefix, section))
	}
	return problems
}

func markdownSections(markdown string) (map[string]string, []string) {
	return markdownSectionsAtLevel(markdown, 2)
}

func markdownSectionsAtLevel(markdown string, level int) (map[string]string, []string) {
	sections := make(map[string]string)
	duplicates := []string{}
	for _, block := range markdownSectionBlocks(markdown, level) {
		if _, exists := sections[block.Title]; exists {
			duplicates = append(duplicates, block.Title)
			continue
		}
		sections[block.Title] = block.Body
	}
	sort.Strings(duplicates)
	return sections, duplicates
}

type markdownSection struct {
	Title string
	Body  string
}

func markdownSectionBlocks(markdown string, level int) []markdownSection {
	prefix := strings.Repeat("#", level) + " "
	blocks := []markdownSection{}
	current := ""
	var content []string
	inFence := false
	fence := ""
	flush := func() {
		if current != "" {
			blocks = append(blocks, markdownSection{Title: current, Body: strings.TrimSpace(strings.Join(content, "\n"))})
		}
	}
	lines := strings.Split(markdown, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			marker := trimmed[:3]
			if !inFence {
				inFence = true
				fence = marker
			} else if marker == fence {
				inFence = false
				fence = ""
			}
		}
		if !inFence && strings.HasPrefix(line, prefix) {
			flush()
			current = strings.TrimSpace(strings.TrimPrefix(line, prefix))
			content = nil
			continue
		}
		if current != "" {
			content = append(content, line)
		}
	}
	flush()
	return blocks
}

func relationshipSpecs(fields frontmatterFields) ([]relationshipSpec, error) {
	value, exists := fields["relationships"]
	if !exists || value == nil {
		return nil, nil
	}
	var specs []relationshipSpec
	encoded, err := yaml.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("frontmatter %q must be a sequence of type and target mappings: %w", "relationships", err)
	}
	if err := yaml.Unmarshal(encoded, &specs); err != nil {
		return nil, fmt.Errorf("frontmatter %q must be a sequence of type and target mappings: %w", "relationships", err)
	}
	for index, spec := range specs {
		if strings.TrimSpace(spec.Type) == "" {
			return nil, fmt.Errorf("relationships[%d] missing non-empty %q", index, "type")
		}
		if strings.TrimSpace(spec.Target) == "" {
			return nil, fmt.Errorf("relationships[%d] missing non-empty %q", index, "target")
		}
	}
	return specs, nil
}

func validProcessInvocation(value string) bool {
	return value == "model" || value == "explicit"
}

func selectPage(pages []*searchPage, selector string) (*searchPage, bool) {
	selector = strings.TrimSpace(selector)
	canonical, hasCanonicalURI := canonicalGnosisURI(selector)
	for _, page := range pages {
		if (hasCanonicalURI && page.document.URI == canonical) || page.document.URI == selector {
			return page, true
		}
	}
	return nil, false
}

func documentURI(vaultName, pagePath string) string {
	vaultName = strings.TrimSpace(vaultName)
	if vaultName == "" {
		vaultName = "default"
	}
	u := &url.URL{
		Scheme: "gnosis",
		Host:   vaultName,
		Path:   path.Join("/", pagePath),
	}
	return u.String()
}

// canonicalGnosisURI normalizes one authority-based gnosis page URI.
func canonicalGnosisURI(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "gnosis" || u.Host == "" {
		return "", false
	}

	vaultName := u.Host
	pagePath := strings.TrimPrefix(u.Path, "/")
	if strings.TrimSpace(pagePath) == "" {
		return "", false
	}
	return documentURI(vaultName, pagePath), true
}

func documentRevision(data []byte) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}

func normalizeDirection(direction Direction) (Direction, error) {
	if direction == "" {
		return DirectionBoth, nil
	}
	switch direction {
	case DirectionOut, DirectionIn, DirectionBoth:
		return direction, nil
	default:
		return "", fmt.Errorf("direction must be %q, %q, or %q", DirectionOut, DirectionIn, DirectionBoth)
	}
}

func relationSet(relations []string) map[string]struct{} {
	if len(relations) == 0 {
		return nil
	}
	result := make(map[string]struct{}, len(relations))
	for _, relation := range relations {
		relation = strings.TrimSpace(relation)
		if relation != "" {
			result[relation] = struct{}{}
		}
	}
	return result
}

type agentGraph struct {
	byURI    map[string]Document
	outgoing map[string][]GraphEdge
	incoming map[string][]GraphEdge
}

func loadAgentGraph(root string) (*agentGraph, []*searchPage, error) {
	source, err := NewSearchSource(root)
	if err != nil {
		return nil, nil, err
	}
	pages, err := source.resolvedPages()
	if err != nil {
		return nil, nil, err
	}
	return newAgentGraph(pages), pages, nil
}

func newAgentGraph(pages []*searchPage) *agentGraph {
	graph := &agentGraph{
		byURI:    make(map[string]Document, len(pages)),
		outgoing: make(map[string][]GraphEdge, len(pages)),
		incoming: make(map[string][]GraphEdge, len(pages)),
	}
	for _, page := range pages {
		graph.byURI[page.document.URI] = page.document
	}
	for _, page := range pages {
		for _, edge := range page.document.Edges {
			target, exists := graph.byURI[edge.To]
			if !exists {
				continue
			}
			resolved := GraphEdge{
				From:     page.document.Ref(),
				To:       target.Ref(),
				Relation: edge.Relation,
				Raw:      edge.Raw,
				Source:   edge.Source,
			}
			graph.outgoing[page.document.URI] = append(graph.outgoing[page.document.URI], resolved)
			graph.incoming[target.URI] = append(graph.incoming[target.URI], resolved)
		}
	}
	for uri := range graph.byURI {
		sortGraphEdges(graph.outgoing[uri])
		sortGraphEdges(graph.incoming[uri])
	}
	return graph
}

func sortGraphEdges(edges []GraphEdge) {
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From.URI != edges[j].From.URI {
			return edges[i].From.URI < edges[j].From.URI
		}
		if edges[i].To.URI != edges[j].To.URI {
			return edges[i].To.URI < edges[j].To.URI
		}
		if edges[i].Relation != edges[j].Relation {
			return edges[i].Relation < edges[j].Relation
		}
		return edges[i].Raw < edges[j].Raw
	})
}

func (g *agentGraph) neighborEdges(uri string, direction Direction, relations map[string]struct{}) []GraphEdge {
	result := []GraphEdge{}
	seen := make(map[string]struct{})
	add := func(edges []GraphEdge) {
		for _, edge := range edges {
			if !matchesRelation(edge.Relation, relations) {
				continue
			}
			key := edge.From.URI + "\x00" + edge.To.URI + "\x00" + edge.Relation
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, edge)
		}
	}
	if direction == DirectionOut || direction == DirectionBoth {
		add(g.outgoing[uri])
	}
	if direction == DirectionIn || direction == DirectionBoth {
		add(g.incoming[uri])
	}
	sortGraphEdges(result)
	return result
}

type graphArc struct {
	next string
	edge GraphEdge
}

type graphPredecessor struct {
	from string
	edge GraphEdge
}

func (g *agentGraph) arcs(uri string, direction Direction, relations map[string]struct{}) []graphArc {
	result := []graphArc{}
	if direction == DirectionOut || direction == DirectionBoth {
		for _, edge := range g.outgoing[uri] {
			if matchesRelation(edge.Relation, relations) {
				result = append(result, graphArc{next: edge.To.URI, edge: edge})
			}
		}
	}
	if direction == DirectionIn || direction == DirectionBoth {
		for _, edge := range g.incoming[uri] {
			if matchesRelation(edge.Relation, relations) {
				result = append(result, graphArc{next: edge.From.URI, edge: edge})
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].next != result[j].next {
			return result[i].next < result[j].next
		}
		return result[i].edge.Relation < result[j].edge.Relation
	})
	return result
}

func matchesRelation(relation string, allowed map[string]struct{}) bool {
	if len(allowed) == 0 {
		return true
	}
	_, exists := allowed[relation]
	return exists
}

func (g *agentGraph) findPath(source, target string, direction Direction, relations map[string]struct{}, maxDepth int) ([]string, []GraphEdge, bool) {
	if _, exists := g.byURI[source]; !exists {
		return nil, nil, false
	}
	if _, exists := g.byURI[target]; !exists {
		return nil, nil, false
	}
	if source == target {
		return []string{source}, []GraphEdge{}, true
	}
	type queueItem struct {
		uri   string
		depth int
	}
	queue := []queueItem{{uri: source}}
	visited := map[string]struct{}{source: {}}
	previous := make(map[string]graphPredecessor)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if maxDepth >= 0 && current.depth >= maxDepth {
			continue
		}
		for _, arc := range g.arcs(current.uri, direction, relations) {
			if _, exists := visited[arc.next]; exists {
				continue
			}
			visited[arc.next] = struct{}{}
			previous[arc.next] = graphPredecessor{from: current.uri, edge: arc.edge}
			if arc.next == target {
				return reconstructGraphPath(source, target, previous)
			}
			queue = append(queue, queueItem{uri: arc.next, depth: current.depth + 1})
		}
	}
	return nil, nil, false
}

func reconstructGraphPath(source, target string, previous map[string]graphPredecessor) ([]string, []GraphEdge, bool) {
	nodes := []string{target}
	edges := []GraphEdge{}
	current := target
	for current != source {
		entry, exists := previous[current]
		if !exists {
			return nil, nil, false
		}
		edges = append(edges, entry.edge)
		current = entry.from
		nodes = append(nodes, current)
	}
	for left, right := 0, len(nodes)-1; left < right; left, right = left+1, right-1 {
		nodes[left], nodes[right] = nodes[right], nodes[left]
	}
	for left, right := 0, len(edges)-1; left < right; left, right = left+1, right-1 {
		edges[left], edges[right] = edges[right], edges[left]
	}
	return nodes, edges, true
}

func (g *agentGraph) refs(uris []string) []DocumentRef {
	result := make([]DocumentRef, 0, len(uris))
	for _, uri := range uris {
		if document, exists := g.byURI[uri]; exists {
			result = append(result, document.Ref())
		}
	}
	return result
}
