---
type: Decision
title: Adopt obsidian-wiki maintenance conventions
description: Bring the vault wiki to obsidian-wiki parity through tiered retrieval, cross-linking, consolidation, and lifecycle metadata carried by procedures and concept definitions.
---

# Decision

Adopt the maintenance conventions of [Ar9av's obsidian-wiki](../references/obsidian-wiki.md) as vault procedures and concept-definition metadata, not as new Go subsystems:

1. **Tiered retrieval.** `query-vault` follows an explicit cost ladder: catalog/index pass, lexical search, optional vector pass, bounded section reads, full reads of at most the top candidates, then bounded multi-hop traversal for path questions. Query cost stays flat as the vault grows.
2. **Cross-linking.** A `link-pages` procedure discovers unlinked mentions of known page titles and aliases, inserts exact links, and records typed `relationships` edges, respecting the configured link format and skipping code blocks and frontmatter.
3. **Consolidation.** `maintain-vault` audits and repairs orphans, stale pages, contradictions, tag fragmentation, duplicate identities, and broken typed relationships, with a consolidation report per pass.
4. **Lifecycle metadata.** Content concept definitions adopt the shared `status`, `tier`, `confidence`, and `superseded_by` fields so pages carry lifecycle and query-ordering signals.
5. **Provenance markers.** Page bodies tag synthesized claims inline: unmarked claims are extracted from sources, `^[inferred]` marks agent generalizations, and `^[ambiguous]` marks unresolved source disagreement.

# Why

The Karpathy LLM-wiki pattern (compile, don't retrieve) only compounds when the compiled artifact is actively maintained: cross-linked, deduplicated, and freshness-checked. obsidian-wiki demonstrates that these operations work reliably as agent-executed skills over plain Markdown, which matches the gnosis procedure model exactly. Keeping the conventions in procedures and concept definitions preserves the OKF bundle format and avoids baking policy into Go code.

# Constraints

- Conventions live in Procedure records and Concept Type definitions; no new Go files, special filenames, or index formats are introduced beyond the existing `index.md` and `log.md`.
- Cross-linking and consolidation change page content only through the normal validated write path.
- Provenance markers are the only sanctioned inline claim annotations; new marker syntax requires a superseding decision.
- Tiered retrieval must not change the compact `QueryResult` contract; full content is always obtained by reading exact pages.
