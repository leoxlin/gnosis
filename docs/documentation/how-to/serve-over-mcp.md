# Serve gnosis over MCP or HTTP

Expose a vault to an agent over MCP stdio:

```bash
gnosis --vault /path/to/workspace serve mcp
```

Configure the agent to start that command as an MCP subprocess. The server
offers eight tools:

- `get_vaults`
- `get_concepts`
- `get_page`
- `get_procedures`
- `trace_graph`
- `search_knowledge`
- `add_memory`
- `search_memory`

Knowledge tools read the effective vault. `get_procedures` discovers eligible
model-invocable Procedures by all-match tags or reads one exact validated
contract. `trace_graph` returns a path or at most 100 deterministically ordered
neighbor edges; truncated neighbor results direct callers to refine direction
or relationship filters. The memory tools require
`GNOSIS_MEMORY_USER_ID` and `GNOSIS_MEMORY_AGENT_ID` and can write through the
selected memory backend.

The same server exposes effective pages as MCP resources over both transports.
Clients can:

- List direct `text/markdown` resources in deterministic pages of at most 100.
- Continue discovery with the opaque cursor returned by the previous page.
- List the `gnosis://{vault}/{+path}` template for canonical page URIs.
- Read a resource with its effective origin and revision in `_meta`.

Resources let an MCP host select and attach vault pages as application-controlled
context. The `get_page` tool remains available when the model should choose and
read a page itself.

To serve the browser atlas, JSON API, and streamable MCP together:

```bash
gnosis --vault /path/to/workspace serve http \
  --address 127.0.0.1:8080
```

Open `http://127.0.0.1:8080/` for the atlas or connect an MCP client to
`http://127.0.0.1:8080/mcp`.

The JSON endpoints are:

- `GET /api/v1/vaults`
- `GET /api/v1/concepts?type=<type>`
- `GET /api/v1/pages`
- `GET /api/v1/page?uri=<gnosis-uri>`
- `GET /api/v1/graph`
- `GET /api/v1/search?question=<question>`

The default address is loopback-only. Review network exposure and memory
backend credentials before binding to another interface.
