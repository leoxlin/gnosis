---
type: Procedure
title: ingest-knowledge
description: Use when supplied evidence should create or update one or more concept pages.
tags: [gnosis, vault]
invocation: model
---

# ingest-knowledge

`ingest-knowledge` compiles supplied evidence into the smallest useful set of durable, connected, and traceable concept records, including the single-concept case.

## Inputs

- Vault configuration, agent rules, and enabled navigation settings.
- Relevant Concept Type definitions and nearby concept pages.
- The supplied source, citations, and any already-recorded conflicting claims.

## Process

1. List exact types with `gnosis get concepts`, then read only the applicable Concept Type definitions with `gnosis get pages '<concept-type URI>' --full`. Read candidate identity matches returned by `gnosis search knowledge --backend lexical --vault <root> '<identity question>'`.
2. Treat the input as evidence. Extract durable claims, relationships, uncertainties, and citations; separate sourced facts from agent inference and tag claims inline: unmarked for extracted, `^[inferred]` for agent generalizations, `^[ambiguous]` for unresolved source disagreement. When the request identifies one concept, retain exactly one concept identity.
3. Integrate by identity. Update matching pages and create only the smallest useful set of new records. When no existing type fits, invoke `gnosis get procedures 'gnosis://_/procedures/vault/create-concept-type.md' --full` and resume after that procedure completes.
4. Build every record from its Concept Type definition, preserve unknown frontmatter, follow the configured link format, and keep claims traceable. Surface contradictions or ambiguous identity instead of silently choosing a side.
5. Persist each complete record with `gnosis apply page '<record URI>' --filename <draft-file>`. When `vault_log` is enabled, add one concise newest-first entry to the nearest `log.md`.
6. When `vault_index` is enabled, run `gnosis index vault --vault <root>`. Run `gnosis validate vault --vault <root>` in every case.

## Completion

Every retained claim has exactly one durable identity and provenance; a single-concept request changes exactly one concept page; enabled navigation reflects the result; and vault validation passes.
