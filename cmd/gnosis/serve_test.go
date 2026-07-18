package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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

	"github.com/modelcontextprotocol/go-sdk/mcp"
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
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), "gnosis atlas") {
		t.Fatalf("GET / = %d %q", response.StatusCode, body)
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

	var page vault.Page
	pageURL := server.URL + "/api/v1/page?uri=" + url.QueryEscape("gnosis://test/note.md")
	if status := getHTTPJSON(t, pageURL, &page); status != http.StatusOK {
		t.Fatalf("GET page status = %d", status)
	}
	if page.Document.URI != "gnosis://test/note.md" || !strings.Contains(page.Markdown, "gnosis://test/procedure.md") {
		t.Fatalf("page = %+v", page)
	}

	var concepts conceptsOutput
	if status := getHTTPJSON(t, server.URL+"/api/v1/concepts?type=Note", &concepts); status != http.StatusOK {
		t.Fatalf("GET concepts status = %d", status)
	}
	if concepts.Type != "Note" || len(concepts.Concepts) != 1 {
		t.Fatalf("concepts = %+v", concepts)
	}

	var search vault.QueryResult
	searchURL := server.URL + "/api/v1/search?backend=lexical&question=small+adequate+design&top=1"
	if status := getHTTPJSON(t, searchURL, &search); status != http.StatusOK {
		t.Fatalf("GET search status = %d", status)
	}
	if len(search.Candidates) != 1 || search.Candidates[0].URI != "gnosis://test/note.md" {
		t.Fatalf("search = %+v", search)
	}

	var failure map[string]string
	if status := getHTTPJSON(t, server.URL+"/api/v1/page?uri=invalid", &failure); status != http.StatusBadRequest {
		t.Fatalf("invalid page status = %d", status)
	}
	if failure["error"] == "" {
		t.Fatalf("invalid page response = %+v", failure)
	}
}

func TestHTTPMCP(t *testing.T) {
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

	listed, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Tools) != 4 {
		t.Fatalf("tools = %d, want 4", len(listed.Tools))
	}
	result := callMCPTool(t, session, "get_page", map[string]any{
		"uri": "gnosis://test/note.md",
	})
	var page vault.Page
	decodeMCPResult(t, result, &page)
	if page.Document.URI != "gnosis://test/note.md" {
		t.Fatalf("page = %+v", page)
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

	conceptsResult := callMCPTool(t, session, "get_concepts", map[string]any{"type": "Note"})
	var concepts conceptsOutput
	decodeMCPResult(t, conceptsResult, &concepts)
	if concepts.Type != "Note" || len(concepts.Concepts) != 1 {
		t.Fatalf("concepts = %+v", concepts)
	}
	if concepts.Concepts[0]["uri"] != "gnosis://test/note.md" {
		t.Fatalf("concept = %+v", concepts.Concepts[0])
	}

	pageResult := callMCPTool(t, session, "get_page", map[string]any{
		"uri": "gnosis://test/note.md",
	})
	var page vault.Page
	decodeMCPResult(t, pageResult, &page)
	if page.Document.URI != "gnosis://test/note.md" || page.Document.Revision == "" {
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
	if len(query.Candidates) != 1 || query.Candidates[0].URI != "gnosis://test/note.md" {
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

func TestMCPSubprocess(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "gnosis")
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Env = append(os.Environ(), "GOCACHE="+filepath.Join(t.TempDir(), "go-cache"))
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build gnosis: %v\n%s", err, output)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var stderr bytes.Buffer
	command := exec.Command(binary, "serve", "mcp", "--vault", mcpTestVault(t))
	command.Stderr = &stderr
	command.Env = append(
		os.Environ(),
		"GNOSIS_DATABASE_URL=",
		"GNOSIS_EMBEDDING_URL=",
		"GNOSIS_EMBEDDING_MODEL=",
	)
	client := mcp.NewClient(&mcp.Implementation{Name: "gnosis-test", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: command}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })

	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Tools) != 4 {
		t.Fatalf("tools = %d, want 4", len(listed.Tools))
	}

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

	searchResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "search_knowledge",
		Arguments: map[string]any{
			"question": "small adequate design",
			"backend":  "lexical",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var query vault.QueryResult
	decodeMCPResult(t, searchResult, &query)
	if len(query.Candidates) == 0 || query.Candidates[0].URI != "gnosis://test/note.md" {
		t.Fatalf("query = %+v", query)
	}

	if err := session.Close(); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("server stderr = %q", stderr.String())
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
	writeCommandFile(t, workspace, "note.md", `---
type: Note
title: Keep it small
description: Prefer the smallest adequate design.
---

Use the simplest design that satisfies the current requirement.
`)
	return workspace
}

func httpTestVault(t *testing.T) string {
	t.Helper()
	workspace := commandVault(t)
	writeCommandFile(t, workspace, "note.md", `---
type: Note
title: Keep it small
description: Prefer the smallest adequate design.
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

type readyWriter struct {
	once  sync.Once
	ready chan struct{}
}

func (w *readyWriter) Write(data []byte) (int, error) {
	w.once.Do(func() { close(w.ready) })
	return len(data), nil
}
