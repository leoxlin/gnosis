---
type: Concept Type
title: Repository Process
description: A repeatable repository-owned workflow that a skill can select and follow.
---

# Repository Process

A **Repository Process** is a repeatable repository-owned workflow that a skill can select and follow. It turns durable repository knowledge into ordered action without making runtime skill packaging the source of truth.

Each process answers these questions:

- When should this workflow govern repository work?
- Which purpose, decisions, directives, concepts, and implementation facts must ground it?
- Which ordered actions, gates, and handoffs make the workflow reliable?
- What evidence or state marks the workflow complete?

## Use this for

- Repeatable planning, implementation, debugging, review, and delivery workflows.
- Repository-specific adaptations of reusable engineering methods.
- Processes that skills should discover and apply from repository knowledge.

Do not use it for one-off work, settled choices, repository purpose, implementation handoffs, or runtime skill packaging; use a Repository Directive, Repository Decision, Repository Purpose, or packaged skill respectively.

## Minimum record

- `## Use when` — observable conditions that select the process.
- `## Knowledge inputs` — only the repository records and implementation facts needed for this process.
- `## Process` — ordered actions, gates, and required handoffs.
- `## Completion` — observable evidence or state that ends the process.

## Schema

```yaml
---
type: Repository Process
title: <process name>
description: <one-line selection condition>
---

# <Process name>

## Use when

## Knowledge inputs

## Process

## Completion
```
