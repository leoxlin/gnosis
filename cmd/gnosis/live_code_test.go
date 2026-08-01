package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gnosis/internal/codeintel"
)

func TestLiveCodeServiceIsSharedAcrossMCPSessions(t *testing.T) {
	workspace := mcpTestVault(t)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	runCommandGit(t, "-C", workspace, "init")
	runCommandGit(t, "-C", workspace, "config", "user.email", "test@example.test")
	runCommandGit(t, "-C", workspace, "config", "user.name", "Test")
	writeCommandFile(t, workspace, "main.go", "package main\nfunc Initial() {}\n")
	runCommandGit(t, "-C", workspace, "add", "main.go")
	runCommandGit(t, "-C", workspace, "commit", "-m", "fixture")
	writeCommandFile(t, workspace, "gnosis.toml", `[vault]
vault_name = "test"
vault_root = "."
vault_index = false
vault_log = false

[[code_scopes]]
name = "app"
root = "."
languages = ["go"]
live = true
freshness_wait = "1s"
`)

	service, err := codeintel.OpenService(context.Background(), workspace)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close() })
	first := connectMCPServer(t, newMCPServerWithOptions(workspace, false, service))
	second := connectMCPServer(t, newMCPServerWithOptions(workspace, false, service))
	initialFirst := liveMCPStatus(t, first)
	initialSecond := liveMCPStatus(t, second)
	if initialFirst.Epoch == "" || initialFirst.Epoch != initialSecond.Epoch || initialFirst.Generation != initialSecond.Generation {
		t.Fatalf("sessions do not share live state: %+v, %+v", initialFirst, initialSecond)
	}

	writeCommandFile(t, workspace, "main.go", "package main\nfunc Changed() {}\n")
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		current := liveMCPStatus(t, first)
		if current.State == codeintel.LiveCurrent && current.Generation != initialFirst.Generation {
			fromSecond := liveMCPStatus(t, second)
			if current.Epoch != fromSecond.Epoch || current.Generation != fromSecond.Generation || current.Published != fromSecond.Published {
				t.Fatalf("sessions diverged after edit: %+v, %+v", current, fromSecond)
			}
			search := callMCPTool(t, second, "search_code", map[string]any{"scope": "app", "query": "Changed"})
			var result codeintel.SearchResult
			decodeMCPResult(t, search, &result)
			if result.Freshness == nil || result.Freshness.Epoch != current.Epoch {
				t.Fatalf("search freshness = %+v", result.Freshness)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("shared live workspace did not publish the edit")
}

func liveMCPStatus(t *testing.T, session *mcp.ClientSession) codeintel.StatusResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "get_code_index_status", Arguments: map[string]any{"scope": "app"}})
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var status codeintel.StatusResult
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatal(err)
	}
	return status
}
