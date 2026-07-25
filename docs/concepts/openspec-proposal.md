---
type: ConceptType
title: OpenSpecProposal
description: An OpenSpec change proposal defining motivation, scope, capabilities, and impact.
path: openspec
---

# OpenSpecProposal

An **OpenSpecProposal** explains why a change is needed and identifies its observable scope before implementation decisions are made.

## Use this for

- OpenSpec `proposal.md` artifacts that define motivation, changes, affected capabilities, and impact.

Do not use it for behavioral requirements (OpenSpecSpec), implementation decisions (OpenSpecDesign), or execution checklists (OpenSpecTasks).

## Minimum record

- `## Why`, `## What Changes`, `## Capabilities`, and `## Impact`.

## Schema

```markdown
---
type: OpenSpecProposal
---

## Why

<!-- What problem does this change solve, and why now? -->

## What Changes

<!-- List the concrete changes. Mark breaking changes with **BREAKING**. -->

## Capabilities

### New Capabilities

<!-- `<kebab-case-name>`: observable behavior covered by the new spec -->

### Modified Capabilities

<!-- `<existing-spec-name>`: requirement-level behavior being changed -->

## Impact

<!-- Affected code, APIs, dependencies, or systems -->
```
