---
type: Procedure
title: create-concept-type
description: Use when a vault needs a new or refined ontological category.
tags: [gnosis, vault]
invocation: model
---

# create-concept-type

`create-concept-type` establishes the smallest stable category that fits the author's examples. It resolves category boundaries before creating a reusable schema.

## Inputs

- Vault configuration and agent rules.
- Nearby Concept Type records and records that may belong to the proposed category.
- Discoverable repository or vault facts.
- The author's decisions about material boundaries, examples, and exclusions.

## Process

1. Resolve the vault and read its configuration and agent rules. List existing types with `gnosis get concepts` and inspect only relevant definitions with `gnosis get pages '<concept-type URI>' --full`.
2. Investigate discoverable facts before asking the author. Ask only for category boundaries, examples, and exclusions.
3. Interview the author one question at a time until the category boundary is clear. If two independent boundaries remain, create two Concept Type records.
4. Name the category only after its boundary is clear. Create `docs/concepts/<kebab-name>.md` with `type: ConceptType`, `title`, and `description`; add frontmatter only when a consumer uses it.
5. Use a definition paragraph followed by `## Use this for`, an exclusion sentence, `## Minimum record`, and `## Schema`. Keep the schema to required fields and sections.
6. Run `gnosis index vault --vault <root>` only when `vault_index` is enabled, then run `gnosis validate vault --vault <root>`.

## Completion

The category has a clear boundary, a concise reusable record shape, enabled navigation reflects it, and vault validation passes.
