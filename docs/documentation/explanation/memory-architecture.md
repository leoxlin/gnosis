# Memory architecture

gnosis implements agent memory in the mem0 style, adapted to a plain-file vault.

## Design

- **Memory pages** are the store: one self-contained statement per page, scoped `user | agent | session | run`, with `observed_at`, `entities`, and a content `hash`.
- **remember** is the write path: extract durable candidates, suppress exact duplicates by hash, retrieve the nearest existing memories, then reconcile each candidate as ADD (new page), UPDATE (revise in place), DELETE (archive with a reason), or NONE. Every operation is an explicit, validated page write.
- **recall** is the read path: scoped retrieval combining lexical search (vector optional), entity-match boosts, and recency, returning provenance with every answer.
- **Audit** is git history plus retained archived pages — the vault needs no separate history database.

## Why this shape

mem0's own trajectory informed it: their v3 moved to accumulate-and-rank over aggressive curation and removed external graph databases, because ranking handles currency and a link graph covers entity context. Plain pages give provenance, portability, and review for free. What gnosis deliberately omits: background summarizers, external graph stores, reranker services, and implicit memory writes — every mutation is author-visible.

## Relationship to durable knowledge

Memories are not a dumping ground. When a memory graduates into project truth, the owning procedure converts it: facts become Concepts, lessons become Reflections, settled choices become Decisions.
