---
type: Repository Decision
title: Consolidate runtime adapters in the `gnosis` plugin
description: Keep workflows as canonical vault knowledge and expose them through two thin `gnosis` gateways.
---

# Decision

Keep repeatable vault and repository workflows as canonical process records in the configured vault. Publish one `gnosis` plugin with exactly two runtime gateways:

- `using-gnosis-vault` selects and follows [Vault Process](../../concepts/vault-process.md) records.
- `using-gnosis-forge` selects and follows [Repository Process](../../concepts/repository-process.md) records.

Each gateway prefers the gnosis MCP contract to discover and invoke processes, with `gnosis process discover` and `gnosis process invoke` as equivalent CLI fallbacks. Discovery exposes compact selection metadata; invocation loads one exact process revision. A process record, rather than plugin packaging or a copied prompt, is the source of truth for its selection conditions, knowledge inputs, ordered work, and completion gate.

# Why

Separate vault and repository plugins, along with task-specific runtime adapters, exposed packaging structure instead of the knowledge model that governs the work. They also made the installed skill surface grow with every process despite process records already providing complete workflows.

Two gateways keep runtime discovery portable while making the type boundary explicit: vault work selects Vault Processes and repository work selects Repository Processes.

Rejected alternatives:

- **Keep separate vault and repository plugins** — rejected because installation and discovery should begin with one `gnosis` capability, not its internal knowledge domains.
- **Keep one packaged skill per process** — rejected because it duplicates canonical process knowledge in runtime packaging.
- **Link gateway skills directly to repository files** — rejected because the configured vault, including its imports and bundled documentation, is the authoritative runtime view.

# Constraints

- Marketplaces and local skill links expose only the `gnosis` plugin and its two gateway skills.
- Gateway skills retrieve process records through the MCP or CLI agent contract; they do not copy process instructions.
- Process records retain their identities, invocation modes, possible effects, selection conditions, and completion gates. Runtime gateways respect effective-page precedence across the configured vault and expose every selected record's origin and revision.
- Only exact `Vault Process` and `Repository Process` records are invocable. Other knowledge remains readable and queryable but cannot become executable merely through wording or tags.
