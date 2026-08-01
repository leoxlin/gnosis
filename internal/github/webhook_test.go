package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestWebhookVerifiesDeduplicatesAndRejectsUnsupported(t *testing.T) {
	root := t.TempDir()
	evidenceDir := filepath.Join(root, "evidence")
	configureWebhook(t, root, evidenceDir, 2048)
	t.Setenv("WEBHOOK_SECRET", "secret")
	mux := http.NewServeMux()
	mux.HandleFunc("POST /github/{owner}/{repository}", Webhook(root))
	body := []byte(`{"repository":{"full_name":"owner/repo","visibility":"private"},"issue":{"node_id":"I_1","updated_at":"2026-07-29T12:00:00Z"}}`)

	first := deliverWebhook(t, mux, body, "issues", "delivery-1", signature(body, "secret"))
	if first.Code != http.StatusOK || !strings.Contains(first.Body.String(), `"created":1`) {
		t.Fatalf("first = %d %s", first.Code, first.Body.String())
	}
	duplicate := deliverWebhook(t, mux, body, "issues", "delivery-1", signature(body, "secret"))
	if duplicate.Code != http.StatusOK || !strings.Contains(duplicate.Body.String(), `"unchanged":1`) {
		t.Fatalf("duplicate = %d %s", duplicate.Code, duplicate.Body.String())
	}
	unsupported := deliverWebhook(t, mux, body, "deployment", "delivery-2", signature(body, "secret"))
	if unsupported.Code != http.StatusAccepted || !strings.Contains(unsupported.Body.String(), `"rejected":1`) {
		t.Fatalf("unsupported = %d %s", unsupported.Code, unsupported.Body.String())
	}
	invalid := deliverWebhook(t, mux, body, "issues", "delivery-3", "sha256:bad")
	if invalid.Code != http.StatusUnauthorized {
		t.Fatalf("invalid signature status = %d", invalid.Code)
	}
}

func TestWebhookBoundsBodyAndMatchesSynchronizedRecord(t *testing.T) {
	root := t.TempDir()
	evidenceDir := filepath.Join(root, "evidence")
	configureWebhook(t, root, evidenceDir, 256)
	t.Setenv("WEBHOOK_SECRET", "secret")
	mux := http.NewServeMux()
	mux.HandleFunc("POST /github/{owner}/{repository}", Webhook(root))
	oversized := []byte(strings.Repeat("x", 257))
	response := deliverWebhook(t, mux, oversized, "issues", "large", signature(oversized, "secret"))
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized status = %d", response.Code)
	}

	body := []byte(`{"repository":{"full_name":"owner/repo","visibility":"private"},"issue":{"node_id":"I_1","updated_at":"2026-07-29T12:00:00Z"}}`)
	first := deliverWebhook(t, mux, body, "issues", "sync-equivalent", signature(body, "secret"))
	if first.Code != http.StatusOK || !strings.Contains(first.Body.String(), `"created":1`) {
		t.Fatalf("first = %d %s", first.Code, first.Body.String())
	}
	second := deliverWebhook(t, mux, body, "issues", "another-delivery", signature(body, "secret"))
	if second.Code != http.StatusOK || !strings.Contains(second.Body.String(), `"unchanged":1`) {
		t.Fatalf("equivalent = %d %s", second.Code, second.Body.String())
	}
}

func configureWebhook(t *testing.T, root, evidenceDir string, maxBody int) {
	t.Helper()
	content := `[vault]
vault_name = "test"
vault_root = "."

[[github]]
repository = "owner/repo"
evidence_dir = "` + evidenceDir + `"
token_env = "GITHUB_TOKEN"
webhook_secret_env = "WEBHOOK_SECRET"
max_body_bytes = ` + strconv.Itoa(maxBody) + `
`
	if err := os.WriteFile(filepath.Join(root, "gnosis.toml"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func deliverWebhook(t *testing.T, handler http.Handler, body []byte, event, delivery, signature string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/github/owner/repo", strings.NewReader(string(body)))
	request.Header.Set("X-GitHub-Event", event)
	request.Header.Set("X-GitHub-Delivery", delivery)
	request.Header.Set("X-Hub-Signature-256", signature)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func signature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
