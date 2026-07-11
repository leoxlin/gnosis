---
type: Vault Process
title: ingest-knowledge
description: Use when a source may update several related concept pages.
invocation: model
effects: [vault-write]
relationships:
  - type: instance_of
    target: ../../concepts/vault-process.md
---

# ingest-knowledge

`ingest-knowledge` compiles source material into the smallest useful set of durable, connected, and traceable concept records.

## Use when

- Ingesting a source that may update several related concepts.
- Extracting durable claims, relationships, uncertainty, and provenance from supplied evidence.

## Knowledge inputs

- Vault configuration, agent rules, and enabled navigation settings.
- Relevant Concept Type definitions and nearby concept pages.
- The supplied source, citations, and any already-recorded conflicting claims.

## Process

1. Resolve the vault from `gnosis.toml` or the current bundle. Read its agent rules, relevant concept definitions, and nearby pages; read root `index.md` or `log.md` only when its matching option is enabled.
2. Treat the input as evidence. Extract durable claims, relationships, uncertainties, and citations; separate sourced facts from agent inference.
3. Integrate by concept identity. Update matching pages and create only the smallest useful set of new pages. Preserve unknown frontmatter and follow the configured link format.
4. Keep claims traceable to their source. Surface contradictions or ambiguous identity instead of silently choosing a side.
5. When `vault_index` is enabled, regenerate affected indexes with `gnosis index -vault <root>`. When `vault_log` is enabled, add a concise newest-first entry to the nearest `log.md`.
6. Run `gnosis validate -vault <root>`.

## Completion

Every retained claim has a durable home and provenance, enabled navigation reflects the result, and vault validation passes.
