# Code intelligence

gnosis can build immutable, syntax-based indexes for explicitly configured local Git repositories. Code indexes are separate from Markdown vault indexing, curated knowledge, and knowledge history.

## Configure a scope

Add a named scope to `.gnosis/config.toml`:

```toml
[[code_scopes]]
name = "app"
root = "/absolute/path/to/repository"
languages = ["go", "typescript", "javascript"]
max_files = 10000
max_file_bytes = 1048576
max_records = 200000
max_diagnostics = 10000
max_results = 100
max_traversal = 1000
live = true
freshness_wait = "2s"
```

Scope names are unique. Roots must be absolute local Git repositories, language allowlists must be non-empty, and all bounds must be positive. gnosis reads only tracked regular files through an `os.Root`; it excludes unsupported, binary, oversized, generated, vendored, dependency, ignored, and untracked content.

Live indexing is opt-in and disabled by default. `freshness_wait` is the only
caller-tunable live bound; it must be greater than zero and no more than 30
seconds. Debounce, maximum batch latency, event/path/directory/batch limits,
retry count, safety interval, retention, and shutdown time are pinned defaults
returned by code-index status.

## Install trusted parsers

The initial release supports Linux on amd64. It pins the MIT-licensed `tree-sitter-language-pack` v1.13.7 and its Go/CGo runtime dependencies; individual bundled grammars retain their upstream licenses. gnosis records the platform, parser ABI, release-manifest digest, platform-bundle digest, and each installed library digest in a gnosis-owned manifest. The upstream release manifest is unsigned, so installation relies on the release and digests pinned in the gnosis binary and should be treated as native-code installation.

Install parsers explicitly before building:

```console
gnosis parsers list
gnosis parsers install go typescript javascript --scope app
gnosis parsers status go typescript javascript --fields language,installed,release,platform,abi,library_digest
```

Installation is the only code-intelligence path that may download parser assets. Builds, reads, serving, and MCP tools verify the local manifest and libraries and fail closed when assets are missing or changed.

## Build and query

```console
gnosis index code --scope app
gnosis get code-index-status --scope app
gnosis search code Handler --scope app
gnosis search code Handler --scope app --fields id,name,qualified_name,kind,path,language,signature,span
gnosis get code-symbol SYMBOL_ID --scope app
gnosis get code-diagnostics --scope app --language go
gnosis graph code SYMBOL_ID --scope app --direction outgoing
gnosis index code --scope app --dispose-generation OLD_GENERATION_ID
```

Each build captures Git revision and dirty content plus configuration, parser, query, normalizer, schema, and bound digests. Publication atomically replaces the scope's `current` pointer only after a complete generation is written. Identical builds are no-ops, failed builds preserve the prior generation, and reads stay pinned to one generation. Change the source or toolchain and rebuild when a read reports `not_current`.

## Keep a scope live while serving

Live indexing is supported on Linux amd64, the same platform as the initial
trusted parser runtime. It uses pinned `fsnotify` v1.9.0 observation, attaches
the watcher before its initial Git scan, and treats notifications only as
bounded hints. The Git index, tracked files, confined final file bytes, and a
second verified snapshot remain authoritative. Duplicate or permission-only
events can advance the freshness watermark without creating a generation;
overflow, Git revision changes, ambiguous renames, source races, and periodic
safety mismatches trigger a full reconciliation.

Start either serving host after enabling a scope:

```console
gnosis serve mcp
gnosis serve http --address 127.0.0.1:8080
```

The host opens one workspace per enabled scope. All HTTP MCP sessions share it;
stdio uses it for the protocol lifetime. A second host or explicit build for
the same scope returns `scope_busy` until the owner shuts down. Shutdown stops
observation and retry work, lets active bounded read callbacks finish, and
closes the watcher, analyzer, source root, store ownership, and writer lease.

Inspect status at any time:

```console
gnosis get code-index-status --scope app
```

The MCP `get_code_index_status` tool is also immediate. `starting` means the
verified baseline is incomplete; `pending` means a newer observed watermark is
queued or analyzing; `current` means the selected generation covers the latest
observed watermark; `degraded` preserves but does not serve the last durable
generation; and `closed` rejects new reads. Fact-bearing tools wait only for
`freshness_wait`, then return typed state, observed and published watermarks,
the last generation, a bounded reason, and recovery guidance without stale
facts. Partial syntax and missing parser coverage can still be current because
their diagnostics represent the current source.

The first live open atomically migrates the legacy v1 generation-ID `current`
pointer to the v2 selector. Explicit builds write v2 with no live epoch. A
prior epoch is historical after restart, so a new workspace proves a new
baseline before becoming current. Do not hand-edit the selector.

Watcher, source, analyzer, and storage failures retry within fixed bounds and
perform full reconciliation before recovery. Configuration, trusted parser
manifest, parser-integrity, or ABI changes require a host restart; native
libraries are never hot-reloaded. Live startup, events, safety checks, retries,
queries, HTTP, and MCP remain offline and never install or select parsers.

To roll back, set `live = false` and restart. Existing v2 generations remain
readable. Use `gnosis index code --scope app` as the explicit-build fallback;
if rolling back to code that predates v2, remove only the disposable code-index
cache and rebuild. Curated knowledge is unaffected.

Results report capability coverage. Parsing, structure, and definitions are syntax-derived; imports and other relationships are syntactic evidence unless explicitly marked otherwise. Partial or unsupported coverage means gnosis did not infer missing semantic facts.

Code generations and parser assets live in the user cache. The disposal command removes only an exact non-current generation while holding the scope writer lease. A retained generation can be selected for rollback before disposing newer non-current generations. Do not edit generation contents, the `current` pointer, or parser manifests in place, because digest verification will reject them.

## MCP tools

The stdio and streamable HTTP servers expose the same read-only operations:

- `search_code`
- `get_code_symbol`
- `trace_code`
- `get_code_diagnostics`
- `get_code_index_status`

The tools accept configured scope names and bounded filters only. They cannot accept repository paths, custom parser queries, install parsers, build generations, or mutate curated knowledge.
