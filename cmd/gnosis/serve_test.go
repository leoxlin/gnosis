package main

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gnosis/internal/vault"
)

func TestMCPTools(t *testing.T) {
	workspace := mcpTestVault(t)
	session := connectMCPServer(t, newMCPServer(workspace))
	ctx := context.Background()

	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(listed.Tools))
	for _, tool := range listed.Tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	want := []string{"get_concepts", "get_page", "get_vaults", "search_knowledge"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("tools = %v, want %v", names, want)
	}

	vaultsResult := callMCPTool(t, session, "get_vaults", map[string]any{})
	var catalog vault.VaultCatalog
	decodeMCPResult(t, vaultsResult, &catalog)
	if len(catalog.Vaults) == 0 || catalog.Vaults[0].Vault != "test" {
		t.Fatalf("vault catalog = %+v", catalog)
	}

	conceptsResult := callMCPTool(t, session, "get_concepts", map[string]any{"type": "Decision"})
	var concepts conceptsOutput
	decodeMCPResult(t, conceptsResult, &concepts)
	if concepts.Type != "Decision" || len(concepts.Concepts) != 1 {
		t.Fatalf("concepts = %+v", concepts)
	}
	if concepts.Concepts[0]["uri"] != "gnosis://test/decision.md" {
		t.Fatalf("concept = %+v", concepts.Concepts[0])
	}

	pageResult := callMCPTool(t, session, "get_page", map[string]any{
		"uri": "gnosis://test/decision.md",
	})
	var page vault.Page
	decodeMCPResult(t, pageResult, &page)
	if page.Document.URI != "gnosis://test/decision.md" || page.Document.Revision == "" {
		t.Fatalf("page = %+v", page)
	}

	searchResult := callMCPTool(t, session, "search_knowledge", map[string]any{
		"question": "small adequate design",
		"backend":  "lexical",
		"top":      1,
		"max_read": 1,
		"depth":    1,
	})
	var query vault.QueryResult
	decodeMCPResult(t, searchResult, &query)
	if len(query.Candidates) != 1 || query.Candidates[0].URI != "gnosis://test/decision.md" {
		t.Fatalf("query = %+v", query)
	}
	if query.Candidates[0].Revision == "" || query.Candidates[0].Origin.Vault != "test" {
		t.Fatalf("candidate provenance = %+v", query.Candidates[0])
	}
}

func TestMCPSearchDefaultsToVector(t *testing.T) {
	t.Setenv("GNOSIS_DATABASE_URL", "")
	t.Setenv("GNOSIS_EMBEDDING_URL", "")
	t.Setenv("GNOSIS_EMBEDDING_MODEL", "")
	session := connectMCPServer(t, newMCPServer(mcpTestVault(t)))

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "search_knowledge",
		Arguments: map[string]any{"question": "what is gnosis?"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || !strings.Contains(mcpResultText(result), "GNOSIS_DATABASE_URL") {
		t.Fatalf("result = %+v, want vector configuration tool error", result)
	}
}

func TestMCPInvalidInputIsToolError(t *testing.T) {
	session := connectMCPServer(t, newMCPServer(mcpTestVault(t)))
	tests := []struct {
		name      string
		arguments map[string]any
	}{
		{name: "get_page", arguments: map[string]any{"uri": "not-a-gnosis-uri"}},
		{name: "get_concepts", arguments: map[string]any{"type": "UnknownType"}},
		{name: "search_knowledge", arguments: map[string]any{"question": "question", "backend": "other"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
				Name: test.name, Arguments: test.arguments,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !result.IsError {
				t.Fatalf("result = %+v, want tool error", result)
			}
			if err := session.Ping(context.Background(), nil); err != nil {
				t.Fatalf("session failed after tool error: %v", err)
			}
		})
	}
}

func TestMCPServerStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	exited := make(chan error, 1)
	server := newMCPServer(mcpTestVault(t))
	go func() {
		exited <- server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "gnosis-test", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	if err := session.Ping(ctx, nil); err != nil {
		cancel()
		t.Fatal(err)
	}
	cancel()

	select {
	case err := <-exited:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("server exit error = %v, want context canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop after cancellation")
	}
}

func connectMCPServer(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "gnosis-test", Version: "0.0.0"}, nil)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		_ = serverSession.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = clientSession.Close()
		_ = serverSession.Close()
	})
	return clientSession
}

func callMCPTool(t *testing.T, session *mcp.ClientSession, name string, arguments map[string]any) *mcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("%s returned tool error: %s", name, mcpResultText(result))
	}
	return result
}

func decodeMCPResult(t *testing.T, result *mcp.CallToolResult, target any) {
	t.Helper()
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func mcpResultText(result *mcp.CallToolResult) string {
	var text strings.Builder
	for _, content := range result.Content {
		if item, ok := content.(*mcp.TextContent); ok {
			text.WriteString(item.Text)
		}
	}
	return text.String()
}

func mcpTestVault(t *testing.T) string {
	t.Helper()
	workspace := commandVault(t)
	writeCommandFile(t, workspace, "decision.md", `---
type: Decision
title: Keep it small
description: Prefer the smallest adequate design.
---

Use the simplest design that satisfies the current requirement.
`)
	return workspace
}
