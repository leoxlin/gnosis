---
type: Concept Type
title: Repository Decision
description: Definition of the Repository Decision concept type — a durable record of a resolved choice, including the selected option, alternatives considered, and the reasons for the choice.
defines: Repository Decision
tags: [okf, ontology, decision, architecture, triage]
timestamp: 2026-07-09T04:08:43Z
---

# Repository Decision

A **Repository Decision** is a durable record of a resolved choice that affects a repository or service. It answers the question:

> What did we decide, why did we decide it, and what did we reject?

It captures the selected option, the alternatives that were considered, the trade-offs that mattered, and the context needed to revisit the choice later without rebuilding the conversation from scratch.

## When to use this type

Use `Repository Decision` for:

- Architectural or technology choices that constrain future work.
- Scope boundaries that exclude otherwise reasonable options.
- API contracts, data models, or integration patterns that are costly to change.
- Process choices that affect how agentic loops operate on the repository.

Do **not** use it for:

- Implementation instructions — use a [Repository Directive](repository-directive.md) for that.
- Status updates or meeting notes.
- OKRs, roadmaps, or milestones.
- Transient discussion that does not settle a question.

## Why this name

The word **decision** is chosen deliberately over related terms:

- **ADR** is a format, not a category, and is often heavier than needed.
- **RFC** implies a request for comment rather than a settled choice.
- **Proposal** suggests the question is still open.
- **Choice** is too informal and lacks the weight of a team-level commitment.

A Repository Decision is narrower: it is a settled choice whose record is meant to be read by future agents and maintainers.

## Relationship to other types

A [Repository Directive](repository-directive.md) may reference one or more `Repository Decision` records when the directive depends on a prior choice. Conversely, a decision may be created while triaging an issue and later be used by the directive that implements the chosen path.

## Suggested body structure

| Section | Purpose |
|---|---|
| `# Decision` | A single statement of the resolved choice. |
| `# Context` | The situation, constraint, or question that required a decision. |
| `# Alternatives considered` | Options that were evaluated and why each was rejected or accepted. |
| `# Trade-offs` | The costs, risks, or constraints accepted by the chosen option. |
| `# Consequences` | What this decision enables or prevents in the repository. |
| `# Related decisions` | Links to other `Repository Decision` records that this one builds on or supersedes. |

## Schema

```yaml
---
type: Repository Decision
title: <short name>
description: <one-line summary>
tags: [decision, <domain>]
timestamp: <ISO 8601 datetime>
---

# Decision

# Context

# Alternatives considered

# Trade-offs

# Consequences

# Related decisions
```
