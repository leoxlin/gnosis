# Code architecture

gnosis is a Go command with narrow package boundaries. The architecture keeps
plain Markdown authoritative while allowing retrieval and serving layers to
evolve independently.

## Package flow

```text
CLI / HTTP / MCP
        │
        ├── memory service ── vault or Mem0
        ├── knowledge change ─ validate / diff / conditional write
        │
        └── search ────────── lexical or pgvector
                │
              vault
                │
        Markdown + configuration
```

`internal/vault` owns configuration, typed pages, canonical identity, links,
composition, graph traversal, procedures, validation, writes, indexes,
scaffolding, and storage backends. It does not depend on search.

`internal/search` builds bounded retrieval over the effective vault view.
Lexical retrieval works directly from live documents. Semantic retrieval adds
an optional PostgreSQL/pgvector index.

`internal/memory` selects either the vault or Mem0 and exposes one scoped
add/search service. The command, MCP, and HTTP layers call these packages rather
than reimplementing domain behavior.

## Interfaces

`cmd/gnosis` defines the Cobra command tree, TOON output, MCP server, and HTTP
API. The browser atlas source lives in `ui/`; esbuild produces the committed
single-file bundle embedded by the command.

The `docs/` tree is both project knowledge and the source of bundled Concept
Types and Procedures. `plugins/gnosis/` packages the procedure gateway skill
for supported agents.

## Design constraints

- Markdown pages are authoritative; semantic indexes are disposable.
- Effective vaults resolve in a fixed precedence order.
- Search depends on vault reads, never the reverse.
- Knowledge-change planning is read-only. Explicitly enabled apply and memory
  tools are the only MCP mutations, and each delegates persistence to its
  owning service.
- Procedure behavior remains in reviewable contracts while Go enforces
  structural invariants.

`mise run checks` exercises these boundaries with formatting, vetting, normal
and race-enabled tests, a build, UI bundle verification, and vault validation.
