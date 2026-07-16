---
type: Decision
title: Use pgvector for semantic knowledge retrieval
description: Keep Markdown authoritative while indexing derived document chunks in pgvector for semantic search.
supersedes: keep-search-sources-and-retrieval-backends-replaceable.md
---

# Decision

Keep configured gnosis Markdown vaults as the authoritative, portable knowledge source and add pgvector as a derived semantic retrieval backend. Split effective documents into bounded chunks, embed them through one configured OpenAI-compatible embeddings endpoint, and store the chunks with their canonical URI, revision, and compact document metadata. Synchronize the derived index only through an explicit command and use cosine distance for semantic search.

Preserve the existing source-independent `Document` and bounded `QueryResult` contracts. Lexical retrieval remains available without external services; vector retrieval is selected explicitly and never changes vault content.

# Why

Semantic retrieval is now an explicit product requirement. The existing retrieval boundary already separates live Markdown loading from ranking, so pgvector can add RAG-oriented recall without making PostgreSQL the knowledge source or weakening author ownership. This follows mem0's useful separation of embeddings, vector storage, metadata, and search while omitting its conversational-memory inference, which does not fit gnosis's durable authored knowledge model.

An explicit synchronization command makes database mutation visible and testable. Exact pgvector cosine search provides perfect recall and avoids speculative HNSW lifecycle and tuning until measured scale requires it.

# Constraints

- Markdown documents and canonical gnosis URIs remain the source of truth; the vector database is disposable derived state.
- Vector rows retain the source URI and revision so results have provenance and stale indexes are detectable.
- Index synchronization replaces one workspace scope atomically and must not alter vault files.
- Embedding and database credentials come from the process environment, not committed vault configuration.
- Semantic results preserve the compact backend-independent query contract; callers read exact pages separately when they need complete content.
- The initial implementation supports one OpenAI-compatible embeddings HTTP shape and pgvector exact cosine search. Add provider abstractions or approximate indexes only when a second provider or measured scale requires them.

# Supersedes

- [Keep search sources and retrieval backends replaceable](keep-search-sources-and-retrieval-backends-replaceable.md)
