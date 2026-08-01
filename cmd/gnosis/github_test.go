package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	toon "github.com/toon-format/toon-go"
	"gnosis/internal/evidence"
	githubsource "gnosis/internal/github"
)

func TestGitHubResultContainsDeterministicOutcomes(t *testing.T) {
	var output bytes.Buffer
	err := writeGitHubResult(&output, "owner/repo", githubsource.Result{
		Created: 2, Unchanged: 3, Tombstoned: 4, Rejected: 5,
		RateLimited: true, RateReset: "123",
		Cursor: evidence.Cursor{Value: "issue:2"},
	})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := toon.Decode(output.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	fields := decoded.(map[string]any)
	for _, key := range []string{
		"created", "unchanged", "tombstoned", "rejected",
		"rate_limited", "rate_reset", "cursor",
	} {
		if _, ok := fields[key]; !ok {
			t.Errorf("missing %s in %s", key, output.String())
		}
	}
}

func TestGitHubWebhookRouteIsOptIn(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/github/webhooks/owner/repo", strings.NewReader("{}"))
	disabled := httptest.NewRecorder()
	newHTTPHandlerWithCodeService(t.TempDir(), false, false, nil).ServeHTTP(disabled, request)
	if disabled.Code != http.StatusNotFound {
		t.Fatalf("disabled status = %d", disabled.Code)
	}

	root := t.TempDir()
	config := `[vault]
vault_name = "test"
vault_root = "."

[[github]]
repository = "owner/repo"
evidence_dir = "` + filepath.Join(root, "evidence") + `"
token_env = "GITHUB_TOKEN"
webhook_secret_env = "WEBHOOK_SECRET"
`
	if err := os.WriteFile(filepath.Join(root, "gnosis.toml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	enabled := httptest.NewRecorder()
	newHTTPHandlerWithCodeService(root, false, true, nil).ServeHTTP(enabled, request)
	if enabled.Code == http.StatusNotFound {
		t.Fatalf("enabled route was not registered")
	}
}
