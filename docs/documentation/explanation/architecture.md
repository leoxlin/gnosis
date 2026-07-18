# Code architecture

gnosis is a small Go module with two packages and no framework beyond cobra.

## Layout

- `cmd/gnosis/` — the CLI. Verb-resource commands, TOON output (AXI conventions), the atlas UI (`ui.html`), and HTTP/MCP servers.
- `internal/vault/` — the vault library: configuration (`config.go`), page model and frontmatter (`page.go`), multi-vault composition (`search.go`, `vaults.go`, `bundle.go`), lexical retrieval (`retrieval.go`), pgvector semantics (`semantic.go`), graph (`agent.go`, `links.go`), procedure contracts (`procedure.go`), writes (`write.go`), indexes (`index.go`), validation (`validate.go`), scaffolding (`scaffold.go`), backends (`backend.go`).
- `docs/` — the project's own vault and the embedded core bundle (`embed.go` bundles concept types and procedures into the binary).
- `plugins/gnosis/` — the agent plugin manifests and vault gateway skill.

## Key design choices

- **Markdown authoritative** — every store except the optional pgvector index is plain files; the database is disposable derived state.
- **Composition** — vaults layer local → imports → core bundle with first-wins precedence, giving one deterministic view without copying.
- **Contracts over code** — procedures and concept lifecycles are vault records; Go enforces structural procedure, link, and reserved-name contracts.
- **Replaceable boundaries** — retrieval backends and storage backends sit behind small interfaces; lexical always works, vector and github-wiki are opt-in.
- **Read-only serving** — MCP and HTTP expose knowledge without mutation paths; writes exist only through `apply page`.

## Testing

Every Go file has a sibling test; `mise run checks` is the full gate: gofmt, vet, tests with the race detector, build, and vault validation.
