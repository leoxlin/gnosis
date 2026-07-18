## ADDED Requirements

### Requirement: HTTP serving exposes read-only knowledge
gnosis SHALL serve the document UI, JSON endpoints for vaults, concepts, pages, graph, and search, and streamable MCP from one configurable HTTP listener.

#### Scenario: Read a page over HTTP
- **WHEN** a client requests an existing canonical page URI
- **THEN** the API returns its document metadata and Markdown as JSON without changing the vault

### Requirement: MCP exposes a small read-only tool surface
gnosis SHALL expose tools for listing vaults and concepts, reading one exact page, and searching knowledge over stdio and streamable HTTP transports.

#### Scenario: Call a read tool
- **WHEN** an MCP client supplies valid tool input
- **THEN** gnosis returns the same current vault data and retrieval semantics as the corresponding internal operation

#### Scenario: Request mutation
- **WHEN** an MCP client inspects the available tool surface
- **THEN** no page, shell, configuration, or index mutation tool is available

### Requirement: Transport channels remain isolated
gnosis SHALL keep MCP frames and HTTP JSON free from ordinary CLI output and route server diagnostics outside protocol payloads.

#### Scenario: Serve MCP in a subprocess
- **WHEN** a client exchanges MCP frames over stdio
- **THEN** every stdout message is valid protocol data and diagnostics do not corrupt the session

### Requirement: Servers stop gracefully
gnosis SHALL propagate command cancellation and shut down HTTP service within a bounded timeout.

#### Scenario: Cancel an HTTP server
- **WHEN** the command context is canceled
- **THEN** gnosis stops accepting work, completes shutdown, and returns without leaking the listener
