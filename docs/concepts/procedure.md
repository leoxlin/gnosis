---
type: ConceptType
title: Procedure
description: A discoverable, invocable workflow for vault or repository work.
path: procedures
---

# Procedure

A **Procedure** is a repeatable workflow that an agent can discover and invoke from the effective vault.

## Use this for

- Repeatable vault or repository work with explicit inputs, steps, and completion evidence.

Do not use it for one-off knowledge, settled choices, implementation handoffs, or runtime packaging.

## Minimum record

- A selection-focused `description` and at least one `use_when` frontmatter value.
- A `tags` frontmatter value identifying its process family. Discovery returns only records whose tags intersect `[gnosis].processes`.
- Unique, non-empty `## Knowledge inputs`, `## Process`, and `## Completion` sections.
- Self-contained instructions. Links and hard-coded knowledge URIs may target other `Procedure` records or gnosis concept records; copy required rules inline and name dynamic runtime inputs without linking them.
- Optional `invocation`, `effects`, and typed `relationships` frontmatter. An `explicit` invocation is omitted from discovery and is invoked only by another active process.

## Schema

```yaml
---
type: Procedure
title: <name>
description: <selection condition>
tags: [<process-family>]
invocation: <model | explicit>
effects: [<read | vault-write | workspace-write | external>]
use_when:
  - <selection condition>
---

# <name>

## Knowledge inputs
## Process
## Completion
```
