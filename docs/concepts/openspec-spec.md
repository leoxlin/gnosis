---
type: ConceptType
title: OpenSpecSpec
description: An OpenSpec specification defining testable normative behavior.
path: openspec
---

# OpenSpecSpec

An **OpenSpecSpec** defines current or changed behavior as normative requirements with testable scenarios.

## Use this for

- OpenSpec main specifications and delta specifications containing requirements and scenarios.

Do not use it for change motivation (OpenSpecProposal), implementation decisions (OpenSpecDesign), or execution checklists (OpenSpecTasks).

## Minimum record

- At least one normative requirement using `SHALL` or `MUST`.
- At least one `#### Scenario` with `WHEN` and `THEN` steps for every requirement.

## Schema

```markdown
---
type: OpenSpecSpec
---

## ADDED Requirements

### Requirement: <!-- requirement name -->
<!-- Testable normative behavior using SHALL or MUST -->

#### Scenario: <!-- scenario name -->
- **WHEN** <!-- condition -->
- **THEN** <!-- expected outcome -->
```
