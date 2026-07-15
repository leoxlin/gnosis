package vault

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
