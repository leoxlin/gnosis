# Memory architecture

gnosis implements agent memory in the mem0 style, adapted to a plain-file vault.

## Design

- **Memory pages** are the local store: one self-contained statement per page, scoped `user | agent | session | run`, with `observed_at`, lifecycle state, and a content `hash`. API-created pages also preserve fixed `user_id` and `agent_id`, optional metadata, and UTC `created_at` and `updated_at`.
- **The shared memory service** selects exactly one backend per operation. Complete hosted or self-hosted Mem0 configuration selects Mem0; when every external-specific variable is absent, it selects the effective writable vault.
- **Vault Add/Search** writes validated pages through the canonical vault writer and filters active Memory pages by both configured identities before bounded lexical ranking. Exact active duplicates return `NOOP` without a write.
- **remember** is the write path: extract durable candidates, suppress exact duplicates by hash, retrieve the nearest existing memories, then reconcile each candidate as ADD (new page), UPDATE (revise in place), DELETE (archive with a reason), or NONE. Every operation is an explicit, validated page write.
- **recall** is the read path: scoped retrieval combining lexical search (vector optional), entity-match boosts, and recency, returning provenance with every answer.
- **Audit** is git history plus retained archived pages — the vault needs no separate history database.

## Why this shape

mem0's own trajectory informed it: their v3 moved to accumulate-and-rank over aggressive curation and removed external graph databases, because ranking handles currency and a link graph covers entity context. Plain pages give provenance, portability, and review for free. gnosis deliberately omits background summarizers, external graph stores, reranker services, cross-backend fallback, synchronization, and retry queues. A configured backend failure is terminal for that operation.

## Relationship to durable knowledge

Memories are not a dumping ground. When a memory graduates into durable knowledge, the owning procedure converts it: facts become Concepts, lessons become Reflections, and governing rules become Policies. Repository-development choices belong in OpenSpec.
