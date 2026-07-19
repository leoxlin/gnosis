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
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"gnosis/internal/vault"
)

const (
	semanticChunkRunes = 6_000
	embeddingBatchSize = 64
	maxEmbeddingBody   = 1 << 20
)

// SemanticConfig identifies the pgvector and embeddings services used by the
// derived semantic index.
type SemanticConfig struct {
	DatabaseURL      string
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
}

type semanticChunk struct {
	document vault.Document
	index    int
	content  string
}

type embeddingClient struct {
	config SemanticConfig
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
	scopeHash := sha256.Sum256([]byte(filepath.Clean(absolute)))
	config := SemanticConfig{
		DatabaseURL:      strings.TrimSpace(os.Getenv("GNOSIS_DATABASE_URL")),
		EmbeddingsURL:    strings.TrimSpace(os.Getenv("GNOSIS_EMBEDDING_URL")),
		EmbeddingsModel:  strings.TrimSpace(os.Getenv("GNOSIS_EMBEDDING_MODEL")),
		EmbeddingsAPIKey: os.Getenv("GNOSIS_EMBEDDING_API_KEY"),
		Scope:            hex.EncodeToString(scopeHash[:]),
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

	conn, err := pgx.Connect(ctx, config.DatabaseURL)
	if err != nil {
		return SemanticIndexResult{}, fmt.Errorf("semantic index: connect database: %w", err)
	}
	defer func() {
		err = errors.Join(err, conn.Close(context.Background()))
	}()
	if err := ensureSemanticSchema(ctx, conn); err != nil {
		return SemanticIndexResult{}, err
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return SemanticIndexResult{}, fmt.Errorf("semantic index: begin replacement: %w", err)
	}
	isCommitted := false
	defer func() {
		if isCommitted {
			return
		}
		err = errors.Join(err, tx.Rollback(context.Background()))
	}()
	if _, err := tx.Exec(ctx, `DELETE FROM gnosis_semantic_chunks WHERE scope = $1`, config.Scope); err != nil {
		return SemanticIndexResult{}, fmt.Errorf("semantic index: delete old chunks: %w", err)
	}

	for i, chunk := range chunks {
		origin, err := json.Marshal(chunk.document.Origin)
		if err != nil {
			return SemanticIndexResult{}, fmt.Errorf("semantic index: encode origin for %q: %w", chunk.document.URI, err)
		}
		vector, err := vectorLiteral(vectors[i])
		if err != nil {
			return SemanticIndexResult{}, fmt.Errorf("semantic index: encode vector for %q: %w", chunk.document.URI, err)
		}
		_, err = tx.Exec(
			ctx,
			`INSERT INTO gnosis_semantic_chunks
			 (scope, uri, chunk, revision, model, type, title, description, origin, content, embedding)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11::vector)`,
			config.Scope,
			chunk.document.URI,
			chunk.index,
			chunk.document.Revision,
			config.EmbeddingsModel,
			chunk.document.Type,
			chunk.document.Title,
			chunk.document.Description,
			string(origin),
			chunk.content,
			vector,
		)
		if err != nil {
			return SemanticIndexResult{}, fmt.Errorf("semantic index: insert chunk for %q: %w", chunk.document.URI, err)
		}
	}

	fingerprint := documentFingerprint(documents)
	_, err = tx.Exec(
		ctx,
		`INSERT INTO gnosis_semantic_indexes (scope, model, fingerprint, dimensions)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (scope) DO UPDATE SET
		 model = EXCLUDED.model,
		 fingerprint = EXCLUDED.fingerprint,
		 dimensions = EXCLUDED.dimensions,
		 indexed_at = now()`,
		config.Scope,
		config.EmbeddingsModel,
		fingerprint,
		len(vectors[0]),
	)
	if err != nil {
		return SemanticIndexResult{}, fmt.Errorf("semantic index: update metadata: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return SemanticIndexResult{}, fmt.Errorf("semantic index: commit replacement: %w", err)
	}
	isCommitted = true
	return SemanticIndexResult{
		Documents:   len(documents),
		Chunks:      len(chunks),
		Scope:       config.Scope,
		Fingerprint: fingerprint,
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

	conn, err := pgx.Connect(ctx, config.DatabaseURL)
	if err != nil {
		return QueryResult{}, fmt.Errorf("semantic query: connect database: %w", err)
	}
	defer func() {
		err = errors.Join(err, conn.Close(context.Background()))
	}()

	var model, fingerprint string
	var dimensions int
	err = conn.QueryRow(
		ctx,
		`SELECT model, fingerprint, dimensions FROM gnosis_semantic_indexes WHERE scope = $1`,
		config.Scope,
	).Scan(&model, &fingerprint, &dimensions)
	if errors.Is(err, pgx.ErrNoRows) {
		return QueryResult{}, errors.New("semantic query: index not found; run semantic index synchronization")
	}
	if err != nil {
		return QueryResult{}, fmt.Errorf("semantic query: read index metadata: %w", err)
	}
	if model != config.EmbeddingsModel {
		return QueryResult{}, fmt.Errorf("semantic query: index model is %q, configured model is %q", model, config.EmbeddingsModel)
	}
	liveFingerprint := documentFingerprint(documents)
	if fingerprint != liveFingerprint {
		return QueryResult{}, errors.New("semantic query: index is stale; run semantic index synchronization")
	}

	vectors, err := (&embeddingClient{config: config}).embed(ctx, []string{question})
	if err != nil {
		return QueryResult{}, fmt.Errorf("semantic query: embed question: %w", err)
	}
	if len(vectors[0]) != dimensions {
		return QueryResult{}, fmt.Errorf(
			"semantic query: embedding dimensions are %d, index dimensions are %d",
			len(vectors[0]),
			dimensions,
		)
	}
	vector, err := vectorLiteral(vectors[0])
	if err != nil {
		return QueryResult{}, fmt.Errorf("semantic query: encode vector: %w", err)
	}

	options = normalizedOptions(options)
	rows, err := conn.Query(
		ctx,
		`SELECT uri, type, title, description, origin, revision, 1 - distance AS score
		 FROM (
		   SELECT DISTINCT ON (uri) uri, type, title, description, origin, revision,
		          embedding <=> $3::vector AS distance
		   FROM gnosis_semantic_chunks
		   WHERE scope = $1 AND model = $2
		   ORDER BY uri, distance, chunk
		 ) nearest
		 ORDER BY distance, uri
		 LIMIT $4`,
		config.Scope,
		config.EmbeddingsModel,
		vector,
		options.Top,
	)
	if err != nil {
		return QueryResult{}, fmt.Errorf("semantic query: search chunks: %w", err)
	}
	defer rows.Close()

	answerType, _ := classifyQuestion(question)
	result = QueryResult{
		AnswerType: answerType,
		Candidates: []Candidate{},
		Path:       []string{},
		ShouldRead: []string{},
		IndexOnly:  false,
	}
	for rows.Next() {
		var candidate Candidate
		var origin []byte
		if err := rows.Scan(
			&candidate.URI,
			&candidate.Type,
			&candidate.Title,
			&candidate.Description,
			&origin,
			&candidate.Revision,
			&candidate.Score,
		); err != nil {
			return QueryResult{}, fmt.Errorf("semantic query: scan candidate: %w", err)
		}
		if err := json.Unmarshal(origin, &candidate.Origin); err != nil {
			return QueryResult{}, fmt.Errorf("semantic query: decode origin for %q: %w", candidate.URI, err)
		}
		candidate.Description = truncateRunes(candidate.Description, maxDescriptionRune)
		candidate.Score = roundScore(candidate.Score)
		result.Candidates = append(result.Candidates, candidate)
		if len(result.ShouldRead) < options.MaxRead {
			result.ShouldRead = append(result.ShouldRead, candidate.URI)
		}
	}
	if err := rows.Err(); err != nil {
		return QueryResult{}, fmt.Errorf("semantic query: read candidates: %w", err)
	}
	return result, nil
}

func validateSemanticConfig(config SemanticConfig) error {
	if strings.TrimSpace(config.DatabaseURL) == "" {
		return errors.New("semantic config: GNOSIS_DATABASE_URL must not be empty")
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

func vectorLiteral(vector []float32) (string, error) {
	if err := validateEmbedding(vector, 0); err != nil {
		return "", err
	}
	values := make([]string, 0, len(vector))
	for _, value := range vector {
		values = append(values, strconv.FormatFloat(float64(value), 'g', -1, 32))
	}
	return "[" + strings.Join(values, ",") + "]", nil
}

func ensureSemanticSchema(ctx context.Context, conn *pgx.Conn) error {
	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS vector`,
		`CREATE TABLE IF NOT EXISTS gnosis_semantic_indexes (
		  scope text PRIMARY KEY,
		  model text NOT NULL,
		  fingerprint text NOT NULL,
		  dimensions integer NOT NULL CHECK (dimensions > 0),
		  indexed_at timestamptz NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS gnosis_semantic_chunks (
		  scope text NOT NULL,
		  uri text NOT NULL,
		  chunk integer NOT NULL CHECK (chunk >= 0),
		  revision text NOT NULL,
		  model text NOT NULL,
		  type text NOT NULL,
		  title text NOT NULL,
		  description text NOT NULL,
		  origin jsonb NOT NULL,
		  content text NOT NULL,
		  embedding vector NOT NULL,
		  PRIMARY KEY (scope, uri, chunk)
		)`,
	}
	for _, statement := range statements {
		if _, err := conn.Exec(ctx, statement); err != nil {
			return fmt.Errorf("semantic index: initialize schema: %w", err)
		}
	}
	return nil
}
