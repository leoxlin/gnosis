---
type: Concept Type
title: Gnosis Process
description: A discoverable, invocable workflow for vault or repository work.
path: gnosis/processes
---

# Gnosis Process

A **Gnosis Process** is a repeatable workflow that an agent can discover and invoke from the effective vault.

## Use this for

- Repeatable vault or repository work with explicit inputs, steps, and completion evidence.

Do not use it for one-off knowledge, settled choices, implementation handoffs, or runtime packaging.

## Minimum record

- A selection-focused `description` and at least one `## Use when` bullet.
- Unique, non-empty `## Knowledge inputs`, `## Process`, and `## Completion` sections.
- Self-contained instructions. Links and hard-coded knowledge URIs may target other `Gnosis Process` records or gnosis concept records; copy required rules inline and name dynamic runtime inputs without linking them.
- Optional `invocation`, `effects`, and typed `relationships` frontmatter.

## Schema

```yaml
---
type: Gnosis Process
title: <name>
description: <selection condition>
invocation: <model | explicit>
effects: [<read | vault-write | workspace-write | external>]
---

# <name>

## Use when
## Knowledge inputs
## Process
## Completion
```
