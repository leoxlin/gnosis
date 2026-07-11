---
type: Gnosis Process
title: ingest-knowledge
description: Use when supplied evidence should create or update one or more concept pages.
invocation: model
effects: [vault-write]
relationships:
  - type: instance_of
    target: ../../concepts/gnosis-process.md
---

# ingest-knowledge

`ingest-knowledge` compiles supplied evidence into the smallest useful set of durable, connected, and traceable concept records, including the single-concept case.

## Use when

- Ingesting a source about one named concept or several related concepts.
- Extracting durable claims, relationships, uncertainty, and provenance from supplied evidence.

## Knowledge inputs

- Vault configuration, agent rules, and enabled navigation settings.
- Relevant Concept Type definitions and nearby concept pages.
- The supplied source, citations, and any already-recorded conflicting claims.

## Process

1. List exact types with `gnosis concepts --pretty`, then read only the applicable Concept Type definitions with `gnosis read --id '<concept-type URI>'`. Read candidate identity matches returned by `gnosis query graph --vault <root> --pretty '<identity question>'`.
2. Treat the input as evidence. Extract durable claims, relationships, uncertainties, and citations; separate sourced facts from agent inference. When the request identifies one concept, retain exactly one concept identity.
3. Integrate by identity. Update matching pages and create only the smallest useful set of new records. When no existing type fits, invoke `gnosis process invoke --id 'gnosis://core/gnosis/processes/create-concept-type.md' --pretty` and resume after that process completes.
4. Build every record from its Concept Type definition, preserve unknown frontmatter, follow the configured link format, and keep claims traceable. Surface contradictions or ambiguous identity instead of silently choosing a side.
5. Persist each complete record with `gnosis write --type '<exact type>' --title '<exact title>' <draft-file>`. When `vault_log` is enabled, add one concise newest-first entry to the nearest `log.md`.
6. When `vault_index` is enabled, run `gnosis index --vault <root>`. Run `gnosis validate --vault <root>` in every case.

## Completion

Every retained claim has exactly one durable identity and provenance; a single-concept request changes exactly one concept page; enabled navigation reflects the result; and vault validation passes.
