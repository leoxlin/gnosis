---
type: ConceptType
title: Decision
description: A durable, non-obvious choice that constrains future work.
path: decisions
---

# Decision

A **Decision** preserves a settled choice whose rationale cannot be recovered reliably from the result.

## Use this for

- Choices with durable constraints, plausible alternatives, or explicit supersession.

Do not use it for routine implementation details, status, or facts available from history.

## Minimum record

- `# Decision` and the essential `# Why`.
- Optional `# Constraints` and `supersedes` link.

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
