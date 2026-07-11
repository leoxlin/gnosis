---
type: Concept Type
title: Repository Process
description: A repeatable repository-owned workflow that an agent can discover, invoke, and follow.
---

# Repository Process

A **Repository Process** is a repeatable repository-owned workflow that an agent can discover, invoke, and follow. It turns durable repository knowledge into an executable contract without making runtime skill packaging the source of truth.

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

The four sections are required, unique, and non-empty. `description` and the bullets under `## Use when` are the compact discovery surface. Invocation returns the complete sections from one exact process revision; gnosis loads the contract, while the agent performs the actions under current user and repository instructions.

Optional frontmatter makes execution intent and graph semantics machine-readable:

- `invocation` — `model` (the default) when an agent may select the process, or `explicit` when the author must name or request it.
- `effects` — zero or more of `read`, `vault-write`, `workspace-write`, and `external`; these declare possible effects rather than granting authority.
- `relationships` — directed typed links, each with a non-empty `type` and Markdown `target`. Body links remain available as generic `links_to` edges.

Only records whose exact effective type is `Vault Process` or `Repository Process` are invocable. gnosis resolves local, imported, and bundled records with normal vault precedence and returns their stable URI, origin, and content revision so an agent can bind its work to the selected source.

## Schema

```yaml
---
type: Repository Process
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
