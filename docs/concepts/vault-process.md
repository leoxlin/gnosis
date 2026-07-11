---
type: Concept Type
title: Vault Process
description: A repeatable vault-owned workflow that a skill can select and follow.
---

# Vault Process

A **Vault Process** is a repeatable vault-owned workflow that a skill can select and follow. It turns durable vault knowledge and configuration into ordered action without making runtime skill packaging the source of truth.

Each process answers these questions:

- When should this workflow govern work in a vault?
- Which configuration, agent rules, concept records, and source facts must ground it?
- Which ordered actions, gates, and author handoffs make the workflow reliable?
- What evidence or state marks the workflow complete?

## Use this for

- Repeatable ingestion, query, maintenance, and ontology workflows.
- Vault-specific adaptations of reusable knowledge-management methods.
- Processes that skills should discover and apply from vault knowledge.

Do not use it for one-off knowledge, a concept type, a source, or runtime skill packaging; use the appropriate concept record or packaged skill instead.

## Minimum record

- `## Use when` — observable conditions that select the process.
- `## Knowledge inputs` — only the vault records, configuration, and source facts needed for this process.
- `## Process` — ordered actions, gates, and required handoffs.
- `## Completion` — observable evidence or state that ends the process.

## Schema

```yaml
---
type: Vault Process
title: <process name>
description: <one-line selection condition>
---

# <Process name>

## Use when

## Knowledge inputs

## Process

## Completion
```
