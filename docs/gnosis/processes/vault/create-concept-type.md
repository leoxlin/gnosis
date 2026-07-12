---
type: Gnosis Process
title: create-concept-type
description: Use when a vault needs a new or refined ontological category.
invocation: model
effects: [vault-write]
relationships:
  - type: instance_of
    target: gnosis://core/concepts/gnosis-process.md
---

# create-concept-type

`create-concept-type` establishes the smallest stable category that fits the author's examples. It resolves category boundaries before creating a reusable schema.

## Use when

- Creating a Concept Type record for a new category of vault knowledge.
- Refining an existing category whose boundary or required record shape is unclear.
- An ingestion process finds no existing type that fits a concept.

## Knowledge inputs

- Vault configuration and agent rules.
- Nearby Concept Type records and records that may belong to the proposed category.
- Discoverable repository or vault facts.
- The author's decisions about material boundaries, examples, and exclusions.

## Process

1. Resolve the vault and read its configuration and agent rules. List existing types with `gnosis concepts --pretty` and inspect only relevant definitions with `gnosis read --id '<concept-type URI>'`.
2. Investigate discoverable facts before asking the author. Ask only for category boundaries, examples, and exclusions.
3. Interview the author one question at a time until the category boundary is clear. If two independent boundaries remain, create two Concept Type records.
4. Name the category only after its boundary is clear. Create `docs/concepts/<kebab-name>.md` with `type: Concept Type`, `title`, and `description`; add frontmatter only when a consumer uses it.
5. Use a definition paragraph followed by `## Use this for`, an exclusion sentence, `## Minimum record`, and `## Schema`. Keep the schema to required fields and sections.
6. Run `gnosis index --vault <root>` only when `vault_index` is enabled, then run `gnosis validate --vault <root>`.

## Completion

The category has a clear boundary, a concise reusable record shape, enabled navigation reflects it, and vault validation passes.
