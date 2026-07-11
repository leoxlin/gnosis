package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"gnosis/internal/vault"
)

func TestServerExposesAgentToolsResourcesAndPrompts(t *testing.T) {
	root := mcpTestVault(t)
	server, err := New(root)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "gnosis-test", Version: "0.1.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })

	toolResult, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(toolResult.Tools))
	for _, tool := range toolResult.Tools {
		names = append(names, tool.Name)
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint || !tool.Annotations.IdempotentHint || tool.Annotations.DestructiveHint == nil || *tool.Annotations.DestructiveHint || tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint {
			t.Fatalf("tool %q is not marked read-only: %+v", tool.Name, tool.Annotations)
		}
		if tool.OutputSchema == nil {
			t.Fatalf("tool %q has no output schema", tool.Name)
		}
	}
	sort.Strings(names)
	wantTools := []string{"discover_processes", "find_path", "invoke_process", "query_knowledge", "read_page", "trace_links"}
	if strings.Join(names, ",") != strings.Join(wantTools, ",") {
		t.Fatalf("tools = %v, want %v", names, wantTools)
	}

	call, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "discover_processes",
		Arguments: map[string]any{
			"request": "answer from recorded knowledge",
			"types":   []string{"Vault Process"},
			"limit":   3,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	discovery := decodeToolResult[vault.ProcessDiscovery](t, call)
	if len(discovery.Processes) != 1 || discovery.Processes[0].URI == "" {
		t.Fatalf("discovery = %+v", discovery)
	}

	invoked, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "invoke_process",
		Arguments: map[string]any{
			"id": discovery.Processes[0].URI,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	invocation := decodeToolResult[vault.ProcessInvocation](t, invoked)
	if invocation.Process.URI != discovery.Processes[0].URI || invocation.Sections.Completion == "" || len(invocation.Relationships) != 1 {
		t.Fatalf("invocation = %+v", invocation)
	}

	readPage, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "read_page",
		Arguments: map[string]any{
			"id": discovery.Processes[0].URI,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	page := decodeToolResult[vault.Page](t, readPage)
	if page.Document.URI != discovery.Processes[0].URI || !strings.Contains(page.Markdown, "# query-vault") {
		t.Fatalf("page = %+v", page)
	}

	traced, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "trace_links",
		Arguments: map[string]any{
			"id":        discovery.Processes[0].URI,
			"direction": "out",
			"relations": []string{"uses"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	neighbors := decodeToolResult[vault.GraphNeighbors](t, traced)
	if len(neighbors.Edges) != 1 || neighbors.Edges[0].To.ID != "concepts/provenance.md" {
		t.Fatalf("neighbors = %+v", neighbors)
	}

	pathCall, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "find_path",
		Arguments: map[string]any{
			"from":      discovery.Processes[0].URI,
			"to":        "concepts/provenance.md",
			"direction": "out",
			"relations": []string{"uses"},
			"max_depth": 1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	graphPath := decodeToolResult[vault.GraphPath](t, pathCall)
	if graphPath.Status != vault.PathFound || len(graphPath.Edges) != 1 {
		t.Fatalf("path = %+v", graphPath)
	}

	query, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "query_knowledge",
		Arguments: map[string]any{
			"question": "recorded knowledge",
			"max_read": 0,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	knowledge := decodeToolResult[vault.QueryResult](t, query)
	if len(knowledge.ShouldRead) != 0 {
		t.Fatalf("should_read = %v, want disabled", knowledge.ShouldRead)
	}

	resources, err := clientSession.ListResources(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var processURI string
	for _, resource := range resources.Resources {
		if resource.Name == "processes/query-vault.md" {
			processURI = resource.URI
			break
		}
	}
	if processURI == "" {
		t.Fatalf("resources = %+v", resources.Resources)
	}
	read, err := clientSession.ReadResource(ctx, &mcp.ReadResourceParams{URI: processURI})
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Contents) != 1 || !strings.Contains(read.Contents[0].Text, "# query-vault") {
		t.Fatalf("resource = %+v", read.Contents)
	}

	prompts, err := clientSession.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(prompts.Prompts) != 1 || prompts.Prompts[0].Title != "query-vault" {
		t.Fatalf("prompts = %+v", prompts.Prompts)
	}
	processPath := filepath.Join(root, "processes", "query-vault.md")
	processMarkdown, err := os.ReadFile(processPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(processPath, append(processMarkdown, []byte("\nLive prompt revision.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	prompt, err := clientSession.GetPrompt(ctx, &mcp.GetPromptParams{Name: prompts.Prompts[0].Name})
	if err != nil {
		t.Fatal(err)
	}
	if len(prompt.Messages) != 1 {
		t.Fatalf("prompt = %+v", prompt)
	}
	text, ok := prompt.Messages[0].Content.(*mcp.TextContent)
	if !ok || !strings.Contains(text.Text, "# query-vault") || !strings.Contains(text.Text, "Live prompt revision.") || strings.Contains(text.Text, discovery.Processes[0].Revision) {
		t.Fatalf("prompt content = %+v", prompt.Messages[0].Content)
	}
}

func decodeToolResult[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()
	var decoded T
	if result.IsError {
		t.Fatalf("tool error: %+v", result.Content)
	}
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

func mcpTestVault(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeMCPTestFile(t, root, "gnosis.toml", `[vault]
vault_name = "mcp-test"
vault_root = "."
vault_index = false
vault_log = false

[vaults.gnosis]
include = []
`)
	writeMCPTestFile(t, root, "concepts/provenance.md", `---
type: Concept
title: Provenance
description: Source identity and history.
---

# Provenance
`)
	writeMCPTestFile(t, root, "processes/query-vault.md", `---
type: Vault Process
title: query-vault
description: Answer from recorded vault knowledge.
invocation: model
effects: [read]
relationships:
  - type: uses
    target: ../concepts/provenance.md
---

# query-vault

## Use when

- Answering from recorded knowledge.

## Knowledge inputs

- [Provenance](../concepts/provenance.md)

## Process

1. Read selected knowledge.

## Completion

The answer is grounded.
`)
	return root
}

func writeMCPTestFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
