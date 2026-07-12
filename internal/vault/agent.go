package vault

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"path"
	"sort"
	"strings"
)

const GnosisProcessType = "Gnosis Process"

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

// Edge is one directed relationship from a document to another effective ID.
type Edge struct {
	To       string `json:"to"`
	Relation string `json:"relation"`
	Raw      string `json:"raw,omitempty"`
	Source   string `json:"source,omitempty"`
}

// DocumentRef is the stable, compact identity shared by agent-facing APIs.
type DocumentRef struct {
	ID          string `json:"id"`
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
		ID:          d.ID,
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

// ProcessSummary is a process descriptor for selection or invocation.
type ProcessSummary struct {
	DocumentRef
	UseWhen    []string `json:"use_when"`
	Invocation string   `json:"invocation"`
	Effects    []string `json:"effects"`
	Tags       []string `json:"tags"`
}

// ProcessDiscovery contains all model-invocable process candidates.
type ProcessDiscovery struct {
	Processes []ProcessSummary `json:"processes"`
}

// GraphEdge is a resolved, directed edge with exact endpoint identities.
type GraphEdge struct {
	From     DocumentRef `json:"from"`
	To       DocumentRef `json:"to"`
	Relation string      `json:"relation"`
	Raw      string      `json:"raw,omitempty"`
	Source   string      `json:"source,omitempty"`
}

// ProcessInvocation binds one exact process revision for an agent to execute.
type ProcessInvocation struct {
	Process       ProcessSummary  `json:"process"`
	Sections      ProcessSections `json:"sections"`
	Relationships []GraphEdge     `json:"relationships"`
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

// ListPages returns every effective page in deterministic ID order.
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

// ReadPage reads one exact effective page by ID or gnosis URI.
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
		return Page{}, fmt.Errorf("no document found with ID or URI %q", selector)
	}
	markdown, err := renderDocumentLinks(page, pages)
	if err != nil {
		return Page{}, err
	}
	return Page{Document: page.document.Ref(), Markdown: markdown}, nil
}

// DiscoverProcesses lists model-invocable process records in stable order.
// Explicit processes are intentionally excluded because a parent process must
// invoke them as part of its own execution contract.
func DiscoverProcesses(root string) (ProcessDiscovery, error) {
	source, err := NewSearchSource(root)
	if err != nil {
		return ProcessDiscovery{}, err
	}
	pages, err := source.resolvedPages()
	if err != nil {
		return ProcessDiscovery{}, err
	}
	processes := make([]ProcessSummary, 0, len(pages))
	for _, page := range pages {
		if !isProcessType(page.document.Type) {
			continue
		}
		if !source.resolution.Config.ProcessEnabled(page.document.Tags) {
			continue
		}
		summary, err := processSummary(page)
		if err != nil {
			return ProcessDiscovery{}, err
		}
		if summary.Invocation == "explicit" {
			continue
		}
		processes = append(processes, summary)
	}
	sort.Slice(processes, func(i, j int) bool { return processes[i].ID < processes[j].ID })
	return ProcessDiscovery{Processes: processes}, nil
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
		return ProcessInvocation{}, fmt.Errorf("no process found with ID or URI %q", selector)
	}
	if !isProcessType(page.document.Type) {
		return ProcessInvocation{}, fmt.Errorf("document %q has non-executable type %q", page.document.ID, page.document.Type)
	}
	summary, err := processSummary(page)
	if err != nil {
		return ProcessInvocation{}, err
	}
	sections, missing, duplicates := parseProcessSections(page.document.Body)
	if len(missing) > 0 || len(duplicates) > 0 {
		return ProcessInvocation{}, fmt.Errorf("process %q has invalid sections", page.document.ID)
	}
	graph := newAgentGraph(pages)
	relationships := graph.outgoing[page.document.ID]
	if relationships == nil {
		relationships = []GraphEdge{}
	}
	return ProcessInvocation{
		Process:       summary,
		Sections:      sections,
		Relationships: append([]GraphEdge(nil), relationships...),
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
		return GraphNeighbors{}, fmt.Errorf("no document found with ID or URI %q", selector)
	}
	edges := graph.neighborEdges(page.document.ID, direction, relationSet(relations))
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
	nodes, edges, found := graph.findPath(fromPage.document.ID, toPage.document.ID, direction, filter, maxDepth)
	if found {
		result.Status = PathFound
		result.Nodes = graph.refs(nodes)
		result.Edges = edges
		return result, nil
	}
	if _, _, reachable := graph.findPath(fromPage.document.ID, toPage.document.ID, direction, filter, -1); reachable {
		result.Status = PathDepthExceeded
	}
	return result, nil
}

func isProcessType(value string) bool {
	return value == GnosisProcessType
}

func processSummary(page *searchPage) (ProcessSummary, error) {
	fields, _, err := parseFrontmatter(string(page.data))
	if err != nil {
		return ProcessSummary{}, err
	}
	_, missing, duplicates := parseProcessSections(page.document.Body)
	if len(missing) > 0 {
		return ProcessSummary{}, fmt.Errorf("process %q missing required section(s): %s", page.document.ID, strings.Join(missing, ", "))
	}
	if len(duplicates) > 0 {
		return ProcessSummary{}, fmt.Errorf("process %q repeats section(s): %s", page.document.ID, strings.Join(duplicates, ", "))
	}
	if strings.TrimSpace(page.document.Description) == "" {
		return ProcessSummary{}, fmt.Errorf("process %q missing non-empty description frontmatter", page.document.ID)
	}
	useWhen, valid := fields.scalars("use_when")
	if !valid {
		return ProcessSummary{}, fmt.Errorf("process %q frontmatter %q must be a scalar or sequence of scalars", page.document.ID, "use_when")
	}
	if len(useWhen) == 0 {
		return ProcessSummary{}, fmt.Errorf("process %q must declare at least one non-empty %q frontmatter value", page.document.ID, "use_when")
	}
	invocation, _ := fields.scalar("invocation")
	invocation = strings.TrimSpace(invocation)
	if invocation == "" {
		invocation = "model"
	}
	if !validProcessInvocation(invocation) {
		return ProcessSummary{}, fmt.Errorf("process %q has unsupported invocation %q", page.document.ID, invocation)
	}
	effects, valid := fields.scalars("effects")
	if !valid {
		return ProcessSummary{}, fmt.Errorf("process %q frontmatter %q must be a scalar or sequence of scalars", page.document.ID, "effects")
	}
	if effects == nil {
		effects = []string{}
	}
	for _, effect := range effects {
		if !validProcessEffect(effect) {
			return ProcessSummary{}, fmt.Errorf("process %q has unsupported effect %q", page.document.ID, effect)
		}
	}
	return ProcessSummary{
		DocumentRef: page.document.Ref(),
		UseWhen:     useWhen,
		Invocation:  invocation,
		Effects:     effects,
		Tags:        append([]string(nil), page.document.Tags...),
	}, nil
}

func parseProcessSections(body string) (ProcessSections, []string, []string) {
	sections, duplicates := markdownSections(body)
	required := []string{"Knowledge inputs", "Process", "Completion"}
	missing := []string{}
	for _, name := range required {
		if strings.TrimSpace(sections[name]) == "" {
			missing = append(missing, name)
		}
	}
	return ProcessSections{
		KnowledgeInputs: sections["Knowledge inputs"],
		Process:         sections["Process"],
		Completion:      sections["Completion"],
	}, missing, duplicates
}

func markdownSections(markdown string) (map[string]string, []string) {
	sections := make(map[string]string)
	duplicates := []string{}
	current := ""
	lines := strings.Split(markdown, "\n")
	var content []string
	inFence := false
	fence := ""
	flush := func() {
		if current == "" {
			return
		}
		value := strings.TrimSpace(strings.Join(content, "\n"))
		if _, exists := sections[current]; exists {
			duplicates = append(duplicates, current)
			return
		}
		sections[current] = value
	}
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
		if !inFence && strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
			flush()
			current = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			content = nil
			continue
		}
		if current != "" {
			content = append(content, line)
		}
	}
	flush()
	sort.Strings(duplicates)
	return sections, duplicates
}

func relationshipSpecs(fields Frontmatter) ([]relationshipSpec, error) {
	node, exists := fields["relationships"]
	if !exists || node == nil {
		return nil, nil
	}
	node = resolveAlias(node)
	if node == nil {
		return nil, nil
	}
	var specs []relationshipSpec
	if err := node.Decode(&specs); err != nil {
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

func validProcessEffect(value string) bool {
	switch value {
	case "read", "vault-write", "workspace-write", "external":
		return true
	default:
		return false
	}
}

func selectPage(pages []*searchPage, selector string) (*searchPage, bool) {
	selector = strings.TrimSpace(selector)
	canonical, hasCanonicalURI := canonicalGnosisURI(selector)
	for _, page := range pages {
		if page.document.ID == selector || page.document.URI == selector || (hasCanonicalURI && page.document.URI == canonical) {
			return page, true
		}
	}
	return nil, false
}

func documentURI(vaultName, id string) string {
	vaultName = strings.TrimSpace(vaultName)
	if vaultName == "" {
		vaultName = "default"
	}
	u := &url.URL{
		Scheme: "gnosis",
		Host:   vaultName,
		Path:   path.Join("/", id),
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
	id := strings.TrimPrefix(u.Path, "/")
	if strings.TrimSpace(id) == "" {
		return "", false
	}
	return documentURI(vaultName, id), true
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
	byID     map[string]Document
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
		byID:     make(map[string]Document, len(pages)),
		outgoing: make(map[string][]GraphEdge, len(pages)),
		incoming: make(map[string][]GraphEdge, len(pages)),
	}
	for _, page := range pages {
		graph.byID[page.document.ID] = page.document
	}
	for _, page := range pages {
		for _, edge := range page.document.Edges {
			target, exists := graph.byID[edge.To]
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
			graph.outgoing[page.document.ID] = append(graph.outgoing[page.document.ID], resolved)
			graph.incoming[target.ID] = append(graph.incoming[target.ID], resolved)
		}
	}
	for id := range graph.byID {
		sortGraphEdges(graph.outgoing[id])
		sortGraphEdges(graph.incoming[id])
	}
	return graph
}

func sortGraphEdges(edges []GraphEdge) {
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From.ID != edges[j].From.ID {
			return edges[i].From.ID < edges[j].From.ID
		}
		if edges[i].To.ID != edges[j].To.ID {
			return edges[i].To.ID < edges[j].To.ID
		}
		if edges[i].Relation != edges[j].Relation {
			return edges[i].Relation < edges[j].Relation
		}
		return edges[i].Raw < edges[j].Raw
	})
}

func (g *agentGraph) neighborEdges(id string, direction Direction, relations map[string]struct{}) []GraphEdge {
	result := []GraphEdge{}
	seen := make(map[string]struct{})
	add := func(edges []GraphEdge) {
		for _, edge := range edges {
			if !matchesRelation(edge.Relation, relations) {
				continue
			}
			key := edge.From.ID + "\x00" + edge.To.ID + "\x00" + edge.Relation
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, edge)
		}
	}
	if direction == DirectionOut || direction == DirectionBoth {
		add(g.outgoing[id])
	}
	if direction == DirectionIn || direction == DirectionBoth {
		add(g.incoming[id])
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

func (g *agentGraph) arcs(id string, direction Direction, relations map[string]struct{}) []graphArc {
	result := []graphArc{}
	if direction == DirectionOut || direction == DirectionBoth {
		for _, edge := range g.outgoing[id] {
			if matchesRelation(edge.Relation, relations) {
				result = append(result, graphArc{next: edge.To.ID, edge: edge})
			}
		}
	}
	if direction == DirectionIn || direction == DirectionBoth {
		for _, edge := range g.incoming[id] {
			if matchesRelation(edge.Relation, relations) {
				result = append(result, graphArc{next: edge.From.ID, edge: edge})
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
	if _, exists := g.byID[source]; !exists {
		return nil, nil, false
	}
	if _, exists := g.byID[target]; !exists {
		return nil, nil, false
	}
	if source == target {
		return []string{source}, []GraphEdge{}, true
	}
	type queueItem struct {
		id    string
		depth int
	}
	queue := []queueItem{{id: source}}
	visited := map[string]struct{}{source: {}}
	previous := make(map[string]graphPredecessor)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if maxDepth >= 0 && current.depth >= maxDepth {
			continue
		}
		for _, arc := range g.arcs(current.id, direction, relations) {
			if _, exists := visited[arc.next]; exists {
				continue
			}
			visited[arc.next] = struct{}{}
			previous[arc.next] = graphPredecessor{from: current.id, edge: arc.edge}
			if arc.next == target {
				return reconstructGraphPath(source, target, previous)
			}
			queue = append(queue, queueItem{id: arc.next, depth: current.depth + 1})
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

func (g *agentGraph) refs(ids []string) []DocumentRef {
	result := make([]DocumentRef, 0, len(ids))
	for _, id := range ids {
		if document, exists := g.byID[id]; exists {
			result = append(result, document.Ref())
		}
	}
	return result
}
