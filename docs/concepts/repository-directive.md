---
type: Concept Type
title: Repository Directive
description: An explicitly requested or explicitly planned implementation handoff for an automated agent.
---

# Repository Directive

A **Repository Directive** is a durable handoff for automated or unattended agent execution. It exists when an author explicitly requests one or explicitly selects `writing-directives-and-decisions` after approving a design.

## Use this for

- Work an automated agent should pick up in a later run.
- Risky or multi-step execution requiring durable scope and acceptance criteria.
- Handoffs that must remain usable without replaying triage.
- An executable implementation plan produced by `writing-directives-and-decisions`.

Do not create one implicitly from ordinary implementation, task tracking, or completed work. Selecting `writing-directives-and-decisions` is an explicit directive request.

## Minimum record

- `# Goal` — the concrete result to produce.
- `# Scope` — what is included and excluded.
- `# Acceptance criteria` — observable completion conditions.
- `# Dependencies` — optional blockers or prior decisions.
- `# Implementation plan` — required when `writing-directives-and-decisions` creates the directive; optional for other sufficiently bounded handoffs.
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

# Implementation plan

# Acceptance criteria
```
