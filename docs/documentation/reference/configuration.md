# Configuration reference

The primary vault configuration resolves in this order: `gnosis.local.toml`, `gnosis.toml`, then `~/.config/gnosis.toml`. Repository settings define the primary vault when present. When that repository is also registered in the user configuration, its other `[[vaults]]` remain available after repository-declared imports. Inside a git work tree with no repository or user configuration, gnosis defaults to a vault named `local` rooted at `docs/` with strict relative links and index/log disabled.

## `[vault]`

| Field | Default | Meaning |
|---|---|---|
| `vault_name` | `local` (in git) | Vault identity; used in URIs. |
| `vault_root` | `docs` (in git) | Directory holding the knowledge pages. |
| `backend` | none | Storage backend; `github-wiki` clones the repo's wiki. |
| `repository` | none | `owner/repo` for the GitHub wiki backend. |
| `link_format` | `relative` | Preferred body-link style: `relative` or `absolute`. |
| `link_format_strict` | `true` in git | Promote link-format violations from warnings to errors. |
| `vault_index` | `false` | Require and generate per-directory `index.md` files. |
| `vault_log` | `false` | Require a root `log.md`; procedures append entries. |

## `[[vaults]]`

Each block imports one vault: `vault_name`, `vault_root`, optional `backend`/`repository`. The composed view is deterministic: local first, imports in declared order, the embedded core bundle last; first source wins per vault-relative path. Cycles are validation errors.

## Environment

`GNOSIS_DATABASE_URL`, `GNOSIS_EMBEDDING_URL`, `GNOSIS_EMBEDDING_MODEL`, `GNOSIS_EMBEDDING_API_KEY` configure the optional vector backend.

Memory operations always require `GNOSIS_MEMORY_USER_ID` and `GNOSIS_MEMORY_AGENT_ID`. With `GNOSIS_MEMORY_API_KEY`, `GNOSIS_MEMORY_PROVIDER`, and `GNOSIS_MEMORY_BASE_URL` all absent, gnosis uses the effective writable vault. If any external-specific variable is set, configuration must be complete and gnosis uses Mem0 exclusively:

- An API key with no provider selects `hosted`; `GNOSIS_MEMORY_BASE_URL` can override its endpoint.
- `GNOSIS_MEMORY_PROVIDER=self-hosted` requires both an API key and `GNOSIS_MEMORY_BASE_URL`.
- Partial configuration and external request failures never fall back to or mutate the vault.

Secrets never live in `gnosis.toml`.

## Reserved names

`index.md` and `log.md` are reserved files; a root-level `documentation/` directory is exempt from the vault entirely.
