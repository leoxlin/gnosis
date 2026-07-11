---
type: Vault Process
title: ingest-concept
description: Use when one named knowledge object should become or update one concept page.
---

# ingest-concept

`ingest-concept` gives one concept a durable, traceable home without duplicating an existing identity or inventing unsupported claims.

## Use when

- A single named idea, entity, decision, reference, or other knowledge object should become or update one concept page.
- A supplied source provides evidence about one identified concept.

## Knowledge inputs

- Vault configuration, agent rules, and enabled navigation settings.
- The relevant Concept Type and pages that may share the concept's identity.
- The supplied source and its provenance.

## Process

1. Resolve the vault and read its configuration, agent rules, relevant Concept Type, and pages that may share the concept's identity. Read the root index only when `vault_index` is enabled.
2. Select exactly one concept and an existing type. When no type fits, use `create-concept-type` and resume after the category exists.
3. Update the identity-matching page or create one concept page. Preserve unknown frontmatter and use the configured link format.
4. Ground claims in the supplied source, label uncertainty or inference, and link relevant existing pages.
5. Regenerate indexes when `vault_index` is enabled, append a concise entry when `vault_log` is enabled, and validate the vault.

## Completion

Exactly one concept page is created or updated, its claims are traceable, enabled navigation reflects it, and vault validation passes.
