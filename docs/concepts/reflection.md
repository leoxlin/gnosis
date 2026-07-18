---
type: ConceptType
title: Reflection
description: A distilled lesson, heuristic, or failure pattern learned from experience.
path: reflections
---

# Reflection

A **Reflection** preserves what was learned: a reusable heuristic, failure pattern, or strategy distilled from events, memories, and outcomes.

By convention, Reflection records live at `gnosis://<vault>/reflections/`.

## Use this for

- Lessons that answer "what lesson was learned?" — reusable guidance grounded in recorded experience.

Do not use it for raw episodes (Event), durable facts (Concept), or binding rules (Policy).

## Minimum record

- `# Reflection` with the lesson as one actionable statement.
- `# Evidence` linking the events, memories, or pages the lesson is distilled from, and `# Application` describing when it applies.

## Lifecycle

- Identity is the lesson. Query for an existing page before creating one; merge new evidence into the existing record instead of duplicating it.
- `status` follows `draft` → `established` as evidence accumulates, then `retired` with a `superseded_by` link when the lesson stops holding; retired pages are retained.
- Strengthen or qualify `# Application` in place as counterevidence arrives; record contradictions explicitly rather than deleting them.
- Delete only a confirmed local duplicate or invalid record after tracing inbound links and obtaining explicit approval; repair every inbound reference.

## Schema

```yaml
---
type: Reflection
title: <the lesson>
description: <one-line summary>
status: <draft | established | retired>
confidence: <optional 0.0-1.0>
superseded_by: <optional successor link>
---
```
