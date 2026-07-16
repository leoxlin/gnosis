---
type: Directive
title: Add pgvector semantic knowledge backend
description: Index effective gnosis document chunks in pgvector and retrieve semantic candidates through the existing query contract.
status: done
---

# Goal

Add an explicit, disposable pgvector index for semantic retrieval without changing Markdown vault authority or the existing lexical retriever.

# Architecture

Reuse `vault.SearchSource`, `vault.Document`, `vault.Candidate`, and `vault.QueryResult`. A single `internal/vault/semantic.go` implementation loads effective documents, chunks and embeds them through the OpenAI-compatible `/embeddings` JSON shape, and uses pgx to synchronize/query pgvector. No provider interface, migration framework, ORM, ANN index, or LLM memory inference is added.

# Tech stack

- Add `github.com/jackc/pgx/v5 v5.10.0`, the current stable PostgreSQL-native driver for Go 1.25. Use the native `pgx.Conn`; do not add `database/sql`, an ORM, or a pgvector wrapper.
- Use `net/http`, `encoding/json`, and standard-library hashing/chunking. Send vector values as validated pgvector text literals and cast SQL parameters to `vector`.

# Global constraints

- Do not modify `README.md`.
- Keep all project references lower-case `gnosis`.
- Secrets are read only from `GNOSIS_DATABASE_URL` and optional `GNOSIS_EMBEDDING_API_KEY`.
- Require `GNOSIS_EMBEDDING_URL` and `GNOSIS_EMBEDDING_MODEL`; accept no hidden provider defaults.
- Use exact cosine search. Add ANN only after measured scale requires it.
- Never mutate vault files during semantic index or search operations.

# Scope

Create the semantic configuration, chunking, OpenAI-compatible embedding client, pgvector schema/synchronization, stale-index fingerprint, and semantic query path. Add unit tests and an opt-in real-pgvector integration test. Do not expose CLI commands or MCP tools in this directive.

# Dependencies

- [`gnosis` purpose](../purpose.md) @ `sha256:b432c4496daa35faed91cb294f41fa67522b1d9959daf29575baea09e59cf38f` — Markdown remains portable author-owned knowledge.
- [Use pgvector for semantic knowledge retrieval](../decisions/use-pgvector-semantic-retrieval.md) @ `sha256:70cff98b8205d5297c7e0900066cbe3bdaa9c3745ffdc7d532fd98721ea79cdc` — supplies derived-index, embedding, provenance, explicit-sync, and exact-search constraints.

# Purpose/Decision Changes

- `../decisions/keep-search-sources-and-retrieval-backends-replaceable.md` @ `sha256:da0e4dc0f3941800a8164fdab6b731eabc79bd1c1f05ef1040a9ed539124ffe7` → `../decisions/use-pgvector-semantic-retrieval.md` @ `sha256:70cff98b8205d5297c7e0900066cbe3bdaa9c3745ffdc7d532fd98721ea79cdc`: persistent semantic retrieval is now authorized while the backend-independent contract remains.

# Implementation plan

### Task 1: Lock semantic contracts with focused tests

**Load:** `internal/vault/agent.go`, `internal/vault/retrieval.go`, `internal/vault/search.go`, and the dependency Decision above.
**Files:** create `internal/vault/semantic_test.go`.
**Interfaces:** test `SemanticConfigFromEnv(root string) (SemanticConfig, error)`, `semanticChunks(Document) []semanticChunk`, `(*embeddingClient).embed(context.Context, []string) ([][]float32, error)`, `documentFingerprint([]Document) string`, and `vectorLiteral([]float32) (string, error)`.

- [x] Add table-driven tests proving missing database URL, embedding URL, and model produce field-specific errors; the scope is a stable SHA-256 of the absolute workspace root and never contains credentials.
- [x] Add chunk tests proving title/type/description prefix every chunk, blank-paragraph boundaries are preferred, oversized paragraphs split at 6,000 runes, chunk indexes are stable, and empty bodies still produce one metadata chunk.
- [x] Add an `httptest.Server` test for one batched `POST` request with `model` and `input`, optional bearer authorization, response-index ordering, non-2xx body errors, empty/mismatched vectors, and non-finite value rejection.
- [x] Add deterministic fingerprint and pgvector literal tests; expect sorted URI/revision input and `[1,-2.5,0]` formatting.
- [x] Run `go test ./internal/vault -run 'Semantic|Embedding|VectorLiteral|DocumentFingerprint'`; expect the new tests to fail because the implementation does not exist.
- [x] Commit: `test: define semantic knowledge backend contracts`.

### Task 2: Implement embedding, chunking, and pgvector synchronization

**Load:** failing tests from Task 1 and `go.mod`.
**Files:** create `internal/vault/semantic.go`; modify `go.mod` and `go.sum`.
**Interfaces:**

```go
type SemanticConfig struct {
    DatabaseURL     string
    EmbeddingsURL   string
    EmbeddingsModel string
    EmbeddingsAPIKey string
    Scope           string
    HTTPClient      *http.Client
}

type SemanticIndexResult struct {
    Documents int `json:"documents"`
    Chunks    int `json:"chunks"`
    Scope     string `json:"scope"`
    Fingerprint string `json:"fingerprint"`
}

func SemanticConfigFromEnv(root string) (SemanticConfig, error)
func SyncSemanticIndex(ctx context.Context, root string, config SemanticConfig) (SemanticIndexResult, error)
func QuerySemanticKnowledge(ctx context.Context, root, question string, options QueryOptions, config SemanticConfig) (QueryResult, error)
```

- [x] Add `github.com/jackc/pgx/v5@v5.10.0`; run `go mod tidy`; expect only pgx and its required transitive modules.
- [x] Implement one shared `validateSemanticConfig` used by both exported operations; a 30-second default HTTP client; 6,000-rune paragraph-aware body chunks with the metadata prefix repeated on every request item; deterministic SHA-256 fingerprinting; embedding batches of at most 64 inputs reassembled by response `index`; finite/equal-dimension validation across every batch and the query; and pgvector text serialization exactly as fixed by Task 1.
- [x] Create `vector` extension and these tables with parameterized DML; do not interpolate data:

```sql
CREATE TABLE IF NOT EXISTS gnosis_semantic_indexes (
  scope text PRIMARY KEY,
  model text NOT NULL,
  fingerprint text NOT NULL,
  dimensions integer NOT NULL CHECK (dimensions > 0),
  indexed_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS gnosis_semantic_chunks (
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
);
```

- [x] In `SyncSemanticIndex`, load all effective documents, embed all chunks before opening the replacement transaction, then delete and replace only `config.Scope`; upsert index metadata in the same transaction. Roll back on any insert or metadata failure.
- [x] In `QuerySemanticKnowledge`, reject blank questions, compare the live document fingerprint and configured model with index metadata, embed once, and select the nearest chunk per URI before limiting candidates:

```sql
SELECT uri, type, title, description, origin, revision, 1 - distance AS score
FROM (
  SELECT DISTINCT ON (uri) uri, type, title, description, origin, revision,
         embedding <=> $3::vector AS distance
  FROM gnosis_semantic_chunks
  WHERE scope = $1 AND model = $2
  ORDER BY uri, distance, chunk
) nearest
ORDER BY distance, uri
LIMIT $4;
```

- [x] Normalize options with `normalizedOptions`, return the existing `QueryResult` shape with answer classification from `classifyQuestion`, deterministic candidates scored through `roundScore`, empty path, candidate URIs in `ShouldRead` bounded by `MaxRead`, and `IndexOnly: false`.
- [x] Run the focused Task 1 command and `go test ./internal/vault`; expect green.
- [x] Commit: `feat: add pgvector semantic knowledge backend`.

### Task 3: Prove behavior against real pgvector

**Load:** `internal/vault/search_test.go` fixture helpers and the completed semantic implementation.
**Files:** extend `internal/vault/semantic_test.go`.
**Interfaces:** opt-in `TestSemanticIndexIntegration` reads `GNOSIS_TEST_DATABASE_URL` and skips when unset.

- [x] Add a fake embeddings HTTP server that returns deterministic orthogonal vectors from input text. Against real pgvector, synchronize a two-page vault, query the semantically nearest page, verify compact provenance/revision fields, edit one page and prove stale-index rejection, re-sync and prove the changed revision is returned, and verify a failed sync preserves the preceding complete scope.
- [x] Run `GNOSIS_TEST_DATABASE_URL='<pgvector URL>' go test ./internal/vault -run TestSemanticIndexIntegration -count=1`; expect pass against PostgreSQL with the `vector` extension.
- [x] Run `go test ./... -count=1`; expect all packages green with the integration test skipped when its URL is absent.
- [x] Commit: `test: verify pgvector semantic retrieval`.

# Acceptance criteria

- Effective Markdown documents can be explicitly synchronized into pgvector without any vault-file diff — run `TestSemanticIndexIntegration` and `git diff --exit-code -- ':!docs/directives/add-pgvector-knowledge-backend.md'`; expect a passing test and no implementation-induced vault mutation.
- Semantic search returns the nearest distinct document through `QueryResult`, including canonical URI, origin, and exact indexed revision — inspect the integration-test assertions; expect the selected page and metadata.
- A changed vault is rejected as stale until re-indexed, and a failed replacement leaves the prior scope queryable — run the integration test; expect both failure-path assertions.
- The lexical `QueryKnowledge` path remains green and dependency-free at runtime — run `go test ./internal/vault -run 'Query|Search'`; expect pass without database environment.
- `go test ./... -count=1`, `go vet ./...`, and `gnosis validate --vault .` pass.

# Completion evidence

- Red: `go test ./internal/vault -run 'Semantic|Embedding|VectorLiteral|DocumentFingerprint'` failed on the missing semantic contracts before production code was added.
- Commits: `b8b1784` (contracts), `1954074` (backend), `0cbda4d` (real pgvector coverage), and `ce575d5` (review correction).
- Real pgvector: `TestSemanticIndexIntegration` passed against `pgvector/pgvector:pg18`, covering synchronization, ranking, provenance, stale rejection, re-indexing, and failed-sync preservation.
- Full gate: `mise run checks` passed formatting, vet, unit tests, race tests, build, and validation of 39 Markdown files.
- Delivery: implementation is preserved in the shared `main` workspace; no auxiliary worktree, remote push, or pull request was created.
