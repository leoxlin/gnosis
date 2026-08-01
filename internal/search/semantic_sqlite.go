package search

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/vec1"
)

const (
	semanticSQLiteSchemaVersion = 1
	semanticSQLiteBusyMillis    = 250
)

type sqliteVectorStore struct {
	path string
}

func (store sqliteVectorStore) replace(ctx context.Context, index semanticIndex) (err error) {
	if err := os.MkdirAll(filepath.Dir(store.path), 0o755); err != nil {
		return fmt.Errorf("create SQLite directory: %w", err)
	}
	db, err := openSQLiteVector(ctx, store.path)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, db.Close())
	}()
	if err := ensureSQLiteVectorSchema(ctx, db); err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin SQLite replacement: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			err = errors.Join(err, tx.Rollback())
		}
	}()

	table := sqliteVectorTable(index.scope)
	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS "`+table+`"`); err != nil {
		return fmt.Errorf("drop old SQLite vectors: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE VIRTUAL TABLE "`+table+`" USING vec1(embedding)`); err != nil {
		return fmt.Errorf("create SQLite vector table: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM gnosis_semantic_chunks WHERE scope = ?`, index.scope); err != nil {
		return fmt.Errorf("delete old SQLite chunks: %w", err)
	}

	for _, chunk := range index.chunks {
		result, err := tx.ExecContext(
			ctx,
			`INSERT INTO gnosis_semantic_chunks
			 (scope, uri, chunk, revision, model, type, title, description, origin, content)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			index.scope,
			chunk.uri,
			chunk.index,
			chunk.revision,
			chunk.model,
			chunk.documentType,
			chunk.title,
			chunk.description,
			chunk.origin,
			chunk.content,
		)
		if err != nil {
			return fmt.Errorf("insert SQLite chunk for %q: %w", chunk.uri, err)
		}
		rowID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("read SQLite chunk row for %q: %w", chunk.uri, err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO "`+table+`" (rowid, embedding) VALUES (?, ?)`,
			rowID,
			sqliteVector(chunk.embedding),
		); err != nil {
			return fmt.Errorf("insert SQLite vector for %q: %w", chunk.uri, err)
		}
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO "`+table+`" (cmd, arg) VALUES ('rebuild', '{"index":"flat","distance":"cos"}')`,
	); err != nil {
		return fmt.Errorf("build SQLite vector index: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO gnosis_semantic_indexes
		 (scope, model, fingerprint, dimensions, vector_table, indexed_at)
		 VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT (scope) DO UPDATE SET
		 model = excluded.model,
		 fingerprint = excluded.fingerprint,
		 dimensions = excluded.dimensions,
		 vector_table = excluded.vector_table,
		 indexed_at = CURRENT_TIMESTAMP`,
		index.scope,
		index.model,
		index.fingerprint,
		index.dimensions,
		table,
	); err != nil {
		return fmt.Errorf("update SQLite metadata: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit SQLite replacement: %w", err)
	}
	committed = true
	return nil
}

func (store sqliteVectorStore) search(ctx context.Context, query semanticSearch) (matches []semanticMatch, err error) {
	if _, err := os.Stat(store.path); errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("index not found; run semantic index synchronization")
	} else if err != nil {
		return nil, fmt.Errorf("stat SQLite index: %w", err)
	}
	db, err := openSQLiteVector(ctx, store.path)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, db.Close())
	}()
	if err := ensureSQLiteVectorSchema(ctx, db); err != nil {
		return nil, err
	}

	var model, fingerprint, table string
	var dimensions int
	err = db.QueryRowContext(
		ctx,
		`SELECT model, fingerprint, dimensions, vector_table
		 FROM gnosis_semantic_indexes WHERE scope = ?`,
		query.scope,
	).Scan(&model, &fingerprint, &dimensions, &table)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("index not found; run semantic index synchronization")
	}
	if err != nil {
		return nil, fmt.Errorf("read SQLite metadata: %w", err)
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
	if table != sqliteVectorTable(query.scope) {
		return nil, errors.New("SQLite vector metadata is incompatible; delete the index and rebuild")
	}

	var chunks int
	if err := db.QueryRowContext(
		ctx,
		`SELECT count(*) FROM gnosis_semantic_chunks WHERE scope = ?`,
		query.scope,
	).Scan(&chunks); err != nil {
		return nil, fmt.Errorf("count SQLite chunks: %w", err)
	}
	rows, err := db.QueryContext(
		ctx,
		`WITH ranked AS (
		   SELECT chunks.uri, chunks.type, chunks.title, chunks.description,
		          chunks.origin, chunks.revision, vectors.distance,
		          row_number() OVER (
		            PARTITION BY chunks.uri
		            ORDER BY vectors.distance, chunks.chunk
		          ) AS rank
		   FROM "`+table+`"(?, ?) AS vectors
		   JOIN gnosis_semantic_chunks AS chunks ON chunks.id = vectors.rowid
		   WHERE chunks.scope = ? AND chunks.model = ?
		 )
		 SELECT uri, type, title, description, origin, revision, 2 - distance AS score
		 FROM ranked
		 WHERE rank = 1
		 ORDER BY distance, uri
		 LIMIT ?`,
		sqliteVector(vector),
		fmt.Sprintf(`{"k":%d}`, chunks),
		query.scope,
		query.model,
		query.top,
	)
	if err != nil {
		return nil, fmt.Errorf("search SQLite chunks: %w", err)
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
			return nil, fmt.Errorf("scan SQLite candidate: %w", err)
		}
		matches = append(matches, match)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read SQLite candidates: %w", err)
	}
	return matches, nil
}

func openSQLiteVector(ctx context.Context, path string) (*sql.DB, error) {
	parameters := url.Values{}
	parameters.Add("_pragma", fmt.Sprintf("busy_timeout(%d)", semanticSQLiteBusyMillis))
	parameters.Add("_pragma", "journal_mode(WAL)")
	parameters.Set("_txlock", "immediate")
	dsn := (&url.URL{Scheme: "file", Path: filepath.Clean(path), RawQuery: parameters.Encode()}).String()
	db, err := driver.Open(dsn, vec1.Register)
	if err != nil {
		return nil, fmt.Errorf("open SQLite index: %w", err)
	}
	db.SetMaxOpenConns(4)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open SQLite index: %w", err)
	}
	return db, nil
}

func ensureSQLiteVectorSchema(ctx context.Context, db *sql.DB) error {
	var version int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("read SQLite schema version: %w", err)
	}
	if version != 0 && version != semanticSQLiteSchemaVersion {
		return fmt.Errorf(
			"SQLite schema version is %d, want %d; delete the index and rebuild",
			version,
			semanticSQLiteSchemaVersion,
		)
	}
	if version == semanticSQLiteSchemaVersion {
		return nil
	}
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE gnosis_semantic_indexes (
		  scope TEXT PRIMARY KEY,
		  model TEXT NOT NULL,
		  fingerprint TEXT NOT NULL,
		  dimensions INTEGER NOT NULL CHECK (dimensions > 0),
		  vector_table TEXT NOT NULL,
		  indexed_at TEXT NOT NULL
		);
		CREATE TABLE gnosis_semantic_chunks (
		  id INTEGER PRIMARY KEY,
		  scope TEXT NOT NULL,
		  uri TEXT NOT NULL,
		  chunk INTEGER NOT NULL CHECK (chunk >= 0),
		  revision TEXT NOT NULL,
		  model TEXT NOT NULL,
		  type TEXT NOT NULL,
		  title TEXT NOT NULL,
		  description TEXT NOT NULL,
		  origin BLOB NOT NULL,
		  content TEXT NOT NULL,
		  UNIQUE (scope, uri, chunk)
		);
		CREATE INDEX gnosis_semantic_chunks_scope ON gnosis_semantic_chunks(scope);
		PRAGMA user_version = 1;
	`); err != nil {
		return fmt.Errorf("initialize SQLite schema: %w", err)
	}
	return nil
}

func sqliteVectorTable(scope string) string {
	hash := sha256.Sum256([]byte(scope))
	return fmt.Sprintf("gnosis_semantic_vectors_%x", hash[:8])
}

func sqliteVector(vector []float32) []byte {
	encoded := make([]byte, len(vector)*4)
	for i, value := range vector {
		binary.NativeEndian.PutUint32(encoded[i*4:], math.Float32bits(value))
	}
	return encoded
}
