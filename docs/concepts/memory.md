---
type: ConceptType
title: Memory
description: A scoped, self-contained agent memory of a durable fact, preference, or observation.
path: memories
---

# Memory

A **Memory** preserves one self-contained durable fact, preference, or observation under an explicit scope, written only through the remember procedure and read through the recall procedure.

By convention, Memory records live at `gnosis://<vault>/memories/`.

## Use this for

- User preferences and persona facts (`scope: user`), agent capabilities and learned limitations (`scope: agent`), and session- or run-scoped durable observations (`scope: session | run`).

Do not use it for conversation transcripts, working state, or knowledge with its own type: facts (Concept), lessons (Reflection), rules (Policy), episodes (Event).

## Minimum record

- `scope`, `observed_at`, and `hash` frontmatter, plus `# Memory` with one self-contained statement using absolute dates and verbatim proper nouns.
- Optional `actor`, `source`, and `entities` (named entities, used for retrieval boosts).

## Lifecycle

- Creation, update, and archival go through [remember](../procedures/vault/remember.md), which reconciles each candidate against the nearest existing memories as ADD, UPDATE, DELETE, or NONE; retrieval goes through [recall](../procedures/vault/recall.md).
- `status` is `active` while current and `archived` when superseded or deleted; archived memories are retained for audit and negative knowledge, never silently removed.
- `hash` is the SHA-256 hex of the `# Memory` statement text; exact duplicates are never written.
- Delete only through the remember procedure's DELETE operation, which archives; physical removal requires explicit author approval after tracing inbound links.

## Schema

```yaml
---
type: Memory
title: <short label>
description: <one-line summary>
scope: <user | agent | session | run>
actor: <optional who stated it>
source: <optional where observed>
observed_at: <ISO 8601 date>
hash: <SHA-256 hex of the statement>
entities: [<optional named entities>]
status: <active | archived>
---
```
