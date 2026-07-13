---
type: ConceptType
title: Purpose
description: The durable statement of why a project or vault exists and where it stops.
path: .
---

# Purpose

A **Purpose** states an enduring outcome and the boundaries that implementation cannot express fully.

By convention, the Purpose record lives at `gnosis://<vault>/purpose.md`.

## Use this for

- Durable outcomes, beneficiaries, sub-purposes, and exclusions.

Do not put architecture, plans, milestones, or tasks here.

## Minimum record

- `# Purpose` and `# Boundaries`.
- Optional `# Sub-purposes` when decomposition adds clarity.

## Lifecycle

- A vault has exactly one effective Purpose. Multiple effective records are an identity conflict that must be repaired before mutation.
- Creation and updates distinguish discoverable facts from author-owned intent. Resolve every material choice about outcome, beneficiaries, sub-purposes, and boundaries, then obtain the author's explicit confirmation of the complete proposed Purpose.
- Update the existing record in place while preserving applicable unknown metadata.
- Delete only a confirmed local origin after tracing every inbound link, explaining the loss of repository intent, and obtaining explicit confirmation that the vault should no longer have a Purpose. Repair or intentionally remove every inbound reference; report imported or bundled records to their owning vault.

## Schema

```yaml
---
type: Purpose
title: <name>
description: <outcome>
---

# Purpose
# Sub-purposes
# Boundaries
```
