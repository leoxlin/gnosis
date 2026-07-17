---
type: Decision
title: Adopt scoped agent memory as explicit vault records
description: Add mem0-style agent memory through Memory pages and remember/recall procedures while keeping Markdown authoritative and writes explicit.
---

# Decision

Adopt agent memory as first-class, explicitly written **Memory** vault records with mem0-style scoping and reconciliation:

- Each Memory page carries `scope` (`user | agent | session | run`), `actor`, `source`, `observed_at`, `entities`, and a content `hash` for exact-duplicate suppression.
- The write path is a `remember` vault procedure: extract self-contained durable facts from a supplied episode, retrieve the most similar existing Memory records, then apply **ADD** (new page), **UPDATE** (revise the matching page in place), **DELETE** (mark the page `status: archived` and retain it), or **NONE** (already captured or not durable). Every write goes through `gnosis apply page` so it is visible and validated.
- The read path is a `recall` vault procedure: scoped retrieval over Memory records that combines lexical matching, optional vector similarity, entity-match boosts, and recency, returning ranked memories with their provenance.
- Audit history comes from git plus retained archived pages; no separate history database exists.

This amends [Use pgvector for semantic knowledge retrieval](use-pgvector-semantic-retrieval.md): its exclusion of mem0-style conversational memory targeted *implicit* background inference over chat streams. Memory writes in gnosis are explicit, author-visible record operations requested through a procedure, which fits the durable authored knowledge model. The retrieval architecture itself is unchanged.

# Why

mem0's own results show that simple accumulation with good retrieval beats aggressive curation, and its v3 removed external graph databases entirely — the strongest signal that a file-based vault with a link graph covers the useful memory design space. Plain Memory pages give provenance, portability, and audit (git log) for free, while procedures keep the reconciliation logic portable across agents without new Go subsystems.

In-place UPDATE with git history preserves both currency and audit; archiving instead of deleting preserves negative knowledge ("checked, no longer true") that the knowledge-research taxonomy requires.

# Constraints

- Memory content is durable facts, preferences, and observations — not conversation transcripts; working state stays out of the vault.
- Extracted facts must be self-contained statements with absolute dates and verbatim proper nouns; pronouns and relative dates are resolved at write time.
- Exact content `hash` duplicates are never written; conflicting facts are reconciled explicitly through UPDATE or DELETE, never silently overwritten.
- Retrieval defaults to the lexical backend; vector similarity stays optional through the existing pgvector configuration.
- No external memory services, graph databases, reranker services, or background summarizers are added.
- Memory procedures never modify non-Memory records; converting a memory into a Concept, Decision, or Reflection goes through the owning procedure.
