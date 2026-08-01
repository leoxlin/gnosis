package main

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	agenttrace "gnosis/internal/trace"
)

func TestMCPRecordTraceWorksOverBothTransportsAndKeepsSessionsUsable(t *testing.T) {
	t.Setenv(agenttrace.EnvDir, filepath.Join(t.TempDir(), "traces"))
	t.Setenv(agenttrace.EnvAgentID, "agent")
	workspace := mcpTestVault(t)
	stdio := connectMCPServer(t, newMCPServer(workspace))

	server := httptest.NewServer(newHTTPHandler(workspace))
	t.Cleanup(server.Close)
	client := mcp.NewClient(&mcp.Implementation{Name: "gnosis-trace-test", Version: "0.0.0"}, nil)
	httpSession, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: server.URL + "/mcp",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = httpSession.Close() })

	arguments := map[string]any{
		"run_id": "run-1", "sequence": 1, "kind": "tool",
		"occurred_at": "2026-07-29T12:00:00Z", "content": "called go test",
		"metadata": map[string]any{"command": "go test ./..."},
	}
	var created, repeated agenttrace.Result
	decodeMCPResult(t, callMCPTool(t, stdio, "record_trace", arguments), &created)
	decodeMCPResult(t, callMCPTool(t, httpSession, "record_trace", arguments), &repeated)
	if created.Status != agenttrace.StatusCreated || repeated.Status != agenttrace.StatusNoop ||
		created.Record.ContentHash != repeated.Record.ContentHash ||
		created.Record.AgentID != "agent" {
		t.Fatalf("created = %+v, repeated = %+v", created, repeated)
	}

	for _, test := range []struct {
		session   *mcp.ClientSession
		arguments map[string]any
		want      string
	}{
		{stdio, map[string]any{
			"run_id": "run-1", "sequence": 2, "kind": "invalid",
			"occurred_at": "2026-07-29T12:00:00Z", "content": "bad",
		}, "kind must be"},
		{httpSession, map[string]any{
			"run_id": "run-1", "sequence": 1, "kind": "tool",
			"occurred_at": "2026-07-29T12:00:00Z", "content": "replacement",
		}, "conflicts"},
	} {
		result, err := test.session.CallTool(context.Background(), &mcp.CallToolParams{
			Name: "record_trace", Arguments: test.arguments,
		})
		if err != nil {
			t.Fatal(err)
		}
		if !result.IsError || !strings.Contains(mcpResultText(result), test.want) {
			t.Fatalf("result = %+v, want %q tool error", result, test.want)
		}
		if err := test.session.Ping(context.Background(), nil); err != nil {
			t.Fatalf("session failed after trace error: %v", err)
		}
		callMCPTool(t, test.session, "get_vaults", map[string]any{})
	}
}

func TestMCPRecordTraceConfigurationErrorKeepsSessionUsable(t *testing.T) {
	t.Setenv(agenttrace.EnvDir, "")
	t.Setenv(agenttrace.EnvAgentID, "")
	session := connectMCPServer(t, newMCPServer(mcpTestVault(t)))
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "record_trace",
		Arguments: map[string]any{
			"run_id": "run", "sequence": 1, "kind": "run",
			"occurred_at": "2026-07-29T12:00:00Z", "content": "started",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || !strings.Contains(mcpResultText(result), agenttrace.EnvAgentID) {
		t.Fatalf("configuration result = %+v", result)
	}
	if err := session.Ping(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
}
