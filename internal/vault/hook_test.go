package vault

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHookConfigurationValidation(t *testing.T) {
	valid := HookConfig{Name: "all", Kind: "command", Scope: "vault", Command: []string{"true"}}
	tests := []struct {
		name  string
		hooks []HookConfig
		want  string
	}{
		{"duplicate", []HookConfig{valid, valid}, "duplicated"},
		{"kind", []HookConfig{{Name: "bad", Kind: "plugin", Scope: "vault"}}, "kind"},
		{"scope", []HookConfig{{Name: "bad", Kind: "command", Scope: "all", Command: []string{"true"}}}, "scope"},
		{"vault target", []HookConfig{{Name: "bad", Kind: "command", Scope: "vault", Target: "gnosis://test/note.md", Command: []string{"true"}}}, "target"},
		{"page target", []HookConfig{{Name: "bad", Kind: "command", Scope: "page", Target: "note.md", Command: []string{"true"}}}, "canonical URI"},
		{"other vault", []HookConfig{{Name: "bad", Kind: "command", Scope: "page", Target: "gnosis://other/note.md", Command: []string{"true"}}}, "vault"},
		{"timeout", []HookConfig{{Name: "bad", Kind: "command", Scope: "vault", Timeout: "0s", Command: []string{"true"}}}, "greater than zero"},
		{"command", []HookConfig{{Name: "bad", Kind: "command", Scope: "vault"}}, "executable"},
		{"command fields", []HookConfig{{Name: "bad", Kind: "command", Scope: "vault", Command: []string{"true"}, URL: "https://example.test"}}, "must not set"},
		{"webhook command", []HookConfig{{Name: "bad", Kind: "webhook", Scope: "vault", URL: "https://example.test", Command: []string{"true"}}}, "must not set command"},
		{"webhook scheme", []HookConfig{{Name: "bad", Kind: "webhook", Scope: "vault", URL: "http://example.test"}}, "HTTPS"},
		{"secret name", []HookConfig{{Name: "bad", Kind: "webhook", Scope: "vault", URL: "https://example.test", SecretEnv: "BAD-NAME"}}, "environment variable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateHooks(test.hooks, "test")
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
	if err := validateHooks([]HookConfig{
		valid,
		{Name: "page", Kind: "webhook", Scope: "page", Target: "gnosis://test/notes/a.md", URL: "https://example.test", Timeout: "1s", SecretEnv: "HOOK_SECRET"},
		{Name: "prefix", Kind: "command", Scope: "prefix", Target: "gnosis://test/notes", Command: []string{"true"}},
	}, "test"); err != nil {
		t.Fatal(err)
	}
}

func TestHookMatchingIsExactAndSegmentAware(t *testing.T) {
	tests := []struct {
		hook HookConfig
		uri  string
		want bool
	}{
		{HookConfig{Scope: "vault"}, "gnosis://test/anything.md", true},
		{HookConfig{Scope: "page", Target: "gnosis://test/notes/a.md"}, "gnosis://test/notes/a.md", true},
		{HookConfig{Scope: "page", Target: "gnosis://test/notes/a.md"}, "gnosis://test/notes/a.md/child", false},
		{HookConfig{Scope: "prefix", Target: "gnosis://test/notes"}, "gnosis://test/notes/a.md", true},
		{HookConfig{Scope: "prefix", Target: "gnosis://test/notes"}, "gnosis://test/notes-other/a.md", false},
	}
	for _, test := range tests {
		if got := test.hook.matches(test.uri); got != test.want {
			t.Fatalf("matches(%q, %q) = %t, want %t", test.hook.Scope, test.uri, got, test.want)
		}
	}
}

func TestVaultWriteEventIsStableAndBounded(t *testing.T) {
	prepared := preparedDocumentWrite{
		current: &effectivePage{document: Document{Revision: "sha256:old"}},
		candidate: &effectivePage{document: Document{
			URI:      "gnosis://test/notes/a.md",
			Revision: "sha256:new",
			Origin:   Origin{Vault: "test", Kind: OriginLocal, Path: "/vault/notes/a.md"},
		}},
	}
	now := time.Date(2026, 7, 29, 13, 0, 0, 123, time.UTC)
	first := newVaultWriteEvent(prepared, "update", "sha256:change", now)
	second := newVaultWriteEvent(prepared, "update", "sha256:change", now.Add(time.Hour))
	if first.ID != second.ID || first.ID == "" || first.Version != 1 {
		t.Fatalf("events = %+v %+v", first, second)
	}
	body, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"version":1`, `"operation":"update"`, `"knowledge_change":"sha256:change"`} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("event = %s, want %s", body, want)
		}
	}
	if strings.Contains(string(body), "content") {
		t.Fatalf("event contains page content: %s", body)
	}
}

func TestDispatchHooksCommandsAreBoundedAndContinue(t *testing.T) {
	t.Setenv("GO_WANT_HOOK_HELPER", "1")
	t.Setenv("HOOK_SECRET", "do-not-print")
	event := testHookEvent()
	hooks := []HookConfig{
		{Name: "unmatched", Kind: "command", Scope: "page", Target: "gnosis://test/other.md", Command: hookHelperCommand("success")},
		{Name: "failed", Kind: "command", Scope: "vault", Command: hookHelperCommand("fail")},
		{Name: "large", Kind: "command", Scope: "vault", Command: hookHelperCommand("large")},
		{Name: "secret", Kind: "command", Scope: "vault", Command: hookHelperCommand("secret")},
		{Name: "last", Kind: "command", Scope: "vault", Command: hookHelperCommand("success")},
		{Name: "signer", Kind: "webhook", Scope: "page", Target: "gnosis://test/other.md", URL: "https://example.test", SecretEnv: "HOOK_SECRET"},
	}
	results := dispatchHooks(context.Background(), hooks, event)
	if len(results) != 4 {
		t.Fatalf("results = %+v", results)
	}
	if results[0].Name != "failed" || results[0].Status != "failed" || results[0].ExitCode == nil || *results[0].ExitCode != 7 {
		t.Fatalf("failed result = %+v", results[0])
	}
	if results[1].Name != "large" || len(results[1].Output) != hookOutputLimit {
		t.Fatalf("large result = %+v, output len %d", results[1], len(results[1].Output))
	}
	if strings.Contains(results[2].Output, "do-not-print") || !strings.Contains(results[2].Output, "[REDACTED]") {
		t.Fatalf("secret result = %+v", results[2])
	}
	if results[3].Name != "last" || results[3].Status != "success" {
		t.Fatalf("last result = %+v", results[3])
	}
}

func TestDispatchHooksTimeoutAndCancellation(t *testing.T) {
	t.Setenv("GO_WANT_HOOK_HELPER", "1")
	event := testHookEvent()
	timeout := dispatchHooks(context.Background(), []HookConfig{{
		Name: "timeout", Kind: "command", Scope: "vault", Timeout: "10ms", Command: hookHelperCommand("wait"),
	}}, event)
	if len(timeout) != 1 || timeout[0].Status != "timeout" {
		t.Fatalf("timeout = %+v", timeout)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	canceled := dispatchHooks(ctx, []HookConfig{{
		Name: "canceled", Kind: "command", Scope: "vault", Command: hookHelperCommand("wait"),
	}}, event)
	if len(canceled) != 1 || canceled[0].Status != "canceled" {
		t.Fatalf("canceled = %+v", canceled)
	}
}

func TestDispatchHooksPostsExactSignedEvent(t *testing.T) {
	t.Setenv("HOOK_SECRET", "test-secret")
	event := testHookEvent()
	var received []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		var err error
		received, err = io.ReadAll(request.Body)
		if err != nil {
			t.Error(err)
		}
		mac := hmac.New(sha256.New, []byte("test-secret"))
		_, _ = mac.Write(received)
		wantSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if got := request.Header.Get("X-Gnosis-Signature"); got != wantSignature {
			t.Errorf("signature = %q, want %q", got, wantSignature)
		}
		if request.Header.Get("X-Gnosis-Event-ID") != event.ID ||
			request.Header.Get("X-Gnosis-Event-Version") != "1" {
			t.Errorf("event headers = %v", request.Header)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	results := dispatchHooks(context.Background(), []HookConfig{{
		Name: "webhook", Kind: "webhook", Scope: "vault", URL: server.URL, SecretEnv: "HOOK_SECRET",
	}}, event)
	want, _ := json.Marshal(event)
	if string(received) != string(want) {
		t.Fatalf("body = %s, want %s", received, want)
	}
	if len(results) != 1 || results[0].Status != "success" || results[0].HTTPStatus != http.StatusNoContent {
		t.Fatalf("results = %+v", results)
	}
}

func TestDispatchHooksReportsHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		http.Error(response, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	results := dispatchHooks(context.Background(), []HookConfig{{
		Name: "webhook", Kind: "webhook", Scope: "vault", URL: server.URL,
	}}, testHookEvent())
	if len(results) != 1 || results[0].Status != "failed" ||
		results[0].HTTPStatus != http.StatusServiceUnavailable ||
		!strings.Contains(results[0].Error, "503") {
		t.Fatalf("results = %+v", results)
	}
}

func TestWriteDocumentDispatchesOnlyChangedCommittedWrites(t *testing.T) {
	t.Setenv("GO_WANT_HOOK_HELPER", "1")
	root := t.TempDir()
	failed := hookHelperCommand("fail")
	succeeded := hookHelperCommand("success")
	writeConfig(t, root, fmt.Sprintf(`[vault]
vault_name = "test"
vault_root = "."

[[hooks]]
name = "failed"
kind = "command"
scope = "vault"
command = [%q, %q, %q, %q]

[[hooks]]
name = "succeeded"
kind = "command"
scope = "vault"
command = [%q, %q, %q, %q]
`, failed[0], failed[1], failed[2], failed[3], succeeded[0], succeeded[1], succeeded[2], succeeded[3]))
	write(t, root, "types/note.md", "---\ntype: Concept\ntitle: Note\npath: notes\n---\n")
	content := []byte("---\ntype: Note\ntitle: Hooked\n---\n")
	first, err := WriteDocument(context.Background(), root, "gnosis://test/notes/hooked.md", content, false)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed || first.Operation != "create" || len(first.Deliveries) != 2 ||
		first.Deliveries[0].Status != "failed" || first.Deliveries[1].Status != "success" {
		t.Fatalf("first = %+v", first)
	}
	var event VaultWriteEvent
	if err := json.Unmarshal([]byte(first.Deliveries[1].Output), &event); err != nil {
		t.Fatal(err)
	}
	if event.URI != first.URI || event.NewRevision != first.Revision || event.Operation != "create" {
		t.Fatalf("event = %+v, result = %+v", event, first)
	}
	second, err := WriteDocument(context.Background(), root, "gnosis://test/notes/hooked.md", content, true)
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed || second.Operation != "no-op" || len(second.Deliveries) != 0 {
		t.Fatalf("second = %+v", second)
	}

	updatedContent := "---\ntype: Note\ntitle: Hooked\n---\n\nupdated\n"
	plan, err := PlanKnowledgeChange(root, KnowledgeChangeInput{
		URI:              first.URI,
		Candidate:        updatedContent,
		ExpectedRevision: second.Revision,
	})
	if err != nil || !plan.Applicable {
		t.Fatalf("plan = %+v, err = %v", plan, err)
	}
	updated, err := ApplyKnowledgeChange(context.Background(), root, KnowledgeChangeInput{
		URI:              first.URI,
		Candidate:        updatedContent,
		ExpectedRevision: second.Revision,
	}, plan.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Operation != "update" || len(updated.Deliveries) != 2 {
		t.Fatalf("updated = %+v", updated)
	}
	if err := json.Unmarshal([]byte(updated.Deliveries[1].Output), &event); err != nil {
		t.Fatal(err)
	}
	if event.KnowledgeChange == nil || *event.KnowledgeChange != plan.Digest {
		t.Fatalf("knowledge change event = %+v", event)
	}

	target := []byte("---\ntype: Note\ntitle: Target\n---\n")
	if _, err := WriteDocument(context.Background(), root, "gnosis://test/notes/target.md", target, false); err != nil {
		t.Fatal(err)
	}
	superseded := []byte("---\ntype: Note\ntitle: Hooked\nsuperseded_by: target.md\n---\n")
	supersession, err := WriteDocument(context.Background(), root, first.URI, superseded, true)
	if err != nil || supersession.Operation != "supersession" {
		t.Fatalf("supersession = %+v, err = %v", supersession, err)
	}
	archived := []byte("---\ntype: Note\ntitle: Hooked\nstatus: archived\nsuperseded_by: target.md\n---\n")
	archive, err := WriteDocument(context.Background(), root, first.URI, archived, true)
	if err != nil || archive.Operation != "archive" {
		t.Fatalf("archive = %+v, err = %v", archive, err)
	}
}

func TestInvalidPlansAndDerivedWritesDoNotDispatchHooks(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		response.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	root := t.TempDir()
	writeConfig(t, root, fmt.Sprintf(`[vault]
vault_name = "test"
vault_root = "."

[[hooks]]
name = "all"
kind = "webhook"
scope = "vault"
url = %q
`, server.URL))
	write(t, root, "types/note.md", "---\ntype: Concept\ntitle: Note\npath: notes\n---\n")
	write(t, root, "notes/existing.md", "---\ntype: Note\ntitle: Existing\n---\n")

	plan, err := PlanKnowledgeChange(root, KnowledgeChangeInput{
		URI:            "gnosis://test/notes/planned.md",
		Candidate:      "---\ntype: Note\ntitle: Planned\n---\n",
		ExpectedAbsent: true,
	})
	if err != nil || !plan.Applicable {
		t.Fatalf("plan = %+v, err = %v", plan, err)
	}
	invalid, err := PlanKnowledgeChange(root, KnowledgeChangeInput{
		URI:            "gnosis://test/notes/invalid.md",
		Candidate:      "---\ntitle: Invalid\n---\n",
		ExpectedAbsent: true,
	})
	if err != nil || invalid.Applicable {
		t.Fatalf("invalid plan = %+v, err = %v", invalid, err)
	}
	if _, _, err := GenerateWorkspaceIndexes(root, IndexOptions{Overwrite: true}); err != nil {
		t.Fatal(err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want no hooks for plans, validation, or indexes", requests)
	}
}

func TestFailedPublicationDoesNotDispatchHooks(t *testing.T) {
	fixture := newGitRemoteFixture(t, "https://example.test/team/hooked.git")
	configureTestRemoteVault(t, fixture)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		response.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	configPath := filepath.Join(fixture.seed, "gnosis.toml")
	config, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	config = append(config, []byte(fmt.Sprintf(`
[[hooks]]
name = "all"
kind = "webhook"
scope = "vault"
url = %q
`, server.URL))...)
	if err := os.WriteFile(configPath, config, 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, "-C", fixture.seed, "add", "gnosis.toml")
	runGit(t, "-C", fixture.seed, "commit", "-m", "configure hook")
	runGit(t, "-C", fixture.seed, "push", fixture.remote, "main")
	rejectTestPushes(t, fixture.remote)

	content := []byte("---\ntype: Note\ntitle: Rejected\n---\n")
	if _, err := WriteDocument(context.Background(), fixture.url, "gnosis://remote/notes/rejected-hook.md", content, false); err == nil {
		t.Fatal("write succeeded with rejected publication")
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want no delivery before publication succeeds", requests)
	}
}

func TestHookHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HOOK_HELPER") != "1" {
		return
	}
	separator := 0
	for i, argument := range os.Args {
		if argument == "--" {
			separator = i
			break
		}
	}
	mode := os.Args[separator+1]
	body, _ := io.ReadAll(os.Stdin)
	switch mode {
	case "success":
		_, _ = os.Stdout.Write(body)
	case "fail":
		os.Exit(7)
	case "large":
		_, _ = io.WriteString(os.Stdout, strings.Repeat("x", hookOutputLimit*2))
	case "secret":
		_, _ = io.WriteString(os.Stdout, os.Getenv("HOOK_SECRET"))
	case "wait":
		time.Sleep(time.Second)
	}
	os.Exit(0)
}

func hookHelperCommand(mode string) []string {
	return []string{os.Args[0], "-test.run=TestHookHelperProcess", "--", mode}
}

func testHookEvent() VaultWriteEvent {
	return VaultWriteEvent{
		Version:     1,
		ID:          "sha256:event",
		Vault:       "test",
		URI:         "gnosis://test/notes/a.md",
		Operation:   "update",
		NewRevision: "sha256:new",
		Origin:      Origin{Vault: "test", Kind: OriginLocal},
		OccurredAt:  "2026-07-29T13:00:00Z",
	}
}
