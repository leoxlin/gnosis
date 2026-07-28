package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	evidencecontext "gnosis/internal/evidencecontext"
	agentmemory "gnosis/internal/memory"
	"gnosis/internal/search"
	"gnosis/internal/vault"
)

const mcpPageResourceTemplate = "gnosis://{vault}/{+path}"
const maxMCPGraphNeighbors = 100

type emptyInput struct{}

type getConceptsInput struct {
	Type string `json:"type,omitempty" jsonschema:"exact concept type"`
}

type conceptsOutput struct {
	ConceptTypes []vault.ConceptTypeSummary `json:"concept_types,omitempty"`
	Type         string                     `json:"type,omitempty"`
	Concepts     []map[string]any           `json:"concepts,omitempty"`
}

type getPageInput struct {
	URI            string `json:"uri" jsonschema:"canonical gnosis URI"`
	ResolveCurrent bool   `json:"resolve_current,omitempty" jsonschema:"follow the bounded supersession chain"`
}

type searchKnowledgeInput struct {
	Question string `json:"question" jsonschema:"knowledge question"`
	Backend  string `json:"backend,omitempty" jsonschema:"retrieval backend: vector or lexical"`
	Top      *int   `json:"top,omitempty" jsonschema:"number of candidate pages"`
	MaxRead  *int   `json:"max_read,omitempty" jsonschema:"maximum pages to recommend reading"`
	Depth    *int   `json:"depth,omitempty" jsonschema:"maximum graph traversal depth"`
}

type addMemoryInput struct {
	Text string `json:"text" jsonschema:"durable memory text"`
}

type searchMemoryInput struct {
	Query string `json:"query" jsonschema:"memory search query"`
	Limit *int   `json:"limit,omitempty" jsonschema:"maximum memories to return, from 1 through 20"`
}

type traceGraphInput struct {
	URI       string   `json:"uri" jsonschema:"canonical source gnosis URI"`
	TargetURI string   `json:"target_uri,omitempty" jsonschema:"canonical target gnosis URI for path mode"`
	Direction string   `json:"direction,omitempty" jsonschema:"edge direction: out, in, or both"`
	Relations []string `json:"relations,omitempty" jsonschema:"relationship type filters"`
	Depth     *int     `json:"depth,omitempty" jsonschema:"maximum path depth"`
	Limit     *int     `json:"limit,omitempty" jsonschema:"maximum neighbor edges, from 1 through 100"`
}

type boundedGraphNeighbors struct {
	vault.GraphNeighbors
	Total        int      `json:"total"`
	Truncated    bool     `json:"truncated"`
	Continuation []string `json:"continuation,omitempty"`
}

type traceGraphOutput struct {
	Mode      string                 `json:"mode"`
	Neighbors *boundedGraphNeighbors `json:"neighbors,omitempty"`
	Path      *vault.GraphPath       `json:"path,omitempty"`
}

type getProceduresInput struct {
	URI  string   `json:"uri,omitempty" jsonschema:"canonical Procedure gnosis URI"`
	Tags []string `json:"tags,omitempty" jsonschema:"require all Procedure tags in discovery mode"`
}

type getProceduresOutput struct {
	Mode       string                   `json:"mode"`
	Procedures []map[string]any         `json:"procedures,omitempty"`
	Procedure  *vault.ProcessInvocation `json:"procedure,omitempty"`
}

func newMCPServer(vaultPath string) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "gnosis", Version: "0.1.0"},
		&mcp.ServerOptions{Capabilities: &mcp.ServerCapabilities{
			Resources: &mcp.ResourceCapabilities{},
		}},
	)
	addMCPResources(server, vaultPath)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_vaults",
		Description: "List the effective gnosis vaults",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, vault.VaultCatalog, error) {
		catalog, err := vault.Vaults(vaultPath)
		return nil, catalog, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_concepts",
		Description: "List concept types or records of one exact type",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input getConceptsInput) (*mcp.CallToolResult, conceptsOutput, error) {
		return getMCPConcepts(vaultPath, input.Type)
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_page",
		Description: "Read one exact gnosis page by canonical URI",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input getPageInput) (*mcp.CallToolResult, vault.Page, error) {
		page, err := vault.ReadPageWithOptions(
			vaultPath,
			input.URI,
			vault.ReadOptions{ResolveCurrent: input.ResolveCurrent},
		)
		return nil, page, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_knowledge",
		Description: "Search gnosis knowledge using vector or lexical retrieval",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input searchKnowledgeInput) (*mcp.CallToolResult, search.QueryResult, error) {
		result, err := searchMCPKnowledge(ctx, vaultPath, input)
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "trace_graph",
		Description: "Trace bounded graph neighbors or a path between canonical gnosis URIs",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input traceGraphInput) (*mcp.CallToolResult, traceGraphOutput, error) {
		result, err := traceMCPGraph(vaultPath, input)
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_procedures",
		Description: "Discover eligible Procedures by tags or read one exact validated contract",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input getProceduresInput) (*mcp.CallToolResult, getProceduresOutput, error) {
		result, err := getMCPProcedures(vaultPath, input)
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_evidence_context",
		Description: "Resolve bounded cited evidence without generating an answer",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input evidencecontext.Request) (*mcp.CallToolResult, evidencecontext.Result, error) {
		result, err := evidencecontext.Resolve(ctx, vaultPath, input)
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_memory",
		Description: "Store one durable memory in the configured user and agent scope",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input addMemoryInput) (*mcp.CallToolResult, agentmemory.Result, error) {
		client, err := agentmemory.NewFromEnv(vaultPath)
		if err != nil {
			return nil, agentmemory.Result{}, err
		}
		result, err := client.Add(ctx, input.Text)
		return nil, result, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_memory",
		Description: "Search durable memories in the configured user and agent scope",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input searchMemoryInput) (*mcp.CallToolResult, agentmemory.Result, error) {
		client, err := agentmemory.NewFromEnv(vaultPath)
		if err != nil {
			return nil, agentmemory.Result{}, err
		}
		result, err := client.Search(ctx, input.Query, input.Limit)
		return nil, result, err
	})
	return server
}

func traceMCPGraph(vaultPath string, input traceGraphInput) (traceGraphOutput, error) {
	input.URI = strings.TrimSpace(input.URI)
	input.TargetURI = strings.TrimSpace(input.TargetURI)
	if !vault.IsCanonicalURI(input.URI) {
		return traceGraphOutput{}, errors.New("trace graph: uri must be a canonical gnosis URI")
	}
	if input.Direction == "" {
		input.Direction = string(vault.DirectionBoth)
	}
	if err := validateDirection(input.Direction); err != nil {
		return traceGraphOutput{}, fmt.Errorf("trace graph: %w", err)
	}
	for _, relation := range input.Relations {
		if strings.TrimSpace(relation) == "" {
			return traceGraphOutput{}, errors.New("trace graph: relations must not contain empty values")
		}
	}

	if input.TargetURI == "" {
		if input.Depth != nil {
			return traceGraphOutput{}, errors.New("trace graph: depth is available only in path mode")
		}
		limit := maxMCPGraphNeighbors
		if input.Limit != nil {
			limit = *input.Limit
		}
		if limit < 1 || limit > maxMCPGraphNeighbors {
			return traceGraphOutput{}, fmt.Errorf(
				"trace graph: limit must be between 1 and %d", maxMCPGraphNeighbors,
			)
		}
		result, err := vault.TraceNeighbors(
			vaultPath, input.URI, vault.Direction(input.Direction), input.Relations,
		)
		if err != nil {
			return traceGraphOutput{}, err
		}
		total := len(result.Edges)
		bounded := boundedGraphNeighbors{GraphNeighbors: result, Total: total}
		if total > limit {
			bounded.Edges = bounded.Edges[:limit]
			bounded.Truncated = true
			bounded.Continuation = []string{
				"Refine direction or relations to continue with a deterministic subset.",
			}
		}
		return traceGraphOutput{Mode: "neighbors", Neighbors: &bounded}, nil
	}

	if !vault.IsCanonicalURI(input.TargetURI) {
		return traceGraphOutput{}, errors.New("trace graph: target_uri must be a canonical gnosis URI")
	}
	if input.Limit != nil {
		return traceGraphOutput{}, errors.New("trace graph: limit is available only in neighbor mode")
	}
	depth := 3
	if input.Depth != nil {
		depth = *input.Depth
	}
	if depth < 0 {
		return traceGraphOutput{}, errors.New("trace graph: depth must be zero or greater")
	}
	result, err := vault.TracePath(
		vaultPath,
		input.URI,
		input.TargetURI,
		vault.Direction(input.Direction),
		input.Relations,
		depth,
	)
	if err != nil {
		return traceGraphOutput{}, err
	}
	return traceGraphOutput{Mode: "path", Path: &result}, nil
}

func getMCPProcedures(vaultPath string, input getProceduresInput) (getProceduresOutput, error) {
	input.URI = strings.TrimSpace(input.URI)
	for index, tag := range input.Tags {
		input.Tags[index] = strings.TrimSpace(tag)
		if input.Tags[index] == "" {
			return getProceduresOutput{}, errors.New("get procedures: tags must not contain empty values")
		}
	}
	if input.URI == "" {
		catalog, err := vault.DiscoverProcesses(vaultPath, input.Tags)
		if err != nil {
			return getProceduresOutput{}, err
		}
		return getProceduresOutput{Mode: "discovery", Procedures: catalog["procedures"]}, nil
	}
	if !vault.IsCanonicalURI(input.URI) {
		return getProceduresOutput{}, errors.New("get procedures: uri must be a canonical gnosis URI")
	}
	if len(input.Tags) != 0 {
		return getProceduresOutput{}, errors.New("get procedures: tags are available only in discovery mode")
	}
	procedure, err := vault.InvokeProcess(vaultPath, input.URI)
	if err != nil {
		return getProceduresOutput{}, err
	}
	return getProceduresOutput{Mode: "contract", Procedure: &procedure}, nil
}

func addMCPResources(server *mcp.Server, vaultPath string) {
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: mcpPageResourceTemplate,
		Name:        "gnosis-page",
		Title:       "gnosis page",
		Description: "Read one effective gnosis page selected by its canonical URI",
		MIMEType:    vault.ResourceMediaType,
	}, func(_ context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		resource, err := vault.ReadResource(vaultPath, request.Params.URI)
		if errors.Is(err, vault.ErrPageNotFound) {
			return nil, mcp.ResourceNotFoundError(request.Params.URI)
		}
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
			URI:      resource.URI,
			MIMEType: resource.MediaType,
			Text:     resource.Markdown,
			Meta: mcp.Meta{
				"origin":   resource.Origin,
				"revision": resource.Revision,
			},
		}}}, nil
	})
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, request mcp.Request) (mcp.Result, error) {
			if method != "resources/list" {
				return next(ctx, method, request)
			}
			params := request.GetParams().(*mcp.ListResourcesParams)
			cursor := ""
			if params != nil {
				cursor = params.Cursor
			}
			page, err := vault.ListResourcePage(vaultPath, cursor)
			if errors.Is(err, vault.ErrInvalidResourceCursor) {
				return nil, &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: err.Error()}
			}
			if err != nil {
				return nil, err
			}
			result := &mcp.ListResourcesResult{
				Resources:  make([]*mcp.Resource, 0, len(page.Resources)),
				NextCursor: page.NextCursor,
			}
			for _, resource := range page.Resources {
				result.Resources = append(result.Resources, &mcp.Resource{
					URI:         resource.URI,
					Name:        resource.URI,
					Title:       resource.Title,
					Description: resource.Description,
					MIMEType:    resource.MediaType,
					Size:        resource.Size,
					Meta: mcp.Meta{
						"origin":   resource.Origin,
						"revision": resource.Revision,
					},
				})
			}
			return result, nil
		}
	})
}

func getMCPConcepts(vaultPath, conceptType string) (*mcp.CallToolResult, conceptsOutput, error) {
	conceptType = strings.TrimSpace(conceptType)
	if conceptType == "" {
		catalog, err := vault.Concepts(vaultPath, "")
		return nil, conceptsOutput{ConceptTypes: catalog.ConceptTypes}, err
	}

	types, err := vault.Concepts(vaultPath, "")
	if err != nil {
		return nil, conceptsOutput{}, err
	}
	found := false
	for _, summary := range types.ConceptTypes {
		if summary.Type == conceptType {
			found = true
			break
		}
	}
	if !found {
		return nil, conceptsOutput{}, fmt.Errorf("unknown concept type %q", conceptType)
	}
	records, err := vault.ConceptRecords(vaultPath, conceptType)
	if err != nil {
		return nil, conceptsOutput{}, err
	}
	return nil, conceptsOutput{Type: conceptType, Concepts: records["concepts"]}, nil
}

func searchMCPKnowledge(ctx context.Context, vaultPath string, input searchKnowledgeInput) (search.QueryResult, error) {
	options := search.QueryOptions{Top: 3, MaxRead: 3, MaxDepth: 3}
	if input.Top != nil {
		options.Top = *input.Top
	}
	if input.MaxRead != nil {
		options.MaxRead = *input.MaxRead
	}
	if input.Depth != nil {
		options.MaxDepth = *input.Depth
	}
	if err := validateQueryOptions(options.Top, options.MaxRead, options.MaxDepth); err != nil {
		return search.QueryResult{}, fmt.Errorf("search knowledge: %w", err)
	}

	backend := input.Backend
	if backend == "" {
		backend = "vector"
	}
	switch backend {
	case "vector":
		config, err := search.SemanticConfigFromEnv(vaultPath)
		if err != nil {
			return search.QueryResult{}, err
		}
		return search.QuerySemantic(ctx, vaultPath, input.Question, options, config)
	case "lexical":
		return search.QueryLexical(vaultPath, input.Question, options)
	default:
		return search.QueryResult{}, fmt.Errorf("search knowledge: unknown backend %q", backend)
	}
}

func newServeCommand(options *rootOptions) *cobra.Command {
	command := &cobra.Command{
		Use:     "serve",
		Short:   "Serve gnosis over a protocol transport",
		Args:    cobra.NoArgs,
		GroupID: "workspace",
		Example: "gnosis serve http --address 127.0.0.1:8080\n" +
			"gnosis serve mcp",
		RunE: func(_ *cobra.Command, _ []string) error {
			return newUsageError(errors.New("serve: missing transport"))
		},
	}
	command.AddCommand(newServeHTTPCommand(options), newServeMCPCommand(options))
	return command
}

func newServeMCPCommand(options *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "mcp [flags]",
		Short: "Serve gnosis tools over MCP stdio",
		Args:  cobra.NoArgs,
		Example: "gnosis serve mcp\n" +
			"gnosis --vault <path> serve mcp",
		RunE: func(command *cobra.Command, _ []string) error {
			return newMCPServer(options.vaultPath).Run(command.Context(), &mcp.StdioTransport{})
		},
	}
}
