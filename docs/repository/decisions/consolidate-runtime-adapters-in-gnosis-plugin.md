---
type: Repository Decision
title: Consolidate runtime adapters in the `gnosis` plugin
description: Package two gateway skills that select canonical Vault Process and Repository Process records through the gnosis CLI.
supersedes: name-knowledge-driven-development-bundle-gnosis-forge.md
---

# Decision

Publish one `gnosis` plugin with exactly two runtime skills:

- `using-gnosis-vault` selects and follows [Vault Process](../../concepts/vault-process.md) records.
- `using-gnosis-forge` selects and follows [Repository Process](../../concepts/repository-process.md) records.

Each gateway uses `gnosis concepts` to discover records and `gnosis read` to read the applicable concept definition and selected process. The records in a configured vault remain the source of truth.

# Why

Separate `gnosis-vault` and `gnosis-forge` plugins, along with task-specific runtime adapters, exposed packaging structure instead of the knowledge model that governs the work. They also made the installed skill surface grow with every process despite the process records already providing selection conditions and complete workflows.

Two gateways keep runtime discovery portable while making the type boundary explicit: vault work selects Vault Processes and repository work selects Repository Processes.

Rejected alternatives:

- **Keep separate vault and forge plugins** — rejected because installation and discovery should begin with one gnosis capability, not its internal knowledge domains.
- **Keep one packaged skill per process** — rejected because it duplicates the vault's canonical process surface in runtime packaging.
- **Link gateway skills directly to repository files** — rejected because the configured vault, including its imports and bundled documentation, is the authoritative runtime view.

# Constraints

- Marketplaces and local skill links expose only the `gnosis` plugin and its two gateway skills.
- Gateway skills retrieve their concept definition and process records with `gnosis concepts` and `gnosis read`; they do not copy process instructions.
- Vault and repository process records retain their existing identities, selection conditions, and completion gates.

# Related decisions

- [Make vault processes knowledge](make-vault-processes-knowledge.md)
- [Make repository processes knowledge](make-repository-processes-knowledge.md)
- [Name the knowledge-driven development bundle `gnosis-forge`](name-knowledge-driven-development-bundle-gnosis-forge.md)
