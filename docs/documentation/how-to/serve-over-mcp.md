# Serve over MCP

Expose read-only vault knowledge to agents over the Model Context Protocol, or browse it over HTTP.

## stdio (agent subprocess)

    gnosis serve mcp

Tools: `get_vaults`, `get_concepts`, `get_page`, `search_knowledge`. Register it in your agent's MCP configuration as a stdio server pointing at the vault (the server honors `--vault`).

## HTTP + streamable MCP

    gnosis serve http --address 127.0.0.1:8080

- `GET /` — the atlas graph UI.
- `GET /api/v1/vaults|concepts|pages|page?uri=...|graph|search?q=...` — JSON.
- `POST /mcp` — streamable HTTP MCP endpoint with the same read-only tools.

## Guarantees

The serve paths are read-only by design; knowledge changes only through `apply page`. See the [serve-read-only decision](../../decisions/serve-read-only-knowledge-over-mcp-stdio.md).
