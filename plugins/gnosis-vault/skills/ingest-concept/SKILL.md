---
name: ingest-concept
description: Ingest one concept into a gnosis OKF/LLM wiki. Use when a single named idea, entity, decision, reference, or other knowledge object should become or update one concept page.
---

# Ingest Concept

1. Resolve the vault and read its configuration, agent rules, relevant concept type, and pages that may share the concept's identity. Read the root index only when `vault_index` is enabled.
2. Select one concept and its existing type. When no type fits, hand off to `create-concept-type` and resume after that category exists.
3. Update the identity-matching page or create one concept page. Preserve unknown frontmatter and follow the configured link format.
4. Ground claims in the supplied source, mark uncertainty or inference, and connect the concept to relevant existing pages.
5. Regenerate indexes when `vault_index` is enabled, append a concise entry when `vault_log` is enabled, and validate the vault.

Finish when exactly one concept page is created or updated, its claims are traceable, enabled navigation reflects it, and validation passes.
