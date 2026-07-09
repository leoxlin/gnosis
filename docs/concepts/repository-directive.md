---
type: Concept Type
title: Repository Directive
description: Definition of the Repository Directive concept type — a minimal, actionable handoff document that an agentic loop consumes to implement a change in a project.
defines: Repository Directive
tags: [okf, ontology, directive, implementation, agent]
timestamp: 2026-07-09T04:08:43Z
---

# Repository Directive

A **Repository Directive** is a minimal, actionable handoff document that an agentic loop consumes to implement a change in a project. It answers the question:

> What concrete change should an agent implement, and how will we know it is done?

It bridges triage and implementation. A directive is created after an issue has been understood and any necessary [Repository Decision](repository-decision.md) records have been made. A future agent loop reads the directive, implements the change, and a merging pull request updates the directive to mark it done and creates a [Repository Delta](repository-delta.md).

## When to use this type

Use `Repository Directive` for:

- Changes spawned by an issue that need structured implementation instructions.
- Work that must be handed from a triage agent to an implementation agent.
- Scope that is too large or risky to implement in an unstructured conversation.

Do **not** use it for:

- Open questions or unsettled choices — use a [Repository Decision](repository-decision.md) for those.
- High-level goals or purpose — use a `Repository Purpose` for those.
- Roadmaps, schedules, or long-term plans.

## Why this name

The word **directive** is chosen deliberately to avoid the baggage of related terms:

- **Plan** implies scheduling, sequencing, and ongoing maintenance.
- **Design** implies upfront specification of structure before implementation.
- **Spec** suggests a heavy, formal document aimed at human review.
- **Ticket** is an issue-tracker artifact, not an ontology concept, and usually describes a problem rather than the work to resolve it.

A Repository Directive is narrower: it is the minimal handoff that an agent needs to act, paired with clear criteria for completion.

## Lifecycle

1. **Trigger**: an issue, bug report, or feature request is created.
2. **Triage**: an agent works with the engineering team to understand the problem, record any [Repository Decision](repository-decision.md) records, and identify dependencies.
3. **Directive creation**: a `Repository Directive` is written with a clear goal, scope, acceptance criteria, and dependency list.
4. **Implementation**: a future agent loop picks up the directive and implements the change.
5. **Completion**: a pull request that lands the change updates the directive file to mark it done and creates at least one [Repository Delta](repository-delta.md).

## Suggested body structure

| Section | Purpose |
|---|---|
| `# Goal` | The concrete change the directive exists to produce. |
| `# Trigger` | Link to the issue or conversation that created the need. |
| `# Scope` | What is in scope and what is explicitly out of scope. |
| `# Dependencies` | Blockers, prerequisites, or decisions that must be resolved first, with links to [Repository Decision](repository-decision.md) records or other directives. |
| `# Acceptance criteria` | Observable conditions that must be true for the directive to be considered complete. |
| `# Implementation notes` | Optional hints, preferred patterns, or pitfalls for the implementing agent. |
| `# Completion` | Set by the implementing PR — links to the change record and pull request, and marks the directive done. |

## Schema

```yaml
---
type: Repository Directive
title: <short name>
description: <one-line summary>
tags: [directive, <domain>]
timestamp: <ISO 8601 datetime>
status: <open | blocked | in-progress | done>
---

# Goal

# Trigger

# Scope

# Dependencies

# Acceptance criteria

# Implementation notes

# Completion
```
