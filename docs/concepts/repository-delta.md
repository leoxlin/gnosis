---
type: Concept Type
title: Repository Delta
description: Definition of the Repository Delta concept type — a durable trace of a change that was implemented from one or more Repository Directives.
defines: Repository Delta
tags: [okf, ontology, delta, implementation, agent, repository-delta]
timestamp: 2026-07-09T04:08:43Z
---

# Repository Delta

A **Repository Delta** is a durable trace of a change that was implemented from one or more [Repository Directive](repository-directive.md) records. It answers the question:

> What change was actually made, which directives did it fulfill, and how do we know it is correct?

It is created by the pull request that implements the directive. The same pull request updates the fulfilled directive to mark it done, so the directive remains as historical context and the delta becomes the permanent artifact of what happened.

## When to use this type

Use `Repository Delta` for:

- Pull requests that implement one or more directives.
- Any completed change whose provenance should remain visible to future agents.
- Work where the actual implementation deviated from the original directive and the deviation needs to be recorded.

Do **not** use it for:

- Ongoing work — use a [Repository Directive](repository-directive.md) for that.
- Open questions or settled choices — use a [Repository Decision](repository-decision.md) for those.
- High-level goals — use a `Repository Purpose` for those.

## Why this name

The word **delta** is chosen deliberately over related terms:

- **Plan** or **design** describe intent, not what actually happened.
- **Ticket** is an issue-tracker artifact, not a record of completed work.
- **Log** is too informal and chronological.
- **Summary** lacks the requirement to link back to directives and verification.
- **Change record** is accurate but verbose; **delta** captures the same idea more concisely.

A Repository Delta is narrower: it is the permanent trace left after a directive has been implemented.

## Lifecycle

1. A [Repository Directive](repository-directive.md) is created during triage.
2. An agent implements the directive and opens a pull request.
3. The pull request updates the directive file to mark it done and creates a `Repository Delta` describing what was done.
4. The directive remains as historical context; the repository delta becomes the durable history of the change.

A valid implementation PR in this ontology MUST:

- Mark at least one `Repository Directive` as done.
- Create at least one `Repository Delta` file.

## Suggested body structure

| Section | Purpose |
|---|---|
| `# Fulfilled directives` | Links to the directive or directives that this change implements. |
| `# Change summary` | Concise description of what was actually changed. |
| `# Pull request` | Link to the implementing pull request and relevant commit references. |
| `# Verification` | Evidence that the acceptance criteria were met — tests, checks, links, or observations. |
| `# Deviations` | Any departures from the directive and the reason for each. Omit if there were none. |
| `# Related decisions` | Links to [Repository Decision](repository-decision.md) records that informed the change. |

## Schema

```yaml
---
type: Repository Delta
title: <short name>
description: <one-line summary>
tags: [delta, <domain>]
timestamp: <ISO 8601 datetime>
status: <completed | superseded | reverted>
---

# Fulfilled directives

# Change summary

# Pull request

# Verification

# Deviations

# Related decisions
```
