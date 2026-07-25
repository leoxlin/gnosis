package memory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestNewFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{
			name: "hosted defaults",
			env: map[string]string{
				EnvAPIKey: "key", EnvUserID: "user", EnvAgentID: "agent",
			},
		},
		{
			name: "hosted alternate url",
			env: map[string]string{
				EnvAPIKey: "key", EnvUserID: "user", EnvAgentID: "agent",
				EnvBaseURL: "https://memory.example.com",
			},
		},
		{
			name: "self hosted",
			env: map[string]string{
				EnvAPIKey: "key", EnvUserID: "user", EnvAgentID: "agent",
				EnvProvider: ProviderSelfHosted, EnvBaseURL: "http://localhost:8888",
			},
		},
		{
			name: "missing api key",
			env: map[string]string{
				EnvUserID: "user", EnvAgentID: "agent",
			},
			wantErr: EnvAPIKey,
		},
		{
			name: "missing user",
			env: map[string]string{
				EnvAPIKey: "key", EnvAgentID: "agent",
			},
			wantErr: EnvUserID,
		},
		{
			name: "missing agent",
			env: map[string]string{
				EnvAPIKey: "key", EnvUserID: "user",
			},
			wantErr: EnvAgentID,
		},
		{
			name: "unknown provider",
			env: map[string]string{
				EnvAPIKey: "key", EnvUserID: "user", EnvAgentID: "agent",
				EnvProvider: "other",
			},
			wantErr: EnvProvider,
		},
		{
			name: "self hosted without url",
			env: map[string]string{
				EnvAPIKey: "key", EnvUserID: "user", EnvAgentID: "agent",
				EnvProvider: ProviderSelfHosted,
			},
			wantErr: EnvBaseURL,
		},
		{
			name: "invalid url",
			env: map[string]string{
				EnvAPIKey: "key", EnvUserID: "user", EnvAgentID: "agent",
				EnvBaseURL: "ftp://memory.example.com",
			},
			wantErr: "http or https",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clearMemoryEnv(t)
			for name, value := range test.env {
				t.Setenv(name, value)
			}
			client, err := NewFromEnv()
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if client.entity.UserID != "user" || client.entity.AgentID != "agent" {
				t.Fatalf("entity = %+v", client.entity)
			}
		})
	}
}

func TestHostedAddAndSearch(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		if got := request.Header.Get("Authorization"); got != "Token key" {
			t.Errorf("authorization = %q", got)
		}
		if got := request.Header.Get("X-API-Key"); got != "" {
			t.Errorf("x-api-key = %q", got)
		}
		body := decodeBody(t, request)
		switch request.URL.Path {
		case "/v3/memories/add/":
			if body["user_id"] != "user" || body["agent_id"] != "agent" {
				t.Errorf("add identity = %#v", body)
			}
			if body["infer"] != true {
				t.Errorf("infer = %#v", body["infer"])
			}
			messages, ok := body["messages"].([]any)
			if !ok || len(messages) != 1 {
				t.Fatalf("messages = %#v", body["messages"])
			}
			message := messages[0].(map[string]any)
			if message["role"] != "user" || message["content"] != "I prefer dark mode" {
				t.Errorf("message = %#v", message)
			}
			writeJSON(t, response, []map[string]any{{
				"id": "memory-1", "event": "ADD",
				"data": map[string]any{"memory": "Prefers dark mode"},
			}})
		case "/v3/memories/search/":
			if body["query"] != "theme preference" || body["top_k"] != float64(1) {
				t.Errorf("search = %#v", body)
			}
			filters := body["filters"].(map[string]any)
			if filters["user_id"] != "user" || filters["agent_id"] != "agent" {
				t.Errorf("filters = %#v", filters)
			}
			writeJSON(t, response, map[string]any{"results": []map[string]any{
				{
					"id": "memory-1", "memory": "Prefers dark mode", "score": 0.9,
					"metadata": map[string]any{"source": "conversation"},
				},
				{"id": "unexpected", "memory": "must be bounded"},
			}})
		default:
			t.Errorf("unexpected request: %s %s", request.Method, request.URL.Path)
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	client, err := New(Config{
		APIKey: "key", UserID: "user", AgentID: "agent", BaseURL: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	added, err := client.Add(context.Background(), "I prefer dark mode")
	if err != nil {
		t.Fatal(err)
	}
	if added.Count != 1 || added.Memories[0].ID != "memory-1" ||
		added.Memories[0].Text != "Prefers dark mode" ||
		added.Memories[0].Event != "ADD" {
		t.Fatalf("added = %+v", added)
	}

	limit := 1
	found, err := client.Search(context.Background(), "theme preference", &limit)
	if err != nil {
		t.Fatal(err)
	}
	if found.Count != 1 || found.Memories[0].ID != "memory-1" ||
		found.Memories[0].Score == nil || *found.Memories[0].Score != 0.9 ||
		found.Memories[0].Metadata["source"] != "conversation" {
		t.Fatalf("found = %+v", found)
	}
	if requests.Load() != 2 {
		t.Fatalf("requests = %d", requests.Load())
	}
}

func TestSelfHostedAddAndDefaultSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("X-API-Key"); got != "key" {
			t.Errorf("x-api-key = %q", got)
		}
		if got := request.Header.Get("Authorization"); got != "" {
			t.Errorf("authorization = %q", got)
		}
		body := decodeBody(t, request)
		switch request.URL.Path {
		case "/memories":
			if body["user_id"] != "user" || body["agent_id"] != "agent" {
				t.Errorf("add identity = %#v", body)
			}
			writeJSON(t, response, map[string]any{"results": []map[string]any{{
				"id": "memory-1", "memory": "Uses self-hosted memory",
			}}})
		case "/search":
			if body["top_k"] != float64(DefaultSearchLimit) {
				t.Errorf("top_k = %#v", body["top_k"])
			}
			filters := body["filters"].(map[string]any)
			if filters["user_id"] != "user" || filters["agent_id"] != "agent" {
				t.Errorf("filters = %#v", filters)
			}
			writeJSON(t, response, map[string]any{"results": []any{}})
		default:
			t.Errorf("unexpected request: %s %s", request.Method, request.URL.Path)
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	client, err := New(Config{
		APIKey: "key", UserID: "user", AgentID: "agent",
		Provider: ProviderSelfHosted, BaseURL: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Add(context.Background(), "Uses self-hosted memory"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Search(context.Background(), "memory host", nil); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidOperationsDoNotSendRequests(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	defer server.Close()

	client, err := New(Config{
		APIKey: "key", UserID: "user", AgentID: "agent", BaseURL: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Add(context.Background(), " \t"); err == nil {
		t.Fatal("blank add succeeded")
	}
	if _, err := client.Search(context.Background(), "\n", nil); err == nil {
		t.Fatal("blank search succeeded")
	}
	for _, limit := range []int{0, MaxSearchLimit + 1} {
		if _, err := client.Search(context.Background(), "query", &limit); err == nil {
			t.Fatalf("limit %d succeeded", limit)
		}
	}
	if requests.Load() != 0 {
		t.Fatalf("requests = %d", requests.Load())
	}
}

func TestUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		http.Error(response, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client, err := New(Config{
		APIKey: "key", UserID: "user", AgentID: "agent", BaseURL: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Add(context.Background(), "remember this"); err == nil ||
		!strings.Contains(err.Error(), "status 503") {
		t.Fatalf("error = %v", err)
	} else if strings.Contains(err.Error(), "mem0:") {
		t.Fatalf("error leaked dependency details: %v", err)
	}
}

func clearMemoryEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		EnvAPIKey, EnvUserID, EnvAgentID, EnvProvider, EnvBaseURL,
	} {
		t.Setenv(name, "")
	}
}

func decodeBody(t *testing.T, request *http.Request) map[string]any {
	t.Helper()
	defer request.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body
}

func writeJSON(t *testing.T, response http.ResponseWriter, body any) {
	t.Helper()
	response.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(response).Encode(body); err != nil {
		t.Fatal(err)
	}
}
