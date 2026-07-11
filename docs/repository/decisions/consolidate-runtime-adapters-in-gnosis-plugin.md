---
type: Repository Decision
title: Consolidate runtime adapters in the `gnosis` plugin
description: Keep workflows as canonical vault knowledge and expose them through two thin `gnosis` gateways.
---

# Decision

Keep repeatable vault and repository workflows as canonical process records in the configured vault. Publish one `gnosis` plugin with exactly two runtime gateways:

- `using-gnosis-vault` selects and follows [Vault Process](../../concepts/vault-process.md) records.
- `using-gnosis-forge` selects and follows [Repository Process](../../concepts/repository-process.md) records.

Each gateway uses `gnosis concepts` to discover records and `gnosis read` to read the applicable concept definition and selected process. A process record, rather than plugin packaging, is the source of truth for its selection conditions, knowledge inputs, ordered work, and completion gate.

# Why

Separate vault and repository plugins, along with task-specific runtime adapters, exposed packaging structure instead of the knowledge model that governs the work. They also made the installed skill surface grow with every process despite process records already providing complete workflows.

Two gateways keep runtime discovery portable while making the type boundary explicit: vault work selects Vault Processes and repository work selects Repository Processes.

Rejected alternatives:

- **Keep separate vault and repository plugins** — rejected because installation and discovery should begin with one `gnosis` capability, not its internal knowledge domains.
- **Keep one packaged skill per process** — rejected because it duplicates canonical process knowledge in runtime packaging.
- **Link gateway skills directly to repository files** — rejected because the configured vault, including its imports and bundled documentation, is the authoritative runtime view.

# Constraints

- Marketplaces and local skill links expose only the `gnosis` plugin and its two gateway skills.
- Gateway skills retrieve their concept definition and process records with `gnosis concepts` and `gnosis read`; they do not copy process instructions.
- Process records retain their identities, selection conditions, and completion gates. Runtime gateways respect the configured vault, including its imports.
