---
type: Concept Type
title: Repository Purpose
description: Definition of the Repository Purpose concept type — a top-level purpose for a repository or service, decomposable into component purposes.
defines: Repository Purpose
tags: [okf, ontology, purpose, repository, service, goal, repository-purpose]
timestamp: 2026-07-09T03:43:13Z
---

# Repository Purpose

A **Repository Purpose** is the single governing answer to the question:

> What is this repo or service *for*?

It captures the intended outcome the artifact exists to produce. It is not a task list, a roadmap, or a business objective. It is the enduring reason the repository or service should exist and continue to evolve.

## When to use this type

Use `Repository Purpose` for:

- The whole repository or service.
- Each major component, sub-directory, or module whose purpose is worth stating independently.

Do **not** use it for:

- OKR objectives or key results.
- Sprint milestones, project plans, or delivery schedules.
- Personal, team, or organizational goals.
- Tactical tasks or feature requests.

## Why this name

The word **purpose** is chosen deliberately to avoid collision with other goal-related concepts:

- **Goal** is too generic and is used by OKRs, personal productivity systems, and optimization problems.
- **Objective** is the standard OKR term for a qualitative aim.
- **Mission** usually applies to an organization or product line, not a single repo.
- **Charter** reads as a project-management document rather than an ontological category.

A Repository Purpose is narrower: it describes what a specific code or knowledge artifact is meant to accomplish.

## Decomposition

A Repository Purpose may be decomposed into **component purposes**. Each component purpose is itself a `Repository Purpose` concept tied to a component, package, or subdirectory. Decomposition stops when a component's reason for existing is trivial or self-evident.

Links from a parent purpose to its component purposes SHOULD use absolute bundle-relative paths. The surrounding prose SHOULD make the *part-of* relationship explicit.

## Suggested body structure

| Section | Purpose |
|---|---|
| `# Purpose` | A single, durable statement of what the repo or service exists to achieve. |
| `# Sub-purposes` | Links to component purposes, one per major component or subdirectory. |
| `# Boundaries` | What the artifact explicitly does not try to be or do. |
| `# Relationship to other goal types` | How this purpose relates to OKRs, milestones, user stories, etc. |

## Schema

```yaml
---
type: Repository Purpose
title: <short name>
description: <one-line summary>
tags: [<artifact-type>, <domain>]
timestamp: <ISO 8601 datetime>
---

# Purpose

# Sub-purposes

# Boundaries
```
