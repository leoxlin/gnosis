---
type: ConceptType
title: OpenSpecTasks
description: An OpenSpec implementation checklist of ordered, verifiable tasks.
path: openspec
---

# OpenSpecTasks

An **OpenSpecTasks** record turns an accepted change into an ordered checklist whose completion reflects verified implementation progress.

## Use this for

- OpenSpec `tasks.md` artifacts that track small implementation and validation steps.

Do not use it for change motivation (OpenSpecProposal), behavioral requirements (OpenSpecSpec), or implementation decisions (OpenSpecDesign).

## Minimum record

- Ordered `- [ ] X.Y Description` checklist items, including relevant validation.

## Schema

```markdown
---
type: OpenSpecTasks
---

## 1. <!-- Task group -->

- [ ] 1.1 <!-- Small, verifiable task -->
- [ ] 1.2 <!-- Validation for the implemented behavior -->
```
