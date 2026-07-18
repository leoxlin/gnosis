---
type: ConceptType
title: Policy
description: A rule, constraint, or permission that governs what should or must be done.
path: policies
---

# Policy

A **Policy** preserves what should or must be done: rules, constraints, permissions, and conditional guidance with their enforcement.

By convention, Policy records live at `gnosis://<vault>/policies/`.

## Use this for

- Normative and conditional knowledge — security controls, permissions, technology-selection rules, and situational guidance that answers "what should or must be done?" or "when does this apply?".

Do not use it for repository-development choices (OpenSpec), facts (Concept), or lessons (Reflection).

## Minimum record

- `# Policy` with the rule in one exact statement.
- `# Rationale`, `# Enforcement` describing how compliance is checked, and optional `# Exceptions`.

## Lifecycle

- Identity is the rule. Query for an existing page before creating one; reject duplicate identity.
- `status` follows `draft` → `active` → `retired`; retired policies are retained with their rationale, never deleted.
- Change a rule in place only as a non-semantic correction; a changed rule retires the old record and creates a new one linked by `superseded_by`.
- Delete only a confirmed local duplicate or invalid `draft` after tracing inbound links and obtaining explicit approval; repair every inbound reference.

## Schema

```yaml
---
type: Policy
title: <the rule>
description: <one-line summary>
status: <draft | active | retired>
applies_to: <optional scope of application>
superseded_by: <optional successor link>
---
```
