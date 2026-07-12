---
type: Decision
title: Keep search sources and retrieval backends replaceable
description: Start with live gnosis vault retrieval while preserving a stable boundary for broader sources and indexed search backends.
---

# Decision

Represent searchable knowledge as source-independent, namespaced documents and keep retrieval behind a replaceable backend boundary. Use configured gnosis Markdown vaults as the first source and an in-memory, field-weighted lexical retriever as the first backend.

# Why

Vault queries need a small, dependency-free implementation now, but search is expected to grow to working trees and external knowledge vaults. Separating document loading from retrieval lets those sources and persistent or semantic backends evolve without redefining the query commands or coupling general search to OKF parsing.

# Constraints

- Query results expose compact document identities and metadata, not backend-specific records or page bodies.
- The initial backend reads live vault files and creates no cache, so read-only queries cannot become stale or mutate knowledge.
- Future sources must provide stable namespaced document identities and resolved links.
- Future backends must preserve the bounded query result contract even when their internal indexes or ranking strategies differ.

# Rejected alternatives

- Hard-coding vault parsing and ranking into the CLI would make broader sources duplicate query behavior.
- Adding a persistent full-text or semantic index now would introduce lifecycle, dependency, and portability costs before gnosis needs that scale.
- Copying obsidian-wiki's filename-keyed substring scoring would make duplicate concept names ambiguous and underuse OKF metadata.

# Related decisions

- [`gnosis` purpose](../purpose.md)
- [Bootstrap `gnosis` knowledge first on OKF](bootstrap-knowledge-first.md)
- [Consolidate runtime adapters in the `gnosis` plugin](consolidate-runtime-adapters-in-gnosis-plugin.md)
