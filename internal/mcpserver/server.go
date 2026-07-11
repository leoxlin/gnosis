// Package mcpserver projects the gnosis agent contract over MCP.
package mcpserver

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"gnosis/internal/vault"
)

const serverVersion = "0.1.0"

type discoverProcessesInput struct {
	Request string   `json:"request" jsonschema:"the agent request that needs a process"`
	Types   []string `json:"types,omitempty" jsonschema:"optional exact process types: Vault Process or Repository Process"`
	Limit   *int     `json:"limit,omitempty" jsonschema:"maximum process candidates to return"`
}

type invokeProcessInput struct {
	ID string `json:"id" jsonschema:"exact process ID or gnosis URI"`
}

type readPageInput struct {
	ID string `json:"id" jsonschema:"exact page ID or gnosis URI"`
}

type traceLinksInput struct {
	ID        string   `json:"id" jsonschema:"exact page ID or gnosis URI"`
	Direction string   `json:"direction,omitempty" jsonschema:"edge direction: out, in, or both"`
	Relations []string `json:"relations,omitempty" jsonschema:"optional relationship type filters"`
}

type findPathInput struct {
	From      string   `json:"from" jsonschema:"source page ID or gnosis URI"`
	To        string   `json:"to" jsonschema:"target page ID or gnosis URI"`
	Direction string   `json:"direction,omitempty" jsonschema:"edge direction: out, in, or both"`
	Relations []string `json:"relations,omitempty" jsonschema:"optional relationship type filters"`
	MaxDepth  *int     `json:"max_depth,omitempty" jsonschema:"maximum traversal depth"`
}

type queryKnowledgeInput struct {
	Question string `json:"question" jsonschema:"question to ask of the configured vault"`
	Top      *int   `json:"top,omitempty" jsonschema:"maximum candidate pages"`
	MaxRead  *int   `json:"max_read,omitempty" jsonschema:"maximum pages recommended for reading; zero disables page recommendations"`
	MaxDepth *int   `json:"max_depth,omitempty" jsonschema:"maximum natural-language path depth"`
}

// New constructs an MCP server bound to one configured vault view.
func New(root string) (*mcp.Server, error) {
	pages, err := vault.ListPages(root)
	if err != nil {
		return nil, err
	}
	processes, err := vault.DiscoverProcesses(root, "", nil, len(pages)+1)
	if err != nil {
		return nil, err
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "gnosis",
		Version: serverVersion,
	}, &mcp.ServerOptions{
		Instructions: "Discover a relevant process before process-governed work. Invoking a process only loads its exact execution contract; the agent executes it under current user and repository rules.",
	})

	registerTools(server, root)
	registerResources(server, root, pages)
	registerProcessPrompts(server, root, processes.Processes)
	return server, nil
}

// Serve runs the configured MCP server over standard input and output.
func Serve(ctx context.Context, root string) error {
	server, err := New(root)
	if err != nil {
		return err
	}
	return server.Run(ctx, &mcp.StdioTransport{})
}

func registerTools(server *mcp.Server, root string) {
	mcp.AddTool(server, readOnlyTool(
		"discover_processes",
		"Discover processes",
		"Rank only executable Vault Process and Repository Process records for an agent request.",
	), func(_ context.Context, _ *mcp.CallToolRequest, input discoverProcessesInput) (*mcp.CallToolResult, *vault.ProcessDiscovery, error) {
		limit := intValue(input.Limit, 5)
		if limit <= 0 {
			return nil, nil, fmt.Errorf("limit must be greater than zero")
		}
		result, err := vault.DiscoverProcesses(root, input.Request, input.Types, limit)
		return nil, &result, err
	})

	mcp.AddTool(server, readOnlyTool(
		"invoke_process",
		"Invoke process",
		"Load one exact process revision, its canonical sections, effects, provenance, and resolved relationships. This does not execute the workflow.",
	), func(_ context.Context, _ *mcp.CallToolRequest, input invokeProcessInput) (*mcp.CallToolResult, *vault.ProcessInvocation, error) {
		result, err := vault.InvokeProcess(root, input.ID)
		return nil, &result, err
	})

	mcp.AddTool(server, readOnlyTool(
		"read_page",
		"Read page",
		"Read one exact effective vault page by the ID or gnosis URI returned by another gnosis operation.",
	), func(_ context.Context, _ *mcp.CallToolRequest, input readPageInput) (*mcp.CallToolResult, *vault.Page, error) {
		result, err := vault.ReadPage(root, input.ID)
		return nil, &result, err
	})

	mcp.AddTool(server, readOnlyTool(
		"trace_links",
		"Trace links",
		"List exact typed inbound, outbound, or bidirectional links adjacent to one vault page.",
	), func(_ context.Context, _ *mcp.CallToolRequest, input traceLinksInput) (*mcp.CallToolResult, *vault.GraphNeighbors, error) {
		result, err := vault.TraceNeighbors(root, input.ID, vault.Direction(input.Direction), input.Relations)
		return nil, &result, err
	})

	mcp.AddTool(server, readOnlyTool(
		"find_path",
		"Find path",
		"Find a deterministic typed path between two exact vault pages and distinguish unknown, disconnected, and depth-limited results.",
	), func(_ context.Context, _ *mcp.CallToolRequest, input findPathInput) (*mcp.CallToolResult, *vault.GraphPath, error) {
		depth := intValue(input.MaxDepth, 3)
		if depth < 0 {
			return nil, nil, fmt.Errorf("max_depth must be zero or greater")
		}
		result, err := vault.TracePath(root, input.From, input.To, vault.Direction(input.Direction), input.Relations, depth)
		return nil, &result, err
	})

	mcp.AddTool(server, readOnlyTool(
		"query_knowledge",
		"Query knowledge",
		"Run bounded live retrieval and the natural-language graph pre-pass over the configured vault.",
	), func(_ context.Context, _ *mcp.CallToolRequest, input queryKnowledgeInput) (*mcp.CallToolResult, *vault.QueryResult, error) {
		options := vault.QueryOptions{
			Top:      intValue(input.Top, 3),
			MaxRead:  intValue(input.MaxRead, 3),
			MaxDepth: intValue(input.MaxDepth, 3),
		}
		if options.Top <= 0 {
			return nil, nil, fmt.Errorf("top must be greater than zero")
		}
		if options.MaxRead < 0 {
			return nil, nil, fmt.Errorf("max_read must be zero or greater")
		}
		if options.MaxDepth <= 0 {
			return nil, nil, fmt.Errorf("max_depth must be greater than zero")
		}
		result, err := vault.QueryKnowledge(root, input.Question, options)
		return nil, &result, err
	})
}

func intValue(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func registerResources(server *mcp.Server, root string, pages []vault.DocumentRef) {
	for _, page := range pages {
		server.AddResource(&mcp.Resource{
			URI:         page.URI,
			Name:        page.ID,
			Title:       page.Title,
			Description: page.Description,
			MIMEType:    "text/markdown",
		}, func(_ context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			result, err := vault.ReadPage(root, request.Params.URI)
			if err != nil {
				return nil, mcp.ResourceNotFoundError(request.Params.URI)
			}
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{{
					URI:      result.Document.URI,
					MIMEType: "text/markdown",
					Text:     result.Markdown,
				}},
			}, nil
		})
	}
}

func registerProcessPrompts(server *mcp.Server, root string, processes []vault.ProcessSummary) {
	for _, process := range processes {
		process := process
		server.AddPrompt(&mcp.Prompt{
			Name:        processPromptName(process),
			Title:       process.Title,
			Description: process.Description,
		}, func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			page, err := vault.ReadPage(root, process.URI)
			if err != nil {
				return nil, err
			}
			text := fmt.Sprintf("Apply the following exact gnosis process revision under current user and repository rules.\n\nURI: %s\nRevision: %s\n\n%s", page.Document.URI, page.Document.Revision, page.Markdown)
			return &mcp.GetPromptResult{
				Description: page.Document.Description,
				Messages: []*mcp.PromptMessage{{
					Role:    "user",
					Content: &mcp.TextContent{Text: text},
				}},
			}, nil
		})
	}
}

func readOnlyTool(name, title, description string) *mcp.Tool {
	closedWorld := false
	nonDestructive := false
	return &mcp.Tool{
		Name:        name,
		Title:       title,
		Description: description,
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint:    true,
			IdempotentHint:  true,
			DestructiveHint: &nonDestructive,
			OpenWorldHint:   &closedWorld,
		},
	}
}

var promptNameCharacters = regexp.MustCompile(`[^a-z0-9]+`)

func processPromptName(process vault.ProcessSummary) string {
	name := promptNameCharacters.ReplaceAllString(strings.ToLower(process.Title), "_")
	name = strings.Trim(name, "_")
	if name == "" {
		name = "process"
	}
	digest := sha256.Sum256([]byte(process.URI))
	return fmt.Sprintf("process_%s_%x", name, digest[:4])
}
