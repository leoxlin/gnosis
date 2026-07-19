# Code architecture

gnosis is a small Go module with focused packages and no framework beyond cobra.

## Layout

- `cmd/gnosis/` — the CLI. Verb-resource commands, TOON output (AXI conventions), and HTTP/MCP servers.
- `internal/search/` — knowledge retrieval over the effective vault view: bounded lexical ranking (`lexical.go`) and the optional pgvector semantic index (`semantic.go`). It depends on `internal/vault/`.
- `internal/vault/` — Markdown storage and exact vault operations: configuration (`config.go`), document identity and reads (`document.go`), page parsing and frontmatter (`page.go`), multi-vault composition (`view.go`, `vaults.go`, `bundle.go`), exact graph traversal (`graph.go`, `links.go`), procedure contracts (`procedure.go`), writes (`write.go`), indexes (`index.go`), validation (`validate.go`), scaffolding (`scaffold.go`), and backends (`backend.go`). It does not depend on `internal/search/`.
- `ui/` — the atlas document UI. Alpine.js source (`src/`) bundled by esbuild (`build.mjs`) into the committed single-file `ui.html`, embedded via `embed.go`; rebuild with `mise run ui`.
- `docs/` — the project's own vault and the embedded core bundle (`embed.go` bundles concept types and procedures into the binary).
- `plugins/gnosis/` — the agent plugin manifests and vault gateway skill.

## Key design choices

- **Markdown authoritative** — every store except the optional pgvector index is plain files; the database is disposable derived state.
- **Composition** — vaults layer local → imports → core bundle with first-wins precedence, giving one deterministic view without copying.
- **Contracts over code** — procedures and concept lifecycles are vault records; Go enforces structural procedure, link, and reserved-name contracts.
- **One-way search boundary** — commands call retrieval through `internal/search`, which loads documents through the narrow `internal/vault.LoadDocuments` boundary; lexical always works, while vector and github-wiki are opt-in.
- **Read-only serving** — MCP and HTTP expose knowledge without mutation paths; writes exist only through `apply page`.

## Testing

Every Go file has a sibling test; `mise run checks` is the full gate: gofmt, vet, tests with the race detector, build, UI bundle freshness, and vault validation.
