---
type: Concept Type
title: Gnosis Directive
description: An explicitly requested durable implementation handoff.
path: gnosis/directives
---

# Gnosis Directive

A **Gnosis Directive** is a bounded handoff for later automated or unattended execution.

## Use this for

- Explicitly requested work that needs durable scope and observable acceptance criteria.

Do not create one implicitly for ordinary implementation, task tracking, or completed work.

## Minimum record

- `status`, `# Goal`, `# Scope`, and `# Acceptance criteria`.
- Optional dependencies; an implementation plan when the handoff requires ordered steps.

## Schema

```yaml
---
type: Gnosis Directive
title: <name>
description: <result>
status: <open | blocked | done>
---

# Goal
# Scope
# Dependencies
# Implementation plan
# Acceptance criteria
```
