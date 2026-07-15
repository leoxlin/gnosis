package vault

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestSemanticConfigFromEnv(t *testing.T) {
	t.Setenv("GNOSIS_DATABASE_URL", "")
	t.Setenv("GNOSIS_EMBEDDING_URL", "")
	t.Setenv("GNOSIS_EMBEDDING_MODEL", "")

	root := t.TempDir()
	if _, err := SemanticConfigFromEnv(root); err == nil || !strings.Contains(err.Error(), "GNOSIS_DATABASE_URL") {
		t.Fatalf("missing database url error = %v", err)
	}

	t.Setenv("GNOSIS_DATABASE_URL", "postgres://secret@localhost/gnosis")
	if _, err := SemanticConfigFromEnv(root); err == nil || !strings.Contains(err.Error(), "GNOSIS_EMBEDDING_URL") {
		t.Fatalf("missing embeddings url error = %v", err)
	}

	t.Setenv("GNOSIS_EMBEDDING_URL", "http://localhost:11434/v1/embeddings")
	if _, err := SemanticConfigFromEnv(root); err == nil || !strings.Contains(err.Error(), "GNOSIS_EMBEDDING_MODEL") {
		t.Fatalf("missing embeddings model error = %v", err)
	}

	t.Setenv("GNOSIS_EMBEDDING_MODEL", "test-model")
	t.Setenv("GNOSIS_EMBEDDING_API_KEY", "embedding-secret")
	first, err := SemanticConfigFromEnv(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := SemanticConfigFromEnv(root)
	if err != nil {
		t.Fatal(err)
	}
	if first.Scope == "" || first.Scope != second.Scope {
		t.Fatalf("scope = %q, second = %q", first.Scope, second.Scope)
	}
	if strings.Contains(first.Scope, "secret") || strings.Contains(first.Scope, root) {
		t.Fatalf("scope exposes configuration: %q", first.Scope)
	}
	if first.EmbeddingsAPIKey != "embedding-secret" {
		t.Fatalf("api key = %q", first.EmbeddingsAPIKey)
	}
}

func TestSemanticChunks(t *testing.T) {
	document := Document{
		URI:         "gnosis://test/concepts/vector.md",
		Title:       "Vector search",
		Type:        "Decision",
		Description: "Choose semantic retrieval.",
		Body:        strings.Repeat("a", 4_000) + "\n\n" + strings.Repeat("b", 3_000),
	}

	chunks := semanticChunks(document)
	if len(chunks) != 2 {
		t.Fatalf("chunks = %d, want 2", len(chunks))
	}
	for i, chunk := range chunks {
		if chunk.index != i {
			t.Fatalf("chunk %d index = %d", i, chunk.index)
		}
		for _, value := range []string{document.Title, document.Type, document.Description} {
			if !strings.Contains(chunk.content, value) {
				t.Fatalf("chunk %d missing metadata %q", i, value)
			}
		}
	}

	oversized := semanticChunks(Document{Title: "Large", Body: strings.Repeat("c", semanticChunkRunes+1)})
	if len(oversized) != 2 {
		t.Fatalf("oversized chunks = %d, want 2", len(oversized))
	}
	if !strings.Contains(oversized[0].content, strings.Repeat("c", semanticChunkRunes)) {
		t.Fatal("first oversized chunk does not preserve the bounded body")
	}

	metadataOnly := semanticChunks(Document{Title: "Metadata only", Type: "Reference"})
	if len(metadataOnly) != 1 || !strings.Contains(metadataOnly[0].content, "Metadata only") {
		t.Fatalf("metadata-only chunks = %+v", metadataOnly)
	}
}

func TestEmbeddingClientEmbed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s", request.Method)
		}
		if authorization := request.Header.Get("Authorization"); authorization != "Bearer test-key" {
			t.Errorf("authorization = %q", authorization)
		}
		var body struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body.Model != "test-model" || strings.Join(body.Input, ",") != "first,second" {
			t.Errorf("request = %+v", body)
		}
		response.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(response, `{"data":[{"index":1,"embedding":[0,1]},{"index":0,"embedding":[1,0]}]}`); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer server.Close()

	client := embeddingClient{config: SemanticConfig{
		EmbeddingsURL:    server.URL,
		EmbeddingsModel:  "test-model",
		EmbeddingsAPIKey: "test-key",
		HTTPClient:       server.Client(),
	}}
	vectors, err := client.embed(context.Background(), []string{"first", "second"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 2 || vectors[0][0] != 1 || vectors[1][1] != 1 {
		t.Fatalf("vectors = %v", vectors)
	}
}

func TestEmbeddingClientBatches(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		var body embeddingRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		data := make([]map[string]any, 0, len(body.Input))
		for i := range body.Input {
			data = append(data, map[string]any{"index": i, "embedding": []float32{float32(i + 1), 1}})
		}
		if err := json.NewEncoder(response).Encode(map[string]any{"data": data}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	inputs := make([]string, embeddingBatchSize+1)
	for i := range inputs {
		inputs[i] = "input"
	}
	client := embeddingClient{config: SemanticConfig{
		EmbeddingsURL:   server.URL,
		EmbeddingsModel: "test-model",
		HTTPClient:      server.Client(),
	}}
	vectors, err := client.embed(context.Background(), inputs)
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != len(inputs) || requests.Load() != 2 {
		t.Fatalf("vectors = %d, requests = %d", len(vectors), requests.Load())
	}
}

func TestEmbeddingClientErrors(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		response string
		want     string
	}{
		{name: "non-success status", status: http.StatusBadRequest, response: `bad request`, want: "bad request"},
		{name: "missing vector", status: http.StatusOK, response: `{"data":[]}`, want: "returned 0 vectors"},
		{name: "mismatched dimensions", status: http.StatusOK, response: `{"data":[{"index":0,"embedding":[1]},{"index":1,"embedding":[1,2]}]}`, want: "dimensions"},
		{name: "non-finite vector", status: http.StatusOK, response: `{"data":[{"index":0,"embedding":[1e999]},{"index":1,"embedding":[1]}]}`, want: "embedding response"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(test.status)
				if _, err := io.WriteString(response, test.response); err != nil {
					t.Errorf("write response: %v", err)
				}
			}))
			defer server.Close()

			client := embeddingClient{config: SemanticConfig{
				EmbeddingsURL:   server.URL,
				EmbeddingsModel: "test-model",
				HTTPClient:      server.Client(),
			}}
			_, err := client.embed(context.Background(), []string{"first", "second"})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestDocumentFingerprint(t *testing.T) {
	documents := []Document{
		{URI: "gnosis://test/b.md", Revision: "two"},
		{URI: "gnosis://test/a.md", Revision: "one"},
	}
	reversed := []Document{documents[1], documents[0]}
	if documentFingerprint(documents) != documentFingerprint(reversed) {
		t.Fatal("fingerprint depends on document order")
	}
	reversed[0].Revision = "changed"
	if documentFingerprint(documents) == documentFingerprint(reversed) {
		t.Fatal("fingerprint ignores revisions")
	}
}

func TestVectorLiteral(t *testing.T) {
	literal, err := vectorLiteral([]float32{1, -2.5, 0})
	if err != nil {
		t.Fatal(err)
	}
	if literal != "[1,-2.5,0]" {
		t.Fatalf("literal = %q", literal)
	}
}

func TestSemanticIndexIntegration(t *testing.T) {
	databaseURL := os.Getenv("GNOSIS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("GNOSIS_TEST_DATABASE_URL is not set")
	}

	embeddings := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var body embeddingRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		data := make([]map[string]any, 0, len(body.Input))
		for i, input := range body.Input {
			vector := []float32{0.5, 0.5}
			switch {
			case strings.Contains(strings.ToLower(input), "alpha orchard"):
				vector = []float32{1, 0}
			case strings.Contains(strings.ToLower(input), "beta harbor"):
				vector = []float32{0, 1}
			}
			data = append(data, map[string]any{"index": i, "embedding": vector})
		}
		if err := json.NewEncoder(response).Encode(map[string]any{"data": data}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer embeddings.Close()

	root := t.TempDir()
	writeConfig(t, root, `[vault]
vault_name = "semantic-test"
vault_root = "docs"
vault_index = false
vault_log = false
`)
	alphaPath := filepath.Join(root, "docs", "alpha.md")
	write(t, root, "docs/alpha.md", `---
type: Reference
title: Alpha orchard
description: Apples and fruit trees.
---

Alpha orchard harvest knowledge.
`)
	write(t, root, "docs/beta.md", `---
type: Reference
title: Beta harbor
description: Ships and ocean cargo.
---

Beta harbor shipping knowledge.
`)
	config := SemanticConfig{
		DatabaseURL:     databaseURL,
		EmbeddingsURL:   embeddings.URL,
		EmbeddingsModel: "test-model",
		Scope:           documentFingerprint([]Document{{URI: root, Revision: t.Name()}}),
		HTTPClient:      embeddings.Client(),
	}
	defer deleteSemanticScope(t, config)

	indexed, err := SyncSemanticIndex(context.Background(), root, config)
	if err != nil {
		t.Fatal(err)
	}
	if indexed.Documents < 2 || indexed.Chunks < indexed.Documents {
		t.Fatalf("index result = %+v", indexed)
	}

	result, err := QuerySemanticKnowledge(
		context.Background(),
		root,
		"alpha orchard",
		QueryOptions{Top: 2, MaxRead: 2},
		config,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Candidates) == 0 || result.Candidates[0].URI != "gnosis://semantic-test/alpha.md" {
		t.Fatalf("candidates = %+v", result.Candidates)
	}
	firstRevision := result.Candidates[0].Revision
	if firstRevision == "" || result.Candidates[0].Origin.Vault != "semantic-test" {
		t.Fatalf("candidate provenance = %+v", result.Candidates[0])
	}

	if err := os.WriteFile(alphaPath, []byte(`---
type: Reference
title: Alpha orchard
description: Apples, pears, and fruit trees.
---

Alpha orchard harvest knowledge changed.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := QuerySemanticKnowledge(
		context.Background(),
		root,
		"alpha orchard",
		QueryOptions{Top: 1},
		config,
	); err == nil || !strings.Contains(err.Error(), "index is stale") {
		t.Fatalf("stale query error = %v", err)
	}

	if _, err := SyncSemanticIndex(context.Background(), root, config); err != nil {
		t.Fatal(err)
	}
	result, err = QuerySemanticKnowledge(
		context.Background(),
		root,
		"alpha orchard",
		QueryOptions{Top: 1, MaxRead: 1},
		config,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].Revision == firstRevision {
		t.Fatalf("reindexed candidates = %+v", result.Candidates)
	}

	failing := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		http.Error(response, "embedding failure", http.StatusInternalServerError)
	}))
	defer failing.Close()
	failedConfig := config
	failedConfig.EmbeddingsURL = failing.URL
	failedConfig.HTTPClient = failing.Client()
	if _, err := SyncSemanticIndex(context.Background(), root, failedConfig); err == nil {
		t.Fatal("failed synchronization returned nil error")
	}
	if _, err := QuerySemanticKnowledge(
		context.Background(),
		root,
		"alpha orchard",
		QueryOptions{Top: 1},
		config,
	); err != nil {
		t.Fatalf("query after failed synchronization: %v", err)
	}
}

func deleteSemanticScope(t *testing.T, config SemanticConfig) {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), config.DatabaseURL)
	if err != nil {
		t.Errorf("connect cleanup: %v", err)
		return
	}
	if _, err := conn.Exec(context.Background(), `DELETE FROM gnosis_semantic_chunks WHERE scope = $1`, config.Scope); err != nil {
		t.Errorf("delete chunks: %v", err)
	}
	if _, err := conn.Exec(context.Background(), `DELETE FROM gnosis_semantic_indexes WHERE scope = $1`, config.Scope); err != nil {
		t.Errorf("delete index: %v", err)
	}
	if err := conn.Close(context.Background()); err != nil {
		t.Errorf("close cleanup: %v", err)
	}
}
