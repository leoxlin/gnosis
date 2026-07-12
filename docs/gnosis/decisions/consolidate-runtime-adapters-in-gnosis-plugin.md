---
type: GnosisDecision
title: Consolidate runtime adapters in the `gnosis` plugin
description: Keep workflows as canonical vault knowledge and expose them through one thin `gnosis` gateway.
---

# Decision

Keep repeatable vault and repository workflows as canonical process records in the configured vault. Publish one `gnosis` plugin with one runtime gateway:

- `using-gnosis` delegates selection of [Gnosis Process](../../concepts/gnosis-process.md) records to a fresh read-only subagent, then follows the selected records in the controlling agent.

The gateway lists exact `GnosisProcess` records with `gnosis concepts -type 'GnosisProcess'`. A fresh selector chooses the smallest applicable chain from that list, reads each selected exact URI with `gnosis read '<gnosis URI>'`, and returns the complete ordered commands to the controlling agent. A process record, rather than plugin packaging or a copied prompt, is the source of truth for its selection conditions, knowledge inputs, ordered work, and completion gate.

# Why

Separate vault and repository plugins, along with task-specific runtime adapters, exposed packaging structure instead of the knowledge model that governs the work. They also made the installed skill surface grow with every process despite process records already providing complete workflows.

One gateway keeps runtime discovery portable while exact-type discovery preserves the executable process boundary. Delegating selection gives every task a fresh, read-only process choice without transferring execution authority or context to the selector.

Rejected alternatives:

- **Keep separate vault and repository plugins** — rejected because installation and discovery should begin with one `gnosis` capability, not its internal knowledge domains.
- **Keep one packaged skill per process** — rejected because it duplicates canonical process knowledge in runtime packaging.
- **Link gateway skills directly to repository files** — rejected because the configured vault, including its imports and bundled documentation, is the authoritative runtime view.

# Constraints

- Marketplaces and local skill links expose only the `gnosis` plugin and its `using-gnosis` gateway skill.
- The gateway retrieves process records through the CLI; it does not copy process instructions.
- A fresh read-only subagent selects the smallest applicable process chain for every task, reads each selected exact URI, and returns a clearly headed full ordered process list. The controlling agent reads and executes the selected exact revisions.
- Process records retain their identities, invocation modes, possible effects, selection conditions, and completion gates. Runtime gateways respect effective-page precedence across the configured vault and expose every selected record's origin and revision.
- Only exact `Gnosis Process` records are invocable. Other knowledge remains readable and queryable but cannot become executable merely through wording or tags.
