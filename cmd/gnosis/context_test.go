package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	toon "github.com/toon-format/toon-go"
	evidencecontext "gnosis/internal/evidencecontext"
)

func TestContextKnowledgeCLI(t *testing.T) {
	adapterContextVault(t)
	var stdout, stderr bytes.Buffer
	err := run([]string{
		"--vault", "test",
		"context", "knowledge", "adaptertoken",
		"--strategy", "lexical",
		"--max-evidence", "2",
		"--max-chars", "1000",
		"--depth", "2",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := toon.Decode(stdout.Bytes()); err != nil {
		t.Fatalf("decode output: %v\n%s", err, stdout.String())
	}
	for _, value := range []string{
		"strategy: lexical",
		"gnosis://test/evidence.md",
		"gnosis://test/support.md",
		",supports",
		"max_evidence: 2",
		"max_chars: 1000",
	} {
		if !strings.Contains(stdout.String(), value) {
			t.Fatalf("output = %q, missing %q", stdout.String(), value)
		}
	}

	err = run([]string{
		"context", "knowledge", "question", "--max-evidence", "0",
	}, &stdout, &stderr)
	if err == nil || exitCode(err) != 2 {
		t.Fatalf("error = %v, exit = %d", err, exitCode(err))
	}
}

func TestContextHTTPAndMCPParity(t *testing.T) {
	workspace := adapterContextVault(t)
	server := httptest.NewServer(newHTTPHandler(workspace))
	t.Cleanup(server.Close)
	session := connectMCPServer(t, newMCPServer(workspace))

	for _, request := range []evidencecontext.Request{
		contextAdapterRequest("adaptertoken"),
		func() evidencecontext.Request {
			request := contextAdapterRequest("adaptertoken")
			request.Constraints.Type = "Policy"
			return request
		}(),
		contextAdapterRequest("zyxwv unmatched"),
	} {
		httpResult, status := postContext(t, server.URL+"/api/v1/context", request)
		if status != http.StatusOK {
			t.Fatalf("HTTP status = %d, result = %+v", status, httpResult)
		}
		mcpResult := callMCPTool(t, session, "get_evidence_context", requestArguments(t, request))
		var mcpContext evidencecontext.Result
		decodeMCPResult(t, mcpResult, &mcpContext)
		if !reflect.DeepEqual(httpResult, mcpContext) {
			t.Fatalf("HTTP = %+v\nMCP = %+v", httpResult, mcpContext)
		}
	}

	invalid := contextAdapterRequest("question")
	zero := 0
	invalid.MaxChars = &zero
	_, status := postContext(t, server.URL+"/api/v1/context", invalid)
	if status != http.StatusBadRequest {
		t.Fatalf("HTTP invalid status = %d", status)
	}
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_evidence_context", Arguments: requestArguments(t, invalid),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatalf("MCP invalid result = %+v", result)
	}
}

func contextAdapterRequest(question string) evidencecontext.Request {
	maxEvidence, maxChars, maxDepth := 2, 1_000, 2
	return evidencecontext.Request{
		Question: question, Strategy: evidencecontext.StrategyLexical,
		MaxEvidence: &maxEvidence, MaxChars: &maxChars, MaxDepth: &maxDepth,
	}
}

func requestArguments(t *testing.T, request evidencecontext.Request) map[string]any {
	t.Helper()
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	arguments := map[string]any{}
	if err := json.Unmarshal(data, &arguments); err != nil {
		t.Fatal(err)
	}
	return arguments
}

func postContext(
	t *testing.T,
	endpoint string,
	request evidencecontext.Request,
) (evidencecontext.Result, int) {
	t.Helper()
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.Post(endpoint, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var result evidencecontext.Result
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		if response.StatusCode == http.StatusOK {
			t.Fatal(err)
		}
	}
	return result, response.StatusCode
}

func adapterContextVault(t *testing.T) string {
	t.Helper()
	workspace := commandVault(t)
	writeCommandFile(t, workspace, "evidence.md", `---
type: Policy
title: Evidence context
description: Evidence context policy.
status: active
tags: [evidence]
source: manual
confidence: 0.9
tier: core
relationships:
  - type: supports
    target: support.md
---

# Evidence context

Evidence context is bounded and cited with adaptertoken.
`)
	writeCommandFile(t, workspace, "support.md", `---
type: Note
title: Context support
description: Supporting evidence context.
---

# Context support

Support evidence preserves revisions for adaptertoken.
`)
	return workspace
}
