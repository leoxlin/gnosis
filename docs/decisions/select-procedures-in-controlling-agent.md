---
type: Decision
title: Select procedures in the controlling agent
description: Keep canonical workflows in the vault and let each scoped gnosis gateway select and execute them directly.
supersedes: consolidate-runtime-adapters-in-gnosis-plugin.md
---

# Decision

Keep repeatable vault and repository workflows as canonical Procedure records in the configured vault. Publish one gnosis plugin with two scoped gateway skills: `using-gnosis` for vault work and `using-gnosis-for-development` for repository development work.

Each gateway discovers its procedure family once, directly selects the smallest applicable dependency-ordered set, reads each exact selected URI, and follows the contracts in the controlling agent. Do not delegate procedure selection to a selector subagent.

# Why

Canonical Procedure records keep workflows portable and precedence-aware without copying their instructions into runtime packaging. Direct selection preserves the author request, repository instructions, and execution context in one agent, while separate vault and development gateways keep discovery scoped to the work being requested.

# Constraints

- Gateway skills retrieve Procedure records through the gnosis CLI; Procedure records remain the workflow source of truth.
- Only exact Procedure records are invocable.
- Selection respects effective-page precedence and uses the exact URI returned by discovery.
- The controlling agent executes every selected contract through its completion gate.

# Supersedes

- [Consolidate runtime adapters in the `gnosis` plugin](consolidate-runtime-adapters-in-gnosis-plugin.md)
