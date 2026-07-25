---
type: ConceptType
title: OpenSpecDesign
description: An OpenSpec design recording the implementation decisions needed to deliver a change.
path: openspec
---

# OpenSpecDesign

An **OpenSpecDesign** records the implementation context, goals, decisions, and risks needed to deliver an accepted change.

## Use this for

- OpenSpec `design.md` artifacts that preserve necessary implementation choices and trade-offs.

Do not use it for change motivation (OpenSpecProposal), behavioral requirements (OpenSpecSpec), or execution checklists (OpenSpecTasks).

## Minimum record

- `## Context`, `## Goals / Non-Goals`, `## Decisions`, and `## Risks / Trade-offs`.

## Schema

```markdown
---
type: OpenSpecDesign
---

## Context

<!-- Current state and constraints -->

## Goals / Non-Goals

<!-- Intended outcomes and explicit exclusions -->

## Decisions

<!-- Necessary implementation choices and rationale -->

## Risks / Trade-offs

<!-- [Risk] → Mitigation -->
```
