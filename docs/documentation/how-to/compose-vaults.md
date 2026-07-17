# Compose vaults

Combine several vaults into one deterministic knowledge view.

## Import local vaults

    gnosis apply workspace --import /path/to/team/docs

Each `--import` appends one `[[vaults]]` entry to `gnosis.toml`, naming the vault after the path's basename. Repeat the flag to import several vaults. The composed view resolves in order: the local vault first, then imports in declared order, then the embedded core bundle — the first source wins for a given vault-relative path. Use `gnosis get vaults` to inspect the effective order.

## Address pages across vaults

Canonical URIs are `gnosis://<vault>/<path>`. The `_` authority (`gnosis://_/procedures/vault/query-vault.md`) matches any vault and is the portable way to reference shared records.

## GitHub wiki backend

    gnosis apply workspace --github-wiki owner/repo --name wiki

The wiki repository is cached as a local git working tree, pulled fast-forward on load, and committed and pushed after mutations. Treat it as a shared vault: coordinate writers, because push conflicts surface as errors.

## Rules

- Imports are read-mostly: `apply page` writes only to the local vault (or to a github-wiki backend, which publishes).
- Detect cycles with `gnosis validate vault` after changing imports.
