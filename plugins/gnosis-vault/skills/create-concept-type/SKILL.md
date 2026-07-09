---
name: create-concept-type
description: Use when creating or refining a `gnosis` Concept Type note for an OKF vault.
---

Create the smallest ontological category that fits the user's examples.

1. Read nearby `docs/concepts/*.md` files first; reuse their shape.
2. Interview the user until the category boundary is clear. Ask one question at a time and wait for the answer.
3. Look up facts in the vault or repo. Ask the user only for boundary decisions, examples, and exclusions.
4. Name the category only after its boundary is clear. If two independent boundaries remain, create two concept types.
5. Write `docs/concepts/<kebab-name>.md` with `type: Concept Type`, `title`, `description`, `defines`, `tags`, and `timestamp`.
6. Use this body shape: definition paragraph, captured questions, `## Use this for`, exclusion sentence, `## Minimum record`, `## Schema`.
7. Keep the schema to required fields and sections. Add optional fields only when the category needs them.
8. Regenerate indexes and validate the vault before handoff.
