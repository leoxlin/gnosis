# Configuration reference

## Resolution

gnosis resolves configuration in this order:

1. `gnosis.local.toml`
2. `gnosis.toml`
3. the user configuration at `~/.config/gnosis.toml`

Repository configuration defines the primary vault when present. User
configuration can contribute additional registered vaults. In a git work tree
with no configuration, gnosis uses an implicit vault named `local`, rooted at
`docs/`, with strict relative links and index/log generation disabled.

Outside a git work tree, the default root is the current directory. Generated
indexes and the log are enabled.

## `[vault]`

| Field | Default | Meaning |
|---|---|---|
| `vault_name` | `local` in an implicit repository vault | Canonical URI authority |
| `vault_root` | `docs` in an implicit repository vault | Knowledge directory |
| `backend` | empty | Primary storage backend; supported value: `github-wiki` |
| `repository` | empty | GitHub `owner/repository` used by that backend |
| `link_format` | `relative` | Preferred internal link form: `relative` or `absolute` |
| `link_format_strict` | `true` in an implicit repository vault | Treat link-style violations as errors |
| `vault_index` | `false` in an implicit repository vault | Require and generate directory `index.md` files |
| `vault_log` | `false` in an implicit repository vault | Require a root `log.md` |

Example standalone vault:

```toml
[vault]
vault_name = "knowledge"
vault_root = "."
link_format = "relative"
link_format_strict = false
vault_index = true
vault_log = true
```

## `[[vaults]]`

Each imported vault has a name and local root:

```toml
[[vaults]]
vault_name = "team"
vault_root = "../team-vault"
```

The effective precedence is the primary vault, declared imports in order, then
the embedded core bundle. The first source for a vault-relative path wins.
Import cycles are invalid.

## Semantic search environment

| Variable | Meaning |
|---|---|
| `GNOSIS_DATABASE_URL` | PostgreSQL connection string |
| `GNOSIS_EMBEDDING_URL` | OpenAI-compatible embeddings endpoint |
| `GNOSIS_EMBEDDING_MODEL` | Embedding model identifier |
| `GNOSIS_EMBEDDING_API_KEY` | Endpoint credential |

These variables configure `gnosis index knowledge` and vector search. Lexical
search does not need them.

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

## Reserved paths

`index.md` and `log.md` have generated structural roles. A root-level
`documentation/` directory is excluded from vault records, allowing product
documentation to coexist with knowledge pages.

Secrets belong in the process environment, not in TOML or Markdown.
