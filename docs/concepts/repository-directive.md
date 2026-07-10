---
type: Concept Type
title: Repository Directive
description: An explicitly requested implementation handoff for an automated agent.
---

# Repository Directive

A **Repository Directive** is a durable handoff for automated or unattended agent execution. It exists only when an author explicitly requests one.

## Use this for

- Work an automated agent should pick up in a later run.
- Risky or multi-step execution requiring durable scope and acceptance criteria.
- Handoffs that must remain usable without replaying triage.

Do not create one implicitly for ordinary planning, implementation, task tracking, or completed work.

## Minimum record

- `# Goal` — the concrete result to produce.
- `# Scope` — what is included and excluded.
- `# Acceptance criteria` — observable completion conditions.
- `# Dependencies` — optional blockers or prior decisions.
- `status` — whether the handoff is open, blocked, or done.

## Schema

```yaml
---
type: Repository Directive
title: <short name>
description: <one-line result>
status: <open | blocked | done>
---

# Goal

# Scope

# Dependencies

# Acceptance criteria
```
