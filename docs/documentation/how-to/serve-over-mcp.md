# Serve gnosis over MCP or HTTP

Expose a vault to an agent over MCP stdio:

```bash
gnosis --vault workspace serve mcp
```

Configure the agent to start that command as an MCP subprocess. The default
server offers seventeen tools:

- `get_vaults`
- `get_concepts`
- `get_page`
- `get_history`
- `get_diff`
- `get_changes`
- `get_evidence_context`
- `audit_knowledge`
- `get_procedures`
- `trace_graph`
- `search_knowledge`
- `propose_knowledge_change`
- `record_trace`
- `get_run_trace`
- `propose_run_learning`
- `add_memory`
- `search_memory`

Knowledge tools read the effective vault. `get_procedures` discovers eligible
model-invocable Procedures by all-match tags or reads one exact validated
contract. `get_evidence_context` returns bounded cited excerpts, typed paths,
retrieval passes, omissions, and gaps without generating an answer.
`audit_knowledge` returns bounded deterministic health findings and recommended
Procedure routing without modifying the vault.
`trace_graph` returns a path or at most 100 deterministically ordered neighbor
edges; truncated neighbor results direct callers to refine direction or
relationship filters. The memory tools require
`GNOSIS_MEMORY_USER_ID` and `GNOSIS_MEMORY_AGENT_ID` and can write through the
selected memory backend.

The trace tools require an absolute `GNOSIS_TRACE_DIR` and a non-empty fixed
`GNOSIS_TRACE_AGENT_ID`. `record_trace` accepts explicit `run`, `plan`, `tool`,
`patch`, `test`, `failure`, `outcome`, `knowledge_use`, and `feedback` entries.
Knowledge-use and feedback entries also require `knowledge_uri` and the exact
`knowledge_revision`; feedback is `helpful`, `harmful`, `irrelevant`, or
`unassessed`.

`get_run_trace` reads one exact run in sequence order. Use `max_entries` and
`max_characters` to bound the response, then pass its sequence `continuation`
as `cursor`. A complete run has an outcome and no gaps, malformed records,
hash mismatches, or truncation; otherwise `diagnostics` explains why it is
incomplete.

`propose_run_learning` accepts explicitly selected run IDs and one
caller-authored learning key. For cross-run candidates, every run must bind the
same Procedure through string `metadata.procedure_uri` and
`metadata.procedure_revision` fields. Each outcome must carry a boolean
`metadata.success`, which classifies its exact evidence as supporting or
contradicting. The tool rechecks every retained content hash, builds only an
Event or Reflection candidate, and returns the normal knowledge-change plan
without writing either traces or curated Markdown. Apply a reviewed applicable
plan separately through operator-enabled `apply_knowledge_change`.

`propose_knowledge_change` validates and diffs one complete typed Markdown
candidate without writing. To register the corresponding mutation tool, the
server operator must opt in:

```bash
gnosis --vault workspace serve mcp --allow-knowledge-writes
```

That adds `apply_knowledge_change`. Apply requires the exact URI, candidate,
expected revision or expected absence, and digest returned during planning. It
refreshes and revalidates the target, rejects stale or changed plans, and never
physically deletes a page. Operator enablement makes the capability available;
the MCP host remains responsible for per-call user approval.

The same server exposes effective pages as MCP resources over both transports.
Clients can:

- List direct `text/markdown` resources in deterministic pages of at most 100.
- Continue discovery with the opaque cursor returned by the previous page.
- List the `gnosis://{vault}/{+path}` template for canonical page URIs.
- Read a resource with its effective origin and revision in `_meta`.

Resources let an MCP host select and attach vault pages as application-controlled
context. The `get_page` tool remains available when the model should choose and
read a page itself.

Resource support is registered before subscription handling. Clients may
subscribe to a concrete canonical page URI and unsubscribe through the standard
MCP resource methods. The server sends `resources/updated` after a successful
server write, or after a later gnosis operation refreshes the vault and observes
a changed revision or effective origin. Added and removed effective pages also
produce `resources/list_changed`. External edits do not produce a notification
until a gnosis server operation observes them; there is no background watcher
or polling loop. Subscriptions are session-local and are removed on unsubscribe
or disconnect.

The three history tools are read-only:

- `get_history` returns bounded newest-first committed history plus any explicit
  current working revision.
- `get_diff` compares two exact content revisions of one canonical page.
- `get_changes` establishes or resumes a committed effective-view cursor.

History and change-feed cursors are opaque. Change cursors are scoped to the
repository identities and effective-vault composition that created them and
expire after incompatible history rewrites or pruning.

To serve the browser atlas, JSON API, and streamable MCP together:

```bash
gnosis --vault workspace serve http \
  --address 127.0.0.1:8080
```

Open `http://127.0.0.1:8080/` for the atlas or connect an MCP client to
`http://127.0.0.1:8080/mcp`.
Add `--allow-knowledge-writes` to the HTTP command only when the streamable MCP
server should also advertise `apply_knowledge_change`.

The JSON endpoints are:

- `GET /api/v1/vaults`
- `GET /api/v1/concepts?type=<type>`
- `GET /api/v1/pages?limit=<n>&cursor=<cursor>&q=<text>&type=<type>`
- `GET /api/v1/page?uri=<gnosis-uri>`
- `GET /api/v1/history?uri=<gnosis-uri>&cursor=<cursor>&limit=<n>`
- `GET /api/v1/diff?uri=<gnosis-uri>&from=<revision>&to=<revision>&limit=<n>`
- `GET /api/v1/changes?cursor=<cursor>&limit=<n>`
- `GET /api/v1/graph`
- `GET /api/v1/graph/neighbors?uri=<gnosis-uri>&direction=<out|in|both>&relation=<relation>&limit=<n>`
- `GET /api/v1/graph/path?uri=<gnosis-uri>&target=<gnosis-uri>&direction=<out|in|both>&relation=<relation>&depth=<n>`
- `GET /api/v1/procedures?tag=<tag>` or `GET /api/v1/procedures?uri=<gnosis-uri>`
- `GET /api/v1/search?question=<question>`
- `POST /api/v1/context`
- `POST /api/v1/audit/knowledge`

The default address is loopback-only. Review network exposure, host approval
policy, and memory backend credentials before binding to another interface or
enabling general knowledge writes.
