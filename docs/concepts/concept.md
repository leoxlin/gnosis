---
type: ConceptType
title: Concept
description: A durable semantic or factual concept.
path: concepts
---

# Concept

A **Concept** preserves what is true: a definition, fact, mechanism, or synthesized understanding that outlives its sources.

By convention, Concept records live at `gnosis://<vault>/concepts/`, alongside Concept Type definitions.

## Use this for

- Technical facts, domain concepts, synthesized explanations, and company or project knowledge that answers "what is true?".

Do not use it for events (Event), lessons (Reflection), rules (Policy), agents or people (Entity), or tools and services (Resource).

## Minimum record

- `# Concept` with the self-contained definition.
- Optional `# Why it matters` and `# Sources`. Synthesized claims carry the inline markers `^[inferred]` or `^[ambiguous]`.

## Lifecycle

- Identity is the concept itself, not its title. Query for an existing page before creating one; reject duplicate identity.
- `status` follows `draft` → `reviewed` → `verified`; `disputed` records a contradiction until resolved; `archived` with a `superseded_by` link replaces deletion.
- Update understanding in place as knowledge grows, preserving unknown metadata; record the change in the nearest `log.md` when `vault_log` is enabled.
- Delete only a confirmed local duplicate or invalid record after tracing inbound links and obtaining explicit approval; repair every inbound reference.

## Schema

```yaml
---
type: Concept
title: <name>
description: <one-line summary>
status: <draft | reviewed | verified | disputed | archived>
confidence: <optional 0.0-1.0>
source: <optional origin of the claim>
valid_from: <optional ISO date>
tier: <optional core | supporting | peripheral>
superseded_by: <optional successor link>
---
```
