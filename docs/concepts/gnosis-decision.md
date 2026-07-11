---
type: Concept Type
title: Gnosis Decision
description: A durable, non-obvious choice that constrains future work.
---

# Gnosis Decision

A **Gnosis Decision** preserves a settled choice whose rationale cannot be recovered reliably from the result.

## Use this for

- Choices with durable constraints, plausible alternatives, or explicit supersession.

Do not use it for routine implementation details, status, or facts available from history.

## Minimum record

- `# Decision` and the essential `# Why`.
- Optional `# Constraints` and `supersedes` link.

## Schema

```yaml
---
type: Gnosis Decision
title: <name>
description: <effect>
supersedes: <optional decision link>
---

# Decision
# Why
# Constraints
```
