# Serve gnosis over MCP or HTTP

Expose a vault to an agent over MCP stdio:

```bash
gnosis --vault /path/to/workspace serve mcp
```

Configure the agent to start that command as an MCP subprocess. The server
offers six tools:

- `get_vaults`
- `get_concepts`
- `get_page`
- `search_knowledge`
- `add_memory`
- `search_memory`

Knowledge tools read the effective vault. The memory tools require
`GNOSIS_MEMORY_USER_ID` and `GNOSIS_MEMORY_AGENT_ID` and can write through the
selected memory backend.

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
