package main

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	agentlearning "gnosis/internal/learning"
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

func TestMCPRunReadAndLearningProposalHaveTransportParity(t *testing.T) {
	t.Setenv(agenttrace.EnvDir, filepath.Join(t.TempDir(), "traces"))
	t.Setenv(agenttrace.EnvAgentID, "fixed-agent")
	workspace := mcpTestVault(t)
	stdio := connectMCPServer(t, newMCPServer(workspace))
	server := httptest.NewServer(newHTTPHandler(workspace))
	t.Cleanup(server.Close)
	client := mcp.NewClient(&mcp.Implementation{Name: "gnosis-learning-test", Version: "0.0.0"}, nil)
	httpSession, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: server.URL + "/mcp",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = httpSession.Close() })

	revision := "sha256:" + strings.Repeat("a", 64)
	for _, entry := range []map[string]any{
		{
			"run_id": "run", "sequence": 0, "kind": "run",
			"occurred_at": "2026-07-29T12:00:00Z", "content": "started",
			"metadata": map[string]any{
				"procedure_uri":      "gnosis://test/procedures/query.md",
				"procedure_revision": revision,
			},
		},
		{
			"run_id": "run", "sequence": 1, "kind": "knowledge_use",
			"occurred_at": "2026-07-29T12:01:00Z", "content": "used procedure",
			"knowledge_uri":      "gnosis://test/procedures/query.md",
			"knowledge_revision": revision,
		},
		{
			"run_id": "run", "sequence": 2, "kind": "feedback",
			"occurred_at": "2026-07-29T12:02:00Z", "content": "helped",
			"knowledge_uri":      "gnosis://test/procedures/query.md",
			"knowledge_revision": revision, "feedback": "helpful",
		},
		{
			"run_id": "run", "sequence": 3, "kind": "outcome",
			"occurred_at": "2026-07-29T12:03:00Z", "content": "finished",
			"metadata": map[string]any{"success": true},
		},
	} {
		callMCPTool(t, stdio, "record_trace", entry)
	}

	var stdioRun, httpRun agenttrace.Run
	decodeMCPResult(t, callMCPTool(t, stdio, "get_run_trace", map[string]any{
		"run_id": "run", "max_entries": 10, "max_characters": 1000,
	}), &stdioRun)
	decodeMCPResult(t, callMCPTool(t, httpSession, "get_run_trace", map[string]any{
		"run_id": "run", "max_entries": 10, "max_characters": 1000,
	}), &httpRun)
	if !stdioRun.Complete || !httpRun.Complete || len(stdioRun.Entries) != 4 ||
		stdioRun.Entries[2].AgentID != "fixed-agent" ||
		stdioRun.Entries[2].ContentHash != httpRun.Entries[2].ContentHash {
		t.Fatalf("stdio = %+v, http = %+v", stdioRun, httpRun)
	}

	arguments := map[string]any{
		"runs": []map[string]any{{"run_id": "run", "learning_key": "bounded"}},
		"type": "Reflection", "uri": "gnosis://test/reflections/bounded.md",
		"title": "Keep reads bounded", "statement": "Keep trace reads bounded.",
		"expected_absent": true,
	}
	var stdioProposal, httpProposal agentlearning.Proposal
	decodeMCPResult(t, callMCPTool(t, stdio, "propose_run_learning", arguments), &stdioProposal)
	decodeMCPResult(t, callMCPTool(t, httpSession, "propose_run_learning", arguments), &httpProposal)
	if !stdioProposal.Plan.Applicable || stdioProposal.Plan.Digest != httpProposal.Plan.Digest ||
		len(stdioProposal.Candidate.Attributions) != 2 {
		t.Fatalf("stdio = %+v, http = %+v", stdioProposal, httpProposal)
	}
	if _, err := os.Stat(filepath.Join(workspace, "reflections", "bounded.md")); !os.IsNotExist(err) {
		t.Fatalf("proposal wrote target: %v", err)
	}

	callMCPTool(t, stdio, "record_trace", map[string]any{
		"run_id": "incomplete", "sequence": 0, "kind": "run",
		"occurred_at": "2026-07-29T12:00:00Z", "content": "started",
	})
	result, err := httpSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "propose_run_learning",
		Arguments: map[string]any{
			"runs": []map[string]any{{"run_id": "incomplete", "learning_key": "bounded"}},
			"type": "Reflection", "uri": "gnosis://test/reflections/incomplete.md",
			"title": "Incomplete", "statement": "Incomplete.", "expected_absent": true,
		},
	})
	if err != nil || !result.IsError || !strings.Contains(mcpResultText(result), "incomplete") {
		t.Fatalf("result = %+v err = %v", result, err)
	}
	if err := httpSession.Ping(context.Background(), nil); err != nil {
		t.Fatalf("session failed after learning error: %v", err)
	}
}
