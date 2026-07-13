package vault

import (
	"fmt"
	"sort"
	"strings"
)

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

// GraphEdge is a resolved, directed edge with exact endpoint identities.
type GraphEdge struct {
	From     DocumentRef `json:"from"`
	To       DocumentRef `json:"to"`
	Relation string      `json:"relation"`
	Raw      string      `json:"raw,omitempty"`
	Source   string      `json:"source,omitempty"`
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
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return nil, err
	}
	pages, err := vault.pages()
	if err != nil {
		return nil, err
	}
	result := make([]DocumentRef, 0, len(pages))
	for _, page := range pages {
		result = append(result, page.document.Ref())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].URI < result[j].URI })
	return result, nil
}

// ReadPage reads one exact effective page by gnosis URI.
func ReadPage(root, selector string) (Page, error) {
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return Page{}, err
	}
	pages, err := vault.pages()
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

func selectPage(pages []*effectivePage, selector string) (*effectivePage, bool) {
	canonical, ok := canonicalGnosisURI(selector)
	if !ok {
		return nil, false
	}
	for _, page := range pages {
		if page.document.URI == canonical {
			return page, true
		}
	}
	return nil, false
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

func loadAgentGraph(root string) (*agentGraph, []*effectivePage, error) {
	vault, err := loadEffectiveVault(root)
	if err != nil {
		return nil, nil, err
	}
	pages, err := vault.resolvedPages()
	if err != nil {
		return nil, nil, err
	}
	return newAgentGraph(pages), pages, nil
}

func newAgentGraph(pages []*effectivePage) *agentGraph {
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
