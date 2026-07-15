---
type: Directive
title: Add MCP server mode
description: Serve gnosis vault listing, concept listing, exact page reads, and semantic or lexical knowledge search over MCP stdio.
status: done
---

# Goal

Run gnosis as a read-only MCP stdio server so agents can retrieve exact and RAG-based vault knowledge through stable tools.

# Architecture

Use the official Go MCP SDK and expose thin tool handlers over existing vault APIs. Mirror mem0's small get/list/search MCP surface, but omit conversational memory inference and mutations because gnosis Markdown is authored durable knowledge. Support stdio only.

# Tech stack

- Add `github.com/modelcontextprotocol/go-sdk v1.6.1`, the stable official SDK compatible with Go 1.25 and the 2025-11-25 MCP specification.
- Use `mcp.NewServer`, generic `mcp.AddTool`, and `mcp.StdioTransport`; do not implement JSON-RPC or transport framing locally.

# Global constraints

- Do not modify `README.md`.
- Never log or print diagnostics to stdout while serving stdio MCP.
- MCP is read-only in this delivery. Do not expose write, delete, update, shell, or arbitrary path tools.
- Use the same vector-by-default and explicit lexical fallback behavior as the CLI.

# Scope

Add `gnosis serve mcp`, four read-only tools, cancellation, and SDK-level tests. HTTP/SSE transport, authentication, hosted service behavior, prompts, resources, and mutation tools are out of scope.

# Dependencies

- [Add pgvector semantic knowledge backend](gnosis://core/directives/add-pgvector-knowledge-backend.md) @ `sha256:ea8620578c83484e95f0d1fccf8a481ffdf5e665eda321a30ad14ec70d3a8530` — supplies the completed semantic retrieval contract.
- [Reshape CLI around resource verbs](gnosis://core/directives/reshape-cli-around-resource-verbs.md) @ `sha256:513da291e5fa349fd1dd837f9cefab4e598d62c554b3f1c01cd93c3d9946ccbe` — supplies the completed final command tree, context propagation, and search defaults.
- [Use pgvector for semantic knowledge retrieval](gnosis://core/decisions/use-pgvector-semantic-retrieval.md) @ `sha256:70cff98b8205d5297c7e0900066cbe3bdaa9c3745ffdc7d532fd98721ea79cdc` — preserves read-only derived retrieval and exact-page separation.

# Implementation plan

### Task 1: Specify MCP tools through an in-process client

**Load:** official MCP Go SDK server/client stdio or in-memory transport examples; final `get` and `search` handlers; `internal/vault/agent.go`.
**Files:** create `cmd/gnosis/serve_test.go`.
**Interfaces:** tools `get_vaults`, `get_concepts`, `get_page`, and `search_knowledge` with typed JSON inputs/outputs. `get_concepts` returns one stable `conceptsOutput` containing `concept_types`, optional `type`, and `concepts`, rather than a Go union.

- [x] Add tests that connect an SDK client to the server, list exactly four tools, call each tool against a fixture vault, decode structured JSON results, and prove invalid URI/type/backend inputs return MCP tool errors rather than terminating the session.
- [x] Assert `search_knowledge` defaults to vector and accepts `backend: "lexical"`, `top`, `max_read`, and `depth`; use lexical in hermetic fixture calls.
- [x] Assert server stdout contains protocol frames only and cancellation closes the session.
- [x] Run `go test ./cmd/gnosis -run MCP`; expect red.
- [x] Commit: `test: define gnosis MCP server contract`.

### Task 2: Implement read-only stdio server mode

**Load:** failing tests and `cmd/gnosis/main.go` context/cancellation behavior.
**Files:** create `cmd/gnosis/serve.go`; modify `cmd/gnosis/main.go`, `go.mod`, and `go.sum`.
**Interfaces:**

```text
gnosis serve mcp [--vault PATH]

get_vaults({}) -> vault.VaultCatalog
get_concepts({"type": "optional exact type"}) -> {"concept_types": [...], "type": "...", "concepts": [...]}
get_page({"uri": "gnosis://..."}) -> vault.Page
search_knowledge({"question":"...", "backend":"vector|lexical", "top":3, "max_read":3, "depth":3}) -> vault.QueryResult
```

- [x] Add `github.com/modelcontextprotocol/go-sdk@v1.6.1` and tidy modules.
- [x] Build a `newMCPServer(vaultPath string)` with only the four tools. Return typed structured results and concise errors; normalize both concept catalog variants into `conceptsOutput`; route every operation directly to the same vault functions used by CLI handlers. Resolve semantic environment lazily inside vector search calls so the server and lexical tools start without database credentials.
- [x] Add `serve` as a Cobra parent and `mcp` as a no-argument child. Run the server with `command.Context()` and `mcp.StdioTransport`; write no startup banner.
- [x] Run focused MCP tests, `go test ./... -count=1`, and `go vet ./...`; expect green.
- [x] Commit: `feat: serve gnosis knowledge over MCP`.

### Task 3: Verify a real subprocess handshake

**Load:** official SDK `mcp.CommandTransport` example.
**Files:** extend `cmd/gnosis/serve_test.go`; update non-README plugin/integration configuration only if a checked-in fixture needs the final invocation.
**Interfaces:** client starts the built command with `serve mcp --vault <fixture>`.

- [x] Build a temporary gnosis binary, connect with SDK `CommandTransport`, initialize, list tools, call `get_page` and lexical `search_knowledge`, close the session, and assert clean process exit.
- [x] Run `go test ./cmd/gnosis -run MCPSubprocess -count=1`; expect pass with no database or network.
- [x] Run `go test -race ./... -count=1`, `go build ./...`, and `gnosis validate --vault .`; expect green.
- [x] Commit: `test: verify gnosis MCP stdio mode`.

# Acceptance criteria

- `gnosis serve mcp` completes a real MCP stdio initialize/list/call/close lifecycle — run `TestMCPSubprocess`; expect all four tools and clean exit.
- MCP semantic search uses the same pgvector backend and stale-index behavior as CLI; lexical search remains an explicit hermetic fallback.
- Tool responses preserve canonical URI, origin, and revision; invalid input becomes tool-level errors without corrupting the protocol stream.
- No mutation or arbitrary filesystem tool is exposed — inspect the exact four-tool list assertion.
- `go test -race ./... -count=1`, `go vet ./...`, `go build ./...`, and vault validation pass.

# Completion evidence

- Red: `go test ./cmd/gnosis -run MCP` failed on the missing `newMCPServer` and normalized `conceptsOutput` contracts before production code was added.
- Commits: `550b2d3` (in-process contracts), `505e7cd` (read-only MCP server), and `261ab9e` (real subprocess handshake).
- Protocol acceptance: `TestMCPSubprocess` built a temporary gnosis binary, initialized over `CommandTransport`, listed exactly four tools, called `get_page` and lexical `search_knowledge`, and closed cleanly with no stderr; successful protocol parsing proves stdout contained only MCP frames.
- Semantic path: in-process tests prove vector defaulting and lazy environment resolution; code review confirms direct routing to the same `QuerySemanticKnowledge` and `QueryKnowledge` functions as the CLI.
- Full gate: `mise run checks` passed formatting, vet, unit tests, race tests, build, and validation of 39 Markdown files.
- Delivery: implementation is preserved in the shared `main` workspace; no auxiliary worktree, remote push, or pull request was created.
