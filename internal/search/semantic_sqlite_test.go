package search

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSQLiteVectorStoreLifecycle(t *testing.T) {
	store := sqliteVectorStore{path: t.TempDir() + "/semantic.db"}
	index := testSemanticIndex("scope-a")
	if err := store.replace(context.Background(), index); err != nil {
		t.Fatal(err)
	}

	matches, err := store.search(context.Background(), semanticSearch{
		scope:       index.scope,
		model:       index.model,
		fingerprint: index.fingerprint,
		embed:       testEmbedding(1, 0),
		top:         2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 2 || matches[0].uri != "gnosis://test/alpha.md" || matches[1].uri != "gnosis://test/beta.md" {
		t.Fatalf("matches = %+v", matches)
	}
	if string(matches[0].origin) != `{"vault":"test","kind":"local","precedence":0}` {
		t.Fatalf("origin = %s", matches[0].origin)
	}

	for name, test := range map[string]struct {
		query semanticSearch
		want  string
	}{
		"model": {query: semanticSearch{
			scope: index.scope, model: "other", fingerprint: index.fingerprint,
			embed: testEmbedding(1, 0), top: 1,
		}, want: "model"},
		"dimensions": {query: semanticSearch{
			scope: index.scope, model: index.model, fingerprint: index.fingerprint,
			embed: testEmbedding(1, 0, 0), top: 1,
		}, want: "dimensions"},
		"fingerprint": {query: semanticSearch{
			scope: index.scope, model: index.model, fingerprint: "stale",
			embed: testEmbedding(1, 0), top: 1,
		}, want: "stale"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := store.search(context.Background(), test.query); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}

	other := testSemanticIndex("scope-b")
	if err := store.replace(context.Background(), other); err != nil {
		t.Fatal(err)
	}
	if _, err := store.search(context.Background(), semanticSearch{
		scope: index.scope, model: index.model, fingerprint: index.fingerprint,
		embed: testEmbedding(1, 0), top: 1,
	}); err != nil {
		t.Fatalf("scope-a after scope-b replacement: %v", err)
	}

	broken := testSemanticIndex(index.scope)
	broken.fingerprint = "replacement"
	broken.chunks[1].embedding = []float32{0, 1, 0}
	if err := store.replace(context.Background(), broken); err == nil {
		t.Fatal("dimension mismatch replacement returned nil error")
	}
	if _, err := store.search(context.Background(), semanticSearch{
		scope: index.scope, model: index.model, fingerprint: index.fingerprint,
		embed: testEmbedding(1, 0), top: 1,
	}); err != nil {
		t.Fatalf("failed replacement did not roll back: %v", err)
	}
}

func TestSQLiteVectorStoreBoundsLockContention(t *testing.T) {
	store := sqliteVectorStore{path: t.TempDir() + "/semantic.db"}
	index := testSemanticIndex("scope")
	if err := store.replace(context.Background(), index); err != nil {
		t.Fatal(err)
	}

	db, err := openSQLiteVector(context.Background(), store.path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE gnosis_semantic_indexes SET fingerprint = fingerprint WHERE scope = ?`, index.scope); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	err = store.replace(context.Background(), index)
	if err == nil {
		t.Fatal("locked replacement returned nil error")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("lock error took %s: %v", elapsed, err)
	}
}

func TestSQLiteVectorStoreValidatesMetadataBeforeEmbedding(t *testing.T) {
	store := sqliteVectorStore{path: t.TempDir() + "/semantic.db"}
	index := testSemanticIndex("scope")
	if err := store.replace(context.Background(), index); err != nil {
		t.Fatal(err)
	}
	called := false
	_, err := store.search(context.Background(), semanticSearch{
		scope: index.scope, model: index.model, fingerprint: "stale", top: 1,
		embed: func() ([]float32, error) {
			called = true
			return []float32{1, 0}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("error = %v", err)
	}
	if called {
		t.Fatal("stale index invoked embedding")
	}
}

func TestSQLiteVectorStoreRejectsIncompatibleSchema(t *testing.T) {
	store := sqliteVectorStore{path: t.TempDir() + "/semantic.db"}
	index := testSemanticIndex("scope")
	if err := store.replace(context.Background(), index); err != nil {
		t.Fatal(err)
	}
	db, err := openSQLiteVector(context.Background(), store.path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 2`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := store.search(context.Background(), semanticSearch{
		scope: index.scope, model: index.model, fingerprint: index.fingerprint,
		embed: testEmbedding(1, 0), top: 1,
	}); err == nil || !strings.Contains(err.Error(), "schema version") {
		t.Fatalf("error = %v", err)
	}
}

func testEmbedding(values ...float32) func() ([]float32, error) {
	return func() ([]float32, error) {
		return values, nil
	}
}

func testSemanticIndex(scope string) semanticIndex {
	return semanticIndex{
		scope:       scope,
		model:       "test-model",
		fingerprint: "fingerprint",
		dimensions:  2,
		chunks: []storedSemanticChunk{
			{
				uri: "gnosis://test/alpha.md", revision: "alpha-revision",
				model: "test-model", documentType: "Reference", title: "Alpha",
				origin:  []byte(`{"vault":"test","kind":"local","precedence":0}`),
				content: "alpha", embedding: []float32{1, 0},
			},
			{
				uri: "gnosis://test/beta.md", revision: "beta-revision",
				model: "test-model", documentType: "Reference", title: "Beta",
				origin:  []byte(`{"vault":"test","kind":"local","precedence":0}`),
				content: "beta", embedding: []float32{0, 1},
			},
		},
	}
}
