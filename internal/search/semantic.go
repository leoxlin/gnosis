package search

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gnosis/internal/vault"
)

const (
	semanticChunkRunes = 6_000
	embeddingBatchSize = 64
	maxEmbeddingBody   = 1 << 20
	defaultVectorStore = "pgvector"
)

// SemanticConfig identifies the vector and embeddings services used by the
// derived semantic index.
type SemanticConfig struct {
	Backend          string
	DatabaseURL      string
	SQLitePath       string
	EmbeddingsURL    string
	EmbeddingsModel  string
	EmbeddingsAPIKey string
	Scope            string
	HTTPClient       *http.Client
}

// SemanticIndexResult summarizes one complete semantic index replacement.
type SemanticIndexResult struct {
	Documents   int    `json:"documents"`
	Chunks      int    `json:"chunks"`
	Scope       string `json:"scope"`
	Fingerprint string `json:"fingerprint"`
	Backend     string `json:"backend"`
	Path        string `json:"path,omitempty"`
}

type semanticChunk struct {
	document vault.Document
	index    int
	content  string
}

type embeddingClient struct {
	config SemanticConfig
}

type semanticStore interface {
	replace(context.Context, semanticIndex) error
	search(context.Context, semanticSearch) ([]semanticMatch, error)
}

type semanticIndex struct {
	scope       string
	model       string
	fingerprint string
	dimensions  int
	chunks      []storedSemanticChunk
}

type storedSemanticChunk struct {
	uri          string
	index        int
	revision     string
	model        string
	documentType string
	title        string
	description  string
	origin       []byte
	content      string
	embedding    []float32
}

type semanticSearch struct {
	scope       string
	model       string
	fingerprint string
	embed       func() ([]float32, error)
	top         int
}

type semanticMatch struct {
	uri          string
	documentType string
	title        string
	description  string
	origin       []byte
	revision     string
	score        float64
}

type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

// SemanticConfigFromEnv loads semantic service configuration without placing
// credentials in gnosis.toml.
func SemanticConfigFromEnv(root string) (SemanticConfig, error) {
	absolute, err := filepath.Abs(root)
	if err != nil {
		return SemanticConfig{}, fmt.Errorf("semantic config: resolve workspace: %w", err)
	}
	backend := strings.TrimSpace(os.Getenv("GNOSIS_VECTOR_BACKEND"))
	if backend == "" {
		backend = defaultVectorStore
	}
	scopeHash := sha256.Sum256([]byte(filepath.Clean(absolute)))
	config := SemanticConfig{
		Backend:          backend,
		DatabaseURL:      strings.TrimSpace(os.Getenv("GNOSIS_DATABASE_URL")),
		SQLitePath:       strings.TrimSpace(os.Getenv("GNOSIS_SQLITE_VECTOR_PATH")),
		EmbeddingsURL:    strings.TrimSpace(os.Getenv("GNOSIS_EMBEDDING_URL")),
		EmbeddingsModel:  strings.TrimSpace(os.Getenv("GNOSIS_EMBEDDING_MODEL")),
		EmbeddingsAPIKey: os.Getenv("GNOSIS_EMBEDDING_API_KEY"),
		Scope:            hex.EncodeToString(scopeHash[:]),
	}
	if config.Backend == "sqlite" && config.SQLitePath == "" {
		catalog, err := vault.Vaults(root)
		if err != nil {
			return SemanticConfig{}, fmt.Errorf("semantic config: resolve vault identity: %w", err)
		}
		if len(catalog.Vaults) == 0 || catalog.Vaults[0].Vault == "core" {
			return SemanticConfig{}, errors.New("semantic config: SQLite requires a named vault")
		}
		cache, err := os.UserCacheDir()
		if err != nil {
			return SemanticConfig{}, fmt.Errorf("semantic config: resolve user cache: %w", err)
		}
		config.SQLitePath = filepath.Join(cache, "gnosis", catalog.Vaults[0].Vault, "semantic.db")
	}
	if err := validateSemanticConfig(config); err != nil {
		return SemanticConfig{}, err
	}
	return config, nil
}

// SyncSemanticIndex atomically replaces the derived index for one workspace.
func SyncSemanticIndex(
	ctx context.Context,
	root string,
	config SemanticConfig,
) (result SemanticIndexResult, err error) {
	if err := validateSemanticConfig(config); err != nil {
		return SemanticIndexResult{}, err
	}
	documents, err := vault.LoadDocuments(root)
	if err != nil {
		return SemanticIndexResult{}, fmt.Errorf("semantic index: load documents: %w", err)
	}

	chunks := make([]semanticChunk, 0, len(documents))
	inputs := make([]string, 0, len(documents))
	for _, document := range documents {
		for _, chunk := range semanticChunks(document) {
			chunks = append(chunks, chunk)
			inputs = append(inputs, chunk.content)
		}
	}
	if len(chunks) == 0 {
		return SemanticIndexResult{}, errors.New("semantic index: no documents to index")
	}
	vectors, err := (&embeddingClient{config: config}).embed(ctx, inputs)
	if err != nil {
		return SemanticIndexResult{}, fmt.Errorf("semantic index: embed documents: %w", err)
	}

	stored := make([]storedSemanticChunk, 0, len(chunks))
	for i, chunk := range chunks {
		origin, err := json.Marshal(chunk.document.Origin)
		if err != nil {
			return SemanticIndexResult{}, fmt.Errorf("semantic index: encode origin for %q: %w", chunk.document.URI, err)
		}
		stored = append(stored, storedSemanticChunk{
			uri:          chunk.document.URI,
			index:        chunk.index,
			revision:     chunk.document.Revision,
			model:        config.EmbeddingsModel,
			documentType: chunk.document.Type,
			title:        chunk.document.Title,
			description:  chunk.document.Description,
			origin:       origin,
			content:      chunk.content,
			embedding:    vectors[i],
		})
	}

	fingerprint := documentFingerprint(documents)
	store, err := semanticStoreFor(config)
	if err != nil {
		return SemanticIndexResult{}, err
	}
	if err := store.replace(ctx, semanticIndex{
		scope:       config.Scope,
		model:       config.EmbeddingsModel,
		fingerprint: fingerprint,
		dimensions:  len(vectors[0]),
		chunks:      stored,
	}); err != nil {
		return SemanticIndexResult{}, fmt.Errorf("semantic index: %w", err)
	}
	return SemanticIndexResult{
		Documents:   len(documents),
		Chunks:      len(chunks),
		Scope:       config.Scope,
		Fingerprint: fingerprint,
		Backend:     semanticBackend(config),
		Path:        config.SQLitePath,
	}, nil
}

// QuerySemantic retrieves distinct pages from the current derived semantic
// index and preserves the stable query result contract.
func QuerySemantic(
	ctx context.Context,
	root string,
	question string,
	options QueryOptions,
	config SemanticConfig,
) (result QueryResult, err error) {
	if strings.TrimSpace(question) == "" {
		return QueryResult{}, errors.New("semantic query: question must not be empty")
	}
	if err := validateSemanticConfig(config); err != nil {
		return QueryResult{}, err
	}
	documents, err := vault.LoadDocuments(root)
	if err != nil {
		return QueryResult{}, fmt.Errorf("semantic query: load documents: %w", err)
	}
	documentsByURI := make(map[string]vault.Document, len(documents))
	for _, document := range documents {
		documentsByURI[document.URI] = document
	}

	liveFingerprint := documentFingerprint(documents)
	store, err := semanticStoreFor(config)
	if err != nil {
		return QueryResult{}, err
	}

	options = normalizedOptions(options)
	matches, err := store.search(ctx, semanticSearch{
		scope:       config.Scope,
		model:       config.EmbeddingsModel,
		fingerprint: liveFingerprint,
		embed: func() ([]float32, error) {
			vectors, err := (&embeddingClient{config: config}).embed(ctx, []string{question})
			if err != nil {
				return nil, fmt.Errorf("embed question: %w", err)
			}
			return vectors[0], nil
		},
		top: options.Top,
	})
	if err != nil {
		return QueryResult{}, fmt.Errorf("semantic query: %w", err)
	}

	answerType, _ := classifyQuestion(question)
	result = QueryResult{
		AnswerType: answerType,
		Candidates: []Candidate{},
		Path:       []string{},
		ShouldRead: []string{},
		IndexOnly:  false,
	}
	for _, match := range matches {
		candidate := Candidate{
			URI:         match.uri,
			Type:        match.documentType,
			Title:       match.title,
			Description: match.description,
			Revision:    match.revision,
			Score:       match.score,
		}
		if err := json.Unmarshal(match.origin, &candidate.Origin); err != nil {
			return QueryResult{}, fmt.Errorf("semantic query: decode origin for %q: %w", candidate.URI, err)
		}
		candidate.Trust = documentsByURI[candidate.URI].Trust
		candidate.Description = truncateRunes(candidate.Description, maxDescriptionRune)
		candidate.Score = roundScore(candidate.Score)
		result.Candidates = append(result.Candidates, candidate)
		if len(result.ShouldRead) < options.MaxRead {
			result.ShouldRead = append(result.ShouldRead, candidate.URI)
		}
	}
	return result, nil
}

func validateSemanticConfig(config SemanticConfig) error {
	switch semanticBackend(config) {
	case "pgvector":
		if strings.TrimSpace(config.DatabaseURL) == "" {
			return errors.New("semantic config: GNOSIS_DATABASE_URL must not be empty")
		}
		if strings.TrimSpace(config.SQLitePath) != "" {
			return errors.New("semantic config: GNOSIS_SQLITE_VECTOR_PATH requires GNOSIS_VECTOR_BACKEND=sqlite")
		}
	case "sqlite":
		if strings.TrimSpace(config.DatabaseURL) != "" {
			return errors.New("semantic config: GNOSIS_DATABASE_URL conflicts with GNOSIS_VECTOR_BACKEND=sqlite")
		}
		if !filepath.IsAbs(config.SQLitePath) {
			return errors.New("semantic config: GNOSIS_SQLITE_VECTOR_PATH must be absolute")
		}
	default:
		return fmt.Errorf("semantic config: GNOSIS_VECTOR_BACKEND must be %q or %q, got %q", "pgvector", "sqlite", config.Backend)
	}
	if strings.TrimSpace(config.EmbeddingsURL) == "" {
		return errors.New("semantic config: GNOSIS_EMBEDDING_URL must not be empty")
	}
	parsedURL, err := url.ParseRequestURI(config.EmbeddingsURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return fmt.Errorf("semantic config: GNOSIS_EMBEDDING_URL is invalid: %q", config.EmbeddingsURL)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("semantic config: GNOSIS_EMBEDDING_URL must use http or https: %q", config.EmbeddingsURL)
	}
	if strings.TrimSpace(config.EmbeddingsModel) == "" {
		return errors.New("semantic config: GNOSIS_EMBEDDING_MODEL must not be empty")
	}
	if strings.TrimSpace(config.Scope) == "" {
		return errors.New("semantic config: scope must not be empty")
	}
	return nil
}

func semanticBackend(config SemanticConfig) string {
	if strings.TrimSpace(config.Backend) == "" {
		return defaultVectorStore
	}
	return strings.TrimSpace(config.Backend)
}

func semanticStoreFor(config SemanticConfig) (semanticStore, error) {
	switch semanticBackend(config) {
	case "pgvector":
		return pgvectorStore{databaseURL: config.DatabaseURL}, nil
	case "sqlite":
		return sqliteVectorStore{path: config.SQLitePath}, nil
	default:
		return nil, validateSemanticConfig(config)
	}
}

func semanticChunks(document vault.Document) []semanticChunk {
	prefix := "Title: " + document.Title + "\nType: " + document.Type
	if document.Description != "" {
		prefix += "\nDescription: " + document.Description
	}
	if document.URI != "" {
		prefix += "\nURI: " + document.URI
	}
	prefix += "\n\n"

	bodies := splitSemanticBody(document.Body)
	chunks := make([]semanticChunk, 0, len(bodies))
	for i, body := range bodies {
		chunks = append(chunks, semanticChunk{
			document: document,
			index:    i,
			content:  prefix + body,
		})
	}
	return chunks
}

func splitSemanticBody(body string) []string {
	body = strings.TrimSpace(body)
	if body == "" {
		return []string{""}
	}

	chunks := []string{}
	var current strings.Builder
	flush := func() {
		if current.Len() == 0 {
			return
		}
		chunks = append(chunks, current.String())
		current.Reset()
	}
	for _, paragraph := range strings.Split(body, "\n\n") {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		runes := []rune(paragraph)
		for len(runes) > semanticChunkRunes {
			flush()
			chunks = append(chunks, string(runes[:semanticChunkRunes]))
			runes = runes[semanticChunkRunes:]
		}
		if len(runes) == 0 {
			continue
		}
		separatorRunes := 0
		if current.Len() > 0 {
			separatorRunes = 2
		}
		if len([]rune(current.String()))+separatorRunes+len(runes) > semanticChunkRunes {
			flush()
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
		}
		current.WriteString(string(runes))
	}
	flush()
	if len(chunks) == 0 {
		return []string{""}
	}
	return chunks
}

func documentFingerprint(documents []vault.Document) string {
	references := make([]string, 0, len(documents))
	for _, document := range documents {
		references = append(references, document.URI+"\x00"+document.Revision)
	}
	sort.Strings(references)
	hash := sha256.Sum256([]byte(strings.Join(references, "\n")))
	return hex.EncodeToString(hash[:])
}

func (c *embeddingClient) embed(ctx context.Context, inputs []string) ([][]float32, error) {
	if len(inputs) == 0 {
		return [][]float32{}, nil
	}
	vectors := make([][]float32, 0, len(inputs))
	dimensions := 0
	for start := 0; start < len(inputs); start += embeddingBatchSize {
		end := min(start+embeddingBatchSize, len(inputs))
		batch, err := c.embedBatch(ctx, inputs[start:end])
		if err != nil {
			return nil, err
		}
		for _, vector := range batch {
			if err := validateEmbedding(vector, dimensions); err != nil {
				return nil, err
			}
			if dimensions == 0 {
				dimensions = len(vector)
			}
			vectors = append(vectors, vector)
		}
	}
	return vectors, nil
}

func (c *embeddingClient) embedBatch(ctx context.Context, inputs []string) ([][]float32, error) {
	body, err := json.Marshal(embeddingRequest{Model: c.config.EmbeddingsModel, Input: inputs})
	if err != nil {
		return nil, fmt.Errorf("embedding request: encode: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.EmbeddingsURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("embedding request: create: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if c.config.EmbeddingsAPIKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.config.EmbeddingsAPIKey)
	}
	client := c.config.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("embedding request: send: %w", err)
	}
	data, readErr := io.ReadAll(io.LimitReader(response.Body, maxEmbeddingBody))
	closeErr := response.Body.Close()
	if readErr != nil {
		return nil, fmt.Errorf("embedding response: read: %w", readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("embedding response: close: %w", closeErr)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("embedding response: %s: %s", response.Status, strings.TrimSpace(string(data)))
	}

	var decoded embeddingResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, fmt.Errorf("embedding response: decode: %w", err)
	}
	if len(decoded.Data) != len(inputs) {
		return nil, fmt.Errorf("embedding response: returned %d vectors for %d inputs", len(decoded.Data), len(inputs))
	}
	vectors := make([][]float32, len(inputs))
	seen := make([]bool, len(inputs))
	for _, item := range decoded.Data {
		if item.Index < 0 || item.Index >= len(inputs) {
			return nil, fmt.Errorf("embedding response: index %d is out of range", item.Index)
		}
		if seen[item.Index] {
			return nil, fmt.Errorf("embedding response: duplicate index %d", item.Index)
		}
		seen[item.Index] = true
		vectors[item.Index] = item.Embedding
	}
	return vectors, nil
}

func validateEmbedding(vector []float32, dimensions int) error {
	if len(vector) == 0 {
		return errors.New("embedding response: vector must not be empty")
	}
	if dimensions > 0 && len(vector) != dimensions {
		return fmt.Errorf("embedding response: vector dimensions are %d, want %d", len(vector), dimensions)
	}
	var norm float64
	for _, value := range vector {
		converted := float64(value)
		if math.IsNaN(converted) || math.IsInf(converted, 0) {
			return errors.New("embedding response: vector values must be finite")
		}
		norm += converted * converted
	}
	if norm == 0 {
		return errors.New("embedding response: cosine vector must not be zero")
	}
	return nil
}
