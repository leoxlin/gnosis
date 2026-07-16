---
type: Directive
title: Reshape CLI around resource verbs
description: Replace legacy listing and query commands with get, search, and resource-scoped index commands that expose semantic knowledge cleanly.
status: done
---

# Goal

Make the gnosis command surface predictable and resource-oriented: `gnosis get vaults`, `gnosis get concepts`, `gnosis search knowledge`, `gnosis index vault`, and `gnosis index knowledge`.

# Architecture

Keep Cobra and the existing command-per-file layout. Reparent existing handlers instead of adding compatibility aliases. The search command defaults to the new vector backend and accepts `--backend lexical` for offline live-vault retrieval.

# Global constraints

- Do not modify `README.md`.
- Remove obsolete top-level `vaults`, `concepts`, and `query` commands completely.
- Preserve `read`, `write`, `graph`, `procedure`, `validate`, `setup`, and `scaffold`; this directive does not rename unrelated mutation or exact-record verbs.
- Keep stdout machine-readable/output-only and errors on stderr.

# Scope

Reorganize command construction and tests, expose semantic sync/search, and update active decisions, procedures, skills, and integration commands that invoke removed forms. Historical completed directive evidence may retain the commands it actually executed.

# Dependencies

- [Add pgvector semantic knowledge backend](add-pgvector-knowledge-backend.md) @ `sha256:ea8620578c83484e95f0d1fccf8a481ffdf5e665eda321a30ad14ec70d3a8530` — supplies the completed `SemanticConfigFromEnv`, `SyncSemanticIndex`, and `QuerySemanticKnowledge` contract.
- [Use pgvector for semantic knowledge retrieval](../decisions/use-pgvector-semantic-retrieval.md) @ `sha256:70cff98b8205d5297c7e0900066cbe3bdaa9c3745ffdc7d532fd98721ea79cdc` — vector search remains explicit derived retrieval.

# Implementation plan

### Task 1: Define the breaking command tree in tests

**Load:** `cmd/gnosis/main.go`, `vaults.go`, `concepts.go`, `query.go`, `index.go`, and command tests.
**Files:** create `cmd/gnosis/get_test.go`, `cmd/gnosis/search_test.go`, and `cmd/gnosis/index_test.go`; modify or delete `cmd/gnosis/vaults_test.go` after moving every assertion; modify `cmd/gnosis/setup_test.go` only if shared helpers move.
**Interfaces:** `newGetCommand(io.Writer)`, `newSearchCommand(io.Writer)`, and grouped `newIndexCommand(io.Writer)`.

- [x] Assert root help contains `get`, `search`, and `index`, and rejects removed `vaults`, `concepts`, and `query` commands.
- [x] Move existing vault/concept JSON and text expectations to `get vaults` and `get concepts [type]`; prove a second concept argument and removed `--type` fail.
- [x] Prove `search knowledge <question>` defaults to vector, `--backend lexical` returns the existing `QueryResult`, unknown backends fail, and `index knowledge` reports `SemanticIndexResult` as text or `--json`.
- [x] Prove bare `index` errors and `index vault` preserves current generated-index behavior.
- [x] Run `go test ./cmd/gnosis -run 'Get|Search|Index|Root'`; expect red.
- [x] Commit: `test: define resource-oriented CLI`.

### Task 2: Reparent commands and delete obsolete entry points

**Load:** failing Task 1 tests and `cmd/gnosis/main.go`.
**Files:** create `cmd/gnosis/get.go` and `cmd/gnosis/search.go`; modify `cmd/gnosis/main.go` and `cmd/gnosis/index.go`; delete `cmd/gnosis/vaults.go`, `cmd/gnosis/concepts.go`, and `cmd/gnosis/query.go`; move `questionArgs`, `validateQueryOptions`, and `writeQueryText` into `search.go`; move all `vaults_test.go` assertions into `get_test.go` and delete the obsolete test file.
**Interfaces:**

```text
gnosis get vaults [--vault PATH] [--json]
gnosis get concepts [TYPE] [--vault PATH] [--json]
gnosis search knowledge QUESTION [--backend vector|lexical] [--top N] [--max-read N] [--depth N] [--vault PATH] [--json]
gnosis index vault [--vault PATH]
gnosis index knowledge [--vault PATH] [--json]
```

- [x] Reuse the existing vault/concept/query renderers and validation; use `command.Context()` for semantic calls. Do not create aliases or deprecation shims.
- [x] Register only the new command parents at root and remove dead constructors/functions after callers move.
- [x] Add `runContext(context.Context, []string, io.Writer, io.Writer) error`; keep `run` as the background-context test wrapper, and have `main` create/defer `signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)` before calling `runContext`. Use `ExecuteContext`; never call `os.Exit` inside Cobra handlers.
- [x] Run focused command tests and `go test ./...`; expect green.
- [x] Commit: `feat: reshape gnosis CLI around resources`.

### Task 3: Remove active obsolete command references

**Load:** all non-README matches from `rg -n 'gnosis (vaults|concepts|query search|query graph|index( |`))' docs plugins integration`.
**Files:** update active `docs/decisions/*.md`, `docs/procedures/**/*.md`, `plugins/gnosis/skills/**/*.md`, and integration scripts/instructions that invoke removed forms; do not rewrite completed directive evidence solely to modernize history.
**Interfaces:** exact replacements are `concepts`→`get concepts`, `query search|query graph`→`search knowledge --backend lexical`, and bare index→`index vault`.

- [x] Apply every active replacement and update prose describing the command tree; preserve procedure semantics and exact URIs.
- [x] Run `rg -n 'gnosis (vaults|concepts|query search|query graph)( |`)' docs/decisions docs/procedures plugins integration`; expect no matches.
- [x] Run `gnosis validate --vault .` and `go test ./... -count=1`; expect green.
- [x] Commit: `docs: update resource-oriented gnosis commands`.

# Acceptance criteria

- The requested examples work exactly — run `go run ./cmd/gnosis get vaults --json` and `go run ./cmd/gnosis get concepts Decision --json`; expect valid catalogs.
- Vector RAG is first-class — with semantic environment configured, run `go run ./cmd/gnosis search knowledge 'semantic question' --json`; expect the pgvector-backed `QueryResult`; with `--backend lexical`, expect no external service use.
- Removed forms fail as unknown commands and have no compatibility aliases — run command tests and root help assertions.
- Index lifecycle is explicit by resource — `index vault` changes generated Markdown only when configured; `index knowledge` changes only derived pgvector state.
- Active workflow knowledge contains no removed invocation, and `go test ./...`, `go vet ./...`, and vault validation pass.

# Completion evidence

- Red: `go test ./cmd/gnosis -run 'Get|Search|Index|Root'` failed on the missing semantic-index result renderer before production command changes.
- Commits: `4791ead` (command contracts), `158928d` (resource-oriented CLI), and `8dc67f7` (active workflow updates).
- Command acceptance: `go run ./cmd/gnosis get vaults --json`, `go run ./cmd/gnosis get concepts Decision --json`, and lexical `search knowledge` returned valid current-vault results; focused tests prove vector defaulting, removed-command failures, and resource-scoped indexing.
- Full gate: `mise run checks` passed formatting, vet, unit tests, race tests, build, and validation of 39 Markdown files.
- Delivery: implementation is preserved in the shared `main` workspace; no auxiliary worktree, remote push, or pull request was created.
