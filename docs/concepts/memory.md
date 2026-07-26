---
type: ConceptType
title: Memory
description: A scoped, self-contained agent memory of a durable fact, preference, or observation.
path: memories
---

# Memory

A **Memory** preserves one self-contained durable fact, preference, or observation under an explicit scope. Agents can write reconciled memories through the remember procedure or explicit scoped records through the memory API; recall and memory search read them.

By convention, Memory records live at `gnosis://<vault>/memories/`.

## Use this for

- User preferences and persona facts (`scope: user`), agent capabilities and learned limitations (`scope: agent`), and session- or run-scoped durable observations (`scope: session | run`).

Do not use it for conversation transcripts, working state, or knowledge with its own type: facts (Concept), lessons (Reflection), rules (Policy), episodes (Event).

## Minimum record

- `scope`, `observed_at`, and `hash` frontmatter, plus `# Memory` with one self-contained statement using absolute dates and verbatim proper nouns.
- Memory API records also carry fixed `user_id` and `agent_id`, UTC `created_at` and `updated_at`, and optional Mem0-compatible `metadata`.
- Optional `actor`, `source`, and `entities` (named entities, used for retrieval boosts).

## Lifecycle

- Creation, update, and archival go through [remember](../procedures/remember.md), which reconciles each candidate against the nearest existing memories as ADD, UPDATE, DELETE, or NONE; retrieval goes through [recall](../procedures/recall.md).
- `status` is `active` while current and `archived` when superseded or deleted; archived memories are retained for audit and negative knowledge, never silently removed.
- `hash` is the SHA-256 hex of the `# Memory` statement text; exact duplicates are never written.
- An exact active memory API duplicate returns `NOOP` without changing `created_at` or `updated_at`.
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
user_id: <required for memory API records>
agent_id: <required for memory API records>
observed_at: <ISO 8601 date>
created_at: <UTC RFC 3339 timestamp for memory API records>
updated_at: <UTC RFC 3339 timestamp for memory API records>
hash: <SHA-256 hex of the statement>
metadata: <optional mapping>
entities: [<optional named entities>]
status: <active | archived>
---
```
