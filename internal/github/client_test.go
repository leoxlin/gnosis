package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gnosis/internal/evidence"
)

func TestSyncCoversObjectsResumesAndNoops(t *testing.T) {
	var failIssues atomic.Bool
	failIssues.Store(true)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		path := request.URL.Path
		if path == "/repos/owner/repo" {
			response.Write([]byte(`{"visibility":"private"}`))
			return
		}
		if path == "/repos/owner/repo/issues" && failIssues.CompareAndSwap(true, false) {
			http.Error(response, "interrupted", http.StatusInternalServerError)
			return
		}
		switch path {
		case "/repos/owner/repo/pulls":
			response.Write([]byte(`[{"node_id":"PR_1","number":1,"updated_at":"2026-07-29T12:00:00Z","html_url":"https://example.test/pr/1"}]`))
		case "/repos/owner/repo/issues":
			response.Write([]byte(`[{"node_id":"I_1","updated_at":"2026-07-29T12:00:00Z"}]`))
		case "/repos/owner/repo/pulls/comments":
			response.Write([]byte(`[{"node_id":"RC_1","updated_at":"2026-07-29T12:00:00Z"}]`))
		case "/repos/owner/repo/issues/comments":
			response.Write([]byte(`[{"node_id":"IC_1","updated_at":"2026-07-29T12:00:00Z"}]`))
		case "/repos/owner/repo/commits":
			response.Write([]byte(`[{"sha":"abc","commit":{"committer":{"date":"2026-07-29T12:00:00Z"}}}]`))
		case "/repos/owner/repo/pulls/1/reviews":
			response.Write([]byte(`[{"node_id":"R_1","submitted_at":"2026-07-29T12:00:00Z"}]`))
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	store, err := evidence.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	client, err := New(Config{
		Vault: "test", Repository: "owner/repo", Token: "token",
		APIURL: server.URL, PerPage: 100, MaxPages: 20,
	}, store, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	client.now = func() time.Time {
		return time.Date(2026, 7, 29, 12, 1, 0, 0, time.UTC)
	}
	first, err := client.Sync(context.Background(), Options{})
	if err == nil || !strings.Contains(err.Error(), "interrupted") || first.Created != 1 {
		t.Fatalf("interrupted sync = %+v, %v", first, err)
	}
	second, err := client.Sync(context.Background(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if second.Created != 5 || second.Unchanged != 1 || !strings.HasPrefix(second.Cursor.Value, "complete:") {
		t.Fatalf("resumed sync = %+v", second)
	}
	third, err := client.Sync(context.Background(), Options{})
	if err != nil || third.Created != 0 || third.Unchanged != 6 {
		t.Fatalf("repeat sync = %+v, %v", third, err)
	}
	latest, err := store.Latest("test", "owner/repo")
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"pull_request", "issue", "review", "issue_comment", "review_comment", "commit"} {
		if _, ok := latest[kind+"\x00"+map[string]string{
			"pull_request": "PR_1", "issue": "I_1", "review": "R_1",
			"issue_comment": "IC_1", "review_comment": "RC_1", "commit": "abc",
		}[kind]]; !ok {
			t.Errorf("missing %s evidence", kind)
		}
	}
}

func TestSyncBoundsBackfillAndReportsRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/repos/owner/repo" {
			response.Write([]byte(`{"visibility":"public"}`))
			return
		}
		response.Write([]byte(`[
			{"node_id":"1","number":1,"updated_at":"2026-07-29T12:00:00Z"},
			{"node_id":"2","number":2,"updated_at":"2026-07-29T12:00:00Z"}
		]`))
	}))
	defer server.Close()
	store, _ := evidence.New(t.TempDir())
	client, _ := New(Config{
		Vault: "test", Repository: "owner/repo", Token: "token",
		APIURL: server.URL, PerPage: 2, MaxPages: 10,
	}, store, server.Client())
	result, err := client.Sync(context.Background(), Options{MaxItems: 1})
	if err != nil || result.Created != 1 || result.Cursor.Value != "pull_request:1" {
		t.Fatalf("bounded result = %+v, %v", result, err)
	}
	resumed, err := client.Sync(context.Background(), Options{MaxItems: 1})
	if err != nil || resumed.Created != 1 || resumed.Unchanged < 1 {
		t.Fatalf("resumed bound = %+v, %v", resumed, err)
	}

	rateServer := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("X-RateLimit-Reset", "123")
		response.WriteHeader(http.StatusForbidden)
	}))
	defer rateServer.Close()
	rateClient, _ := New(Config{
		Vault: "test", Repository: "owner/repo", Token: "token",
		APIURL: rateServer.URL, PerPage: 100, MaxPages: 10,
	}, store, rateServer.Client())
	rate, err := rateClient.Sync(context.Background(), Options{})
	if err != nil || !rate.RateLimited || rate.RateReset != "123" {
		t.Fatalf("rate result = %+v, %v", rate, err)
	}
}

func TestCompleteReconciliationAppendsTombstone(t *testing.T) {
	var present atomic.Bool
	present.Store(true)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/repos/owner/repo":
			response.Write([]byte(`{"visibility":"private"}`))
		case "/repos/owner/repo/issues":
			if present.Load() {
				response.Write([]byte(`[{"node_id":"I_1","updated_at":"2026-07-29T12:00:00Z"}]`))
			} else {
				response.Write([]byte(`[]`))
			}
		default:
			response.Write([]byte(`[]`))
		}
	}))
	defer server.Close()
	store, _ := evidence.New(t.TempDir())
	client, _ := New(Config{
		Vault: "test", Repository: "owner/repo", Token: "token",
		APIURL: server.URL, PerPage: 100, MaxPages: 20,
	}, store, server.Client())
	if result, err := client.Sync(context.Background(), Options{}); err != nil || result.Created != 1 {
		t.Fatalf("initial result = %+v, %v", result, err)
	}
	present.Store(false)
	result, err := client.Sync(context.Background(), Options{Reconcile: true})
	if err != nil || result.Tombstoned != 1 {
		t.Fatalf("reconcile result = %+v, %v", result, err)
	}
}
