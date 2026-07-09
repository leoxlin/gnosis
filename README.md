# Gnosis

Gnosis is a Go toolkit for an OKF-compatible LLM wiki.

## Backends

* [Obsidian](https://obsidian.md/) — local markdown vault workflow.

## CLI

Set up a new vault:

```bash
go run ./cmd/gnosis setup -vault ./my-vault
```

Validate a vault:

```bash
go run ./cmd/gnosis validate -vault ./my-vault
```

Repair the base vault shape without overwriting existing files:

```bash
go run ./cmd/gnosis scaffold -vault ./my-vault
```

## Agent Skill

Repo-local skill instructions live in [skills/gnosis-vault/SKILL.md](skills/gnosis-vault/SKILL.md).
