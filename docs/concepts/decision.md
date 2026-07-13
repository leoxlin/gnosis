---
type: ConceptType
title: Decision
description: A durable, non-obvious choice that constrains future work.
path: decisions
---

# Decision

A **Decision** preserves a settled choice whose rationale cannot be recovered reliably from the result.

By convention, the Decision records lives at `gnosis://<vault>/decisions/`.

## Use this for

- Choices with durable constraints, plausible alternatives, or explicit supersession.

Do not use it for routine implementation details, status, or facts available from history.

## Minimum record

- `# Decision` and the essential `# Why`.
- Optional `# Constraints` and `supersedes` link.

## Lifecycle

- Identity is the settled non-obvious choice, not merely its title. Query for an existing choice before creating or selecting a Decision.
- Creation resolves the material alternatives, rationale, constraints, and every author-owned choice, obtains explicit author confirmation, and rejects duplicate identity.
- Apply non-semantic corrections in place while preserving unknown metadata. A changed choice, rationale, or constraint creates a new author-confirmed Decision whose `supersedes` field links the prior record; preserve the prior record unchanged.
- Prefer correction or supersession after a Decision has governed work. Delete only a confirmed local duplicate or invalid record after tracing inbound links and supersession history, obtaining explicit approval for that deletion, and repairing or intentionally removing every inbound reference. Report imported or bundled records to their owning vault.

## Schema

```yaml
---
type: Decision
title: <name>
description: <effect>
supersedes: <optional decision link>
---

# Decision
# Why
# Constraints
```
