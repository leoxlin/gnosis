---
type: Concept Type
title: Vault Process
description: A repeatable vault-owned workflow that an agent can discover, invoke, and follow.
---

# Vault Process

A **Vault Process** is a repeatable vault-owned workflow that an agent can discover, invoke, and follow. It turns durable vault knowledge and configuration into an executable contract without making runtime skill packaging the source of truth.

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

The four sections are required, unique, and non-empty. `description` and the bullets under `## Use when` are the compact discovery surface. Invocation returns the complete sections from one exact process revision; gnosis loads the contract, while the agent performs the actions under current user and repository instructions.

Optional frontmatter makes execution intent and graph semantics machine-readable:

- `invocation` — `model` (the default) when an agent may select the process, or `explicit` when the author must name or request it.
- `effects` — zero or more of `read`, `vault-write`, `workspace-write`, and `external`; these declare possible effects rather than granting authority.
- `relationships` — directed typed links, each with a non-empty `type` and Markdown `target`. Body links remain available as generic `links_to` edges.

Only records whose exact effective type is `Vault Process` or `Repository Process` are invocable. gnosis resolves local, imported, and bundled records with normal vault precedence and returns their stable URI, origin, and content revision so an agent can bind its work to the selected source.

## Schema

```yaml
---
type: Vault Process
title: <process name>
description: <one-line selection condition>
invocation: <model | explicit>
effects: [<read | vault-write | workspace-write | external>]
relationships:
  - type: <relationship>
    target: <relative Markdown path>
---

# <Process name>

## Use when

## Knowledge inputs

## Process

## Completion
```
