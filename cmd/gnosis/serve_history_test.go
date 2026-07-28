package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gnosis/internal/vault"
)

func TestHistoryHTTPAndMCPParity(t *testing.T) {
	root := historyCommandVault(t)
	uri := "gnosis://test/note.md"
	expectedHistory, err := vault.ReadPageHistory(root, uri, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	from := expectedHistory.Entries[len(expectedHistory.Entries)-1].Revision
	allHistory, err := vault.ReadPageHistory(root, uri, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	from = allHistory.Entries[len(allHistory.Entries)-1].Revision
	to := allHistory.Entries[0].Revision
	expectedDiff, err := vault.DiffPage(root, uri, from, to, 1000)
	if err != nil {
		t.Fatal(err)
	}
	expectedChanges, err := vault.ChangesSince(root, "", 1)
	if err != nil {
		t.Fatal(err)
	}

	handler := newHTTPHandler(root)
	var httpHistory vault.PageHistoryResult
	getHistoryJSON(
		t,
		handler,
		"/api/v1/history?uri="+url.QueryEscape(uri)+"&limit=1",
		&httpHistory,
	)
	var httpDiff vault.PageDiffResult
	getHistoryJSON(
		t,
		handler,
		"/api/v1/diff?uri="+url.QueryEscape(uri)+
			"&from="+url.QueryEscape(from)+"&to="+url.QueryEscape(to)+"&limit=1000",
		&httpDiff,
	)
	var httpChanges vault.ChangeFeedResult
	getHistoryJSON(t, handler, "/api/v1/changes?limit=1", &httpChanges)

	session := connectMCPServer(t, newMCPServer(root))
	var mcpHistory vault.PageHistoryResult
	decodeMCPResult(t, callMCPTool(t, session, "get_history", map[string]any{
		"uri": uri, "limit": 1,
	}), &mcpHistory)
	var mcpDiff vault.PageDiffResult
	decodeMCPResult(t, callMCPTool(t, session, "get_diff", map[string]any{
		"uri": uri, "from_revision": from, "to_revision": to, "limit": 1000,
	}), &mcpDiff)
	var mcpChanges vault.ChangeFeedResult
	decodeMCPResult(t, callMCPTool(t, session, "get_changes", map[string]any{
		"limit": 1,
	}), &mcpChanges)

	for name, values := range map[string][]any{
		"history": {expectedHistory, httpHistory, mcpHistory},
		"diff":    {expectedDiff, httpDiff, mcpDiff},
		"changes": {expectedChanges, httpChanges, mcpChanges},
	} {
		if !reflect.DeepEqual(values[0], values[1]) ||
			!reflect.DeepEqual(values[0], values[2]) {
			t.Fatalf("%s parity = %#v", name, values)
		}
	}
}

func TestHistoryTransportErrorsAndTools(t *testing.T) {
	root := historyCommandVault(t)
	handler := newHTTPHandler(root)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/diff?uri="+
			url.QueryEscape("gnosis://test/note.md")+
			"&from=sha256:missing&to=sha256:also-missing",
		nil,
	)
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("diff status = %d", response.Code)
	}
	response = httptest.NewRecorder()
	request = httptest.NewRequest(
		http.MethodGet,
		"/api/v1/changes?cursor=not-a-cursor",
		nil,
	)
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("changes status = %d", response.Code)
	}

	session := connectMCPServer(t, newMCPServer(root))
	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"get_history", "get_diff", "get_changes"} {
		if !hasMCPTool(tools.Tools, name) {
			t.Fatalf("tools = %+v, missing %s", tools.Tools, name)
		}
	}
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "get_diff",
		Arguments: map[string]any{
			"uri":           "gnosis://test/note.md",
			"from_revision": "sha256:missing",
			"to_revision":   "sha256:also-missing",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || !strings.Contains(mcpResultText(result), vault.ErrRevisionNotFound.Error()) {
		t.Fatalf("MCP diff error = %+v", result)
	}
}

func getHistoryJSON(t *testing.T, handler http.Handler, endpoint string, target any) {
	t.Helper()
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, endpoint, nil)
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d", endpoint, response.Code)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatal(err)
	}
}
