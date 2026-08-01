package search

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

type pgvectorStore struct {
	databaseURL string
}

func (store pgvectorStore) replace(ctx context.Context, index semanticIndex) (err error) {
	conn, err := pgx.Connect(ctx, store.databaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer func() {
		err = errors.Join(err, conn.Close(context.Background()))
	}()
	if err := ensurePGVectorSchema(ctx, conn); err != nil {
		return err
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replacement: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			err = errors.Join(err, tx.Rollback(context.Background()))
		}
	}()
	if _, err := tx.Exec(ctx, `DELETE FROM gnosis_semantic_chunks WHERE scope = $1`, index.scope); err != nil {
		return fmt.Errorf("delete old chunks: %w", err)
	}
	for _, chunk := range index.chunks {
		vector, err := vectorLiteral(chunk.embedding)
		if err != nil {
			return fmt.Errorf("encode vector for %q: %w", chunk.uri, err)
		}
		if _, err := tx.Exec(
			ctx,
			`INSERT INTO gnosis_semantic_chunks
			 (scope, uri, chunk, revision, model, type, title, description, origin, content, embedding)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11::vector)`,
			index.scope,
			chunk.uri,
			chunk.index,
			chunk.revision,
			chunk.model,
			chunk.documentType,
			chunk.title,
			chunk.description,
			string(chunk.origin),
			chunk.content,
			vector,
		); err != nil {
			return fmt.Errorf("insert chunk for %q: %w", chunk.uri, err)
		}
	}
	if _, err := tx.Exec(
		ctx,
		`INSERT INTO gnosis_semantic_indexes (scope, model, fingerprint, dimensions)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (scope) DO UPDATE SET
		 model = EXCLUDED.model,
		 fingerprint = EXCLUDED.fingerprint,
		 dimensions = EXCLUDED.dimensions,
		 indexed_at = now()`,
		index.scope,
		index.model,
		index.fingerprint,
		index.dimensions,
	); err != nil {
		return fmt.Errorf("update metadata: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit replacement: %w", err)
	}
	committed = true
	return nil
}

func (store pgvectorStore) search(ctx context.Context, query semanticSearch) (matches []semanticMatch, err error) {
	conn, err := pgx.Connect(ctx, store.databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	defer func() {
		err = errors.Join(err, conn.Close(context.Background()))
	}()

	var model, fingerprint string
	var dimensions int
	err = conn.QueryRow(
		ctx,
		`SELECT model, fingerprint, dimensions FROM gnosis_semantic_indexes WHERE scope = $1`,
		query.scope,
	).Scan(&model, &fingerprint, &dimensions)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("index not found; run semantic index synchronization")
	}
	if err != nil {
		return nil, fmt.Errorf("read index metadata: %w", err)
	}
	if model != query.model {
		return nil, fmt.Errorf("index model is %q, configured model is %q", model, query.model)
	}
	if fingerprint != query.fingerprint {
		return nil, errors.New("index is stale; run semantic index synchronization")
	}
	vector, err := query.embed()
	if err != nil {
		return nil, err
	}
	if len(vector) != dimensions {
		return nil, fmt.Errorf(
			"embedding dimensions are %d, index dimensions are %d",
			len(vector),
			dimensions,
		)
	}
	literal, err := vectorLiteral(vector)
	if err != nil {
		return nil, fmt.Errorf("encode vector: %w", err)
	}

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
		query.scope,
		query.model,
		literal,
		query.top,
	)
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var match semanticMatch
		if err := rows.Scan(
			&match.uri,
			&match.documentType,
			&match.title,
			&match.description,
			&match.origin,
			&match.revision,
			&match.score,
		); err != nil {
			return nil, fmt.Errorf("scan candidate: %w", err)
		}
		matches = append(matches, match)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read candidates: %w", err)
	}
	return matches, nil
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

func ensurePGVectorSchema(ctx context.Context, conn *pgx.Conn) error {
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
			return fmt.Errorf("initialize schema: %w", err)
		}
	}
	return nil
}
