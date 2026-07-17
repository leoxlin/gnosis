---
type: ConceptType
title: Event
description: A dated episode, action, or observation worth remembering.
path: events
---

# Event

An **Event** preserves what happened: incidents, actions, tool executions, conversations of record, and direct observations, each anchored to a time.

By convention, Event records live at `gnosis://<vault>/events/`.

## Use this for

- Episodic and perceptual knowledge — incidents, deployments, decisions enacted, experiments, and observations that answer "what happened?" or "what was observed?".

Do not use it for durable facts (Concept), distilled lessons (Reflection), or agent working state, which never enters the vault.

## Minimum record

- `occurred_at` frontmatter and `# Event` describing what happened.
- Optional `# Context` and `# Outcome`; link causes and effects with typed `relationships` (`causes`, `caused_by`, `resolved_by`).

## Lifecycle

- Identity is the episode at its time. Events are append-only: correct a record by creating a new Event and setting `superseded_by` on the prior one, never by rewriting history.
- `status` is `recorded` on creation, `verified` once independently confirmed, and `disputed` while accounts conflict.
- Reflect on clusters of Events into Reflection records through the owning procedure instead of editing events into lessons.
- Delete only a confirmed local duplicate or invalid record after tracing inbound links and obtaining explicit approval; repair every inbound reference.

## Schema

```yaml
---
type: Event
title: <what happened>
description: <one-line summary>
occurred_at: <ISO 8601 timestamp>
actor: <optional who acted>
source: <optional where observed>
status: <recorded | verified | disputed>
superseded_by: <optional corrective event link>
---
```
