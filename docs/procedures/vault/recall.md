---
type: Procedure
title: recall
description: Use when answering from scoped Memory records, such as user preferences or agent lessons.
tags: [gnosis, vault, memory]
invocation: model
---

# recall

`recall` answers from Memory records only, ranking scoped candidates by relevance, entity match, and recency, and returning each memory's provenance. It never writes.

## Inputs

- The author's question and any requested `scope`, `entities`, or recency bounds.
- Vault configuration and the effective Memory Concept Type definition.
- Memory candidates returned by lexical and, when configured, vector search.

## Process

1. Resolve the vault and the requested filters. Default to all active scopes when none are given.
2. Run `gnosis search knowledge --backend lexical --vault <root> "<question>"` and keep only Memory records; add `gnosis search knowledge --backend vector --vault <root> "<question>"` when semantic retrieval is configured, merging candidates by URI.
3. Exclude `status: archived` memories unless the author asks for history or the question is about a change of mind.
4. Rank the remaining candidates: scope match first, then shared `entities` with the question, then `observed_at` recency. Prefer the newest memory when several cover one subject; archived predecessors explain the change.
5. Read only the top candidates with `gnosis get pages '<URI>' --full`; never scan the whole `memories/` directory.
6. Answer with each memory's statement and provenance (`scope`, `actor`, `source`, `observed_at`). Label `^[inferred]` memories as inference. When no memory applies, report the gap and suggest `remember` if the author supplies the fact.

## Completion

The answer is grounded in ranked Memory records with visible provenance; archived memories surface only as history; no files changed.
