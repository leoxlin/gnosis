package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"testing"

	"gnosis/internal/vault"
)

func pagesTestVault(t *testing.T) string {
	t.Helper()
	workspace := commandVault(t)
	writeCommandFile(t, workspace, "alpha.md", `---
type: Note
title: Alpha note
description: First test page.
---
`)
	writeCommandFile(t, workspace, "beta.md", `---
type: Note
title: Beta note
description: Second test page.
---
`)
	writeCommandFile(t, workspace, "delta.md", `---
type: Concept
title: Delta concept
description: Third test page.
---
`)
	writeCommandFile(t, workspace, "gamma.md", `---
type: Concept
title: Gamma concept
description: Fourth test page.
---
`)
	return workspace
}

func pageURIs(listing pagesResponse) []string {
	uris := make([]string, 0, len(listing.Pages))
	for _, page := range listing.Pages {
		uris = append(uris, page.URI)
	}
	return uris
}

func TestPagesHTTPPagination(t *testing.T) {
	handler := newHTTPHandler(pagesTestVault(t))

	var legacy pagesResponse
	getHistoryJSON(t, handler, "/api/v1/pages", &legacy)
	ordered := pageURIs(legacy)
	if len(ordered) < 4 || legacy.Total != len(ordered) || legacy.NextCursor != "" {
		t.Fatalf("legacy pages = %+v", legacy)
	}
	if !sort.StringsAreSorted(ordered) {
		t.Fatalf("pages are not in canonical-URI order = %v", ordered)
	}

	var first pagesResponse
	getHistoryJSON(t, handler, "/api/v1/pages?limit=3", &first)
	if !reflect.DeepEqual(pageURIs(first), ordered[:3]) || first.Total != len(ordered) ||
		first.NextCursor == "" {
		t.Fatalf("first page = %+v", first)
	}
	var second pagesResponse
	getHistoryJSON(
		t,
		handler,
		"/api/v1/pages?limit=1000&cursor="+url.QueryEscape(first.NextCursor),
		&second,
	)
	if !reflect.DeepEqual(pageURIs(second), ordered[3:]) || second.Total != len(ordered) ||
		second.NextCursor != "" {
		t.Fatalf("second page = %+v", second)
	}

	var filtered pagesResponse
	getHistoryJSON(t, handler, "/api/v1/pages?q=gamma", &filtered)
	if !reflect.DeepEqual(pageURIs(filtered), []string{"gnosis://test/gamma.md"}) || filtered.Total != 1 {
		t.Fatalf("q filter = %+v", filtered)
	}
	getHistoryJSON(t, handler, "/api/v1/pages?type=Concept", &filtered)
	if !reflect.DeepEqual(
		pageURIs(filtered),
		[]string{"gnosis://test/delta.md", "gnosis://test/gamma.md"},
	) || filtered.Total != 2 {
		t.Fatalf("type filter = %+v", filtered)
	}
	getHistoryJSON(t, handler, "/api/v1/pages?q=note&type=Note&limit=1", &filtered)
	if !reflect.DeepEqual(pageURIs(filtered), []string{"gnosis://test/alpha.md"}) ||
		filtered.Total != 2 || filtered.NextCursor == "" {
		t.Fatalf("combined filter = %+v", filtered)
	}
}

func TestPagesHTTPCursorAndLimitErrors(t *testing.T) {
	handler := newHTTPHandler(pagesTestVault(t))
	tests := []struct {
		name     string
		endpoint string
		status   int
	}{
		{name: "malformed cursor", endpoint: "/api/v1/pages?cursor=not-a-cursor!", status: http.StatusBadRequest},
		{
			name: "expired cursor",
			endpoint: "/api/v1/pages?cursor=" + url.QueryEscape(
				base64.RawURLEncoding.EncodeToString([]byte("gnosis://test/c.md")),
			),
			status: http.StatusGone,
		},
		{name: "non-integer limit", endpoint: "/api/v1/pages?limit=lots", status: http.StatusBadRequest},
		{name: "limit out of range", endpoint: "/api/v1/pages?limit=1001", status: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, test.endpoint, nil)
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("GET %s status = %d, want %d", test.endpoint, response.Code, test.status)
			}
		})
	}
}

func TestGraphTraversalHTTPParity(t *testing.T) {
	workspace := mcpTestVault(t)
	writeMCPGraphProcedureFixtures(t, workspace)
	handler := newHTTPHandler(workspace)
	session := connectMCPServer(t, newMCPServer(workspace))

	neighborArguments := map[string]any{
		"uri": "gnosis://test/source.md", "direction": "out", "limit": 1,
	}
	var httpNeighbors boundedGraphNeighbors
	getHistoryJSON(
		t,
		handler,
		"/api/v1/graph/neighbors?uri="+url.QueryEscape("gnosis://test/source.md")+"&direction=out&limit=1",
		&httpNeighbors,
	)
	var mcpGraph traceGraphOutput
	decodeMCPResult(t, callMCPTool(t, session, "trace_graph", neighborArguments), &mcpGraph)
	if mcpGraph.Neighbors == nil || !reflect.DeepEqual(*mcpGraph.Neighbors, httpNeighbors) {
		t.Fatalf("neighbors = %+v, want transport parity with %+v", httpNeighbors, mcpGraph.Neighbors)
	}

	pathArguments := map[string]any{
		"uri":        "gnosis://test/source.md",
		"target_uri": "gnosis://test/target.md",
		"direction":  "out",
		"relations":  []string{"supports"},
		"depth":      2,
	}
	var httpPath vault.GraphPath
	getHistoryJSON(
		t,
		handler,
		"/api/v1/graph/path?uri="+url.QueryEscape("gnosis://test/source.md")+
			"&target="+url.QueryEscape("gnosis://test/target.md")+
			"&direction=out&relation=supports&depth=2",
		&httpPath,
	)
	decodeMCPResult(t, callMCPTool(t, session, "trace_graph", pathArguments), &mcpGraph)
	if mcpGraph.Path == nil || !reflect.DeepEqual(*mcpGraph.Path, httpPath) {
		t.Fatalf("path = %+v, want transport parity with %+v", httpPath, mcpGraph.Path)
	}
}

func TestGraphTraversalHTTPErrors(t *testing.T) {
	workspace := mcpTestVault(t)
	writeMCPGraphProcedureFixtures(t, workspace)
	handler := newHTTPHandler(workspace)
	source := url.QueryEscape("gnosis://test/source.md")
	target := url.QueryEscape("gnosis://test/target.md")
	endpoints := map[string]string{
		"source uri":           "/api/v1/graph/neighbors?uri=bad",
		"unknown source":       "/api/v1/graph/neighbors?uri=" + url.QueryEscape("gnosis://test/missing.md"),
		"direction":            "/api/v1/graph/neighbors?uri=" + source + "&direction=sideways",
		"empty relation":       "/api/v1/graph/neighbors?uri=" + source + "&relation=",
		"neighbor depth":       "/api/v1/graph/neighbors?uri=" + source + "&depth=1",
		"neighbor target":      "/api/v1/graph/neighbors?uri=" + source + "&target=" + target,
		"neighbor limit range": "/api/v1/graph/neighbors?uri=" + source + "&limit=0",
		"path missing target":  "/api/v1/graph/path?uri=" + source,
		"path target uri":      "/api/v1/graph/path?uri=" + source + "&target=bad",
		"path depth":           "/api/v1/graph/path?uri=" + source + "&target=" + target + "&depth=-1",
		"path limit":           "/api/v1/graph/path?uri=" + source + "&target=" + target + "&limit=1",
		"path depth parse":     "/api/v1/graph/path?uri=" + source + "&target=" + target + "&depth=far",
	}
	for name, endpoint := range endpoints {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, endpoint, nil)
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("GET %s status = %d, want 400", endpoint, response.Code)
			}
		})
	}
}

func TestProceduresHTTPParity(t *testing.T) {
	workspace := mcpTestVault(t)
	writeMCPGraphProcedureFixtures(t, workspace)
	handler := newHTTPHandler(workspace)
	session := connectMCPServer(t, newMCPServer(workspace))

	var httpDiscovery getProceduresOutput
	getHistoryJSON(t, handler, "/api/v1/procedures?tag=test&tag=mcp", &httpDiscovery)
	var mcpProcedures getProceduresOutput
	decodeMCPResult(t, callMCPTool(t, session, "get_procedures", map[string]any{
		"tags": []string{"test", "mcp"},
	}), &mcpProcedures)
	if httpDiscovery.Mode != "discovery" || !reflect.DeepEqual(mcpProcedures, httpDiscovery) {
		t.Fatalf("discovery = %+v, want transport parity with %+v", httpDiscovery, mcpProcedures)
	}

	var unfiltered getProceduresOutput
	getHistoryJSON(t, handler, "/api/v1/procedures", &unfiltered)
	if unfiltered.Mode != "discovery" {
		t.Fatalf("unfiltered discovery = %+v", unfiltered)
	}
	found := false
	for _, procedure := range unfiltered.Procedures {
		if procedure["uri"] == "gnosis://test/procedures/model.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("unfiltered discovery missing test procedure = %+v", unfiltered.Procedures)
	}

	var httpContract getProceduresOutput
	getHistoryJSON(
		t,
		handler,
		"/api/v1/procedures?uri="+url.QueryEscape("gnosis://test/procedures/model.md"),
		&httpContract,
	)
	var mcpContract getProceduresOutput
	decodeMCPResult(t, callMCPTool(t, session, "get_procedures", map[string]any{
		"uri": "gnosis://test/procedures/model.md",
	}), &mcpContract)
	if httpContract.Mode != "contract" || httpContract.Procedure == nil ||
		!reflect.DeepEqual(mcpContract, httpContract) {
		t.Fatalf("contract = %+v, want transport parity with %+v", httpContract, mcpContract)
	}
}

func TestProceduresHTTPErrors(t *testing.T) {
	workspace := mcpTestVault(t)
	writeMCPGraphProcedureFixtures(t, workspace)
	handler := newHTTPHandler(workspace)
	endpoints := map[string]string{
		"unknown uri": "/api/v1/procedures?uri=" + url.QueryEscape("gnosis://test/procedures/missing.md"),
		"invalid uri": "/api/v1/procedures?uri=bad",
		"uri and tag": "/api/v1/procedures?uri=" + url.QueryEscape("gnosis://test/procedures/model.md") + "&tag=test",
		"empty tag":   "/api/v1/procedures?tag=",
	}
	for name, endpoint := range endpoints {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, endpoint, nil)
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("GET %s status = %d, want 400", endpoint, response.Code)
			}
		})
	}
}
