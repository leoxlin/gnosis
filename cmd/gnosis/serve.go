package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"gnosis/internal/vault"
)

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
	URI string `json:"uri" jsonschema:"canonical gnosis URI"`
}

type searchKnowledgeInput struct {
	Question string `json:"question" jsonschema:"knowledge question"`
	Backend  string `json:"backend,omitempty" jsonschema:"retrieval backend: vector or lexical"`
	Top      *int   `json:"top,omitempty" jsonschema:"number of candidate pages"`
	MaxRead  *int   `json:"max_read,omitempty" jsonschema:"maximum pages to recommend reading"`
	Depth    *int   `json:"depth,omitempty" jsonschema:"maximum graph traversal depth"`
}

func newMCPServer(vaultPath string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "gnosis", Version: "0.1.0"}, nil)
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
		page, err := vault.ReadPage(vaultPath, input.URI)
		return nil, page, err
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_knowledge",
		Description: "Search gnosis knowledge using vector or lexical retrieval",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input searchKnowledgeInput) (*mcp.CallToolResult, vault.QueryResult, error) {
		result, err := searchMCPKnowledge(ctx, vaultPath, input)
		return nil, result, err
	})
	return server
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

func searchMCPKnowledge(ctx context.Context, vaultPath string, input searchKnowledgeInput) (vault.QueryResult, error) {
	options := vault.QueryOptions{Top: 3, MaxRead: 3, MaxDepth: 3}
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
		return vault.QueryResult{}, fmt.Errorf("search knowledge: %w", err)
	}

	backend := input.Backend
	if backend == "" {
		backend = "vector"
	}
	switch backend {
	case "vector":
		config, err := vault.SemanticConfigFromEnv(vaultPath)
		if err != nil {
			return vault.QueryResult{}, err
		}
		return vault.QuerySemanticKnowledge(ctx, vaultPath, input.Question, options, config)
	case "lexical":
		return vault.QueryKnowledge(vaultPath, input.Question, options)
	default:
		return vault.QueryResult{}, fmt.Errorf("search knowledge: unknown backend %q", backend)
	}
}

func newServeCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "serve",
		Short: "Serve gnosis over a protocol transport",
		Args:  cobra.NoArgs,
	}
	command.AddCommand(newServeMCPCommand())
	return command
}

func newServeMCPCommand() *cobra.Command {
	var vaultPath string
	command := &cobra.Command{
		Use:   "mcp [flags]",
		Short: "Serve read-only gnosis tools over MCP stdio",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			return newMCPServer(vaultPath).Run(command.Context(), &mcp.StdioTransport{})
		},
	}
	command.Flags().StringVar(&vaultPath, "vault", defaultVault, "path to the OKF vault")
	return command
}
