package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gnosis/internal/vault"
)

func TestKnowledgeAuditAdaptersHaveParityAndDoNotMutate(t *testing.T) {
	workspace := t.TempDir()
	writeCommandFile(t, workspace, "gnosis.toml", `[vault]
vault_name = "test"
vault_root = "."
entry_points = ["gnosis://test/entry.md"]
vault_index = false
vault_log = false
`)
	for name, title := range map[string]string{
		"entry.md": "Entry",
		"a.md":     "A",
		"b.md":     "B",
	} {
		writeCommandFile(t, workspace, name, "---\ntype: Note\ntitle: "+title+"\n---\n")
	}
	request := vault.KnowledgeAuditRequest{
		Classes: []vault.FindingClass{vault.FindingOrphan},
		Types:   []string{"Note"}, PageLimit: 10, FindingLimit: 10,
	}
	before := commandFixtureFiles(t, workspace)

	direct, err := vault.AuditKnowledge(workspace, request)
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := run([]string{
		"--vault", workspace, "audit", "knowledge",
		"--class", "orphan", "--type", "Note",
		"--page-limit", "10", "--finding-limit", "10",
	}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(newHTTPHandler(workspace))
	t.Cleanup(server.Close)
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.Post(
		server.URL+"/api/v1/audit/knowledge", "application/json", bytes.NewReader(body),
	)
	if err != nil {
		t.Fatal(err)
	}
	var httpResult vault.KnowledgeAuditResult
	if err := json.NewDecoder(response.Body).Decode(&httpResult); err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("HTTP audit status = %d", response.StatusCode)
	}

	session := connectMCPServer(t, newMCPServer(workspace))
	mcpResult := callMCPTool(t, session, "audit_knowledge", map[string]any{
		"classes":       []string{"orphan"},
		"types":         []string{"Note"},
		"page_limit":    10,
		"finding_limit": 10,
	})
	var mcpAudit vault.KnowledgeAuditResult
	decodeMCPResult(t, mcpResult, &mcpAudit)

	directJSON, _ := json.Marshal(direct)
	httpJSON, _ := json.Marshal(httpResult)
	mcpJSON, _ := json.Marshal(mcpAudit)
	if !bytes.Equal(directJSON, httpJSON) || !bytes.Equal(directJSON, mcpJSON) {
		t.Fatalf("adapter mismatch:\ndirect=%+v\nhttp=%+v\nmcp=%+v", direct, httpResult, mcpAudit)
	}
	for _, finding := range direct.Findings {
		if !strings.Contains(stdout.String(), finding.ID) ||
			!strings.Contains(stdout.String(), "class: "+string(finding.Class)) {
			t.Fatalf("CLI output missing finding %+v:\n%s", finding, stdout.String())
		}
		for _, uri := range finding.URIs {
			if !strings.Contains(stdout.String(), uri) {
				t.Fatalf("CLI output missing URI %s:\n%s", uri, stdout.String())
			}
		}
	}
	if after := commandFixtureFiles(t, workspace); !reflect.DeepEqual(before, after) {
		t.Fatalf("audit adapters mutated vault: before=%v after=%v", before, after)
	}
}

func TestKnowledgeAuditAdaptersRejectUnknownClass(t *testing.T) {
	workspace := mcpTestVault(t)
	var stdout, stderr bytes.Buffer
	err := run([]string{
		"--vault", workspace, "audit", "knowledge", "--class", "unknown",
	}, &stdout, &stderr)
	if err == nil {
		t.Fatal("CLI accepted unknown audit class")
	}

	server := httptest.NewServer(newHTTPHandler(workspace))
	t.Cleanup(server.Close)
	response, err := http.Post(
		server.URL+"/api/v1/audit/knowledge",
		"application/json",
		bytes.NewBufferString(`{"classes":["unknown"]}`),
	)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("HTTP unknown class status = %d", response.StatusCode)
	}
	_ = response.Body.Close()

	session := connectMCPServer(t, newMCPServer(workspace))
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "audit_knowledge", Arguments: map[string]any{
			"classes": []string{"unknown"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatalf("MCP accepted unknown class: %+v", result)
	}
}

func commandFixtureFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		result[relative] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}
