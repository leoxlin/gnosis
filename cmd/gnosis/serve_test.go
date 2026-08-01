package main

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	agentmemory "gnosis/internal/memory"
	knowledge "gnosis/internal/search"
	"gnosis/internal/vault"
)

func TestHTTPAPIAndUI(t *testing.T) {
	workspace := httpTestVault(t)
	server := httptest.NewServer(newHTTPHandler(workspace))
	t.Cleanup(server.Close)

	response, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK ||
		!strings.HasPrefix(response.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("GET / = %d %q", response.StatusCode, response.Header.Get("Content-Type"))
	}

	var graph vault.KnowledgeGraph
	if status := getHTTPJSON(t, server.URL+"/api/v1/graph", &graph); status != http.StatusOK {
		t.Fatalf("GET graph status = %d", status)
	}
	if len(graph.Nodes) < 2 || !hasGraphEdge(graph.Edges, "gnosis://test/note.md", "gnosis://test/procedure.md") {
		t.Fatalf("graph = %+v", graph)
	}
	var catalog vault.VaultCatalog
	if status := getHTTPJSON(t, server.URL+"/api/v1/vaults", &catalog); status != http.StatusOK {
		t.Fatalf("GET vaults status = %d", status)
	}
	if len(catalog.Vaults) == 0 || catalog.Vaults[0].Vault != "test" {
		t.Fatalf("vaults = %+v", catalog)
	}
	var pages struct {
		Pages []vault.DocumentRef `json:"pages"`
	}
	if status := getHTTPJSON(t, server.URL+"/api/v1/pages", &pages); status != http.StatusOK {
		t.Fatalf("GET pages status = %d", status)
	}
	if len(pages.Pages) < 2 {
		t.Fatalf("pages = %+v", pages)
	}
	if pages.Pages[0].Trust.Revision == "" {
		t.Fatalf("page-list trust = %+v", pages.Pages[0].Trust)
	}

	var page vault.Page
	pageURL := server.URL + "/api/v1/page?uri=" + url.QueryEscape("gnosis://test/note.md")
	if status := getHTTPJSON(t, pageURL, &page); status != http.StatusOK {
		t.Fatalf("GET page status = %d", status)
	}
	if page.Document.URI != "gnosis://test/note.md" || !strings.Contains(page.Markdown, "gnosis://test/procedure.md") {
		t.Fatalf("page = %+v", page)
	}
	if page.Document.Trust.Status != "reviewed" || page.Document.Trust.Revision != page.Document.Revision {
		t.Fatalf("page trust = %+v", page.Document.Trust)
	}

	var rendered struct {
		HTML string `json:"html"`
	}
	if status := getHTTPJSON(t, pageURL, &rendered); status != http.StatusOK {
		t.Fatalf("GET page html status = %d", status)
	}
	if !strings.Contains(rendered.HTML, `<a href="gnosis://test/procedure.md">implementation procedure</a>`) {
		t.Fatalf("page html = %q", rendered.HTML)
	}
	if strings.Contains(rendered.HTML, "type: Note") {
		t.Fatalf("page html leaked frontmatter = %q", rendered.HTML)
	}

	var concepts conceptsOutput
	if status := getHTTPJSON(t, server.URL+"/api/v1/concepts?type=Note", &concepts); status != http.StatusOK {
		t.Fatalf("GET concepts status = %d", status)
	}
	if concepts.Type != "Note" || len(concepts.Concepts) != 1 {
		t.Fatalf("concepts = %+v", concepts)
	}

	var search knowledge.QueryResult
	searchURL := server.URL + "/api/v1/search?backend=lexical&question=small+adequate+design&top=1"
	if status := getHTTPJSON(t, searchURL, &search); status != http.StatusOK {
		t.Fatalf("GET search status = %d", status)
	}
	if len(search.Candidates) != 1 || search.Candidates[0].URI != "gnosis://test/note.md" {
		t.Fatalf("search = %+v", search)
	}
	if search.Candidates[0].Trust.Status != page.Document.Trust.Status ||
		search.Candidates[0].Trust.Revision != page.Document.Trust.Revision {
		t.Fatalf("search trust = %+v, page trust = %+v", search.Candidates[0].Trust, page.Document.Trust)
	}

	var failure map[string]string
	if status := getHTTPJSON(t, server.URL+"/api/v1/page?uri=invalid", &failure); status != http.StatusBadRequest {
		t.Fatalf("invalid page status = %d", status)
	}
	if failure["error"] == "" {
		t.Fatalf("invalid page response = %+v", failure)
	}
}

func TestHTTPConfiguredRemoteImportRefreshesDuringServerLifetime(t *testing.T) {
	fixture := newCommandRemoteFixture(t, "https://example.test/team/http-import.git")
	workspace := t.TempDir()
	writeCommandFile(t, workspace, "gnosis.toml", `[[vaults]]
vault_name = "remote"
vault_root = "`+fixture.url+`"
`)
	server := httptest.NewServer(newHTTPHandler(workspace))
	t.Cleanup(server.Close)
	endpoint := server.URL + "/api/v1/page?uri=" + url.QueryEscape("gnosis://remote/notes/remote.md")

	var page vault.Page
	if status := getHTTPJSON(t, endpoint, &page); status != http.StatusOK {
		t.Fatalf("initial page status = %d", status)
	}
	if page.Document.Title != "Remote note" {
		t.Fatalf("initial page = %+v", page.Document)
	}

	updateCommandRemoteNote(t, fixture, "Refreshed remote note")
	if status := getHTTPJSON(t, endpoint, &page); status != http.StatusOK {
		t.Fatalf("refreshed page status = %d", status)
	}
	if page.Document.Title != "Refreshed remote note" {
		t.Fatalf("refreshed page = %+v", page.Document)
	}
}

func TestHTTPMCPTransport(t *testing.T) {
	server := httptest.NewServer(newHTTPHandler(httpTestVault(t)))
	t.Cleanup(server.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "gnosis-http-test", Version: "0.0.0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: server.URL + "/mcp",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	})
	if err := session.Ping(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPServerStopsOnCancellation(t *testing.T) {
	workspace := httpTestVault(t)
	ctx, cancel := context.WithCancel(context.Background())
	exited := make(chan error, 1)
	ready := make(chan struct{})
	output := &readyWriter{ready: ready}
	go func() {
		exited <- serveHTTP(ctx, "127.0.0.1:0", workspace, output)
	}()

	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP server did not start")
	}
	cancel()
	select {
	case err := <-exited:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP server did not stop after cancellation")
	}
}

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
	want := []string{
		"add_memory",
		"audit_knowledge",
		"get_changes",
		"get_concepts",
		"get_diff",
		"get_evidence_context",
		"get_history",
		"get_page",
		"get_procedures",
		"get_run_trace",
		"get_vaults",
		"propose_knowledge_change",
		"propose_run_learning",
		"record_trace",
		"search_knowledge",
		"search_memory",
		"trace_graph",
	}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("tools = %v, want %v", names, want)
	}

	vaultsResult := callMCPTool(t, session, "get_vaults", map[string]any{})
	var catalog vault.VaultCatalog
	decodeMCPResult(t, vaultsResult, &catalog)
	if len(catalog.Vaults) == 0 || catalog.Vaults[0].Vault != "test" {
		t.Fatalf("vault catalog = %+v", catalog)
	}

	conceptsResult := callMCPTool(t, session, "get_concepts", map[string]any{"type": "Note"})
	var concepts conceptsOutput
	decodeMCPResult(t, conceptsResult, &concepts)
	if concepts.Type != "Note" || len(concepts.Concepts) != 1 {
		t.Fatalf("concepts = %+v", concepts)
	}
	if concepts.Concepts[0]["uri"] != "gnosis://test/note.md" {
		t.Fatalf("concept = %+v", concepts.Concepts[0])
	}
	if concepts.Concepts[0]["trust"] == nil {
		t.Fatalf("concept trust = %+v", concepts.Concepts[0])
	}

	pageResult := callMCPTool(t, session, "get_page", map[string]any{
		"uri": "gnosis://test/note.md",
	})
	var page vault.Page
	decodeMCPResult(t, pageResult, &page)
	if page.Document.URI != "gnosis://test/note.md" || page.Document.Revision == "" {
		t.Fatalf("page = %+v", page)
	}
	if page.Document.Trust.Status != "reviewed" ||
		page.Document.Trust.Revision != page.Document.Revision {
		t.Fatalf("page trust = %+v", page.Document.Trust)
	}

	currentResult := callMCPTool(t, session, "get_page", map[string]any{
		"uri":             "gnosis://test/old.md",
		"resolve_current": true,
	})
	var old vault.Page
	decodeMCPResult(t, currentResult, &old)
	if old.Document.URI != "gnosis://test/old.md" || old.CurrentResolution == nil ||
		old.CurrentResolution.Current != "gnosis://test/note.md" {
		t.Fatalf("resolved page = %+v", old)
	}

	searchResult := callMCPTool(t, session, "search_knowledge", map[string]any{
		"question": "small adequate design",
		"backend":  "lexical",
		"top":      1,
		"max_read": 1,
		"depth":    1,
	})
	var query knowledge.QueryResult
	decodeMCPResult(t, searchResult, &query)
	if len(query.Candidates) != 1 || query.Candidates[0].URI != "gnosis://test/note.md" {
		t.Fatalf("query = %+v", query)
	}
	if query.Candidates[0].Revision == "" || query.Candidates[0].Origin.Vault != "test" {
		t.Fatalf("candidate provenance = %+v", query.Candidates[0])
	}
	if query.Candidates[0].Trust.Status != page.Document.Trust.Status ||
		query.Candidates[0].Trust.Revision != page.Document.Trust.Revision {
		t.Fatalf("candidate trust = %+v", query.Candidates[0].Trust)
	}
}

func TestMCPKnowledgeChangeToolsAreGatedAndWorkOverBothTransports(t *testing.T) {
	workspace := mcpTestVault(t)
	writeCommandFile(t, workspace, "types/note.md", `---
type: Concept
title: Note
description: A test note.
path: notes
---
`)

	defaultSession := connectMCPServer(t, newMCPServer(workspace))
	defaultTools, err := defaultSession.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasMCPTool(defaultTools.Tools, "propose_knowledge_change") ||
		hasMCPTool(defaultTools.Tools, "apply_knowledge_change") {
		t.Fatalf("default tools = %+v", defaultTools.Tools)
	}

	enabledSession := connectMCPServer(t, newMCPServerWithKnowledgeWrites(workspace, true))
	enabledTools, err := enabledSession.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasMCPTool(enabledTools.Tools, "apply_knowledge_change") {
		t.Fatalf("write-enabled tools = %+v", enabledTools.Tools)
	}

	candidate := `---
type: Note
title: Planned
description: MCP knowledge change.
---

# Planned
`
	arguments := map[string]any{
		"uri":             "gnosis://test/notes/planned.md",
		"candidate":       candidate,
		"expected_absent": true,
	}
	proposalResult := callMCPTool(t, defaultSession, "propose_knowledge_change", arguments)
	var plan vault.KnowledgeChangePlan
	decodeMCPResult(t, proposalResult, &plan)
	if !plan.Applicable || plan.Operation != "create" {
		t.Fatalf("plan = %+v", plan)
	}

	changed := maps.Clone(arguments)
	changed["candidate"] = strings.Replace(candidate, "# Planned", "# Changed", 1)
	changed["digest"] = plan.Digest
	result, err := enabledSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "apply_knowledge_change", Arguments: changed,
	})
	if err != nil || !result.IsError || !strings.Contains(mcpResultText(result), "digest changed") {
		t.Fatalf("changed digest result = %+v err = %v", result, err)
	}

	writeCommandFile(t, workspace, "notes/planned.md", candidate)
	stale := maps.Clone(arguments)
	stale["digest"] = plan.Digest
	result, err = enabledSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "apply_knowledge_change", Arguments: stale,
	})
	if err != nil || !result.IsError || !strings.Contains(mcpResultText(result), "actual revision") {
		t.Fatalf("stale result = %+v err = %v", result, err)
	}

	invalidResult := callMCPTool(t, defaultSession, "propose_knowledge_change", map[string]any{
		"uri":             "gnosis://test/notes/invalid.md",
		"candidate":       "not typed markdown",
		"expected_absent": true,
	})
	var invalid vault.KnowledgeChangePlan
	decodeMCPResult(t, invalidResult, &invalid)
	if invalid.Applicable || len(invalid.Findings) == 0 {
		t.Fatalf("invalid plan = %+v", invalid)
	}

	successArguments := map[string]any{
		"uri":             "gnosis://test/notes/applied.md",
		"candidate":       strings.Replace(candidate, "Planned", "Applied", 2),
		"expected_absent": true,
	}
	successProposal := callMCPTool(t, defaultSession, "propose_knowledge_change", successArguments)
	decodeMCPResult(t, successProposal, &plan)
	successArguments["digest"] = plan.Digest
	appliedResult := callMCPTool(t, enabledSession, "apply_knowledge_change", successArguments)
	var applied vault.KnowledgeChangeResult
	decodeMCPResult(t, appliedResult, &applied)
	if !applied.Changed || applied.Revision == "" {
		t.Fatalf("applied = %+v", applied)
	}

	server := httptest.NewServer(newHTTPHandlerWithKnowledgeWrites(workspace, true))
	t.Cleanup(server.Close)
	client := mcp.NewClient(&mcp.Implementation{Name: "gnosis-test", Version: "0.0.0"}, nil)
	httpSession, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: server.URL + "/mcp",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = httpSession.Close() })
	httpTools, err := httpSession.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !hasMCPTool(httpTools.Tools, "propose_knowledge_change") ||
		!hasMCPTool(httpTools.Tools, "apply_knowledge_change") {
		t.Fatalf("HTTP tools = %+v", httpTools.Tools)
	}
	httpArguments := map[string]any{
		"uri":             "gnosis://test/notes/http-applied.md",
		"candidate":       strings.Replace(candidate, "Planned", "HTTP applied", 2),
		"expected_absent": true,
	}
	httpProposal := callMCPTool(t, httpSession, "propose_knowledge_change", httpArguments)
	decodeMCPResult(t, httpProposal, &plan)
	httpArguments["digest"] = plan.Digest
	httpApplied := callMCPTool(t, httpSession, "apply_knowledge_change", httpArguments)
	decodeMCPResult(t, httpApplied, &applied)
	if !applied.Changed {
		t.Fatalf("HTTP apply = %+v", applied)
	}
}

func TestMCPGraphAndProcedureParity(t *testing.T) {
	workspace := mcpTestVault(t)
	writeMCPGraphProcedureFixtures(t, workspace)
	session := connectMCPServer(t, newMCPServer(workspace))

	neighborsResult := callMCPTool(t, session, "trace_graph", map[string]any{
		"uri":       "gnosis://test/source.md",
		"direction": "out",
		"limit":     1,
	})
	var graph traceGraphOutput
	decodeMCPResult(t, neighborsResult, &graph)
	wantNeighbors, err := vault.TraceNeighbors(
		workspace, "gnosis://test/source.md", vault.DirectionOut, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if graph.Mode != "neighbors" || graph.Neighbors == nil ||
		graph.Neighbors.Total != len(wantNeighbors.Edges) ||
		len(graph.Neighbors.Edges) != 1 || !graph.Neighbors.Truncated ||
		len(graph.Neighbors.Continuation) != 1 ||
		graph.Neighbors.Node.Revision != wantNeighbors.Node.Revision ||
		!sameGraphEdge(graph.Neighbors.Edges[0], wantNeighbors.Edges[0]) {
		t.Fatalf("neighbors = %+v, want parity with %+v", graph, wantNeighbors)
	}

	pathResult := callMCPTool(t, session, "trace_graph", map[string]any{
		"uri":        "gnosis://test/source.md",
		"target_uri": "gnosis://test/target.md",
		"direction":  "out",
		"relations":  []string{"supports"},
		"depth":      2,
	})
	decodeMCPResult(t, pathResult, &graph)
	wantPath, err := vault.TracePath(
		workspace,
		"gnosis://test/source.md",
		"gnosis://test/target.md",
		vault.DirectionOut,
		[]string{"supports"},
		2,
	)
	if err != nil {
		t.Fatal(err)
	}
	if graph.Mode != "path" || graph.Path == nil ||
		graph.Path.Status != wantPath.Status ||
		len(graph.Path.Nodes) != len(wantPath.Nodes) ||
		len(graph.Path.Edges) != len(wantPath.Edges) ||
		graph.Path.Nodes[1].Origin != wantPath.Nodes[1].Origin ||
		graph.Path.Nodes[1].Revision != wantPath.Nodes[1].Revision ||
		!sameGraphEdge(graph.Path.Edges[0], wantPath.Edges[0]) {
		t.Fatalf("path = %+v, want parity with %+v", graph, wantPath)
	}

	discoveryResult := callMCPTool(t, session, "get_procedures", map[string]any{
		"tags": []string{"test", "mcp"},
	})
	var procedures getProceduresOutput
	decodeMCPResult(t, discoveryResult, &procedures)
	wantCatalog, err := vault.DiscoverProcesses(workspace, []string{"test", "mcp"})
	if err != nil {
		t.Fatal(err)
	}
	if procedures.Mode != "discovery" || len(procedures.Procedures) != 1 ||
		len(wantCatalog["procedures"]) != 1 {
		t.Fatalf("procedures = %+v, want parity with %+v", procedures, wantCatalog)
	}
	wantProcedure := wantCatalog["procedures"][0]
	wantTrust := wantProcedure["trust"].(vault.TrustProjection)
	gotTrust, ok := procedures.Procedures[0]["trust"].(map[string]any)
	if !ok {
		t.Fatalf("procedure trust = %#v", procedures.Procedures[0]["trust"])
	}
	gotOrigin, ok := gotTrust["origin"].(map[string]any)
	if !ok {
		t.Fatalf("procedure origin = %#v", gotTrust["origin"])
	}
	if procedures.Procedures[0]["uri"] != wantProcedure["uri"] ||
		procedures.Procedures[0]["invocation"] != "model" ||
		gotTrust["revision"] != wantTrust.Revision ||
		gotOrigin["vault"] != wantTrust.Origin.Vault ||
		gotOrigin["kind"] != string(wantTrust.Origin.Kind) {
		t.Fatalf("procedures = %+v, want parity with %+v", procedures, wantCatalog)
	}

	contractResult := callMCPTool(t, session, "get_procedures", map[string]any{
		"uri": "gnosis://test/procedures/model.md",
	})
	decodeMCPResult(t, contractResult, &procedures)
	wantContract, err := vault.InvokeProcess(workspace, "gnosis://test/procedures/model.md")
	if err != nil {
		t.Fatal(err)
	}
	if procedures.Mode != "contract" || procedures.Procedure == nil ||
		procedures.Procedure.Process.Revision != wantContract.Process.Revision ||
		procedures.Procedure.Process.Origin != wantContract.Process.Origin ||
		procedures.Procedure.Sections != wantContract.Sections ||
		len(procedures.Procedure.Steps) != len(wantContract.Steps) {
		t.Fatalf("procedure = %+v, want parity with %+v", procedures, wantContract)
	}
}

func TestMCPGraphAndProcedureErrorsKeepSessionUsable(t *testing.T) {
	workspace := mcpTestVault(t)
	writeMCPGraphProcedureFixtures(t, workspace)
	session := connectMCPServer(t, newMCPServer(workspace))
	tests := []struct {
		name      string
		tool      string
		arguments map[string]any
	}{
		{name: "source uri", tool: "trace_graph", arguments: map[string]any{"uri": "bad"}},
		{name: "target uri", tool: "trace_graph", arguments: map[string]any{
			"uri": "gnosis://test/source.md", "target_uri": "bad",
		}},
		{name: "direction", tool: "trace_graph", arguments: map[string]any{
			"uri": "gnosis://test/source.md", "direction": "sideways",
		}},
		{name: "relation", tool: "trace_graph", arguments: map[string]any{
			"uri": "gnosis://test/source.md", "relations": []string{" "},
		}},
		{name: "neighbor depth", tool: "trace_graph", arguments: map[string]any{
			"uri": "gnosis://test/source.md", "depth": 1,
		}},
		{name: "path depth", tool: "trace_graph", arguments: map[string]any{
			"uri": "gnosis://test/source.md", "target_uri": "gnosis://test/target.md", "depth": -1,
		}},
		{name: "neighbor limit", tool: "trace_graph", arguments: map[string]any{
			"uri": "gnosis://test/source.md", "limit": 0,
		}},
		{name: "path limit", tool: "trace_graph", arguments: map[string]any{
			"uri": "gnosis://test/source.md", "target_uri": "gnosis://test/target.md", "limit": 1,
		}},
		{name: "procedure uri", tool: "get_procedures", arguments: map[string]any{"uri": "bad"}},
		{name: "contract tags", tool: "get_procedures", arguments: map[string]any{
			"uri": "gnosis://test/procedures/model.md", "tags": []string{"test"},
		}},
		{name: "empty tag", tool: "get_procedures", arguments: map[string]any{"tags": []string{""}}},
		{name: "invalid contract", tool: "get_procedures", arguments: map[string]any{
			"uri": "gnosis://test/procedures/invalid.md",
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
				Name: test.tool, Arguments: test.arguments,
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

func TestMCPProcedureDiscoveryDoesNotChangeCatalogs(t *testing.T) {
	workspace := mcpTestVault(t)
	session := connectMCPServer(t, newMCPServer(workspace))
	ctx := context.Background()
	beforeTools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	beforePrompts, err := session.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	writeCommandFile(t, workspace, "procedures/added.md", validMCPProcedure("Added", "catalog"))
	result := callMCPTool(t, session, "get_procedures", map[string]any{
		"tags": []string{"catalog"},
	})
	var procedures getProceduresOutput
	decodeMCPResult(t, result, &procedures)
	if len(procedures.Procedures) != 1 ||
		procedures.Procedures[0]["uri"] != "gnosis://test/procedures/added.md" {
		t.Fatalf("procedures = %+v", procedures)
	}

	afterTools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	afterPrompts, err := session.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(afterTools.Tools) != len(beforeTools.Tools) ||
		len(afterPrompts.Prompts) != len(beforePrompts.Prompts) {
		t.Fatalf(
			"catalogs changed: tools %d -> %d, prompts %d -> %d",
			len(beforeTools.Tools), len(afterTools.Tools),
			len(beforePrompts.Prompts), len(afterPrompts.Prompts),
		)
	}
}

func TestMCPResources(t *testing.T) {
	workspace := mcpTestVault(t)
	imported := filepath.Join(workspace, "imported")
	writeCommandFile(t, workspace, "gnosis.toml", `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false

[[vaults]]
vault_name = "imported"
vault_root = "imported"
`)
	writeCommandFile(t, imported, "gnosis.toml", `[vault]
vault_name = "imported"
vault_root = "."
vault_index = false
vault_log = false
`)
	writeCommandFile(t, imported, "note.md", "---\ntype: Note\ntitle: Shadowed\n---\n")
	session := connectMCPServer(t, newMCPServer(workspace))
	ctx := context.Background()

	templates, err := session.ListResourceTemplates(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(templates.ResourceTemplates) != 1 {
		t.Fatalf("resource templates = %+v", templates.ResourceTemplates)
	}
	template := templates.ResourceTemplates[0]
	if template.URITemplate != mcpPageResourceTemplate ||
		template.MIMEType != vault.ResourceMediaType {
		t.Fatalf("resource template = %+v", template)
	}

	listed, err := session.ListResources(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var descriptor *mcp.Resource
	for i, resource := range listed.Resources {
		if i > 0 && listed.Resources[i-1].URI >= resource.URI {
			t.Fatalf("resources are not ordered: %q before %q", listed.Resources[i-1].URI, resource.URI)
		}
		if resource.URI == "gnosis://test/note.md" {
			descriptor = resource
		}
		if resource.URI == "gnosis://imported/note.md" {
			t.Fatal("resource discovery exposed a shadowed page")
		}
	}
	if descriptor == nil || descriptor.Title != "Keep it small" ||
		descriptor.MIMEType != vault.ResourceMediaType || descriptor.Size == 0 ||
		descriptor.Meta["revision"] == "" {
		t.Fatalf("resource descriptor = %+v", descriptor)
	}

	content := readMCPResource(t, session, descriptor.URI)
	if content.URI != descriptor.URI || content.MIMEType != vault.ResourceMediaType ||
		!strings.Contains(content.Text, "simplest design") ||
		content.Meta["revision"] != descriptor.Meta["revision"] ||
		content.Meta["origin"] == nil {
		t.Fatalf("resource content = %+v, descriptor = %+v", content, descriptor)
	}

	for _, uri := range []string{"gnosis://test/missing.md", "not-a-gnosis-uri"} {
		_, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
		assertMCPErrorCode(t, err, mcp.CodeResourceNotFound)
	}
	_, err = session.ListResources(ctx, &mcp.ListResourcesParams{Cursor: "not a cursor"})
	assertMCPErrorCode(t, err, jsonrpc.CodeInvalidParams)
	if err := session.Ping(ctx, nil); err != nil {
		t.Fatalf("session failed after resource errors: %v", err)
	}
}

func TestMCPDirectRemoteTargetRefreshesDuringServerLifetime(t *testing.T) {
	fixture := newCommandRemoteFixture(t, "https://example.test/team/mcp-direct.git")
	session := connectMCPServer(t, newMCPServer(fixture.url))
	arguments := map[string]any{"uri": "gnosis://remote/notes/remote.md"}

	resources, err := session.ListResources(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if resourceTitle(resources.Resources, arguments["uri"].(string)) != "Remote note" {
		t.Fatalf("initial resources = %+v", resources.Resources)
	}
	result := callMCPTool(t, session, "get_page", arguments)
	var page vault.Page
	decodeMCPResult(t, result, &page)
	if page.Document.Title != "Remote note" {
		t.Fatalf("initial page = %+v", page.Document)
	}

	updateCommandRemoteNote(t, fixture, "Refreshed remote note")
	resource := readMCPResource(t, session, arguments["uri"].(string))
	if !strings.Contains(resource.Text, "# Refreshed remote note") {
		t.Fatalf("refreshed resource = %+v", resource)
	}
	resources, err = session.ListResources(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if resourceTitle(resources.Resources, arguments["uri"].(string)) != "Refreshed remote note" {
		t.Fatalf("refreshed resources = %+v", resources.Resources)
	}
	result = callMCPTool(t, session, "get_page", arguments)
	decodeMCPResult(t, result, &page)
	if page.Document.Title != "Refreshed remote note" {
		t.Fatalf("refreshed page = %+v", page.Document)
	}
}

func TestMCPMemoryTools(t *testing.T) {
	setMemoryEnv(t, memoryAPIServer(t).URL)
	workspace := mcpTestVault(t)
	before, err := vault.ReadPage(workspace, "gnosis://test/note.md")
	if err != nil {
		t.Fatal(err)
	}
	session := connectMCPServer(t, newMCPServer(workspace))

	addedResult := callMCPTool(t, session, "add_memory", map[string]any{
		"text": "I prefer dark mode",
	})
	var added agentmemory.Result
	decodeMCPResult(t, addedResult, &added)
	if added.Count != 1 || added.Memories[0].ID != "memory-1" ||
		added.Memories[0].Event != "ADD" {
		t.Fatalf("added = %+v", added)
	}

	searchResult := callMCPTool(t, session, "search_memory", map[string]any{
		"query": "theme preference",
		"limit": 1,
	})
	var found agentmemory.Result
	decodeMCPResult(t, searchResult, &found)
	if found.Count != 1 || found.Memories[0].Text != "Prefers dark mode" {
		t.Fatalf("found = %+v", found)
	}

	after, err := vault.ReadPage(workspace, "gnosis://test/note.md")
	if err != nil {
		t.Fatal(err)
	}
	if after.Document.Revision != before.Document.Revision {
		t.Fatalf("page revision changed from %q to %q", before.Document.Revision, after.Document.Revision)
	}
}

func TestMCPVaultMemoryTools(t *testing.T) {
	clearCommandMemoryEnv(t)
	t.Setenv(agentmemory.EnvUserID, "user")
	t.Setenv(agentmemory.EnvAgentID, "agent")
	workspace := mcpTestVault(t)
	session := connectMCPServer(t, newMCPServer(workspace))
	assertVaultMemoryTools(t, session)
}

func TestMCPMemoryErrorsKeepSessionUsable(t *testing.T) {
	for _, name := range []string{
		agentmemory.EnvAPIKey,
		agentmemory.EnvUserID,
		agentmemory.EnvAgentID,
		agentmemory.EnvProvider,
		agentmemory.EnvBaseURL,
	} {
		t.Setenv(name, "")
	}
	workspace := mcpTestVault(t)
	session := connectMCPServer(t, newMCPServer(workspace))

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "add_memory",
		Arguments: map[string]any{"text": "remember this"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || !strings.Contains(mcpResultText(result), agentmemory.EnvUserID) {
		t.Fatalf("configuration result = %+v", result)
	}
	if err := session.Ping(context.Background(), nil); err != nil {
		t.Fatalf("session failed after configuration error: %v", err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		http.Error(response, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(upstream.Close)
	setMemoryEnv(t, upstream.URL)

	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "add_memory",
		Arguments: map[string]any{"text": " \t"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || !strings.Contains(mcpResultText(result), "text must not be empty") {
		t.Fatalf("invalid add result = %+v", result)
	}

	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "search_memory",
		Arguments: map[string]any{"query": "memory", "limit": 21},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || !strings.Contains(mcpResultText(result), "between 1 and 20") {
		t.Fatalf("invalid search result = %+v", result)
	}

	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "search_memory",
		Arguments: map[string]any{"query": "memory"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || !strings.Contains(mcpResultText(result), "status 503") {
		t.Fatalf("upstream search result = %+v", result)
	}

	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "add_memory",
		Arguments: map[string]any{"text": "remember this"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || !strings.Contains(mcpResultText(result), "status 503") {
		t.Fatalf("upstream result = %+v", result)
	}
	if err := session.Ping(context.Background(), nil); err != nil {
		t.Fatalf("session failed after memory errors: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(workspace, "memories", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("external failures wrote vault memories: %v", matches)
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

func TestMCPSubprocess(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "gnosis")
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "go-cache"))
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build gnosis: %v\n%s", err, output)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	mcpTestVault(t)
	command := exec.Command(binary, "serve", "mcp", "--vault", "test")
	client := mcp.NewClient(&mcp.Implementation{Name: "gnosis-test", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: command}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })

	pageResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_page",
		Arguments: map[string]any{"uri": "gnosis://test/note.md"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var page vault.Page
	decodeMCPResult(t, pageResult, &page)
	if page.Document.URI != "gnosis://test/note.md" || page.Document.Revision == "" {
		t.Fatalf("page = %+v", page)
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

func hasMCPTool(tools []*mcp.Tool, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
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

func readMCPResource(t *testing.T, session *mcp.ClientSession, uri string) *mcp.ResourceContents {
	t.Helper()
	result, err := session.ReadResource(context.Background(), &mcp.ReadResourceParams{URI: uri})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("resource contents = %+v", result.Contents)
	}
	return result.Contents[0]
}

func assertMCPErrorCode(t *testing.T, err error, want int64) {
	t.Helper()
	var rpcError *jsonrpc.Error
	if !errors.As(err, &rpcError) || rpcError.Code != want {
		t.Fatalf("error = %v, want MCP code %d", err, want)
	}
}

func resourceTitle(resources []*mcp.Resource, uri string) string {
	for _, resource := range resources {
		if resource.URI == uri {
			return resource.Title
		}
	}
	return ""
}

func assertVaultMemoryTools(t *testing.T, session *mcp.ClientSession) {
	t.Helper()
	addedResult := callMCPTool(t, session, "add_memory", map[string]any{
		"text": "I prefer dark mode",
	})
	var added agentmemory.Result
	decodeMCPResult(t, addedResult, &added)
	if added.Count != 1 || added.Memories[0].Event != "ADD" ||
		added.Memories[0].Backend != agentmemory.BackendVault ||
		added.Memories[0].CreatedAt == "" || added.Memories[0].UpdatedAt == "" {
		t.Fatalf("added = %+v", added)
	}

	searchResult := callMCPTool(t, session, "search_memory", map[string]any{
		"query": "dark mode",
		"limit": 1,
	})
	var found agentmemory.Result
	decodeMCPResult(t, searchResult, &found)
	if found.Count != 1 || found.Memories[0].ID != added.Memories[0].ID ||
		found.Memories[0].Text != "I prefer dark mode" ||
		found.Memories[0].Score == nil ||
		found.Memories[0].Backend != agentmemory.BackendVault {
		t.Fatalf("found = %+v", found)
	}
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

func memoryAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/v3/memories/add/":
			writeHTTPJSON(response, http.StatusOK, []map[string]any{{
				"id": "memory-1", "event": "ADD",
				"data": map[string]any{"memory": "Prefers dark mode"},
			}})
		case "/v3/memories/search/":
			writeHTTPJSON(response, http.StatusOK, map[string]any{"results": []map[string]any{{
				"id": "memory-1", "memory": "Prefers dark mode", "score": 0.9,
			}}})
		default:
			t.Errorf("unexpected memory request: %s %s", request.Method, request.URL.Path)
			http.NotFound(response, request)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func setMemoryEnv(t *testing.T, baseURL string) {
	t.Helper()
	t.Setenv(agentmemory.EnvAPIKey, "key")
	t.Setenv(agentmemory.EnvUserID, "user")
	t.Setenv(agentmemory.EnvAgentID, "agent")
	t.Setenv(agentmemory.EnvProvider, agentmemory.ProviderHosted)
	t.Setenv(agentmemory.EnvBaseURL, baseURL)
}

func mcpTestVault(t *testing.T) string {
	t.Helper()
	workspace := commandVault(t)
	writeCommandFile(t, workspace, "note.md", `---
type: Note
title: Keep it small
description: Prefer the smallest adequate design.
status: reviewed
confidence: 0.9
---

Use the simplest design that satisfies the current requirement. ^[inferred]
`)
	writeCommandFile(t, workspace, "old.md", `---
type: Concept
title: Old guidance
status: archived
superseded_by: note.md
---
`)
	return workspace
}

func writeMCPGraphProcedureFixtures(t *testing.T, workspace string) {
	writeCommandFile(t, workspace, "gnosis.toml", `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false

[[vaults]]
vault_name = "imported"
vault_root = "imported"
`)
	writeCommandFile(t, workspace, "imported/gnosis.toml", `[vault]
vault_name = "imported"
vault_root = "."
vault_index = false
vault_log = false
`)
	writeCommandFile(t, workspace, "imported/target.md", `---
type: Note
title: Shadowed target
description: Lower-precedence target.
---
`)
	writeCommandFile(t, workspace, "source.md", `---
type: Note
title: Source
description: Graph source.
relationships:
  - type: supports
    target: target.md
  - type: supports
    target: other.md
---
`)
	writeCommandFile(t, workspace, "target.md", `---
type: Note
title: Effective target
description: Higher-precedence target.
---
`)
	writeCommandFile(t, workspace, "other.md", `---
type: Note
title: Other target
description: Another graph target.
---
`)
	writeCommandFile(
		t, workspace, "procedures/model.md", validMCPProcedure("Model procedure", "mcp"),
	)
	writeCommandFile(t, workspace, "procedures/invalid.md", `---
type: Procedure
title: Invalid procedure
description: An invalid explicit Procedure.
tags: [test]
invocation: explicit
---
`)
}

func validMCPProcedure(title, tag string) string {
	return "---\n" +
		"type: Procedure\n" +
		"title: " + title + "\n" +
		"description: A valid MCP test Procedure.\n" +
		"tags: [test, " + tag + "]\n" +
		"---\n\n" +
		"## Inputs\n\nFacts.\n\n" +
		"## Process\n\n1. Act.\n\n" +
		"## Completion\n\nThe action is complete.\n"
}

func httpTestVault(t *testing.T) string {
	t.Helper()
	workspace := commandVault(t)
	writeCommandFile(t, workspace, "note.md", `---
type: Note
title: Keep it small
description: Prefer the smallest adequate design.
status: reviewed
confidence: 0.9
---

Follow the [implementation procedure](procedure.md).
`)
	writeCommandFile(t, workspace, "procedure.md", `---
type: Procedure
title: Apply the note
description: Apply the selected design.
---

Build only what the note requires.
`)
	return workspace
}

func getHTTPJSON(t *testing.T, endpoint string, target any) int {
	t.Helper()
	response, err := http.Get(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	return response.StatusCode
}

func hasGraphEdge(edges []vault.GraphEdge, from, to string) bool {
	for _, edge := range edges {
		if edge.From.URI == from && edge.To.URI == to {
			return true
		}
	}
	return false
}

func sameGraphEdge(left, right vault.GraphEdge) bool {
	return left.From.URI == right.From.URI &&
		left.From.Origin == right.From.Origin &&
		left.From.Revision == right.From.Revision &&
		left.To.URI == right.To.URI &&
		left.To.Origin == right.To.Origin &&
		left.To.Revision == right.To.Revision &&
		left.Relation == right.Relation &&
		left.Raw == right.Raw &&
		left.Source == right.Source
}

type readyWriter struct {
	once  sync.Once
	ready chan struct{}
}

func (w *readyWriter) Write(data []byte) (int, error) {
	w.once.Do(func() { close(w.ready) })
	return len(data), nil
}
