---
name: gnosis-vault
description: Use when working on the Gnosis vault, OKF-compatible markdown notes, Gnosis templates, or the Go validator/scaffolder in this repository.
---

# Gnosis Vault

Use this skill when creating, editing, or validating Gnosis vault content.

## Workflow

1. Read the vault's `AGENTS.md` and `docs/gnosis/agent-context/coordination.md` if they exist.
2. Keep every markdown note with parseable YAML frontmatter and a non-empty `type`.
3. Prefer one concept per file and absolute root-relative markdown links.
4. Record material handoff context under `docs/gnosis/agent-context/` in the vault.
5. Run the Go validator before handoff:

```bash
go run ./cmd/gnosis validate -vault <vault-path>
```

## Repo Tooling

- CLI entrypoint: `cmd/gnosis/main.go`
- Vault package: `internal/vault`
- Default vault path: current directory (use `-vault <path>` to target another vault)

## Subagent Coordination

Assign subagents disjoint write scopes. Shared files such as `<vault-path>/index.md` and `<vault-path>/log.md` should be integrated by the coordinator unless explicitly assigned.
