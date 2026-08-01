# Configuration reference

## Resolution

gnosis builds the selectable vault registry from:

1. the nearest `gnosis.local.toml`, otherwise the nearest `gnosis.toml`
2. the user configuration at `~/.config/gnosis.toml`

`--vault <name>` selects one canonical name declared by either source. An
omitted flag selects only the nearest local `[vault]`; without one, operational
commands return a configuration error. Paths and Git URLs are valid only as
`vault_root` values. Identical declarations of one name and normalized target
are de-duplicated; declarations that reuse a name for different targets are
rejected.

## `[vault]`

| Field | Default | Meaning |
|---|---|---|
| `vault_name` | required | Canonical URI authority |
| `vault_root` | required for filesystem and Git vaults | Knowledge directory |
| `backend` | empty | Primary storage backend: `github-wiki` or `s3` |
| `repository` | empty | GitHub `owner/repository` used by that backend |
| `s3_bucket` | empty | Bucket required by the `s3` backend |
| `s3_region` | empty | AWS region required by the `s3` backend |
| `s3_prefix` | empty | Optional normalized object prefix for the `s3` backend |
| `entry_points` | empty | Canonical page URIs excluded from orphan findings |
| `link_format` | `relative` | Preferred internal link form: `relative` or `absolute` |
| `link_format_strict` | `false` | Treat link-style violations as errors |
| `vault_index` | `true` | Require and generate directory `index.md` files |
| `vault_log` | `true` | Require a root `log.md` |

Example standalone vault:

```toml
[vault]
vault_name = "knowledge"
vault_root = "."
entry_points = ["gnosis://knowledge/home.md"]
link_format = "relative"
link_format_strict = false
vault_index = true
vault_log = true
```

Example S3-backed vault:

```toml
[vault]
vault_name = "knowledge"
backend = "s3"
s3_bucket = "example-knowledge"
s3_region = "us-east-1"
s3_prefix = "vaults/knowledge"
```

S3 vaults synchronize one committed snapshot into the user cache before reads.
Validated changes publish content-addressed objects and conditionally replace
the current snapshot pointer. `vault_root` and `repository` are invalid with
this backend.

## `[[vaults]]`

Each imported vault has a name and either a local root or an explicit HTTPS or
SSH Git URL:

```toml
[[vaults]]
vault_name = "team"
vault_root = "../team-vault"
```

```toml
[[vaults]]
vault_name = "remote"
vault_root = "https://github.com/example/knowledge-vault.git"
```

The effective precedence is the primary vault, declared imports in order, then
the embedded core bundle. The first source for a vault-relative path wins.
Import cycles are invalid.

Remote imports are cloned into
`os.UserCacheDir()/gnosis/git-vaults/<sha256-of-normalized-url>` and refreshed
with a fast-forward-only pull during composition. Repeated declarations of the
same normalized URL reuse one checkout. An imported remote participates in
reads, validation, search, and serving, but it does not become the workspace's
publisher. To mutate and publish that repository, declare its URL under a
canonical name and select that name with `--vault`.

Remote authentication uses existing Git credential helpers and SSH
configuration. Supported locators begin with `https://` or `ssh://`. Embedded
HTTPS credentials, SSH passwords, queries, fragments, branch or revision
selection, repository subdirectories, and SCP-like `git@host:path` syntax are
not supported.

## `[[hooks]]`

Hooks run synchronously, in declaration order, after a changed authoritative
page is persisted and any required remote publication succeeds. A delivery
failure is returned with the successful mutation; it does not roll back,
repeat, or queue the write. Plans, rejected writes, no-ops, reads, and generated
indexes do not run hooks.

Each hook requires a unique `name`, a `kind` of `command` or `webhook`, and one
scope:

| Scope | Target |
|---|---|
| `vault` | No `target`; matches every page in the configured vault |
| `page` | One exact canonical page URI |
| `prefix` | One canonical URI prefix; matching is path-segment aware |

`timeout` is optional, defaults to `10s`, and cannot exceed `60s`.

Command hooks use an exact argument vector without a shell. The event JSON is
provided on standard input:

```toml
[[hooks]]
name = "refresh-search"
kind = "command"
scope = "prefix"
target = "gnosis://knowledge/concepts"
timeout = "5s"
command = ["/usr/local/bin/refresh-search", "--source", "gnosis"]
```

Webhook hooks POST the same JSON bytes. HTTPS is required except for loopback
development URLs. `secret_env` names an environment variable containing an
HMAC-SHA256 key; gnosis sends the resulting
`X-Gnosis-Signature: sha256=<hex>` header without exposing the secret:

```toml
[[hooks]]
name = "notify-catalog"
kind = "webhook"
scope = "vault"
timeout = "10s"
url = "https://catalog.example.com/hooks/gnosis"
secret_env = "GNOSIS_CATALOG_HOOK_SECRET"
```

The bounded version 1 event envelope has this shape:

```json
{
  "version": 1,
  "id": "sha256:...",
  "vault": "knowledge",
  "uri": "gnosis://knowledge/concepts/example.md",
  "operation": "update",
  "prior_revision": "sha256:...",
  "new_revision": "sha256:...",
  "origin": {
    "vault": "knowledge",
    "kind": "local",
    "root": "/path/to/vault",
    "path": "/path/to/vault/concepts/example.md",
    "precedence": 0
  },
  "occurred_at": "2026-07-29T13:00:00Z",
  "knowledge_change": "sha256:..."
}
```

The event ID is deterministic from the vault, URI, new revision, and operation.
`prior_revision` and `knowledge_change` are omitted when unavailable. Webhooks
also receive `X-Gnosis-Event-Version` and `X-Gnosis-Event-ID` headers.

## `[[github]]`

Each GitHub evidence source belongs to the selected named vault and requires an
explicit repository allowlist entry:

```toml
[[github]]
repository = "owner/repository"
evidence_dir = "/var/lib/gnosis/github-evidence"
token_env = "GITHUB_TOKEN"
webhook_secret_env = "GITHUB_WEBHOOK_SECRET"
per_page = 100
max_pages = 20
max_body_bytes = 2097152
```

`evidence_dir` must be absolute. Raw records and cursor state use owner-only
filesystem permissions and never enter the Markdown vault. `token_env` is
required; `webhook_secret_env` is required only when serving GitHub webhooks.
The bounded defaults shown above apply when the three numeric fields are
omitted. Secrets are resolved from the environment for each operation and are
not persisted.

Select S3 instead of an evidence directory with the same explicit location
fields used by S3 vaults:

```toml
[[github]]
repository = "owner/repository"
evidence_backend = "s3"
s3_bucket = "example-knowledge"
s3_region = "us-east-1"
s3_prefix = "evidence/team"
token_env = "GITHUB_TOKEN"
```

The filesystem backend is the default. S3 evidence stores immutable records
and tombstones under deterministic keys and advances cursors conditionally
only after durable record writes. Configuration never contains AWS access
keys, secret keys, or session tokens.

## Semantic search environment

| Variable | Meaning |
|---|---|
| `GNOSIS_VECTOR_BACKEND` | Vector storage: `pgvector` (default) or `sqlite` |
| `GNOSIS_DATABASE_URL` | PostgreSQL connection string required by pgvector |
| `GNOSIS_SQLITE_VECTOR_PATH` | Optional absolute SQLite index path |
| `GNOSIS_EMBEDDING_URL` | OpenAI-compatible embeddings endpoint |
| `GNOSIS_EMBEDDING_MODEL` | Embedding model identifier |
| `GNOSIS_EMBEDDING_API_KEY` | Endpoint credential |

SQLite defaults to `<user-cache>/gnosis/<vault-name>/semantic.db`. These
variables configure `gnosis index knowledge` and vector search. Lexical search
does not need them.

## Memory environment

`GNOSIS_MEMORY_USER_ID` and `GNOSIS_MEMORY_AGENT_ID` are always required for
memory operations.

When `GNOSIS_MEMORY_API_KEY`, `GNOSIS_MEMORY_PROVIDER`, and
`GNOSIS_MEMORY_BASE_URL` are all absent, memory uses the effective writable
vault. If any is present, external configuration must be complete:

| Variable | Meaning |
|---|---|
| `GNOSIS_MEMORY_API_KEY` | Mem0 credential; also selects external memory |
| `GNOSIS_MEMORY_PROVIDER` | `hosted` or `self-hosted`; omitted means `hosted` |
| `GNOSIS_MEMORY_BASE_URL` | Optional hosted endpoint override; required for `self-hosted` |

Partial external configuration and request failures do not fall back to the
vault.

## Agent-run trace environment

| Variable | Meaning |
|---|---|
| `GNOSIS_TRACE_DIR` | Absolute owner-protected directory for append-only trace records |
| `GNOSIS_TRACE_AGENT_ID` | Fixed non-empty agent identity applied to trace capture, reads, and learning |

Both values are required for `record_trace`, `get_run_trace`, and
`propose_run_learning`. Trace records remain outside the curated vault.

## Reserved paths

`index.md` and `log.md` have generated structural roles. A root-level
`documentation/` directory is excluded from vault records, allowing product
documentation to coexist with knowledge pages.

Secrets belong in the process environment, not in TOML or Markdown.
